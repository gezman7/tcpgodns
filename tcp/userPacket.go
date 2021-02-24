package tcp


type userPacket struct {
	packetId uint8
	sessionId uint8
	lastSeenPid uint8
	data []byte
	flags uint8
	//todo:checksum
}



func (up *userPacket) SetAll(packetId uint8,sessionId uint8, lastSeenPid uint8,flags uint8, data []byte){
	up.packetId = packetId
	up.sessionId =sessionId
	up.flags = flags
	up.lastSeenPid = lastSeenPid
	up.data=data
}

