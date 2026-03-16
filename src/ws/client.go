package ws

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	Server *Server
	Ip     string
	Port   string
	Conn   *websocket.Conn
	Send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func NewClient(server *Server, w http.ResponseWriter, r *http.Request) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't upgrade connection to websocket: %w", err)
	}

	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		Server: server,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Ip:     host,
		Port:   port,
		ctx:    ctx,
		cancel: cancel,
	}

	go func() { server.Connect <- client }()

	go client.read()
	go client.write()

	return client, nil
}

func (client *Client) ToString() string {
	return fmt.Sprintf("Client: {ip: %s", client.Ip)
}

// Close shuts everything down cleanly
func (client *Client) Close() {
	client.once.Do(func() {
		client.cancel()
		close(client.Send)
		client.Conn.Close()
		client.Server.Disconnect <- client
	})
}

func (client *Client) read() {
	defer client.Close()

	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error { client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.ReplaceAll(message, []byte{'\n'}, []byte{' '}))
		log.Printf("New Message From client: %s", string(message))
	}
}

func (client *Client) write() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()
	defer client.Close()

	for {
		select {
		case <-client.ctx.Done():
			// Send a clean close frame to the peer
			client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			client.Conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case message, ok := <-client.Send:
			// Set timeout for next write
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Close connection
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Open writer
			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Write message to websocket
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			// Close writer
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Set timeout for next write
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			// Ping client
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
