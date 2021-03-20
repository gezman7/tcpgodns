package tcpgodns

import (
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type UserPacket struct {
	Id          uint8
	SessionId   uint8
	LastSeenPid uint8
	Data        []byte
	Flags       uint8
	//todo:checksum
}

const (
	NO_OP            = 0
	DATA             = 1
	CONNECT          = 2
	ESTABLISHED      = 3
	CLOSE            = 4
	CLOSED           = 5
)

func (up UserPacket) PacketToBytes() []byte {

	header := []byte{up.Id, up.SessionId, up.LastSeenPid, up.Flags}
	data := up.Data[:]
	return append(header, data...)
}

func BytesToPacket(data []byte) UserPacket {

	return UserPacket{
		Id:          data[0],
		SessionId:   data[1],
		LastSeenPid: data[2],
		Data:        data[4:],
		Flags:       data[3],
	}
}

func EmptyPacket(sessionId uint8, lastSeenPid uint8) UserPacket {
	return UserPacket{
		Id:          0,
		SessionId:   sessionId,
		LastSeenPid: lastSeenPid,
		Data:        nil,
		Flags:       NO_OP,
	}
}

func ConnectPacket() UserPacket {
	buf := make([]byte, 8)
	time := getTime()
	binary.PutVarint(buf, time)

	return UserPacket{
		Id:          0,
		SessionId:   0,
		LastSeenPid: 0,
		Data:        buf,
		Flags:       CONNECT,
	}
}

func ClosePacket(sessionId uint8, msg string) UserPacket {
	var arr []byte

	copy(arr[:], []byte(msg))


	return UserPacket{
		Id:          0,
		SessionId:   sessionId,
		LastSeenPid: 0,
		Data:        arr,
		Flags:       CLOSE,
	}
}

func (up UserPacket) GetInterval() (interval int, ok bool) {

	if up.Flags != ESTABLISHED {
		fmt.Printf("cannot get interval since packet is not ESTABLISHED packet")
		return
	}
	sentTime, _ := binary.Varint(up.Data)
	curTime := getTime()
	interval = int(curTime - sentTime)
	ok = true
	return
}

func getTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}


func encode(packet UserPacket) string {
	bytesToEncode := packet.PacketToBytes()

	encoded := base32.HexEncoding.EncodeToString(bytesToEncode)
	hostnameEncoded := strings.ReplaceAll(encoded, "=", "y")
	return hostnameEncoded

}

func decode(str string) UserPacket {
	hostnameEncoded := strings.ReplaceAll(str, "y", "=")

	bytes, err := base32.HexEncoding.DecodeString(hostnameEncoded)
	if err != nil {
		fmt.Printf("Error with income decoding.")
	}

	return BytesToPacket(bytes)

}
