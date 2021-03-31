package tunnel

import (
	"bytes"
	"testing"
	"time"
)

func TestHandleProxyRead(t *testing.T) {

	s:=CreateSession(true,1,100)
	go s.handleProxyRead()
	bytesToStream:= []byte("abcd")
	s.reader <- bytesToStream
	packet:=s.FwdBuffer.next(0,0)

	if bytes.Equal(packet.Data,bytesToStream) {
		t.Errorf("a user packet should have passed to the fwd buffer with the right bytes.")
	}

}

func TestHandleProxyWrite(t *testing.T) {

	bytesToStream:= []byte("abcd")

	s:=CreateSession(true,1,100)
	packet1 := dataPacket(1, 1, 1, bytesToStream)

	go func() {
		for {
			var data = <-s.writer

			if !bytes.Equal(data,bytesToStream) {
				t.Errorf("Bytes should have come to the writer via go routine")
			}
		}}()

	s.handleProxyWrite(packet1)
	time.Sleep(time.Second*time.Duration(1))

}