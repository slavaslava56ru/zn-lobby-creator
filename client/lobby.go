package client

import (
	"fmt"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type lobby struct {
	name       string
	owner      *Client
	maxPlayers int
	hub        *Hub

	// клиентв
	clients map[*Client]int64

	// канал для рассылки сообщения всем в лобби
	broadcast chan []byte

	// Регистрируем клиентов
	register chan *Client

	destroyLobby chan bool

	// Удаляем клиентов
	unregister chan *Client

	availableIndexList []int64
}

func NewLobby(owner *Client, lobbyName string, maxPlayers int, hub *Hub) *lobby {

	availableIndexList := make([]int64, 0, maxPlayers-1)
	for i := 0; i < (maxPlayers); i++ {
		availableIndexList = append(availableIndexList, int64(i))
	}

	return &lobby{
		owner:              owner,
		name:               lobbyName,
		hub:                hub,
		maxPlayers:         maxPlayers,
		broadcast:          make(chan []byte),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		destroyLobby:       make(chan bool),
		clients:            make(map[*Client]int64),
		availableIndexList: availableIndexList,
	}
}

func (l *lobby) Run() {

	// добавляем owner
	l.clients[l.owner] = 0
	log.Println("Lobby created")
	defer func() {

		delete(l.hub.lobbys, l.name)
		l.clients = nil
		log.Println("Lobby destroyed")
	}()
	for {

		select {
		case client := <-l.register:
			l.clients[client] = 0

			players := make([]*userItem, 0, len(l.clients))
			for client := range l.clients {

				players = append(players, &userItem{
					Name:        client.Name,
					PlayerIndex: l.clients[client],
				})
			}
			l.broadcast <- getMessage("user_joined_to_lobby", players)
			log.Printf("add new Client to lobby %s", client.Name)
		case client := <-l.unregister:

			delete(l.clients, client)
			players := make([]*userItem, 0, len(l.clients))
			for client := range l.clients {

				players = append(players, &userItem{
					Name:        client.Name,
					PlayerIndex: l.clients[client],
				})
			}
			l.broadcast <- getMessage("user_leaved_from_lobby", players)
			log.Println(fmt.Printf("delete client from lobby %s", client.Name))
		case message := <-l.broadcast:

			// перебираем всех клиентов
			for client := range l.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(l.clients, client)
				}
			}
		case <-l.destroyLobby:
			return
		}
	}
}
