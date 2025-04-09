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

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			conn.Close()
			continue
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

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req structs.CurrencyRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				log.Printf("Connection closed by client %s (EOF)", conn.RemoteAddr())
			} else {
				log.Printf("Failed to decode request from %s: %+v", conn.RemoteAddr(), err)
			}
			return
		}

		log.Printf("Received request from %s: %+v", conn.RemoteAddr(), req)

		if req.Get == quitCommand {
			return
		}

		result := structs.Find(currencies, req.Get)

		if err := enc.Encode(&result); err != nil {
			log.Printf("Failed to encode/send response to %s: %v", conn.RemoteAddr(), err)
			return
		}
	}
}
