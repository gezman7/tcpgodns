package tcpgodns

import (
	"fmt"
	"sort"
	"sync"
)

// A map to hold and manage the packet received from remote machine.
type MsgRcv struct {
	mu        sync.Mutex
	packetMap map[uint8]UserPacket
}

func MessageReceivedFactory() *MsgRcv {
	return &MsgRcv{
		packetMap: make(map[uint8]UserPacket),
	}
}

func (mr *MsgRcv) Add(packet UserPacket, ack uint8) {

	// already got ACK for higher PID
	if packet.Id <= ack {
		fmt.Printf("last recieved Id:%d is higher then recived packet id:%d\n", ack, packet.Id)
		return
	}

	// Already accepted the Packet
	if _, ok := mr.packetMap[packet.Id]; ok {
		fmt.Printf("Packet %d already in Packet map\n", packet.Id)
		return
	}

	fmt.Printf("Adding Packet %d  to Packet map\n", packet.Id)
	mr.packetMap[packet.Id] = packet

}

/*
Pop All packets that directly connected to the last seen Packet in MsgRcv and
updates the last Sent Packet.
*/
func (mr *MsgRcv) PopAvailablePackets(ack uint8) (packets []UserPacket, lastSeen uint8) {

	mr.mu.Lock()

	next := ack + 1

	keys := make([]int, 0, len(mr.packetMap))

	for k := range mr.packetMap {
		keys = append(keys, int(k))
	}

	sort.Ints(keys)
	packets = make([]UserPacket, len(mr.packetMap))
	lastSeen = ack

	for i, k := range keys {
		if next == uint8(k) {
			packets[i] = mr.packetMap[uint8(k)]
			fmt.Printf("Removing Packet %d  from Packet map to the connection\n", packets[i].Id)

			lastSeen = next
			next++
			delete(mr.packetMap, uint8(k))
		} else {
			break
		}
	}
	mr.mu.Unlock()

	return packets, lastSeen
}
