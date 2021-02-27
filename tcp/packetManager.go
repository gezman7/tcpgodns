package tcp

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sync/atomic"
)

type PacketManager struct {
	SessionId    uint8
	LastReceived uint8
	SentHeap     MessageSentHeap
	ReceivedMap  *MessageReceived
	Conuter      int32
}

const MaxNum = 256

func PacketManagerFactory() *PacketManager {
	sentHeap := make(MessageSentHeap, 0)
	heap.Init(&sentHeap)
	receivedMap := MessageReceivedFactory()
	sessionId := rand.Intn(MaxNum)

	return &PacketManager{
		SessionId:    uint8(sessionId),
		LastReceived: 0,
		SentHeap:     sentHeap,
		ReceivedMap:  receivedMap,
		Conuter:      1,
	}
}

func (pm *PacketManager) ManageOutcome(data []byte) {

	packetId := uint8(atomic.AddInt32(&pm.Conuter, 1))

	userPacket := UserPacket{
		Id:          packetId,
		sessionId:   pm.SessionId,
		lastSeenPid: pm.LastReceived,
		Data:        data,
		flags:       0, //todo: flags
	}
	item := MessageItemFactory(userPacket)

	fmt.Printf("sent packet: %d", userPacket.Id)
	heap.Push(&pm.SentHeap, &item)
}

func (pm *PacketManager) ManageIncome(packet UserPacket) []byte {
	pm.LastReceived = packet.lastSeenPid
	pm.ReceivedMap.Add(packet)
	packets := pm.ReceivedMap.PopAvailablePackets()
	return ToBytes(packets)
}

func ToBytes(packets []UserPacket) []byte {
	ByteStream := make([]byte, 0) // todo: adjust constants of slice
	for _, p := range packets {
		slice := p.Data[:]
		ByteStream = append(ByteStream, slice...)
	}

	return ByteStream
}
