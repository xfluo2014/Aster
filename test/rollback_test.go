package test

import (
	"blockEmulator/chain"
	"blockEmulator/core"
	"blockEmulator/params"
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestRollBack(t *testing.T) {
	accounts := []string{"000000000001", "00000000002", "00000000003", "00000000004", "00000000005", "00000000006"}
	as := make([]*core.AccountState, 0)
	for idx := range accounts {
		as = append(as, &core.AccountState{
			Balance: big.NewInt(int64(idx)),
		})
	}
	fp := "./record/ldb/s0/N0"
	db, err := rawdb.NewLevelDBDatabase(fp, 0, 1, "accountState", false)
	if err != nil {
		log.Panic(err)
	}
	params.ShardNum = 1
	pcc := &params.ChainConfig{
		ChainID:        0,
		NodeID:         0,
		ShardID:        0,
		Nodes_perShard: uint64(1),
		ShardNums:      4,
		BlockSize:      uint64(params.MaxBlockSize_global),
		BlockInterval:  uint64(params.Block_Interval),
		InjectSpeed:    uint64(params.InjectSpeed),
	}
	CurChain, _ := chain.NewBlockChain(pcc, db)
	CurChain.PrintBlockChain()
	CurChain.AddAccounts(accounts, as)
	CurChain.PrintBlockChain()

	// generate blocks in the first chain
	b1 := CurChain.GenerateBlock(0)
	CurChain.AddBlock(b1)
	CurChain.PrintBlockChain()

	b2 := CurChain.GenerateBlock(0)

	// enumerate the block miner, so the blockHash is changed.
	// generate a block with smaller blockHash
	nowMiner := 1
	new_b2 := CurChain.GenerateBlock(nowMiner)
	for {
		if bytes.Compare(new_b2.Hash, b2.Hash) == -1 {
			break
		}
		nowMiner++
		new_b2 = CurChain.GenerateBlock(nowMiner)
	}

	CurChain.AddBlock(b2)
	CurChain.PrintBlockChain()

	b3 := CurChain.GenerateBlock(0)
	CurChain.AddBlock(b3)
	CurChain.PrintBlockChain()

	b4 := CurChain.GenerateBlock(0)
	CurChain.AddBlock(b4)
	CurChain.PrintBlockChain()

	// try to fork by using block new_b2
	CurChain.JudgeForkValid_RollBack(new_b2)
	CurChain.PrintBlockChain()

	b5 := CurChain.GenerateBlock(0)
	CurChain.AddBlock(b5)
	CurChain.PrintBlockChain()

	CurChain.CloseBlockChain()
}
