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
	// Lobby is the place users can create, join, and participate in games
	Lobby interface {
		AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error
		RemoveUser(u db.Username)
	}

	lobbyImpl struct {
		log           *log.Logger
		upgrader      *websocket.Upgrader
		players       map[db.Username]player
		games         []game
		maxGames      int
		userAdditions chan userAddition
		userRemovals  chan db.Username
	}

	userAddition struct {
		u    db.Username
		w    http.ResponseWriter
		r    *http.Request
		done chan<- error
	}
)

// NewLobby creates a new game lobby
func NewLobby(log *log.Logger) Lobby {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	l := lobbyImpl{
		log:           log,
		upgrader:      u,
		games:         make([]game, 1),
		players:       make(map[db.Username]player),
		maxGames:      5,
		userAdditions: make(chan userAddition, 16),
		userRemovals:  make(chan db.Username, 16),
	}
	go l.run()
	return l
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l lobbyImpl) AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error {
	done := make(chan error, 1)
	ua := userAddition{
		u:    u,
		w:    w,
		r:    r,
		done: done,
	}
	l.userAdditions <- ua
	return <-done
}

func (l lobbyImpl) RemoveUser(u db.Username) {
	l.userRemovals <- u
}

// TODO: lobby: get games
// func (l lobby) GetGames(u db.Username) map[game]bool {
// 	// TODO: make GameInfo struct with game id, started date, players, and other info, return an array of that
// 	m := make(map[game]bool, len(l.games))
// 	for _, g := range gl.games {
// 		m[g] = g.Has(u)
// 	}
// 	return m
// }

func (l lobbyImpl) run() {
	for {
		select {
		case ua, ok := <-l.userAdditions:
			if !ok {
				l.log.Println("lobby closing because user registration queue closed")
				return
			}
			err := l.add(ua)
			ua.done <- err
		case u, ok := <-l.userRemovals:
			if !ok {
				l.log.Println("lobby closing because user removal queue closed")
			}
			l.remove(u)
		}
	}
}

func (l lobbyImpl) add(ua userAddition) error {
	if _, ok := l.players[ua.u]; ok {
		return errors.New("user already in the game lobby")
	}
	conn, err := l.upgrader.Upgrade(ua.w, ua.r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	p := newPlayer(l.log, l, ua.u, conn)
	l.players[ua.u] = p
	return nil
}

func (l lobbyImpl) remove(u db.Username) {
	p, ok := l.players[u]
	if !ok {
		return
	}
	delete(l.players, u)
	p.sendMessage(userRemoveMessage(u))
}
