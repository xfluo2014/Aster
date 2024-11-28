package networks

import (
	"blockEmulator/params"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func WriteBandwidthCSV(cnt, bandwidth int) {
	fmt.Printf("WriteCSV\n")
	dirpath := fmt.Sprintf("%s/", params.DataWrite_path)
	targetPath := dirpath + "/dynamicBandwidth_BeginWith=" + strconv.Itoa(1) + "_Node" + strconv.Itoa(NodeID) + ".csv"

	if cnt == 0 {
		err := os.MkdirAll(dirpath, os.ModePerm)
		if err != nil {
			log.Panic(err)
		}
	}

	file, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()

	// 创建CSV写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	if cnt == 0 {
		writer.Write([]string{"time", "bandwidth"})

	}

	writer.Write([]string{strconv.FormatInt(time.Now().UnixMilli(), 10), strconv.Itoa(bandwidth)})

}
