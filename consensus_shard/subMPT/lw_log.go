package subMPT

import (
	"blockEmulator/params"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type LWLog struct {
	Llog *log.Logger
}

func NewLwLog(sid, nid int, nodetype string) *LWLog {
	writer1 := os.Stdout
	pfx := fmt.Sprintf("LW-S%dN%d: ", sid, nid)
	dirpath := params.LogWrite_path + "/S" + strconv.Itoa(sid) + nodetype
	err := os.MkdirAll(dirpath, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
	writer2, err := os.OpenFile(dirpath+"/N"+strconv.Itoa(nid)+".log", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Panic(err)
	}
	pl := log.New(io.MultiWriter(writer1, writer2), pfx, log.Lshortfile|log.Ldate|log.Ltime|log.Lmicroseconds)
	fmt.Println()

	return &LWLog{
		Llog: pl,
	}
}
