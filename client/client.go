package client

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 20 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 20 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 15 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Name         string
	hub          *Hub
	currentLobby *lobby

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// рутина для чтения входящих сообщений
func (c *Client) readPump(ctx context.Context, cancel context.CancelFunc) {

	defer func() {

		cancel()
		log.Println("Close write goroutine")
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {

		log.Print("Read message")
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {

			log.Printf("error: %v", err)
			cancel()
			return
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {

			log.Printf("error: %v", err)
			cancel()
			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.handleInput(string(message))
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// рутина для отправки сообщения
func (c *Client) writePump(ctx context.Context, cancel context.CancelFunc) {

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		close(c.send)
		c.hub.unregister <- c
		c.conn.Close()
		cancel()
		log.Println("Close write goroutine")
	}()
	for {
		select {
		case message, ok := <-c.send:

			if !ok {
				log.Print("Websocket publication stopped by producer")
				return
			}
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Print("Could not set write deadline on websocket connection")
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Print("Could not write message over websocket")
				return
			}
		case <-ticker.C:

			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Print("Could not set write deadline on websocket connection")
			}
			err = c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// serveWs handles websocket requests from the peer.
func ServeWs(ctx context.Context, hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	playerName := r.Header.Get("PlayerName")
	if len(playerName) < 1 {

		log.Println("Player name not found in headers")
		return
	}
	client := &Client{Name: playerName, hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	clientContext, cancel := context.WithCancel(ctx)
	go client.writePump(clientContext, cancel)
	go client.readPump(clientContext, cancel)
}
