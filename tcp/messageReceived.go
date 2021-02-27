package tcp

import (
	"fmt"
	"sort"
	"sync"
)

type MessageReceived struct {
	lastProcessed uint8
	mu            sync.Mutex
	packetMap     map[uint8]UserPacket
}

func MessageReceivedFactory() *MessageReceived {
	return &MessageReceived{
		lastProcessed: 0,
		packetMap:     make(map[uint8]UserPacket),
	}
}

func (mr *MessageReceived) Add(packet UserPacket) {

	// already got ACK for higher PID
	if packet.Id <= mr.lastProcessed {
		return
	}

	// Already accepted the Packet
	if _, ok := mr.packetMap[packet.Id]; ok {
		fmt.Printf("Packet %d already in Packet map", packet.Id)
		return
	}
	mr.packetMap[packet.Id] = packet

}

/*
Pop All packets that directly connected to the last sent Packet in MessageReceived and
updates the last Sent Packet.
*/
func (mr *MessageReceived) PopAvailablePackets() []UserPacket {

	mr.mu.Lock()

	next := mr.lastProcessed + 1

	keys := make([]int, 0, len(mr.packetMap))

	for k := range mr.packetMap {
		keys = append(keys, int(k))
	}

	sort.Ints(keys)
	packets := make([]UserPacket, len(mr.packetMap))
	lastSent := mr.lastProcessed

	for i, k := range keys {
		if next == uint8(k) {
			packets[i] =  mr.packetMap[uint8(k)]
			lastSent = next
			next++
			delete(mr.packetMap,uint8(k))
		} else {
			break
		}
	}
	mr.lastProcessed = lastSent
	mr.mu.Unlock()

	return packets

}
