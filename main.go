package main

import (
	"blockEmulator/build"
	"blockEmulator/consensus_shard/subMPT"
	"blockEmulator/core"
	"blockEmulator/networks"
	"blockEmulator/params"
	rand2 "crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/pflag"
	"io"
	"log"
	"math/big"
	"math/rand"
	"os"
	"time"
)

var (
	shardNum       int
	nodeNum        int
	shardID        int
	nodeID         int
	modID          int
	followerId     int
	proposerId     int
	isSupervisor   bool
	isWallet       bool
	isGen          bool
	isLWNode       bool
	isFollower     bool
	proposerNum    int
	followerNum    int
	requireSignNum int
	totalSignerNum int
	totalDataSize  int
	blockSize      int
	runTime        int
	bandwidth      int
	injectSpd      int
	dynamicBdw     bool
	join           bool
)

func generateRandomHexString(length int) (string, error) {
	// 计算所需的字节数，因为每个 16 进制字符代表 4 位二进制（即半个字节）
	// 所以长度为 40 的 16 进制字符串需要 20 个字节
	bytes := make([]byte, length/2)
	if _, err := io.ReadFull(rand2.Reader, bytes); err != nil {
		return "", err
	}
	// 将字节数组转换为 16 进制字符串
	return hex.EncodeToString(bytes), nil
}
func ProposerGenerateTxs1(txSize int, uniqueSource int) []*core.Transaction {
	txs := make([]*core.Transaction, 0)
	hexString1, _ := generateRandomHexString(40)
	hexString2, _ := generateRandomHexString(40)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(uniqueSource)))
	for i := 0; i < txSize; i++ {
		//payerInt := r.Int63n(200000) + 123456
		//payeeInt := r.Int63n(200000) + 123456
		valueInt := r.Int63n(123456)
		//payerString := strconv.Itoa(int(payerInt))
		payerString := hexString1
		//payeeString := strconv.Itoa(int(payeeInt))
		payeeString := hexString2
		valueBigInt := new(big.Int).SetInt64(valueInt)

		tx := core.NewTransaction(payerString, payeeString, valueBigInt, uint64(uniqueSource+i))
		txs = append(txs, tx)
	}

	return txs
}

func ProposerGenerateTxs2(txSize int, uniqueSource int) []*core.Transaction {
	txs := make([]*core.Transaction, 0)
	//hexString1, _ := generateRandomHexString(40)
	//hexString2, _ := generateRandomHexString(40)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(uniqueSource)))
	for i := 0; i < txSize; i++ {
		hexString1, _ := generateRandomHexString(40)
		hexString2, _ := generateRandomHexString(40)
		//payerInt := r.Int63n(200000) + 123456
		//payeeInt := r.Int63n(200000) + 123456
		valueInt := r.Int63n(123456)
		//payerString := strconv.Itoa(int(payerInt))
		payerString := hexString1
		//payeeString := strconv.Itoa(int(payeeInt))
		payeeString := hexString2
		valueBigInt := new(big.Int).SetInt64(valueInt)

		tx := core.NewTransaction(payerString, payeeString, valueBigInt, uint64(uniqueSource+i))
		txs = append(txs, tx)
	}

	return txs
}

func generateRandomHexString1(length int) (string, error) {
	// 计算所需的字节数，因为每个 16 进制字符代表 4 位二进制（即半个字节）
	// 所以长度为 40 的 16 进制字符串需要 20 个字节
	bytes := make([]byte, length/2)
	if _, err := io.ReadFull(rand2.Reader, bytes); err != nil {
		return "", err
	}
	// 将字节数组转换为 16 进制字符串
	return hex.EncodeToString(bytes), nil
}
func data2tx(data []string, nonce uint64) (*core.Transaction, bool) {
	if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] {
		val, ok := new(big.Int).SetString(data[8], 10)
		if !ok {
			log.Panic("new int failed\n")
		}
		tx := core.NewTransaction4(data[3][2:], data[4][2:], val, nonce, time.Now())
		return tx, true
	}
	return &core.Transaction{}, false
}
func ProposerGenerateTxs3(txSize int, uniqueSource int) []*core.Transaction {
	//txs := make([]*core.Transaction, 0)

	txfile, err := os.Open("C:\\Users\\1\\Desktop\\2000000to2999999_BlockTransaction.csv")
	if err != nil {
		log.Panic(err)
	}
	defer txfile.Close()
	reader := csv.NewReader(txfile)
	txlist := make([]*core.Transaction, 0) // save the txs in this epoch (round)
	nowDataNum := 0
	for {
		data, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		if tx, ok := data2tx(data, uint64(nowDataNum)); ok {
			txlist = append(txlist, tx)
			nowDataNum++
		}

		if nowDataNum == txSize {
			break
		}
	}

	return txlist
}
func main() {

	//fp := "./record/ldb/test"
	//db, err := rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false)
	//triedb := trie.NewDatabaseWithConfig(db, &trie.Config{
	//	Cache:     0,
	//	Preimages: true,
	//})
	//statusTrie := trie.NewEmpty(triedb)
	//StateRoot := statusTrie.Hash().Bytes()
	//
	//st, err := trie.New(trie.TrieID(common.BytesToHash(StateRoot)), triedb)
	////st.Update([]byte("aaaa123456789123456789123456789123456789"), []byte("12345"))
	//
	////max1 := 0
	//
	//txs := make([]*core.Transaction, 0)
	//
	////txs := ProposerGenerateTxs3(1000000, 0)
	//for i := 0; i < 600000; i++ {
	//	hexString1, _ := generateRandomHexString1(40)
	//	hexString2, _ := generateRandomHexString1(40)
	//	txs = append(txs, core.NewTransaction(hexString1, hexString2, big.NewInt(0), 1))
	//}
	//
	//now := time.Now()
	//for i, tx := range txs {
	//	ib := new(big.Int)
	//	ib.Add(ib, params.Init_Balance)
	//	s_state := &core.AccountState{
	//		Nonce:   uint64(i),
	//		Balance: ib,
	//	}
	//	s_balance := s_state.Balance
	//	if s_balance.Cmp(tx.Value) == -1 {
	//		fmt.Printf("the balance is less than the transfer amount\n")
	//	}
	//	s_state.Deduct(tx.Value)
	//	st.Update([]byte(tx.Sender), s_state.Encode())
	//
	//	ib.Add(ib, params.Init_Balance)
	//	r_state := &core.AccountState{
	//		Nonce:   uint64(i),
	//		Balance: ib,
	//	}
	//
	//	r_state.Deposit(tx.Value)
	//	st.Update([]byte(tx.Recipient), r_state.Encode())
	//}
	//
	//fmt.Println(time.Since(now).Milliseconds())
	//rt, ns := st.Commit(false)
	//err = triedb.Update(trie.NewWithNodeSet(ns))
	//if err != nil {
	//	log.Panic()
	//}
	//err = triedb.Commit(rt, false)
	//fmt.Println(time.Since(now).Milliseconds())
	//
	//txs = make([]*core.Transaction, 0)
	//
	////txs := ProposerGenerateTxs3(1000000, 0)
	//for i := 0; i < 400000; i++ {
	//	hexString1, _ := generateRandomHexString1(40)
	//	hexString2, _ := generateRandomHexString1(40)
	//	txs = append(txs, core.NewTransaction(hexString1, hexString2, big.NewInt(0), 1))
	//}
	//
	//st, err = trie.New(trie.TrieID(common.BytesToHash(rt.Bytes())), triedb)
	//
	//now = time.Now()
	//for i, tx := range txs {
	//	ib := new(big.Int)
	//	ib.Add(ib, params.Init_Balance)
	//	s_state := &core.AccountState{
	//		Nonce:   uint64(i),
	//		Balance: ib,
	//	}
	//	s_balance := s_state.Balance
	//	if s_balance.Cmp(tx.Value) == -1 {
	//		fmt.Printf("the balance is less than the transfer amount\n")
	//	}
	//	s_state.Deduct(tx.Value)
	//	st.Update([]byte(tx.Sender), s_state.Encode())
	//
	//	ib.Add(ib, params.Init_Balance)
	//	r_state := &core.AccountState{
	//		Nonce:   uint64(i),
	//		Balance: ib,
	//	}
	//
	//	r_state.Deposit(tx.Value)
	//	st.Update([]byte(tx.Recipient), r_state.Encode())
	//}
	//
	//t := time.Now()
	//fmt.Println(time.Since(now).Milliseconds())
	//rt, ns = st.Commit(false)
	//err = triedb.Update(trie.NewWithNodeSet(ns))
	//if err != nil {
	//	log.Panic()
	//}
	//err = triedb.Commit(rt, false)
	//fmt.Println(time.Since(t).Milliseconds())
	//return

	//bytes := []byte(strconv.Itoa(0))
	//
	//now := time.Now()
	//for i := 1; i <= 20000000; i++ {
	//
	//	if i%1000000 == 0 {
	//		fmt.Println(strconv.Itoa(i) + " " + strconv.Itoa(int(time.Since(now).Milliseconds())))
	//		now = time.Now()
	//	}
	//	//hexString1, _ := generateRandomHexString1(40)
	//	st.Update(strs[i], bytes)
	//	//st.Update([]byte(strconv.Itoa(i)), []byte(strconv.Itoa(i)))
	//}
	//
	//
	////rrt, ns := st.Commit(false)
	////err = triedb.Update(trie.NewWithNodeSet(ns))
	////if err != nil {
	////	log.Panic(err)
	////}
	////err = triedb.Commit(rrt, false)
	////if err != nil {
	////	log.Panic(err)
	////}
	//
	//iterator := st.NodeIterator([]byte("0"))
	//
	//now = time.Now()
	//idx := 0
	//count := 0
	//for true {
	//	count++
	//	if iterator.Leaf() {
	//		idx++
	//		if idx%1000000 == 0 {
	//			fmt.Println(strconv.Itoa(idx) + " " + strconv.Itoa(int(time.Since(now).Milliseconds())))
	//			now = time.Now()
	//		}
	//		path := iterator.Path()
	//		//fmt.Println(path)
	//		if len(path) > max1 {
	//			max1 = len(path)
	//		}
	//		//fmt.Println(string(iterator.LeafKey()))
	//		//fmt.Println(string(iterator.LeafBlob()))
	//	}
	//
	//	if !iterator.Next(true) {
	//		break
	//	}
	//}
	//
	////fmt.Println(max1)
	//fmt.Println(count)

	//st, err = trie.New(trie.TrieID(common.BytesToHash(rrt.Bytes())), triedb)
	//st.Update([]byte("aaaa123456789123456789123456789123456789"), []byte("12345"))
	//rrt, ns = st.Commit(false)
	//if err != nil {
	//	log.Panic(err)
	//}
	//err = triedb.Update(trie.NewWithNodeSet(ns))
	//if err != nil {
	//	log.Panic(err)
	//}
	//err = triedb.Commit(rrt, false)
	//if err != nil {
	//	log.Panic(err)
	//}

	//return
	//
	//st.Update([]byte("aaaa123456789123456789123456789123456789"), []byte("12345"))
	//st.Update([]byte("aaba123456789123456789123456789123456789"), []byte("12345"))
	//bytes, _ := st.Get([]byte("aaba123456789123456789123456789123456789"))
	//fmt.Println(string(bytes))
	//rt, ns := st.Commit(false)
	//
	//triedb.Update(trie.NewWithNodeSet(ns))
	//StateRoot = rt.Bytes()

	//return
	//
	//fp1 := "./record1/ldb"
	//fp2 := "./record2/ldb"
	//db1, _ := rawdb.NewLevelDBDatabase(fp1, 0, 1, "accountState", false)
	//db2, _ := rawdb.NewLevelDBDatabase(fp2, 0, 1, "accountState", false)
	//triedb1 := trie.NewDatabaseWithConfig(db1, &trie.Config{
	//	Cache:     0,
	//	Preimages: true,
	//})
	//triedb2 := trie.NewDatabaseWithConfig(db2, &trie.Config{
	//	Cache:     0,
	//	Preimages: true,
	//})
	//statusTrie1 := trie.NewEmpty(triedb1)
	//statusTrie2 := trie.NewEmpty(triedb2)
	//StateRoot1 := statusTrie1.Hash().Bytes()
	//StateRoot2 := statusTrie2.Hash().Bytes()
	//for i := 0; i < len(StateRoot1); i++ {
	//	fmt.Print(StateRoot1[i])
	//	fmt.Print(" ")
	//}
	//fmt.Println()
	//for i := 0; i < len(StateRoot2); i++ {
	//	fmt.Print(StateRoot2[i])
	//	fmt.Print(" ")
	//}
	//for i := 0; i < len(StateRoot1); i++ {
	//	if StateRoot1[i] != StateRoot2[i] {
	//		fmt.Println("error")
	//	}
	//}
	//
	//if string(StateRoot1) != string(StateRoot2) {
	//	fmt.Println("error")
	//}
	//st, _ := trie.New(trie.TrieID(common.BytesToHash(StateRoot)), triedb)
	//txs := ProposerGenerateTxs1(1000000, 1)
	//
	//now := time.Now()
	//for i, tx := range txs {
	//	// senderIn = true
	//	// fmt.Printf("the sender %s is in this shard %d, \n", tx.Sender, bc.ChainConfig.ShardID)
	//	// modify local accountstate
	//	s_state_enc, _ := st.Get([]byte(tx.Sender))
	//	var s_state *core.AccountState
	//	if s_state_enc == nil {
	//		// fmt.Println("missing account SENDER, now adding account")
	//		ib := new(big.Int)
	//		ib.Add(ib, params.Init_Balance)
	//		s_state = &core.AccountState{
	//			Nonce:   uint64(i),
	//			Balance: ib,
	//		}
	//	} else {
	//		ib := new(big.Int)
	//		ib.Add(ib, params.Init_Balance)
	//		s_state = core.DecodeAS(s_state_enc)
	//		//s_state = &core.AccountState{
	//		//	Nonce:   uint64(i),
	//		//	Balance: ib,
	//		//}
	//	}
	//	s_balance := s_state.Balance
	//	if s_balance.Cmp(tx.Value) == -1 {
	//		fmt.Printf("the balance is less than the transfer amount\n")
	//		continue
	//	}
	//	s_state.Deduct(tx.Value)
	//	err := st.Update([]byte(tx.Sender), s_state.Encode())
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	// fmt.Printf("the recipient %s is in this shard %d, \n", tx.Recipient, bc.ChainConfig.ShardID)
	//	// recipientIn = true
	//	// modify local state
	//	r_state_enc, _ := st.Get([]byte(tx.Recipient))
	//	var r_state *core.AccountState
	//	if r_state_enc == nil {
	//		// fmt.Println("missing account RECIPIENT, now adding account")
	//		ib := new(big.Int)
	//		ib.Add(ib, params.Init_Balance)
	//		r_state = &core.AccountState{
	//			Nonce:   uint64(i),
	//			Balance: ib,
	//		}
	//	} else {
	//		ib := new(big.Int)
	//		ib.Add(ib, params.Init_Balance)
	//		//r_state = core.DecodeAS(r_state_enc)
	//		r_state = &core.AccountState{
	//			Nonce:   uint64(i),
	//			Balance: ib,
	//		}
	//	}
	//	r_state.Deposit(tx.Value)
	//	err = st.Update([]byte(tx.Recipient), r_state.Encode())
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//
	//}
	//
	//fmt.Println(time.Now().Sub(now).Milliseconds()) //100万笔 7417 19374 4000
	//
	//t := time.Now()
	//rt, ns := st.Commit(false)
	//t2 := time.Now()
	//fmt.Println(t2.Sub(t).Milliseconds()) //100万笔 3567
	//
	//err := triedb.Update(trie.NewWithNodeSet(ns))
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//t3 := time.Now()
	//fmt.Println(t3.Sub(t2).Milliseconds()) //100万笔 2920
	//
	//err = triedb.Commit(rt, false)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//fmt.Println(time.Now().Sub(t3).Milliseconds()) //100万笔 17337
	//
	//return

	//runtime.GOMAXPROCS(1)

	//a := make([]int64, 0)
	//for i := 0; i < 20000; i++ {
	//	//time.Sleep(1 * time.Microsecond)
	//	a = append(a, time.Now().UnixMicro())
	//}
	//for i := 0; i < 20000; i++ {
	//	fmt.Println(a[i])
	//}
	//return

	pflag.IntVarP(&shardNum, "shardNum", "S", 2, "indicate that how many shards are deployed")
	pflag.IntVarP(&nodeNum, "nodeNum", "N", 4, "indicate how many nodes of each shard are deployed")
	pflag.IntVarP(&shardID, "shardID", "s", 0, "id of the shard to which this node belongs, for example, 0")
	pflag.IntVarP(&nodeID, "nodeID", "n", 0, "id of this node, for example, 0")
	pflag.IntVarP(&modID, "modID", "m", 3, "choice Committee Method,for example, 0, [CLPA_Broker,CLPA,Broker,Relay] ")

	pflag.IntVarP(&requireSignNum, "RequireSignNum", "v", 1, "number of sign a block required")
	pflag.IntVarP(&totalSignerNum, "TotalSignerNum", "z", 1, "number of signers")
	pflag.IntVarP(&totalDataSize, "TotalDataSize", "x", 200000, "number of signers")
	pflag.IntVarP(&blockSize, "BlockSize", "b", 10000, "block size")
	pflag.IntVarP(&runTime, "RunTime", "i", 20000, "run time")
	pflag.BoolVarP(&isSupervisor, "supervisor", "c", false, "whether this node is a supervisor")
	pflag.BoolVarP(&isWallet, "wallet/pureclient", "w", false, "whether this node is a pure client / wallet")
	pflag.BoolVarP(&isLWNode, "lightWeightNode", "l", false, "whether this node is a lightWeight node")
	pflag.BoolVarP(&isGen, "gen", "g", false, "generation bat")
	pflag.BoolVar(&dynamicBdw, "dynamicBandwidth", false, "dynamicBandwidth")
	pflag.IntVarP(&bandwidth, "bandwidth", "B", 1000, "bandwidth(mb/s)")
	pflag.IntVarP(&injectSpd, "injectSpd", "k", 10000, "inject speed")

	pflag.IntVarP(&proposerNum, "ProposerNumber", "p", nodeNum/2, "number of proposer")
	//是否是follower
	pflag.BoolVarP(&isFollower, "follower", "F", false, "whether is follower")
	pflag.IntVar(&followerId, "fid", 0, "id of the follower, for example, 0")
	pflag.IntVar(&proposerId, "pid", 0, "id of proposer, for example, 0")
	pflag.BoolVar(&join, "join", false, "")
	pflag.IntVar(&followerNum, "fnum", 1, "num of follower")
	pflag.Parse()

	params.NodesInShard = nodeNum
	params.RequiredSignNum = requireSignNum
	params.ProposerNum = proposerNum
	params.TotalSignerNum = totalSignerNum
	params.TotalDataSize = totalDataSize
	params.MaxBlockSize_global = blockSize
	params.RunTime = runTime
	params.InjectSpeed = injectSpd
	params.StartDynamicBandwidth = dynamicBdw

	params.FollowerId = followerId
	params.FollowerNum = followerNum
	params.PropserId = proposerId
	params.IsFollower = isFollower
	params.Join = join

	params.Bandwidth = bandwidth * 1024 * 1024
	//networks.InitLimiter(bandwidth*1024*1024, nodeID)

	networks.InitNetworkTools()
	if isGen {
		build.GenerateBatFile(nodeNum, shardNum, modID)
		build.GenerateShellFile(nodeNum, shardNum, modID)
		build.LightWeight_GenerateBatFile(nodeNum, shardNum, modID)
		return
	}

	file, err := os.ReadFile("./ipTable.json")
	if err != nil {
		// handle error
		fmt.Println(err)
	}
	// Create a map to store the IP addresses
	var ipMap map[string]map[uint]string
	// Unmarshal the JSON data into the map
	err = json.Unmarshal(file, &ipMap)
	if err != nil {
		// handle error
		fmt.Println(err)
	}

	params.IPmap_nodeTable = ipMap

	if isFollower {
		node := subMPT.NewFollowerNode(uint(shardID), uint(proposerId), uint(followerId), join)
		node.RunFollowerNode()
	} else {
		var fids []uint
		var pids []uint
		for i := uint(0); i < uint(followerNum); i++ {
			fids = append(fids, i)
		}
		for i := uint(0); i < uint(proposerNum); i++ {
			if i != uint(shardID) {
				pids = append(pids, i)
			}
		}

		config := initConfig(uint64(proposerId), 0)
		node := subMPT.NewProposerNode(uint(shardID), uint(proposerId), fids, pids, config)
		node.RunProposerNode()
	}

	return
	//if isSupervisor {
	//	build.BuildSupervisor(uint64(nodeNum), uint64(shardNum), uint64(modID))
	//} else if isWallet {
	//	build.BuildClient(uint64(nodeNum), uint64(shardNum), uint64(modID))
	//} else if isLWNode {
	//	build.BuildLWNode(uint64(nodeID), uint64(nodeNum), uint64(shardID), uint64(shardNum), uint64(modID))
	//} else {
	//	build.BuildNewPbftNode(uint64(nodeID), uint64(nodeNum), uint64(shardID), uint64(shardNum), uint64(modID))
	//}
}
func initConfig(nid, sid uint64) *params.ChainConfig {
	// Read the contents of ipTable.json
	file, err := os.ReadFile("./ipTable.json")
	if err != nil {
		// handle error
		fmt.Println(err)
	}
	// Create a map to store the IP addresses
	var ipMap map[string]map[uint]string
	// Unmarshal the JSON data into the map
	err = json.Unmarshal(file, &ipMap)
	if err != nil {
		// handle error
		fmt.Println(err)
	}

	params.IPmap_nodeTable = ipMap

	// check the correctness of params
	//if len(ipMap)-1 < int(snm) {
	//	log.Panicf("Input ShardNumber = %d, but only %d shards in ipTable.json.\n", snm, len(ipMap)-1)
	//}
	//for shardID := 0; shardID < len(ipMap)-1; shardID++ {
	//	if len(ipMap[uint64(shardID)]) < int(nnm) {
	//		log.Panicf("Input NodeNumber = %d, but only %d nodes in Shard %d.\n", nnm, len(ipMap[uint64(shardID)]), shardID)
	//	}
	//}

	pcc := &params.ChainConfig{
		ChainID:        sid,
		NodeID:         nid,
		ShardID:        sid,
		Nodes_perShard: uint64(params.NodesInShard),
		ShardNums:      0,
		BlockSize:      uint64(params.MaxBlockSize_global),
		BlockInterval:  uint64(params.Block_Interval),
		InjectSpeed:    uint64(params.InjectSpeed),
	}
	return pcc
}
