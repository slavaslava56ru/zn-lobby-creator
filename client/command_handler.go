package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type message struct {
	Action string      `json:"action"`
	Body   interface{} `json:"body"`
}

type lobbyItem struct {
	Name           string `json:"name"`
	CurrentPlayers int    `json:"current_players"`
	MaxPlayers     int    `json:"max_players"`
}

type userItem struct {
	Name        string `json:"name"`
	PlayerIndex int64  `json:"player_index"`
}

func (c *Client) handleInput(input string) {

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
		c.currentLobby.broadcast <- getMessage("send_message_lobby", split[2])

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

		c.currentLobby = NewLobby(c, split[1], maxPlayers, c.hub)
		go c.currentLobby.Run()

		c.hub.muLobby.Lock()
		c.hub.lobbys[split[1]] = c.currentLobby
		c.hub.muLobby.Unlock()
		c.send <- getMessage("lobby_created", c.conn.RemoteAddr().String())

	case "/destroy_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not found %s", split[0]))
			return
		}

		if c.currentLobby.owner != c {
			c.send <- getMessage("error", "destroy lobby can only owner")
			return
		}

		c.currentLobby.broadcast <- getMessage("lobby_destroyed", "")
		c.currentLobby.destroyLobby <- true

	case "/get_lobby_list":

		c.hub.muLobby.RLock()
		lobbyList := make([]*lobbyItem, 0, len(c.hub.lobbys))
		for _, lobby := range c.hub.lobbys {

			lobbyList = append(lobbyList, &lobbyItem{
				Name:           lobby.name,
				CurrentPlayers: len(lobby.clients),
				MaxPlayers:     lobby.maxPlayers,
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
		lobby, exist := c.hub.lobbys[split[1]]
		c.hub.muLobby.RUnlock()
		if !exist {
			c.send <- getMessage("error", fmt.Sprintf("lobby not found %s", split[0]))
			return
		}
		lobby.register <- c

	case "/leave_from_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not exist %s", split[0]))
			return
		}

		c.currentLobby.unregister <- c
		c.currentLobby = nil

	case "/get_all_players_from_lobby":

		if c.currentLobby == nil {
			c.send <- getMessage("error", fmt.Sprintf("lobby not exist %s", split[0]))
			return
		}

		players := make([]*userItem, 0, len(c.currentLobby.clients))
		for client := range c.currentLobby.clients {

			players = append(players, &userItem{
				Name:        client.Name,
				PlayerIndex: c.currentLobby.clients[client],
			})
		}
		c.send <- getMessage("all_players", players)

	default:
		c.send <- getMessage("error", fmt.Sprintf("unknown command %s", split[0]))
	}
}

func getMessage(action string, body interface{}) []byte {

	message := &message{
		Action: action,
		Body:   body,
	}
	messageBytes, _ := json.Marshal(message)
	return messageBytes
}
