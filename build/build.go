package build

import (
	//lightweight "blockEmulator/consensus_shard/light-weight"
	"blockEmulator/consensus_shard/pbft_all"
	"blockEmulator/params"
	"blockEmulator/supervisor"
	"time"
)

//func generateLogicNetTopo(nnm, snm uint64) {
//	// params.IPmap_nodeTable[0] = make(map[uint64]string)
//	// params.IPmap_nodeTable[0][0] = "127.0.0.1:22158"
//	// params.IPmap_nodeTable[0][1] = "127.0.0.1:22157"
//	// params.IPmap_nodeTable[0][2] = "127.0.0.1:22156"
//	// params.IPmap_nodeTable[0][3] = "127.0.0.1:22155"
//	// params.IPmap_nodeTable[0][4] = "127.0.0.1:22154"
//	// params.IPmap_nodeTable[0][5] = "127.0.0.1:22153"
//
//	// Read the contents of ip.json file
//	file, err := os.ReadFile("ip.json")
//	if err != nil {
//		// handle error
//		fmt.Println(err)
//		return
//	}
//
//	// Create a map to store the IP addresses
//	var ipMap map[uint64]map[uint64]string
//
//	// Unmarshal the JSON data into the map
//	err = json.Unmarshal(file, &ipMap)
//	if err != nil {
//		// handle error
//		fmt.Println(err)
//		return
//	}
//
//	params.IPmap_nodeTable = ipMap
//
//	params.IPmap_nodeTable[params.DeciderShard] = make(map[uint64]string)
//	params.IPmap_nodeTable[params.DeciderShard][0] = params.SupervisorAddr
//	params.NodesInShard = int(nnm)
//	params.ShardNum = int(snm)
//}

func initConfig(nid, nnm, sid, snm uint64) *params.ChainConfig {
	//generateLogicNetTopo(nnm, snm)
	pcc := &params.ChainConfig{
		ChainID:        sid,
		NodeID:         nid,
		ShardID:        sid,
		Nodes_perShard: uint64(params.NodesInShard),
		ShardNums:      snm,
		BlockSize:      uint64(params.MaxBlockSize_global),
		BlockInterval:  uint64(params.Block_Interval),
		InjectSpeed:    uint64(params.InjectSpeed),
	}
	return pcc
}

func BuildSupervisor(nnm, snm, mod uint64) {
	var measureMod []string
	if mod == 0 || mod == 2 {
		measureMod = params.MeasureBrokerMod
	} else {
		measureMod = params.MeasureRelayMod
	}

	lsn := new(supervisor.Supervisor)
	lsn.NewSupervisor(params.SupervisorAddr, initConfig(123, nnm, 123, snm), params.CommitteeMethod[mod], measureMod...)
	time.Sleep(10000 * time.Millisecond)
	go lsn.SupervisorTxHandling()
	lsn.TcpListen()
}

func BuildNewPbftNode(nid, nnm, sid, snm, mod uint64) {
	worker := pbft_all.NewPbftNode(sid, nid, initConfig(nid, nnm, sid, snm), params.CommitteeMethod[mod])
	if nid == 0 {
		go worker.Propose()
		worker.TcpListen()
	} else {
		worker.TcpListen()
	}
}

//func BuildLWNode(nid, nnm, sid, snm, mod uint64) {
//	if nid >= nnm {
//		return
//	}
//
//	lwNode := lightweight.NewLightWeightNode(uint(sid), uint(nid), initConfig(nid, nnm, sid, snm), params.CommitteeMethod[mod])
//	go func() {
//		time.Sleep(5 * time.Second)
//		lwNode.RunLWNode()
//	}()
//	lwNode.TcpListen()
//}
