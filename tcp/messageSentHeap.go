package tcp

import (
	"container/heap"
	"time"
)

type messageItem struct{
	timeSent   time.Time
	userPacket userPacket
	packetId   uint8
	index      int
}

type messageSentHeap []*messageItem

func (p messageSentHeap) Len() int { return len(p) }

func (p messageSentHeap) Less(i, j int) bool {
	return p[i].timeSent.Before(p[j].timeSent)
}

func (p messageSentHeap) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

func (p *messageSentHeap) Push(x interface{}) {
	n := len(*p)
	item := x.(*messageItem)
	item.index = n
	*p = append(*p, item)
}

func (p *messageSentHeap) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*p = old[0 : n-1]
	return item
}

func (p *messageSentHeap) update(item *messageItem, timeSent time.Time) {
	item.timeSent = timeSent
	heap.Fix(p, item.index)
}

