package ws

import (
	"bytes"
	"context"
	"fmt"
	device "pi_dash/src/device/types"
	comps "pi_dash/src/views/components"
	"time"
)

type Server struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Connect    chan *Client
	Disconnect chan *Client
}

func NewServer() *Server {
	server := &Server{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte, 16),
		Connect:    make(chan *Client, 16),
		Disconnect: make(chan *Client, 16),
	}

	go server.start()

	return server
}

func (server *Server) start() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client, ok := <-server.Connect:
			if ok {
				server.Clients[client] = true
			}
		case client, ok := <-server.Disconnect:
			if ok {
				delete(server.Clients, client)
			}
		case message, ok := <-server.Broadcast:
			if ok {
				for client := range server.Clients {
					client.Send <- message
				}
			}
		case <-ticker.C:
			go server.broadcastDeviceData()

		}
	}
}

func (server *Server) broadcastDeviceData() {
	device_data, err := device.NewData()
	if err != nil {
		fmt.Printf("ERROR: retrieving device data: %v\n", err)
		return
	}
	var html bytes.Buffer
	if err := comps.DeviceData(device_data).Render(context.Background(), &html); err != nil {
		fmt.Printf("ERROR: rendering device data: %v\n", err)
		return
	}

	// Non-blocking send to Broadcast
	select {
	case server.Broadcast <- html.Bytes():
	default:
		fmt.Println("WARNING: broadcast channel full, skipping tick")
	}
}
