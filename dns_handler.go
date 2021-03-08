package tcpgodns

import (
	"encoding/base32"
	"fmt"
	"github.com/miekg/dns"
	"strings"
	"unsafe"
)

func (pm *PacketManager) HandleDnsClient() {

	for {
		var data = <-pm.packetChannel // Listen for local incoming packets.

		fmt.Printf("handleDnsClient: packetId:%d\n", data.Id)

		str := encode(data)
		var msg dns.Msg
		query := str + ".tcpgodns.com." // todo: change domain to custom
		msg.SetQuestion(query, dns.TypeTXT)

		fmt.Printf("Question:%v\n", msg)

		in, err := dns.Exchange(&msg, ":5553") // todo: create main parameters
		if in == nil || err != nil {
			fmt.Printf("Error with the exchange error:%s\n", err.Error())
			continue
		}
		if t, ok := in.Answer[0].(*dns.TXT); ok {

			userPacket := decode(t.Txt[0])
			fmt.Printf("recived userPacket id:%v via dns\n ", userPacket.Id)
			pm.ToLocalTcp(userPacket)
		}

	}
}

func (pm *PacketManager) ClientResendNoOp(){
	for {

		// check if need to resend packet according to ack
		// if needed create next packet according to ack

		fmt.Printf("Sending NO-OP packet for income data")
		noOpPacket:= EmptyPacket(pm.SessionId,pm.localAck)
		str := encode(noOpPacket)
		var msg dns.Msg

		query := str + ".tcpgodns.com." // todo: change domain to custom
		msg.SetQuestion(query, dns.TypeTXT)

		fmt.Printf("Question:%v\n", msg)

		in, err := dns.Exchange(&msg, ":5553") // todo: create main parameters
		if in == nil || err != nil {
			fmt.Printf("Error with the exchange error:%s\n", err.Error())
			continue
		}
		if t, ok := in.Answer[0].(*dns.TXT); ok {

			userPacket := decode(t.Txt[0])
			fmt.Printf("recived userPacket id:%v via dns\n ", userPacket.Id)
			pm.ToLocalTcp(userPacket)
		}

	}
}


func (pm *PacketManager) HandleDNSResponse(w dns.ResponseWriter, req *dns.Msg) {

	var resp dns.Msg
	resp.SetReply(req)
	resp.Response = true
	question := req.Question[0]
	fmt.Printf("recived dns query\n")
	i := strings.Index(question.Name, ".tcpgodns.com")
	encoded := question.Name[0:i]
	str, err := base32.HexEncoding.DecodeString(encoded)
	if err != nil {
		fmt.Println("Error while decoding query")
		return
	}
	userPacket := BytesToPacket(str)
	pm.ToLocalTcp(userPacket)

	packetToSend := pm.NextPacketToSend(userPacket.LastSeenPid)
	encodedToSend := encode(packetToSend)

	fmt.Printf("userPacket Parsed packetId:%d\n ", userPacket.Id)

	msg := []string{encodedToSend}
	fmt.Printf("size of array:%d size of answer:%d\n", uint16(unsafe.Sizeof(msg)), uint16(len(msg[0])))
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
		println("respAnswer:%v", resp.Answer)
	}
	w.WriteMsg(&resp)

}
