package tcp

type UserPacket struct {
	Id          uint8
	sessionId   uint8
	lastSeenPid uint8
	Data        []byte
	flags       uint8
	//todo:checksum
}

func UserPacketFactory(packetId uint8, sessionId uint8, lastSeenPid uint8, flags uint8, data []byte) UserPacket {

	return UserPacket{
		Id:          packetId,
		sessionId:   sessionId,
		flags:       flags,
		lastSeenPid: lastSeenPid,
		Data:        data,
	}

}
func (up *UserPacket) SetAll(packetId uint8, sessionId uint8, lastSeenPid uint8, flags uint8, data []byte) {

}

func UserPacketComparator(a, b interface{}) int {
	aAsserted := a.(UserPacket)
	bAsserted := b.(UserPacket)
	switch {
	case aAsserted.Id > bAsserted.Id:
		return 1
	case aAsserted.Id < bAsserted.Id:
		return -1
	default:
		return 0
	}
}

func (up *UserPacket) CompareTo(o UserPacket) int {
	if up.Id < o.Id {
		return -1
	}

	if up.Id == o.Id {
		return 0
	}
	return 1
}
