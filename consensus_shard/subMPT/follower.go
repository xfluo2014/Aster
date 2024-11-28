package subMPT

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/trie"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	//"github.com/ethereum/go-ethereum/trie"
	"log"
	"math/big"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// FollowerNode (Follower) has the following functions:
// 1. Processing Txs received from the proposer (Executing Txs and Updating Trie stroage)
// 2. Verfying Txs received from proposer
type FollowerNode struct {
	MyLock sync.Mutex
	// Follower config
	FollowerID uint
	ProposerID uint
	// Following which proposer
	NodeID     uint
	ShardID    uint
	IPaddr     string
	ProposerIP string

	// network config
	db     ethdb.Database // to save the mpt
	triedb *trie.Database
	// logger
	lwLog *LWLog

	// tcp control
	tcpln       net.Listener
	tcpPoolLock sync.Mutex

	// stop control
	stop             atomic.Bool
	ProcessTxsPool   []*core.Transaction
	VerifyingTxsPool []*core.Transaction

	StateRoot []byte

	ProcessTxLock sync.Mutex

	FollowerNum  int
	ip_nodeTable map[string]map[uint]string

	MPTMutex sync.Mutex
}

func NewFollowerNode(sID, pID uint, fID uint, join bool) *FollowerNode {
	f := new(FollowerNode)
	f.ip_nodeTable = params.IPmap_nodeTable
	// node config setting
	f.FollowerID = fID
	f.ProposerID = pID
	f.ShardID = sID
	f.FollowerNum = params.FollowerNum
	f.lwLog = NewLwLog(int(sID), int(fID), "Follower")
	if join {
		//f.IPaddr = params.IPmap_nodeTable["f"+strconv.Itoa(int(sID))][uint(fID)]
		startPort := 40000 // 起始端口号
		endPort := 41000   // 结束端口号
		port := 0
		for port = startPort; port <= endPort; port++ {
			address := fmt.Sprintf(":%d", port)
			listener, err := net.Listen("tcp", address)
			if err == nil {
				fmt.Printf("Port %d is available\n", port)
				listener.Close() // 关闭监听器，因为我们只是检查端口是否可用
				break
			} else {
				if opError, ok := err.(*net.OpError); ok {
					if opError.Err.Error() == "address already in use" {
						fmt.Printf("Port %d is already in use\n", port)
					} else {
						fmt.Printf("Port %d: %v\n", port, err)
					}
				} else {
					fmt.Printf("Port %d: %v\n", port, err)
				}
			}
		}
		f.IPaddr = "127.0.0.1:" + strconv.Itoa(port)
		joinMsg := &message.Follower_JoinMsgs{
			Ip_port: f.IPaddr,
		}
		msgBytes, err := json.Marshal(joinMsg)
		if err != nil {
			f.lwLog.Llog.Panic("err:", err)
		}
		send_msg := message.MergeMessage(message.F_Join, msgBytes)
		networks.TcpDial(send_msg, f.ip_nodeTable["p"][f.ProposerID])

	} else {
		f.IPaddr = params.IPmap_nodeTable["f"+strconv.Itoa(int(sID))][uint(fID)]
	}

	// network config setting
	//f.IPaddr = params.IPmap_nodeTable["f"+strconv.Itoa(int(sID))][uint(fID)]
	//f.ProposerIP = params.IPmap_nodeTable["p"][uint(pID)]

	// logger config

	// stop init
	f.stop.Store(false)

	// chain config setting
	fp := "./expTest/record/ldb/s" + strconv.FormatUint(uint64(sID), 10) + "n" + strconv.FormatUint(uint64(fID), 10)
	var err error
	f.db, err = rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}

	f.triedb = trie.NewDatabaseWithConfig(f.db, &trie.Config{
		Cache:     0,
		Preimages: true,
	})

	statusTrie := trie.NewEmpty(f.triedb)
	f.StateRoot = statusTrie.Hash().Bytes()

	return f
}

func (f *FollowerNode) RunFollowerNode() {
	//stopChan := make(chan bool)

	go f.TcpListen()
	//go func() {
	//	//networks.DynamicBandwidth()
	//	time.Sleep(time.Duration(params.RunTime) * time.Second)
	//	f.lwLog.Llog.Println("send to stop chan")
	//	stopChan <- true
	//}()

	for {

		if f.stop.Load() {
			return
		}

		time.Sleep(time.Second)
		//select {
		//case <-stopChan:
		//	f.lwLog.Llog.Println("Get stop chan")
		//	f.closeFollowerNode()
		//	return
		//}
	}
}

var txs1 atomic.Int32

func (f *FollowerNode) ProcessTxs(msg *message.Allocate_TxMsgs) ([][]byte, []*core.Transaction) {
	txs := msg.Txs
	var TxHashs = make([][]byte, 0)
	if len(txs) == 0 {
		return TxHashs, make([]*core.Transaction, 0)
	}
	f.MPTMutex.Lock()
	defer f.MPTMutex.Unlock()
	st, err := trie.New(trie.TrieID(common.BytesToHash(f.StateRoot)), f.triedb)
	if err != nil {
		log.Panic(err)
	}

	for _, tx := range txs {
		tx.AllocateTime = time.Now()
	}

	now := time.Now()

	for i, tx := range txs {

		senderPrefix := tx.Sender[:1]
		recipientPrefix := tx.Recipient[:1]
		dec1, err := strconv.ParseInt(senderPrefix, 16, 32)
		if err != nil {
			fmt.Println("转换错误:", err)
			continue
		}
		dec2, err := strconv.ParseInt(recipientPrefix, 16, 32)
		if err != nil {
			fmt.Println("转换错误:", err)
			continue
		}
		destId1 := int(dec1) % f.FollowerNum
		destId2 := int(dec2) % f.FollowerNum

		// fmt.Printf("tx %d: %s, %s\n", i, tx.Sender, tx.Recipient)
		// senderIn := false
		if uint(destId1) == f.FollowerID {
			// senderIn = true
			// fmt.Printf("the sender %s is in this shard %d, \n", tx.Sender, bc.ChainConfig.ShardID)
			// modify local accountstate
			s_state_enc, _ := st.Get([]byte(tx.Sender))
			var s_state *core.AccountState
			if s_state_enc == nil {
				// fmt.Println("missing account SENDER, now adding account")
				ib := new(big.Int)
				ib.Add(ib, params.Init_Balance)
				s_state = &core.AccountState{
					Nonce:   uint64(i),
					Balance: ib,
				}
			} else {
				s_state = core.DecodeAS(s_state_enc)
			}
			s_balance := s_state.Balance
			if s_balance.Cmp(tx.Value) == -1 {
				fmt.Printf("the balance is less than the transfer amount\n")
				continue
			}
			s_state.Deduct(tx.Value)
			st.Update([]byte(tx.Sender), s_state.Encode())

		}
		// recipientIn := false
		if uint(destId2) == f.FollowerID {
			// fmt.Printf("the recipient %s is in this shard %d, \n", tx.Recipient, bc.ChainConfig.ShardID)
			// recipientIn = true
			// modify local state
			r_state_enc, _ := st.Get([]byte(tx.Recipient))
			var r_state *core.AccountState
			if r_state_enc == nil {
				// fmt.Println("missing account RECIPIENT, now adding account")
				ib := new(big.Int)
				ib.Add(ib, params.Init_Balance)
				r_state = &core.AccountState{
					Nonce:   uint64(i),
					Balance: ib,
				}
			} else {
				r_state = core.DecodeAS(r_state_enc)
			}
			r_state.Deposit(tx.Value)
			st.Update([]byte(tx.Recipient), r_state.Encode())

		}

		tx.ExecutionTime = time.Now()

		TxHashs = append(TxHashs, tx.TxHash)
	}

	Microseconds := time.Since(now).Microseconds()
	timeMsg := &message.ExecuteTimeMsg{
		Txs:          uint(len(txs)),
		FollowerId:   f.FollowerID,
		Microseconds: Microseconds,
	}
	msgBytes, err := json.Marshal(timeMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.F_ExecuteTimeMsg, msgBytes)

	f.lwLog.Llog.Printf("has handle AllocatedTxs, txhash len is %d \n", len(TxHashs))

	// Transfer this message
	go networks.TcpDial(send_msg, f.ip_nodeTable["p"][f.ProposerID])

	rt, ns := st.Commit(false)

	err = f.triedb.Update(trie.NewWithNodeSet(ns))
	if err != nil {
		log.Panic()
	}
	err = f.triedb.Commit(rt, false)
	if err != nil {
		log.Panic(err)
	}
	f.StateRoot = rt.Bytes()

	//dirpath := fmt.Sprintf("%s/mptlatency/", params.DataWrite_path)
	//// create or open the CSV file
	//if _, err := os.Stat(dirpath); os.IsNotExist(err) {
	//	err := os.MkdirAll(dirpath, 0775)
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	//if err != nil {
	//	//	w.pl.Plog.Printf("S%dN%d : error creating the result directory %v \n", w.ShardID, w.NodeID, err)
	//	//}
	//}
	//file, err := os.OpenFile(dirpath+fmt.Sprintf("/MPTLatency"+strconv.Itoa(int(f.FollowerID))+".csv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer file.Close()
	//
	////txs1.Add(int32(len(txs)))
	//// write the timestamp to the CSV file
	//writer := csv.NewWriter(file)
	//err = writer.Write([]string{
	//	strconv.Itoa(int(txs1.Add(int32(len(txs))))),
	//	strconv.Itoa(int(milliseconds)),
	//})
	//if err != nil {
	//	fmt.Println(err)
	//}
	//writer.Flush()

	return TxHashs, msg.Txs
}

func (f *FollowerNode) VerfyingTxs(msg *message.Verfying_TxMsgs) {

}

func (f *FollowerNode) closeFollowerNode() {
	f.lwLog.Llog.Println("Now close Follower node ...")
	//write_csv(int(f.ShardID), int(f.FollowerID), f.db)
	f.stop.Store(true)
	err := f.tcpln.Close()
	if err != nil {
		return
	}
}
