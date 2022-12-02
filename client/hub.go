package client

import (
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// клиенты
	clients map[*Client]bool

	clientByName map[string]*Client

	// клиенты
	lobbys map[string]*lobby

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
			h.clientByName[client.Name] = client

			log.Printf("Add new Client %s", client.Name)
		case client := <-h.unregister:

			log.Printf("Start unregister Client %s", client.Name)
			if client.currentLobby != nil && client.currentLobby.owner == client {
				client.currentLobby.destroyLobby <- true
			}
			log.Printf("Destroyed lobby by %s", client.Name)
			delete(h.clientByName, client.Name)
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
					delete(h.clientByName, client.Name)
					delete(h.clients, client)
				}
			}
		}
	}
}
