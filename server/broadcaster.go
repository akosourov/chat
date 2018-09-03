package main

import "log"

// Broadcaster listen chat events and send messages to participants
type Broadcaster struct {
	entering chan Client
	leaving  chan Client
	messages chan Message
}

// Client represents client with unique name and net address and channel
// for communicate with the broadcaster
type Client struct {
	name string
	addr string
	ch   chan Message
}

// Message is a base data for communicate between client and broadcaster
type Message struct {
	from     Client
	text     string
	internal bool
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		entering: make(chan Client),
		leaving:  make(chan Client),
		messages: make(chan Message),
	}
}

// Listen starts listen chat events
func (b *Broadcaster) Listen() {
	clients := make(map[string]map[string]Client)
	for {
		select {
		case msg := <-b.messages:
			// send incoming message to all clients except sender
			log.Printf("[DEBUG] Got incoming message %v\n", msg)
			for name, clis := range clients {
				if name != msg.from.name {
					for _, c := range clis {
						c.ch <- Message{from: msg.from, text: msg.text}
					}
				}
			}
		case cli := <-b.entering:
			log.Printf("[DEBUG] Got entering client %v\n", cli)

			// add new client
			_, ok := clients[cli.name]
			if !ok {
				clients[cli.name] = map[string]Client{cli.addr: cli}
			} else {
				clients[cli.name][cli.addr] = cli
			}

			// send others that client was connected
			text := cli.name + " is online"
			for name, clis := range clients {
				if name != cli.name && !ok {
					for _, c := range clis {
						c.ch <- Message{from: cli, text: text, internal: true}
					}
				}
			}
		case cli := <-b.leaving:
			log.Printf("[DEBUG] Got leaving client %v\n", cli)

			// delete from clients
			clis, ok := clients[cli.name]
			single := true
			if ok {
				if len(clis) == 1 {
					delete(clients, cli.name)
				} else {
					single = false
					delete(clis, cli.addr)
				}
			} else {
				log.Printf("[WARN] Client %v must be in clients map\n", cli)
			}

			// send others that client was disconnected
			text := cli.name + " is offline"
			for name, clis := range clients {
				if name != cli.name && single {
					for _, c := range clis {
						c.ch <- Message{from: cli, text: text, internal: true}
					}
				}
			}
		}
	}
}
