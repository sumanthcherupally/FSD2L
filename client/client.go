package client

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/consensus"
	"GO-DAG/p2p"
	"GO-DAG/serialize"
	"GO-DAG/storage"
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var rirAddress = "172.18.0.1:7111"

// Client ...
type Client struct {
	PrivateKey *ecdsa.PrivateKey
	// should I keep the dag here or run a go routine for storage layer
	Send   chan p2p.Msg
	DAG    *dt.DAG
	PGraph *dt.PrefixGraph
}

// IssueTransaction ...
func (cli *Client) IssueTransaction(bgpmsg []byte, txType uint32) []byte {
	var tx dt.Transaction
	tx.TxType = txType
	tx.Timestamp = time.Now().UnixNano()
	tx.Msg = bgpmsg
	copy(tx.From[:], Crypto.SerializePublicKey(&cli.PrivateKey.PublicKey))

	copy(tx.LeftTip[:], Crypto.DecodeToBytes(consensus.GetTip(cli.DAG, 0.001)))
	copy(tx.RightTip[:], Crypto.DecodeToBytes(consensus.GetTip(cli.DAG, 0.001)))
	// pow.PoW(&tx, 3)
	// fmt.Println("After pow")
	b := serialize.Encode32(tx)
	var msg p2p.Msg
	msg.ID = 32
	h := Crypto.Hash(b)
	sign := Crypto.Sign(h[:], cli.PrivateKey)
	msg.Payload = append(b, sign...)
	msg.LenPayload = uint32(len(msg.Payload))
	cli.Send <- msg
	storage.AddTransaction(cli.DAG, cli.PGraph, tx, sign)
	return h[:]
}

// RunAPI implements RESTAPI
func (cli *Client) RunAPI() {

	// pathWithdraw and pathRevoke are incomplete
	http.HandleFunc("/pathAnnounce", func(w http.ResponseWriter, r *http.Request) {

		// parse the post Request
		// get the hash value
		// generate a transaction
		// respond with TxID

		var q dt.PathAnnounceMsg
		err := json.NewDecoder(r.Body).Decode(&q)
		fmt.Println(q)
		if err != nil {
			log.Println(err)
			// respond with bad request
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b, _ := json.Marshal(q)
		cli.IssueTransaction(b, 3)
		w.WriteHeader(http.StatusOK)
		// may be wrap it in a json object
		return
	})

	http.HandleFunc("/prefixAllocate", func(w http.ResponseWriter, r *http.Request) {

		// parse the post Request
		// get the hash value
		// generate a transaction
		// respond with TxID

		var q dt.AllocateMsg
		err := json.NewDecoder(r.Body).Decode(&q)
		fmt.Println(q)
		if err != nil {
			log.Println(err)
			// respond with bad request
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b, _ := json.Marshal(q)
		// send the transaction to the RIR with a
		// http post request and get the response with RIR signature

		url := "http://" + rirAddress + "/allocatePrefix"
		request := bytes.NewBuffer(b)

		log.Println("Sent to the RIR")

		resp, err := http.Post(url, "application/json", request)

		log.Println("Response from the RIR")

		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		cli.IssueTransaction(body, 3)
		w.WriteHeader(http.StatusOK)
		// may be wrap it in a json object
		return
	})

	http.ListenAndServe(":8989", nil)
}
