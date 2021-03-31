package tunnel

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
	serverErrorCount int
	errorRetries int
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
		serverErrorCount: 20,
		errorRetries: 20,
	}
}

// a go routine that listens for the packet channel of the provided session, parses the channel and sends the information to the dns server
func (d *DnsClient) HandleClient(packetChannel chan UserPacket, answerHandler func(serverPacket UserPacket), onClose func()) {

	for {
		var outPacket = <-packetChannel // HandleProxyTCP for local incoming packets.

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
			d.exitProgram(onClose, outPacket)

		}
		answerHandler(response)

	}
}

func (d *DnsClient) Dial() (ok bool, sessionId uint8, rtt int) {
	retries := d.options.dialRetries

	for retries != 0 {

		packet := connectPacket()

		responsePacket, err := d.dnsExchange(packet)
		if err != nil {
			fmt.Printf("Error durring connect phase.\n error:%v retries:%d\n", err.Error(), retries)
			retries--
			continue
		}
		if responsePacket.Flags == ESTABLISHED {
			fmt.Printf("Connected succefully to the server. sessionId:%d\n", responsePacket.SessionId)

			if rtt, ok := responsePacket.interval(); ok {
				fmt.Printf("Recived rtt from server succefully to the server. sessionId:%d\n", responsePacket.SessionId)
				return true, responsePacket.SessionId, rtt
			} else {

			}

		}

		time.Sleep(time.Millisecond * time.Duration(d.options.dialInterval))
	}
	return

}

// A go routine looping for keeping the connection alive and allowing the server to initiate data send by continuous empty querying the server
func (d *DnsClient) HandleResend(interval int,nextPacketHandler func() (packet UserPacket), answerHandler func(serverPacket UserPacket)) {
	for {
		time.Sleep(time.Millisecond * time.Duration(interval))

		packet := nextPacketHandler()

		responsePacket, err := d.dnsExchange(packet)
		if err != nil {
			continue
		}
		d.options.serverErrorCount = d.options.errorRetries
		answerHandler(responsePacket)
	}
}

// the internal dns exchange who parses the data provided by the session, encode it and pass it to the server.
func (d *DnsClient) dnsExchange(outPacket UserPacket) (inPacket UserPacket, err error) {
	str := encode(outPacket)
	var msg dns.Msg

	query := str + d.options.domain
	msg.SetQuestion(query, dns.TypeTXT)

	in, err := dns.Exchange(&msg, d.options.address+":"+d.options.port)

	if in == nil || err != nil {
		fmt.Printf("Error with the exchange error:%s\n", err.Error())
		d.options.serverErrorCount = d.options.serverErrorCount-1
		if d.options.serverErrorCount <=0{
			fmt.Printf("could not connect to server sevral times, exiting.")

			os.Exit(1)
		}
		return
	}
	if t, ok := in.Answer[0].(*dns.TXT); ok {

		inPacket = decode(t.Txt[0])
	}
	return
}

func (d *DnsClient) exitProgram(onClose func(), outPacket UserPacket) {
	fmt.Printf("Server did not recived the Connection closed message. sending one more packet and closing anyway\n")
	go d.dnsExchange(outPacket)
	time.Sleep(time.Millisecond * 30)

	fmt.Printf("Closing session and exiting progrem\n")

	onClose()
	os.Exit(0)
}
