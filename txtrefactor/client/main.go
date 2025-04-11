package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const quitCommand = "__quit__"

type Client struct {
	network string
	address string
	conn    net.Conn
	reader  *bufio.Reader
	Dialer  *net.Dialer
}

func NewClient(network, address string) *Client {
	return &Client{
		network: network,
		address: address,
		Dialer: &net.Dialer{
			Timeout:   time.Second * 30,
			KeepAlive: time.Minute * 5,
		},
	}
}

func (c *Client) Connect() error {
	var err error
	connMaxRetries := 3
	connSleepRetry := time.Second * 1

	for connTries := 0; connTries < connMaxRetries; connTries++ {
		log.Println("Attempting to connect to ", c.address)
		c.conn, err = c.Dialer.Dial(c.network, c.address)
		if err == nil {
			c.reader = bufio.NewReader(c.conn)
			log.Println("Connected to currency service: ", c.address)
			return nil
		}

		log.Println("failed to connect: ", err)
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			if connTries >= connMaxRetries-1 {
				break
			}
			log.Printf("Retrying connection in %v...\n", connSleepRetry)
			time.Sleep(connSleepRetry)
			connSleepRetry *= 2
		} else {
			log.Println("Unrecoverable error during dial: ", err)
			return err
		}
	}
	return fmt.Errorf("max connection retries reached for %s", c.address)
}

func (c *Client) SendRequest(request string) error {
	req := fmt.Sprintf("GET %s\n", request)
	_, err := c.conn.Write([]byte(req))
	return err
}

func (c *Client) ReadResponse() {
	readDeadline := time.Now().Add(time.Second * 2)
	c.conn.SetReadDeadline(readDeadline)

	responseReceived := false
	for {
		responseLine, readErr := c.reader.ReadString('\n')
		if readErr != nil {
			if nerr, ok := readErr.(net.Error); ok && nerr.Timeout() {
				if !responseReceived {
					log.Println("No response received within timeout")
				}
				break
			}
			log.Println("failed to read response: ", readErr)
			break
		}
		log.Print(responseLine)
		responseReceived = true

		readDeadline = time.Now().Add(time.Millisecond * 500)
		c.conn.SetReadDeadline(readDeadline)
	}
	c.conn.SetReadDeadline(time.Time{})
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) RunInteractive() {
	userInputReader := bufio.NewReader(os.Stdin)
	looping := true
	for looping {
		fmt.Print("currency> ")
		userInput, _ := userInputReader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		switch userInput {
		case "q", "quit":
			log.Println("Exiting...")
			_ = c.SendRequest(quitCommand)
			looping = false
		case "":
			continue
		default:
			if err := c.SendRequest(userInput); err != nil {
				log.Println("failed to send request: ", err)
				if _, ok := err.(net.Error); ok {
					looping = false
				}
				continue
			}

			log.Println("--- Server Response ---")
			c.ReadResponse()
			log.Println("---- End Response ----")
		}
	}
}

func main() {
	var addr string
	var network string
	flag.StringVar(&addr, "e", "localhost:4040", "service endpoint [ip addr or socket path]")
	flag.StringVar(&network, "n", "tcp", "network protocol [tcp,unix]")
	flag.Parse()

	client := NewClient(network, addr)

	if err := client.Connect(); err != nil {
		log.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer client.Close()

	client.RunInteractive()

	log.Println("Waiting for 1 second before closing...")
	time.Sleep(1 * time.Second)
	log.Println("Program finished")
}
