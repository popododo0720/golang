package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"main/structs"
	"net"
	"strings"
	"time"
)

const quitCommand = "__quit__"

var (
	currencies = structs.Load("data.csv")
)

func main() {
	var addr string
	var network string
	flag.StringVar(&addr, "e", ":4040", "service endpoint [ip addr or socket path]")
	flag.StringVar(&network, "n", "tcp", "network protocol [tcp,unix]")
	flag.Parse()

	switch network {
	case "tcp", "tcp4", "tcp6", "unix":
	default:
		log.Fatalln("unsupported network protocol:", network)
	}

	ln, err := net.Listen(network, addr)
	if err != nil {
		log.Fatalln("failed to create listener:", err)
	}
	defer ln.Close()

	log.Println("**** Glovbal Currency Service ****")
	log.Printf("Service started: (%s) %s\n", network, addr)

	acceptDelay := time.Millisecond * 10
	acceptCount := 0

	for {
		conn, err := ln.Accept()
		if err != nil {
			switch e := err.(type) {
			case net.Error:
				if e.Timeout() {
					if acceptCount > 5 {
						log.Printf("unable to connect after %d retries: %v", acceptCount, err)
						return
					}
					acceptDelay *= 2
					acceptCount++
					time.Sleep(acceptDelay)
					continue
				}
			default:
				log.Println(err)
				conn.Close()
				continue
			}
			acceptDelay = time.Millisecond * 10
			acceptCount = 0
		}
		log.Println("Connected to ", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		log.Printf("closing connection for %s", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			log.Println("error closing connection: ", err)
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(time.Second * 45)); err != nil {
		log.Println("failed to set deadline:", err)
		return
	}

	reader := bufio.NewReader(conn)

	for {
		cmdLine, err := reader.ReadString('\n')
		if err != nil {
			switch err := err.(type) {
			case net.Error:
				if err.Timeout() {
					fmt.Println("deadline reached, disconnecting...")
				}
				fmt.Println("network error: ", err)
				return
			default:
				if err == io.EOF {
					fmt.Printf("Connection closed by client %s (EOF)", conn.RemoteAddr())
					return
				}
				continue
			}
		}
		reader.Reset(conn)

		cmd, param := parseCommand(cmdLine)
		if cmd == "" {
			if _, err := fmt.Fprint(conn, "Invalid command\n"); err != nil {
				log.Println("failed to write:", err)
				return
			}
			continue
		}

		log.Printf("Received request from %s: %+v %+v", conn.RemoteAddr(), cmd, param)

		if param == quitCommand {
			return
		}

		switch strings.ToUpper(cmd) {
		case "GET":
			result := structs.Find(currencies, param)
			if len(result) == 0 {
				if _, err := fmt.Fprint(conn, "Nothing found\n"); err != nil {
					log.Println("failed to write:", err)
				}
				continue
			}

			for _, cur := range result {
				_, err := fmt.Fprintf(
					conn,
					"%s %s %s %s\n",
					cur.Name, cur.Code, cur.Number, cur.Country,
				)

				if err != nil {
					log.Println("failed to write response: ", err)
					return
				}
			}
		default:
			if _, err := fmt.Fprintf(conn, "Invalid command\n"); err != nil {
				log.Println("failed to write:", err)
				return
			}
		}
	}
}

func parseCommand(cmdLine string) (cmd, param string) {
	parts := strings.Split(cmdLine, " ")
	if len(parts) != 2 {
		return "", ""
	}
	cmd = strings.TrimSpace(parts[0])
	param = strings.TrimSpace(parts[1])
	return
}
