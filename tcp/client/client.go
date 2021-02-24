package main

import (
	"io"
	"tcpgodns/tcp"
)

func main() {
	pr, pw := io.Pipe()

	listenConn:=tcp.ListenLocally("7778")

	go tcp.HandlePipeWriter(listenConn, pw)

	 dialConn := tcp.DialLocally("9999")

	 go tcp.HandlePipeReader(dialConn, pr)

	for {
	}
}
