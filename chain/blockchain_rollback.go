package chain

import (
	"blockEmulator/core"
	"bytes"
)

// judge this chain should fork or not.
// i.e., test the block' hash is smaller than current one at that height.
func (bc *BlockChain) JudgeForkValid_RollBack(b *core.Block) bool {
	// find the block in the current chain at that height
	if b.Header.Number > bc.CurrentBlock.Header.Number {
		return false
	}
	queryBlockHash := bc.CurrentBlock.Hash[:]
	var curBlockAtSameHeight *core.Block
	for nowheight := bc.CurrentBlock.Header.Number; nowheight >= b.Header.Number; nowheight-- {
		// get a block from db
		block, err1 := bc.Storage.GetBlock(queryBlockHash)
		if err1 != nil {
			return false
		}
		if nowheight == b.Header.Number {
			curBlockAtSameHeight = block
		}
		queryBlockHash = block.Header.ParentBlockHash
	}

	// judge whether the new block' parentHash is matched.
	if !bytes.Equal(b.Header.ParentBlockHash, curBlockAtSameHeight.Header.ParentBlockHash) {
		return false
	}

	// judge whether the new block' hash is smaller
	if bytes.Compare(curBlockAtSameHeight.Hash, b.Hash) <= 0 {
		return false
	}

	// ---------------------------
	// true, chain roll back below
	// find parent block
	ParentBlock, err1 := bc.Storage.GetBlock(b.Header.ParentBlockHash)
	if err1 != nil {
		return false
	}

	bc.CurrentBlock = ParentBlock
	bc.AddBlock(b)
	return true
}

// judge fork with a given map
func (bc *BlockChain) JudgeForkValid_RollBack_withMap(b *core.Block, chainHeight2Hash map[uint64]string, hashToBlock map[string]*core.Block) bool {
	// find the block in the current chain at that height
	if b.Header.Number > bc.CurrentBlock.Header.Number {
		return false
	}
	curBlockHashAtSameHeight, ok_hash := chainHeight2Hash[b.Header.Number]
	if !ok_hash {
		return false
	}
	curBlockAtSameHeight, ok_block := hashToBlock[curBlockHashAtSameHeight]
	if !ok_block {
		return false
	}

	// judge whether the new block' parentHash is matched.
	if !bytes.Equal(b.Header.ParentBlockHash, curBlockAtSameHeight.Header.ParentBlockHash) {
		return false
	}

	// judge whether the new block' hash is smaller
	if bytes.Compare(curBlockAtSameHeight.Hash, b.Hash) <= 0 {
		return false
	}

	// ---------------------------
	// true, chain roll back below
	// find parent block
	ParentBlock, ok_parentBlock := hashToBlock[string(b.Header.ParentBlockHash)]
	if !ok_parentBlock {
		return false
	}

	bc.CurrentBlock = ParentBlock
	bc.AddBlock(b)
	chainHeight2Hash[b.Header.Number] = string(b.Hash)
	return true
}
