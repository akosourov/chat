package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
)

const end = '\n' // assume this is sygnal for end of message

func main() {
	conn, err := net.Dial("tcp", ":8000")
	if err != nil {
		log.Fatal("[WARN] Can't connect to server: ", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("[WARN] Can't close connection: ", err)
		}
	}()

	// read data from server and display it (to stdout)
	done1 := make(chan struct{})
	go func() {
		defer func() {
			done1 <- struct{}{}
		}()
		input := bufio.NewReader(conn)
		output := bufio.NewWriter(os.Stdout)

		for {
			// read message from server
			msg, err := input.ReadString(end)
			switch {
			case err == io.EOF:
				log.Println("[DEBUG] server closed connection")
				return
			case err != nil:
				log.Println("[WARN] Reading error from server: ", err)
				return
			}

			// write message to stdout
			output.WriteString(msg)
			if err := output.Flush(); err != nil {
				log.Println("[WARN] Can't write to server: ", err)
				return
			}
		}
	}()

	// read data from stdin and send it to server
	done2 := make(chan struct{})
	go func() {
		defer func() {
			done2 <- struct{}{}
		}()
		input := bufio.NewReader(os.Stdin)
		output := bufio.NewWriter(conn)

		for {
			// read msg from stdin
			msg, err := input.ReadString(end)
			switch {
			case err == io.EOF:
				log.Println("[DEBUG] Can't read from stdin: ", err)
				return
			case err != nil:
				log.Println("[WARN] Can't read from stdin: ", err)
				return
			}

			// send msg to server
			output.WriteString(msg)
			if err := output.Flush(); err != nil {
				log.Println("[WARN] Can't write to server: ", err)
				return
			}
		}
	}()

	// wait until one of goroutines is finished.
	select {
	case <-done1:
	case <-done2:
	}
}
