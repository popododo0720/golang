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
const dataPath = "data.csv"

type Server struct {
	network      string
	address      string
	listener     net.Listener
	currencies   []structs.Currency
	shutdownChan chan struct{}
}

func NewServer(network, address, dataPath string) (*Server, error) {
	currencies := structs.Load(dataPath)
	return &Server{
		network:      network,
		address:      address,
		currencies:   currencies,
		shutdownChan: make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	ln, err := net.Listen(s.network, s.address)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = ln
	defer s.listener.Close()

	log.Println("**** Glovbal Currency Service ****")
	log.Printf("Service started: (%s) %s\n", s.network, s.address)

	for {
		select {
		case <-s.shutdownChan:
			log.Println("Shutting down server...")
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Println("Accept error: ", err)
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				continue
			}
			log.Println("Connected to ", conn.RemoteAddr())
			handler := NewConnectionHandler(conn, s.currencies)
			go handler.Handle()
		}
	}
}

func (s *Server) Shutdown() {
	close(s.shutdownChan)
	if s.listener != nil {
		s.listener.Close()
	}
}

type ConnectionHandler struct {
	conn       net.Conn
	reader     *bufio.Reader
	currencies []structs.Currency
}

func NewConnectionHandler(conn net.Conn, currencies []structs.Currency) *ConnectionHandler {
	return &ConnectionHandler{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		currencies: currencies,
	}
}

func (h *ConnectionHandler) Handle() {
	defer func() {
		log.Printf("closing connection for %s", h.conn.RemoteAddr())
		if err := h.conn.Close(); err != nil {
			log.Println("error closing connection: ", err)
		}
	}()

	if err := h.conn.SetDeadline(time.Now().Add(time.Second * 45)); err != nil {
		log.Println("failed to set deadline:", err)
		return
	}

	for {
		cmdLine, err := h.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Connection closed by client %s (EOF)", h.conn.RemoteAddr())
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Connection timeout for %s", h.conn.RemoteAddr())
				return
			}
			log.Printf("Error reading from %s: %v", h.conn.RemoteAddr(), err)
			return
		}

		cmd, param := parseCommand(cmdLine)
		if cmd == "" {
			fmt.Fprint(h.conn, "Invalid command\n")
			continue
		}

		log.Printf("Received request from %s: %+v %+v", h.conn.RemoteAddr(), cmd, param)

		if param == quitCommand {
			return
		}

		switch strings.ToUpper(cmd) {
		case "GET":
			h.handleGet(param)
		default:
			fmt.Fprintf(h.conn, "Invalid command\n")
		}

		if err := h.conn.SetDeadline(time.Now().Add(time.Second * 45)); err != nil {
			log.Println("failed to set deadline:", err)
			return
		}
	}
}

func (h *ConnectionHandler) handleGet(param string) {
	result := structs.Find(h.currencies, param)
	if len(result) == 0 {
		fmt.Fprint(h.conn, "Nothing found\n")
		return
	}

	for _, cur := range result {
		_, err := fmt.Fprintf(
			h.conn,
			"%s %s %s %s\n",
			cur.Name, cur.Code, cur.Number, cur.Country,
		)

		if err != nil {
			log.Println("failed to write response: ", err)
			return
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

func main() {
	var addr string
	var network string
	flag.StringVar(&addr, "e", ":4040", "service endpoint [ip addr or socket path]")
	flag.StringVar(&network, "n", "tcp", "network protocol [tcp,unix]")
	flag.Parse()

	switch network {
	case "tcp", "tcp4", "tcp6", "unix":
	default:
		log.Fatalln("unsupported network protocol: ", network)
	}

	server, err := NewServer(network, addr, dataPath)
	if err != nil {
		log.Fatalln("failed to create server: ", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalln("server stopped with error: ", err)
	} else {
		log.Println("Server stopped gracefully.")
	}
}
