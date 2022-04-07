package sharding

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	pow "GO-DAG/Pow"
	"GO-DAG/serialize"
	"log"
	"time"
)

//VerifyDiscovery verifies shard signal from discovery
func VerifyDiscovery(tx dt.ShardSignal, signature []byte) bool {
	//Verify signature using PuK
	//Return valid or not
	s := serialize.Encode35(tx)
	SerialKey := tx.From
	PublicKey := Crypto.DeserializePublicKey(SerialKey[:])
	h := Crypto.Hash(s)
	sigVerify := Crypto.Verify(signature, PublicKey, h[:])
	return sigVerify
}

//VerifyShardTransaction verifies shard transaction
func VerifyShardTransaction(tx dt.ShardTransaction, signature []byte, difficulty int) bool {
	//Verify Puk
	//Verify signature
	//Verify Pow
	//Verify Shard number
	s := serialize.Encode36(tx)
	SerialKey := tx.From
	PublicKey := Crypto.DeserializePublicKey(SerialKey[:])
	h := Crypto.Hash(s)
	sigVerify := Crypto.Verify(signature, PublicKey, h[:])
	if sigVerify == false {
		log.Println("INVALID SIGNATURE")
	}
	// also verify the shardNo with the nonce
	return sigVerify
}

//MakeShardingtx Call on recieving sharding signal from discovery after forwarding it
func MakeShardingtx(Puk []byte, Signal dt.ShardSignal) (dt.ShardTransaction, error) {
	difficulty := 4
	var tx dt.ShardTransaction
	copy(tx.From[:], Puk)
	tx.Timestamp = time.Now().UnixNano()
	tx.Nonce = 0
	tx.ShardNo = 0
	tx.Identifier = Signal.Identifier
	pow.PoW(&tx, difficulty)
	return tx, nil
}
