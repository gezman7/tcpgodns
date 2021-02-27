package main

import (
	"container/heap"
	"fmt"
	"tcpgodns/tcp"
	"time"
)

func main() {

	var mr = tcp.MessageReceivedFactory()
	var packet = tcp.UserPacketFactory(4, 1, 0, 0, []byte("hello4"))
	var packet1 = tcp.UserPacketFactory(1, 1, 0, 0, []byte("hello1"))
	var packet2 =tcp.UserPacketFactory(2, 1, 0, 0, []byte("hello2"))
	var packet3 = tcp.UserPacketFactory(3, 1, 0, 0, []byte("hello3"))

	mr.Add(packet3)
	mr.Add(packet1)
	mr.Add(packet2)
	mr.Add(packet)
packets:= mr.PopAvailablePackets()
bytes:= tcp.ToBytes(packets)
fmt.Println(string(bytes))
for _,p := range packets{
fmt.Println(string(p.Id),string(p.Data))
	}

	pq := make(tcp.MessageSentHeap, len(packets))
	i:=0
	for _, packet := range packets {
		pq[i] = &tcp.MessageItem{
			Sent:    time.Now(),
			Packet: packet,
			Index:    i,
		}
		i++
	}
	heap.Init(&pq)

	// Insert a new item and then modify its priority.

	newPacket:=	tcp.UserPacketFactory(5, 1, 0, 0, []byte("hello5"))
	item:= tcp.MessageItemFactory(newPacket)

	heap.Push(&pq, &item)

	oldestPacket:= heap.Pop(&pq)

	fmt.Println(string(oldestPacket.(*tcp.MessageItem).Packet.Data))
	oldestPacket= heap.Pop(&pq)

	fmt.Println(string(oldestPacket.(*tcp.MessageItem).Packet.Data))

	oldestPacket= heap.Pop(&pq)

	fmt.Println(string(oldestPacket.(*tcp.MessageItem).Packet.Data))

}
