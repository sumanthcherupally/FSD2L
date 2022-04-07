package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	discvServer string
)

// FindPeers fetches some peers by querying the discovery service
// The Bootstrap disocvery nodes are provided in a file
func FindPeers(host *PeerID) []PeerID {
	var peers []PeerID

	// reading from the file containing discv nodes
	f, err := os.Open("bootstrapNodes.txt")
	if err != nil {
		log.Fatal("Problem opening the file containing bootstrap nodes")
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("Incosistent file containing bootstrap nodes")
	}

	// bootstrapNodes is the list of some of the discv nodes
	bootstrapNodes := strings.Split(string(b), "\n")

	// iteratively trying to query discovery nodes until a successful response
	for _, addr := range bootstrapNodes {
		peers, err = queryDiscoveryService(addr, host)
		if err != nil {
			log.Println("Failed to query discv node", addr)
		} else {
			discvServer = addr
			break
		}
	}
	return peers
}

func queryDiscoveryService(servAddr string, localID *PeerID) ([]PeerID, error) {

	// querying the discovery service
	conn, err := net.Dial("tcp", servAddr)
	log.Println("Dialed the discovery server")
	if err != nil {
		return nil, err
	}
	ip := conn.LocalAddr().String()
	ip = ip[:strings.IndexByte(ip, ':')]
	localID.IP = serializeIPAddr(ip)
	query := []byte{0x04}
	query = append(query, localID.IP...)
	query = append(query, localID.PublicKey...)
	_, err = conn.Write(query)
	if err != nil {
		log.Println(err)
	}
	buf := make([]byte, 1024)

	// response from discovery service
	l, _ := conn.Read(buf)
	var peers [][]byte
	err = json.Unmarshal(buf[:l], &peers)
	if err != nil {
		return nil, err
	}
	var p []PeerID
	for _, peer := range peers {
		var s PeerID
		s.IP = peer[:4]
		s.PublicKey = peer[4:69]
		r := bytes.NewReader(peer[69:])
		binary.Read(r, binary.LittleEndian, &s.ShardID)
		p = append(p, s)
	}
	if len(p) > 1 {
		log.Println("shard assigned", p[0].ShardID)
		localID.ShardID = p[0].ShardID
	} else {
		localID.ShardID = 1
	}
	log.Println("Number of peers", len(p))
	return p, nil
}

func constructUpdateShardID(p PeerID) []byte {
	b := []byte{0x06}
	b = append(b, p.IP...)
	b = append(b, p.PublicKey...)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, p.ShardID)
	b = append(b, buf.Bytes()...)
	return b
}

func updateShardID(host PeerID) {
	conn, _ := net.Dial("tcp", discvServer)
	conn.Write(constructUpdateShardID(host))
	conn.Close()
}

func serializeIPAddr(IP string) []byte {
	ipFields := strings.Split(IP, ".")
	buf := new(bytes.Buffer)
	var i int
	for _, fields := range ipFields {
		i, _ = strconv.Atoi(fields)
		binary.Write(buf, binary.LittleEndian, uint8(i))
	}
	return buf.Bytes()
}
