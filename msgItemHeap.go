package tcpgodns

import (
	"container/heap"
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



