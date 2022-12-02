package hub

import (
	"fmt"
	"sync"
	"zn/lobby"

	"go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	logger *zap.SugaredLogger

	// клиенты
	clients map[string]*Client

	mu           sync.RWMutex
	clientByName map[string]*Client

	// клиенты
	muLobby sync.RWMutex
	lobbys  map[string]*lobby.Lobby

	// канал для рассылки сообщения всем
	broadcast chan []byte
}

func NewHub(logger *zap.SugaredLogger) *Hub {
	return &Hub{
		logger:       logger,
		broadcast:    make(chan []byte),
		clients:      make(map[string]*Client),
		clientByName: make(map[string]*Client),
		lobbys:       make(map[string]*lobby.Lobby),
	}
}

func (h *Hub) RegisterClient(client *Client) {

	h.mu.Lock()
	h.clients[client.uuid] = client
	h.clientByName[client.name] = client
	h.mu.Unlock()

	h.logger.Info("Add new Client %s", client.name)
}

func (h *Hub) UnregisterClient(client *Client) {

	if client.currentLobby != nil && client.currentLobby.IsOwner(client.uuid) {
		client.currentLobby.Close()
	}

	h.mu.Lock()
	delete(h.clients, client.uuid)
	delete(h.clientByName, client.name)
	h.mu.Unlock()

	h.logger.Info("Unregister client %s", client.name)
}

func (h *Hub) GetClient(clientUUID string) (*Client, bool) {

	h.mu.RLock()
	client, ok := h.clients[clientUUID]
	h.mu.RUnlock()

	return client, ok
}

func (h *Hub) BroadcastToLobby(lobbyName string, message []byte) error {

	h.mu.RLock()
	defer h.mu.RUnlock()

	lobbyTemp, exist := h.lobbys[lobbyName]
	if !exist {
		return fmt.Errorf("lobby not found")
	}

	clients := lobbyTemp.GetClients()

	for _, client := range clients {
		client, exist := h.GetClient(client)
		if !exist {
			continue
		}
		client.send <- message
	}

	return nil
}

func (h *Hub) Run() {

	for {

		select {

		case message := <-h.broadcast:

			// перебираем всех клиентов
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:

					if client.currentLobby != nil && client.currentLobby.IsOwner(client.uuid) {
						client.currentLobby.Close()
					}
					close(client.send)
					h.mu.Lock()
					delete(h.clientByName, client.name)
					h.mu.Unlock()
					delete(h.clients, client.uuid)
				}
			}
		}
	}
}
