package message

var (
	LW_BlockBroadcast1 MessageType = "lightWeight_block_Broadcast"
	LW_SignBroadcast1  MessageType = "lightWeight_sign_Broadcast"
)

type LW_BlockMsg1 struct {
	// Block      core.Block
	EncodedBlock []byte
	FromNodeID   uint
}

type LW_SignMsg1 struct {
	Signature  []byte
	FromNodeID uint
	BlockHash  []byte
}
