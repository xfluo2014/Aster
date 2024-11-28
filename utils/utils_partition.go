package utils

import (
	"blockEmulator/params"
	"log"
	"math/big"
	"strconv"
)

// the default method
func Addr2Shard(addr Address) int {
	last16_addr := addr
	if len(addr) > 16 {
		last16_addr = addr[len(addr)-16:]
	}
	num, err := strconv.ParseUint(last16_addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	return int(num) % params.ShardNum
}

// mod method
func ModBytes(data []byte, mod uint) uint {
	num := new(big.Int).SetBytes(data)
	result := new(big.Int).Mod(num, big.NewInt(int64(mod)))
	return uint(result.Int64())
}
