package tunnel

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"sync/atomic"
)

type Session struct {
	SessionId      uint8           // id of the session
	localAck       uint16          // the last packet received in local machine
	remoteAck      uint16          // the last packet received in remote machine
	RcvBuffer      *rcvBuffer      // the received packets buffer.
	FwdBuffer      *sendBuffer     // the sent packet buffer.
	PidCounter     int32           // the counter for creating packet ID
	writer         chan []byte     // a channel to write to the TCP proxy
	reader         chan []byte     // a channel to read from the TCP proxy
	PacketChannel  chan UserPacket // a channel to pass userPackets from the session management to the dns client.
	isClient       bool            // indicator weather the session belongs to the client or server.
	ResendInterval int             // the interval for the resend from the sendBuffer and no-op packets.
	closed         bool            // flag to indicate weather the connection is closed, to prevent miss use and panic.
	proxyTCP       *tcpCommunicator  // the connection wrapper with the local TCP
}

const MinInterval = 10 // minimum interval for POC purpose. will override if the RTT is bigger.

//CreateSession creating a new session with all it's channels and Data Structures.
func CreateSession(isClient bool, sessionId uint8, rtt int) (session *Session) {
	cr := make(chan []byte)
	cw := make(chan []byte)
	packetChannel := make(chan UserPacket)

	resendInterval := math.Max(MinInterval, float64(rtt))

	session = &Session{
		SessionId:      sessionId,
		localAck:       0, // Id of the // last Packet received in local machine. - TO SEND TO REMOTE
		remoteAck:      0, // Id of the last packet ACKed from Remote - TO PROCESS LOCALLY
		RcvBuffer:      newRcvBuffer(),
		FwdBuffer:      newSendBuffer(),
		PidCounter:     0,
		reader:         cr,
		writer:         cw,
		PacketChannel:  packetChannel,
		isClient:       isClient,
		ResendInterval: int(resendInterval),
		closed:         false,
	}

	return session

}

//ConnectProxy connecting the session to the TCP proxy sockets using the session channel and initiating the async go routines to handle the TCP connection
func (s *Session) ConnectProxy(port string) {
	var proxy *tcpCommunicator

	if s.isClient {
		proxy = listenLocally(port)
	} else {
		proxy = dialLocally(port)
	}
	s.proxyTCP = proxy

	go proxy.setReader(s.reader, s.onLocalClose)
	go proxy.setWriter(s.writer, s.onLocalClose)
	go s.handleProxyRead()

}

//handleProxyRead read byte stream in a constant loop from the proxy tcp, parse it to userPacket and send it to the dns handler.
func (s *Session) handleProxyRead() {

	for {
		// received byte stream from the local TCP conn
		var byteStream = <-s.reader

		// close goroutine with a "close" key word
		if bytes.Equal(byteStream, []byte("close")) {
			return
		}

		packetId := uint16(atomic.AddInt32(&s.PidCounter, 1))
userPacket:= dataPacket(packetId,s.SessionId,s.localAck,byteStream)
		fmt.Printf("HandleProxylTCP: %d bytes warpped in user Packet id:%d\n", len(byteStream), userPacket.Id)

		if s.isClient {
			s.PacketChannel <- userPacket // send packet to dns client
		}

		s.FwdBuffer.add(userPacket, s.isClient) // add packet to the send buffer with client flag to indicates the packet has been sent.

	}
}

// Acknowledging unordered received packet, adding it to the received buffer,
//and popping all packets in the correct order according to the last read packet.
func (s *Session) handleProxyWrite(packet UserPacket) {

	if packet.Id <= s.localAck {

		fmt.Printf("localAck:%d is higher then recived packet id:%d\n", s.localAck, packet.Id)
		return
	}
	s.RcvBuffer.add(packet)
	packets, lastSeen := s.RcvBuffer.popRange(s.localAck)

	fmt.Printf("lastSeenPacket: %d ,sending %d packets to LocalTcp\n", lastSeen, len(packets))

	s.writer <- toBytes(packets)

	s.localAck = lastSeen // updating the last sent packet to tcp

}
// pull from the buffer the next packet or in case of no new packet, build and send NO_OP packet.
func (s *Session) NextPacket() (packet UserPacket) {

	packetToSend := s.FwdBuffer.next(s.remoteAck, s.ResendInterval*2)

	//empty packet returned from the buffer
	if packetToSend.Id == 0 {
		packet = noOpPacket(s.SessionId, s.localAck)
		//fmt.Printf("No data to send. Sending NO-OP packet. localAck id:%d\n", s.localAck)
	} else {
		packet = packetToSend
		packet.LastSeenPid = s.localAck
		fmt.Printf("sending packet id:%d . remote ack:%d\n", packet.Id, s.remoteAck)
	}

	return
}

func (s *Session) onLocalClose(msg string) {

	if s.closed {
		return
	}

	closeMsg := "The remote connection was closed with error:" + msg

	if s.isClient {
		onCloseClient(s.SessionId, closeMsg, s.PacketChannel)
	} else {
		onCloseServer(s.SessionId, closeMsg, s.FwdBuffer)
	}
}

func (s *Session) CloseSession() {
	s.closed = true
	s.proxyTCP.Close()

}

func onCloseClient(id uint8, msg string, channel chan UserPacket) {
	packet := closePacket(id, msg)
	channel <- packet

}

func onCloseServer(id uint8, msg string, buf *sendBuffer) {
	packet := closePacket(id, msg)
	buf.add(packet, false)

}

//HandleServerAnswer is a handler for an incoming packet from the server. This method is only invoked in a client session.
func (s *Session) HandleServerAnswer(packet UserPacket) {

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
		s.proxyTCP.Close()
		os.Exit(0)
	}

	if packet.Flags == CLOSED {
		fmt.Printf("The server  accepted the close request. exiting progrem.\n")
		s.proxyTCP.Close()
		os.Exit(0)

	}

	// Packet is with data, passing it the proxy TCP
	s.handleProxyWrite(packet)
}

func toBytes(packets []UserPacket) []byte {
	var ByteStream []byte
	for _, p := range packets {
		slice := p.Data[:]
		ByteStream = append(ByteStream, slice...)
	}
	return ByteStream
}
