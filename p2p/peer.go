package p2p

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

const (
	hsMsg   = 0x00
	pingMsg = 0x01
	pongMsg = 0x02
	discMsg = 0x03
)

const (
	baseprotoLength = uint32(16)
	pingMsgInterval = 10 * time.Second
)

// PeerID is a structure to identify each unique peer in the P2P network
// using IPv4 Addr and PublicKey
type PeerID struct {
	IP        []byte
	PublicKey []byte
	ShardID   uint32
}

// Equals compares PeerIDs
func (p1 *PeerID) Equals(p2 PeerID) bool {
	if (bytes.Compare(p1.IP, p2.IP) == 0) && (bytes.Compare(p1.PublicKey, p2.PublicKey) == 0) {
		return true
	}
	return false
}

// Peer represents a connected remote node
type Peer struct {
	mux    *sync.Mutex
	ID     PeerID
	rw     net.Conn
	in     chan Msg // recieves the read msgs
	ec     chan error
	closed chan struct{}
}

// GetPeerConn returns the peer socket
func (p *Peer) GetPeerConn() net.Conn {
	return p.rw
}

func newPeer(c net.Conn, pID PeerID) *Peer {
	peer := Peer{
		ID:     pID,
		rw:     c,
		in:     make(chan Msg, 5), // buffered channel to send the msgs
		ec:     make(chan error),
		closed: make(chan struct{}),
		mux:    new(sync.Mutex),
	}
	return &peer
}

// Send ...
func (p *Peer) Send(msg Msg) error {
	err := SendMsg(p.rw, msg)
	return err
}

func (p *Peer) readLoop(readErr chan error) {
	for {
		msg, err := ReadMsg(p.rw)
		msg.Sender = p.ID
		if err != nil {
			select {
			case readErr <- err:
			case <-p.closed:
			}
			return
		}
		err = p.handleMsg(msg)
		if err != nil {
			select {
			case p.ec <- err:
			case <-p.closed:
			}
			return
		}
	}
}

// GetMsg returns the msg from the peer
func (p *Peer) GetMsg() (Msg, error) {
	var msg Msg
	select {
	case msg = <-p.in:
	case <-p.closed:
		return msg, errors.New("peer suspended")
	}
	return msg, nil
}

// pingLoop is used to check the status of connection
func (p *Peer) pingLoop(e chan error) {
	for {
		time.Sleep(pingMsgInterval)
		var ping Msg
		ping.ID = pingMsg
		if err := SendMsg(p.rw, ping); err != nil {
			select {
			case e <- err:
			case <-p.closed:
			}
			break
		}
	}
}

func (p *Peer) handleMsg(msg Msg) error {
	switch {
	// case msg.ID == pingMsg:
	// 	var pong Msg
	// 	pong.ID = pongMsg
	// 	SendMsg(p.rw, pong)
	case msg.ID == discMsg:
		// close the connection

		p.rw.Close()
		return errors.New("peer requested to disconnect")
	case msg.ID > 31:
		select {
		case p.in <- msg:
		case <-p.closed:
			return errors.New("peer closed")
		}
	}
	return nil
}

func (p *Peer) run() {
	// setup the peer and run readLoop and pingLoop
	// wait for errors and terminate the both go routines
	var writeErr chan error
	var readErr chan error
	writeErr = make(chan error)
	readErr = make(chan error)

	var wg sync.WaitGroup
	wg.Add(2)

	go p.pingLoop(writeErr)
	go p.readLoop(readErr)

	select {
	case err := <-readErr:
		log.Println(err)
	case err := <-writeErr:
		log.Println(err)
	case err := <-p.ec:
		log.Println(err)
	}
	close(p.closed)
	wg.Wait()
}
