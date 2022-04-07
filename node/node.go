package node

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/p2p"
	"GO-DAG/serialize"
	"log"
)

// var (
// 	f, _    = os.Create("temp/logFile.txt")
// 	logLock sync.Mutex
// )

// New ...
func New(hostID *p2p.PeerID, dag *dt.DAG, netGraph *dt.PrefixGraph, PrivKey Crypto.PrivateKey) chan p2p.Msg {

	var srv p2p.Server
	srv.HostID = *hostID
	srv.BroadcastMsg = make(chan p2p.Msg)
	srv.NewPeer = make(chan p2p.Peer)
	srv.RemovePeer = make(chan p2p.Peer)
	srv.ShardTransactions = make(chan dt.ShardTransactionCh)
	srv.ShardingSignal = make(chan dt.ShardSignal)
	srv.PrivateKey = PrivKey
	go srv.Run()
	go func() {
		for {
			p := <-srv.NewPeer
			go handle(&p, srv.BroadcastMsg, dag, srv.ShardingSignal, srv.ShardTransactions, srv.RemovePeer)
		}
	}()
	return srv.BroadcastMsg
}

func handleMsg(msg p2p.Msg, send chan p2p.Msg, dag *dt.DAG, p *p2p.Peer, ShardSignalch chan dt.ShardSignal, Shardtxch chan dt.ShardTransactionCh) {
	// check for transactions or request for transactions
	if msg.ID == 32 {
		// transaction
		tx, sign := serialize.Decode32(msg.Payload, msg.LenPayload)
		// log.Println(hex.EncodeToString(tx.LeftTip[:]))
		if validTransaction(msg.Payload, tx.From[:]) {
			// tr := storage.AddTransaction(dag, tx, sign)
			// if tr == 1 {
			// 	send <- msg
			// 	// logLock.Lock()
			// 	// f.WriteString(fmt.Sprintf("%d %d %d\n", p.ID.IP, time.Now().Minute(), time.Now().Second()))
			// 	// logLock.Unlock()
			// 	// fmt.Println(time.Now())
			// } else if tr == 2 {
			// 	var msg p2p.Msg
			// 	msg.ID = 34
			// 	left := serialize.EncodeToHex(tx.LeftTip[:])
			// 	right := serialize.EncodeToHex(tx.RightTip[:])
			// 	dag.Mux.Lock()
			// 	if _, t1 := dag.Graph[left]; !t1 {
			// 		msg.Payload = tx.LeftTip[:]
			// 		msg.LenPayload = uint32(len(msg.Payload))
			// 		p.Send(msg)
			// 	}
			// 	if _, t2 := dag.Graph[right]; !t2 {
			// 		msg.Payload = tx.RightTip[:]
			// 		msg.LenPayload = uint32(len(msg.Payload))
			// 		p.Send(msg)
			// 	}
			// 	dag.Mux.Unlock()
			// }
			var sTx dt.ForwardTx
			sTx.Tx = tx
			sTx.Signature = sign
			sTx.Peer = p.GetPeerConn()
			sTx.Forward = true
			dag.StorageCh <- sTx
		}
	} else if msg.ID == 34 {
		// request for transaction
		hash := msg.Payload
		dag.Mux.Lock()
		v, ok := dag.Graph[Crypto.EncodeToHex(hash)]
		dag.Mux.Unlock()
		if ok {
			tx := v.Tx
			sign := v.Signature
			var respMsg p2p.Msg
			respMsg.ID = 33
			respMsg.Payload = append(serialize.Encode32(tx), sign...)
			respMsg.LenPayload = uint32(len(respMsg.Payload))
			p.Send(respMsg)
		}
	} else if msg.ID == 33 {
		tx, sign := serialize.Decode32(msg.Payload, msg.LenPayload)
		if validTransaction(msg.Payload, tx.From[:]) {
			// tr := storage.AddTransaction(dag, tx, sign)
			// if tr == 2 {
			// 	var msg p2p.Msg
			// 	msg.ID = 34
			// 	left := serialize.EncodeToHex(tx.LeftTip[:])
			// 	right := serialize.EncodeToHex(tx.RightTip[:])
			// 	dag.Mux.Lock()
			// 	if _, t1 := dag.Graph[left]; !t1 {
			// 		msg.Payload = tx.LeftTip[:]
			// 		msg.LenPayload = uint32(len(msg.Payload))
			// 		p.Send(msg)
			// 	}
			// 	if _, t2 := dag.Graph[right]; !t2 {
			// 		msg.Payload = tx.RightTip[:]
			// 		msg.LenPayload = uint32(len(msg.Payload))
			// 		p.Send(msg)
			// 	}
			// 	dag.Mux.Unlock()
			// }
			var sTx dt.ForwardTx
			sTx.Tx = tx
			sTx.Signature = sign
			sTx.Peer = p.GetPeerConn()
			dag.StorageCh <- sTx

		}
	} else if msg.ID == 35 { //Shard signal
		signal, _ := serialize.Decode35(msg.Payload, msg.LenPayload)
		select {
		case ShardSignalch <- signal:
			send <- msg
		default:
		}
	} else if msg.ID == 36 { //Sharding tx from other nodes
		tx, sign := serialize.Decode36(msg.Payload, msg.LenPayload)
		// if sh.VerifyShardTransaction(tx, sign, 4) {
		// 	Shardtxch <- tx
		// }
		var sch dt.ShardTransactionCh
		sch.Tx = tx
		sch.Sign = sign
		Shardtxch <- sch
	}
}

// read the messages and handle
func handle(p *p2p.Peer, send chan p2p.Msg, dag *dt.DAG, ShardSignalch chan dt.ShardSignal, Shardtxch chan dt.ShardTransactionCh, errChan chan p2p.Peer) {
	for {
		msg, err := p.GetMsg()
		if err != nil {
			errChan <- *p
			log.Println(err)
			break
		}
		go handleMsg(msg, send, dag, p, ShardSignalch, Shardtxch)
	}
}

func validTransaction(msg []byte, PublicKey []byte) bool {

	sTx := msg[:len(msg)-72]
	sign := msg[len(msg)-72:]
	h := Crypto.Hash(sTx)
	return Crypto.Verify(sign, Crypto.DeserializePublicKey(PublicKey), h[:])
	// return true
}

func getAllKeys(dag *dt.DAG) []string {

	var hashes []string
	dag.Mux.Lock()
	for k := range dag.Graph {
		hashes = append(hashes, k)
	}
	dag.Mux.Unlock()
	return hashes
}
