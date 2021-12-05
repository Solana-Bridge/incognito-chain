package report

const (
	TIMEBEACON_FILE      = "timebeacon.csv"
	DATABEACON_FILE      = "databeacon.csv"
	TIMESHARD_FILE       = "timeshard.csv"
	DATASHARD_FILE       = "datashard.csv"
	BLOCKINFOSHARD_FILE  = "blkinfoshard.csv"
	BLOCKINFOBEACON_FILE = "blkinfobeacon.csv"
)

const (
	FEATURESIZE     = "featureStateDB"
	BLKSIZE         = "blocksize"
	CONSENSUSSIZE   = "consensusStateDB"
	TXSSTATESIZE    = "transactionStateDB"
	REWARDSIZE      = "rewardStateDB"
	SLASHSIZE       = "slashStateDB"
	STOREROOTHASH   = "storeRootsHash"
	STOREBLKBYHASH  = "storeBlkByHash"
	STOREFINALHASH  = "storeFinalizedBlockHashByIndex"
	STORECROSSSHARD = "storeCrossShardInfo"
	BACKUPVIEW      = "backupViews"
	TOTALSIZE       = "totalincreasesize"
	TOTALDATAWRITE  = "totalDataWriteInOneBlk"
	TOTALBLKSHARD   = "totalBlkShard"
)

const (
	PROCSTORCONSENSUSTIME     = "procstorConsensus"
	PROCSTORCONSENSUSWTIME    = "procstorWriteConsensus"
	PROCSTORSLASINGTIME       = "procstorSlashing"
	PROCSTORSLASINGWTIME      = "procstorWriteSlashing"
	PROCSTORFEATURETOKENTIME  = "procstorFeatureToken"
	PROCSTORFEATUREBRIDGETIME = "procstorFeatureBridge"
	PROCSTORFEATUREPORTALTIME = "procstorFeaturePortal"
	PROCSTORFEATUREPDETIME    = "procstorFeaturePDE"
	PROCSTORFEATUREWTIME      = "procstorWriteFeature"
	PROCSTORREWARDTIME        = "procstorReward"
	PROCSTORREWARDWTIME       = "procstorWriteReward"

	STOREROOTHASHTIME   = "storeRootsHashTime"
	STOREBLKBYHASHTIME  = "storeBlkByHashTime"
	STOREFINALHASHTIME  = "storeFinalizedBlockHashByIndexTime"
	STORECROSSSHARDTIME = "storeCrossShardInfoTime"
	BACKUPVIEWTIME      = "backupViewsTime"
)

const (
	VTXS             = "validatetxs"
	VBLKSIG          = "validateblksig"
	GETVIEW          = "getviews"
	FETCHBLKSBC      = "fetchblksbeacon"
	GETCOMMITTEE1    = "getcommitteefromview"
	GETCOMMITTEE2    = "getcommitteefrombeacon"
	EPOCH            = "blkepoch"
	TOTALBEACON      = "totalbeaconblks"
	TOTALTXS         = "totaltxs"
	TOTALXTXS        = "totalcrosstxs"
	TXSSIZE          = "txssize"
	TRADESIZE        = "txtradesize"
	TRANFSIZE        = "txtfprisize"
	INSSIZE          = "inssize"
	TOTALINS         = "totalins"
	SIZE             = "blksize"
	BLKHEIGHT        = "blkheight"
	UPBSTATE         = "updatebeststate"
	VBSTATE          = "verifybeststate"
	VPOSPROC         = "validatepostprocessing"
	VPREPROC         = "validatepreprocessing"
	VPREPROCGETBLKS  = "vpreprocgetprevblk"
	VPREPROCUNMBLKS  = "vpreprocunmarshalprevblks"
	VPREPROCHKHEADER = "vpreproccheckheader"
	VPREPROCMERKLE   = "vpreprocprocessmerkle"
	VPREPROCCREINS   = "vpreproccreateins"
	VPREPROCGETFEE   = "vpreprocbuildtxsfee"
	VPREGETFBLKS     = "vpreprocgetfinalizebcblks"
	VPREVMINERTXS    = "vpreprocverifyminertxs"
	VPREVRESPTXSMETA = "vpreprocverifyrespmetatxs"
	VPREVRESPTXSINS  = "vpreprocverifyrespinstxs"
	PROCSAL          = "processsalaryins"
	PROSTORE         = "processstoreblk"
	TOTAL            = "inserttime"

	PROSTORETXVIEWT    = "processstoretxviewtime"
	PROSTORETXVIEWS    = "processstoretxviewsize"
	PROSTOREXTXVIEWT   = "processstorecrosstxviewtime"
	PROSTOREXTXVIEWS   = "processstorecrosstxviewsize"
	PROSTORETOKENINITT = "procstoretokeninittime"
	PROSTORETOKENINITS = "procstoretokeninitsize"
)

const (
	TXByPubkeySize = "txByPubkeySize"
	TXBySerialSize = "txBySerialSize"
	SerialNumSize  = "serialNumSize"
	OTASize        = "OTASize"
	OTABurnSize    = "OTABurnSize"
	SNDSize        = "SNDSize"
	SNDBurnSize    = "SNDBurnSize"
	CMSize         = "totalcmSize"
	CMBurnSize     = "totalcmBurnSize"
	OCoinSize      = "totaloCoinSize"
	OCoinBurnSize  = "totaloCoinBurnSize"

	XOTASize       = "XSHARDOTASize"
	XOTABurnSize   = "XSHARDOTABurnSize"
	XSNDSize       = "XSHARDSNDSize"
	XSNDBurnSize   = "XSHARDSNDBurnSize"
	XCMSize        = "XSHARDtotalcmSize"
	XCMBurnSize    = "XSHARDtotalcmBurnSize"
	XOCoinSize     = "XSHARDtotaloCoinSize"
	XOCoinBurnSize = "XSHARDtotaloCoinBurnSize"
)

var (
	ColByFile = map[string][]string{
		TIMESHARD_FILE: {
			GETVIEW,
			FETCHBLKSBC,
			TOTALBEACON,
			GETCOMMITTEE2,
			VBLKSIG,
			VPREPROC,
			VBSTATE,
			VTXS,
			UPBSTATE,
			VPOSPROC,
			PROCSAL,
			PROSTORE,
			TOTAL,
			BLKHEIGHT,
			EPOCH,
			TOTALTXS,
			TOTALINS,
			VPREPROCGETBLKS,
			VPREPROCUNMBLKS,
			VPREPROCMERKLE,
			VPREPROCCREINS,
			VPREPROCGETFEE,
			VPREGETFBLKS,
			VPREVMINERTXS,
			VPREVRESPTXSMETA,
			VPREVRESPTXSINS,
			PROSTORETXVIEWT,
			PROSTOREXTXVIEWT,
		},
		DATABEACON_FILE: {
			FEATURESIZE,
			BLKSIZE,
			CONSENSUSSIZE,
			REWARDSIZE,
			SLASHSIZE,
			STOREROOTHASH,
			STOREBLKBYHASH,
			STOREFINALHASH,
			STORECROSSSHARD,
			BACKUPVIEW,
			TOTALSIZE,
			TOTALBLKSHARD,
			TOTALINS,
			BLKHEIGHT,
			EPOCH,
		},
		DATASHARD_FILE: {
			TOTALTXS,
			TOTALXTXS,
			TOTALINS,
			EPOCH,
			BLKHEIGHT,
			PROSTORETOKENINITS,
			TOTALBEACON,
			PROSTORETXVIEWS,
			PROSTOREXTXVIEWS,
			CONSENSUSSIZE,
			TXSSTATESIZE,
			FEATURESIZE,
			REWARDSIZE,
			SLASHSIZE,
			BLKSIZE,
			OTASize,
			OTABurnSize,
			SNDSize,
			SNDBurnSize,
			CMSize,
			CMBurnSize,
			OCoinSize,
			OCoinBurnSize,
			TXByPubkeySize,
			TXBySerialSize,
			SerialNumSize,
			XOTASize,
			XOTABurnSize,
			XSNDSize,
			XSNDBurnSize,
			XCMSize,
			XCMBurnSize,
			XOCoinSize,
			XOCoinBurnSize,
		},
		TIMEBEACON_FILE: {
			PROCSTORCONSENSUSTIME,
			PROCSTORCONSENSUSWTIME,
			PROCSTORSLASINGTIME,
			PROCSTORSLASINGWTIME,
			PROCSTORFEATURETOKENTIME,
			PROCSTORFEATUREBRIDGETIME,
			PROCSTORFEATUREPORTALTIME,
			PROCSTORFEATUREPDETIME,
			PROCSTORFEATUREWTIME,
			PROCSTORREWARDTIME,
			PROCSTORREWARDWTIME,
			STOREROOTHASHTIME,
			STOREBLKBYHASHTIME,
			STOREFINALHASHTIME,
			STORECROSSSHARDTIME,
			BACKUPVIEWTIME,
			PROSTORE,
			BLKHEIGHT,
			EPOCH,
		},
	}
)
