package main

import (
	"fmt"
	"net"
)

func serverWorker(clientIp net.IP) {
	fmt.Printf("Got connection from %v\n", clientIp.String())
}
