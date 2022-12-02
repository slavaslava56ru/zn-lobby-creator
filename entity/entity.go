package entity

type Message struct {
	Action string      `json:"action"`
	Body   interface{} `json:"body"`
}

type LobbyItem struct {
	Name           string `json:"name"`
	CurrentPlayers int    `json:"current_players"`
	MaxPlayers     int    `json:"max_players"`
}

type UserItem struct {
	Name        string `json:"name"`
	PlayerIndex int64  `json:"player_index"`
}
