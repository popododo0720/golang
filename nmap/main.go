package main

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

func scanPort(ipAddress string, port int, wg *sync.WaitGroup) {
	defer wg.Done()
	address := ipAddress + ":" + strconv.Itoa(port)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	defer conn.Close()

	fmt.Printf("Port %d is open\n", port)
}

func main() {
	var wg sync.WaitGroup
	targetIp := "192.168.0.40"
	startPort := 1
	lastPort := 65535

	for port := startPort; port <= lastPort; port++ {
		wg.Add(1)
		go scanPort(targetIp, port, &wg)
	}

	wg.Wait()
}
