package main

import (
	"bufio"
	"errors"
	"log"
	"net"
	"strings"
)

const (
	end                 = '\n' // assume this is signal for end of message
	serverName          = "Server"
	maxAttemptEnterName = 5
)

// Server represents TCP Server
type Server struct {
	listener net.Listener
	addr     string
}

// NewServer creates new TCP server
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// Start runs server
func (s *Server) Start() {
	// Start server
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Fatal("[WARN] Can't create tcp server: ", err)
	}
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Fatal("[WARN] Can't close listener: ", err)
		}
	}()

	// run broadcaster
	broadcaster := NewBroadcaster()
	go broadcaster.Listen()

	// listen forever
	log.Println("[INFO] Start accept incoming connections on ", s.addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[WARN] Can't accept connection: ", err)
			continue
		}

		// handle client concurrenly
		go s.handleConnection(conn, broadcaster)
	}
}

func (s *Server) handleConnection(conn net.Conn, b *Broadcaster) {
	addr := conn.RemoteAddr().String()
	log.Println("[DEBUG] Handle connection ", addr)

	defer func() {
		log.Println("[DEBUG] Close connection ", addr)
		err := conn.Close()
		if err != nil {
			log.Println("[WARN] Can't close connection: ", err)
		}
	}()

	// prompt client name
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	name, err := s.nameHandshake(rw)
	if err != nil {
		log.Println("[INFO] Can't get name: ", err)
		return
	}

	// accept incoming messages from chat and write to client
	cliChan := make(chan Message)
	go func() {
		for msg := range cliChan {
			from := msg.from.name
			if msg.internal {
				from = serverName
			}
			err := s.sendMsg(rw, from, msg.text)
			if err != nil {
				log.Println("[WARN] Can't send msg from cliChan: ", err)
			}
		}
		log.Println("[DEBUG] Clichan was closed")
	}()

	cli := Client{name: name, ch: cliChan, addr: addr}
	b.entering <- cli

	// read incoming messages from client and forward to chat
	var msg string
	for err == nil {
		msg, err = s.readMsg(rw)
		log.Printf("[DEBUG] Got message %v. Forward it to broadcaster\n", msg)
		if err == nil {
			b.messages <- Message{from: cli, text: msg}
		}
	}
	if err != nil {
		log.Println("[DEBUG] Can't read from connection: ", err)
	}

	// notify to broadcaster that client was disconnected
	close(cliChan)
	b.leaving <- cli
}

func (s *Server) nameHandshake(rw *bufio.ReadWriter) (name string, err error) {
	err = s.sendMsg(rw, serverName, "Welcome to the chat! Enter your name")
	if err != nil {
		log.Println("[DEBUG] Can't write to connection: ", err)
		return
	}

	name, err = s.readMsg(rw)
	if err != nil {
		log.Println("[DEBUG] Can't read from connection: ", err)
		return
	}
	attempt := 0
	for name == "" {
		attempt++
		if attempt > maxAttemptEnterName {
			err = s.sendMsg(rw, serverName, "Sorry. Try later")
			if err == nil {
				err = errors.New("Max retry exhausted")
			}
			return
		}
		err = s.sendMsg(rw, serverName, "Invalid name. Enter name")
		if err != nil {
			log.Println("[DEBUG] Can't write to connection: ", err)
			return
		}
		name, err = s.readMsg(rw)
		if err != nil {
			log.Println("[DEBUG] Can't read from connection: ", err)
			return
		}
	}

	err = s.sendMsg(rw, serverName, name+", you are connected to the chat")
	if err != nil {
		log.Println("[DEBUG] Can't write to connection: ", err)
		return
	}
	return
}

func (s *Server) sendMsg(w *bufio.ReadWriter, from, msg string) error {
	w.WriteString(from + ": " + msg + string(end))
	return w.Flush()
}

func (s *Server) readMsg(w *bufio.ReadWriter) (msg string, err error) {
	msg, err = w.ReadString(end)
	msg = strings.Trim(msg, string(end))
	return
}

func main() {
	server := NewServer(":8000")
	server.Start()
}
