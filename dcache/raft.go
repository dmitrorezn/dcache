package main

func runRaft() {
	//memStore := raft.NewInmemStore()
	//logCache, err := raft.NewLogCache(100, memStore)
	//if err != nil {
	//	log.Println("NewLogCache", err)
	//}
	//workdir, err := os.Getwd()
	//if err != nil {
	//	log.Println("Getwd", err)
	//}
	//snapshotStore, err := raft.NewFileSnapshotStoreWithLogger(workdir, 1, logger)
	//if err != nil {
	//	log.Println("NewFileSnapshotStoreWithLogger", err)
	//}
	//raftAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(localhost, cfg.Port))
	//if err != nil {
	//	log.Println("ResolveTCPAddr", err)
	//}
	//nodeID := raftAddr.String()
	//_ = nodeID
	//log.Println("nodeID", nodeID)
	//
	//tr, err := raft.NewTCPTransport(
	//	cfg.LeaderAddr,
	//	raftAddr,
	//	10,
	//	cfg.Timeout,
	//	os.Stdout,
	//)
	//if err != nil {
	//	log.Fatal("NewTCPTransport", err)
	//}
	//raftCfg := raft.DefaultConfig()
	//raftCfg.LocalID = raft.ServerID(nodeID)
	//log.Println("DefaultConfig", raftCfg)
	//
	//_raft, err := raft.NewRaft(raftCfg, store, logCache, memStore, snapshotStore, tr)
	//if err != nil {
	//	log.Fatal("NewRaft", err)
	//}
	//_ = _raft
	//replicatedStorage := storage.NewReplicator(store, raftCfg.LocalID, _raft)
	//log.Println("NewReplicator")
	//
	//peerCfg, err := raft.ReadPeersJSON(filepath.Join(workdir, "peers.json"))
	//if err != nil {
	//	log.Fatal("ReadPeersJSON", err)
	//}
	//_ = peerCfg
	//if !cfg.IsLeader {
	//	var prevIdx uint64
	//	idx, err := os.ReadFile("index.bin")
	//	if err != nil {
	//		log.Println("ReadFile", err)
	//		prevIdx = 0
	//	} else {
	//		var n int
	//		prevIdx, n = binary.Uvarint(idx)
	//		if len(idx) != n {
	//			log.Fatal("error parse prevIdx with Uvarint", err)
	//		}
	//	}
	//
	//	f := _raft.AddVoter(
	//		raft.ServerID(cfg.LeaderAddr),
	//		raft.ServerAddress(cfg.LeaderAddr),
	//		prevIdx,
	//		cfg.Timeout,
	//	)
	//	if err = f.Error(); err != nil {
	//		log.Fatal("AddVoter->Error", err)
	//	}
	//	log.Println("AddVoter Index", f.Index())
	//}
}
