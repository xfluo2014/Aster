package pbft_all

import (
	"blockEmulator/message"
	"blockEmulator/networks"
	"encoding/json"
	"log"
)

func (p *PbftConsensusNode) SendProofMsg(blockHash string, txroot []byte, txhash string, clientIp string, blockHeight uint64, isExist bool, keylist, valuelist [][]byte) {
	txocp := &message.TxOnChainProof{
		IsExist:     isExist,
		Txhash:      txhash,
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		ShardId:     p.ShardID,
		TxRoot:      txroot,
		KeyList:     keylist,
		ValueList:   valuelist,
	}
	txocpByte, err := json.Marshal(txocp)
	if err != nil {
		log.Panic(err)
	}
	send_msg := message.MergeMessage(message.CTxOnChainProof, txocpByte)

	// send message
	go networks.TcpDial(send_msg, clientIp)
}

func (p *PbftConsensusNode) handleClientTxSend(content []byte) {
	cts := new(message.ClientTxSend)
	err := json.Unmarshal(content, cts)
	if err != nil {
		log.Panic()
	}
	p.pl.Plog.Printf("S%dN%d : has received the ClientTxSend message\n", p.ShardID, p.NodeID)
	// record the tx, if this tx is on chain, then send proof back.
	p.txRequest2Client[string(cts.Tx.TxHash)] = cts.ClientIp
	p.CurChain.Txpool.AddTx2Pool(cts.Tx)
}

// handleClientRequestTxProof
func (p *PbftConsensusNode) handleClientRequestTxProof(content []byte) {
	crtxp := new(message.ClientRequestTxProof)
	err := json.Unmarshal(content, crtxp)
	if err != nil {
		log.Panic()
	}
	p.pl.Plog.Printf("S%dN%d : has received the ClientRequestTxProof message\n", p.ShardID, p.NodeID)

	isOnChain, bhash, txroot, bheight, keylist, valuelist := p.CurChain.TxOnChainVerify(crtxp.TxHash)

	p.pl.Plog.Printf("S%dN%d : TxOnChainVerify %s \n%d\n", p.ShardID, p.NodeID, bhash, len(keylist))

	p.SendProofMsg(bhash, txroot, string(crtxp.TxHash), crtxp.ClientIp, bheight, isOnChain, keylist, valuelist)
	p.pl.Plog.Printf("S%dN%d : has sent the ClientRequestTxProof message, isExist %t \n", p.ShardID, p.NodeID, isOnChain)
}

// handle account state request
func (p *PbftConsensusNode) handleRequestAccountState(content []byte) {
	cras := new(message.ClientRequestAccountState)
	err := json.Unmarshal(content, cras)
	if err != nil {
		log.Panic()
	}

	as, shardId, blockHeight, blockHash, stroot := p.CurChain.FetchAccounts([]string{cras.AccountAddr})

	// generate message and send it to client
	asc := &message.AccountStateForClient{
		AccountAddr:   cras.AccountAddr,
		AccountState:  as[0],
		ShardID:       shardId,
		BlockHeight:   blockHeight,
		BlockHash:     blockHash,
		StateTrieRoot: stroot,
	}

	ascByte, err := json.Marshal(asc)
	if err != nil {
		log.Panic(err)
	}
	send_msg := message.MergeMessage(message.CAccountStateForClient, ascByte)

	// send message
	go networks.TcpDial(send_msg, cras.ClientIp)
}
