package tcpgodns

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type UserPacket struct {
	Id          uint8
	sessionId   uint8
	lastSeenPid uint8
	Data        []byte
	flags       uint8
	//todo:checksum
}

func userPacketFactory(packetId uint8, sessionId uint8, lastSeenPid uint8, flags uint8, data []byte) UserPacket {

	return UserPacket{
		Id:          packetId,
		sessionId:   sessionId,
		flags:       flags,
		lastSeenPid: lastSeenPid,
		Data:        data,
	}
}

type PacketManager struct {
	SessionId    uint8
	LastReceived uint8
	SentHeap     SentHeap
	ReceivedMap  *MsgRcv
	Conuter      int32
}

type msgItem struct {
	Sent   time.Time
	Packet UserPacket
	Index  int
}

const MaxNum = 256

func ManagerFactory() *PacketManager {
	sentHeap := make(SentHeap, 0)
	heap.Init(&sentHeap)
	receivedMap := MessageReceivedFactory()
	sessionId := rand.Intn(MaxNum)

	return &PacketManager{
		SessionId:    uint8(sessionId),
		LastReceived: 0,
		SentHeap:     sentHeap,
		ReceivedMap:  receivedMap,
		Conuter:      0,
	}
}

func msgItemFactory(packet UserPacket) msgItem {
	return msgItem{
		Sent:   time.Now(),
		Packet: packet,
	}
}

// Wanted API for PacketManager

func (pm *PacketManager) Read() []byte {
	return nil
}

func (pm *PacketManager) ManageOutcome(data []byte, c chan UserPacket) uint8 {

	packetId := uint8(atomic.AddInt32(&pm.Conuter, 1))

	userPacket := UserPacket{
		Id:          packetId,
		sessionId:   pm.SessionId,
		lastSeenPid: pm.LastReceived,
		Data:        data,
		flags:       0, //todo: flags
	}
	item := msgItemFactory(userPacket)

	fmt.Printf("registerd sentPacket: %d\n", userPacket.Id)
	heap.Push(&pm.SentHeap, &item)
	c <- userPacket
	return userPacket.Id
}

func (pm *PacketManager) ManageIncome(packet UserPacket) []byte {
	pm.LastReceived = packet.lastSeenPid
	pm.ReceivedMap.Add(packet)
	packets := pm.ReceivedMap.PopAvailablePackets()
	return ToBytes(packets)
}

func (pm *PacketManager) ManageResend(lastSeen uint8, c chan UserPacket) {
	flag := true
	curTime := time.Now()
	fmt.Printf("inside ManageResend -lastSeen:%d\n", lastSeen)
	for flag {
		if pm.SentHeap.Len() == 0 {
			fmt.Printf("heap is empty: no unseen messages in heap. lastSeen:%d\n", lastSeen)
			return
		}
		nextPacket := heap.Pop(&pm.SentHeap)

		if nextPacket.(*msgItem).Sent.After(curTime){
			heap.Push(&pm.SentHeap, nextPacket)
			return
		}
		if nextPacket.(*msgItem).Packet.Id <= lastSeen {
			continue
		}

		if nextPacket.(*msgItem).Packet.Id > lastSeen {
			packetToSend := UserPacket{
				Id:          nextPacket.(*msgItem).Packet.Id,
				sessionId:   nextPacket.(*msgItem).Packet.sessionId,
				lastSeenPid: pm.LastReceived,
				Data:        nextPacket.(*msgItem).Packet.Data,
				flags:       0,
			}
			fmt.Printf("resending packet:%v\n", packetToSend)
			c <- packetToSend
			msgItem := msgItemFactory(packetToSend)
			heap.Push(&pm.SentHeap, &msgItem)
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

type SentHeap []*msgItem

func (p SentHeap) Len() int { return len(p) }

func (p SentHeap) Less(i, j int) bool {
	return p[i].Sent.Before(p[j].Sent)
}

func (p SentHeap) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].Index = i
	p[j].Index = j
}

func (p *SentHeap) Push(x interface{}) {
	n := len(*p)
	item := x.(*msgItem)
	item.Index = n
	*p = append(*p, item)
}

func (p *SentHeap) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*p = old[0 : n-1]
	return item
}

func (p *SentHeap) update(item *msgItem, timeSent time.Time) {
	item.Sent = timeSent
	heap.Fix(p, item.Index)
}

type MsgRcv struct {
	lastProcessed uint8
	mu            sync.Mutex
	packetMap     map[uint8]UserPacket
}

func MessageReceivedFactory() *MsgRcv {
	return &MsgRcv{
		lastProcessed: 0,
		packetMap:     make(map[uint8]UserPacket),
	}
}

func (mr *MsgRcv) Add(packet UserPacket) {

	// already got ACK for higher PID
	if packet.Id <= mr.lastProcessed {
		return
	}

	// Already accepted the Packet
	if _, ok := mr.packetMap[packet.Id]; ok {
		fmt.Printf("Packet %d already in Packet map", packet.Id)
		return
	}

	fmt.Printf("Adding Packet %d  to Packet map", packet.Id)
	mr.packetMap[packet.Id] = packet

}

/*
Pop All packets that directly connected to the last sent Packet in MsgRcv and
updates the last Sent Packet.
*/
func (mr *MsgRcv) PopAvailablePackets() []UserPacket {

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
			packets[i] = mr.packetMap[uint8(k)]
			fmt.Printf("Removing Packet %d  from Packet map to the connection", packets[i].Id)

			lastSent = next
			next++
			delete(mr.packetMap, uint8(k))
		} else {
			break
		}
	}
	mr.lastProcessed = lastSent
	mr.mu.Unlock()

	return packets

}
