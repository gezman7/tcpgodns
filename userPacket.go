package tcpgodns

type UserPacket struct {
	Id          uint8
	SessionId   uint8
	LastSeenPid uint8
	Data        []byte
	Flags       uint8
	//todo:checksum
}

const (
	NO_OP = 0
	DATA  = 1
	TIMER = 2
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
