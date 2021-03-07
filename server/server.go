package main

import (
	"github.com/miekg/dns"
	"tcpgodns"
)

func main() {

	cr := make(chan []byte,512)
	cw := make(chan []byte,512)
	manager := tcpgodns.ManagerFactory(cr, cw)
	go tcpgodns.ConnectLocally(cr, cw,"9998")

	dns.HandleFunc(".", manager.HandleDNSResponse)
	dns.ListenAndServe(":5553", "udp", nil)
}
