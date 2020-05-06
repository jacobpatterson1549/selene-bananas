package game

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby interface {
		AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error
		RemoveUser(u db.Username)
	}

	lobby struct {
		log      *log.Logger
		upgrader *websocket.Upgrader
		rand     *rand.Rand
		words    map[string]bool
		userDao  db.UserDao
		players  map[db.Username]*player
		games    map[int]*game
		maxGames int
		messages chan message
	}
)

// NewLobby creates a new game lobby
func NewLobby(log *log.Logger, ws WordsSupplier, userDao db.UserDao, rand *rand.Rand) (Lobby, error) {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	words, err := ws.Words()
	if err != nil {
		return nil, err
	}
	l := lobby{
		log:      log,
		upgrader: u,
		rand:     rand,
		words:    words,
		userDao:  userDao,
		games:    make(map[int]*game),
		players:  make(map[db.Username]*player),
		maxGames: 5,
		messages: make(chan message, 16),
	}
	go l.run()
	return &l, nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *lobby) AddUser(u db.Username, w http.ResponseWriter, r *http.Request) error {
	conn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	p := l.newPlayer(u, conn)
	l.messages <- message{Type: playerCreate, Player: &p}
	return nil
}

func (l lobby) newPlayer(u db.Username, conn *websocket.Conn) player {
	s := socket{
		log:      l.log,
		conn:     conn,
		messages: make(chan message, 16),
	}
	p := player{
		log:      l.log,
		username: u,
		lobby:    &l,
		socket:   &s,
		messages: make(chan message, 16),
	}
	s.player = &p
	go s.readMessages()
	go s.writeMessages()
	go p.run()
	return p
}

// RemoveUser removes the user from the lobby an a game, if any
func (l *lobby) RemoveUser(u db.Username) {
	l.messages <- message{Type: playerDelete, Player: &player{username: u}}
}

func (l *lobby) run() {
	messageHandlers := map[messageType]func(message){
		gameCreate:   l.handleGameCreate,
		gameJoin:     l.handleGameJoin,
		gameDelete:   l.handleGameDelete,
		gameInfos:    l.handleGameInfos,
		playerCreate: l.handlePlayerCreate,
		playerDelete: l.handlePlayerDelete,
	}
	for m := range l.messages {
		mh, ok := messageHandlers[m.Type]
		if !ok {
			l.log.Printf("lobby does not know how to handle messageType %v", m.Type)
			continue
		}
		mh(m)
	}
	for _, p := range l.players {
		p.messages <- message{Type: playerDelete, Info: "lobby closing"}
	}
	for _, g := range l.games {
		g.messages <- message{Type: gameDelete, Info: "lobby closing"}
	}
	l.log.Printf("lobby closed")
}

func (l *lobby) handleGameCreate(m message) {
	if len(l.games) >= l.maxGames {
		m.Player.messages <- message{Type: socketError, Info: "the maximum number of games have already been created"}
		return
	}
	id := 1
	for existingID := range l.games {
		if existingID != id {
			break
		}
		id++
	}
	g := l.newGame(m.Player, id)
	go g.run()
	l.games[id] = &g
	l.handleGameJoin(message{Type: gameJoin, Player: m.Player, GameID: id})
}

func (l *lobby) newGame(p *player, id int) game {
	// TODO: for createdAt, have TimeFunc variable that is a function which returns a time.Time, (time.Now) => TimeFunc().Format(...), share with jwt token
	g := game{
		log:         l.log,
		lobby:       l,
		id:          id,
		createdAt:   time.Now().Format(time.UnixDate),
		state:       gameNotStarted,
		words:       l.words,
		players:     make(map[db.Username]*gamePlayerState, 2),
		userDao:     l.userDao,
		maxPlayers:  8,
		numNewTiles: 21,
		messages:    make(chan message, 64),
		shuffleUnusedTilesFunc: func(tiles []tile) {
			l.rand.Shuffle(len(tiles), func(i, j int) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			})
		},
		shufflePlayersFunc: func(players []*player) {
			l.rand.Shuffle(len(players), func(i, j int) {
				players[i], players[j] = players[j], players[i]
			})
		},
	}
	g.initializeUnusedTiles()
	return g
}

func (l *lobby) handleGameJoin(m message) {
	g, ok := l.games[m.GameID]
	if !ok {
		m.Player.messages <- message{Type: socketError, Info: fmt.Sprintf("no game with id %v, please refresh games", m.GameID)}
		return
	}
	g.messages <- message{Type: gameJoin, Player: m.Player}
	m.Player.messages <- message{Type: gameJoin, Game: g}
}

func (l *lobby) handleGameDelete(m message) {
	g, ok := l.games[m.GameID]
	if !ok {
		if m.Player != nil {
			m.Player.messages <- message{Type: socketError, Info: fmt.Sprintf("no game with id %v, please refresh games", m.GameID)}
		}
		return
	}
	delete(l.games, m.GameID)
	g.messages <- message{Type: gameDelete, Info: m.Info}
}

func (l *lobby) handleGameInfos(m message) {
	c := make(chan gameInfo, len(l.games))
	for _, g := range l.games {
		g.messages <- message{Type: gameInfos, Player: m.Player, GameInfoChan: c}
	}
	n := len(l.games)
	s := make([]gameInfo, n)
	if n > 0 {
		i := 0
		for {
			s[i] = <-c
			i++
			if i == n {
				break
			}
		}
	}
	sort.Slice(s, func(i, j int) bool {
		return s[i].CreatedAt < s[j].CreatedAt
	})
	m.Player.messages <- message{Type: gameInfos, GameInfos: s}
}

func (l *lobby) handlePlayerCreate(m message) {
	_, ok := l.players[m.Player.username]
	if ok {
		m.Player.messages <- message{Type: socketError, Info: "player already in lobby, replacing connection"}
	}
	l.players[m.Player.username] = m.Player
	l.log.Printf("%v joined the lobby", m.Player.username)
}

func (l lobby) handlePlayerDelete(m message) {
	p, ok := l.players[m.Player.username]
	if !ok {
		l.log.Printf("player %v not in lobby, cannot remove", m.Player.username)
		return
	}
	delete(l.players, p.username)
	p.messages <- message{Type: playerDelete}
	l.log.Printf("%v left the lobby", m.Player.username)
}
