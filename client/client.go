package main

import (
	"flag"
	"tcpgodns/tunnel"
)

type clientOptions struct {
	localPort     string
	remoteDnsPort string
	remoteAddress string
}

func main() {
	options := parseCommand()

	dns := tunnel.NewDnsClient(tunnel.DnsOptions{IsDefault: true})

	ok, sessionId, rtt := dns.Dial()
	if ok {

		session := tunnel.CreateSession(true, sessionId, rtt)
		session.ConnectProxy(options.localPort)
		RunDnsTunnel(session, dns)
		for {
		}
	}

}

func parseCommand() clientOptions {
	portPtr := flag.String("port", "9999", "The local port to listen to tcp. start with a dial from that port. ")
	addrPtr := flag.String("addr", "", "The DNS server adderss. for debugging purpose its default as local ")
	remotePtr := flag.String("dns", "5553", "The DNS server port. for debugging purpose its default as 5553 ")
	flag.Parse()

	options := clientOptions{
		localPort:     *portPtr,
		remoteDnsPort: *remotePtr,
		remoteAddress: *addrPtr,
	}
	return options
}

func RunDnsTunnel(session *tunnel.Session, dns *tunnel.DnsClient) {
	go dns.HandleClient(session.PacketChannel, session.HandleServerAnswer, session.CloseSession)
	go dns.HandleResend(session.ResendInterval, session.NextPacket, session.HandleServerAnswer)
}
