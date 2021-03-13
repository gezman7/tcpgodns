package tcpgodns

import (
	"container/heap"
	"fmt"
	"time"
)

type SentHeap []*msgItem

func msgItemFactory(packet UserPacket) msgItem {
	return msgItem{
		Sent:   time.Now(),
		Packet: packet,
	}
}
type msgItem struct {
	Sent   time.Time
	Packet UserPacket
	Index  int
}

func (p SentHeap) Len() int { return len(p) }

func (p SentHeap) Less(i, j int) bool {
	if p[j]==nil{
		return true
	}
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



