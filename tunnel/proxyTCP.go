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
	isClient       bool
}

const ServerWindowSize float64 = 120 // ~ 255/(8/5) -4 - TXT max length with the reduction from base32 and header bits
const ClientWindowSize float64 = 23  // ~ 63/(8/5) -4  -12 bytes- hostname max length with the reduction from base32 and header bits

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
		isClient:       true,
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
		isClient:       false,
	}
}

func (c *tcpCommunicator) setReader(reader chan []byte, onClose func(msg string)) {
	defer c.conn.Close()

	buf := make([]byte, MaxByteSize)
	for {

		n, err := c.conn.Read(buf)
		if err != nil {
			onClose(err.Error())
			c.openConnection = false
			fmt.Printf("error on local tcp reader: %v \n", err.Error())
			return
		}
		windowSize := ClientWindowSize

		if !c.isClient {
			windowSize = ServerWindowSize
		}
		fmt.Printf("recived %d bytes from local TCP conn:%v \n", n, c.conn.LocalAddr().String())
		for alreadySent := 0; alreadySent < n; {
			restToSend := n - alreadySent
			sizeToSend := math.Min(windowSize, float64(restToSend))
			sendBuffer := make([]byte, int(sizeToSend))

			copy(sendBuffer, buf[alreadySent:alreadySent+int(sizeToSend)])
			reader <- sendBuffer
			alreadySent += int(sizeToSend)
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
