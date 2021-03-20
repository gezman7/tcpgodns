package main

import (
	"flag"
	tgd "tcpgodns"
)


type clientOptions struct {
	localPort     string
	remoteDnsPort string
	remoteAddress string
}

func main() {
	portPtr := flag.String("port", "9999", "The local port to listen to tcp. start with a dial from that port. ")
	flag.Parse()

	options := clientOptions{
		localPort:     *portPtr,
		remoteDnsPort: "5553",
		remoteAddress: "",
	}

	tcpCommunicator := tgd.ListenLocally(options.localPort)
	dns := tgd.NewDnsClient(tgd.DnsOptions{IsDefault: true})

	ok, sessionId, rtt := dns.DialToDns()
	if ok {

		session := tgd.EstablishSession(true, sessionId, rtt)
		session.ConnectLocal(tcpCommunicator)
		RunDnsTunnel(session, dns)

		for {
		}

	}

}

func RunDnsTunnel(session *tgd.Session, dns *tgd.DnsClient) {
	go dns.HandleDnsClient(session)
	go dns.HandleResendOrNoOp(session)

}
