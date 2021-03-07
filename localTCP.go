package tcpgodns

import (
	"fmt"
	"net"
	"os"
)

func ConnectLocally(cr chan []byte, cw chan []byte,port string) {
	conn := listenLocally(port)

	go handleWrite(conn, cw)
	go handleRead(conn, cr)

}

func handleRead(conn net.Conn, cr chan []byte) {
	defer conn.Close()

	buf := make([]byte, 31) //todo: add constant
	for {
		// read up to 130 bytes
		n, err := conn.Read(buf[0:])
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("recived %d byte to read from local TCP conn:%v forwarding to DNS query\n", n, conn.LocalAddr().String())
		cr <- buf
	}
}
func handleWrite(conn net.Conn, cw chan []byte) {
	defer conn.Close()

	for {
		var data = <-cw
		fmt.Printf("recived %d bytes to write to local TCP conn:%d\n", len(data), conn.LocalAddr())
		conn.Write(data)
	}
}



func listenLocally(port string) net.Conn {

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return conn
}