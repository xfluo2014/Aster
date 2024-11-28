package subMPT

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"blockEmulator/trie"
	"bufio"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"log"
	"math/big"
	"net"
	"strings"
)

// light weight node begin listen message
func (f *FollowerNode) TcpListen() {
	f.IPaddr = "0.0.0.0:" + strings.Split(f.IPaddr, ":")[1]
	ln, err := net.Listen("tcp", f.IPaddr)
	f.tcpln = ln
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	for {
		conn, err := f.tcpln.Accept()
		if err != nil {
			return
		}
		go f.preHandleMessageRequest(conn)
	}
}

// handle
func (f *FollowerNode) preHandleMessageRequest(con net.Conn) {
	defer con.Close()
	clientReader := bufio.NewReader(con)
	for {
		if f.stop.Load() {
			return
		}
		clientRequest, err := clientReader.ReadBytes('\n')
		if f.stop.Load() {
			return
		}
		switch err {
		case nil:
			f.tcpPoolLock.Lock()
			f.handleMessage(clientRequest)
			f.tcpPoolLock.Unlock()
		case io.EOF:
			log.Println("client closed the connection by terminating the process")
			return
		default:
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (f *FollowerNode) handleMessage(msg []byte) {
	msgType, content := message.SplitMessage(msg)
	switch msgType {
	case message.P_TxAllocate:
		go f.handleAllocatedTxs(content)
	case message.F_TxVerfied:
		f.handleVerifyTxs(content)
	case message.CStop:
		f.closeFollowerNode()
	case message.P_RequestData:
		go f.handleProposerRequest(content)
	case message.P_AllocateData:
		go f.handleProposerAllocateData(content)
	case message.P_AllocFId:
		go f.handleProposerAllocateFID(content)
	}
}

func (f *FollowerNode) handleAllocatedTxs(content []byte) {
	f.ProcessTxLock.Lock()
	defer f.ProcessTxLock.Unlock()
	TxsMsg := new(message.Allocate_TxMsgs)
	err := json.Unmarshal(content, TxsMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	// log here
	f.lwLog.Llog.Printf("Allocated Txs are received! Tx size is %d\n", len(TxsMsg.Txs))

	f.MyLock.Lock()
	TxHashs, Txs1 := f.ProcessTxs(TxsMsg)
	f.MyLock.Unlock()
	txMsg := &message.Precessed_TxMsgs{
		Txs:        Txs1,
		TxHashs:    TxHashs,
		FollowerID: f.FollowerID,
		Proofs:     make([]byte, 0),
		EpochId:    TxsMsg.EpochId,
		StateRoot:  f.StateRoot,
	}
	msgBytes, err := json.Marshal(txMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.F_Precessed, msgBytes)

	f.lwLog.Llog.Printf("has handle AllocatedTxs, txhash len is %d \n", len(TxHashs))

	// Transfer this message
	networks.TcpDial(send_msg, f.ip_nodeTable["p"][f.ProposerID])
	f.lwLog.Llog.Printf("has send gather msg\n")
}

func (f *FollowerNode) handleVerifyTxs(content []byte) {
	TxsMsg := new(message.Verfying_TxMsgs)
	err := json.Unmarshal(content, TxsMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	// log here
	f.lwLog.Llog.Printf("Precessed Txs are received\n")

	f.VerfyingTxs(TxsMsg)

}

func (f *FollowerNode) handleProposerRequest(content []byte) {
	ReqMsg := new(message.P_RequestDataMsgs)
	err := json.Unmarshal(content, ReqMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}

	f.MPTMutex.Lock()
	defer f.MPTMutex.Unlock()
	st, err := trie.New(trie.TrieID(common.BytesToHash(f.StateRoot)), f.triedb)

	Keys := make([]string, 0)
	acs := make([]*core.AccountState, 0)
	for _, key := range ReqMsg.Keys {
		s_state_enc, _ := st.Get([]byte(key))
		var s_state *core.AccountState
		if s_state_enc == nil {
			// fmt.Println("missing account SENDER, now adding account")
			ib := new(big.Int)
			ib.Add(ib, params.Init_Balance)
			s_state = &core.AccountState{
				Nonce:   uint64(0),
				Balance: ib,
			}
		} else {
			s_state = core.DecodeAS(s_state_enc)
		}
		acs = append(acs, s_state)
		Keys = append(Keys, key)
	}

	responseDataMsg := &message.F_ResponseDataMsgs{
		Fid:   f.FollowerID,
		Keys:  Keys,
		Value: acs,
	}
	msgBytes, err := json.Marshal(responseDataMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.F_ResponseData, msgBytes)
	networks.TcpDial(send_msg, f.ip_nodeTable["p"][f.ProposerID])

}

func (f *FollowerNode) handleProposerAllocateData(content []byte) {
	AllocMsg := new(message.P_AllocateDataMsgs)
	err := json.Unmarshal(content, AllocMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	f.MPTMutex.Lock()
	defer f.MPTMutex.Unlock()
	st, err := trie.New(trie.TrieID(common.BytesToHash(f.StateRoot)), f.triedb)
	for idx, key := range AllocMsg.Keys {
		st.Update([]byte(key), AllocMsg.Value[idx].Encode())
	}
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
	readyDataMsg := &message.F_ReadyMsgs{
		Fid: f.FollowerID,
	}
	msgBytes, err := json.Marshal(readyDataMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.F_Ready, msgBytes)
	networks.TcpDial(send_msg, f.ip_nodeTable["p"][f.ProposerID])
}

func (f *FollowerNode) handleProposerAllocateFID(content []byte) {
	AllocMsg := new(message.P_AllocFidMsgs)
	err := json.Unmarshal(content, AllocMsg)
	if err != nil {
		f.lwLog.Llog.Panic("err:", err)
	}
	f.FollowerID = AllocMsg.Fid
}
