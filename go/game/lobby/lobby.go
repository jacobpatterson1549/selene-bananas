package lobby

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/game/socket"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		debug          bool
		log            *log.Logger
		upgrader       *websocket.Upgrader
		maxGames       int
		maxSockets     int
		socketCfg      socket.Config
		gameCfg        controller.Config
		sockets        map[game.PlayerName]messageHandler
		games          map[game.ID]messageHandler
		addPlayers     chan playerSocket
		socketMessages chan game.Message
		gameMessages   chan game.Message
	}

	// Config contiains the properties to create a lobby
	Config struct {
		// Debug is a flag that causes the lobby to log the types messages that are read
		Debug bool
		// Log is used fot log errors and other information
		Log *log.Logger
		// MaxGames is the maximum number of games the lobby supports
		MaxGames int
		// MaxSockets is the maximum number of sockets the lobby supports
		MaxSockets int
		// GameCfg is used to create new games
		GameCfg controller.Config
		// SocketCfg is used to create new sockets
		SocketCfg socket.Config
	}

	playerSocket struct {
		game.PlayerName
		socket.Socket
	}

	messageHandler struct {
		done          chan<- struct{}
		writeMessages chan<- game.Message
	}
)

// NewLobby creates a new game lobby
func (cfg Config) NewLobby() Lobby {
	u := new(websocket.Upgrader)
	u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Println(reason)
	}
	l := Lobby{
		debug:          cfg.Debug,
		log:            cfg.Log,
		upgrader:       u,
		maxGames:       cfg.MaxGames,
		maxSockets:     cfg.MaxSockets,
		gameCfg:        cfg.GameCfg,
		socketCfg:      cfg.SocketCfg,
		sockets:        make(map[game.PlayerName]messageHandler, cfg.MaxSockets),
		games:          make(map[game.ID]messageHandler, cfg.MaxGames),
		addPlayers:     make(chan playerSocket),
		socketMessages: make(chan game.Message),
		gameMessages:   make(chan game.Message),
	}
	return l
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *Lobby) AddUser(playerName game.PlayerName, w http.ResponseWriter, r *http.Request) error {
	// TODO: lock to ensure there are not too many players
	conn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s := l.socketCfg.NewSocket(conn, playerName)
	ps := playerSocket{
		PlayerName: playerName,
		Socket:     s,
	}
	l.addPlayers <- ps
	return nil
}

// RemoveUser removes the user from the lobby and a game, if any
func (l *Lobby) RemoveUser(playerName game.PlayerName) {
	l.socketMessages <- game.Message{
		Type: game.PlayerDelete,
	}
}

// Run runs the lobby
func (l *Lobby) Run(done <-chan struct{}) {
	go func() {
		defer func() {
			for _, gmh := range l.games {
				gmh.done <- struct{}{}
			}
			for _, smh := range l.sockets {
				smh.done <- struct{}{} // readMessages
				smh.done <- struct{}{} // writeMessages
			}
		}()
		for {
			select {
			case <-done:
				return
			case ps := <-l.addPlayers:
				l.addSocket(ps)
			case m := <-l.socketMessages:
				if l.debug {
					l.log.Printf("lobby reading socket message with type %v", m.Type)
				}
				switch m.Type {
				case game.Create:
					l.createGame(m)
				case game.Infos:
					l.handleGameInfos(m)
				case game.PlayerDelete:
					delete(l.sockets, m.PlayerName)
				default:
					l.sendGameMessage(m)
				}
			case m := <-l.gameMessages:
				if l.debug {
					l.log.Printf("lobby reading game message with type %v", m.Type)
				}
				l.sendSocketMessage(m)
			}
		}
	}()
}

func (l *Lobby) createGame(m game.Message) {
	if len(l.games) == l.maxGames {
		l.sendSocketErrorMessage(m, fmt.Sprintf("the maximum number of games have already been created (%v)", l.maxGames))
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
	done := make(chan struct{}, 2)
	writeMessages := make(chan game.Message)
	g.Run(done, writeMessages, l.gameMessages)
	gmh := messageHandler{
		done:          done,
		writeMessages: writeMessages,
	}
	l.games[id] = gmh
	writeMessages <- game.Message{
		Type:       game.Join,
		PlayerName: m.PlayerName,
	}
}

func (l *Lobby) handleGameInfos(m game.Message) {
	smh, ok := l.sockets[m.PlayerName]
	if !ok {
		l.sendSocketErrorMessage(m, fmt.Sprintf("no socket to send game infos to for playerName=%v", m.PlayerName))
		return
	}
	infosC := make(chan game.Info)
	for _, gmh := range l.games {
		gmh.writeMessages <- game.Message{
			Type:         game.Infos,
			PlayerName:   m.PlayerName,
			GameInfoChan: infosC,
		}
	}
	infos := make([]game.Info, len(l.games))
	i := 0
	for range l.games {
		select {
		case gameInfo := <-infosC:
			infos[i] = gameInfo
			i++
		}
	}
	close(infosC)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt < infos[j].CreatedAt
	})
	smh.writeMessages <- game.Message{
		Type:      game.Infos,
		GameInfos: infos,
	}
}

// addSocket adds the playerSocket to the socket messageHandlers and returns the merged inbound message and error channels
func (l *Lobby) addSocket(ps playerSocket) {
	done := make(chan struct{}, 2)
	writeMessages := ps.Socket.Run(done, l.socketMessages)
	mh := messageHandler{
		done:          done,
		writeMessages: writeMessages,
	}
	if _, ok := l.sockets[ps.PlayerName]; ok {
		l.log.Printf("message handler for %v already exists, replacing", ps.PlayerName)
		l.removeSocket(ps.PlayerName)
	}
	l.sockets[ps.PlayerName] = mh
}

// removeSocket removes a socket from the messageHandlers
func (l *Lobby) removeSocket(pn game.PlayerName) {
	mh, ok := l.sockets[pn]
	if !ok {
		l.log.Printf("no socket to remove for %v", pn)
		return
	}
	delete(l.sockets, pn)
	mh.done <- struct{}{} // readMessages
	mh.done <- struct{}{} // writeMessages
}

// sends a message to the game with the id specified in the message's GameID field
func (l *Lobby) sendGameMessage(m game.Message) {
	gmh, ok := l.games[m.GameID]
	if !ok {
		l.sendSocketErrorMessage(m, fmt.Sprintf("no game with id %v, please refresh games", m.GameID))
		return
	}
	gmh.writeMessages <- m
}

func (l *Lobby) sendSocketMessage(m game.Message) {
	if m.Type == game.Delete && m.GameID != 0 {
		delete(l.games, m.GameID)
		return
	}
	smh, ok := l.sockets[m.PlayerName]
	if !ok {
		l.log.Printf("no socket for player named '%v' to send message to: %v", m.PlayerName, m)
		return
	}
	smh.writeMessages <- m
}

// sendSocketErrorMessage sends the info to the player of the specified message
func (l *Lobby) sendSocketErrorMessage(m game.Message, info string) {
	smh, ok := l.sockets[m.PlayerName]
	if !ok {
		l.log.Printf("no socket for player named '%v' to send message to: %v", m.PlayerName, m)
		return
	}
	smh.writeMessages <- game.Message{
		Type:       game.SocketError,
		PlayerName: m.PlayerName,
		Info:       info,
	}
}
