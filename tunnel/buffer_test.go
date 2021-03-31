package tunnel

import (
	"testing"
)

func TestBufferRcv(t *testing.T) {
	packet1 := dataPacket(1, 1, 1, []byte("1"))
	packet2 := dataPacket(2, 1, 1, []byte("2"))
	packet3 := dataPacket(3, 1, 1, []byte("3"))
	t.Log("Adding 3 packets in the wrong order ")
	buf := newRcvBuffer()
	buf.add(packet2)
	buf.add(packet3)
	buf.add(packet1)

	packets, ack := buf.popRange(0)

	if len(packets) != 3 {
		t.Errorf("not all packet popped as excpected. exprected:%d, got: %d", 3, len(packets))
	}
	if ack != 3 {
		t.Errorf("ack returned is wrong expected ")
	}
	if packets[0].Id != packet1.Id {
		t.Error("Wrong packet popped first")
	}
	if packets[1].Id != packet2.Id {
		t.Error("Wrong packet popped second.")
	}
	if packets[2].Id != packet3.Id {
		t.Error("Wrong packet popped last")
	}
}
func TestBufferSend(t *testing.T) {
	packet1 := dataPacket(1, 1, 1, []byte("1"))
	packet2 := dataPacket(2, 1, 1, []byte("2"))
	packet3 := dataPacket(3, 1, 1, []byte("3"))
	t.Log("Adding 3 packets  ")
	buf := newSendBuffer()

	buf.add(packet1, false)
	buf.add(packet2, false)
	buf.add(packet3, false)
	packetToSend := buf.next(0, 0)

	if packetToSend.Id != packet1.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet1.Id, packetToSend.Id)
	}
	packetToSend = buf.next(0, 0)

	if packetToSend.Id != packet2.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet1.Id, packetToSend.Id)
	}
	packetToSend = buf.next(0, 0)

	if packetToSend.Id != packet3.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet1.Id, packetToSend.Id)
	}
	packetToSend = buf.next(1, 0)

	if packetToSend.Id != packet2.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet1.Id, packetToSend.Id)
	}


	packetToSend = buf.next(2, 0)

	if packetToSend.Id != packet3.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet2.Id, packetToSend.Id)
	}
	packetToSend = buf.next(2, 0)

	if packetToSend.Id != packet3.Id {
		t.Errorf("packer:%d should have come next but packet %d came", packet3.Id, packetToSend.Id)
	}

	packetToSend = buf.next(3, 0)

	if packetToSend.Id != 0 {
		t.Errorf("no packet should have pooped, but packet:%d did",packetToSend.Id)
	}

}
