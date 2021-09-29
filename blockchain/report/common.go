package report

const (
	TIMEBEACON_FILE      = "timebeacon.csv"
	DATABEACON_FILE      = "databeacon.csv"
	TIMESHARD_FILE       = "timeshard.csv"
	DATASHARD_FILE       = "databeacon.csv"
	BLOCKINFOSHARD_FILE  = "blkinfoshard.csv"
	BLOCKINFOBEACON_FILE = "blkinfobeacon.csv"
)

const (
	FEATURESIZE     = "featureStateDB"
	BLKSIZE         = "blocksize"
	CONSENSUSSIZE   = "consensusStateDB"
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
)

var (
	ColByFile = map[string][]string{
		TIMESHARD_FILE: {
			BLKHEIGHT,
			VTXS,
			VBLKSIG,
			GETVIEW,
			FETCHBLKSBC,
			GETCOMMITTEE1,
			GETCOMMITTEE2,
			EPOCH,
			TOTALBEACON,
			TOTALTXS,
			TOTALINS,
			UPBSTATE,
			VBSTATE,
			VPOSPROC,
			VPREPROC,
			VPREPROCGETBLKS,
			VPREPROCUNMBLKS,
			VPREPROCHKHEADER,
			VPREPROCMERKLE,
			VPREPROCCREINS,
			VPREPROCGETFEE,
			VPREGETFBLKS,
			VPREVMINERTXS,
			VPREVRESPTXSMETA,
			VPREVRESPTXSINS,
			PROCSAL,
			PROSTORE,
			TOTAL,
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
			BLKHEIGHT,
			EPOCH,
		},
	}
)
