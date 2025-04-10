package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const prompt = "currency"
const quitCommand = "__quit__"

func main() {
	var addr string
	var network string
	flag.StringVar(&addr, "e", "localhost:4040", "service endpoint [ip addr or socket path]")
	flag.StringVar(&network, "n", "tcp", "network protocol [tcp,unix]")
	flag.Parse()

	dialer := &net.Dialer{
		Timeout:   time.Second * 30,
		KeepAlive: time.Minute * 5,
	}

	var (
		conn           net.Conn
		err            error
		connTries      = 0
		connMaxRetries = 3
		connSleepRetry = time.Second * 1
	)

	for connTries < connMaxRetries {
		fmt.Println("Attempting to connect to", addr)
		conn, err = dialer.Dial(network, addr)
		if err == nil {
			break
		}

		fmt.Println("failed to connect:", err)
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			connTries++
			if connTries >= connMaxRetries {
				fmt.Println("Max connection retries reached.")
				os.Exit(1)
			}
			fmt.Printf("Retrying connection in %v...\n", connSleepRetry)
			time.Sleep(connSleepRetry)
			connSleepRetry *= 2
		} else {
			fmt.Println("Unrecoverable error during dial:", err)
			os.Exit(1)
		}
	}
	defer conn.Close()

	fmt.Println("Connected to currency service:", addr)

	serverReader := bufio.NewReader(conn)

	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	for i := 0; i < 2; i++ {
		initialMsg, readErr := serverReader.ReadString('\n')
		if readErr != nil {
			if nerr, ok := readErr.(net.Error); ok && nerr.Timeout() {
				fmt.Println("Warning: Did not receive expected initial message(s) from server.")
				break
			}
			fmt.Println("Warning: Error reading initial server message:", readErr)
			break
		}
		fmt.Print("Server: ", initialMsg)
	}
	conn.SetReadDeadline(time.Time{})

	fmt.Println("Enter search string or 'quit' to exit")

	userInputReader := bufio.NewReader(os.Stdin)

	looping := true
	for looping {
		fmt.Print(prompt, "> ")
		userInput, _ := userInputReader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		var requestParam string

		switch userInput {
		case "q", "quit":
			fmt.Println("Exiting...")
			looping = false
			requestParam = quitCommand
		case "":
			continue
		default:
			requestParam = userInput
		}

		req := fmt.Sprintf("GET %s\n", requestParam)
		_, writeErr := conn.Write([]byte(req))
		if writeErr != nil {
			fmt.Println("failed to send request:", writeErr)
			if _, ok := writeErr.(net.Error); ok {
				looping = false
			}
			continue
		}

		if requestParam == quitCommand {
			break
		}

		fmt.Println("--- Server Response ---")
		readDeadline := time.Now().Add(time.Second * 2)
		conn.SetReadDeadline(readDeadline)

		responseReceived := false
		for {
			responseLine, readErr := serverReader.ReadString('\n')
			if readErr != nil {
				if nerr, ok := readErr.(net.Error); ok && nerr.Timeout() {
					if !responseReceived {
						fmt.Println("(No response received within timeout)")
					}
					break
				}
				fmt.Println("failed to read response:", readErr)
				looping = false
				break
			}
			fmt.Print(responseLine)
			responseReceived = true

			readDeadline = time.Now().Add(time.Millisecond * 500)
			conn.SetReadDeadline(readDeadline)
		}
		conn.SetReadDeadline(time.Time{})
		fmt.Println("--- End Response ---")

	}

	fmt.Println("Waiting for 1 second before closing...")
	time.Sleep(1 * time.Second)
	fmt.Println("Program finished")
}
