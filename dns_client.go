package tcpgodns

import (
	"fmt"
	"github.com/miekg/dns"
	"os"
	"time"
)

type DnsClient struct {
	options DnsOptions
}

type DnsOptions struct {
	dialRetries  int
	dialInterval int
	domain       string
	address      string
	port         string
	IsDefault    bool
}

//

func NewDnsClient(options DnsOptions) *DnsClient {
	if options.IsDefault {
		options = getDefaultOptions()
	}
	return &DnsClient{options: options}
}

func getDefaultOptions() DnsOptions {
	return DnsOptions{
		dialRetries:  3,
		dialInterval: 100,
		domain:       ".tcpgodns.com.",
		address:      "",
		port:         "5553",
		IsDefault:    true,
	}
}
func (d *DnsClient) HandleDnsClient(s *Session) {

	for {
		var outPacket = <-s.packetChannel // HandleLocalTCP for local incoming packets.

		if outPacket.Flags == CLOSE {
			fmt.Printf("local Connection closed. sending CLOSE message to server msg:%v\n", string(outPacket.Data))

		} else if outPacket.Flags != NO_OP {
			fmt.Printf("handleDnsClient - sending packetId:%d seesionId:%d\n", outPacket.Id, outPacket.SessionId)
		}

		response, err := d.dnsExchange(outPacket)
		if err != nil {
			fmt.Printf("Error on dnsExcange:%v\n", err.Error())

			continue
		}

		if outPacket.Flags == CLOSE && response.Flags != CLOSED {
			fmt.Printf("Server did not recived the Connection closed message. sending one more packet and closing anyway\n")
			go d.dnsExchange(outPacket)
			time.Sleep(time.Millisecond * 30)
			s.localTcp.conn.Close()
			os.Exit(0)

		}
		s.handleServerAnswer(response)

	}
}

func (d *DnsClient) DialToDns() (ok bool, sessionId uint8, rtt int) {
	retries := d.options.dialRetries

	for retries != 0 {

		packet := ConnectPacket()

		responsePacket, err := d.dnsExchange(packet)
		if err != nil {
			fmt.Printf("Error durring connect phase.\n error:%v retries:%d\n", err.Error(), retries)
			retries--
			continue
		}
		if responsePacket.Flags == ESTABLISHED {
			fmt.Printf("Connected succefully to the server. sessionId:%d\n", responsePacket.SessionId)

			if rtt, ok := responsePacket.GetInterval(); ok {
				fmt.Printf("Recived rtt from server succefully to the server. sessionId:%d\n", responsePacket.SessionId)
				return true, responsePacket.SessionId, rtt
			} else {

			}

		}

		time.Sleep(time.Millisecond * time.Duration(d.options.dialInterval))
	}
	return

}

func (d *DnsClient) HandleResendOrNoOp(s *Session) {
	for {
		time.Sleep(time.Millisecond * time.Duration(s.resendInterval))

		packet := s.NextPacketOrNoOp()

		responsePacket, err := d.dnsExchange(packet)
		if err != nil {
			continue
		}

		s.handleServerAnswer(responsePacket)
	}
}

func (d *DnsClient) dnsExchange(outPacket UserPacket) (inPacket UserPacket, err error) {
	str := encode(outPacket)
	var msg dns.Msg

	query := str + d.options.domain // todo: change domain to custom
	msg.SetQuestion(query, dns.TypeTXT)

	in, err := dns.Exchange(&msg, d.options.address+":"+d.options.port) // todo: create main parameters

	if in == nil || err != nil {
		fmt.Printf("Error with the exchange error:%s\n", err.Error())
		return
		//todo: create custom error for in==nil
	}
	if t, ok := in.Answer[0].(*dns.TXT); ok {

		inPacket = decode(t.Txt[0])
	}
	return
}

