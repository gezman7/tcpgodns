package tcpgodns

import (
	"github.com/oleiade/lane"
	"sync"
	"time"
)

type timedPacket struct {
	packet   UserPacket
	lastSent time.Time
}

func newTimedPacket(packet UserPacket) timedPacket {
	return timedPacket{packet: packet,
		lastSent: time.Now()}
}

type SendQueue struct {
	queue *lane.Queue
	mu    sync.Mutex
}

type RcvPQueue struct {
	pQueue   *lane.PQueue
	contains map[uint8]bool
	mu       sync.Mutex
}

func NewRcvPQueue() *RcvPQueue {

	return &RcvPQueue{pQueue: lane.NewPQueue(lane.MINPQ),
		contains: make(map[uint8]bool)}
}

func (r *RcvPQueue) Add(packet UserPacket) {

	if _, ok := r.contains[packet.Id]; ok {
		return
	}
	r.contains[packet.Id] = true
	r.pQueue.Push(packet, int(packet.Id))
}

func (r *RcvPQueue) PopRange(localAck uint8) (packets []UserPacket, ack uint8) {

	packets = make([]UserPacket, r.pQueue.Size())
	index := 0
	ack = localAck

	seen := int(localAck)
	r.mu.Lock()
	for !r.pQueue.Empty() {
		_, id := r.pQueue.Head()

		if id <= seen {
			temp, _ := r.pQueue.Pop()
			delete(r.contains, temp.(UserPacket).Id)

		}
		if id > seen+1 {
			return packets, ack
		}

		if id == seen+1 {
			temp, _ := r.pQueue.Pop()
			packets[index] = temp.(UserPacket)
			delete(r.contains, temp.(UserPacket).Id)
			ack = packets[index].Id
			seen++
			index++
		}
	}
	r.mu.Unlock()

	return packets, ack
}

func NewSendQueue() *SendQueue {
	queue := lane.NewQueue()
	return &SendQueue{queue: queue}
}

func (sq *SendQueue) AddSent(packet UserPacket) {
	sq.queue.Enqueue(newTimedPacket(packet))
}

func (sq *SendQueue) Add(packet UserPacket) {
	sq.queue.Enqueue(timedPacket{
		packet:   packet,
		lastSent: time.Time{},
	})
}

func (sq *SendQueue) GetNext(ack uint8, intervalMs int) (packet UserPacket) {

	sq.mu.Lock()
	for !sq.queue.Empty() {
		oldest := sq.queue.Head()

		//got ack - dropping packet
		if oldest.(timedPacket).packet.Id <= ack {
			sq.queue.Dequeue()
			continue
		}
		// no ack: check GetNext for interval.
		if oldest.(timedPacket).lastSent.Before(time.Now().Add(-time.Millisecond * time.Duration(intervalMs))) {
			packet = oldest.(timedPacket).packet
			sq.queue.Dequeue()
			resent := newTimedPacket(packet)
			sq.queue.Enqueue(resent)
			break
		} else {
			packet = UserPacket{}
			break
		}
	}

	sq.mu.Unlock()

	return packet

}
