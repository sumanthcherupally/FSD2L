package main

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/client"
	"GO-DAG/node"
	"GO-DAG/p2p"
	"GO-DAG/serialize"
	"GO-DAG/storage"
	"math/rand"
	"sync"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	var PrivateKey Crypto.PrivateKey
	if Crypto.CheckForKeys() {
		PrivateKey = Crypto.LoadKeys()
	} else {
		PrivateKey = Crypto.GenerateKeys()
	}
	// database.OpenDB()
	var ID p2p.PeerID
	ID.PublicKey = Crypto.SerializePublicKey(&PrivateKey.PublicKey)
	var dag dt.DAG
	var netGraph dt.PrefixGraph
	netGraph.Graph = make(map[string]dt.ASNode)
	v := constructGenisis()
	genisisHash := Crypto.Hash(serialize.Encode32(v.Tx))
	dag.Graph = make(map[string]dt.Vertex)
	dag.Graph[Crypto.EncodeToHex(genisisHash[:])] = v
	var ch chan p2p.Msg
	storageCh := make(chan dt.ForwardTx, 20)
	dag.StorageCh = storageCh
	ch = node.New(&ID, &dag, &netGraph, PrivateKey)
	// initializing the storage layer
	var st storage.Server
	st.DAG = &dag
	st.NET = &netGraph
	st.ForwardingCh = ch
	st.ServerCh = storageCh
	go st.Run()
	var wg sync.WaitGroup
	wg.Add(1)
	var cli client.Client
	cli.PrivateKey = PrivateKey
	cli.Send = ch
	cli.DAG = &dag
	cli.PGraph = &netGraph
	cli.RunAPI()
	wg.Wait()
}

func constructGenisis() dt.Vertex {
	var tx dt.Transaction
	var v dt.Vertex
	v.Tx = tx
	v.Signature = make([]byte, 72)
	return v
}
