// Package lobby handles players connecting to games and communication between games and players
package lobby

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
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
		gameCfg        gameController.Config
		sockets        map[player.Name]messageHandler
		games          map[game.ID]gameMessageHandler
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
		GameCfg gameController.Config
		// SocketCfg is used to create new sockets
		SocketCfg socket.Config
	}

	// playerSocket ise used to add players from http requests.
	playerSocket struct {
		playerName player.Name
		http.ResponseWriter
		*http.Request
		result chan<- error
	}

	// messageHandler is a channel that can write game messages and be cancelled.
	messageHandler struct {
		writeMessages chan<- game.Message
		context.CancelFunc
	}

	// gameMessageHandler is a messageHandler with a game info field.
	gameMessageHandler struct {
		info game.Info
		messageHandler
	}
)

// NewLobby creates a new game lobby.
func (cfg Config) NewLobby() (*Lobby, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
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
		sockets:        make(map[player.Name]messageHandler, cfg.MaxSockets),
		games:          make(map[game.ID]gameMessageHandler, cfg.MaxGames),
		addSockets:     make(chan playerSocket),
		socketMessages: make(chan game.Message),
		gameMessages:   make(chan game.Message),
	}
	return &l, nil
}

// validate ensures the configuration has no errors.
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

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username.
func (l *Lobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	playerName := player.Name(username)
	result := make(chan error)
	ps := playerSocket{
		playerName:     playerName,
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

// RemoveUser removes the user from the lobby and a game, if any.
func (l *Lobby) RemoveUser(username string) {
	playerName := player.Name(username)
	l.socketMessages <- game.Message{
		Type:       game.PlayerDelete,
		PlayerName: playerName,
	}
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context) {
	for { // BLOCKING
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

// handleSocketMessage sends the message from the socket to the game or performs special handling.
func (l *Lobby) handleSocketMessage(ctx context.Context, m game.Message) {
	if l.debug {
		l.log.Printf("lobby reading socket message with type %v", m.Type)
	}
	switch m.Type {
	case game.Create:
		l.createGame(ctx, m)
	case game.PlayerDelete:
		delete(l.sockets, m.PlayerName)
	default:
		l.sendGameMessage(m)
	}
}

// handleGameMessage sends the message from the game to the socket for the player or performs special handling.
func (l *Lobby) handleGameMessage(ctx context.Context, m game.Message) {
	if l.debug {
		l.log.Printf("lobby reading game message with type %v", m.Type)
	}
	switch m.Type {
	case game.Infos:
		l.handleGameInfoChanged(m)
	default:
		l.sendSocketMessage(m)
	}
}

// createGame creates and adds the a game to the lobby if there is room.
// The player sending the message also joins it.
func (l *Lobby) createGame(ctx context.Context, m game.Message) {
	var err error
	defer func() {
		if err != nil {
			l.sendSocketMessage(game.Message{
				Type:       game.Leave,
				PlayerName: m.PlayerName,
			})
			l.sendSocketMessage(game.Message{
				Type:       game.SocketError,
				PlayerName: m.PlayerName,
				Info:       err.Error(),
			})
		}
	}()
	if len(l.games) >= l.maxGames {
		err = fmt.Errorf("the maximum number of games have already been created (%v)", l.maxGames)
		return
	}
	var id game.ID = 1
	for existingID := range l.games {
		if existingID != id {
			break
		}
		id++
	}
	// TODO: ensure the gameConfig of the Lobby is not modified
	gameCfg := l.gameCfg
	gameCfg.WordsConfig = gameController.WordsConfig{ // TODO: hack, should be passed in as pointer to struct in message
		CheckOnSnag:       m.CheckOnSnag,
		Penalize:          m.Penalize,
		MinLength:         m.MinLength,
		AllowDuplicates:   m.AllowDuplicates,
		FinishedAllowMove: m.FinishedAllowMove,
	}
	g, err := gameCfg.NewGame(id)
	if err != nil {
		return
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	removeGameFunc := func() {
		l.removeGame(id)
		cancelFunc()
	}
	writeMessages := make(chan game.Message)
	go g.Run(ctx, removeGameFunc, writeMessages, l.gameMessages)
	mh := messageHandler{
		writeMessages: writeMessages,
		CancelFunc:    cancelFunc,
	}
	l.games[id] = gameMessageHandler{
		messageHandler: mh,
	}
	writeMessages <- game.Message{ // this will update the game's info
		Type:       game.Join,
		PlayerName: m.PlayerName,
		NumCols:    m.NumCols,
		NumRows:    m.NumRows,
	}
}

// removeGame removes a game from the messageHandlers.
func (l *Lobby) removeGame(id game.ID) {
	mh, ok := l.games[id]
	if !ok {
		l.log.Printf("no game to remove with id %v", id)
		return
	}
	delete(l.games, id)
	mh.CancelFunc()
	l.gameInfosChanged()
}

// addSocket creates and adds the playerSocket to the socket messageHandlers and returns the merged inbound message and error channels.
func (l *Lobby) addSocket(ctx context.Context, ps playerSocket) {
	conn, err := l.upgrader.Upgrade(ps.ResponseWriter, ps.Request, nil)
	defer func() {
		if err != nil && conn != nil {
			socket.CloseConn(conn, l.log, err.Error())
		}
		ps.result <- err
	}()
	if len(l.sockets) >= l.maxSockets {
		err = fmt.Errorf("lobby full")
		return
	}
	if err != nil {
		err = fmt.Errorf("upgrading to websocket connection: %w", err)
		return
	}
	s, err := l.socketCfg.NewSocket(conn, ps.playerName)
	if err != nil {
		err = fmt.Errorf("creating socket: %w", err)
		return
	}
	socketCtx, cancelFunc := context.WithCancel(ctx)
	removeSocketFunc := func() {
		l.removeSocket(ps.playerName)
		cancelFunc()
	}
	writeMessages := make(chan game.Message)
	go s.Run(socketCtx, removeSocketFunc, l.socketMessages, writeMessages)
	mh := messageHandler{
		writeMessages: writeMessages,
		CancelFunc:    cancelFunc,
	}
	if _, ok := l.sockets[ps.playerName]; ok {
		l.log.Printf("message handler for %v already exists, replacing", ps.playerName)
		l.removeSocket(ps.playerName)
	}
	l.sockets[ps.playerName] = mh
	infos := l.gameInfos()
	m := game.Message{
		Type:      game.Infos,
		GameInfos: infos,
	}
	writeMessages <- m
}

// removeSocket removes a socket from the messageHandlers.
func (l *Lobby) removeSocket(pn player.Name) {
	mh, ok := l.sockets[pn]
	if !ok {
		l.log.Printf("no socket to remove for %v", pn)
		return
	}
	delete(l.sockets, pn)
	mh.CancelFunc()
}

// sendGameMessage sends a message to the game with the id specified in the message's GameID field.
func (l *Lobby) sendGameMessage(m game.Message) {
	gmh, ok := l.games[m.GameID]
	if !ok {
		l.sendSocketErrorMessage(m, fmt.Sprintf("no game with id %v, please refresh games", m.GameID))
		return
	}
	gmh.writeMessages <- m
}

// sendSocketMessage sends a message to the socket for the player the message is for.
func (l *Lobby) sendSocketMessage(m game.Message) {
	smh, ok := l.sockets[m.PlayerName]
	if !ok {
		l.log.Printf("no socket for player named '%v' to send message to: %v", m.PlayerName, m)
		return
	}
	smh.writeMessages <- m
}

// sendSocketErrorMessage sends the info to the player of the specified message.
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

// gameInfos gets the gameInfos for the lobby.
func (l Lobby) gameInfos() []game.Info {
	infos := make([]game.Info, 0, len(l.games))
	for _, mh := range l.games {
		infos = append(infos, mh.info)
	}
	return infos
}

// handleGameInfo updates the game info for the game.
func (l *Lobby) handleGameInfoChanged(m game.Message) {
	if len(m.GameInfos) != 1 {
		log.Printf("wanted 1 gameInfo to have changed, got %v", len(m.GameInfos))
		return
	}
	i := m.GameInfos[0]
	mh, ok := l.games[i.ID]
	if !ok {
		l.log.Printf("no game to update info for with id %v", i.ID)
		return
	}
	mh.info = i
	l.games[i.ID] = mh
	l.gameInfosChanged()
}

// gameInfosChanged notifies all sockets that the game infos have changed by sending them the new infos.
func (l Lobby) gameInfosChanged() {
	infos := l.gameInfos()
	m := game.Message{
		Type:      game.Infos,
		GameInfos: infos,
	}
	for _, mh := range l.sockets {
		mh.writeMessages <- m
	}
}
