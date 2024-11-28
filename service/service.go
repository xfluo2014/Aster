package service

import (
	"blockEmulator/client"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var Clt = client.NewClient("127.0.0.1:17777")

type InputTransaction struct {
	From  string
	To    string
	Value float64
}

func SendTx2Network(c *gin.Context) {
	var it InputTransaction
	c.Bind(&it)

	// Clt.SendTx2Worker(it.From, it.To, it.Value)
	Clt.SendTx2Worker("23156132", "13544131", 10)

	c.JSON(http.StatusOK, 1)
}

type InputAccountAddr struct {
	Addr string
}

// It seems that this func need an address only.
func RequestAccountState(c *gin.Context) {
	var iaa InputAccountAddr
	c.Bind(&iaa)
	// replace "23156132" to iaa.Addr below.

	// delete old account state
	delete(Clt.AccountStateRequestMap, "23156132")
	Clt.SendAccountStateRequest2Worker("23156132")
	ret := "Try for 30s... Cannot access blockEmulator Network."
	beginRequestTime := time.Now()
	for time.Since(beginRequestTime) < time.Second*30 {
		if _, ok := Clt.AccountStateRequestMap["23156132"]; ok {
			ret = "Account: 23156132\n" + "Balance: " + Clt.AccountStateRequestMap["23156132"].Balance.String()
			break
		}
	}
	c.JSON(http.StatusOK, ret)
}
