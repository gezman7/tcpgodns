package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"tcpgodns"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "8.8.8.8:53")
	if err != nil {
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		os.Exit(1)
	}

	query := tcpgodns.Query("hello")
	fmt.Printf("query:%s\n", string(query))

	n, err := conn.Write(query)
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("number of bytes sent:%d\n", n)
	buf := make([]byte, 1024)
	n, err = conn.Read(buf[0:])
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(n)

	fmt.Println(string(buf[0:n]))
	var p tcpgodns.Parser
	if _, err := p.Start(buf); err != nil {
		panic(err)
	}
	wantName := "tcpgodns.com."

	for {
		q, err := p.Question()
		if err == tcpgodns.ErrSectionDone {
			break
		}
		if err != nil {
			panic(err)
		}

		if q.Name.String() != wantName {
			continue
		}

		fmt.Println("Found question for name", wantName)
		if err := p.SkipAllQuestions(); err != nil {
			panic(err)
		}
		break
	}

	var gotIPs []net.IP
	for {
		h, err := p.AnswerHeader()
		if err == tcpgodns.ErrSectionDone {
			break
		}
		if err != nil {
			panic(err)
		}

		if (h.Type != tcpgodns.TypeA && h.Type != tcpgodns.TypeAAAA) || h.Class != tcpgodns.ClassINET {
			continue
		}

		if !strings.EqualFold(h.Name.String(), wantName) {
			if err := p.SkipAnswer(); err != nil {
				panic(err)
			}
			continue
		}

		switch h.Type {
		case tcpgodns.TypeA:
			r, err := p.AResource()
			if err != nil {
				panic(err)
			}
			gotIPs = append(gotIPs, r.A[:])
		case tcpgodns.TypeAAAA:
			r, err := p.AAAAResource()
			if err != nil {
				panic(err)
			}
			gotIPs = append(gotIPs, r.AAAA[:])

		}
	}
	fmt.Printf(string(gotIPs[0]))
	os.Exit(0)
}
