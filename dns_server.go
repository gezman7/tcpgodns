package tcpgodns

import (
	"fmt"
	"github.com/miekg/dns"
	"math"
	"strings"
)

func (sm *sessionManger) HandleDNSResponse(w dns.ResponseWriter, req *dns.Msg) {

	//	fmt.Printf("Recived dns query\n") //todo: to verbose log

	var resp dns.Msg
	resp.SetReply(req)
	resp.Response = true
	question := req.Question[0]
	i := strings.Index(question.Name, ".tcpgodns.com") // todo: parametrized constants
	encoded := question.Name[0:i]

	clientPacket := decode(encoded)

	var session *Session
	var serverPacket UserPacket

	if _, ok := sm.SessionMap[clientPacket.SessionId];
		!ok && clientPacket.Flags != CONNECT {
		fmt.Printf("sessionId:%d not in map. dismissing response\n")
		return
	}

	switch packetType := clientPacket.Flags; packetType {
	case CONNECT:

		newSessionId, _ := sm.EstablishNewSession()
		fmt.Printf("Recived connect request from client. establish session. sessionId:%d\n", newSessionId)

		serverPacket = clientPacket
		serverPacket.Flags = ESTABLISHED
		serverPacket.SessionId = newSessionId

		rtt, ok := clientPacket.GetInterval()
		var interval float64

		if ok == false {
			interval = MinInterval / 2
		} else {
			interval = math.Max(MinInterval, float64(rtt))
		}
		sm.SessionMap[newSessionId].resendInterval = int(interval * 2)

	case DATA:

		fmt.Printf("Recived packetId:%d from sessionId:%d\n", clientPacket.Id, clientPacket.SessionId)
		session = sm.SessionMap[clientPacket.SessionId]

		if clientPacket.LastSeenPid > session.remoteAck {
			fmt.Printf("Updating Remote ack of sessionId:%d from lastSeenPid. remoteAck:%d lastSeenPid:%d\n", session.SessionId, session.remoteAck, clientPacket.LastSeenPid)

			session.remoteAck = clientPacket.LastSeenPid
		}

		session.ToLocalTcp(clientPacket)
		serverPacket = session.NextPacketOrNoOp()
	case NO_OP:
		session = sm.SessionMap[clientPacket.SessionId]
		if clientPacket.LastSeenPid > session.remoteAck {
			fmt.Printf("Updating Remote ack of sessionId:%d from lastSeenPid. remoteAck:%d lastSeenPid:%d\n", session.SessionId, session.remoteAck, clientPacket.LastSeenPid)

			session.remoteAck = clientPacket.LastSeenPid
		}
		serverPacket = session.NextPacketOrNoOp()
	case CLOSE:
		session = sm.SessionMap[clientPacket.SessionId]

		fmt.Printf("Recived a request to close connection from sessionId:%d. closing connection and session\n", session.SessionId)
		session.CloseSession()
		delete(sm.SessionMap, clientPacket.SessionId)
		 serverPacket = UserPacket{
			 Flags: CLOSED,
		 }
	}

	encodedToSend := encode(serverPacket)

	msg := []string{encodedToSend}
	for _, q := range req.Question {
		a := dns.TXT{
			Hdr: dns.RR_Header{
				Name:     q.Name,
				Rrtype:   dns.TypeTXT,
				Class:    dns.ClassINET,
				Ttl:      0,
				Rdlength: uint16(1),
			},
			Txt: msg,
		}
		resp.Answer = append(resp.Answer, &a)
	}
	w.WriteMsg(&resp)
}
