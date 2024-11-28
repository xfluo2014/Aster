package chain

import (
	"blockEmulator/core"

	"blockEmulator/trie"
)

func isSmartContractTx(tx *core.Transaction) bool {
	return false
}

func UpdateBySmartContract(tx *core.Transaction, tr *trie.Trie) {

}
