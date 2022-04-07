package serialize

import (
	dt "GO-DAG/DataTypes"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"reflect"
)

//EncodeToHex converts byte slice to the string
func EncodeToHex(data []byte) string {
	return hex.EncodeToString(data)
}

//DecodeToBytes converts string to byte slice
func DecodeToBytes(data string) []byte {
	b, _ := hex.DecodeString(data)
	return b
}

//Encode32 serializes transaction to byte slice
func Encode32(t dt.Transaction) []byte {
	// iterating over a struct is painful in golang
	var b []byte
	b = append(b, EncodeToBytes(t.Timestamp)...)
	b = append(b, EncodeToBytes(t.TxType)...)
	b = append(b, EncodeToBytes(t.From)...)
	b = append(b, EncodeToBytes(t.LeftTip)...)
	b = append(b, EncodeToBytes(t.RightTip)...)
	b = append(b, EncodeToBytes(t.Nonce)...)
	b = append(b, t.Msg...)
	return b
}

// Encode35 serializes shardsignal
func Encode35(t dt.ShardSignal) []byte {
	var b []byte
	v := reflect.ValueOf(&t).Elem()
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i)
		b = append(b, EncodeToBytes(value.Interface())...)
	}
	return b
}

// Encode36 serializes shardtransaction
func Encode36(t dt.ShardTransaction) []byte {
	var b []byte
	v := reflect.ValueOf(&t).Elem()
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i)
		b = append(b, EncodeToBytes(value.Interface())...)
	}
	return b
}

//EncodeToBytes Converts any type to byte slice, supports strings,integers etc.
func EncodeToBytes(x interface{}) []byte {
	// encode based on type as binary package doesn't support strings
	switch x.(type) {
	default:
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, x)
		return buf.Bytes()
	}
}

//Decode32 Converts back byte slice to transaction
func Decode32(payload []byte, lenPayload uint32) (dt.Transaction, []byte) {

	signature := make([]byte, 72)
	copy(signature, payload[lenPayload-72:])

	var tx dt.Transaction

	txS := payload[:lenPayload-72]
	r := bytes.NewReader(txS[:8])
	binary.Read(r, binary.LittleEndian, &tx.Timestamp)

	r = bytes.NewReader(txS[8:12])
	binary.Read(r, binary.LittleEndian, &tx.TxType)

	copy(tx.From[:], txS[12:77])
	copy(tx.LeftTip[:], txS[77:109])
	copy(tx.RightTip[:], txS[109:141])

	r = bytes.NewReader(txS[141:145])
	binary.Read(r, binary.LittleEndian, &tx.Nonce)

	tx.Msg = append(tx.Msg, txS[145:]...)

	return tx, signature
}

//Decode35 Converts back byte slice to transaction
func Decode35(payload []byte, lenPayload uint32) (dt.ShardSignal, []byte) {
	signature := payload[lenPayload-72:]
	r := bytes.NewReader(payload[:lenPayload-72])
	var tx dt.ShardSignal
	err := binary.Read(r, binary.LittleEndian, &tx)
	if err != nil {
		log.Println(err)
	}
	return tx, signature
}

//Decode36 Converts back byte slice to transaction
func Decode36(payload []byte, lenPayload uint32) (dt.ShardTransaction, []byte) {
	signature := payload[lenPayload-72:]
	r := bytes.NewReader(payload[:lenPayload-72])
	var tx dt.ShardTransaction
	err := binary.Read(r, binary.LittleEndian, &tx)
	if err != nil {
		log.Println(err, "error due to serialization")
	}
	return tx, signature
}

// DecodePathAnnounceMsg decodes the Msg []byte of Tx
func DecodePathAnnounceMsg(payload []byte) dt.PathAnnounceMsg {
	var Pmsg dt.PathAnnounceMsg
	err := json.Unmarshal(payload, &Pmsg)
	if err != nil {
		log.Println(err)
	}
	return Pmsg
}

// EncodePathAnnounceMsg encodes the msg to a byte slice
func EncodePathAnnounceMsg(Pmsg dt.PathAnnounceMsg) []byte {
	var b []byte
	b, err := json.Marshal(Pmsg)
	if err != nil {
		log.Println(err)
	}
	return b
}

// EncodePathWithdrawMsg encodes the struct to byte slice
func EncodePathWithdrawMsg(PWmsg dt.PathWithdrawMsg) []byte {

	var b []byte
	b, err := json.Marshal(PWmsg)
	if err != nil {
		log.Println(err)
	}
	return b

}

// DecodePathWithdrawMsg decodes the Msg []byte of Tx
func DecodePathWithdrawMsg(payload []byte) dt.PathWithdrawMsg {
	var Pmsg dt.PathWithdrawMsg
	err := json.Unmarshal(payload, &Pmsg)
	if err != nil {
		log.Println(err)
	}
	return Pmsg
}

// EncodeAllocateMsg encodes the struct to byte slice
func EncodeAllocateMsg(Amsg dt.AllocateMsg) []byte {
	var b []byte
	b, err := json.Marshal(Amsg)
	if err != nil {
		log.Println(err)
	}
	return b
}

// DecodeAllocateMsg decodes the Msg []byte of Tx
func DecodeAllocateMsg(payload []byte) (dt.AllocateMsg, []byte) {
	type responseMsg struct {
		MsgType      int
		MsgBody      []byte
		RIRSignature []byte
	}
	var msg responseMsg
	var Amsg dt.AllocateMsg
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(msg.MsgBody, &Amsg)
	if err != nil {
		log.Println(err)
	}
	return Amsg, msg.RIRSignature
}

// EncodeRevokeMsg encodes the struct to byte slice
func EncodeRevokeMsg(Rmsg dt.RevokeMsg) []byte {
	var b []byte
	b, err := json.Marshal(Rmsg)
	if err != nil {
		log.Println(err)
	}
	return b
}

// DecodeRevokeMsg decodes the Msg []byte of Tx
func DecodeRevokeMsg(payload []byte) (dt.RevokeMsg, []byte) {
	type responseMsg struct {
		MsgType      int
		MsgBody      []byte
		RIRSignature []byte
	}
	var msg responseMsg
	var Rmsg dt.RevokeMsg
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(payload, &Rmsg)
	if err != nil {
		log.Println(err)
	}
	return Rmsg, msg.RIRSignature
}
