package tcpgodns

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"os"
	"sync/atomic"
)

type Session struct {
	conn           net.Conn
	SessionId      uint8
	localAck       uint8
	remoteAck      uint8
	RcvBuffer      *RcvPQueue
	FwdBuffer      *SendQueue
	PidCounter     int32
	writer         chan []byte
	reader         chan []byte
	packetChannel  chan UserPacket
	isClient       bool
	resendInterval int
	closed         bool
	localTcp       *tcpCommunicator
}

const MinInterval = 40

func EstablishSession(isClient bool, sessionId uint8, rtt int) (session *Session) {
	cr := make(chan []byte)
	cw := make(chan []byte)
	packetChannel := make(chan UserPacket)

	resendInterval := math.Max(MinInterval, float64(rtt))

	session = &Session{
		SessionId:      sessionId,
		localAck:       0, // Id of the // last Packet received in local machine. - TO SEND TO REMOTE
		remoteAck:      0, // Id of the last packet ACKed from Remote - TO PROCESS LOCALLY
		RcvBuffer:      NewRcvPQueue(),
		FwdBuffer:      NewSendQueue(),
		PidCounter:     0,
		reader:         cr,
		writer:         cw,
		packetChannel:  packetChannel,
		isClient:       isClient,
		resendInterval: int(resendInterval),
		closed:         false,
	}

	return session

}

func (s *Session) ConnectLocal(communicator *tcpCommunicator) {
	s.localTcp = communicator

	go s.HandleLocalTCP()
	go communicator.handleRead(s.reader, s.OnLocalClose)
	go communicator.handleWrite(s.writer, s.OnLocalClose)

}

func (s *Session) HandleLocalTCP() {

	for {
		// received byte stream from the local TCP conn
		var byteStream = <-s.reader

		// close goroutine with a "close" key word
		if bytes.Equal(byteStream, []byte("close")) {
			return
		}

		packetId := uint8(atomic.AddInt32(&s.PidCounter, 1))

		userPacket := UserPacket{
			Id:          packetId,
			SessionId:   s.SessionId,
			LastSeenPid: s.localAck,
			Data:        byteStream,
			Flags:       DATA,
		}

		fmt.Printf("HandleLocalTCP: %d bytes warpped in user Packet id:%d\n", len(byteStream), userPacket.Id)

		if s.isClient {
			s.packetChannel <- userPacket // send packet to dns client
			s.FwdBuffer.AddSent(userPacket)
		} else {
			s.FwdBuffer.Add(userPacket) // add packet to the send buffer
		}
	}
}

// Acknowledging unordered received packet, adding it to the received buffer,
//and popping all packets in the correct order according to the last read packet.
func (s *Session) ToLocalTcp(packet UserPacket) {

	if packet.Id <= s.localAck {
		fmt.Printf("localAck:%d is higher then recived packet id:%d\n", s.localAck, packet.Id)
		return
	}
	s.RcvBuffer.Add(packet)
	packets, lastSeen := s.RcvBuffer.PopRange(s.localAck)

	fmt.Printf("lastSeenPacket: %d ,sending %d packets to LocalTcp\n", lastSeen, len(packets))

	s.writer <- toBytes(packets)

	s.localAck = lastSeen // updating the last sent packet to tcp

}

func toBytes(packets []UserPacket) []byte {
	ByteStream := make([]byte, len(packets)*255)
	for _, p := range packets {
		slice := p.Data[:]
		ByteStream = append(ByteStream, slice...)
	}
	return ByteStream
}

func (s *Session) NextPacketOrNoOp() (packet UserPacket) {

	packetToSend := s.FwdBuffer.GetNext(s.remoteAck, s.resendInterval*2) // todo manage resent interval

	//empty packet returned from the buffer
	if packetToSend.Id == 0 {
		packet = EmptyPacket(s.SessionId, s.localAck)
		//fmt.Printf("No data to send. Sending NO-OP packet. localAck id:%d\n", s.localAck)
	} else {
		packet = packetToSend
		packet.LastSeenPid = s.localAck
		fmt.Printf("sending packet id:%d . remote ack:%d\n", packet.Id, s.remoteAck)
	}

	return
}

func (s *Session) OnLocalClose(msg string) {

	if s.closed {
		return
	}

	closeMsg := "The remote connection was closed with error:" + msg

	if s.isClient {
		onCloseClient(s.SessionId, closeMsg, s.packetChannel)
	} else {
		onCloseServer(s.SessionId, closeMsg, s.FwdBuffer)
	}
}

func (s *Session) CloseSession() {
	s.closed = true
	s.localTcp.Close()

}

func onCloseClient(id uint8, msg string, channel chan UserPacket) {
	packet := ClosePacket(id, msg)
	channel <- packet

}

func onCloseServer(id uint8, msg string, buf *SendQueue) {
	packet := ClosePacket(id, msg)
	buf.Add(packet)

}


func (s *Session) handleServerAnswer(packet UserPacket) {

	// If packet is NO-OP, updating the last PID and dropping.
	if packet.Flags == NO_OP {
		if packet.LastSeenPid > s.remoteAck {
			fmt.Printf("Updating Remote ack from lastSeenPid. remoteAck:%d lastSeenPid:%d\n", s.remoteAck, packet.LastSeenPid)
			s.remoteAck = packet.LastSeenPid
		}
		return
	}

	if packet.Flags == CLOSE {
		fmt.Printf("The server closed the connection. msg:%v\n exiting progrem.\n", string(packet.Data))
		s.localTcp.Close()
		os.Exit(0)
	}

	if packet.Flags == CLOSED {
		fmt.Printf("The server  accepted the close request. exiting progrem.\n")
		s.localTcp.Close()
		os.Exit(0)

	}

	s.ToLocalTcp(packet)
}
