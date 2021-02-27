package tcp

import (
	"fmt"
	"io"
	"net"
	"os"
)

func HandlePipeReader(conn net.Conn, pr *io.PipeReader) {
	defer conn.Close()

	var buf [31]byte
	for {
		n, err := pr.Read(buf[0:])
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("reader:%s\n", string(buf[:]))

		_, err = conn.Write(buf[0:n])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

	}

}
func HandlePipeWriter(conn net.Conn, pw *io.PipeWriter) {
	// close connection on exit
	defer conn.Close()

	var buf [31]byte
	for {
		// read up to 512 bytes
		n, err := conn.Read(buf[0:])
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("writer:%s\n", string(buf[:]))

		_, err = pw.Write(buf[0:n])
		if err != nil {
			return
		}

	}
}
func DialLocally(port string) *net.TCPConn {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return conn
}

func ListenLocally(port string) net.Conn {

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


func ForwardTraffic(conn net.Conn, pr *io.PipeReader) {
	defer conn.Close()

	var buf [31]byte
	counter :=0
	for {
		n, err := pr.Read(buf[0:])
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		counter++
		fmt.Printf("ForwardTraffic:%s\n", string(buf[:]))

		// Packet = Packet to data(counter)
		// encodedData = encoded the data(Packet)
		// query = create the DNS query(encodedData) OR DNS response
		// fire DNSRequest(query) OR RESPONSE

		_, err = conn.Write(buf[0:n])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

	}

}


