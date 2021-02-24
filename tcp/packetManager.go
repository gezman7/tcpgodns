package tcp

import (
	"container/heap"
	"math/rand"
	"sync/atomic"
	"time"

)

type packetManager struct{
	sessionId   uint8
	lastSeenPid uint8
	messageSent messageSentHeap
	idConuter int32

}


func (pm *packetManager) Init(message *userPacket){
	heap.Init(&pm.messageSent)
	pm.idConuter = 1
	pm.lastSeenPid =0
	pm.sessionId = uint8(rand.Uint32())
}

func (pm *packetManager) AddSentMessage(data *[]byte){

	packetId :=uint8(atomic.AddInt32(&pm.idConuter, 1))

	userPacket := userPacket{
		packetId:    packetId,
		sessionId:   pm.sessionId,
		lastSeenPid: pm.lastSeenPid,
		data:        *data,
		flags:       0, //todo: flags
	}
	item := messageItem{
		timeSent:   time.Now(),
		userPacket: userPacket,
		packetId:   packetId,
		index:      0,
	}

	// SendUserPacket
	heap.Push(&pm.messageSent, item)
}


