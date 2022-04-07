package sync

import (
	"GO-DAG/Crypto"
	dt "GO-DAG/DataTypes"
	"GO-DAG/p2p"
	"encoding/json"
	"log"
)

// Sync copies the DAG from other nodes
func Sync(dag *dt.DAG, p p2p.Peer) {
	var msg p2p.Msg
	msg.ID = 33
	p.Send(msg)
	replyMsg, _ := p.GetMsg()
	var hashes []string
	json.Unmarshal(replyMsg.Payload, &hashes)
	log.Println(len(hashes))
	for _, hash := range hashes {
		msg.ID = 34
		msg.Payload = Crypto.DecodeToBytes(hash)
		msg.LenPayload = uint32(len(msg.Payload))
		p.Send(msg)
		// replyMsg, _ = p.GetMsg()
		// tx, sign := serialize.Decode(replyMsg.Payload, replyMsg.LenPayload)
		// storage.AddTransaction(dag, tx, sign)
	}
}
