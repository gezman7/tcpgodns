package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	dt "tcpgodns/datatype"

)

const SpecialDomain = "tcpgodns"

func main() {
	var err = server()
	if err != nil {
		fmt.Println(err)
	}
}

func server() error {

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:5656")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	defer conn.Close()

	for {
		var m dt.DnsMessage

		buf := make([]byte, 512)
		_, addr, _ := conn.ReadFromUDP(buf) // get the return address

		// validate it's a dns message.
		err := m.Unpack(buf)
		if err != nil {
			return err
		}

		if len(m.Questions) == 0 {
			fmt.Println("no question in dns packet")
			os.Exit(1)
		}

		q := m.Questions[0]
		if q.Type != dt.TypeTXT {
			fmt.Println("not a txt question. resolve forward")
			//os.Exit(1)
		}

		domains := strings.Split(string(q.Name.Data[:]), ".")
		fmt.Printf("domains: %+v\n", domains)

		index, found := FindSpecialDomain(domains)
		if !found {
			fmt.Println("regular dns query - forwarding to 8.8.8.8")
		}
		if index == 0 {
			fmt.Println("empty message!")
			return nil
		}

		encodedMessage := GetEncodedMessage(domains, index)
		fmt.Printf("encodedMessage :\n %+v\n", encodedMessage)
		fmt.Printf("message received Q:\n %+v\n", m.Questions)
		fmt.Printf("address to return :\n %+v\n", addr)
	}

	return nil

}

func GetEncodedMessage(domains []string, index int) string {
	if index == 1 {
		return domains[0]
	}
	message := domains[:index]
	return strings.Join(message, ".")

}
func FindSpecialDomain(slice []string) (int, bool) {
	for i, item := range slice {
		if item == SpecialDomain {
			return i, true
		}
	}
	return -1, false
}
