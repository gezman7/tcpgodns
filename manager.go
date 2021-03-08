package tcpgodns

import (
	"container/heap"
	"encoding/base32"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

type PacketManager struct {
	SessionId     uint8
	localAck      uint8
	remoteAck     uint8
	resendHeap    SentHeap
	ReceivedMap   *MsgRcv
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
	receivedMap := MessageReceivedFactory()
	sessionId := rand.Intn(MaxNum)
	packetChannel := make(chan UserPacket)
	return &PacketManager{
		SessionId:     uint8(sessionId),
		localAck:      0, // Id of the last Packet received in local machine. - TO SEND TO REMOTE
		remoteAck:     0, // Id of the last packet ACKed from Remote - TO PROCESS LOCALLY
		resendHeap:    sentHeap, // todo: change mechanism of sent heap (maybe make it packet map as well
		ReceivedMap:   receivedMap,
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
		item := msgItemFactory(userPacket)

		fmt.Printf("FromLocalTcp: registred sentPacket: %d\n", userPacket.Id)

		heap.Push(&pm.resendHeap, &item) // add packet to the sending heap

		if pm.isClient {
			pm.packetChannel <- userPacket // send packet to dns
		}
	}
}

// Acknowledging unordered received packet, adding it to the received Map,
//and popping all packets in the correct order according to the last read packet.
func (pm *PacketManager) ToLocalTcp(packet UserPacket) {
	pm.ReceivedMap.Add(packet, pm.localAck)
	packets, lastSeen := pm.ReceivedMap.PopAvailablePackets(pm.localAck)

	pm.localAck = lastSeen

	fmt.Printf("Saving lastSeen packet id:%d\n", lastSeen)
	fmt.Printf("sending %d packets to LocalTcp\n", len(packets))
	pm.channelWrite <- ToBytes(packets)
}

func (pm *PacketManager) ManageResend(lastSeen uint8, c chan UserPacket) {
	curTime := time.Now()
	fmt.Printf("inside ManageResend -lastSeen:%d\n", lastSeen)
	for {
		if pm.resendHeap.Len() == 0 {
			fmt.Printf("heap is empty: no unseen messages in heap. lastSeen:%d\n", lastSeen)
			return
		}
		nextPacket := heap.Pop(&pm.resendHeap)

		if nextPacket.(*msgItem).Sent.After(curTime) {
			heap.Push(&pm.resendHeap, nextPacket)
			return
		}
		if nextPacket.(*msgItem).Packet.Id <= lastSeen {
			continue
		}

		if nextPacket.(*msgItem).Packet.Id > lastSeen {
			packetToSend := UserPacket{
				Id:          nextPacket.(*msgItem).Packet.Id,
				SessionId:   nextPacket.(*msgItem).Packet.SessionId,
				LastSeenPid: pm.localAck,
				Data:        nextPacket.(*msgItem).Packet.Data,
				Flags:       DATA,
			}
			fmt.Printf("resending packet:%v\n", packetToSend)
			c <- packetToSend
			msgItem := msgItemFactory(packetToSend)
			heap.Push(&pm.resendHeap, &msgItem)
		}

	}
}

func (pm PacketManager) NextPacketToSend(gotAck uint8) UserPacket {
	curTime := time.Now()
	for {
		if pm.resendHeap.Len() == 0 {
			fmt.Printf("heap is empty: no unseen messages in heap. lastSeen:%d\n", gotAck)
			return EmptyPacket(pm.SessionId, pm.localAck)
		}
		nextPacket := heap.Pop(&pm.resendHeap)

		if nextPacket.(*msgItem).Sent.After(curTime) { // we search all the heap,and no packet found.
			heap.Push(&pm.resendHeap, nextPacket)
			return EmptyPacket(pm.SessionId, pm.localAck)

		}
		if nextPacket.(*msgItem).Packet.Id <= gotAck { //drop packet
			continue
		}

		if nextPacket.(*msgItem).Packet.Id > gotAck {
			fmt.Printf("Next packet found with unAcked Id:%d ack:%d \n", nextPacket.(*msgItem).Packet.Id, gotAck)

			packetToSend := UserPacket{
				Id:          nextPacket.(*msgItem).Packet.Id,
				SessionId:   nextPacket.(*msgItem).Packet.SessionId,
				LastSeenPid: pm.localAck,
				Data:        nextPacket.(*msgItem).Packet.Data,
				Flags:       DATA,
			}
			fmt.Printf("resending packet:%v\n", packetToSend)
			msgItem := msgItemFactory(packetToSend)
			heap.Push(&pm.resendHeap, &msgItem)
			return packetToSend

		}
	}
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
