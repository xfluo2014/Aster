package message

import (
	"blockEmulator/core"
)

var (
	P_TxInject       MessageType = "Proposer_Tx_Inject"
	P_BlockBroadcast MessageType = "Proposer_block_Broadcast"
	P_TxAllocate     MessageType = "Proposer_Tx_Allocate"
	F_TxAllocated    MessageType = "Follower_Tx_Allocated"
	F_TxVerfied      MessageType = "Follower_Tx_Verfied"
	F_Precessed      MessageType = "Follower_Precessed"
	F_Join           MessageType = "Follower_Join"
	P_RequestData    MessageType = "Proposer_RequestData"
	P_AllocateData   MessageType = "Proposer_AllocateData"
	F_ResponseData   MessageType = "Follower_ResponseData"
	F_Ready          MessageType = "Follower_Ready"
	P_AllocFId       MessageType = "P_AllocFId"

	F_ExecuteTimeMsg MessageType = "ExecuteTimeMsg"
)

type Inject_TxMsg struct {
	// Block      core.Block
	Txs        []byte
	ProposerID uint
}

type BlockMsg struct {
	// Block      core.Block
	EncodedBlock []byte
	ProposerID   uint
}

type Allocate_TxMsgs struct {
	Txs        []*core.Transaction
	ProposerID uint
	FollowerID uint
	EpochId    uint
}

type Precessed_TxMsgs struct {
	Txs        []*core.Transaction
	TxHashs    [][]byte
	Proofs     []byte
	FollowerID uint
	EpochId    uint
	StateRoot  []byte
}

type Verfying_TxMsgs struct {
	TxIDs       []byte
	Txroot      []byte
	BlockHash   uint
	BlockHeight uint
}

type Follower_JoinMsgs struct {
	Ip_port string
}

type P_RequestDataMsgs struct {
	Keys []string
}

type F_ResponseDataMsgs struct {
	Fid   uint
	Keys  []string
	Value []*core.AccountState
}

type F_ReadyMsgs struct {
	Fid uint
}

type P_AllocateDataMsgs struct {
	Keys  []string
	Value []*core.AccountState
}

type P_AllocFidMsgs struct {
	Fid uint
}

type ExecuteTimeMsg struct {
	FollowerId   uint
	Txs          uint
	Microseconds int64
}
