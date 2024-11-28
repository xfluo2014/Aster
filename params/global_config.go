package params

var (
	Block_Interval      = 500    // generate new block interval
	MaxBlockSize_global = 2000   // the block contains the maximum number of transactions
	InjectSpeed         = 200    // the transaction inject speed
	TotalDataSize       = 200000 // the total number of txs
	BatchSize           = 20000  // supervisor read a batch of txs then send them, it should be larger than inject speed
	BrokerNum           = 10
	RequiredSignNum     = 2

	TotalSignerNum        = 3
	StartDynamicBandwidth = false
	NodesInShard          = 4
	ShardNum              = 4
	LastMod               = 2
	DataWrite_path        = "./expTest/result/"      // measurement data result output path
	LogWrite_path         = "./expTest/log"          // log output path
	SupervisorAddr        = "127.0.0.1:18800"        //supervisor ip address
	FileInput             = `./txN_100W_aN_4096.csv` //the raw BlockTransaction data path
	IpTableMap            = `./ip.json`
	RunTime               = 200
	Bandwidth             = 1000
	Delay                 = -1 // The delay of network (ms) when sending. 0 if delay < 0
	JitterRange           = -1 // The jitter range of delay (ms). Jitter follows a uniform distribution. 0 if JitterRange < 0.

	PropserId   = 0
	ProposerNum = 1

	FollowerNum = 3

	FollowerId = 0

	IsFollower bool

	Join bool
)
