package main

import (
	"fmt"
	tgd "tcpgodns"
)

func main() {


	cr := make(chan []byte)
	cw := make(chan []byte)
	mf := tgd.ManagerFactory(cr, cw)
	go mf.HandleDnsClient()
	fmt.Println(mf)
	go tgd.ConnectLocally(cr, cw,"9999")
	go mf.LocalStreamIncome()
	for {
	}
}
