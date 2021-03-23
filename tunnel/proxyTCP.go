package tunnel

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"os"
)

type tcpCommunicator struct {
	conn           net.Conn
	port           string
	openConnection bool
}

const WindowSize = 120 // ~ 255/(8/5) -4 - hostname max length with the reduction from base32 and header bits
const MaxByteSize = 1024

// open a tcp listener and return the warped connection
func listenLocally(port string) (communicator *tcpCommunicator) {

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
	return &tcpCommunicator{
		conn:           conn,
		port:           port,
		openConnection: true,
	}
}

// dial to a tcp listener and return the warped connection

func dialLocally(port string) (communicator *tcpCommunicator) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Printf("error in dialTcp error:%v\n",
			err.Error())
		os.Exit(1)
	}

	return &tcpCommunicator{
		conn:           conn,
		port:           port,
		openConnection: true,
	}
}

func (c *tcpCommunicator) setReader(reader chan []byte, onClose func(msg string)) {
	defer c.conn.Close()

	buf := make([]byte, MaxByteSize)
	for {

		n, err := c.conn.Read(buf[0:])
		if err != nil {
			onClose(err.Error())
			c.openConnection = false
			fmt.Printf("error on local tcp reader: %v \n", err.Error())
			return
		}

		buf = bytes.Trim(buf[0:], "\x00")
		fmt.Printf("recived %d bytes from local TCP conn:%v \n", n, c.conn.LocalAddr().String())
		var i int
		for i <= n {
			edge := math.Min(float64(i+WindowSize), float64(n))
			reader <- buf[i:int(edge)]
			i = i + WindowSize
		}
		if c.openConnection == false {
			return
		}

	}
}
func (c *tcpCommunicator) setWriter(writer chan []byte, onClose func(msg string)) {

	defer c.conn.Close()

	for {
		var data = <-writer
		data = bytes.Trim(data[0:], "\x00")

		fmt.Printf("recived %d bytes to write to local TCP s.conn:%d \n", len(data), c.conn.LocalAddr())
		_, err := c.conn.Write(data)
		if err != nil {
			onClose(err.Error())
			c.openConnection = false
			fmt.Printf("error on local tcp write:%v\n", err.Error())
			return
		}

		if c.openConnection == false {
			return
		}
	}
}

func (c *tcpCommunicator) Close() {
	c.openConnection = false
	c.conn.Close()
}
