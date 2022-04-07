package main

import(
	"fmt"
	"net"
	// "math/rand"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"encoding/json"
	"encoding/binary"
	"encoding/hex"
	"time"
	"os"
	"log"
	"bytes"
)

type sensordata struct {
	Data string
	SensorName string
	//SensorID string
	SmID string
	Start int64
	//End int64
}

func main() {
	conn,err := net.Dial("tcp",os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	simulteSmartMeter(conn)
}

func simulteSmartMeter(conn net.Conn) {
	fmt.Println("GENERATING FAKE SMARTMETER DATA")
	var fakeData sensordata
	fakeData.Data = "lighton"
	fakeData.SensorName = "livingroom-sensor"
	fakeData.SmID = loadSmID()
	for {
		fakeData.Start = time.Now().Unix()
		fmt.Println("================================================")
		color.Set(color.FgBlue)
		fmt.Println("		SmartMeterID -",fakeData.SmID)
		fmt.Println("		SensorData -",fakeData.Data)
		fmt.Println("		SensorName -",fakeData.SensorName)
		fmt.Println("		Timestamp -",fakeData.Start)
		color.Unset()
		msg,_ := json.Marshal(fakeData)
		var l uint32
		l = uint32(len(msg))
		buf := new(bytes.Buffer)
		binary.Write(buf,binary.LittleEndian, l)
		msg = append(buf.Bytes(),msg...)
		fmt.Println("SENDING DATA TO GATEWAY")
		conn.Write(msg)
		time.Sleep(30*time.Second)
	}
}

func loadSmID() string{
	filename := "SmID"
	var ID string
	if _,err := os.Stat(filename) ; os.IsNotExist(err) {
		f,_ := os.Create(filename)
		id := uuid.New()
		ID = hex.EncodeToString(id[:])
		f.WriteString(ID)
	} else {
		f,_ := os.Open(filename)
		text := make([]byte,1024)
		l,_ := f.Read(text)
		ID = string(text[:l])
	}
	return ID
}