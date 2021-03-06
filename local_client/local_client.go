package main

import (
	"fmt"
	tgd "tcpgodns"
)

func main() {
	c := make(chan tgd.UserPacket)

	go packetPrinter(c)

	packetManager := tgd.ManagerFactory()
	go tcpToUserPacket(c, packetManager)

	for{}
}

func packetPrinter(c chan tgd.UserPacket) {

	for {
		var data = <-c
		fmt.Printf("packetPrinter: %v\n", data)

	}
}

func tcpToUserPacket(c chan tgd.UserPacket, packetManager *tgd.PacketManager) {
	conn := tgd.ListenLocally("9999")
	defer conn.Close()

	buf := make([]byte, 31) //todo: add constant
	for {
		// read up to 512 bytes
		n, err := conn.Read(buf[0:])
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("recived from local tcp connection %d bytes:%s\n", n, string(buf[:]))
		packetManager.ManageOutcome(buf, c)

		//resend := rand.Intn(int(idSent)) - 3// JUST FOR TEST -RESEND
		//if resend > 0 {
		//	packetManager.ManageResend(uint8(resend), c)
		//}

	}
}
