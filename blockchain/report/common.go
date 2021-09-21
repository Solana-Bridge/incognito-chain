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
	}
)
