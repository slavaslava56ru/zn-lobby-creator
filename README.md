## zn-lobby-creator

Microservice for work with game lobby. Works with websocket.
Default port 8080

# Allowed command:

- /w playerName message - send message to player 
- /lobby message - send message to lobby
- /create_lobby lobbby_name 12 - create lobby with name (name should uniq), 12 - max player count
- /destroy_lobby - destroy current lobby (can only owner)
- /get_lobby_list - get all available lobbys
- /join_to_lobby lobby_name - joined to lobby with name
- /leave_from_lobby - leave from current lobby
- /get_all_players_from_lobby - get all players from current lobby

# Output format:
```
{
  "action": "",
  "body": {}
}
```

if errror:
```
{
  "action": "error",
  "body": "message"
}
```

action list if ok:
- send_message_for_player - only for sended client
- send_message_lobby - for all client of lobby
- lobby_created - for owner
- lobby_destroyed - for all client of lobby
- lobby_list - for runned client
- user_joined_to_lobby - for all client of lobby
- user_leaved_from_lobby - for all client of lobby
- all_players - for runned client
