package game

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Lobby tracts the connections for all users
	Lobby interface {
		// AddUser adds a user to the lobby
		AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error
		//RemoveUser removes a user from the lobby
		RemoveUser(u db.Username) error
		GetGames(u db.Username) map[Game]bool
	}

	gameLobby struct {
		upgrader *websocket.Upgrader
		players  map[db.Username]player
		games    []Game
		maxGames int
	}
)

// NewLobby creates a new Lobby for games
func NewLobby(log *log.Logger) Lobby {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	return gameLobby{
		upgrader: u,
		games:    make([]Game, 1),
		players:  make(map[db.Username]player),
		maxGames: 5,
	}
}

func (gl gameLobby) AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error {
	if _, ok := gl.players[u]; ok {
		return errors.New("user already in the game lobby")
	}
	conn, err := gl.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	p := player{
		username: u,
		conn:     conn,
	}
	go p.readMessages()
	go p.writeMessages()
	gl.players[u] = p
	return nil
}

func (gl gameLobby) RemoveUser(u db.Username) error {
	if _, ok := gl.players[u]; !ok {
		return errors.New("user not in the game lobby")
	}
	for i, g := range gl.games {
		g.Remove(u)
		if g.IsEmpty() {
			gl.games = append(gl.games[:i], gl.games[i+1:]...)
		}
	}
	delete(gl.players, u)
	return nil
}

func (gl gameLobby) GetGames(u db.Username) map[Game]bool {
	// TODO: make GameInfo struct with game id, started date, players, and other info, return an array of that
	m := make(map[Game]bool, len(gl.games))
	for _, g := range gl.games {
		m[g] = g.Has(u)
	}
	return m
}
