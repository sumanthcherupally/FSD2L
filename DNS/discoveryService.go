package main

import (
	"GO-DAG/Crypto"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// used to sign the messages
var (
	PrivateKey    Crypto.PrivateKey
	totalNodes    int
	maxNodes      int
	currNodeCount int
)

type peerAddr struct {
	ShardID     uint32
	networkAddr []byte
	PublicKey   []byte
}

// better add a database as the backup
type liveNodes struct {
	GWNodes gatewayNodes
	SNNodes storageNodes
}

// stores all the gateway nodes in a shard
type shardNodes struct {
	ShardID uint32
	Nodes   []peerAddr
}

type gatewayNodes struct {
	mux    sync.RWMutex
	shards []shardNodes
}

type storageNodes struct {
	mux   sync.RWMutex
	Nodes []peerAddr
}

type shardSignal struct {
	Identifier [32]byte
	From       [65]byte
}

func find(list []peerAddr, element peerAddr) bool {
	for _, node := range list {
		if bytes.Compare(element.networkAddr, node.networkAddr) == 0 {
			return true
		}
	}
	return false
}

func triggerNodes(nodes []peerAddr) {

	for _, node := range nodes {
		conn, err := net.Dial("tcp", parseAddr(node.networkAddr)+":6666")
		if err != nil {
			log.Println(err)
		}
		buf := []byte{0x05}
		conn.Write(buf)
		conn.Close()
	}
}

func (nodes *liveNodes) appendTo(peer peerAddr, GS bool) {
	if GS {
		nodes.GWNodes.mux.Lock()
		var shard shardNodes
		var i int
		for i, shard = range nodes.GWNodes.shards {
			if peer.ShardID == shard.ShardID {
				break
			}
		}
		shard.Nodes = append(shard.Nodes, peer)
		nodes.GWNodes.shards[i] = shard

		if len(shard.Nodes) >= maxNodes {
			go nodes.initiateSharding(shard)
		}

		// trigger signal to all nodes to generate transactions
		nodes.GWNodes.mux.Unlock()
	} else {
		nodes.SNNodes.mux.Lock()
		nodes.SNNodes.Nodes = append(nodes.SNNodes.Nodes, peer)
		nodes.SNNodes.mux.Unlock()
	}
}

func (nodes *liveNodes) remove(GWNodes []peerAddr, SNNodes []peerAddr) {
	// temporarily halt , write after testing the other shit
}

func constructShardSignal() []byte {
	var shardSignal []byte
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	h := Crypto.Hash(b)
	shardSignal = append(shardSignal, h[:]...)
	shardSignal = append(shardSignal, Crypto.SerializePublicKey(&PrivateKey.PublicKey)...)
	return shardSignal
}

func constructPing() []byte {
	ping := []byte{0x01}
	ping = append(ping, []byte{0x00}...)
	return ping
}

func checkPong(b []byte) bool {
	if b[0] == 0x02 {
		return true
	}
	return false
}

func parseAddr(b []byte) string {
	addr := strconv.Itoa(int(b[0])) + "." + strconv.Itoa(int(b[1])) + "."
	addr += strconv.Itoa(int(b[2])) + "." + strconv.Itoa(int(b[3]))
	return addr
}

func sendShardSignal(nodes shardNodes) {
	randNode := nodes.Nodes[rand.Intn(len(nodes.Nodes))]
	conn, _ := net.Dial("tcp", parseAddr(randNode.networkAddr)+":8060")
	b := constructShardSignal()
	hash := Crypto.Hash(b)
	sign := Crypto.Sign(hash[:], PrivateKey)
	b = append(b, sign...)
	l := new(bytes.Buffer)
	binary.Write(l, binary.LittleEndian, uint32(len(b)))
	b = append(l.Bytes(), b...)
	h := new(bytes.Buffer)
	var id uint32
	id = 35
	binary.Write(h, binary.LittleEndian, id)
	b = append(h.Bytes(), b...)
	conn.Write(b)
	conn.Close()
}

func (nodes *liveNodes) updateShard(ShardID uint32, IP []byte, PubKey []byte) {
	nodes.GWNodes.mux.Lock()
	var peer peerAddr
	peer.networkAddr = IP
	peer.PublicKey = PubKey
	peer.ShardID = ShardID
	for i, shard := range nodes.GWNodes.shards {
		if ShardID == shard.ShardID {
			fmt.Println("shardID updated")
			shard.Nodes = append(shard.Nodes, peer)
			nodes.GWNodes.shards[i] = shard
			break
		}
	}
	nodes.GWNodes.mux.Unlock()
}

func (nodes *liveNodes) initiateSharding(prevShard shardNodes) {
	nodes.GWNodes.mux.Lock()
	time.Sleep(2 * time.Second)
	sendShardSignal(prevShard)
	for i, shard := range nodes.GWNodes.shards {
		if prevShard.ShardID == shard.ShardID {
			// deleting the old shard
			nodes.GWNodes.shards[i] = nodes.GWNodes.shards[len(nodes.GWNodes.shards)-1]
			nodes.GWNodes.shards = nodes.GWNodes.shards[:(len(nodes.GWNodes.shards) - 1)]
			break
		}
	}
	// creating two new shards
	var nextShard1, nextShard2 shardNodes
	nextShard1.ShardID = prevShard.ShardID * (10)
	nextShard2.ShardID = prevShard.ShardID*(10) + 1
	nodes.GWNodes.shards = append(nodes.GWNodes.shards, nextShard1, nextShard2)
	fmt.Println(len(nodes.GWNodes.shards))
	nodes.GWNodes.mux.Unlock()
}

// temporary method optimize if needed
func getRandomPeers(nodes *liveNodes, peer peerAddr, gs bool) []peerAddr {
	// find an efficient way to select the peers
	var peers []peerAddr
	if gs {
		nodes.GWNodes.mux.RLock()
		nodes.SNNodes.mux.RLock()
		var shard shardNodes
		for _, shard = range nodes.GWNodes.shards {
			if shard.ShardID == peer.ShardID {
				break
			}
		}
		perm1 := rand.Perm(len(shard.Nodes))
		// perm2 := rand.Perm(len(nodes.SNNodes.Nodes))
		if len(shard.Nodes) > 1 {
			peers = append(peers, shard.Nodes[perm1[0]])
			peers = append(peers, shard.Nodes[perm1[1]])
		} else if len(shard.Nodes) == 1 {
			peers = append(peers, shard.Nodes[perm1[0]])
		}
		nodes.SNNodes.mux.RUnlock()
		nodes.GWNodes.mux.RUnlock()
	} else {
		nodes.SNNodes.mux.RLock()
		l := len(nodes.SNNodes.Nodes)
		perm := rand.Perm(l)
		if l == 1 {
			peers = append(peers, nodes.SNNodes.Nodes[perm[0]])
		} else if l > 1 {
			peers = append(peers, nodes.SNNodes.Nodes[perm[0]])
			peers = append(peers, nodes.SNNodes.Nodes[perm[1]])
		}
		nodes.SNNodes.mux.RUnlock()
	}
	return peers
}

func deserialize(b []byte) ([]byte, []byte, bool) {
	var gs bool
	gs = false
	if b[0] == 0x04 {
		gs = true
	}
	return b[1:5], b[5:], gs
}

func handleConnection(conn net.Conn, nodes *liveNodes) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	l, err := conn.Read(buffer)
	if err != nil {
		log.Println(err)
		return
	}
	if buffer[0] == 0x06 {
		IP, PubKey, ShardIDBytes := buffer[1:5], buffer[5:70], buffer[70:l]
		r := bytes.NewReader(ShardIDBytes)
		var ShardID uint32
		binary.Read(r, binary.LittleEndian, &ShardID)
		var newPeer peerAddr
		newPeer.PublicKey = PubKey
		newPeer.networkAddr = IP
		newPeer.ShardID = ShardID
		nodes.appendTo(newPeer, true)
	} else {
		IP, PubKey, GS := deserialize(buffer[:l])
		var newPeer peerAddr
		if GS {
			currNodeCount++
			nodes.GWNodes.mux.Lock()
			x := rand.Intn(len(nodes.GWNodes.shards))
			newPeer.ShardID = nodes.GWNodes.shards[x].ShardID
			nodes.GWNodes.mux.Unlock()

			var ShardingInProgress bool
			for {
				ShardingInProgress = false
				length := len(nodes.GWNodes.shards[x].Nodes)
				if length == 0 && len(nodes.GWNodes.shards) > 1 {
					ShardingInProgress = true
				}
				if !ShardingInProgress {
					for _, shard := range nodes.GWNodes.shards {
						fmt.Println(fmt.Sprintf("%d - %d", shard.ShardID, len(shard.Nodes)))
					}
					break
				}
				time.Sleep(time.Second)
			}
			fmt.Println(newPeer.ShardID)
		} else {
			newPeer.ShardID = 0
		}
		newPeer.PublicKey = PubKey
		newPeer.networkAddr = IP
		randomPeers := getRandomPeers(nodes, newPeer, GS)
		var p [][]byte
		for _, peer := range randomPeers {
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, peer.ShardID)
			p = append(p, append(append(peer.networkAddr, peer.PublicKey...), buf.Bytes()...))
		}
		b, _ := json.Marshal(p)
		conn.Write(b)
		nodes.appendTo(newPeer, GS)

		// if currNodeCount >= totalNodes {
		// 	time.Sleep(30 * time.Second)
		// 	nodes.GWNodes.mux.Lock()
		// 	for _, shard := range nodes.GWNodes.shards {
		// 		go triggerNodes(shard.Nodes)
		// 	}
		// 	nodes.GWNodes.mux.Unlock()
		// }
	}
	return
}

// add a database to store the peers
func checkDB(nodes *liveNodes) {
	//
	return
}

func main() {
	rand.Seed(time.Now().UnixNano())
	port := os.Args[1]
	// argument contains total number of nodes in the test network
	totalNodes, _ = strconv.Atoi(os.Args[2])
	maxNodes, _ = strconv.Atoi(os.Args[3])

	log.Println("Discovery service running on port", port)
	var nodes liveNodes
	var firstShard shardNodes
	firstShard.ShardID = 1
	nodes.GWNodes.shards = append(nodes.GWNodes.shards, firstShard)
	listener, err := net.Listen("tcp", ":"+port)
	if Crypto.CheckForKeys() {
		PrivateKey = Crypto.LoadKeys()
	} else {
		PrivateKey = Crypto.GenerateKeys()
	}

	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("New connection", conn.RemoteAddr().String())
		handleConnection(conn, &nodes)

		if currNodeCount == totalNodes {
			// write all the nodes to a file
			f, err := os.Create("bc_nodes.txt")
			if err != nil {
				log.Fatal(err)
			}

			for _, shard := range nodes.GWNodes.shards {
				for _, node := range shard.Nodes {
					nodeAddr := parseAddr(node.networkAddr)
					f.WriteString(nodeAddr + "\n")
				}
			}
			break
		}
	}
}
