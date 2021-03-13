package tcpgodns

import (
	"container/heap"
	"encoding/base32"
	"fmt"
	"math/rand"
	"sync/atomic"
)

type PacketManager struct {
	SessionId     uint8
	localAck      uint8
	remoteAck     uint8
	resendHeap    SentHeap
	RcvMap        *MsgMap
	FwdMap        *MsgMap
	PidCounter    int32
	channelWrite  chan []byte
	channelRead   chan []byte
	packetChannel chan UserPacket
	isClient      bool
}

const MaxNum = 256

func ManagerFactory(cr chan []byte, cw chan []byte, isClient bool) *PacketManager {
	sentHeap := make(SentHeap, 0)
	heap.Init(&sentHeap)
	rcvMap := MsgMapFactory()
	fwdMap := MsgMapFactory()

	sessionId := rand.Intn(MaxNum)
	packetChannel := make(chan UserPacket)
	return &PacketManager{
		SessionId:     uint8(sessionId),
		localAck:      0,        // Id of the last Packet received in local machine. - TO SEND TO REMOTE
		remoteAck:     0,        // Id of the last packet ACKed from Remote - TO PROCESS LOCALLY
		resendHeap:    sentHeap, // todo: change mechanism of sent heap (maybe make it packet map as well
		RcvMap:        rcvMap,
		FwdMap:        fwdMap,
		PidCounter:    0,
		channelRead:   cr,
		channelWrite:  cw,
		packetChannel: packetChannel,
		isClient:      isClient,
	}
}

func (pm *PacketManager) FromLocalTcp() {

	for {
		var data = <-pm.channelRead // received byte stream from the local TCP connection
		packetId := uint8(atomic.AddInt32(&pm.PidCounter, 1))

		userPacket := UserPacket{
			Id:          packetId,
			SessionId:   pm.SessionId,
			LastSeenPid: pm.localAck,
			Data:        data,
			Flags:       DATA, //todo: Flags
		}
		//item := msgItemFactory(userPacket)

		fmt.Printf("FromLocalTcp: registred sentPacket: %d\n", userPacket.Id)

		pm.FwdMap.Add(userPacket)
		//heap.Push(&pm.resendHeap, &item) // add packet to the sending heap

		if pm.isClient {
			pm.packetChannel <- userPacket // send packet to dns
		}
	}
}

// Acknowledging unordered received packet, adding it to the received Map,
//and popping all packets in the correct order according to the last read packet.
func (pm *PacketManager) ToLocalTcp(packet UserPacket) {

	if packet.Id <= pm.localAck {
		fmt.Printf("last recieved ack:%d is higher then recived packet id:%d\n", pm.localAck, packet.Id)
		return
	}
	pm.RcvMap.Add(packet)
	packets, lastSeen := pm.RcvMap.PopRange(pm.localAck)


	fmt.Printf("Saving lastSeen packet id:%d\n", lastSeen)
	fmt.Printf("sending %d packets to LocalTcp\n", len(packets))

	pm.channelWrite <- ToBytes(packets)

	pm.localAck = lastSeen // updating the last sent packet to tcp

}

func ToBytes(packets []UserPacket) []byte {
	ByteStream := make([]byte, 0) // todo: adjust constants of slice
	for _, p := range packets {
		slice := p.Data[:]
		ByteStream = append(ByteStream, slice...)
	}

	return ByteStream
}

func encode(packet UserPacket) string {
	bytesToEncode := packet.PacketToBytes()
	return base32.HexEncoding.EncodeToString(bytesToEncode)
}

func decode(str string) UserPacket {

	bytes, err := base32.HexEncoding.DecodeString(str)
	if err != nil {
		fmt.Printf("Error with income decoding.")
	}

	return BytesToPacket(bytes)

}
