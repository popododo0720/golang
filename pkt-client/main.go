package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"main/structs"
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
		Timeout:   time.Second * 300,
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
		fmt.Println("creating socket to", addr)
		conn, err = dialer.Dial(network, addr)
		if err != nil {
			fmt.Println("failed to create socket: ", err)
			switch nerr := err.(type) {
			case net.Error:
				if nerr.Timeout() {
					connTries++
					fmt.Println("trying again in: ", connSleepRetry)
					time.Sleep(connSleepRetry)
					continue
				}
				fmt.Println("unable to recover from network error:", nerr)
				os.Exit(1)
			default:
				fmt.Println("non-network error during dial:", err)
				os.Exit(1)
			}
		}
		break
	}

	if err != nil {
		fmt.Println("failed to create connection...")
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("connected to currency service: ", addr)
	fmt.Println("Enter search string or *")

	reader := bufio.NewReader(os.Stdin)

	var param string
	looping := true
	for looping {
		fmt.Print(prompt, "> ")
		param, _ = reader.ReadString('\n')
		param = strings.TrimSpace(param)

		switch param {
		case "q", "quit":
			fmt.Println("Exiting...")
			looping = false
			param = quitCommand
		case "":
			continue
		}

		req := structs.CurrencyRequest{Get: param}

		if err := json.NewEncoder(conn).Encode(&req); err != nil {
			switch err := err.(type) {
			case net.Error:
				fmt.Println("failed to send request: ", err)
				looping = false
			default:
				fmt.Println("failed to encode request: ", err)
			}
			continue
		}

		var currencies []structs.Currency
		if err = json.NewDecoder(conn).Decode(&currencies); err != nil {
			switch err := err.(type) {
			case net.Error:
				fmt.Println("failed to receive response: ", err)
				looping = false
			default:
				fmt.Println("failed to decode response: ", err)
			}
			continue
		}

		if len(currencies) == 0 {
			fmt.Println("No currencies found")
		} else {
			fmt.Println(currencies)
		}
	}

	fmt.Println("waiting for 1 second before closing ...")
	time.Sleep(1 * time.Second)
	fmt.Println("Program finished")
}
