package p2p

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/serialize"
	sh "GO-DAG/sharding"
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	haltServer sync.Mutex
)

type handshakeMsg struct {
	ID PeerID
}

func (msg *handshakeMsg) encode() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, msg.ID.ShardID)
	ret := append(msg.ID.IP, msg.ID.PublicKey...)
	return append(ret, b.Bytes()...)
}

func validateHandshakeMsg(reply []byte) (PeerID, error) {
	// figure out some validation criteria
	var p PeerID
	p.IP = reply[:4]
	p.PublicKey = reply[4:69]
	buf := bytes.NewReader(reply[69:])
	var sid uint32
	binary.Read(buf, binary.LittleEndian, &sid)
	p.ShardID = sid
	return p, nil
}

// Server ...
type Server struct {
	peers             []Peer
	maxPeers          uint32
	HostID            PeerID
	ec                chan error
	mux               sync.Mutex
	NewPeer           chan Peer
	BroadcastMsg      chan Msg
	RemovePeer        chan Peer
	ShardingSignal    chan dt.ShardSignal
	ShardTransactions chan dt.ShardTransactionCh
	PrivateKey        *ecdsa.PrivateKey
	// ...
}

// GetRandomPeer ...
func (srv *Server) GetRandomPeer() Peer {
	var p Peer
	for {
		time.Sleep(time.Second)
		srv.mux.Lock()
		if len(srv.peers) > 0 {
			p = srv.peers[0]
			srv.mux.Unlock()
			break
		}
		srv.mux.Unlock()
	}
	return p
}

func (srv *Server) findPeer(peer PeerID) bool {
	for _, p := range srv.peers {
		if p.ID.Equals(peer) && p.ID.ShardID == peer.ShardID {
			return true
		}
	}
	return false
}

// setupConn validates a handshake with the other peer
// Adds the new peer to the list of known peers
func (srv *Server) setupConn(conn net.Conn) error {

	msg, err := ReadMsg(conn)
	if err != nil {
		return err
	}

	// validate hanshake message
	pid, err := validateHandshakeMsg(msg.Payload)

	// reply with a proper hanshake
	if msg.ID == 0x00 {
		var hMsg handshakeMsg
		hMsg.ID = srv.HostID
		buf := hMsg.encode()
		var msg Msg
		msg.ID = hsMsg
		msg.LenPayload = uint32(len(buf))
		msg.Payload = buf
		if err := SendMsg(conn, msg); err != nil {
			return err
		}
	} else {
		return errors.New("bad handshake")
	}

	p := newPeer(conn, pid)
	srv.AddPeer(p)
	srv.NewPeer <- *p
	log.Println("New connection from", parseAddr(p.ID.IP))
	return nil
}

func (srv *Server) performHandshake(c net.Conn, p PeerID) error {
	// define handshake msg

	hmsg := handshakeMsg{srv.HostID}
	buf := hmsg.encode()
	var msg Msg
	msg.ID = hsMsg
	msg.LenPayload = uint32(len(buf))
	msg.Payload = buf

	// sending the handshake msg
	if err := SendMsg(c, msg); err != nil {
		return err
	}
	reply, err := ReadMsg(c)
	if err != nil {
		return err
	}
	// validate the reply figure out
	pid, err := validateHandshakeMsg(reply.Payload)
	if !pid.Equals(p) {
		return errors.New("Invalid Handshake Msg")
	}
	return nil
}

// AddPeer ...
func (srv *Server) AddPeer(p *Peer) {
	srv.mux.Lock()
	srv.peers = append(srv.peers, *p)
	srv.mux.Unlock()
	go p.run()
	return
}

// RemovePeer ...
func (srv *Server) removePeer(peer Peer) {
	// terminate the corresponding go routine and cleanup
	for i, p := range srv.peers {
		if p.ID.Equals(peer.ID) {
			srv.peers[i] = srv.peers[len(srv.peers)-1]
			srv.peers = srv.peers[:len(srv.peers)-1]
			break
		}
	}
	return
}

func (srv *Server) listenForConns() {
	listener, _ := net.Listen("tcp", ":8060")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		ip := conn.RemoteAddr().String()
		ip = ip[:strings.IndexByte(ip, ':')]
		// sharding signal from discovery service
		if ip == discvServer[:strings.IndexByte(discvServer, ':')] {
			log.Println("sharding signal recieved")
			msg, _ := ReadMsg(conn)
			shSignal, _ := serialize.Decode35(msg.Payload, msg.LenPayload)
			srv.ShardingSignal <- shSignal
			conn.Close()
			srv.BroadcastMsg <- msg
		} else {
			go srv.setupConn(conn)
		}
	}
}

func parseAddr(b []byte) string {
	addr := strconv.Itoa(int(b[0])) + "." + strconv.Itoa(int(b[1])) + "."
	addr += strconv.Itoa(int(b[2])) + "." + strconv.Itoa(int(b[3])) + ":8060"
	return addr
}

func (srv *Server) initiateConnection(pID PeerID) (net.Conn, error) {
	conn, err := net.Dial("tcp", parseAddr(pID.IP))
	if err != nil {
		return conn, err
	}

	err = srv.performHandshake(conn, pID)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (srv *Server) discOldPeers() {
	srv.mux.Lock()
	for _, p := range srv.peers {
		if p.ID.ShardID != srv.HostID.ShardID && p.ID.ShardID != 0 {
			var msg Msg
			msg.ID = 3
			msg.LenPayload = 0
			err := p.Send(msg)
			log.Println(err)
			srv.removePeer(p)
		}
	}
	srv.mux.Unlock()
}

// Run starts the server
func (srv *Server) Run() {

	srv.ec = make(chan error)
	// start the server
	go srv.listenForConns()
	time.Sleep(time.Second)

	// start the discovery and request peer
	var pIds []PeerID
	pIds = FindPeers(&srv.HostID)
	// iteratively connect with peers
	for _, pID := range pIds {
		// handshake phase
		conn, err := srv.initiateConnection(pID)
		if err != nil {
			log.Println(err)
		} else {
			p := newPeer(conn, pID)
			srv.AddPeer(p)
			srv.NewPeer <- *p
		}
	}

	var tempPeers []PeerID
	go func() {
		for {
			signal := <-srv.ShardingSignal
			log.Println("sharding signal received")
			var sign []byte
			Shardingtx, err := sh.MakeShardingtx(srv.HostID.PublicKey, signal)
			srv.HostID.ShardID = srv.HostID.ShardID*10 + Shardingtx.ShardNo
			Shardingtx.ShardNo = srv.HostID.ShardID
			log.Println("sharding transaction created", Shardingtx.ShardNo)

			copy(Shardingtx.IP[:], srv.HostID.IP)

			if err != nil {
				log.Println(err)
				continue
			}
			var msg Msg
			msg.ID = 36
			msg.Payload = serialize.Encode36(Shardingtx)
			h := Crypto.Hash(msg.Payload)
			sign = Crypto.Sign(h[:], srv.PrivateKey)
			msg.Payload = append(msg.Payload, sign...)
			msg.LenPayload = uint32(len(msg.Payload))
			srv.BroadcastMsg <- msg
			time.Sleep(time.Second)

			var shPeers []PeerID
			l := 0

			for {
				// check the number of peers with same shardNo
				for _, p := range tempPeers {
					if p.ShardID == srv.HostID.ShardID {
						shPeers = append(shPeers, p)
						l++
					}
				}
				if l > 0 {
					break
				}
				time.Sleep(time.Second)
			}
			// disconect with the other peers
			// connect with these peers with same shardNo
			// check if the routing reaches storage node or some other mechanism
			time.Sleep(time.Second)
			i := 0

			for _, pID := range shPeers {

				srv.mux.Lock()
				d := srv.findPeer(pID)
				srv.mux.Unlock()

				if !d {
					conn, err := srv.initiateConnection(pID)
					if err != nil {
						log.Println(err)
					} else {
						p := newPeer(conn, pID)
						srv.AddPeer(p)
						srv.NewPeer <- *p
						i++
					}
				}

				if i > 2 {
					break
				}
			}
			// send the discovery server appropriate information
			time.Sleep(2 * time.Second)
			updateShardID(srv.HostID)
			srv.discOldPeers()
			log.Println("sharding complete")
			var emptySlice []PeerID
			tempPeers = emptySlice
		}
	}()

	for {
		select {
		// listen
		case msg := <-srv.BroadcastMsg:
			Send(msg, srv.peers)
		case <-srv.ec:
			log.Fatal("error")
		case p := <-srv.RemovePeer:
			srv.mux.Lock()
			srv.removePeer(p)
			srv.mux.Unlock()
		case sch := <-srv.ShardTransactions:
			tx := sch.Tx
			sign := sch.Sign
			var p PeerID
			p.IP = make([]byte, 4)
			p.PublicKey = make([]byte, 65)
			copy(p.IP, tx.IP[:])
			copy(p.PublicKey, tx.From[:])
			p.ShardID = tx.ShardNo
			dup := false

			if p.Equals(srv.HostID) {
				dup = true
			}
			for _, peer := range tempPeers {
				if peer.Equals(p) {
					dup = true
				}
			}
			if !dup {
				log.Println("sharding transaction received", tx.ShardNo)
				tempPeers = append(tempPeers, p)
				var msg Msg
				msg.ID = 36
				msg.Payload = append(serialize.Encode36(tx), sign...)
				msg.LenPayload = uint32(len(msg.Payload))
				Send(msg, srv.peers)
			}
		}
	}
}

// Send ...
func Send(msg Msg, peers []Peer) {
	i := 0
	perm := rand.Perm(len(peers))
	for _, j := range perm {
		p := peers[j]
		if !p.ID.Equals(msg.Sender) {
			SendMsg(p.rw, msg)
			i++
		}
	}
}
