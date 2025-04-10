package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"main/structs"
	"net"
	"os"
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
		fmt.Println("unsupported network protocol")
		os.Exit(1)
	}

	ln, err := net.Listen(network, addr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
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

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req structs.CurrencyRequest
		if err := dec.Decode(&req); err != nil {
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
				enc := json.NewEncoder(conn)
				if encerr := enc.Encode(&structs.CurrencyError{Error: err.Error()}); encerr != nil {
					fmt.Printf("failed error encoding client %s err:", encerr)
					return
				}
				continue
			}

		}

		log.Printf("Received request from %s: %+v", conn.RemoteAddr(), req)

		if req.Get == quitCommand {
			return
		}

		result := structs.Find(currencies, req.Get)

		if err := enc.Encode(&result); err != nil {
			switch err := err.(type) {
			case net.Error:
				fmt.Println("failed to send response:", err)
				return
			default:
				if encerr := enc.Encode(&structs.CurrencyError{Error: err.Error()}); encerr != nil {
					fmt.Println("failed to send error:", encerr)
					return
				}
				continue
			}
		}

		if err := conn.SetDeadline(time.Now().Add(time.Second * 90)); err != nil {
			fmt.Println("failed to set deadline:", err)
			return
		}
	}
}
