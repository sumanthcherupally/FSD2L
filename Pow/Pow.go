package pow

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/serialize"
	"strings"
)

//PoW calculates the Nonce field in tx to match the difficulty
func PoW(item interface{}, difficulty int) {
	switch tx := item.(type) {
	case *dt.Transaction:
		for {
			s := serialize.Encode32(*tx)
			hash := Crypto.Hash(s)
			h := Crypto.EncodeToHex(hash[:])
			if h[:difficulty] == strings.Repeat("0", difficulty) {
				break
			}
			tx.Nonce++
		}
	case *dt.ShardTransaction:
		for {
			s := serialize.Encode36(*tx)
			hash := Crypto.Hash(s)
			h := Crypto.EncodeToHex(hash[:])
			if h[:difficulty] == strings.Repeat("0", difficulty) {
				break
			}
			tx.Nonce++
			tx.ShardNo = tx.ShardNo ^ 1 //flip everytime 1 to 0 to 1
		}
	}
}

//VerifyPoW verifies if the nonce field of tx matches the difficulty
func VerifyPoW(tx interface{}, difficulty int) bool {
	// optimize this method by passing serial form of transaction
	var s []byte
	switch t := tx.(type) {
	case dt.Transaction:
		s = serialize.Encode32(t)
	case dt.ShardTransaction:
		s = serialize.Encode36(t)
	}
	hash := Crypto.Hash(s)
	h := Crypto.EncodeToHex(hash[:])
	if h[:difficulty] == strings.Repeat("0", difficulty) {
		return true
	}
	return false
}
