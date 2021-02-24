package main

import (
	"io"
	"tcpgodns/tcp"
)

func main() {
	pr, pw := io.Pipe()

	conn:=tcp.ListenLocally("9999")

	go tcp.HandlePipeWriter(conn, pw)

	conn1:= tcp.DialLocally("7777")

	go tcp.HandlePipeReader(conn1, pr)

	for {
	}

}
