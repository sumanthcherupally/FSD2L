package storage

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	db "GO-DAG/database"
	"GO-DAG/p2p"
	"GO-DAG/serialize"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	// "net"
)

var orphanedTransactions = make(map[string][]dt.Vertex)
var mux sync.Mutex

var (
	f, _    = os.Create("temp/logFile.txt")
	f2, _   = os.Create("temp/logFile2.txt")
	logLock sync.Mutex
)

// Server ...
type Server struct {
	ForwardingCh chan p2p.Msg
	ServerCh     chan dt.ForwardTx
	DAG          *dt.DAG
	NET          *dt.PrefixGraph
}

//Hash returns the SHA256 hash value
func Hash(b []byte) []byte {
	h := sha256.Sum256(b)
	return []byte(h[:])
}

func getRIRKey() []byte {
	pubKey := "04466332a7b074b174619228e35283059b5c61a0e0d08291e7f89f09b005e3726f2cebed92546095d2aded43047c47d5a7913f8af42c65f6407b0938389afe53c1"
	p, err := hex.DecodeString(pubKey)
	if err != nil {
		log.Fatal(err)
	}
	return p
}

//AddTransaction checks if transaction if already present in the dag, if not adds to dag and database and returns true else returns false
func AddTransaction(dag *dt.DAG, netGraph *dt.PrefixGraph, tx dt.Transaction, signature []byte) int {
	// change this function for the storage node
	fmt.Println("----------------------------------------------------")
	var node dt.Vertex
	var duplicationCheck int
	duplicationCheck = 0
	s := serialize.Encode32(tx)
	Txid := Hash(s)
	h := serialize.EncodeToHex(Txid[:])
	dag.Mux.Lock()
	if _, ok := dag.Graph[h]; !ok { //Duplication check
		node.Tx = tx
		node.Signature = signature
		var tip [32]byte
		if tx.LeftTip == tip && tx.RightTip == tip {
			dag.Genisis = h
			dag.Graph[h] = node
			duplicationCheck = 1
			log.Println(tx)
			panic("Found anaother genesis !!!!!!!!")
			// db.AddToDb(Txid, s)
		} else {
			left := serialize.EncodeToHex(tx.LeftTip[:])
			right := serialize.EncodeToHex(tx.RightTip[:])
			l, okL := dag.Graph[left]
			r, okR := dag.Graph[right]
			if !okL || !okR {
				if !okL {
					duplicateOrphanTx := false
					// log.Println("left orphan transaction")
					log.Println(left)
					mux.Lock()
					for _, orphantx := range orphanedTransactions[left] {
						if bytes.Compare(orphantx.Signature, node.Signature) == 0 {
							duplicateOrphanTx = true
							break
						}
					}
					if !duplicateOrphanTx {
						orphanedTransactions[left] = append(orphanedTransactions[left], node)
					}
					mux.Unlock()
					duplicationCheck = 2
				}
				if !okR {
					duplicateOrphanTx := false
					// log.Println("right orphan transaction")
					log.Println(right)
					mux.Lock()
					for _, orphantx := range orphanedTransactions[right] {
						if bytes.Compare(orphantx.Signature, node.Signature) == 0 {
							duplicateOrphanTx = true
							break
						}
					}
					if !duplicateOrphanTx {
						orphanedTransactions[right] = append(orphanedTransactions[right], node)
					}
					mux.Unlock()
					duplicationCheck = 2
				}
			} else {
				var validity bool = false
				// var bgpM dt.BGPMssgHolder
				var txType uint32
				txType = tx.TxType
				// bgpM = tx.Msg
				if txType == 1 { //Allocate
					var bgpM dt.AllocateMsg
					bgpM, rirSign := serialize.DecodeAllocateMsg(tx.Msg)
					sAllocate, _ := json.Marshal(bgpM)
					shash := Crypto.Hash(sAllocate)
					if !Crypto.Verify(rirSign, Crypto.DeserializePublicKey(getRIRKey()), shash[:]) {
						fmt.Println("RIR signature invalid for the allocate msg")
						return 3
					}
					fmt.Println("Verified RIR signature")
					fmt.Println("Allocated prefix-", bgpM.Prefix, "to AS-", bgpM.DestinationAS)
					prefix := bgpM.Prefix
					var asNode dt.ASNode
					asNode.ASno = bgpM.DestinationAS
					asNode.Connections = make(map[int32][]int32)
					netGraph.Mux.Lock()
					netGraph.Graph[prefix] = asNode
					netGraph.Length++
					netGraph.Mux.Unlock()
					validity = true
				} else if txType == 2 { //Revoke
					var bgpM dt.RevokeMsg
					bgpM, rirSign := serialize.DecodeRevokeMsg(tx.Msg)
					fmt.Println("Verified RIR signature")
					fmt.Println("Revoked prefix-", bgpM.Prefix, "from AS-", bgpM.DestinationAS)
					// check the RIR signature validity with RIR public key which is stored in a file
					// validate the signature
					sRevoke, _ := json.Marshal(bgpM)
					shash := Crypto.Hash(sRevoke)
					if !Crypto.Verify(rirSign, Crypto.DeserializePublicKey(getRIRKey()), shash[:]) {
						fmt.Println("RIR signature invalid for the revoke msg")
						return 3
					}
					prefix := bgpM.Prefix
					netGraph.Mux.Lock()
					delete(netGraph.Graph, prefix)
					netGraph.Mux.Unlock()
					validity = true
				} else if txType == 3 { //Update
					// Not recording start dates and end dates currently in the graph
					validity = true
				} else if txType == 4 { //Path Announce
					var bgpM dt.PathAnnounceMsg
					bgpM = serialize.DecodePathAnnounceMsg(tx.Msg)
					prefix := bgpM.Prefix
					netGraph.Mux.Lock()
					var found int
					if bgpM.Path[0] == netGraph.Graph[prefix].ASno {
						for i, prev := range bgpM.Path {
							found = 0
							if i == 0 {
								found = 1
							} else {
								for _, next := range netGraph.Graph[prefix].Connections[bgpM.Path[i-1]] {
									if prev == next {
										found = 1
										break
									}
								}
								if found == 0 {
									break
								}
							}
						}
						if found == 1 {
							l := len(bgpM.Path) - 1
							netGraph.Graph[prefix].Connections[bgpM.Path[l]] = append(netGraph.Graph[prefix].Connections[bgpM.Path[l]], bgpM.DestinationAS)
							// netGraph.Graph[prefix].Connections[bgpM.DestinationAS] = make([]int32)
							validity = true
							fmt.Println("Path announcement reveived for", prefix)
							fmt.Println("Path announcement validated")
						} else {
							fmt.Println("Path announcement reveived for", prefix)
							fmt.Println("INVALID Path announcement - Path dosent exist")
						}
					}

					netGraph.Mux.Unlock()
				} else if txType == 5 { //Path withdraw
					var bgpM dt.PathWithdrawMsg
					bgpM = serialize.DecodePathWithdrawMsg(tx.Msg)
					prefix := bgpM.Prefix
					netGraph.Mux.Lock()
					if conns, ok := netGraph.Graph[prefix].Connections[bgpM.SourceAS]; ok {
						for i, p := range conns {
							if p == bgpM.DestinationAS {
								conns[i] = conns[len(conns)-1]
								conns = conns[:len(conns)-1]
								netGraph.Graph[prefix].Connections[bgpM.SourceAS] = conns
								validity = true
								break
							}
						}
					}
					netGraph.Mux.Lock()
				}
				// validity := true
				if validity {
					dag.Graph[h] = node
					dag.Length++
					if left == right {
						l.Neighbours = append(l.Neighbours, h)
						dag.Graph[left] = l
					} else {
						l.Neighbours = append(l.Neighbours, h)
						dag.Graph[left] = l
						r.Neighbours = append(r.Neighbours, h)
						dag.Graph[right] = r
					}
					duplicationCheck = 1
					fmt.Println("Valid Transaction")
				} else {
					fmt.Println("Invalid Transaction")
					duplicationCheck = 3
				}
				// db.AddToDb(Txid, s)
			}
		}
	}
	dag.Mux.Unlock()
	if duplicationCheck == 1 {
		checkorphanedTransactions(h, dag, netGraph)
	}
	fmt.Println("----------------------------------------------------")
	return duplicationCheck
}

//GetTransactiondb Wrapper function for GetTransaction in db module
func GetTransactiondb(Txid []byte) (dt.Transaction, []byte) {
	stream := db.GetValue(Txid)
	var retval dt.Transaction
	var sig []byte
	retval, sig = serialize.Decode32(stream, uint32(len(stream)))
	return retval, sig
}

//CheckifPresentDb Wrapper function for CheckKey in db module
func CheckifPresentDb(Txid []byte) bool {
	return db.CheckKey(Txid)
}

//GetAllHashes Wrapper function for GetAllKeys in db module
func GetAllHashes() [][]byte {
	return db.GetAllKeys()
}

//checkorphanedTransactions Checks if any other transaction already arrived has any relation with this transaction, Used in the AddTransaction function
func checkorphanedTransactions(h string, dag *dt.DAG, prefixGraph *dt.PrefixGraph) {

	mux.Lock()
	list, ok := orphanedTransactions[h]
	mux.Unlock()

	if ok {
		for _, node := range list {
			if AddTransaction(dag, prefixGraph, node.Tx, node.Signature) == 1 {
				// log.Println("resolved transaction")
			}
		}
	}
	// log.Println(len(orphanedTransactions))
	mux.Lock()
	delete(orphanedTransactions, h)
	mux.Unlock()
	return
}

// Run ...
func (srv *Server) Run() {
	for {
		node := <-srv.ServerCh
		if node.Forward {
			bt := time.Now().UnixNano()
			dup := AddTransaction(srv.DAG, srv.NET, node.Tx, node.Signature)
			if dup == 1 {
				at := time.Now().UnixNano()
				f2.WriteString(fmt.Sprintf("%d\n", at-bt))
				var msg p2p.Msg
				logLock.Lock()
				f.WriteString(fmt.Sprintf("%s %d %d\n", node.Peer.RemoteAddr().String(), time.Now().Minute(), time.Now().Second()))
				logLock.Unlock()
				msg.ID = 32
				msg.Payload = append(serialize.Encode32(node.Tx), node.Signature...)
				msg.LenPayload = uint32(len(msg.Payload))
				srv.ForwardingCh <- msg
			} else if dup == 2 {
				var msg p2p.Msg
				msg.ID = 34
				left := serialize.EncodeToHex(node.Tx.LeftTip[:])
				right := serialize.EncodeToHex(node.Tx.RightTip[:])
				srv.DAG.Mux.Lock()
				if _, t1 := srv.DAG.Graph[left]; !t1 {
					msg.Payload = node.Tx.LeftTip[:]
					msg.LenPayload = uint32(len(msg.Payload))
					p2p.SendMsg(node.Peer, msg)
				}
				if _, t2 := srv.DAG.Graph[right]; !t2 {
					msg.Payload = node.Tx.RightTip[:]
					msg.LenPayload = uint32(len(msg.Payload))
					p2p.SendMsg(node.Peer, msg)
				}
				srv.DAG.Mux.Unlock()
			} else if dup == 3 {
				fmt.Println("Transaction Invalidated due to history")
			}
		} else {
			dup := AddTransaction(srv.DAG, srv.NET, node.Tx, node.Signature)
			if dup == 2 {
				var msg p2p.Msg
				msg.ID = 34
				left := serialize.EncodeToHex(node.Tx.LeftTip[:])
				right := serialize.EncodeToHex(node.Tx.RightTip[:])
				srv.DAG.Mux.Lock()
				if _, t1 := srv.DAG.Graph[left]; !t1 {
					msg.Payload = node.Tx.LeftTip[:]
					msg.LenPayload = uint32(len(msg.Payload))
					p2p.SendMsg(node.Peer, msg)
				}
				if _, t2 := srv.DAG.Graph[right]; !t2 {
					msg.Payload = node.Tx.RightTip[:]
					msg.LenPayload = uint32(len(msg.Payload))
					p2p.SendMsg(node.Peer, msg)
				}
				srv.DAG.Mux.Unlock()
			}
		}
	}
}
