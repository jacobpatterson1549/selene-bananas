package lobby

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		debug            bool
		log              *log.Logger
		upgrader         *websocket.Upgrader
		rand             *rand.Rand
		words            map[string]bool
		socketCfg        socket.Config
		sockets          map[game.PlayerName]game.MessageHandler
		gameCfg          controller.Config
		games            map[game.ID]game.MessageHandler
		maxGames         int
		messages         chan game.Message
		newPlayerSockets chan playerSocket
	}

	// Config contiains the properties to create a lobby
	Config struct {
		Debug   bool
		Log     *log.Logger
		Rand    *rand.Rand
		UserDao db.UserDao
	}

	playerSocket struct {
		game.PlayerName
		game.MessageHandler
	}
)

var _ game.MessageHandler = &Lobby{}

// NewLobby creates a new game lobby
func (cfg Config) NewLobby(ws game.WordsSupplier) (Lobby, error) {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	l := Lobby{
		debug:            cfg.Debug,
		log:              cfg.Log,
		upgrader:         u,
		rand:             cfg.Rand,
		games:            make(map[game.ID]game.MessageHandler),
		sockets:          make(map[game.PlayerName]game.MessageHandler),
		maxGames:         5,
		messages:         make(chan game.Message, 16),
		newPlayerSockets: make(chan playerSocket, 8),
	}
	l.socketCfg = socket.Config{
		Debug: cfg.Debug,
		Log:   l.log,
		Lobby: &l,
	}
	words := ws.Words()
	l.gameCfg = controller.Config{
		Debug:       cfg.Debug,
		Log:         l.log,
		Lobby:       &l,
		UserDao:     cfg.UserDao,
		Words:       words,
		MaxPlayers:  8,
		NumNewTiles: 21,
		ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
			l.rand.Shuffle(len(tiles), func(i, j int) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			})
		},
		ShufflePlayersFunc: func(sockets []game.PlayerName) {
			l.rand.Shuffle(len(sockets), func(i, j int) {
				sockets[i], sockets[j] = sockets[j], sockets[i]
			})
		},
	}
	go l.run()
	return l, nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *Lobby) AddUser(playerName game.PlayerName, w http.ResponseWriter, r *http.Request) error {
	conn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s := l.socketCfg.NewSocket(playerName, conn)
	ps := playerSocket{
		PlayerName:     playerName,
		MessageHandler: &s,
	}
	l.newPlayerSockets <- ps
	return nil
}

// RemoveUser removes the user from the lobby and a game, if any
func (l *Lobby) RemoveUser(n game.PlayerName) {
	// TODO: this is probably broken
	l.Handle(game.Message{
		Type:       game.PlayerDelete,
		PlayerName: n,
	})
}

// Handle adds a message to the queue
func (l *Lobby) Handle(m game.Message) {
	l.messages <- m
}

func (l *Lobby) run() {
	defer l.close()
	messageHandlers := map[game.MessageType]func(game.Message){
		// game.Leave: TODO
		game.Create:         l.handleGameCreate,
		game.Join:           l.handleGameMessage,
		game.Leave:          l.handleSocketMessage,
		game.Delete:         l.handleGameDelete,
		game.StatusChange:   l.handleGameMessage,
		game.Snag:           l.handleGameMessage,
		game.Swap:           l.handleGameMessage,
		game.TilesMoved:     l.handleGameMessage,
		game.BoardRefresh:   l.handleSocketMessage,
		game.Infos:          l.handleGameInfos,
		game.PlayerDelete:   l.handlePlayerDelete,
		game.SocketInfo:     l.handleSocketMessage,
		game.SocketError:    l.handleSocketMessage,
		game.SocketHTTPPing: l.handleSocketMessage,
	}
	for {
		select {
		case m := <-l.messages:
			if l.debug {
				l.log.Printf("lobby handling message with type %v", m.Type)
			}
			mh, ok := messageHandlers[m.Type]
			if !ok {
				l.log.Printf("lobby does not know how to handle messageType %v", m.Type)
				continue
			}
			if _, ok := l.sockets[m.PlayerName]; !ok {
				l.log.Printf("lobby does not have socket for '%v'", m.PlayerName)
				continue
			}
			mh(m)
		case ps := <-l.newPlayerSockets:
			l.handlePlayerCreate(ps)
		}
	}
}

func (l *Lobby) close() {
	for _, s := range l.sockets {
		s.Handle(game.Message{
			Type: game.PlayerDelete,
			Info: "lobby closing",
		})
	}
	for _, g := range l.games {
		g.Handle(game.Message{
			Type: game.Delete,
			Info: "lobby closing",
		})
	}
	l.log.Printf("lobby closed")
}

func (l *Lobby) handleGameCreate(m game.Message) {
	s := l.sockets[m.PlayerName]
	if len(l.games) >= l.maxGames {
		s.Handle(game.Message{
			Type: game.SocketError,
			Info: "the maximum number of games have already been created",
		})
		return
	}
	var id game.ID = 1
	for existingID := range l.games {
		if existingID != id {
			break
		}
		id++
	}
	g := l.gameCfg.NewGame(id)
	l.games[id] = &g
	s.Handle(game.Message{
		Type:   game.Join,
		GameID: id,
	})
	g.Handle(game.Message{
		Type:       game.Join,
		PlayerName: m.PlayerName,
		GameID:     id,
	})
}

func (l *Lobby) handleGameDelete(m game.Message) {
	g, ok := l.games[m.GameID]
	if !ok {
		s := l.sockets[m.PlayerName]
		s.Handle(game.Message{
			Type: game.SocketError,
			Info: fmt.Sprintf("no game to delete with id %v", m.GameID),
		})
		return
	}
	delete(l.games, m.GameID)
	g.Handle(m)
}

func (l *Lobby) handleGameInfos(m game.Message) {
	s := l.sockets[m.PlayerName]
	n := len(l.games)
	c := make(chan game.Info, len(l.games))
	for _, g := range l.games {
		g.Handle(game.Message{
			Type:         game.Infos,
			PlayerName:   m.PlayerName,
			GameInfoChan: c,
		})
	}
	infos := make([]game.Info, n)
	if n > 0 {
		i := 0
		for {
			gameInfo, ok := <-c
			if !ok {
				s.Handle(game.Message{
					Type: game.SocketError,
					Info: "could not get game infos, the outbound game info channel closed unexpectedly",
				})
				return
			}
			infos[i] = gameInfo
			i++
			if i == n {
				break
			}
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt < infos[j].CreatedAt
	})
	s.Handle(game.Message{
		Type:      game.Infos,
		GameInfos: infos,
	})
}

func (l *Lobby) handlePlayerCreate(ps playerSocket) {
	if previousSocket, ok := l.sockets[ps.PlayerName]; ok {
		previousSocket.Handle(game.Message{
			Type: game.PlayerDelete,
			Info: "logged in from second location, replacing connection",
		})
	}
	l.sockets[ps.PlayerName] = ps
	l.log.Printf("%v joined the lobby", ps.PlayerName)
}

func (l Lobby) handlePlayerDelete(m game.Message) {
	s, ok := l.sockets[m.PlayerName]
	if !ok {
		l.log.Printf("player %v not in lobby, cannot remove", m.PlayerName)
		return
	}
	delete(l.sockets, m.PlayerName)
	s.Handle(game.Message{
		Type: game.PlayerDelete,
	})
	l.log.Printf("%v left the lobby", m.PlayerName)
}

func (l *Lobby) handleGameMessage(m game.Message) {
	p := l.sockets[m.PlayerName]
	g, ok := l.games[m.GameID]
	if !ok {
		p.Handle(game.Message{
			Type: game.SocketError,
			Info: fmt.Sprintf("no game with id %v, please refresh games", m.GameID),
		})
		return
	}
	g.Handle(m)
}

func (l *Lobby) handleSocketMessage(m game.Message) {
	p := l.sockets[m.PlayerName]
	p.Handle(m)
}
