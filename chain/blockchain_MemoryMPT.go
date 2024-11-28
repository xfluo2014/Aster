// Here the BlockChain structrue is defined
// each node in this system will maintain a BlockChain object.

package chain

import (
	"blockEmulator/core"
	"blockEmulator/params"
	"blockEmulator/storage"
	"blockEmulator/utils"
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"blockEmulator/trie"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type BlockChain struct {
	db              ethdb.Database      // the leveldb database to store in the disk, for status trie
	triedb          *trie.Database      // the trie database which helps to store the status trie
	ChainConfig     *params.ChainConfig // the chain configuration, which can help to identify the chain
	CurrentBlock    *core.Block         // the top block in this BlockChain
	Storage         *storage.Storage    // Storage is the bolt-db to store the blocks
	PrecessedTxpool *core.TxPool        // the transaction pool
	PartitionMap    map[string]uint64   // the partition map which is defined by some algorithm can help account parition
	pmlock          sync.RWMutex

	Txpool *core.TxPool
}

// Write Partition Map
func (bc *BlockChain) Update_PartitionMap(key string, val uint64) {
	bc.pmlock.Lock()
	defer bc.pmlock.Unlock()
	bc.PartitionMap[key] = val
}

// Get parition (if not exist, return default)
func (bc *BlockChain) Get_PartitionMap(key string) uint64 {
	bc.pmlock.RLock()
	defer bc.pmlock.RUnlock()
	if _, ok := bc.PartitionMap[key]; !ok {
		return uint64(utils.Addr2Shard(key))
	}
	return bc.PartitionMap[key]
}

// Send a transaction to the pool (need to decide which pool should be sended)
func (bc *BlockChain) SendTx2Pool(txs []*core.Transaction) {
	bc.PrecessedTxpool.AddTxs2Pool(txs)
}

// handle transactions and modify the status trie
func (bc *BlockChain) GetUpdateStatusTrie(txs []*core.Transaction) common.Hash {
	// return common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	// the empty block (length of txs is 0) condition
	if len(txs) == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	}
	// build trie from the triedb (in disk)
	beginTime := time.Now()
	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
	fmt.Println("Trie New Time cost: ", time.Since(beginTime).Milliseconds())
	if err != nil {
		log.Panic(err)
	}
	cnt := 0
	// handle transactions, the signature check is ignored here
	beginTime = time.Now()
	for i, tx := range txs {
		if !tx.Relayed && (bc.Get_PartitionMap(tx.Sender) == bc.ChainConfig.ShardID || tx.HasBroker) {
			// modify local accountstate
			s_state_enc, _ := st.Get([]byte(tx.Sender))
			var s_state *core.AccountState
			if s_state_enc == nil {
				ib := new(big.Int)
				ib.Add(ib, params.Init_Balance)
				s_state = &core.AccountState{
					Nonce:   uint64(i),
					Balance: ib,
				}
			} else {
				s_state = core.DecodeAS(s_state_enc)
			}
			if s_state.Balance.Cmp(tx.Value) == -1 {
				fmt.Printf("the balance is less than the transfer amount\n")
				continue
			}
			s_state.Deduct(tx.Value)
			st.Update([]byte(tx.Sender), s_state.Encode())
		}

		if bc.Get_PartitionMap(tx.Recipient) == bc.ChainConfig.ShardID || tx.HasBroker {
			// modify local state
			r_state_enc, _ := st.Get([]byte(tx.Recipient))
			var r_state *core.AccountState
			if r_state_enc == nil {
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
	}
	fmt.Println("St Update Time cost: ", time.Since(beginTime).Milliseconds())
	// commit the memory trie to the database in the disk
	if cnt == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	}
	beginTime = time.Now()
	rt, ns := st.Commit(false)
	err = bc.triedb.Update(trie.NewWithNodeSet(ns))
	if err != nil {
		log.Panic()
	}
	err = bc.triedb.Commit(rt, false)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("Trie Store Time cost: ", time.Since(beginTime).Milliseconds())
	fmt.Println("modified account number is ", cnt)
	return rt
}

// generate (mine) a block, this function return a block
func (bc *BlockChain) GenerateBlock(miner int) *core.Block {
	// pack the transactions from the txpool
	txs := bc.PrecessedTxpool.PackTxs(bc.ChainConfig.BlockSize)
	// handle transactions to build root
	rt := bc.GetUpdateStatusTrie(txs)
	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash[:],
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Now(),
	}
	bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(txs)
	bh.BloomFilter = *GetBloomFilter(txs)
	b := core.NewBlock(bh, txs)
	b.Header.Miner = uint64(miner)
	b.Hash = b.Header.Hash()
	return b
}

func (bc *BlockChain) GenerateBlock2(miner int, ProcessedTxpool *[]*core.Transaction) *core.Block {
	// pack the transactions from the txpool

	txNum := bc.ChainConfig.BlockSize
	if uint64(len(*ProcessedTxpool)) < txNum {
		txNum = uint64(len(*ProcessedTxpool))
	}
	txs1 := (*ProcessedTxpool)[:txNum]
	*ProcessedTxpool = (*ProcessedTxpool)[txNum:]

	// handle transactions to build root
	//rt := bc.GetUpdateStatusTrie(txs)
	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash[:],
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Now(),
	}
	//bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(txs1)
	bh.BloomFilter = *GetBloomFilter(txs1)
	b := core.NewBlock(bh, txs1)
	b.Header.Miner = uint64(miner)
	b.Hash = b.Header.Hash()
	return b
}

// new a genisis block, this func will be invoked only once for a BlockChain object
func (bc *BlockChain) NewGenisisBlock() *core.Block {
	body := make([]*core.Transaction, 0)
	bh := &core.BlockHeader{
		Number: 0,
	}
	// build a new trie database by db
	triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
	bc.triedb = triedb
	statusTrie := trie.NewEmpty(triedb)
	bh.StateRoot = statusTrie.Hash().Bytes()
	bh.TxRoot = GetTxTreeRoot(body)
	bh.BloomFilter = *GetBloomFilter(body)
	b := core.NewBlock(bh, body)
	b.Hash = b.Header.Hash()
	return b
}

// add the genisis block in a BlockChain
func (bc *BlockChain) AddGenisisBlock(gb *core.Block) {
	bc.Storage.AddBlock(gb)
	newestHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		log.Panic()
	}
	curb, err := bc.Storage.GetBlock(newestHash)
	if err != nil {
		log.Panic()
	}
	bc.CurrentBlock = curb
}

// add a block
func (bc *BlockChain) AddBlock(b *core.Block) bool {
	if b == nil {
		fmt.Println("this block is nil")
		return false
	}

	if b.Header.Number != bc.CurrentBlock.Header.Number+1 {
		fmt.Println("the block height is not correct")
		return false
	}
	if !bytes.Equal(b.Header.ParentBlockHash, bc.CurrentBlock.Hash) {
		fmt.Println("err parent block hash")
		return false
	}

	// if the treeRoot is existed in the node, the transactions is no need to be handled again
	// if b.Header.Miner != bc.ChainConfig.NodeID {
	// 	rt := bc.GetUpdateStatusTrie(b.Body)
	// 	fmt.Println(bc.CurrentBlock.Header.Number+1, "the root = ", rt.Bytes())
	// }
	// _, err := trie.(trie.TrieID(common.BytesToHash(b.Header.StateRoot)), bc.triedb)
	//_, err := bc.db.Get(b.Header.StateRoot)
	//if err != nil {
	//	rt := bc.GetUpdateStatusTrie(b.Body)
	//	fmt.Println(bc.CurrentBlock.Header.Number+1, "the root = ", rt.Bytes())
	//}
	bc.CurrentBlock = b
	bc.Storage.AddBlock(b)
	return true
}

// new a BlockChain.
// the ChainConfig is pre-defined to identify the BlockChain; the db is the status trie database in disk
func NewBlockChain(cc *params.ChainConfig, db ethdb.Database) (*BlockChain, error) {
	fmt.Println("Generating a new BlockChain", db)
	bc := &BlockChain{
		db:              db,
		ChainConfig:     cc,
		PrecessedTxpool: core.NewTxPool(),
		Storage:         storage.NewStorage(cc),
		PartitionMap:    make(map[string]uint64),
	}
	curHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		fmt.Println("Get newest block hash err")
		// if the Storage bolt database cannot find the newest blockhash,
		// it means the BlockChain should be built in height = 0
		if err.Error() == "cannot find the newest block hash" {
			genisisBlock := bc.NewGenisisBlock()
			bc.AddGenisisBlock(genisisBlock)
			fmt.Println("New genisis block")
			return bc, nil
		}
		log.Panic()
	}

	// there is a BlockChain in the storage
	fmt.Println("Existing BlockChain found")
	curb, err := bc.Storage.GetBlock(curHash)
	if err != nil {
		log.Panic()
	}

	bc.CurrentBlock = curb
	triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
	bc.triedb = triedb
	// check the existence of the trie database
	_, err = trie.New(trie.TrieID(common.BytesToHash(curb.Header.StateRoot)), triedb)
	if err != nil {
		log.Panic()
	}
	fmt.Println("The status trie can be built")
	fmt.Println("Generated a new BlockChain successfully")
	return bc, nil
}

// check a block is valid or not in this BlockChain config
func (bc *BlockChain) IsValidBlock(b *core.Block) error {
	if string(b.Header.ParentBlockHash) != string(bc.CurrentBlock.Hash) {
		fmt.Println("the parentblock hash is not equal to the current block hash")
		return errors.New("the parentblock hash is not equal to the current block hash")
	} else if string(GetTxTreeRoot(b.Body)) != string(b.Header.TxRoot) {
		fmt.Println("the transaction root is wrong")
		return errors.New("the transaction root is wrong")
	}
	return nil
}

// add accounts
func (bc *BlockChain) AddAccounts(ac []string, as []*core.AccountState) {
	fmt.Printf("The len of accounts is %d, now adding the accounts\n", len(ac))

	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash[:],
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Time{},
	}
	// handle transactions to build root
	rt := common.BytesToHash(bc.CurrentBlock.Header.StateRoot)
	if len(ac) != 0 {
		st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
		if err != nil {
			log.Panic(err)
		}
		for i, addr := range ac {
			if bc.Get_PartitionMap(addr) == bc.ChainConfig.ShardID {
				ib := new(big.Int)
				ib.Add(ib, as[i].Balance)
				new_state := &core.AccountState{
					Balance: ib,
					Nonce:   as[i].Nonce,
				}
				st.Update([]byte(addr), new_state.Encode())
			}
		}
		rrt, ns := st.Commit(false)
		err = bc.triedb.Update(trie.NewWithNodeSet(ns))
		if err != nil {
			log.Panic(err)
		}
		err = bc.triedb.Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}
		rt = rrt
	}

	emptyTxs := make([]*core.Transaction, 0)
	bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(emptyTxs)
	bh.BloomFilter = *GetBloomFilter(emptyTxs)
	b := core.NewBlock(bh, emptyTxs)
	b.Header.Miner = 0
	b.Hash = b.Header.Hash()

	bc.CurrentBlock = b
	bc.Storage.AddBlock(b)
}

// check a transaction is on-chain or not.
func (bc *BlockChain) TxOnChainVerify(txhash []byte) (bool, string, []byte, uint64, [][]byte, [][]byte) {
	keylist, valuelist := make([][]byte, 0), make([][]byte, 0)
	bitMapIdxofTx := utils.ModBytes(txhash, 2048)
	nowblockHash := bc.CurrentBlock.Hash
	nowheight := bc.CurrentBlock.Header.Number
	for ; nowheight > 0; nowheight-- {
		// get a block from db
		block, err1 := bc.Storage.GetBlock(nowblockHash)
		if err1 != nil {
			return false, err1.Error(), nil, 0, keylist, valuelist
		}

		// If no value in bloom filter, then the tx must not be in this block
		if !block.Header.BloomFilter.Test(bitMapIdxofTx) {
			nowblockHash = block.Header.ParentBlockHash
			continue
		}

		// now try to find whether this tx is in this block
		isdone := false

		// further work: use merkle proof
		triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
		transactionTree := trie.NewEmpty(triedb)
		for _, tx := range block.Body {
			transactionTree.Update(tx.TxHash, tx.Encode())
		}
		if !bytes.Equal(transactionTree.Hash().Bytes(), block.Header.TxRoot) {
			return false, "err Tx root", nil, 0, keylist, valuelist
		}
		proof := rawdb.NewMemoryDatabase()
		if err := transactionTree.Prove(txhash, 0, proof); err == nil {
			isdone = true
			it := proof.NewIterator(nil, nil)
			for it.Next() {
				keylist = append(keylist, it.Key())
				valuelist = append(valuelist, it.Value())
			}

			// test whether this proof is right or not
			proof2 := rawdb.NewMemoryDatabase()
			listLen := len(keylist)
			for i := 0; i < listLen; i++ {
				proof2.Put(keylist[i], valuelist[i])
			}
			if _, err := trie.VerifyProof(common.Hash(block.Header.TxRoot), txhash, proof2); err != nil {
				return false, "err verify", nil, 0, keylist, valuelist
			}
		}

		if isdone {
			return true, string(nowblockHash), (block.Header.TxRoot), nowheight, keylist, valuelist
		} else {
			nowblockHash = block.Header.ParentBlockHash
		}
	}
	return false, "no block has this tx", nil, 0, keylist, valuelist
}

// fetch accounts
func (bc *BlockChain) FetchAccounts(addrs []string) ([]*core.AccountState, int, int, []byte, []byte) {
	res := make([]*core.AccountState, 0)
	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
	if err != nil {
		log.Panic(err)
	}
	for _, addr := range addrs {
		asenc, _ := st.Get([]byte(addr))
		var state_a *core.AccountState
		if asenc == nil {
			ib := new(big.Int)
			ib.Add(ib, params.Init_Balance)
			state_a = &core.AccountState{
				Nonce:   uint64(0),
				Balance: ib,
			}
		} else {
			state_a = core.DecodeAS(asenc)
		}
		res = append(res, state_a)
	}
	return res, int(bc.ChainConfig.ShardID), int(bc.CurrentBlock.Header.Number), bc.CurrentBlock.Hash, bc.CurrentBlock.Header.StateRoot
}

// close a BlockChain, close the database inferfaces
func (bc *BlockChain) CloseBlockChain() {
	bc.Storage.DataBase.Close()
	bc.triedb.CommitPreimages()
}

// print the details of a BlockChain
func (bc *BlockChain) PrintBlockChain() string {
	vals := []interface{}{
		bc.CurrentBlock.Header.Number,
		bc.CurrentBlock.Hash,
		bc.CurrentBlock.Header.StateRoot,
		bc.CurrentBlock.Header.Time,
		bc.triedb,
	}
	res := fmt.Sprintf("%v\n", vals)
	fmt.Println(res)
	return res
}
