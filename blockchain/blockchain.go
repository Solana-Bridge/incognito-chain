package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/dataaccessobject/flatfile"
	"github.com/incognitochain/incognito-chain/dataaccessobject/stats"
	coinIndexer "github.com/incognitochain/incognito-chain/transaction/coin_indexer"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/blockchain/committeestate"
	"github.com/incognitochain/incognito-chain/blockchain/pdex"
	"github.com/incognitochain/incognito-chain/blockchain/signaturecounter"
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/prque"
	"github.com/incognitochain/incognito-chain/config"
	configpkg "github.com/incognitochain/incognito-chain/config"
	"github.com/incognitochain/incognito-chain/consensus_v2/signatureschemes/bridgesig"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/multiview"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	bnbrelaying "github.com/incognitochain/incognito-chain/relaying/bnb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
	"github.com/incognitochain/incognito-chain/syncker/finishsync"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/txpool"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/pkg/errors"
)

type BlockChain struct {
	BeaconChain *BeaconChain
	ShardChain  []*ShardChain
	config      Config
	cacheConfig CacheConfig
	cQuitSync   chan struct{}

	IsTest bool

	beaconViewCache             *lru.Cache
	committeeByEpochCache       *lru.Cache
	committeeByEpochProcessLock sync.Mutex
}

// Config is a descriptor which specifies the blockchain instblockchain/beaconstatefulinsts.goance configuration.
type Config struct {
	BTCChain        *btcrelaying.BlockChain
	BNBChainState   *bnbrelaying.BNBChainState
	DataBase        map[int]incdb.Database
	FlatFileManager map[int]*flatfile.FlatFileManager
	MemCache        *memcache.MemoryCache
	Interrupt       <-chan struct{}
	RelayShards     []byte
	// NodeMode          string
	BlockGen          *BlockGenerator
	TxPool            TxPool
	TempTxPool        TxPool
	CRemovedTxs       chan metadata.Transaction
	FeeEstimator      map[byte]FeeEstimator
	IsBlockGenStarted bool
	PubSubManager     Pubsub
	Syncker           Syncker
	Server            Server
	ConsensusEngine   ConsensusEngine
	Highway           Highway
	OutCoinByOTAKeyDb *incdb.Database
	IndexerWorkers    int64
	IndexerToken      string
	PoolManager       *txpool.PoolManager

	relayShardLck sync.Mutex
	usingNewPool  bool
}

type CacheConfig struct {
	triegc               map[byte]*prque.Prque // Priority queue mapping block numbers to tries to gc
	trieJournalPath      map[int]string
	trieJournalCacheSize int
	blockTrieInMemory    uint64
	trieNodeLimit        common.StorageSize
	trieImgsLimit        common.StorageSize
}

func NewFlatFileConfig(config *Config) {

	config.FlatFileManager = make(map[int]*flatfile.FlatFileManager)

	for chainID, db := range config.DataBase {
		path := db.GetPath() + "/flatfile"
		flatFileManager, err := flatfile.NewFlatFile(path, 5000)
		if err != nil {
			return
		}
		config.FlatFileManager[chainID] = flatFileManager
	}
}

func NewCacheConfig(cfg *Config) (CacheConfig, error) {

	cacheConfig := configCache32GB
	cacheConfig.triegc = make(map[byte]*prque.Prque)
	trieJournal := make(map[int]string)

	for i := 0; i < common.MaxShardNumber; i++ {
		cacheConfig.triegc[byte(i)] = prque.New(nil)
	}

	for chainID, db := range cfg.DataBase {

		journalPath := db.GetPath() + "/metadata.bin"
		if _, err := os.Stat(journalPath); os.IsNotExist(err) {
			_, err := os.Create(journalPath)
			if err != nil {
				return CacheConfig{}, err
			}
		}
		trieJournal[chainID] = journalPath

	}

	cacheConfig.trieJournalPath = trieJournal

	return cacheConfig, nil
}

func NewBlockChain(isTest bool) *BlockChain {
	bc := &BlockChain{}
	bc.config.IsBlockGenStarted = false
	bc.IsTest = isTest
	bc.beaconViewCache, _ = lru.New(100)
	bc.cQuitSync = make(chan struct{})
	return bc
}

/*
Init - init a blockchain view from config
*/
func (blockchain *BlockChain) Init(config *Config) error {
	// Enforce required config fields.
	if config.DataBase == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Database is not config"))
	}
	NewFlatFileConfig(config)
	blockchain.config = *config
	blockchain.config.IsBlockGenStarted = false
	blockchain.IsTest = false
	blockchain.beaconViewCache, _ = lru.New(100)
	blockchain.committeeByEpochCache, _ = lru.New(100)
	cacheConfig, err := NewCacheConfig(config)
	if err != nil {
		return err
	}
	blockchain.cacheConfig = cacheConfig

	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if configpkg.Param().TxPoolVersion == 0 {
		blockchain.config.usingNewPool = false
	} else {
		blockchain.config.usingNewPool = true
	}

	if err := blockchain.InitChainState(); err != nil {
		return err
	}
	blockchain.cQuitSync = make(chan struct{})

	EnableIndexingCoinByOTAKey = config.OutCoinByOTAKeyDb != nil
	if EnableIndexingCoinByOTAKey {
		allTokens := make(map[common.Hash]interface{})
		tokenStates, err := blockchain.ListAllPrivacyCustomTokenAndPRV()
		if err != nil {
			return err
		}
		for tokenID := range tokenStates {
			allTokens[tokenID] = true
		}

		outcoinIndexer, err = coinIndexer.NewOutCoinIndexer(config.IndexerWorkers, *config.OutCoinByOTAKeyDb, config.IndexerToken, allTokens)
		if err != nil {
			return err
		}
		if config.IndexerWorkers > 0 {
			txDbs := make([]*statedb.StateDB, 0)
			bestBlocks := make([]uint64, 0)
			for shard := 0; shard < common.MaxShardNumber; shard++ {
				txDbs = append(txDbs, blockchain.GetBestStateTransactionStateDB(byte(shard)))
				bestBlocks = append(bestBlocks, blockchain.GetBestStateShard(byte(shard)).ShardHeight)
			}
			cfg := &coinIndexer.IndexerInitialConfig{TxDbs: txDbs, BestBlocks: bestBlocks}
			go outcoinIndexer.Start(cfg)
		}
	}
	return nil
}

// InitChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
func (blockchain *BlockChain) InitChainState() error {
	// Determine the state of the chain database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	stats.IsEnableBPV3Stats = config.Param().IsEnableBPV3Stats
	blockchain.BeaconChain = NewBeaconChain(multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.BeaconChainKey)
	var err error
	blockchain.BeaconChain.hashHistory, err = lru.New(1000)
	if err != nil {
		return err
	}
	//check if bestview is not stored, then init
	bcDB := blockchain.GetBeaconChainDatabase()
	if _, err := rawdbv2.GetBeaconViews(bcDB); err != nil {
		err := blockchain.initBeaconState()
		if err != nil {
			Logger.log.Error("debug beacon state init error", err)
			return err
		}
	} else {
		//if restore fail, return err
		if err := blockchain.RestoreBeaconViews(); err != nil {
			Logger.log.Error("debug restore beacon fail, init", err)
			return err
		}
	}

	Logger.log.Infof("Init Beacon View height %+v", blockchain.BeaconChain.GetBestView().GetHeight())

	finishsync.NewDefaultFinishSyncMsgPool()
	go func() {
		for {
			finishsync.DefaultFinishSyncMsgPool.Clean(blockchain.BeaconChain.GetBestView().(*BeaconBestState).GetSyncingValidatorsString())
			time.Sleep(time.Minute * 5)
		}
	}()

	//beaconHash, err := statedb.GetBeaconBlockHashByIndex(blockchain.GetBeaconBestState().GetBeaconConsensusStateDB(), 1)
	//panic(beaconHash.String())
	wl, err := blockchain.GetWhiteList()
	if err != nil {
		Logger.log.Errorf("Can not get whitelist txs, error %v", err)
	}
	whiteListTx = make(map[string]bool)
	for k := range wl {
		whiteListTx[k] = true
	}
	blockchain.ShardChain = make([]*ShardChain, blockchain.GetBeaconBestState().ActiveShards)
	for shardID := byte(0); int(shardID) < blockchain.GetBeaconBestState().ActiveShards; shardID++ {
		tp, err := blockchain.config.PoolManager.GetShardTxsPool(shardID)
		if err != nil {
			return err
		}
		tv := NewTxsVerifier(
			nil,
			tp,
			wl,
			nil,
		)
		tp.UpdateTxVerifier(tv)
		blockchain.ShardChain[shardID] = NewShardChain(int(shardID), multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.GetShardChainKey(shardID), tp, tv)
		blockchain.ShardChain[shardID].hashHistory, err = lru.New(1000)
		if err != nil {
			return err
		}

		//check if bestview is not stored, then init
		if _, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID); err != nil {
			err := blockchain.InitShardState(shardID)
			if err != nil {
				Logger.log.Error("debug shard state init error", err)
				return err
			}
		} else {
			//if restore fail, return err
			if err := blockchain.RestoreShardViews(shardID); err != nil {
				Logger.log.Error("debug restore shard fail, init", err)
				return err
			}
		}

		sBestState := blockchain.ShardChain[shardID].GetBestState()
		txDB := sBestState.GetCopiedTransactionStateDB()

		blockchain.ShardChain[shardID].TxsVerifier.UpdateTransactionStateDB(txDB)
		Logger.log.Infof("Init Shard View shardID %+v, height %+v", shardID, blockchain.ShardChain[shardID].GetFinalViewHeight())
	}

	return nil
}

var whiteListTx map[string]bool

func (blockchain *BlockChain) GetWhiteList() (map[string]interface{}, error) {
	netID := config.Param().Name
	res := map[string]interface{}{}
	whitelistData, err := ioutil.ReadFile("./whitelist.json")
	if err != nil {
		return nil, err
	}
	whiteList := map[string][]string{}
	err = json.Unmarshal(whitelistData, &whiteList)
	if err != nil {
		return nil, err
	}
	if wlByNetID, ok := whiteList[netID]; ok {
		for _, txHash := range wlByNetID {
			res[txHash] = true
		}
	}
	return res, nil
}

func (blockchain *BlockChain) WhiteListTx() map[string]bool {
	return whiteListTx
}

/*
// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
*/
func (blockchain *BlockChain) InitShardState(shardID byte) error {
	// Create a new block from genesis block and set it as best block of chain
	initShardBlock := types.ShardBlock{}
	initShardBlock = *genesisShardBlock
	initShardBlock.Header.ShardID = shardID
	initShardBlockHeight := initShardBlock.Header.Height
	var shardCommitteeState committeestate.ShardCommitteeState

	initShardState := NewBestStateShardWithConfig(shardID, shardCommitteeState)
	beaconBlocks, err := blockchain.GetBeaconBlockByHeight(initShardBlockHeight)
	if err != nil {
		return NewBlockChainError(FetchBeaconBlockError, err)
	}
	genesisBeaconBlock := beaconBlocks[0]

	err = initShardState.initShardBestState(blockchain, blockchain.GetShardChainDatabase(shardID), &initShardBlock, genesisBeaconBlock)
	if err != nil {
		return err
	}
	committeeChange := committeestate.NewCommitteeChange()
	committeeChange.ShardCommitteeAdded[shardID] = initShardState.GetShardCommittee()

	err = blockchain.processStoreShardBlock(initShardState, &initShardBlock, committeeChange, []*types.BeaconBlock{genesisBeaconBlock})
	if err != nil {
		return err
	}

	return nil
}

func (blockchain *BlockChain) initBeaconState() error {
	initBlock := genesisBeaconBlock
	var committeeState committeestate.BeaconCommitteeState

	initBeaconBestState := NewBeaconBestStateWithConfig(committeeState)
	err := initBeaconBestState.initBeaconBestState(initBlock, blockchain, blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	initBlockHash := initBeaconBestState.BestBlock.Header.Hash()
	initBlockHeight := initBeaconBestState.BestBlock.Header.Height
	// Insert new block into beacon chain
	if err := statedb.StoreAllShardCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.GetShardCommittee()); err != nil {
		return err
	}
	if err := statedb.StoreBeaconCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.GetBeaconCommittee()); err != nil {
		return err
	}

	committees := initBeaconBestState.GetShardCommitteeFlattenList()
	missingSignatureCounter := signaturecounter.NewDefaultSignatureCounter(committees)
	initBeaconBestState.SetMissingSignatureCounter(missingSignatureCounter)

	consensusRootHash, _, err := initBeaconBestState.consensusStateDB.Commit(true)
	err = initBeaconBestState.consensusStateDB.Database().TrieDB().Commit(consensusRootHash, false, nil)
	if err != nil {
		return err
	}
	initBeaconBestState.consensusStateDB.ClearObjects()
	if err := rawdbv2.StoreBeaconBlockByHash(blockchain.GetBeaconChainDatabase(), initBlockHash, &initBeaconBestState.BestBlock); err != nil {
		Logger.log.Error("Error store beacon block", initBeaconBestState.BestBlockHash, "in beacon chain")
		return err
	}
	rawdbv2.StoreFinalizedBeaconBlockHashByIndex(blockchain.GetBeaconChainDatabase(), initBlockHeight, initBlockHash)

	// State Root Hash
	bRH := BeaconRootHash{
		ConsensusStateDBRootHash: consensusRootHash,
		FeatureStateDBRootHash:   common.EmptyRoot,
		RewardStateDBRootHash:    common.EmptyRoot,
		SlashStateDBRootHash:     common.EmptyRoot,
	}

	initBeaconBestState.ConsensusStateDBRootHash = consensusRootHash
	if err := rawdbv2.StoreBeaconRootsHash(blockchain.GetBeaconChainDatabase(), initBlockHash, bRH); err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	// Insert new block into beacon chain
	blockchain.BeaconChain.multiView.AddView(initBeaconBestState)
	if err := blockchain.BackupBeaconViews(blockchain.GetBeaconChainDatabase()); err != nil {
		Logger.log.Error("Error Store best state for block", blockchain.GetBeaconBestState().BestBlockHash, "in beacon chain")
		return NewBlockChainError(UnExpectedError, err)
	}

	return nil
}

func (bc *BlockChain) Stop() {

	Logger.log.Info("Blockchain Stop")

	if ShardSyncMode == common.FULL_SYNC_MODE {

		Logger.log.Info("Blockchain Stop, begin commit for fast sync mode")

		for i, shardChain := range bc.ShardChain {

			bc.ShardChain[i].insertLock.Lock()
			defer bc.ShardChain[i].insertLock.Unlock()

			shardBestState := shardChain.GetBestState()
			shardID := shardBestState.ShardID
			db := bc.GetShardChainDatabase(shardID)
			consensusTrieDB := shardBestState.consensusStateDB.Database().TrieDB()
			transactionTrieDB := shardBestState.transactionStateDB.Database().TrieDB()
			featureTrieDB := shardBestState.featureStateDB.Database().TrieDB()
			rewardTrieDB := shardBestState.rewardStateDB.Database().TrieDB()
			slashTrieDB := shardBestState.slashStateDB.Database().TrieDB()

			Logger.log.Infof("Blockchain Stop, start commit shard %+v, best height %+v in fast sync mode", i, shardBestState.ShardHeight)

			for _, view := range shardChain.GetAllView() {
				shardView := view.(*ShardBestState)
				sRH, err := GetShardRootsHashByBlockHash(db, shardID, shardView.BestBlockHash)
				if err != nil {
					Logger.log.Error("Blockchain Stop, GetShardRootsHashByBlockHash shardID %+v, blockhash %+v,  error: ", shardID, shardView.BestBlockHash, err)
					continue
				}
				if err := consensusTrieDB.Commit(sRH.ConsensusStateDBRootHash, false, nil); err != nil {
					Logger.log.Errorf("Blockchain Stop, Commit Consensus DB %+v, error %+v", sRH.ConsensusStateDBRootHash, err)
				}
				if err := transactionTrieDB.Commit(sRH.TransactionStateDBRootHash, false, nil); err != nil {
					Logger.log.Errorf("Blockchain Stop, Commit Transaction DB %+v, error %+v", sRH.TransactionStateDBRootHash, err)
				}
				if err := featureTrieDB.Commit(sRH.FeatureStateDBRootHash, false, nil); err != nil {
					Logger.log.Errorf("Blockchain Stop, Commit Feature DB %+v, error %+v", sRH.FeatureStateDBRootHash, err)
				}
				if err := rewardTrieDB.Commit(sRH.RewardStateDBRootHash, false, nil); err != nil {
					Logger.log.Errorf("Blockchain Stop, Commit Reward DB %+v, error %+v", sRH.RewardStateDBRootHash, err)
				}
				if err := slashTrieDB.Commit(sRH.SlashStateDBRootHash, false, nil); err != nil {
					Logger.log.Errorf("Blockchain Stop, Commit Slash DB %+v, error %+v", sRH.SlashStateDBRootHash, err)
				}
			}

			Logger.log.Infof("Blockchain Stop, finish commit shard %+v, best height %+v in fast sync mode", i, shardBestState.ShardHeight)

			finalizedBlock := shardChain.multiView.GetFinalView().GetBlock()
			err := rawdbv2.StoreLatestPivotBlock(db, shardID, *finalizedBlock.Hash())
			if err != nil {
				Logger.log.Errorf("StoreLatestPivotBlock, Shard %d, height %d, hash %+v, err %+v", shardID, finalizedBlock.GetHeight(), *finalizedBlock.Hash(), err)
			}

			for !bc.cacheConfig.triegc[shardID].Empty() {
				sRH := bc.cacheConfig.triegc[shardID].PopItem().(ShardRootHash)
				consensusTrieDB.Dereference(sRH.ConsensusStateDBRootHash)
				transactionTrieDB.Dereference(sRH.TransactionStateDBRootHash)
				featureTrieDB.Dereference(sRH.FeatureStateDBRootHash)
				rewardTrieDB.Dereference(sRH.RewardStateDBRootHash)
				slashTrieDB.Dereference(sRH.SlashStateDBRootHash)
			}

			if size, _ := consensusTrieDB.Size(); size != 0 {
				Logger.log.Error("Dangling consensus trie nodes after full cleanup")
			}
			if size, _ := transactionTrieDB.Size(); size != 0 {
				Logger.log.Error("Dangling transaction trie nodes after full cleanup")
			}
			if size, _ := featureTrieDB.Size(); size != 0 {
				Logger.log.Error("Dangling feature trie nodes after full cleanup")
			}
			if size, _ := rewardTrieDB.Size(); size != 0 {
				Logger.log.Error("Dangling reward trie nodes after full cleanup")
			}
			if size, _ := slashTrieDB.Size(); size != 0 {
				Logger.log.Error("Dangling slash trie nodes after full cleanup")
			}

			if bc.cacheConfig.trieJournalPath != nil {
				if path := bc.cacheConfig.trieJournalPath[int(shardBestState.ShardID)]; path != "" {
					// the below disk layer is just one => save one time is enough
					transactionTrieDB.SaveCache(path)
				}
			}
		}

		Logger.log.Info("Blockchain Stop, successfully commit for fast sync mode")
	}
}

func (blockchain *BlockChain) GetClonedBeaconBestState() (*BeaconBestState, error) {
	result := NewBeaconBestState()
	err := result.cloneBeaconBestStateFrom(blockchain.GetBeaconBestState())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAllShardBestState() map[byte]*ShardBestState {
	result := make(map[byte]*ShardBestState)
	for _, v := range blockchain.ShardChain {
		sidState := NewShardBestState()
		err := sidState.cloneShardBestStateFrom(blockchain.ShardChain[v.GetShardID()].GetBestState())
		if err != nil {
			return nil
		}
		result[byte(v.GetShardID())] = sidState
	}
	return result
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAShardBestState(shardID byte) (*ShardBestState, error) {
	result := NewShardBestState()
	err := result.cloneShardBestStateFrom(blockchain.ShardChain[int(shardID)].GetBestState())
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (blockchain *BlockChain) GetCurrentBeaconBlockHeight(shardID byte) uint64 {
	return blockchain.GetBeaconBestState().BestBlock.Header.Height
}

func (blockchain BlockChain) RandomCommitmentsProcess(usableInputCoins []coin.PlainCoin, randNum int, shardID byte, tokenID *common.Hash) (commitmentIndexs []uint64, myCommitmentIndexs []uint64, commitments [][]byte) {
	if int(shardID) >= common.MaxShardNumber {
		return nil, nil, nil
	}
	param := transaction.NewRandomCommitmentsProcessParam(usableInputCoins, randNum, blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB(), shardID, tokenID)
	return transaction.RandomCommitmentsProcess(param)
}

func (blockchain BlockChain) RandomCommitmentsAndPublicKeysProcess(numOutputs int, shardID byte, tokenID *common.Hash) ([]uint64, [][]byte, [][]byte, [][]byte, error) {
	if int(shardID) >= common.MaxShardNumber {
		return nil, nil, nil, nil, fmt.Errorf("shardID %v is out of range, maxShardNumber is %v", shardID, common.MaxShardNumber)
	}
	db := blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB()
	lenOTA, err := statedb.GetOTACoinLength(db, *tokenID, shardID)
	if err != nil || lenOTA == nil {
		return nil, nil, nil, nil, err
	}

	indices := make([]uint64, 0)
	publicKeys := make([][]byte, 0)
	commitments := make([][]byte, 0)
	assetTags := make([][]byte, 0)
	// these coins either all have asset tags or none does
	hasAssetTags := true
	for i := 0; i < numOutputs; i++ {
		idx, _ := common.RandBigIntMaxRange(lenOTA)
		coinBytes, err := statedb.GetOTACoinByIndex(db, *tokenID, idx.Uint64(), shardID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		coinDB := new(coin.CoinV2)
		if err := coinDB.SetBytes(coinBytes); err != nil {
			return nil, nil, nil, nil, err
		}

		publicKey := coinDB.GetPublicKey()
		// we do not use burned coins since they will reduce the privacy level of the transaction.
		if common.IsPublicKeyBurningAddress(publicKey.ToBytesS()) {
			i--
			continue
		}

		commitment := coinDB.GetCommitment()
		indices = append(indices, idx.Uint64())
		publicKeys = append(publicKeys, publicKey.ToBytesS())
		commitments = append(commitments, commitment.ToBytesS())

		if hasAssetTags {
			assetTag := coinDB.GetAssetTag()
			if assetTag != nil {
				assetTags = append(assetTags, assetTag.ToBytesS())
			} else {
				hasAssetTags = false
			}
		}
	}

	return indices, publicKeys, commitments, assetTags, nil
}

func (blockchain *BlockChain) GetActiveShardNumber() int {
	return blockchain.GetBeaconBestState().ActiveShards
}

func (blockchain *BlockChain) GetShardIDs() []int {
	shardIDs := []int{}
	for i := 0; i < blockchain.GetActiveShardNumber(); i++ {
		shardIDs = append(shardIDs, i)
	}
	return shardIDs
}

// -------------- Start of Blockchain retriever's implementation --------------
func (blockchain *BlockChain) SetIsBlockGenStarted(value bool) {
	blockchain.config.IsBlockGenStarted = value
}

func (blockchain *BlockChain) AddTxPool(txpool TxPool) {
	blockchain.config.TxPool = txpool
}

func (blockchain *BlockChain) AddTempTxPool(temptxpool TxPool) {
	blockchain.config.TempTxPool = temptxpool
}

func (blockchain *BlockChain) SetFeeEstimator(feeEstimator txpool.FeeEstimator, shardID byte) {
	if len(blockchain.config.FeeEstimator) == 0 {
		blockchain.config.FeeEstimator = make(map[byte]FeeEstimator)
	}

	blockchain.config.FeeEstimator[shardID] = feeEstimator
	for shardID := byte(0); int(shardID) < blockchain.GetBeaconBestState().ActiveShards; shardID++ {
		blockchain.ShardChain[shardID].TxsVerifier.UpdateFeeEstimator(feeEstimator)
	}
}

func (blockchain *BlockChain) InitChannelBlockchain(cRemovedTxs chan metadata.Transaction) {
	blockchain.config.CRemovedTxs = cRemovedTxs
}

// -------------- End of Blockchain retriever's implementation --------------

// -------------- Start of Blockchain BackUp And Restore --------------
func CalculateNumberOfByteToRead(amountBytes int) []byte {
	var result = make([]byte, 8)
	binary.LittleEndian.PutUint32(result, uint32(amountBytes))
	return result
}

func GetNumberOfByteToRead(value []byte) (int, error) {
	var result uint32
	err := binary.Read(bytes.NewBuffer(value), binary.LittleEndian, &result)
	if err != nil {
		return -1, err
	}
	return int(result), nil
}

func (blockchain *BlockChain) BackupShardChain(writer io.Writer, shardID byte) error {
	bestStateBytes, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		return err
	}
	shardBestState := &ShardBestState{}
	err = json.Unmarshal(bestStateBytes, shardBestState)
	bestShardHeight := shardBestState.ShardHeight
	var i uint64
	for i = 1; i < bestShardHeight; i++ {
		shardBlocks, err := blockchain.GetShardBlockByHeight(i, shardID)
		if err != nil {
			return err
		}
		var shardBlock *types.ShardBlock
		for _, v := range shardBlocks {
			shardBlock = v
		}
		data, err := json.Marshal(shardBlocks)
		if err != nil {
			return err
		}
		_, err = writer.Write(CalculateNumberOfByteToRead(len(data)))
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Shard %+v Block %+v", shardBlock.Header.ShardID, i)
		}
		if i == bestShardHeight-1 {
			Logger.log.Infof("Finish Backup Shard %+v with Block %+v", shardBlock.Header.ShardID, i)
		}
	}
	return nil
}

func (blockchain *BlockChain) BackupBeaconChain(writer io.Writer) error {
	bestStateBytes, err := rawdbv2.GetBeaconViews(blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	beaconBestState := &BeaconBestState{}
	err = json.Unmarshal(bestStateBytes, beaconBestState)
	bestBeaconHeight := beaconBestState.BeaconHeight
	var i uint64
	for i = 1; i < bestBeaconHeight; i++ {
		beaconBlocks, err := blockchain.GetBeaconBlockByHeight(i)
		if err != nil {
			return err
		}
		beaconBlock := beaconBlocks[0]
		data, err := json.Marshal(beaconBlock)
		if err != nil {
			return err
		}
		numOfByteToRead := CalculateNumberOfByteToRead(len(data))
		_, err = writer.Write(numOfByteToRead)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Beacon Block %+v", i)
		}
		if i == bestBeaconHeight-1 {
			Logger.log.Infof("Finish Backup Beacon with Block %+v", i)
		}
	}
	return nil
}

//TODO:
// current implement: backup all view data
// Optimize: backup view -> backup view hash instead of view
// restore: get view from hash and create new view, then insert into multiview
/*
Backup all BeaconView into Database
*/
func (blockchain *BlockChain) BackupBeaconViews(db incdb.KeyValueWriter) error {
	allViews := []*BeaconBestState{}
	for _, v := range blockchain.BeaconChain.multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*BeaconBestState))
	}
	b, _ := json.Marshal(allViews)
	return rawdbv2.StoreBeaconViews(db, b)
}

/*
Restart all BeaconView from Database
*/
func (blockchain *BlockChain) RestoreBeaconViews() error {
	allViews := []*BeaconBestState{}
	bcDB := blockchain.GetBeaconChainDatabase()
	b, err := rawdbv2.GetBeaconViews(bcDB)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &allViews)
	if err != nil {
		return err
	}
	sID := []int{}
	for i := 0; i < config.Param().ActiveShards; i++ {
		sID = append(sID, i)
	}

	blockchain.BeaconChain.multiView.Reset()
	for _, v := range allViews {
		includePdexv3 := false
		if v.BeaconHeight >= config.Param().PDexParams.Pdexv3BreakPointHeight {
			includePdexv3 = true
		}
		if err := v.RestoreBeaconViewStateFromHash(blockchain, true, includePdexv3); err != nil {
			return NewBlockChainError(BeaconError, err)
		}
		if v.NumberOfFixedShardBlockValidator == 0 {
			v.NumberOfFixedShardBlockValidator = config.Param().CommitteeSize.NumberOfFixedShardBlockValidator
		}
		v.pdeStates, err = pdex.InitStatesFromDB(v.featureStateDB, v.BeaconHeight)
		if err != nil {
			return err
		}
		if v.NumberOfShardBlock == nil || len(v.NumberOfShardBlock) == 0 {
			v.NumberOfShardBlock = make(map[byte]uint)
			for i := 0; i < v.ActiveShards; i++ {
				shardID := byte(i)
				v.NumberOfShardBlock[shardID] = 0
			}
		}
		// finish reproduce
		if !blockchain.BeaconChain.multiView.AddView(v) {
			panic("Restart beacon views fail")
		}
	}
	for _, beaconState := range allViews {
		if beaconState.missingSignatureCounter == nil {
			block := beaconState.BestBlock
			err = beaconState.initMissingSignatureCounter(blockchain, &block)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/*
Backup shard views
*/
func (blockchain *BlockChain) BackupShardViews(db incdb.KeyValueWriter, shardID byte) error {
	allViews := []*ShardBestState{}
	for _, v := range blockchain.ShardChain[shardID].multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*ShardBestState))
	}
	return rawdbv2.StoreShardBestState(db, shardID, allViews)
}

/*
Restart all BeaconView from Database
*/
var (
	testGetFlatFileIndex        = float64(0)
	testRestoreOneBlock         = float64(0)
	testCommitMemOneBlock       = float64(0)
	testCommitDiskOneBlock      = float64(0)
	testGetRootOneBlock         = float64(0)
	testGetSObjOneBlock         = float64(0)
	testDeserializeSObjOneBlock = float64(0)
	testReadFile                = float64(0)
)

func (blockchain *BlockChain) RestoreShardViews(shardID byte) error {

	allViews := []*ShardBestState{}

	b, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		Logger.log.Errorf("RestoreShardBestState shardID %+v, GetShardBestState error %+v", shardID, err)
		return err
	}

	if err = json.Unmarshal(b, &allViews); err != nil {
		Logger.log.Errorf("RestoreShardBestState shardID %+v, Unmarshal views error %+v", shardID, err)
		return err
	}

	blockchain.ShardChain[shardID].multiView.Reset()
	isRepair := false
	for _, v := range allViews {
		block, _, err := blockchain.GetShardBlockByHash(v.BestBlockHash)
		if err != nil || block == nil {
			Logger.log.Errorf("RestoreShardBestState shardID %+v, GetShardBlockByHash error %+v", shardID, err)
			continue
		}
		v.BestBlock = block
		err = v.InitStateRootHash(blockchain.GetShardChainDatabase(shardID), blockchain, isRepair)
		if err != nil {
			now := time.Now()
			Logger.log.Errorf("RestoreShardBestState shardID %+v, InitStateRootHash error %+v", shardID, err)
			if err := blockchain.RepairShardViewStateDB(shardID, allViews); err != nil {
				Logger.log.Errorf("RepairShardViewStateDB shardID %+v, error %+v", shardID, err)
				return err
			}
			isRepair = true
			Logger.log.Infof("Benchmark Time RestoreShardBestState \n"+
				"Commit Mem %.4f \n"+
				"Commit Disk %.4f \n"+
				"Get Root %.4f \n"+
				"Repair State %.4f \n"+
				"Read File Time: %.4f \n"+
				"Deserialize State Object: %.4f \n"+
				"Restore State Object: %.4f \n"+
				"Deserialize Byte 1: %.4f \n"+
				"Deserialize Byte 2: %.4f \n"+
				"Read file by index: %.4f \n"+
				"Total: %.2f \n",
				testCommitMemOneBlock,
				testCommitDiskOneBlock,
				testGetRootOneBlock,
				testRestoreOneBlock,
				testGetFlatFileIndex,
				testDeserializeSObjOneBlock,
				testGetSObjOneBlock,
				statedb.TestBenchmarkDeserialize1,
				statedb.TestBenchmarkDeserialize2,
				testReadFile,
				time.Since(now).Seconds())
		}

		version := committeestate.VersionByBeaconHeight(v.BeaconHeight,
			config.Param().ConsensusParam.StakingFlowV2Height,
			config.Param().ConsensusParam.StakingFlowV3Height,
		)
		v.shardCommitteeState = InitShardCommitteeState(version,
			v.consensusStateDB,
			v.ShardHeight, v.ShardID,
			block, blockchain)
		err = v.tryUpgradeCommitteeState(blockchain)
		if err != nil {
			panic(err)
		}
		if v.BeaconHeight > config.Param().ConsensusParam.BlockProducingV3Height {
			if err := v.checkAndUpgradeStakingFlowV3Config(); err != nil {
				return err
			}
		}
		if v.NumberOfFixedShardBlockValidator == 0 {
			v.NumberOfFixedShardBlockValidator = config.Param().CommitteeSize.NumberOfFixedShardBlockValidator
		}
		if !blockchain.ShardChain[shardID].multiView.AddView(v) {
			return errors.New("Restart shard views fail")
		}
	}

	if len(blockchain.ShardChain[shardID].multiView.GetAllViewsWithBFS()) == 0 {
		return errors.New("unable to restore any shard best state")
	}

	return nil
}

func (blockchain *BlockChain) RepairShardViewStateDB(
	shardID byte,
	views []*ShardBestState,
) error {

	start := time.Now()
	Logger.log.Infof("Repairing Shard Best View statedb...")

	db := blockchain.GetShardChainDatabase(shardID)
	flatFileManager := blockchain.config.FlatFileManager[int(shardID)]

	pivotBlock, err := blockchain.GetPivotBlock(shardID)
	if err != nil {
		return err
	}

	simulateMultiViews := multiview.NewMultiView()
	for _, v := range views {
		simulateMultiViews.AddView(v)
	}
	finalView := simulateMultiViews.GetFinalView()
	finalViewHash := finalView.GetHash()
	finalBlock, _, err := blockchain.GetShardBlockByHashWithShardID(*finalViewHash, shardID)
	if err != nil {
		return err
	}

	Logger.log.Infof("Repair State, Shard %d, Latest Pivot (Commit) Block %d, Latest Final Block %d",
		shardID, pivotBlock.GetHeight(), finalBlock.GetHeight())

	stateDBs, err := blockchain.tryRepairStateFromPivotToFinal(
		db, flatFileManager,
		shardID,
		pivotBlock, finalBlock,
	)
	if err != nil {
		return err
	}

	for _, view := range views {

		restoreFromBestView, err := blockchain.restoreOldBlocks(*view.GetHash(), *finalBlock.Hash())
		if err != nil {
			return err
		}
		viewStateDBs := make([]*statedb.StateDB, 5)
		for i := range stateDBs {
			viewStateDBs[i] = stateDBs[i].Copy()
		}

		if err := repairStateDB(
			blockchain,
			shardID,
			viewStateDBs,
			flatFileManager,
			db,
			restoreFromBestView,
		); err != nil {
			return err
		}

		view.consensusStateDB = viewStateDBs[SHARD_CONSENSUS_STATEDB]
		if calculatedRoot, _ := view.consensusStateDB.IntermediateRoot(true); calculatedRoot != view.ConsensusStateDBRootHash {
			return fmt.Errorf("Repair State Error, expect consensus root hash %+v, got %+v", view.ConsensusStateDBRootHash, calculatedRoot)
		}
		view.transactionStateDB = viewStateDBs[SHARD_TRANSACTION_STATEDB]
		if calculatedRoot, _ := view.transactionStateDB.IntermediateRoot(true); calculatedRoot != view.TransactionStateDBRootHash {
			return fmt.Errorf("Repair State Error, expect transaction root hash %+v, got %+v", view.TransactionStateDBRootHash, calculatedRoot)
		}
		view.featureStateDB = viewStateDBs[SHARD_FEATURE_STATEDB]
		if calculatedRoot, _ := view.featureStateDB.IntermediateRoot(true); calculatedRoot != view.FeatureStateDBRootHash {
			return fmt.Errorf("Repair State Error, expect feature root hash %+v, got %+v", view.FeatureStateDBRootHash, calculatedRoot)
		}
		view.rewardStateDB = viewStateDBs[SHARD_REWARD_STATEDB]
		if calculatedRoot, _ := view.rewardStateDB.IntermediateRoot(true); calculatedRoot != view.RewardStateDBRootHash {
			return fmt.Errorf("Repair State Error, expect reward root hash %+v, got %+v", view.RewardStateDBRootHash, calculatedRoot)
		}
		view.slashStateDB = viewStateDBs[SHARD_SLASH_STATEDB]
		if calculatedRoot, _ := view.slashStateDB.IntermediateRoot(true); calculatedRoot != view.SlashStateDBRootHash {
			return fmt.Errorf("Repair State Error, expect slash root hash %+v, got %+v", view.SlashStateDBRootHash, calculatedRoot)
		}
	}

	Logger.log.Infof("Finish Repair State, Shard %d, time elapsed %.3f second", shardID, time.Now().Sub(start).Seconds())

	return nil
}

func (blockchain *BlockChain) tryRepairStateFromPivotToFinal(
	db incdb.Database, flatFileManager *flatfile.FlatFileManager,
	shardID byte,
	pivotBlock, finalBlock *types.ShardBlock) ([]*statedb.StateDB, error) {

	tempShardHash := common.Hash{}
	stateDBs := make([]*statedb.StateDB, 5)
	isRepairFromPivot := false

	// pivot block is also a finalized block => pivot block only <= final block
	if pivotBlock.GetHeight() < finalBlock.GetHeight() {
		tempShardHash = pivotBlock.Header.Hash()
		isRepairFromPivot = true
	} else {
		tempShardHash = finalBlock.Header.Hash()
	}

	sRH, err := GetShardRootsHashByBlockHash(db, shardID, tempShardHash)
	if err != nil {
		return stateDBs, err
	}
	dbWarper := statedb.NewDatabaseAccessWrapperWithConfig(ShardSyncMode, db,
		blockchain.cacheConfig.trieJournalPath[int(shardID)], blockchain.cacheConfig.trieJournalCacheSize)
	stateDBs[SHARD_CONSENSUS_STATEDB], err = statedb.NewWithPrefixTrie(sRH.ConsensusStateDBRootHash, dbWarper)
	if err != nil {
		return stateDBs, err
	}
	stateDBs[SHARD_TRANSACTION_STATEDB], err = statedb.NewWithPrefixTrie(sRH.TransactionStateDBRootHash, dbWarper)
	if err != nil {
		return stateDBs, err
	}
	stateDBs[SHARD_FEATURE_STATEDB], err = statedb.NewWithPrefixTrie(sRH.FeatureStateDBRootHash, dbWarper)
	if err != nil {
		return stateDBs, err
	}
	stateDBs[SHARD_REWARD_STATEDB], err = statedb.NewWithPrefixTrie(sRH.RewardStateDBRootHash, dbWarper)
	if err != nil {
		return stateDBs, err
	}
	stateDBs[SHARD_SLASH_STATEDB], err = statedb.NewWithPrefixTrie(sRH.SlashStateDBRootHash, dbWarper)
	if err != nil {
		return stateDBs, err
	}

	if isRepairFromPivot {
		restoreFromFinalize, err := blockchain.restoreOldBlocks(*finalBlock.Hash(), *pivotBlock.Hash())
		if err != nil {
			return stateDBs, err
		}
		if err := repairStateDB(
			blockchain,
			shardID,
			stateDBs,
			flatFileManager,
			db,
			restoreFromFinalize,
		); err != nil {
			return stateDBs, err
		}
	}

	return stateDBs, nil
}

func (blockchain *BlockChain) restoreOldBlocks(head, pivot common.Hash) ([]common.Hash, error) {

	blockHashes := []common.Hash{head}

	for blockHashes[len(blockHashes)-1].String() != pivot.String() {
		block, _, err := blockchain.GetShardBlockByHash(blockHashes[len(blockHashes)-1])
		if err != nil {
			return nil, err
		}
		blockHashes = append(blockHashes, block.GetPrevHash())
	}

	return blockHashes[:len(blockHashes)-1], nil
}

func repairStateDB(
	bc *BlockChain,
	shardID byte,
	stateDBs []*statedb.StateDB,
	flatFileManager *flatfile.FlatFileManager,
	db incdb.Database,
	restore []common.Hash) error {

	if len(restore) == 0 {
		return nil
	}

	for len(restore) != 0 {
		startTime1 := time.Now()
		nextBlock := restore[len(restore)-1]

		Logger.log.Debugf("Repair State, Shard %+v, restore next block %+v, remain %+v", shardID, nextBlock, len(restore))

		wantSRH, err := GetShardRootsHashByBlockHash(db, shardID, nextBlock)
		if err != nil {
			return err
		}

		if isCorruptState(wantSRH, statedb.NewDatabaseAccessWarper(db)) {

			startTime5 := time.Now()

			allStateObjects, flatFileIndexes, err := GetStateObjectFromFlatFile(
				stateDBs,
				flatFileManager,
				db,
				nextBlock,
			)
			if err != nil {
				return err
			}
			testGetSObjOneBlock += time.Since(startTime5).Seconds()
			Logger.log.Infof("Benchmark Time Get State One Block %f", time.Since(startTime5).Seconds())

			gotSRH := make([]common.Hash, 5)
			startTime2 := time.Now()
			for i := range allStateObjects {
				stateObjects := allStateObjects[i]
				stateDB := stateDBs[i]

				for objKey, obj := range stateObjects {
					if err := stateDB.SetStateObject(obj.GetType(), objKey, obj.GetValue()); err != nil {
						return err
					}
					if obj.IsDeleted() {
						stateDB.MarkDeleteStateObject(obj.GetType(), objKey)
					}
				}

				rootHash, _, err := stateDB.Commit(true)
				if err != nil {
					return err
				}
				gotSRH[i] = rootHash
			}
			testCommitMemOneBlock += time.Since(startTime2).Seconds()
			Logger.log.Infof("Benchmark Time Mem Commit One Block %f", time.Since(startTime2).Seconds())

			blockToCommit, _, err := bc.GetShardBlockByHash(nextBlock)
			if err != nil {
				return err
			}
			batch := db.NewBatch()

			startTime3 := time.Now()

			if err := commitTrieToDisk(
				batch,
				shardID,
				stateDBs,
				gotSRH,
				blockToCommit,
				flatFileManager,
				flatFileIndexes,
			); err != nil {
				return err
			}

			if err := batch.Write(); err != nil {
				return err
			}
			testCommitDiskOneBlock += time.Since(startTime3).Seconds()
			Logger.log.Infof("Benchmark Time Disk Commit One Block %f", time.Since(startTime3).Seconds())

			startTime4 := time.Now()
			if calculatedRoot, _ := stateDBs[SHARD_CONSENSUS_STATEDB].IntermediateRoot(true); calculatedRoot != wantSRH.ConsensusStateDBRootHash {
				return fmt.Errorf("Repair State Error, Shard %d, height %d, hash %+v, expect consensus root hash %+v, got %+v",
					shardID, blockToCommit.GetHeight(), *blockToCommit.Hash(), wantSRH.ConsensusStateDBRootHash, calculatedRoot)
			}
			if calculatedRoot, _ := stateDBs[SHARD_TRANSACTION_STATEDB].IntermediateRoot(true); calculatedRoot != wantSRH.TransactionStateDBRootHash {
				return fmt.Errorf("Repair State Error, Shard %d, height %d, hash %+v, expect transaction root hash %+v, got %+v",
					shardID, blockToCommit.GetHeight(), *blockToCommit.Hash(), wantSRH.TransactionStateDBRootHash, calculatedRoot)
			}
			if calculatedRoot, _ := stateDBs[SHARD_FEATURE_STATEDB].IntermediateRoot(true); calculatedRoot != wantSRH.FeatureStateDBRootHash {
				return fmt.Errorf("Repair State Error, Shard %d, height %d, hash %+v, expect feature root hash %+v, got %+v",
					shardID, blockToCommit.GetHeight(), *blockToCommit.Hash(), wantSRH.FeatureStateDBRootHash, calculatedRoot)
			}
			if calculatedRoot, _ := stateDBs[SHARD_REWARD_STATEDB].IntermediateRoot(true); calculatedRoot != wantSRH.RewardStateDBRootHash {
				return fmt.Errorf("Repair State Error, Shard %d, height %d, hash %+v, expect reward root hash %+v, got %+v",
					shardID, blockToCommit.GetHeight(), *blockToCommit.Hash(), wantSRH.RewardStateDBRootHash, calculatedRoot)
			}
			if calculatedRoot, _ := stateDBs[SHARD_SLASH_STATEDB].IntermediateRoot(true); calculatedRoot != wantSRH.SlashStateDBRootHash {
				return fmt.Errorf("Repair State Error, Shard %d, height %d, hash %+v, expect slash root hash %+v, got %+v",
					shardID, blockToCommit.GetHeight(), *blockToCommit.Hash(), wantSRH.SlashStateDBRootHash, calculatedRoot)
			}
			testGetRootOneBlock += time.Since(startTime4).Seconds()
			Logger.log.Infof("Benchmark Time Get Root One Block %f", time.Since(startTime4).Seconds())
		}

		restore = restore[:len(restore)-1]
		testRestoreOneBlock += time.Since(startTime1).Seconds()
		Logger.log.Infof("Benchmark Time Restore One Block %f", time.Since(startTime1).Seconds())
	}

	return nil
}

func isCorruptState(sRH *ShardRootHash, db statedb.DatabaseAccessWarper) bool {

	if _, err := statedb.NewWithPrefixTrie(sRH.ConsensusStateDBRootHash, db); err != nil {
		return true
	}
	if _, err := statedb.NewWithPrefixTrie(sRH.TransactionStateDBRootHash, db); err != nil {
		return true
	}
	if _, err := statedb.NewWithPrefixTrie(sRH.FeatureStateDBRootHash, db); err != nil {
		return true
	}
	if _, err := statedb.NewWithPrefixTrie(sRH.RewardStateDBRootHash, db); err != nil {
		return true
	}
	if _, err := statedb.NewWithPrefixTrie(sRH.SlashStateDBRootHash, db); err != nil {
		return true
	}

	return false
}

func (blockchain *BlockChain) GetShardStakingTx(shardID byte, beaconHeight uint64) (map[string]string, error) {
	//build staking tx
	beaconConsensusRootHash, err := blockchain.GetBeaconConsensusRootHash(blockchain.GetBeaconBestState(), beaconHeight)
	if err != nil {
		Logger.log.Error("Cannot restore shard, beacon not ready!")
		return nil, err
	}

	beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
	if err != nil {
		Logger.log.Error("Cannot restore shard, beacon not ready!")
		return nil, err
	}
	mapStakingTx, err := beaconConsensusStateDB.GetAllStakingTX(blockchain.GetShardIDs())

	if err != nil {
		fmt.Println(err)
		panic("Something wrong when retrieve mapStakingTx")
	}

	sdb := blockchain.GetShardChainDatabase(byte(shardID))
	shardStakingTx := map[string]string{}
	for _, stakingtx := range mapStakingTx {
		if stakingtx != common.HashH([]byte{0}).String() {
			stakingTxHash, _ := common.Hash{}.NewHashFromStr(stakingtx)
			blockHash, txindex, err := rawdbv2.GetTransactionByHash(sdb, *stakingTxHash)
			if err != nil { //no transaction in this node
				continue
			}
			shardBlockBytes, err := rawdbv2.GetShardBlockByHash(sdb, blockHash)
			if err != nil { //no transaction in this node
				panic("Have transaction but cannot found block")
			}
			shardBlock := types.NewShardBlock()
			err = json.Unmarshal(shardBlockBytes, shardBlock)
			if err != nil {
				panic("Cannot unmarshal shardblock")
			}
			if shardBlock.GetShardID() != int(shardID) {
				continue
			}
			txData := shardBlock.Body.Transactions[txindex]
			committeePk := txData.GetMetadata().(*metadata.StakingMetadata).CommitteePublicKey
			shardStakingTx[committeePk] = stakingtx
		}
	}
	return shardStakingTx, nil
}

// -------------- End of Blockchain BackUp And Restore --------------

func (blockchain *BlockChain) GetWantedShard(isBeaconCommittee bool) map[byte]struct{} {
	res := map[byte]struct{}{}
	if isBeaconCommittee {
		for sID := byte(0); sID < byte(config.Param().ActiveShards); sID++ {
			res[sID] = struct{}{}
		}
	} else {
		blockchain.config.relayShardLck.Lock()
		for _, sID := range blockchain.config.RelayShards {
			res[sID] = struct{}{}
		}
		blockchain.config.relayShardLck.Unlock()
	}
	return res
}

// GetConfig returns blockchain's config
func (blockchain *BlockChain) GetConfig() *Config {
	return &blockchain.config
}

func (blockchain *BlockChain) GetBeaconChainDatabase() incdb.Database {
	return blockchain.config.DataBase[common.BeaconChainID]
}

func (blockchain *BlockChain) GetShardChainDatabase(shardID byte) incdb.Database {
	return blockchain.config.DataBase[int(shardID)]
}

func (blockchain *BlockChain) GetBeaconViewStateDataFromBlockHash(
	blockHash common.Hash, includeCommittee, includePdexv3 bool,
) (*BeaconBestState, error) {
	v, ok := blockchain.beaconViewCache.Get(blockHash)
	if ok {
		return v.(*BeaconBestState), nil
	}
	bcDB := blockchain.GetBeaconChainDatabase()
	rootHash, err := rawdbv2.GetBeaconRootsHash(bcDB, blockHash)
	if err != nil {
		return nil, err
	}
	bRH := &BeaconRootHash{}
	err = json.Unmarshal(rootHash, bRH)
	if err != nil {
		return nil, err
	}

	beaconView := &BeaconBestState{
		BestBlockHash:            blockHash,
		ActiveShards:             config.Param().ActiveShards, //we assume active shard not change (if not, we must store active shard in db)
		ConsensusStateDBRootHash: bRH.ConsensusStateDBRootHash,
		FeatureStateDBRootHash:   bRH.FeatureStateDBRootHash,
		RewardStateDBRootHash:    bRH.RewardStateDBRootHash,
		SlashStateDBRootHash:     bRH.SlashStateDBRootHash,
	}

	// @NOTICE: beaconBestState.NumberOfShardBlock this field is initialized with zero value only
	// DO NOT use data beaconBestState.NumberOfShardBlock when init from this process
	beaconView.NumberOfShardBlock = make(map[byte]uint)
	for i := 0; i < beaconView.ActiveShards; i++ {
		shardID := byte(i)
		beaconView.NumberOfShardBlock[shardID] = 0
	}
	err = beaconView.RestoreBeaconViewStateFromHash(blockchain, includeCommittee, includePdexv3)
	if err != nil {
		Logger.log.Error(err)
	}
	return beaconView, err
}

func (blockchain *BlockChain) IsAfterNewZKPCheckPoint(beaconHeight uint64) bool {
	if beaconHeight == 0 {
		beaconHeight = blockchain.GetBeaconBestState().GetHeight()
	}

	return beaconHeight >= config.Param().BCHeightBreakPointNewZKP
}

func (blockchain *BlockChain) IsAfterPrivacyV2CheckPoint(beaconHeight uint64) bool {
	if beaconHeight == 0 {
		beaconHeight = blockchain.GetBeaconBestState().GetHeight()
	}

	return beaconHeight >= config.Param().BCHeightBreakPointPrivacyV2
}

func (blockchain *BlockChain) IsAfterPdexv3CheckPoint(beaconHeight uint64) bool {
	if beaconHeight == 0 {
		beaconHeight = blockchain.GetBeaconBestState().GetHeight()
	}
	return beaconHeight >= config.Param().PDexParams.Pdexv3BreakPointHeight
}

func (s *BlockChain) AddRelayShard(sid int) error {
	s.config.relayShardLck.Lock()
	for _, shard := range s.config.RelayShards {
		if shard == byte(sid) {
			s.config.relayShardLck.Unlock()
			return errors.New("already relay this shard" + strconv.Itoa(sid))
		}
	}
	s.config.RelayShards = append(s.config.RelayShards, byte(sid))
	s.config.relayShardLck.Unlock()
	return nil
}

func (s *BlockChain) RemoveRelayShard(sid int) {
	s.config.relayShardLck.Lock()
	for idx, shard := range s.config.RelayShards {
		if shard == byte(sid) {
			s.config.RelayShards = append(s.config.RelayShards[:idx], s.config.RelayShards[idx+1:]...)
			break
		}
	}
	s.config.relayShardLck.Unlock()
	return
}

// GetEpochLength return the current length of epoch
// it depends on current final view height
func (bc *BlockChain) GetCurrentEpochLength(beaconHeight uint64) uint64 {
	params := config.Param()
	if params.EpochParam.EpochV2BreakPoint == 0 {
		return params.EpochParam.NumberOfBlockInEpoch
	}
	changeEpochBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	if beaconHeight > changeEpochBreakPoint {
		return params.EpochParam.NumberOfBlockInEpochV2
	} else {
		return params.EpochParam.NumberOfBlockInEpoch
	}
}

func (bc *BlockChain) GetEpochByHeight(beaconHeight uint64) uint64 {
	params := config.Param()
	totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	newEpochBlockHeight := totalBlockBeforeBreakPoint + 1
	if beaconHeight < newEpochBlockHeight {
		if beaconHeight%params.EpochParam.NumberOfBlockInEpoch == 0 {
			return beaconHeight / params.EpochParam.NumberOfBlockInEpoch
		} else {
			return beaconHeight/params.EpochParam.NumberOfBlockInEpoch + 1
		}
	} else {
		newEpochBlocks := beaconHeight - totalBlockBeforeBreakPoint
		numberOfNewEpochs := newEpochBlocks / params.EpochParam.NumberOfBlockInEpochV2
		if newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2 != 0 {
			numberOfNewEpochs++
		}
		return (totalBlockBeforeBreakPoint / params.EpochParam.NumberOfBlockInEpoch) +
			numberOfNewEpochs
	}
}

func (bc *BlockChain) GetEpochNextHeight(beaconHeight uint64) (uint64, bool) {
	beaconHeight++
	return bc.getEpochAndIsFistHeightInEpoch(beaconHeight)
}

func (bc *BlockChain) getEpochAndIsFistHeightInEpoch(beaconHeight uint64) (uint64, bool) {
	params := config.Param()
	totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	newEpochBlockHeight := totalBlockBeforeBreakPoint + 1
	if beaconHeight < newEpochBlockHeight {
		newEpoch := beaconHeight/params.EpochParam.NumberOfBlockInEpoch + 1
		if beaconHeight%params.EpochParam.NumberOfBlockInEpoch == 1 {
			if beaconHeight == 1 {
				return newEpoch, false
			} else {
				return newEpoch, true
			}
		} else {
			if beaconHeight%params.EpochParam.NumberOfBlockInEpoch == 0 {
				return newEpoch - 1, false
			} else {
				return newEpoch, false
			}
		}
	} else {
		newEpochBlocks := beaconHeight - totalBlockBeforeBreakPoint
		numberOfNewEpochs := newEpochBlocks / params.EpochParam.NumberOfBlockInEpochV2
		numberOfOldEpochs := params.EpochParam.EpochV2BreakPoint - 1
		if newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2 != 0 {
			numberOfNewEpochs++
		}
		if newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2 == 1 {
			return numberOfOldEpochs + numberOfNewEpochs, true
		} else {
			return numberOfOldEpochs + numberOfNewEpochs, false
		}
	}
}

func (bc *BlockChain) IsFirstBeaconHeightInEpoch(beaconHeight uint64) bool {
	_, ok := bc.getEpochAndIsFistHeightInEpoch(beaconHeight)
	return ok
}

func (bc *BlockChain) IsLastBeaconHeightInEpoch(beaconHeight uint64) bool {
	_, ok := bc.getEpochAndIsFistHeightInEpoch(beaconHeight + 1)
	return ok
}

func (bc *BlockChain) GetRandomTimeInEpoch(epoch uint64) uint64 {
	params := config.Param()
	if epoch < params.EpochParam.EpochV2BreakPoint {
		return (epoch-1)*params.EpochParam.NumberOfBlockInEpoch + params.EpochParam.RandomTime
	} else {
		totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
		numberOfNewEpoch := epoch - params.EpochParam.EpochV2BreakPoint
		beaconHeightRandomTimeAfterBreakPoint := numberOfNewEpoch*params.EpochParam.NumberOfBlockInEpochV2 + params.EpochParam.RandomTimeV2
		res := totalBlockBeforeBreakPoint + beaconHeightRandomTimeAfterBreakPoint
		return res
	}
}

func (bc *BlockChain) GetFirstBeaconHeightInEpoch(epoch uint64) uint64 {
	params := config.Param()
	if epoch < params.EpochParam.EpochV2BreakPoint {
		return (epoch-1)*params.EpochParam.NumberOfBlockInEpoch + 1
	} else {
		totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
		numberOfNewEpoch := epoch - params.EpochParam.EpochV2BreakPoint
		lastBeaconHeightAfterBreakPoint := numberOfNewEpoch*params.EpochParam.NumberOfBlockInEpochV2 + 1
		return totalBlockBeforeBreakPoint + lastBeaconHeightAfterBreakPoint
	}
}

func (bc *BlockChain) GetLastBeaconHeightInEpoch(epoch uint64) uint64 {
	params := config.Param()
	if epoch < params.EpochParam.EpochV2BreakPoint {
		return epoch * params.EpochParam.NumberOfBlockInEpoch
	} else {
		totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
		numberOfNewEpoch := epoch - params.EpochParam.EpochV2BreakPoint + 1
		lastBeaconHeightAfterBreakPoint := numberOfNewEpoch * params.EpochParam.NumberOfBlockInEpochV2
		return totalBlockBeforeBreakPoint + lastBeaconHeightAfterBreakPoint
	}
}

func (bc *BlockChain) GetBeaconBlockOrderInEpoch(beaconHeight uint64) (uint64, uint64) {
	params := config.Param()
	totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	if beaconHeight < totalBlockBeforeBreakPoint {
		return beaconHeight % params.EpochParam.NumberOfBlockInEpoch, params.EpochParam.NumberOfBlockInEpoch - beaconHeight%params.EpochParam.NumberOfBlockInEpoch
	} else {
		newEpochBlocks := beaconHeight - totalBlockBeforeBreakPoint
		return newEpochBlocks % params.EpochParam.NumberOfBlockInEpochV2, params.EpochParam.NumberOfBlockInEpochV2 - newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2
	}
}

func (bc *BlockChain) IsGreaterThanRandomTime(beaconHeight uint64) bool {
	params := config.Param()
	totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	if beaconHeight < totalBlockBeforeBreakPoint {
		return beaconHeight%params.EpochParam.NumberOfBlockInEpoch > params.EpochParam.RandomTime
	} else {
		newEpochBlocks := beaconHeight - totalBlockBeforeBreakPoint
		return newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2 > params.EpochParam.RandomTimeV2
	}
}

func (bc *BlockChain) IsEqualToRandomTime(beaconHeight uint64) bool {
	params := config.Param()
	totalBlockBeforeBreakPoint := params.EpochParam.NumberOfBlockInEpoch * (params.EpochParam.EpochV2BreakPoint - 1)
	if beaconHeight < totalBlockBeforeBreakPoint {
		return beaconHeight%params.EpochParam.NumberOfBlockInEpoch == params.EpochParam.RandomTime
	} else {
		newEpochBlocks := beaconHeight - totalBlockBeforeBreakPoint
		return newEpochBlocks%params.EpochParam.NumberOfBlockInEpochV2 == params.EpochParam.RandomTimeV2
	}
}

func (blockchain *BlockChain) getShardCommitteeFromBeaconHash(
	hash common.Hash, shardID byte,
) (
	[]incognitokey.CommitteePublicKey, error,
) {
	committees, err := blockchain.getShardCommitteeForBlockProducing(hash, shardID)
	if err != nil {
		return []incognitokey.CommitteePublicKey{}, err
	}
	return committees, nil
}

func (blockchain *BlockChain) getShardCommitteeForBlockProducing(
	hash common.Hash, shardID byte,
) ([]incognitokey.CommitteePublicKey, error) {
	committees := []incognitokey.CommitteePublicKey{}
	res, has := blockchain.BeaconChain.committeesInfoCache.Get(getCommitteeCacheKey(hash, shardID))
	if !has {
		bRH, err := GetBeaconRootsHashByBlockHash(blockchain.GetBeaconChainDatabase(), hash)
		if err != nil {
			return committees, err
		}

		stateDB, err := statedb.NewWithPrefixTrie(
			bRH.ConsensusStateDBRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
		if err != nil {
			return committees, err
		}
		committees = statedb.GetOneShardCommittee(stateDB, shardID)

		blockchain.BeaconChain.committeesInfoCache.Add(getCommitteeCacheKey(hash, shardID), committees)
	} else {
		committees = res.([]incognitokey.CommitteePublicKey)
	}

	return committees, nil
}

// AddFinishedSyncValidators add finishedSyncValidators from message to all current beacon views
func (blockchain *BlockChain) AddFinishedSyncValidators(committeePublicKeys []string, signatures [][]byte, shardID byte) {
	validCommitteePublicKeys := verifyFinishedSyncValidatorsSign(committeePublicKeys, signatures)
	bestView := blockchain.BeaconChain.multiView.GetBestView().(*BeaconBestState)
	syncPool, _ := incognitokey.CommitteeKeyListToString(bestView.beaconCommitteeState.GetSyncingValidators()[shardID])
	finishsync.DefaultFinishSyncMsgPool.AddFinishedSyncValidators(
		validCommitteePublicKeys,
		syncPool,
		shardID,
	)

}

func verifyFinishedSyncValidatorsSign(committeePublicKeys []string, signatures [][]byte) []string {
	committeePublicKeyStructs, _ := incognitokey.CommitteeBase58KeyListToStruct(committeePublicKeys)
	validFinishedSyncValidators := []string{}
	for i, key := range committeePublicKeyStructs {
		isValid, err := bridgesig.Verify(key.MiningPubKey[common.BridgeConsensus], []byte(wire.CmdMsgFinishSync), signatures[i])
		if err != nil {
			Logger.log.Errorf("Verify finish Sync Validator Sign failed, err", committeePublicKeys[i], signatures[i], err)
			continue
		}
		if !isValid {
			Logger.log.Errorf("Verify finish Sync Validator Sign failed", committeePublicKeys[i], signatures[i])
			continue
		}
		validFinishedSyncValidators = append(validFinishedSyncValidators, committeePublicKeys[i])
	}

	return validFinishedSyncValidators
}

func (bc *BlockChain) GetAllCommitteeStakeInfo(epoch uint64) (map[int][]*statedb.StakerInfo, error) {
	height := bc.GetLastBeaconHeightInEpoch(epoch)
	var beaconConsensusRootHash common.Hash
	beaconConsensusRootHash, err := bc.GetBeaconConsensusRootHash(bc.GetBeaconBestState(), height)
	if err != nil {
		return nil, NewBlockChainError(ProcessSalaryInstructionsError, fmt.Errorf("Beacon Consensus Root Hash of Height %+v not found ,error %+v", height, err))
	}
	beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(bc.GetBeaconChainDatabase()))
	if err != nil {
		return nil, NewBlockChainError(ProcessSalaryInstructionsError, err)
	}
	if cState, has := bc.committeeByEpochCache.Peek(epoch); has {
		if result, ok := cState.(map[int][]*statedb.CommitteeState); ok {
			return statedb.GetAllCommitteeStakeInfo(beaconConsensusStateDB, result), nil
		}
	}
	allCommitteeState := statedb.GetAllCommitteeState(beaconConsensusStateDB, bc.GetShardIDs())
	bc.committeeByEpochCache.Add(epoch, allCommitteeState)
	return statedb.GetAllCommitteeStakeInfo(beaconConsensusStateDB, allCommitteeState), nil
}

func (bc *BlockChain) GetAllCommitteeStakeInfoSlashingVersion(epoch uint64) (map[int][]*statedb.StakerInfoSlashingVersion, error) {
	height := bc.GetLastBeaconHeightInEpoch(epoch)
	var beaconConsensusRootHash common.Hash
	beaconConsensusRootHash, err := bc.GetBeaconConsensusRootHash(bc.GetBeaconBestState(), height)
	if err != nil {
		return nil, NewBlockChainError(ProcessSalaryInstructionsError, fmt.Errorf("Beacon Consensus Root Hash of Height %+v not found ,error %+v", height, err))
	}
	beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(bc.GetBeaconChainDatabase()))
	if err != nil {
		return nil, NewBlockChainError(ProcessSalaryInstructionsError, err)
	}
	if cState, has := bc.committeeByEpochCache.Peek(epoch); has {
		if result, ok := cState.(map[int][]*statedb.CommitteeState); ok {
			bc.committeeByEpochProcessLock.Lock()
			defer bc.committeeByEpochProcessLock.Unlock()
			return statedb.GetAllCommitteeStakeInfoSlashingVersion(beaconConsensusStateDB, result), nil
		}
	}
	allCommitteeState := statedb.GetAllCommitteeState(beaconConsensusStateDB, bc.GetShardIDs())
	bc.committeeByEpochCache.Add(epoch, allCommitteeState)
	bc.committeeByEpochProcessLock.Lock()
	defer bc.committeeByEpochProcessLock.Unlock()
	return statedb.GetAllCommitteeStakeInfoSlashingVersion(beaconConsensusStateDB, allCommitteeState), nil
}

func (blockchain *BlockChain) GetPoolManager() *txpool.PoolManager {
	return blockchain.config.PoolManager
}

func (blockchain *BlockChain) UsingNewPool() bool {
	return blockchain.config.usingNewPool
}

func (blockchain *BlockChain) GetShardFixedNodes() []incognitokey.CommitteePublicKey {

	beaconFinalView := blockchain.BeaconChain.GetFinalViewState()
	shardCommittees := beaconFinalView.GetShardCommittee()
	numberOfFixedNode := beaconFinalView.NumberOfFixedShardBlockValidator
	m := []incognitokey.CommitteePublicKey{}

	for _, shardCommittee := range shardCommittees {
		m = append(m, shardCommittee[:numberOfFixedNode]...)
	}

	return m
}
