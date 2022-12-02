package client

import (
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {

	// клиенты
	clients map[*Client]bool

	mu           sync.RWMutex
	clientByName map[string]*Client

	// клиенты
	muLobby sync.RWMutex
	lobbys  map[string]*lobby

	// канал для рассылки сообщения всем
	broadcast chan []byte

	// Регистируем клиентов
	register chan *Client

	// Удаляем клиентов
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		clients:      make(map[*Client]bool),
		clientByName: make(map[string]*Client),
		lobbys:       make(map[string]*lobby),
	}
}

func (h *Hub) Run() {

	for {

		select {
		case client := <-h.register:
			h.clients[client] = true

			h.mu.Lock()
			h.clientByName[client.Name] = client
			h.mu.Unlock()

			log.Printf("Add new Client %s", client.Name)
		case client := <-h.unregister:

			if client.currentLobby != nil && client.currentLobby.owner == client {
				client.currentLobby.destroyLobby <- true
			}
			h.mu.Lock()
			delete(h.clientByName, client.Name)
			h.mu.Unlock()
			delete(h.clients, client)
			log.Printf("delete Client %s", client.Name)
		case message := <-h.broadcast:

			// перебираем всех клиентов
			for client := range h.clients {
				select {
				case client.send <- message:
				default:

					if client.currentLobby != nil && client.currentLobby.owner == client {
						client.currentLobby.destroyLobby <- true
					}
					close(client.send)
					h.mu.Lock()
					delete(h.clientByName, client.Name)
					h.mu.Unlock()
					delete(h.clients, client)
				}
			}
		}
	}
}
