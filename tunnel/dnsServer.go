package tunnel

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"math"
	"math/rand"
	"strings"
	"time"
)

type dnsServer struct {
	SessionMap map[uint8]*Session
	fwdPort    string
	domain     string
}

const MaxNum = 256

//DnsServer create the session manager for the server to manage incoming different clients
func DnsServer(port string, domain string) *dnsServer {

	sessionMap := make(map[uint8]*Session)

	return &dnsServer{

		fwdPort:    port,
		SessionMap: sessionMap,
		domain:     domain,
	}
}

//NewSession finds an open slot in the session map, and create a new session on that slot.
func (ds *dnsServer) NewSession() (openSlot uint8, err error) {

	openSlot, err = ds.findOpenSlot(uint8(rand.Intn(MaxNum)), 10)
	if err != nil {
		return
	}

	ds.SessionMap[openSlot] = CreateSession(false, openSlot, 0)
	ds.SessionMap[openSlot].ConnectProxy(ds.fwdPort)
	return
}

func (ds *dnsServer) findOpenSlot(id uint8, attempts int) (openSlot uint8, err error) {

	if attempts == 0 {
		err = errors.New("could not found an open slot in given attempts")
		return
	}
	rand.Seed(time.Now().UTC().UnixNano())
	if _, occupied := ds.SessionMap[id]; occupied {
		return ds.findOpenSlot(uint8(rand.Intn(MaxNum)), attempts-1)
	} else {
		openSlot = id
		fmt.Printf("assiging sessionId: %d\n", openSlot)
		return
	}

}

// A custom DNS server response handler to decode, parse and process the dns queries arrived from the DNS client
func (ds *dnsServer) HandleDNSResponse(w dns.ResponseWriter, req *dns.Msg) {

	var resp dns.Msg
	resp.SetReply(req)
	resp.Response = true

	if len(req.Question) == 0 || req.Question[0].Qtype != dns.TypeTXT {
		fmt.Printf("Unsupported query arrived, aborting. request:%v\n", req.String())
		return
	}

	clientPacket := extractPacket(req, ds.domain)

	if _, ok := ds.SessionMap[clientPacket.SessionId];
		!ok && clientPacket.Flags != CONNECT {
		fmt.Printf("sessionId:%d in clientPacket is not in the SessionMap. dismissing response\n", clientPacket.SessionId)
		return
	}

	var serverPacket UserPacket
	switch packetType := clientPacket.Flags; packetType {

	case CONNECT:
		serverPacket = ds.handleConnectPacket(clientPacket)
	case DATA:
		serverPacket = ds.handleDataPacket(clientPacket)
	case NO_OP:
		serverPacket = ds.handleNoOpPacket(clientPacket)
	case CLOSE:
		serverPacket = ds.handleClosePacket(clientPacket)
	}

	asResponse(serverPacket, req, &resp)
	w.WriteMsg(&resp)
}

func asResponse(serverPacket UserPacket, req *dns.Msg, resp *dns.Msg) {
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
}

func extractPacket(req *dns.Msg, domain string) UserPacket {

	question := req.Question[0]
	i := strings.Index(question.Name, "."+domain)
	encoded := question.Name[0:i]

	clientPacket := decode(encoded)
	return clientPacket
}

func (ds *dnsServer) handleConnectPacket(clientPacket UserPacket) (serverPacket UserPacket) {

	newSessionId, _ := ds.NewSession()

	fmt.Printf("Recived connect request from client. establish session. sessionId:%d\n", newSessionId)

	serverPacket = clientPacket
	serverPacket.Flags = ESTABLISHED
	serverPacket.SessionId = newSessionId

	rtt, ok := clientPacket.interval()
	var interval float64

	if ok == false {
		interval = MinInterval / 2
	} else {
		interval = math.Max(MinInterval, float64(rtt))
	}
	ds.SessionMap[newSessionId].ResendInterval = int(interval * 2)
	return serverPacket
}

func (ds *dnsServer) handleDataPacket(clientPacket UserPacket) (serverPacket UserPacket) {

	fmt.Printf("Recived packetId:%d from sessionId:%d\n", clientPacket.Id, clientPacket.SessionId)

	session := ds.SessionMap[clientPacket.SessionId]

	if clientPacket.LastSeenPid > session.remoteAck {

		fmt.Printf("Updating Remote ack of sessionId:%d from lastSeenPid. remoteAck:%d lastSeenPid:%d\n", session.SessionId, session.remoteAck, clientPacket.LastSeenPid)

		session.remoteAck = clientPacket.LastSeenPid
	}

	session.handleProxyWrite(clientPacket)
	serverPacket = session.NextPacket()
	return serverPacket
}

func (ds *dnsServer) handleNoOpPacket(clientPacket UserPacket) (serverPacket UserPacket) {

	session := ds.SessionMap[clientPacket.SessionId]
	if clientPacket.LastSeenPid > session.remoteAck {
		fmt.Printf("Updating Remote ack of sessionId:%d from lastSeenPid. remoteAck:%d lastSeenPid:%d\n", session.SessionId, session.remoteAck, clientPacket.LastSeenPid)

		session.remoteAck = clientPacket.LastSeenPid
	}
	serverPacket = session.NextPacket()
	return serverPacket
}

func (ds *dnsServer) handleClosePacket(clientPacket UserPacket) (serverPacket UserPacket) {
	session := ds.SessionMap[clientPacket.SessionId]

	fmt.Printf("Recived a request to close connection from sessionId:%d. closing connection and session\n", session.SessionId)
	session.CloseSession()
	delete(ds.SessionMap, clientPacket.SessionId)
	serverPacket = UserPacket{
		Flags: CLOSED,
	}
	return serverPacket
}
