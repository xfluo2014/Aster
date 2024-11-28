package subMPT

import (
	"blockEmulator/core"
	"blockEmulator/message"
	"blockEmulator/networks"
	"blockEmulator/params"
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// light weight node begin listen message
func (lw *ProposerNode) TcpListen() {
	lw.IPaddr = "0.0.0.0:" + strings.Split(lw.IPaddr, ":")[1]
	ln, err := net.Listen("tcp", lw.IPaddr)
	lw.tcpln = ln
	if err != nil {
		lw.lwLog.Llog.Panic("err:", err)
	}
	for {
		conn, err := lw.tcpln.Accept()
		if err != nil {
			return
		}
		go lw.preHandleMessageRequest(conn)
	}
}

// handle
func (p *ProposerNode) preHandleMessageRequest(con net.Conn) {
	defer con.Close()
	clientReader := bufio.NewReader(con)
	for {
		clientRequest, err := clientReader.ReadBytes('\n')
		if p.stop.Load() {
			return
		}
		switch err {
		case nil:
			p.tcpPoolLock.Lock()
			p.handleMessage(clientRequest)
			p.tcpPoolLock.Unlock()
		case io.EOF:
			log.Println("client closed the connection by terminating the process")
			return
		default:
			log.Printf("error: %v\n", err)
			return
		}
	}
}

func (p *ProposerNode) handleMessage(msg []byte) {
	msgType, content := message.SplitMessage(msg)
	switch msgType {
	case message.P_BlockBroadcast:

		go p.handleBlockMsg(content)
	case message.F_Precessed:
		go p.handleProcessedTxs(content)
	case message.F_Join:
		go p.handleFollowerJoin(content)
	case message.F_ResponseData:
		go p.handleFollowerResponse(content)
	case message.F_Ready:
		go p.handleFollowerReady(content)
	case message.F_ExecuteTimeMsg:
		go p.handleExecuteTimeMsg(content)

	}
}

func (p *ProposerNode) handleBlockMsg(content []byte) {
	p.VerifyLock.Lock()
	blk := new(message.BlockMsg)
	err := json.Unmarshal(content, blk)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	pBlock := core.DecodeB(blk.EncodedBlock)

	// log here
	p.lwLog.Llog.Printf("Block Msg is received from %d\n", blk.ProposerID)

	p.VerifyBlock = pBlock
	p.VerifyingBlock(pBlock)

}

func (p *ProposerNode) handleProcessedTxs(content []byte) {
	TxsMsg := new(message.Precessed_TxMsgs)
	err := json.Unmarshal(content, TxsMsg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	// log here
	p.lwLog.Llog.Printf("Precessed Txs are received\n")

	p.GatherTxs(TxsMsg)

	p.GatherLock.Lock()

	if _, exists := p.GatherMap[TxsMsg.EpochId]; !exists {
		p.GatherMap[TxsMsg.EpochId] = make(map[uint][]byte)
	}
	p.GatherMap[TxsMsg.EpochId][TxsMsg.FollowerID] = TxsMsg.StateRoot
	if len(p.GatherMap[TxsMsg.EpochId]) == params.FollowerNum {
		if p.ShardID == 0 {
			go p.blockGenerate(TxsMsg.EpochId)
		} else {
			//states := make([]byte, 0)
			////p.GatherLock.Lock()
			//for i := uint(0); i < uint(params.FollowerNum); i++ {
			//	states = append(states, p.GatherMap[TxsMsg.EpochId][i]...)
			//}
			//
			////p.GatherLock.Unlock()
			//
			//var buff bytes.Buffer
			//
			//enc := gob.NewEncoder(&buff)
			//err := enc.Encode(states)
			//if err != nil {
			//	log.Panic(err)
			//}
			//hash := sha256.Sum256(buff.Bytes())
			//if string(p.VerifyBlock.Header.StateRoot) == string(hash[:]) {
			//	p.lwLog.Llog.Printf("Block is valid \n")
			//} else {
			//	p.lwLog.Llog.Printf("Block is invalid \n")
			//}
			//p.VerifyLock.Unlock()
		}
	}

	p.GatherLock.Unlock()
}

func (p *ProposerNode) handleFollowerJoin(content []byte) {
	JoinMsg := new(message.Follower_JoinMsgs)
	err := json.Unmarshal(content, JoinMsg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	newNodeId := len(p.FollowerIDs)
	p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(newNodeId)] = JoinMsg.Ip_port
	params.IPmap_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(newNodeId)] = JoinMsg.Ip_port

	allocFidMsg := &message.P_AllocFidMsgs{
		Fid: uint(newNodeId),
	}
	msgBytes, err := json.Marshal(allocFidMsg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	send_msg := message.MergeMessage(message.P_AllocFId, msgBytes)
	networks.TcpDial(send_msg, p.ip_nodeTable["f"+strconv.Itoa(int(p.NodeID))][uint(newNodeId)])

	ctx := context.Background()
	p.ConsistentHash.AddNode(ctx, strconv.Itoa(newNodeId), 1)

	params.FollowerNum++
	p.FollowerIDs = append(p.FollowerIDs, uint(newNodeId))
}

func (p *ProposerNode) handleFollowerResponse(content []byte) {
	Msg := new(message.F_ResponseDataMsgs)
	err := json.Unmarshal(content, Msg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	p.ReceiveRequestDataMapLock.Lock()
	p.ReceiveRequestDataMap[Msg.Fid] = Msg
	p.ReceiveRequestDataMapLock.Unlock()
}

func (p *ProposerNode) handleFollowerReady(content []byte) {
	Msg := new(message.F_ReadyMsgs)
	err := json.Unmarshal(content, Msg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	p.ReadyMapLock.Lock()
	defer p.ReadyMapLock.Unlock()
	p.ReadyMap[Msg.Fid] = true
}

func (p *ProposerNode) handleExecuteTimeMsg(content []byte) {
	Msg := new(message.ExecuteTimeMsg)
	err := json.Unmarshal(content, Msg)
	if err != nil {
		p.lwLog.Llog.Panic("err:", err)
	}
	p.lock1.Lock()
	defer p.lock1.Unlock()

	dirpath := fmt.Sprintf("%s/mptlatency/", params.DataWrite_path)
	// create or open the CSV file
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		err := os.MkdirAll(dirpath, 0775)
		if err != nil {
			fmt.Println(err)
		}
	}
	file, err := os.OpenFile(dirpath+fmt.Sprintf("/MPTLatency.csv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	writer := csv.NewWriter(file)
	fileInfo, err := file.Stat()
	if err != nil {
		log.Panic(err)
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write([]string{"idx", "Txs", "TotalTxs", "Microseconds(this Txs)"}); err != nil {
			log.Panic(err)
		}
		writer.Flush()
	}
	if err != nil {
		log.Fatal(err)
	}

	err = writer.Write([]string{
		strconv.Itoa(int(p.Idx)),
		strconv.Itoa(int(Msg.Txs)),
		strconv.Itoa(int(p.txcount.Load())),
		strconv.Itoa(int(Msg.Microseconds)),
	})
	p.Idx += 1
	if err != nil {
		fmt.Println(err)
	}
	writer.Flush()

}
