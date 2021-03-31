package tunnel

import (
	"github.com/oleiade/lane"
	"sync"
	"time"
)

// A user packet warped with sent time to be managed by the session buffers
type timedPacket struct {
	packet   UserPacket
	lastSent time.Time
}

// The outgoing message buffer resendQ, to manage outgoing resend messages.
type sendBuffer struct {
	resendQ *lane.PQueue
	sendQ   *lane.Queue
	mu      sync.Mutex
}

// The incoming message bugger priority pQueue, to manage incoming messages and safe transfer to the tcp proxy.
type rcvBuffer struct {
	pQueue   *lane.PQueue
	contains map[uint16]bool
	mu       sync.Mutex
}

// ctor for the incoming messages buffer
func newRcvBuffer() *rcvBuffer {

	return &rcvBuffer{pQueue: lane.NewPQueue(lane.MINPQ),
		contains: make(map[uint16]bool)}
}

// Adding a new packet to the incoming messages buffer.
func (r *rcvBuffer) add(packet UserPacket) {

	if _, ok := r.contains[packet.Id]; ok {
		return
	}
	r.contains[packet.Id] = true
	r.pQueue.Push(packet, int(packet.Id))
}

// querying the buffer for available packets to trasfer to the proxy tcp socket.
func (r *rcvBuffer) popRange(localAck uint16) (packets []UserPacket, ack uint16) {

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

// Ctor for the sending message buffer
func newSendBuffer() *sendBuffer {
	resendQ := lane.NewPQueue(lane.MINPQ)
	sendQ := lane.NewQueue()
	return &sendBuffer{resendQ: resendQ,
		sendQ: sendQ}
}

// Adding userPacket to the sending message buffer with or without timestamp.
func (sq *sendBuffer) add(packet UserPacket, isSent bool) {

	if isSent {
		sq.resendQ.Push(timedPacket{
			packet:   packet,
			lastSent: time.Time{},
		}, int(packet.Id))
	} else {
		sq.sendQ.Enqueue(packet)

	}
}

// Getting the next available packet to send based on the passed sending interval and ack given.
func (sq *sendBuffer) next(ack uint16, intervalMs int) (packet UserPacket) {

	sq.mu.Lock()
	if !sq.sendQ.Empty() {
		packet = sq.sendQ.Dequeue().(UserPacket)
		sq.add(packet, true)
	} else {
		for !sq.resendQ.Empty() {
			oldest, id := sq.resendQ.Head()

			//got ack - dropping packet
			if id <= int(ack) {
				sq.resendQ.Pop()
				continue
			}

			// no ack: check next for interval.
			if oldest.(timedPacket).lastSent.Before(time.Now().Add(-time.Millisecond * time.Duration(intervalMs))) {
				packet = oldest.(timedPacket).packet
				sq.resendQ.Pop()
				resent := newTimedPacket(packet)
				sq.resendQ.Push(resent, int(packet.Id))
				break
			} else {
				packet = UserPacket{}
				break
			}
		}
	}

	sq.mu.Unlock()

	return packet

}

func newTimedPacket(packet UserPacket) timedPacket {
	return timedPacket{packet: packet,
		lastSent: time.Now()}
}
