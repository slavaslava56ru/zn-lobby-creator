package hub

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"zn/entity"
	"zn/lobby"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	uuid   string
	logger *zap.SugaredLogger

	name         string
	currentLobby *lobby.Lobby

	hub *Hub

	// Buffered channel of outbound messages.
	send chan []byte

	ipAddress string
}

func NewClient(logger *zap.SugaredLogger, name string, hub *Hub, send chan []byte, ipAddress string) *Client {

	return &Client{
		uuid:      uuid.New().String(),
		logger:    logger,
		name:      name,
		hub:       hub,
		send:      send,
		ipAddress: ipAddress,
	}
}

func (c *Client) Start() {

	c.logger.Info("Register new client")
	c.hub.RegisterClient(c)
}

func (c *Client) Close() {

	c.logger.Info("Unregister new client")
	c.hub.UnregisterClient(c)
}

func (c *Client) HandleInput(input string) {

	if len(input) < 1 {
		c.send <- getMessage("error", "unknown command, bad length")
		return
	}

	firstCharacter := input[0:1]
	if firstCharacter != "/" {
		c.send <- getMessage("error", "unknown command, bad format")
		return
	}

	split := strings.SplitN(input, " ", 2)

	switch split[0] {

	case "/w":

		split = strings.SplitN(input, " ", 3)
		if len(split) != 3 {
			c.send <- getMessage("error", "need player_name and message")
			return
		}

		c.hub.mu.RLock()
		client, exist := c.hub.clientByName[split[1]]
		c.hub.mu.RUnlock()

		if !exist {
			c.send <- getMessage("error", "need correct player_name")
			return
		}

		client.send <- getMessage("send_message_for_player", split[2])

	case "/lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not found %s", split[0]))
			return
		}

		if len(split) != 2 {
			c.send <- getMessage("error", "unknown command, bad length split")
			return
		}
		err := c.hub.BroadcastToLobby(c.currentLobby.GetName(), getMessage("send_message_lobby", split[2]))
		if err != nil {
			c.logger.Infof("Error on send message to lobby %s", err.Error())
		}

	case "/create_lobby":

		if c.currentLobby != nil {
			c.send <- getMessage("error", "lobby already exist")
			return
		}
		split = strings.SplitN(input, " ", 3)
		if len(split) != 3 {
			c.send <- getMessage("error", "unknown command, bad length split")
			return
		}

		maxPlayers, err := strconv.Atoi(split[2])
		if err != nil {
			c.send <- getMessage("error", "unknown command, bad max players")
			return
		}

		c.hub.muLobby.RLock()
		_, exist := c.hub.lobbys[split[1]]
		c.hub.muLobby.RUnlock()
		if exist {
			c.send <- getMessage("error", "lobby with selected name is exist")
			return
		}

		c.currentLobby = lobby.NewLobby(c.uuid, split[1], maxPlayers)

		c.hub.muLobby.Lock()
		c.hub.lobbys[split[1]] = c.currentLobby
		c.hub.muLobby.Unlock()
		c.send <- getMessage("lobby_created", c.ipAddress)

	case "/destroy_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not found %s", split[0]))
			return
		}

		if !c.currentLobby.IsOwner(c.uuid) {
			c.send <- getMessage("error", "destroy lobby can only owner")
			return
		}

		err := c.hub.BroadcastToLobby(c.currentLobby.GetName(), getMessage("lobby_destroyed", ""))
		if err != nil {
			c.logger.Infof("Error on send message destory lobby %s", err.Error())
		}
		c.currentLobby.Close()

	case "/get_lobby_list":

		c.hub.muLobby.RLock()
		lobbyList := make([]*entity.LobbyItem, 0, len(c.hub.lobbys))
		for _, lobbyItem := range c.hub.lobbys {

			lobbyList = append(lobbyList, &entity.LobbyItem{
				Name:           lobbyItem.GetName(),
				CurrentPlayers: lobbyItem.GetCurrentPlayers(),
				MaxPlayers:     lobbyItem.GetMaxPlayers(),
			})
		}
		c.hub.muLobby.RUnlock()
		c.send <- getMessage("lobby_list", lobbyList)

	case "/join_to_lobby":

		if len(split) != 2 {
			c.send <- getMessage("error", "unknown command, bad length split")
			return
		}

		if c.currentLobby != nil {
			c.send <- getMessage("error", "lobby already exist")
			return
		}

		c.hub.muLobby.RLock()
		lobbyItem, exist := c.hub.lobbys[split[1]]
		c.hub.muLobby.RUnlock()
		if !exist {
			c.send <- getMessage("error", fmt.Sprintf("lobby not found %s", split[0]))
			return
		}
		clientsUUID := lobbyItem.RegisterClient(c.uuid)
		c.currentLobby = lobbyItem

		players := make([]*entity.UserItem, 0, len(clientsUUID))
		for index, client := range clientsUUID {

			client, exist := c.hub.GetClient(client)
			if !exist {

				c.logger.Infof("Client not exist in hub")
				continue
			}
			players = append(players, &entity.UserItem{
				Name:        client.name,
				PlayerIndex: int64(index),
			})
		}

		err := c.hub.BroadcastToLobby(lobbyItem.GetName(), getMessage("user_joined_to_lobby", players))
		if err != nil {
			c.logger.Infof("Error on send message destory lobby %s", err.Error())
		}

		c.logger.Infof("add new Client to lobby %s", c.name)

	case "/leave_from_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not exist %s", split[0]))
			return
		}

		clientsUUID := c.currentLobby.UnregisterClient(c.uuid)

		players := make([]*entity.UserItem, 0, len(clientsUUID))
		for index, client := range clientsUUID {

			client, exist := c.hub.GetClient(client)
			if !exist {

				c.logger.Infof("Client not exist in hub")
				continue
			}
			players = append(players, &entity.UserItem{
				Name:        client.name,
				PlayerIndex: int64(index),
			})
		}

		err := c.hub.BroadcastToLobby(c.currentLobby.GetName(), getMessage("user_leaved_from_lobby", players))
		c.currentLobby = nil
		if err != nil {
			c.logger.Infof("Error on send message delete player from lobby %s", err.Error())
		}

		c.logger.Infof("delete Client from lobby %s", c.name)

	case "/get_all_players_from_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not exist %s", split[0]))
			return
		}

		clients := c.currentLobby.GetClients()
		players := make([]*entity.UserItem, 0, len(clients))
		for index, client := range clients {

			client, exist := c.hub.GetClient(client)
			if !exist {

				c.logger.Infof("Client not exist in hub")
				continue
			}
			players = append(players, &entity.UserItem{
				Name:        client.name,
				PlayerIndex: int64(index),
			})
		}
		c.send <- getMessage("all_players", players)

	default:
		c.send <- getMessage("error", fmt.Sprintf("unknown command %s", split[0]))
	}
}

func getMessage(action string, body interface{}) []byte {

	message := &entity.Message{
		Action: action,
		Body:   body,
	}
	messageBytes, _ := json.Marshal(message)
	return messageBytes
}
