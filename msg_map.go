package tcpgodns

import (
	"fmt"
	"sort"
	"sync"
)

// A map to hold and manage the packet received from remote machine.
type MsgMap struct {
	mu        sync.Mutex
	packetMap map[uint8]UserPacket
}

func MsgMapFactory() *MsgMap {
	return &MsgMap{
		packetMap: make(map[uint8]UserPacket),
	}
}

func (m *MsgMap) Add(packet UserPacket) {

	// Already accepted the Packet
	if _, ok := m.packetMap[packet.Id]; ok {
		fmt.Printf("Packet %d already in Packet map\n", packet.Id)
		return
	}

	fmt.Printf("Adding Packet %d  to Packet map\n", packet.Id)
	m.packetMap[packet.Id] = packet

}

/*
Pop All packets that directly connected to the last seen Packet in MsgMap and
updates the last Sent Packet.
*/
func (m *MsgMap) PopRange(ack uint8) (packets []UserPacket, lastSeen uint8) {

	m.mu.Lock()

	next := ack + 1

	packets = make([]UserPacket, len(m.packetMap))
	i := 0
	lastSeen = ack
	keys := m.getSortedIds()

	for _, k := range keys {

		// packet k is older then the ack: throwing the packet.
		if next > k {
			delete(m.packetMap, k)
		}

		// at least one packet is missing for ordered pop
		if next < k {
			break
		}

		// next packet available and should pop.
		if next == k {
			packets[i] = m.packetMap[k]
			i++
			fmt.Printf("Removing Packet %d  from Packet map to the connection\n", packets[i].Id)

			lastSeen = next
			next++
			delete(m.packetMap, k)

		}

	}
	m.mu.Unlock()

	return packets, lastSeen
}

func (m *MsgMap) GetNext(ack uint8) (packet UserPacket) {
	m.mu.Lock()

	next := ack + 1

	keys := m.getSortedIds()

	for _, k := range keys {

		// packet k is older then the ack: throwing the packet.
		if next > k {
			delete(m.packetMap, k)
		}

		// next packet available and should pop.
		if next <= k {
			packet = m.packetMap[k]

			fmt.Printf("getting Packet %d to send. RemoteAck:%d \n", packet.Id, ack)

		}

	}
	m.mu.Unlock()

	return packet
}

func (m MsgMap) IsEmpty() bool {
	return len(m.packetMap) == 0
}

func (m MsgMap) getSortedIds() []uint8 {
	keys := make([]int, 0, len(m.packetMap))
	uKeys := make([]uint8, 0, len(keys))

	for k := range m.packetMap {
		keys = append(keys, int(k))
	}

	sort.Ints(keys)

	for k := range keys {
		uKeys = append(uKeys, uint8(k))
	}

	return uKeys
}
