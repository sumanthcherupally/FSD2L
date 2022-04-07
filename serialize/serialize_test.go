package serialize

import (
	dt "GO-DAG/DataTypes"
	"testing"
	"time"
)

func TestTransactionSerialization(t *testing.T) {
	var tx dt.Transaction
	var Pmsg dt.PathAnnounceMsg

	tx.Timestamp = time.Now().UnixNano()
	tx.TxType = 4
	tx.Nonce = 1

	Pmsg.Prefix = "192.168.0.1/24"
	Pmsg.DestinationAS = 2

	tx.Msg = append(tx.Msg, EncodePathAnnounceMsg(Pmsg)...)

	txS := Encode32(tx)
	var signature [72]byte
	txS = append(txS, signature[:]...)
	txD, _ := Decode32(txS, uint32(len(txS)))

	PmsgD := DecodePathAnnounceMsg(txD.Msg)

	if PmsgD.Prefix != Pmsg.Prefix {
		t.Errorf("serialization Failed")
	}

}
