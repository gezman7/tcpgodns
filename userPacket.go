package tcpgodns

type UserPacket struct {
	Id          uint8
	SessionId   uint8
	LastSeenPid uint8
	Data        []byte
	Flags       uint8
	//todo:checksum
}

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
