package tunnel

import (
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// an application level network packet to make a reliable connection over unreliable channel.
type UserPacket struct {
	Id          uint16
	SessionId   uint8
	LastSeenPid uint16
	Data        []byte
	Flags       uint8
	//todo:checksum
}

// UserPacket flags
const (
	NO_OP       = 0
	DATA        = 1
	CONNECT     = 2
	ESTABLISHED = 3
	CLOSE       = 4
	CLOSED      = 5
)

// Markers for the header information byte split.
const (
	ID_START        = 0
	ID_END          = 2
	LAST_SEEN_START = 2
	LAST_SEEN_END   = 4
	SESSION_ID      = 4
	FLAGS           = 5
	START_DATA      = 6
)

func (up UserPacket) packetToBytes() []byte {

	header := make([]byte, 6)
	binary.BigEndian.PutUint16(header[ID_START:ID_END], up.Id)
	binary.BigEndian.PutUint16(header[LAST_SEEN_START:LAST_SEEN_END], up.LastSeenPid)
	header[SESSION_ID] = up.SessionId
	header[FLAGS] = up.Flags
	data := up.Data[:]
	return append(header, data...)
}

func bytesToPacket(data []byte) UserPacket {

	return UserPacket{
		Id:          binary.BigEndian.Uint16(data[ID_START:ID_END]),
		LastSeenPid: binary.BigEndian.Uint16(data[LAST_SEEN_START:LAST_SEEN_END]),
		SessionId:   data[SESSION_ID],
		Data:        data[START_DATA:],
		Flags:       data[FLAGS],
	}
}

func noOpPacket(sessionId uint8, lastSeenPid uint16) UserPacket {
	return UserPacket{
		Id:          0,
		SessionId:   sessionId,
		LastSeenPid: lastSeenPid,
		Data:        nil,
		Flags:       NO_OP,
	}
}

func dataPacket(id uint16,sessionId uint8, lastSeenPid uint16, data []byte) UserPacket {
	return UserPacket{
		Id:          id,
		SessionId:   sessionId,
		LastSeenPid: lastSeenPid,
		Data:        data,
		Flags:       DATA,
	}
}
func connectPacket() UserPacket {
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

func closePacket(sessionId uint8, msg string) UserPacket {
	var arr []byte

	copy(arr[:], msg)

	return UserPacket{
		Id:          0,
		SessionId:   sessionId,
		LastSeenPid: 0,
		Data:        arr,
		Flags:       CLOSE,
	}
}

func (up UserPacket) interval() (interval int, ok bool) {

	if up.Flags != ESTABLISHED && up.Flags != CONNECT {
		fmt.Printf("cannot get interval since the packet is not in CONNECT-ESTABLISHED flow.")
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
	bytesToEncode := packet.packetToBytes()

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

	return bytesToPacket(bytes)

}
