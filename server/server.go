package main

import (
	"flag"
	"github.com/miekg/dns"
	"tcpgodns/tunnel"
)

// The Server executable program.
func main() {
	port, dnsPort, domainName := parseCommand()

	manager := tunnel.DnsServer(port, domainName)

	dns.HandleFunc(".", manager.HandleDNSResponse)
	dns.ListenAndServe(":"+dnsPort, "udp", nil)
}

func parseCommand() (string, string, string) {
	portPtr := flag.String("port", "9998", "The local port to forward the inbound TCP data -MUST BE LISTENED")
	dnsPtr := flag.String("dns", "5553", "The listening port for incoming dns messages")
	domainPtr := flag.String("domain", "tcpgodns.com", "The domain to parse out of queries")
	flag.Parse()
	return *portPtr, *dnsPtr, *domainPtr
}
