package game

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby interface {
		AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error
		RemoveUser(u db.Username)
		addGame(u db.Username)
		getGameInfos(u db.Username)
	}

	lobbyImpl struct {
		log           *log.Logger
		upgrader      *websocket.Upgrader
		words         map[string]bool
		players       map[db.Username]player
		games         []game
		maxGames      int
		userAdditions chan userAddition
		userRemovals  chan db.Username
		gameCreates   chan db.Username
		gameInfos     chan db.Username
	}

	userAddition struct {
		u    db.Username
		w    http.ResponseWriter
		r    *http.Request
		done chan<- error
	}
)

// NewLobby creates a new game lobby
func NewLobby(log *log.Logger, ws WordsSupplier) (Lobby, error) {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	words, err := ws.Words()
	if err != nil {
		return nil, err
	}
	l := lobbyImpl{
		log:           log,
		upgrader:      u,
		words:         words,
		games:         make([]game, 1),
		players:       make(map[db.Username]player),
		maxGames:      5,
		userAdditions: make(chan userAddition, 16),
		userRemovals:  make(chan db.Username, 16),
		gameCreates:   make(chan db.Username, 4),
		gameInfos:     make(chan db.Username, 16),
	}
	go l.run()
	return l, nil
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

// RemoveUser removes the user from the lobby an a game, if any
func (l lobbyImpl) RemoveUser(u db.Username) {
	l.userRemovals <- u
}

func (l lobbyImpl) addGame(u db.Username) {
	l.gameCreates <- u
}

func (l lobbyImpl) getGameInfos(u db.Username) {
	l.gameInfos <- u
}

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
				return
			}
			l.remove(u)
		case r, ok := <-l.gameInfos:
			if !ok {
				l.log.Println("lobby closing because game info queue closed")
				return
			}
			l.sendGameInfos(r)
		case u, ok := <-l.gameCreates:
			if !ok {
				l.log.Println("lobby closing because game creation queue closed")
				return
			}
			l.sendNewGame(u)
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
	p.sendMessage(infoMessage{Type: userRemove, Username: p.username()})
}

func (l lobbyImpl) sendGameInfos(u db.Username) {
	p, ok := l.players[u]
	if !ok {
		l.log.Printf("Could not send game info to nonexistant player: %v", u)
	}
	n := len(l.gameInfos)
	c := make(chan gameInfo, n)
	for _, g := range l.games {
		g.infoRequest(gameInfoRequest{u: u, c: c})
	}
	gameInfos := make([]gameInfo, n)
	i := 0
	for {
		gameInfo, ok := <-c
		if !ok {
			l.log.Printf("game info stream closed unexpectedly")
			return
		}
		gameInfos[i] = gameInfo
		i++
		if i == n {
			sort.Slice(gameInfos, func(i, j int) bool {
				return gameInfos[i].CreatedAt < gameInfos[j].CreatedAt
			})
			p.sendMessage(gameInfosMessage(gameInfos))
			return
		}
	}
}

func (l lobbyImpl) sendNewGame(u db.Username) {
	p, ok := l.players[u]
	if !ok {
		l.log.Printf("cannot create new game for nonexistant user: %v", u)
	}
	g := newGame(l.log, l.words, p)
	if len(l.games) >= l.maxGames {
		p.sendMessage(infoMessage{Info: "the maximum number of games have already been created"})
		return
	}
	l.games = append(l.games, g)
	l.getGameInfos(u)
}
