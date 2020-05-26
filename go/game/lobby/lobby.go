// Package lobby handles players connecting to games and communication between games and players
package lobby

import (
	"context"
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
		addSockets     chan playerSocket
		socketMessages chan game.Message
		gameMessages   chan game.Message
	}

	// Config contiains the properties to create a lobby
	Config struct {
		// Debug is a flag that causes the lobby to log the types messages that are read
		Debug bool
		// Log is used to log errors and other information
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
		http.ResponseWriter
		*http.Request
		result chan<- error
	}

	messageHandler struct {
		writeMessages chan<- game.Message
		context.CancelFunc
	}
)

// NewLobby creates a new game lobby
func (cfg Config) NewLobby() (*Lobby, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	u := new(websocket.Upgrader)
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
		addSockets:     make(chan playerSocket),
		socketMessages: make(chan game.Message),
		gameMessages:   make(chan game.Message),
	}
	return &l, nil
}

func (cfg Config) validate() error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case cfg.MaxGames <= 0:
		return fmt.Errorf("must allow at least one game")
	case cfg.MaxSockets <= 0:
		return fmt.Errorf("must allow at least one socket")
	}
	return nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username
func (l *Lobby) AddUser(playerName game.PlayerName, w http.ResponseWriter, r *http.Request) error {
	result := make(chan error)
	ps := playerSocket{
		PlayerName:     playerName,
		ResponseWriter: w,
		Request:        r,
		result:         result,
	}
	l.addSockets <- ps
	if err := <-result; err != nil {
		return err
	}
	return nil
}

// RemoveUser removes the user from the lobby and a game, if any
func (l *Lobby) RemoveUser(playerName game.PlayerName) {
	l.socketMessages <- game.Message{
		Type: game.PlayerDelete,
	}
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context) {
	for { // BLOCKS
		select {
		case <-ctx.Done():
			return
		case ps := <-l.addSockets:
			l.addSocket(ctx, ps)
		case m := <-l.socketMessages:
			l.handleSocketMessage(ctx, m)
		case m := <-l.gameMessages:
			l.handleGameMessage(ctx, m)
		}
	}
}

func (l *Lobby) handleSocketMessage(ctx context.Context, m game.Message) {
	if l.debug {
		l.log.Printf("lobby reading socket message with type %v", m.Type)
	}
	switch m.Type {
	case game.Create:
		l.createGame(ctx, m)
	case game.Infos:
		l.handleGameInfos(m)
	case game.PlayerDelete:
		delete(l.sockets, m.PlayerName)
	default:
		l.sendGameMessage(m)
	}
}

func (l *Lobby) handleGameMessage(ctx context.Context, m game.Message) {
	if l.debug {
		l.log.Printf("lobby reading game message with type %v", m.Type)
	}
	l.sendSocketMessage(m)
}

// createGame creates and adds the a game to the lobby if there is room.
// The player sending the message also joins it.
func (l *Lobby) createGame(ctx context.Context, m game.Message) {
	if len(l.games) >= l.maxGames {
		m = game.Message{
			Type:       game.SocketWarning,
			PlayerName: m.PlayerName,
			Info:       fmt.Sprintf("the maximum number of games have already been created (%v)", l.maxGames),
		}
		l.sendSocketMessage(m)
		return
	}
	var id game.ID = 1
	for existingID := range l.games {
		if existingID != id {
			break
		}
		id++
	}
	g, err := l.gameCfg.NewGame(id)
	if err != nil {
		l.sendSocketErrorMessage(m, err.Error())
		return
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	removeGameFunc := func() {
		l.removeGame(id)
		cancelFunc()
	}
	writeMessages := make(chan game.Message)
	go g.Run(ctx, removeGameFunc, writeMessages, l.gameMessages)
	gmh := messageHandler{
		writeMessages: writeMessages,
		CancelFunc:    cancelFunc,
	}
	l.games[id] = gmh
	writeMessages <- game.Message{
		Type:       game.Join,
		PlayerName: m.PlayerName,
	}
}

// removeGame removes a game from the messageHandlers
func (l *Lobby) removeGame(id game.ID) {
	mh, ok := l.games[id]
	if !ok {
		l.log.Printf("no game to remove with id %v", id)
		return
	}
	delete(l.games, id)
	mh.CancelFunc()
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

// addSocket creates and adds the playerSocket to the socket messageHandlers and returns the merged inbound message and error channels
func (l *Lobby) addSocket(ctx context.Context, ps playerSocket) {
	conn, err := l.upgrader.Upgrade(ps.ResponseWriter, ps.Request, nil)
	closeConnFunc := func(reason string) {
		if conn != nil {
			socket.CloseConn(conn, reason)
		}
		ps.result <- fmt.Errorf(reason)
	}
	if len(l.sockets) >= l.maxSockets {
		closeConnFunc(fmt.Sprintf("lobby full"))
		return
	}
	if err != nil {
		closeConnFunc(fmt.Sprintf("upgrading to websocket connection: %w", err))
		return
	}
	s, err := l.socketCfg.NewSocket(conn, ps.PlayerName)
	if err != nil {
		closeConnFunc(fmt.Sprintf("creating socket: %w", err))
		return
	}
	socketCtx, cancelFunc := context.WithCancel(ctx)
	removeSocketFunc := func() {
		l.removeSocket(ps.PlayerName)
		cancelFunc()
	}
	writeMessages := make(chan game.Message)
	go s.Run(socketCtx, removeSocketFunc, l.socketMessages, writeMessages)
	mh := messageHandler{
		writeMessages: writeMessages,
		CancelFunc:    cancelFunc,
	}
	if _, ok := l.sockets[ps.PlayerName]; ok {
		l.log.Printf("message handler for %v already exists, replacing", ps.PlayerName)
		l.removeSocket(ps.PlayerName)
	}
	l.sockets[ps.PlayerName] = mh
	ps.result <- nil
}

// removeSocket removes a socket from the messageHandlers
func (l *Lobby) removeSocket(pn game.PlayerName) {
	mh, ok := l.sockets[pn]
	if !ok {
		l.log.Printf("no socket to remove for %v", pn)
		return
	}
	delete(l.sockets, pn)
	mh.CancelFunc()
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
