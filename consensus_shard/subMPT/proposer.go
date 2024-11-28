package subMPT

import (
	"blockEmulator/chain"
	"blockEmulator/core"
	"blockEmulator/local"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/utils"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Our system has two roles: Proposer and Follower
// ProposerNode (Proposer) has the following functions:
// 1. Scheduling Txs (Allocate Txs to each follower)
// 2. Generate blocks (gather processed Txs and bundle these Txs into blocks)
// 3. Broadcast blocks to other proposers
// 4. Verifying blocks (verifies Txs in reveived block, allocates Txs in block to follower)
type ProposerNode struct {
	stopchan chan bool
	// Node config
	NodeID      uint
	ShardID     uint
	IPaddr      string
	NeighborIDs []uint
	FollowerIDs []uint

	// network config :['nodeType',nodeID:"127.0.0.1"], nodeType:('p','f') respect [proposer, follower]
	ip_nodeTable map[string]map[uint]string

	// chain config
	CurChain         *chain.BlockChain // all node in the shard maintain the same blockchain
	db               ethdb.Database    // to save the mpt
	chainHeight2Hash map[uint64]string

	// chain lock
	chainLock sync.Mutex

	// logger
	lwLog *LWLog

	// tcp control
	tcpln       net.Listener
	tcpPoolLock sync.Mutex

	// stop control
	stop atomic.Bool

	// about block Pool record
	chainHeightWithBlockAdds []int       // record the height of blockchain when a block is up-to-chain
	blockGenerateTimestamp   []time.Time // record the time the block generated.
	blockArriveTimestamp     []time.Time // record the time the block arrived.
	blockBeforeMPTTimestamp  []time.Time // record the time the sign is all collected.
	dataSizeInEpoch          []int

	blockArriveMap    map[string]time.Time // blockHash -> arrive time
	blockBeforeMPTmap map[string]time.Time // blockHash -> time before mpt

	// block generate control
	schedulingTxsSignal chan bool
	blockHashGenDone    map[string]bool

	PendingTxpool   []*core.Transaction
	AllocatedTxpool map[string]*core.Transaction
	ProcessedTxpool []*core.Transaction

	PendingTxpoolLock   sync.Mutex
	AllocatedTxpoolLock sync.Mutex
	ProcessedTxpoolLock sync.Mutex
	EpochId             atomic.Uint32

	GatherMap  map[uint]map[uint][]byte
	GatherLock sync.Mutex

	VerifyBlock *core.Block

	VerifyLock sync.Mutex

	ConsistentHash *local.ConsistentHash

	ReceiveRequestDataMapLock sync.Mutex
	ReceiveRequestDataMap     map[uint]*message.F_ResponseDataMsgs

	ReadyMapLock sync.Mutex
	ReadyMap     map[uint]bool

	lock1   sync.Mutex
	txcount atomic.Int64
	Idx     uint
}

func NewProposerNode(sID, nID uint, fIDs []uint, pIDs []uint, pcc *params.ChainConfig) *ProposerNode {
	p := new(ProposerNode)
	p.Idx = 0
	p.ReceiveRequestDataMap = make(map[uint]*message.F_ResponseDataMsgs)
	p.EpochId.Store(1)
	p.AllocatedTxpool = make(map[string]*core.Transaction)
	p.PendingTxpool = make([]*core.Transaction, 0)
	p.ProcessedTxpool = make([]*core.Transaction, 0)
	p.ReadyMap = make(map[uint]bool)
	for i := 0; i < len(fIDs); i++ {
		p.ReadyMap[fIDs[i]] = true
	}
	// node config setting
	p.NodeID = nID
	p.ShardID = sID
	p.FollowerIDs = fIDs[:]
	p.NeighborIDs = pIDs[:]
	p.IPaddr = params.IPmap_nodeTable["p"][uint(nID)]

	// network config setting
	p.ip_nodeTable = params.IPmap_nodeTable

	// logger config
	p.lwLog = NewLwLog(int(sID), int(nID), "Proposer")

	// stop init
	p.stop.Store(false)

	// chain config setting
	fp := "./expTest/record/ldb/s" + strconv.FormatUint(uint64(sID), 10) + "/p" + strconv.FormatUint(uint64(nID), 10)
	var err error
	p.db, err = rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	p.CurChain, err = chain.NewBlockChain(pcc, p.db)
	if err != nil {
		p.lwLog.Llog.Panic("err:", "cannot new a blockchain")
	}

	p.chainHeight2Hash = make(map[uint64]string)
	p.chainHeight2Hash[p.CurChain.CurrentBlock.Header.Number] = string(p.CurChain.CurrentBlock.Hash)

	// block pool init
	p.chainHeightWithBlockAdds = make([]int, 1)
	p.blockGenerateTimestamp = make([]time.Time, 1)
	p.dataSizeInEpoch = make([]int, 1)
	p.blockArriveTimestamp = make([]time.Time, 1)
	p.blockBeforeMPTTimestamp = make([]time.Time, 1)

	// append block
	p.chainHeightWithBlockAdds[0] = 0
	p.blockGenerateTimestamp[0] = p.CurChain.CurrentBlock.Header.Time
	p.dataSizeInEpoch[0] = len(p.CurChain.CurrentBlock.Encode())
	p.blockArriveTimestamp[0] = time.Now()
	p.blockBeforeMPTTimestamp[0] = time.Now()

	p.blockArriveMap = make(map[string]time.Time)
	p.blockBeforeMPTmap = make(map[string]time.Time)

	// block pool lock

	// block generate control
	p.schedulingTxsSignal = make(chan bool)
	p.blockHashGenDone = make(map[string]bool)

	p.GatherMap = make(map[uint]map[uint][]byte)

	localHashRing := local.NewSkiplistHashRing()
	murmurHasher := local.NewMurmurHasher()

	localMigrator := func(ctx context.Context, dataKeys map[string]struct{}, from, to string) error {
		//t.Logf("from: %s, to: %s, data keys: %v", from, to, dataKeys)

		FollowersIP := make(map[uint]string, 0)
		for _, i := range p.FollowerIDs {
			FollowersIP[i] = p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][i]
		}

		keys := make([]string, 0)
		for k, _ := range dataKeys {
			keys = append(keys, k)
		}

		requestDataMsg := &message.P_RequestDataMsgs{
			Keys: keys,
		}
		msgBytes, err := json.Marshal(requestDataMsg)
		if err != nil {
			p.lwLog.Llog.Panic("err:", err)
		}
		send_msg := message.MergeMessage(message.P_RequestData, msgBytes)

		p.lwLog.Llog.Printf("now trying to allocate Txs\n")
		fromIp, _ := strconv.Atoi(from)
		toIp, _ := strconv.Atoi(to)
		// Transfer this message
		networks.TcpDial(send_msg, p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(fromIp)])

		for true {
			p.ReceiveRequestDataMapLock.Lock()
			if _, exist := p.ReceiveRequestDataMap[uint(fromIp)]; !exist {
				p.ReceiveRequestDataMapLock.Unlock()
				time.Sleep(100 * time.Millisecond)
				continue
			} else {
				p.ReceiveRequestDataMapLock.Unlock()
				break
			}
		}
		p.ReceiveRequestDataMapLock.Lock()
		msgs := p.ReceiveRequestDataMap[uint(fromIp)]

		allocateDataMsg := &message.P_AllocateDataMsgs{
			Keys:  msgs.Keys,
			Value: msgs.Value,
		}
		msgBytes, err = json.Marshal(allocateDataMsg)
		if err != nil {
			p.lwLog.Llog.Panic("err:", err)
		}
		send_msg = message.MergeMessage(message.P_AllocateData, msgBytes)
		networks.TcpDial(send_msg, p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(toIp)])
		delete(p.ReceiveRequestDataMap, uint(fromIp))
		p.ReceiveRequestDataMapLock.Unlock()

		return nil
	}
	consistentHash := local.NewConsistentHash(
		localHashRing,
		murmurHasher,
		localMigrator,
		// 每个 node 对应的虚拟节点个数为权重 * replicas
		local.WithReplicas(2),
		// 加锁 5 s 后哈希环的锁自动释放
		local.WithLockExpireSeconds(5),
	)

	p.ConsistentHash = consistentHash
	ctx := context.Background()
	for i := 0; i < len(fIDs); i++ {
		if err := p.ConsistentHash.AddNode(ctx, strconv.Itoa(i), 1); err != nil {
			log.Panic("init hash ring error")
		}
	}

	return p
}

func (p *ProposerNode) RunProposerNode() {

	go p.TcpListen()

	p.stopchan = make(chan bool)

	//beginTime := time.Now()
	// tx in block generate
	randomSource := int(p.CurChain.CurrentBlock.Header.Number*10) + int(p.ShardID*1888+p.NodeID)
	txs := make([]*core.Transaction, 0)
	if p.ShardID == 0 {
		p.lwLog.Llog.Println("Proposer Tx Generate Start")
		txs = ProposerGenerateTxs3(params.TotalDataSize, randomSource)
		p.lwLog.Llog.Println("Proposer Tx Generate Over")
	}

	go func() {
		if p.ShardID != 0 {
			return
		}
		for len(txs) != 0 {
			if p.stop.Load() {
				return
			}
			selectTxs := params.InjectSpeed
			if selectTxs > len(txs) {
				selectTxs = len(txs)
			}
			//p.CurChain.PrecessedTxpool.AddTxs2Pool(txs[:selectTxs])

			p.PendingTxpoolLock.Lock()
			transactions := txs[:selectTxs]
			for _, tx := range transactions {
				tx.Time = time.Now()
			}
			p.PendingTxpool = append(p.PendingTxpool, transactions...)
			//for _,tx := range p.PendingTxpool {
			//	tx.Time = time.Now()
			//}
			p.PendingTxpoolLock.Unlock()
			p.lwLog.Llog.Printf("Proposer Tx injected by InjectSpeed,len is %d, PendingTxpool len is \n", selectTxs, len(p.PendingTxpool))
			txs = txs[selectTxs:]
			time.Sleep(time.Second)

		}
	}()

	//for time.Since(beginTime) < time.Second*time.Duration(params.TotalDataSize/80000) {
	//	time.Sleep(time.Second)
	//}

	//go func() {
	//	for true {
	//		if p.stop.Load() {
	//			return
	//		}
	//		//
	//
	//		p.schedulingTxsSignal <- true
	//		time.Sleep(time.Second * 2)
	//	}
	//}()

	go func() {
		p.schedulingTxsSignal <- true
	}()

	//go func() {
	//	//networks.DynamicBandwidth()
	//	time.Sleep(time.Duration(params.RunTime) * time.Second)
	//	p.lwLog.Llog.Println("send to stop chan")
	//	stopChan <- true
	//}()

	//go func() {
	//	idx := 0
	//	for true {
	//		if p.stop.Load() {
	//			return
	//		}
	//		p.blockGenerate(uint(idx))
	//		idx++
	//		time.Sleep(time.Duration(params.Block_Interval) * time.Millisecond)
	//	}
	//}()

	for {
		select {
		case <-p.schedulingTxsSignal:
			p.SchedulingTxs()
		case <-p.stopchan:
			p.lwLog.Llog.Println("Get stop chan")
			p.closeProposerNode()
			close(p.schedulingTxsSignal)

			stopmsg := message.MergeMessage(message.CStop, []byte("this is a stop message~"))
			p.lwLog.Llog.Printf("Proposer: now sending cstop message to all nodes\n")
			for i := 0; i < params.FollowerNum; i++ {
				networks.TcpDial(stopmsg, p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(i)])
			}
			return
		}
	}
}

var poollencount atomic.Int32

func (p *ProposerNode) SchedulingTxs() {

	// 并行的将交易分拣并分发给不同的 Follower
	//for _, tx := range p.PendingTxpool {
	//	wg.Add(1)
	//	go func() { tx }()
	//}
	p.PendingTxpoolLock.Lock()
	p.lwLog.Llog.Println("start SchedulingTxs,len of PendingTxpool is ", len(p.PendingTxpool))
	if len(p.PendingTxpool) == 0 {
		poollencount.Add(1)
		if poollencount.Load() >= 10 {
			p.closeProposerNode()
			p.stopchan <- true
			close(p.schedulingTxsSignal)
			stopmsg := message.MergeMessage(message.CStop, []byte("this is a stop message~"))
			p.lwLog.Llog.Printf("Proposer: now sending cstop message to all nodes\n")
			for i := 0; i < params.FollowerNum; i++ {
				networks.TcpDial(stopmsg, p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(i)])
			}
			return
		}

		go func() {
			time.Sleep(time.Second)
			p.schedulingTxsSignal <- true
		}()
		p.PendingTxpoolLock.Unlock()
		return
	}
	poollencount.Store(0)

	//txSize := len(p.PendingTxpool)
	txSize := params.MaxBlockSize_global
	if txSize > len(p.PendingTxpool) {
		txSize = len(p.PendingTxpool)
	}
	//if txSize > 10000 {
	//	txSize = 10000
	//}
	//localTxpool := make([]*core.Transaction, len(p.PendingTxpool))
	localTxpool := make([]*core.Transaction, txSize)
	copy(localTxpool, p.PendingTxpool)
	//fmt.Println("len of localTxpool is ", len(localTxpool))
	p.PendingTxpool = p.PendingTxpool[txSize:]
	//fmt.Println("len of localTxpool is ", len(localTxpool))
	//fmt.Println("len of PendingTxpool is ", len(p.PendingTxpool))
	p.PendingTxpoolLock.Unlock()

	allocMap := make(map[int][]*core.Transaction)

	for i := 0; i < len(p.FollowerIDs); i++ {
		allocMap[int(p.FollowerIDs[i])] = make([]*core.Transaction, 0)
	}

	var allocMapLock sync.Mutex
	var w sync.WaitGroup
	for i := 0; i < len(localTxpool); i++ {
		w.Add(1)
		//i := i
		go func(i int) {
			defer w.Done()
			//fmt.Println("i is ", i, "len of localTxpool is ", len(localTxpool))
			tx := localTxpool[i]
			senderPrefix := tx.Sender[:1]
			recipientPrefix := tx.Recipient[:1]
			dec1, err := strconv.ParseInt(senderPrefix, 16, 32)
			if err != nil {
				fmt.Println("转换错误:", err)
				return
			}
			dec2, err := strconv.ParseInt(recipientPrefix, 16, 32)
			if err != nil {
				fmt.Println("转换错误:", err)
				return
			}
			//ctx := context.Background()
			//node1, err := p.ConsistentHash.GetNode(ctx, senderPrefix)
			//if err != nil {
			//	log.Panic(err)
			//	return
			//}
			//destId1, _ := strconv.Atoi(node1)
			//
			//node2, err := p.ConsistentHash.GetNode(ctx, recipientPrefix)
			//if err != nil {
			//	log.Panic(err)
			//	return
			//}

			//destId2, _ := strconv.Atoi(node2)

			destId1 := int(dec1) % len(p.FollowerIDs)
			destId2 := int(dec2) % len(p.FollowerIDs)
			defer allocMapLock.Unlock()

			if destId1 == destId2 {
				allocMapLock.Lock()
				allocMap[destId1] = append(allocMap[destId1], tx)
			} else {
				allocMapLock.Lock()
				allocMap[destId1] = append(allocMap[destId1], tx)
				allocMap[destId2] = append(allocMap[destId2], tx)
			}
		}(i)
	}

	w.Wait()

	FollowersIP := make(map[uint]string, 0)
	for _, i := range p.FollowerIDs {
		FollowersIP[i] = p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][i]
	}
	var wg sync.WaitGroup
	epochId := p.EpochId.Add(1)
	for fid, fip := range FollowersIP {
		allocMapLock.Lock()
		go p.TxAllocate(fid, fip, allocMap[int(fid)], &wg, uint(epochId))
		allocMapLock.Unlock()
		wg.Add(1)
	}
	wg.Wait()

	p.lwLog.Llog.Println("end SchedulingTxs")
}

func (p *ProposerNode) GatherTxs(msg *message.Precessed_TxMsgs) {

	p.lwLog.Llog.Printf("start GatherTxs,len is %d\n", len(msg.TxHashs))
	//将 Follower 执行过并返回的交易汇集到 proposer 的交易池
	p.AllocatedTxpoolLock.Lock()
	for i := 0; i < len(msg.TxHashs); i++ {
		txHash := msg.TxHashs[i]
		tx := p.AllocatedTxpool[string(txHash)]
		if tx != nil {
			//p.lwLog.Llog.Printf("tx is not nil")
			tx.AllocateTime = msg.Txs[i].AllocateTime
			tx.ExecutionTime = msg.Txs[i].ExecutionTime
			p.ProcessedTxpool = append(p.ProcessedTxpool, tx)
		}
		delete(p.AllocatedTxpool, string(txHash))
	}
	p.AllocatedTxpoolLock.Unlock()

	p.lwLog.Llog.Println("end GatherTxs")

}

func (p *ProposerNode) TxAllocate(FollowerID uint, FollowerIP string, Txs []*core.Transaction, wg *sync.WaitGroup, epochId uint) {
	defer wg.Done()
	if _, exist := p.ReadyMap[FollowerID]; !exist {
		return
	}

	txMsg := &message.Allocate_TxMsgs{
		Txs:        Txs,
		ProposerID: p.ShardID,
		FollowerID: FollowerID,
		EpochId:    epochId,
	}
	msgBytes, err := json.Marshal(txMsg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.P_TxAllocate, msgBytes)

	p.lwLog.Llog.Printf("now trying to allocate Txs\n")

	// Transfer this message
	networks.TcpDial(send_msg, FollowerIP)

	p.AllocatedTxpoolLock.Lock()
	for i := 0; i < len(Txs); i++ {
		tx := Txs[i]
		p.AllocatedTxpool[string(tx.TxHash)] = tx
	}
	p.AllocatedTxpoolLock.Unlock()

	// log here
	p.lwLog.Llog.Printf("%d Txs is allocated to follower %d,len of AllocatedTxpool is %d\n", len(Txs), FollowerID, len(p.AllocatedTxpool))

}

func (p *ProposerNode) blockGenerate(EpochId uint) *core.Block {

	p.chainLock.Lock()
	defer p.chainLock.Unlock()

	//states := make([]byte, 0)
	//
	//p.GatherLock.Lock()
	//for i := uint(0); i < uint(params.FollowerNum); i++ {
	//	if _,exist:=p.GatherMap[EpochId];exist{
	//		if _,exist2 := p.GatherMap[EpochId][i];exist2{
	//			states = append(states, p.GatherMap[EpochId][i]...)
	//		}
	//	}
	//}
	//
	//p.GatherLock.Unlock()
	//
	//var buff bytes.Buffer
	//
	//enc := gob.NewEncoder(&buff)
	//err := enc.Encode(states)
	//if err != nil {
	//	log.Panic(err)
	//}
	//hash := sha256.Sum256(buff.Bytes())

	hash := []byte("123")
	if _, ok := p.blockHashGenDone[string(p.CurChain.CurrentBlock.Hash)]; ok {
		p.lwLog.Llog.Println("Block in this height is generated, skip & not re-generate")
		return nil
	}

	//p.lwLog.Llog.Println("be a proposer now...")

	p.ProcessedTxpoolLock.Lock()
	txNum := len(p.ProcessedTxpool)
	//txNum := params.MaxBlockSize_global
	//if len(p.ProcessedTxpool) < txNum {
	//	txNum = len(p.ProcessedTxpool)
	//}
	txs1 := p.ProcessedTxpool[:txNum]
	if len(txs1) == 0 {
		p.ProcessedTxpoolLock.Unlock()
		return nil
	}
	p.lwLog.Llog.Printf("start blockGenerate,len of ProcessedTxpool is %d\n ", len(p.ProcessedTxpool))
	p.ProcessedTxpool = p.ProcessedTxpool[txNum:]

	// handle transactions to build root
	//rt := bc.GetUpdateStatusTrie(txs)
	bh := &core.BlockHeader{
		ParentBlockHash: p.CurChain.CurrentBlock.Hash[:],
		Number:          p.CurChain.CurrentBlock.Header.Number + 1,
		Time:            time.Now(),
	}

	bh.StateRoot = hash[:]
	//bh.TxRoot = GetTxTreeRoot(txs1)
	bh.TxRoot = hash[:]

	bs := bitset.New(2048)

	for _, tx := range txs1 {
		if tx == nil {
			continue
		}
		bs.Set(utils.ModBytes(tx.TxHash, 2048))
	}
	bh.BloomFilter = *bs
	b := core.NewBlock(bh, txs1)
	b.Header.Miner = uint64(0)
	b.Hash = b.Header.Hash()

	block := b

	//block := p.CurChain.GenerateBlock2(int(p.NodeID), &p.ProcessedTxpool)
	p.ProcessedTxpoolLock.Unlock()

	p.blockHashGenDone[string(block.Header.ParentBlockHash)] = true
	if !bytes.Equal(block.Header.ParentBlockHash, p.CurChain.CurrentBlock.Hash) {
		p.lwLog.Llog.Panic("err:", "Err when generating block")
	}

	// log here
	p.CurChain.AddBlock(block)
	Record(block)
	p.lwLog.Llog.Printf("block is generated, height = %d\n", block.Header.Number)
	go func() {
		time.Sleep(time.Millisecond * time.Duration(params.Block_Interval))
		p.schedulingTxsSignal <- true
	}()

	fmt.Println("blockBroadcast...")
	p.txcount.Add(int64(10000))
	//go p.blockBroadcast(block)
	//p.lwLog.Llog.Println("end blockGenerate")
	return block
}

var txs atomic.Int32

func Record(block *core.Block) {
	dirpath := fmt.Sprintf("%s/interval=%d_blockSize=%d_totalSize=%d_Inj=%v_bandwidth=%v_runtime=%v/", params.DataWrite_path, params.Block_Interval,
		params.MaxBlockSize_global, params.TotalDataSize, params.InjectSpeed, params.Bandwidth, params.RunTime)
	// create or open the CSV file
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		os.MkdirAll(dirpath, 0775)
		//if err != nil {
		//	w.pl.Plog.Printf("S%dN%d : error creating the result directory %v \n", w.ShardID, w.NodeID, err)
		//}
	}
	file, err := os.OpenFile(dirpath+fmt.Sprintf("/%d.csv", params.FollowerNum), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// write the timestamp to the CSV file
	writer := csv.NewWriter(file)
	//colName := []string{"Txs", "Time", "Latency"}
	//colName := []string{"Latency","Txs"}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Panic(err)
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write([]string{"blockHeight", "blockGenerate", "blockBeforeMPT", "blockUpToChain", "dataSize", "txs", "latency", "queuetime", "executiontime"}); err != nil {
			log.Panic(err)
		}
		//writer.Write([]string{
		//	strconv.Itoa(0),
		//	strconv.Itoa(int(time.Now().UnixMilli())),
		//	strconv.Itoa(0),
		//})
		writer.Flush()
	}
	latencySum := int64(0)
	queueLatencySum := int64(0)
	executionTimeSum := int64(0)
	for _, tx := range block.Body {
		latencySum += int64(time.Since(tx.Time).Milliseconds())
		queueLatencySum += int64(tx.AllocateTime.Sub(tx.Time).Milliseconds())
		executionTimeSum += int64(tx.ExecutionTime.Sub(tx.AllocateTime).Milliseconds())
	}

	if err := writer.Error(); err != nil {
		log.Fatal(err)
	}
	blockBeforeMPT := time.Now().UnixMilli()

	// get the current timestamp in milliseconds
	blockUpToChain := time.Now().UnixMilli()
	writer.Write([]string{
		strconv.FormatInt(int64(block.Header.Number), 10),
		strconv.FormatInt(block.Header.Time.UnixMilli(), 10),
		strconv.FormatInt(blockBeforeMPT, 10),
		strconv.FormatInt(blockUpToChain, 10),
		strconv.Itoa(len(block.Encode())),
		strconv.Itoa(len(block.Body)),
		strconv.FormatInt(latencySum, 10),
		strconv.FormatInt(queueLatencySum, 10),
		strconv.FormatInt(executionTimeSum, 10),
	})
	writer.Flush()
	//LatencySum := int64(0)
	//for _, tx := range block.Body {
	//	LatencySum += time.Since(tx.Time).Milliseconds()
	//}

	//for _, tx := range block.Body {
	//	writer.Write([]string{
	//		//strconv.Itoa(len(block.Body)),
	//		//strconv.Itoa(int(time.Now().UnixMilli())),
	//		strconv.Itoa(int(time.Since(tx.Time).Milliseconds())),
	//		//strconv.Itoa(int(time.Since(tx.Time).Milliseconds())),
	//	})
	//}

	//if block.Header.Number == 1 {
	//	title := []string{"blockHeight", "blockGenerate", "blockArrived", "blockBeforeMPT", "blockUpToChain", "dataSize", "txs", "latency"}
	//	writer.Write(title)
	//}

	//latencySum := int64(0)
	//for _, tx := range block.Body {
	//	latencySum += int64(time.Since(tx.Time).Milliseconds())
	//}
	//
	//if err := writer.Error(); err != nil {
	//	log.Fatal(err)
	//}
	//blockBeforeMPT := time.Now().UnixMilli()
	//
	//// get the current timestamp in milliseconds
	//blockUpToChain := time.Now().UnixMilli()

	//directory := "./record/ldb/s0n1" // 替换为你要计算的目录路径
	//
	//totalSize, err := calculateTotalSize(directory)
	//
	//txs.Add(int32(len(block.Body)))

	//for _, tx := range block.Body {
	//	writer.Write([]string{
	//		strconv.Itoa(int(time.Since(tx.Time).Milliseconds())),
	//	})
	//}

	//writer.Flush()
}

func calculateTotalSize(dir string) (int64, error) {
	var totalSize int64

	// Walk 函数用于遍历目录树
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 忽略目录，只处理文件
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return totalSize, nil
}
func (p *ProposerNode) VerifyingBlock(block *core.Block) {
	txSize := len(block.Body)
	//localTxpool := make([]*core.Transaction, len(p.PendingTxpool))
	localTxpool := make([]*core.Transaction, txSize)
	copy(localTxpool, block.Body)

	allocMap := make(map[int][]*core.Transaction)

	for i := 0; i < len(p.FollowerIDs); i++ {
		allocMap[int(p.FollowerIDs[i])] = make([]*core.Transaction, 0)
	}

	var allocMapLock sync.Mutex
	var w sync.WaitGroup
	for i := 0; i < len(localTxpool); i++ {
		w.Add(1)
		i := i
		go func() {
			defer w.Done()
			//fmt.Println("i is ", i, "len of localTxpool is ", len(localTxpool))
			tx := localTxpool[i]
			senderPrefix := tx.Sender[:1]
			recipientPrefix := tx.Recipient[:1]
			dec1, err := strconv.ParseInt(senderPrefix, 16, 32)
			if err != nil {
				fmt.Println("转换错误:", err)
				return
			}
			dec2, err := strconv.ParseInt(recipientPrefix, 16, 32)
			if err != nil {
				fmt.Println("转换错误:", err)
				return
			}
			destId1 := int(dec1) % len(p.FollowerIDs)
			destId2 := int(dec2) % len(p.FollowerIDs)
			defer allocMapLock.Unlock()

			if destId1 == destId2 {
				allocMapLock.Lock()
				allocMap[destId1] = append(allocMap[destId1], tx)
			} else {
				allocMapLock.Lock()
				allocMap[destId1] = append(allocMap[destId1], tx)
				allocMap[destId2] = append(allocMap[destId2], tx)
			}
		}()
	}

	w.Wait()

	FollowersIP := make(map[uint]string, 0)
	for _, i := range p.FollowerIDs {
		FollowersIP[i] = p.ip_nodeTable["f"+strconv.Itoa(int(p.ShardID))][i]
	}
	var wg sync.WaitGroup
	epochId := p.EpochId.Add(1)
	for fid, fip := range FollowersIP {
		allocMapLock.Lock()
		go p.TxAllocate(fid, fip, allocMap[int(fid)], &wg, uint(epochId))
		allocMapLock.Unlock()
		wg.Add(1)
	}
	wg.Wait()

	p.lwLog.Llog.Println("end SchedulingTxs")
}

func (p *ProposerNode) blockBroadcast(block *core.Block) {
	if block == nil {
		return
	}

	lwbmsg := &message.BlockMsg{
		EncodedBlock: block.Encode(),
		ProposerID:   p.ShardID,
	}
	lwbByte, err := json.Marshal(lwbmsg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.P_BlockBroadcast, lwbByte)

	NeighborsIP := make([]string, 0)
	for _, i := range p.NeighborIDs {
		fmt.Println(p.ip_nodeTable["p"][i])
		NeighborsIP = append(NeighborsIP, p.ip_nodeTable["p"][i])
	}

	// broadcast this message
	networks.Broadcast(p.IPaddr, NeighborsIP, send_msg)

	// log here
	p.lwLog.Llog.Printf("block is broadcast\n")
}

func (p *ProposerNode) closeProposerNode() {
	p.lwLog.Llog.Println("Now close Proposer node ...")
	write_csv(int(p.ShardID), int(p.NodeID), p.chainHeightWithBlockAdds, p.dataSizeInEpoch, p.blockGenerateTimestamp, p.blockArriveTimestamp, p.blockBeforeMPTTimestamp)
	p.stop.Store(true)
	p.tcpln.Close()
	p.chainLock.Lock()
	p.CurChain.CloseBlockChain()
	p.chainLock.Unlock()
}
