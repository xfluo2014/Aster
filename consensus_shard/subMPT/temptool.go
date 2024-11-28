package subMPT

import (
	"blockEmulator/core"
	"blockEmulator/params"
	rand2 "crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func last16bitsToInt(data []byte) int {
	if len(data) < 2 {
		panic("Byte slice length must be at least 2")
	}

	// 截取最后 2 个字节
	last16bits := data[len(data)-2:]

	// 将字节序列转换为 int
	var result int
	for _, b := range last16bits {
		result = result<<8 + int(b)
	}

	return result
}
func generateRandomHexString(length int) (string, error) {
	// 计算所需的字节数，因为每个 16 进制字符代表 4 位二进制（即半个字节）
	// 所以长度为 40 的 16 进制字符串需要 20 个字节
	bytes := make([]byte, length/2)
	if _, err := io.ReadFull(rand2.Reader, bytes); err != nil {
		return "", err
	}
	// 将字节数组转换为 16 进制字符串
	return hex.EncodeToString(bytes), nil
}
func ProposerGenerateTxs(txSize int, uniqueSource int) []*core.Transaction {
	txs := make([]*core.Transaction, 0)

	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(uniqueSource)))
	for i := 0; i < txSize; i++ {
		hexString1, _ := generateRandomHexString(40)
		hexString2, _ := generateRandomHexString(40)
		//payerInt := r.Int63n(200000) + 123456
		//payeeInt := r.Int63n(200000) + 123456
		valueInt := r.Int63n(123456)
		//payerString := strconv.Itoa(int(payerInt))
		payerString := hexString1
		//payeeString := strconv.Itoa(int(payeeInt))
		payeeString := hexString2
		valueBigInt := new(big.Int).SetInt64(valueInt)

		tx := core.NewTransaction(payerString, payeeString, valueBigInt, uint64(uniqueSource+i))
		txs = append(txs, tx)
	}

	return txs
}

func data2tx(data []string, nonce uint64) (*core.Transaction, bool) {
	if data[6] == "0" && data[7] == "0" && len(data[3]) > 16 && len(data[4]) > 16 && data[3] != data[4] {
		val, ok := new(big.Int).SetString(data[8], 10)
		if !ok {
			log.Panic("new int failed\n")
		}
		tx := core.NewTransaction4(data[3][2:], data[4][2:], val, nonce, time.Now())
		return tx, true
	}
	return &core.Transaction{}, false
}

func ProposerGenerateTxs3(txSize int, uniqueSource int) []*core.Transaction {
	//txs := make([]*core.Transaction, 0)

	txfile, err := os.Open("/home/huanglab/xbzWorkSpace/MPT/2000000to2999999_BlockTransaction.csv")
	//txfile, err := os.Open("C:\\Users\\1\\Desktop\\2000000to2999999_BlockTransaction.csv")
	if err != nil {
		log.Panic(err)
	}
	defer txfile.Close()
	reader := csv.NewReader(txfile)
	txlist := make([]*core.Transaction, 0) // save the txs in this epoch (round)
	nowDataNum := 0
	for {
		data, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}
		if tx, ok := data2tx(data, uint64(nowDataNum)); ok {
			txlist = append(txlist, tx)
			nowDataNum++
		}

		if nowDataNum == txSize {
			break
		}
	}

	return txlist
}
func write_csv(sid, nid int, blockHeights []int, datasize []int, blockGenerateTS []time.Time, blockArriveTS []time.Time, blockBeforeMPTTS []time.Time) {
	dirpath := fmt.Sprintf("%s/lw_nodeNum=%d_proposerNum=%d_signerNum=%d_requiredSign=%d_blockSize=%d_PreloaddataSize=%d_bandwidth=%d/", params.DataWrite_path,
		params.NodesInShard, params.ProposerNum, params.TotalSignerNum, params.RequiredSignNum, params.MaxBlockSize_global, params.TotalDataSize, params.Bandwidth)
	err := os.MkdirAll(dirpath, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}

	file, err := os.OpenFile(dirpath+fmt.Sprintf("/success.csv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// write the timestamp to the CSV file
	writer := csv.NewWriter(file)
	//colName := []string{"Txs", "Time", "Latency"}
	//colName := []string{"Latency","Txs"}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Panic(err)
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write([]string{"blockHeight", "blockGenerate", "blockBeforeMPT", "blockUpToChain", "dataSize", "txs", "latency", "queuetime", "executiontime"}); err != nil {
			log.Panic(err)
		}
		//writer.Write([]string{
		//	strconv.Itoa(0),
		//	strconv.Itoa(int(time.Now().UnixMilli())),
		//	strconv.Itoa(0),
		//})
		writer.Flush()
	}

	targetPath := dirpath + "/Shard" + strconv.Itoa(sid) + "_Node" + strconv.Itoa(nid) + ".csv"
	f, err := os.Open(targetPath)
	if err != nil {
		file, er := os.Create(targetPath)
		if er != nil {
			panic(er)
		}
		defer file.Close()

		w := csv.NewWriter(file)
		title := []string{"blockHeight", "dataSize", "blockGenerate", "blockArrived", "blockBeforeMPT"}
		w.Write(title)
		w.Flush()

		for i, val := range blockHeights {
			w.Write([]string{
				strconv.Itoa(val),
				strconv.FormatInt(int64(datasize[i]), 10),
				strconv.FormatInt(blockGenerateTS[i].UnixMilli(), 10),
				strconv.FormatInt(blockArriveTS[i].UnixMilli(), 10),
				strconv.FormatInt(blockBeforeMPTTS[i].UnixMilli(), 10),
			})
			w.Flush()
		}
		f.Close()
	}
}
