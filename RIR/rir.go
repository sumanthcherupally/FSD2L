package main

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// RIR is a central authority in BGP_DAG setup used to authenticate allocating,
// updating and revoking prefixes to the nodes

type responseMsg struct {
	MsgType      int
	MsgBody      []byte
	RIRSignature []byte
}

func fatalErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func listenForRequests(PrivateKey Crypto.PrivateKey) {

	http.HandleFunc("/allocatePrefix", func(w http.ResponseWriter, r *http.Request) {

		var allocateMsg dt.AllocateMsg

		err := json.NewDecoder(r.Body).Decode(&allocateMsg)

		fatalErr(err)

		b, err := json.Marshal(allocateMsg)

		fatalErr(err)

		hash := Crypto.Hash(b)

		sign := Crypto.Sign(hash[:], PrivateKey)

		var resp responseMsg

		resp.MsgType = 1
		resp.MsgBody = b
		resp.RIRSignature = sign

		s, err := json.Marshal(resp)

		fatalErr(err)

		w.WriteHeader(http.StatusOK)
		w.Write(s)

	})

	http.HandleFunc("/revokePrefix", func(w http.ResponseWriter, r *http.Request) {

		var revokeMsg dt.RevokeMsg

		err := json.NewDecoder(r.Body).Decode(&revokeMsg)

		fatalErr(err)

		b, err := json.Marshal(revokeMsg)

		fatalErr(err)

		hash := Crypto.Hash(b)

		sign := Crypto.Sign(hash[:], PrivateKey)

		var resp responseMsg

		resp.MsgType = 2
		resp.MsgBody = b
		resp.RIRSignature = sign

		s, err := json.Marshal(resp)

		fatalErr(err)

		w.WriteHeader(http.StatusOK)
		w.Write(s)

	})

	http.ListenAndServe(":7111", nil)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var PrivateKey Crypto.PrivateKey
	if Crypto.CheckForKeys() {
		PrivateKey = Crypto.LoadKeys()
	} else {
		PrivateKey = Crypto.GenerateKeys()
	}

	fmt.Println(hex.EncodeToString(Crypto.SerializePublicKey(&PrivateKey.PublicKey)))

	listenForRequests(PrivateKey)
}
