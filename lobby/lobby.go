package lobby

import (
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Hub maintains the set of active clientsUUID and broadcasts messages to the
// clientsUUID.
type Lobby struct {
	uuid   string
	logger *zap.SugaredLogger

	name       string
	ownerUUID  string
	maxPlayers int

	// клиентв
	mu          sync.RWMutex
	clientsUUID map[string]int64

	availableIndexList []int64
}

func NewLobby(ownerUUID string, lobbyName string, maxPlayers int) *Lobby {

	availableIndexList := make([]int64, 0, maxPlayers-1)
	for i := 0; i < (maxPlayers); i++ {
		availableIndexList = append(availableIndexList, int64(i))
	}

	clientsUUID := make(map[string]int64)
	clientsUUID[ownerUUID] = 0
	return &Lobby{
		uuid:               uuid.New().String(),
		ownerUUID:          ownerUUID,
		name:               lobbyName,
		maxPlayers:         maxPlayers,
		clientsUUID:        clientsUUID,
		availableIndexList: availableIndexList,
	}
}

func (l *Lobby) RegisterClient(clientUUID string) []string {

	l.mu.Lock()
	l.clientsUUID[clientUUID] = 0

	players := make([]string, 0, len(l.clientsUUID))
	for client := range l.clientsUUID {

		players = append(players, client)
	}
	l.mu.Unlock()

	return players
}

func (l *Lobby) UnregisterClient(clientUUID string) []string {

	l.mu.Lock()
	delete(l.clientsUUID, clientUUID)
	l.clientsUUID[clientUUID] = 0

	players := make([]string, 0, len(l.clientsUUID))
	for client := range l.clientsUUID {

		players = append(players, client)
	}
	l.mu.Unlock()
	return players
}

func (l *Lobby) GetClients() []string {

	l.mu.Lock()

	players := make([]string, 0, len(l.clientsUUID))
	for client := range l.clientsUUID {

		players = append(players, client)
	}
	l.mu.Unlock()
	return players
}

func (l *Lobby) IsOwner(clientOwner string) bool {

	return l.ownerUUID == clientOwner
}

func (l *Lobby) GetUuid() string {

	return l.uuid
}

func (l *Lobby) GetName() string {

	return l.name
}

func (l *Lobby) GetCurrentPlayers() int {

	l.mu.RLock()
	count := len(l.clientsUUID)
	l.mu.RUnlock()
	return count
}

func (l *Lobby) GetMaxPlayers() int {

	return l.maxPlayers
}

func (l *Lobby) Close() {

	l.mu.Lock()
	l.clientsUUID = nil
	l.mu.Unlock()
	l.logger.Info("Lobby destroyed")
}
