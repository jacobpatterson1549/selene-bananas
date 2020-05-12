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
	player_socket "github.com/jacobpatterson1549/selene-bananas/go/game/player"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		log        *log.Logger
		upgrader   *websocket.Upgrader
		rand       *rand.Rand
		words      map[string]bool
		playerCfg  player_socket.Config
		players    map[game.PlayerName]player
		gameCfg    controller.Config
		games      map[game.ID]game.MessageHandler
		maxGames   int
		messages   chan game.Message
		newPlayers chan player
	}

	player struct {
		name   game.PlayerName
		gameID game.ID
		game.MessageHandler
	}
)

var _ game.MessageHandler = &Lobby{}

// New creates a new game lobby
func New(log *log.Logger, ws game.WordsSupplier, userDao db.UserDao, rand *rand.Rand) (Lobby, error) {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	words, err := ws.Words()
	if err != nil {
		return Lobby{}, err
	}
	l := Lobby{
		log:        log,
		upgrader:   u,
		rand:       rand,
		words:      words,
		games:      make(map[game.ID]game.MessageHandler),
		players:    make(map[game.PlayerName]player),
		maxGames:   5,
		messages:   make(chan game.Message, 16),
		newPlayers: make(chan player, 8),
	}
	l.playerCfg = player_socket.Config{
		Log:   l.log,
		Lobby: &l,
	}
	l.gameCfg = controller.Config{
		Log:         l.log,
		Lobby:       &l,
		UserDao:     userDao,
		Words:       words,
		MaxPlayers:  8,
		NumNewTiles: 21,
		// TODO: make defaults in Game
		ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
			l.rand.Shuffle(len(tiles), func(i, j int) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			})
		},
		ShufflePlayersFunc: func(players []game.PlayerName) {
			l.rand.Shuffle(len(players), func(i, j int) {
				players[i], players[j] = players[j], players[i]
			})
		},
	}
	go l.run()
	return l, nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *Lobby) AddUser(name game.PlayerName, w http.ResponseWriter, r *http.Request) error {
	conn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	playerSocket := l.playerCfg.NewPlayer(name, conn)
	p := player{
		name,
		game.ID(0),
		&playerSocket,
	}
	l.newPlayers <- p
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
		game.Delete:         l.handleGameDelete,
		game.StatusChange:   l.handleGameMessage,
		game.Snag:           l.handleGameMessage,
		game.Swap:           l.handleGameMessage,
		game.TilesMoved:     l.handleGameMessage,
		game.Infos:          l.handleGameInfos,
		game.PlayerDelete:   l.handlePlayerDelete,
		game.SocketInfo:     l.handlePlayerMessage,
		game.SocketError:    l.handlePlayerMessage,
		game.SocketHTTPPing: l.handlePlayerMessage,
	}
	for {
		select {
		case m, ok := <-l.messages:
			if !ok {
				return
			}
			mh, ok := messageHandlers[m.Type]
			if !ok {
				l.log.Printf("lobby does not know how to handle messageType %v", m.Type)
				continue
			}
			if _, ok := l.players[m.PlayerName]; !ok {
				l.log.Printf("lobby does not have player named '%v'", m.PlayerName)
				continue
			}
			mh(m)
		case p, ok := <-l.newPlayers:
			if !ok {
				return
			}
			l.handlePlayerCreate(p)
		}
	}
}

func (l *Lobby) close() {
	for _, p := range l.players {
		p.Handle(game.Message{
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
	p := l.players[m.PlayerName]
	if len(l.games) >= l.maxGames {
		p.Handle(game.Message{
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
	p.Handle(game.Message{
		Type:       game.Join,
		GameID:     id,
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
		return
	}
	delete(l.games, m.GameID)
	g.Handle(game.Message{
		Type: game.Delete,
	})
	for n, p := range l.players {
		var info string
		switch n {
		case m.PlayerName:
			info = "game deleted"
		default:
			info = fmt.Sprintf("%v deleted the game", m.PlayerName)
		}
		p.Handle(game.Message{
			Type: game.Delete,
			Info: info,
		})
	}
}

func (l *Lobby) handleGameInfos(m game.Message) {
	p := l.players[m.PlayerName]
	n := len(l.games)
	c := make(chan game.Info, len(l.games))
	for _, g := range l.games {
		g.Handle(game.Message{
			Type:         game.Infos,
			PlayerName:   m.PlayerName,
			GameInfoChan: c,
		})
	}

	s := make([]game.Info, n)
	if n > 0 {
		i := 0
		for {
			gameInfo, ok := <-c
			if !ok {
				p.Handle(game.Message{
					Type: game.SocketError,
					Info: "could not get game infos, the outbound game info channel closed unexpectedly",
				})
				return
			}
			s[i] = gameInfo
			i++
			if i == n {
				break
			}
		}
	}
	sort.Slice(s, func(i, j int) bool {
		return s[i].CreatedAt < s[j].CreatedAt
	})
	p.Handle(game.Message{
		Type:      game.Infos,
		GameInfos: s,
	})
}

func (l *Lobby) handlePlayerCreate(p player) {
	if _, ok := l.players[p.name]; ok {
		p.Handle(game.Message{
			Type: game.SocketError,
			Info: "player already in lobby, replacing connection",
		})
	}
	l.players[p.name] = p
	l.log.Printf("%v joined the lobby", p.name)
}

func (l Lobby) handlePlayerDelete(m game.Message) {
	p, ok := l.players[m.PlayerName]
	if !ok {
		l.log.Printf("player %v not in lobby, cannot remove", m.PlayerName)
		return
	}
	delete(l.players, m.PlayerName)
	p.Handle(game.Message{
		Type: game.PlayerDelete,
	})
	l.log.Printf("%v left the lobby", m.PlayerName)
}

func (l *Lobby) handleGameMessage(m game.Message) {
	p := l.players[m.PlayerName]
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

func (l *Lobby) handlePlayerMessage(m game.Message) {
	p := l.players[m.PlayerName]
	p.Handle(m)
}
