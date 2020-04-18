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
		AddUser(u db.User, w http.ResponseWriter, r *http.Request) error
		//RemoveUser removes a user from the lobby
		RemoveUser(u db.User) error
	}

	gameLobby struct {
		upgrader *websocket.Upgrader
		games    map[int]Game
		players  map[db.User]player
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
		games:    make(map[int]Game),
		players:  make(map[db.User]player),
		maxGames: 5,
	}
}

func (gl gameLobby) AddUser(u db.User, w http.ResponseWriter, r *http.Request) error {
	if _, ok := gl.players[u]; ok {
		return errors.New("user already in the game lobby")
	}
	conn, err := gl.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	p := player{
		user: u,
		conn: conn,
	}
	go p.readMessages()
	go p.writeMessages()
	gl.players[u] = p
	return nil
}

func (gl gameLobby) RemoveUser(u db.User) error {
	if _, ok := gl.players[u]; !ok {
		return errors.New("user not already in the game lobby")
	}
	for _, g := range gl.games {
		if g.Has(u) {
			g.Remove(u)
		}
	}
	delete(gl.players, u)
	return nil
}
