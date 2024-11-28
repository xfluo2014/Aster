package message

import (
	"blockEmulator/core"
)

// message that client send
var (
	CClientTxSend              MessageType = "ClientTxSend"
	CClientRequestTxProof      MessageType = "ClientRequestTxProof"
	CClientRequestAccountState MessageType = "RequestAccountState"
)

// client send tx to workers
type ClientTxSend struct {
	Tx       *core.Transaction
	ClientIp string
}

// client send a request for the proof of a tx
type ClientRequestTxProof struct {
	TxHash   []byte
	ClientIp string
}

// client send a request to workers to get account state
type ClientRequestAccountState struct {
	AccountAddr string
	ClientIp    string
}

// message that client received
var (
	CTxOnChainProof        MessageType = "TxOnChainProof"
	CAccountStateForClient MessageType = "AccountStateForClient"
)

// workers give the proof of a given tx
type TxOnChainProof struct {
	IsExist     bool
	TxRoot      []byte
	Txhash      string
	BlockHash   string
	ShardId     uint64
	BlockHeight uint64
	// need a data structrue more for proof.
	KeyList   [][]byte
	ValueList [][]byte
}

// worker send account state to a client
type AccountStateForClient struct {
	AccountAddr   string
	ShardID       int
	BlockHeight   int
	BlockHash     []byte
	StateTrieRoot []byte
	AccountState  *core.AccountState
}
