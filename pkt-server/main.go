package main

import (
	"bufio"
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

	reader := bufio.NewReader(conn)

	for {
		buf, err := reader.ReadBytes('}')
		if err != nil {
			if err == io.EOF {
				log.Printf("Connection closed by client %s (EOF)", conn.RemoteAddr())
			} else {
				log.Printf("Connection read error for %s: %v", conn.RemoteAddr(), err)
			}
			return
		}
		reader.Reset(conn)

		var req structs.CurrencyRequest
		if err := json.Unmarshal(buf, &req); err != nil {
			log.Printf("Failed to unmarshal request from %s: %v (data: %q)", conn.RemoteAddr(), err, string(buf))

			cerr, jerr := json.Marshal(&structs.CurrencyError{Error: err.Error()})
			if jerr != nil {
				log.Println("failed to marshar CurrencyError: ", jerr)
				continue
			}

			if _, werr := conn.Write(cerr); werr != nil {
				log.Println("failed to write to CurrencyError: ", werr)
				continue
			}
			continue
		}

		if req.Get == quitCommand {
			log.Printf("Client %s requested quit.", conn.RemoteAddr())
			return
		}

		log.Printf("Received request from %s: %+v", conn.RemoteAddr(), req)

		result := structs.Find(currencies, req.Get)

		rsp, err := json.Marshal(&result)
		if err != nil {
			log.Println("failed to marshal data: ", err)
			if _, err := fmt.Fprintf(conn, `{"currency_error":"internal error"}`); err != nil {
				log.Printf("failed to write to client: %v", err)
				return
			}
			continue
		}

		if _, err := conn.Write(rsp); err != nil {
			log.Println("failed to write response: ", err)
			return
		}
	}
}
