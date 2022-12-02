package websocket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"log"
	"net"
	"net/http"
	"time"
	"zn/hub"
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

type server struct {
	port       int
	logger     *zap.SugaredLogger
	httpServer *http.Server
}

func NewServer(
	hub *hub.Hub,
	port int,
	logger *zap.SugaredLogger,
) *server {

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(context.Background(), logger, hub, w, r)
	})
	return &server{
		port: port,
		httpServer: &http.Server{
			Handler: mux,
		},
		logger: logger,
	}
}

func (s *server) Start() error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		s.logger.Errorf("create net listener failed: %w", err)
		return err
	}

	go func() {
		port := listener.Addr().(*net.TCPAddr).Port
		s.logger.Infof("start http server on port %d", port)
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatalf("serve http server failed: %v", err)
		}
		s.logger.Infof("http server stopped")
	}()

	return nil
}

func (s *server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.httpServer.SetKeepAlivesEnabled(false)
	if err := s.httpServer.Shutdown(ctx); err != nil {

		s.logger.Errorf("private http privateServer shutdown failed: %w", err)
		return err
	}

	return nil
}

// serveWs handles websocket requests from the peer.
func serveWs(ctx context.Context, logger *zap.SugaredLogger, hubService *hub.Hub, w http.ResponseWriter, r *http.Request) {
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

	send := make(chan []byte, 256)
	newClient := hub.NewClient(logger, playerName, hubService, send, conn.RemoteAddr().String())
	newClient.Start()

	clientContext, cancel := context.WithCancel(ctx)
	go readPump(clientContext, newClient, conn, cancel)
	go writePump(clientContext, newClient, conn, cancel, send)
}

// рутина для чтения входящих сообщений
func readPump(ctx context.Context, newClient *hub.Client, conn *websocket.Conn, cancel context.CancelFunc) {

	defer func() {

		cancel()
		log.Println("Close write goroutine")
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetPongHandler(func(string) error { _ = conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {

		log.Print("Read message")
		err := conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {

			log.Printf("error: %v", err)
			cancel()
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {

			log.Printf("error: %v", err)
			cancel()
			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		newClient.HandleInput(string(message))
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// рутина для отправки сообщения
func writePump(ctx context.Context, newClient *hub.Client, conn *websocket.Conn, cancel context.CancelFunc, send chan []byte) {

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		newClient.Close()
		close(send)
		conn.Close()
		cancel()
		log.Println("Close write goroutine")
	}()
	for {
		select {
		case message, ok := <-send:

			if !ok {
				log.Print("Websocket publication stopped by producer")
				return
			}
			err := conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Print("Could not set write deadline on websocket connection")
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Print("Could not write message over websocket")
				return
			}
		case <-ticker.C:

			err := conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Print("Could not set write deadline on websocket connection")
			}
			err = conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
