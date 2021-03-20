package main

import (
	"flag"
	"github.com/miekg/dns"
	"tcpgodns"
)

func main() {
	portPtr := flag.String("port", "4222", "The local port to forward the inbound TCP data -MUST BE LISTENED")

	manager := tcpgodns.NewSessionManger(*portPtr)

	dns.HandleFunc(".", manager.HandleDNSResponse)
	dns.ListenAndServe(":5553", "udp", nil)
}
