package tcp

import (
	"container/heap"
	"time"
)

type MessageItem struct{
	Sent   time.Time
	Packet UserPacket
	Index  int
}

func MessageItemFactory(packet UserPacket) MessageItem {
	return MessageItem{
		Sent:   time.Now(),
		Packet: packet,
	}
}
type MessageSentHeap []*MessageItem

func (p MessageSentHeap) Len() int { return len(p) }

func (p MessageSentHeap) Less(i, j int) bool {
	return p[i].Sent.Before(p[j].Sent)
}

func (p MessageSentHeap) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].Index = i
	p[j].Index = j
}

func (p *MessageSentHeap) Push(x interface{}) {
	n := len(*p)
	item := x.(*MessageItem)
	item.Index = n
	*p = append(*p, item)
}

func (p *MessageSentHeap) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*p = old[0 : n-1]
	return item
}

func (p *MessageSentHeap) update(item *MessageItem, timeSent time.Time) {
	item.Sent = timeSent
	heap.Fix(p, item.Index)
}

