package socket

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Runner handles sending messages to different sockets.
	// The runner allows for players to open multiple sockets, but multiple sockets cannot play in the same game before first leaving.
	Runner struct {
		log           *log.Logger
		upgrader      Upgrader
		playerSockets map[player.Name]map[net.Addr]chan<- message.Message
		playerGames   map[player.Name]map[game.ID]net.Addr
		socketOut     chan message.Message
		RunnerConfig
	}

	// RunnerConfig is used to create a socket Runner.
	RunnerConfig struct {
		// Debug is a flag that causes the game to log the types messages that are read.
		Debug bool
		// The maximum number of sockets.
		MaxSockets int
		// The maximum number of sockets each player can open.  Must be no more than maxSockets.
		MaxPlayerSockets int
		// The config for creating new sockets
		SocketConfig Config
	}

	// Upgrader turns a http request into a websocket.
	Upgrader interface {
		// Upgrade creates a Conn from the HTTP request.
		Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error)
	}
)

// NewRunner creates a new socket runner from the config.
func (cfg RunnerConfig) NewRunner(log *log.Logger) (*Runner, error) {
	if err := cfg.validate(log); err != nil {
		return nil, fmt.Errorf("creating socket runner: validation: %w", err)
	}
	u := newGorillaUpgrader()
	r := Runner{
		log:           log,
		upgrader:      u,
		playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message, cfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]net.Addr),
		RunnerConfig:  cfg,
	}
	return &r, nil
}

// validate ensures the configuration has no errors.
func (cfg RunnerConfig) validate(log *log.Logger) error {
	switch {
	case log == nil:
		return fmt.Errorf("log required")
	case cfg.MaxPlayerSockets < 1:
		return fmt.Errorf("each player must be able to open at least one socket")
	case cfg.MaxSockets < cfg.MaxPlayerSockets:
		return fmt.Errorf("players cannot create more sockets than the runner allows")
	}
	return nil
}

// Run consumes messages from the message channel.  This channel is used to create sockets and send messages from games to them.
// The messages recieved from sockets are sent on an "out" channel to be read by games.
func (r *Runner) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	r.socketOut = make(chan message.Message)
	out := make(chan message.Message)
	go func() {
		defer func() {
			for pn := range r.playerSockets {
				m := message.Message{
					PlayerName: pn,
				}
				r.removePlayer(ctx, m) // close the socket
			}
			close(out)
		}()
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				r.handleLobbyMessage(ctx, m)
			case m := <-r.socketOut:
				r.handleSocketMessage(ctx, m, out)
			}
		}
	}()
	return out
}

// handleAddSocket adds a socket and sends a response if on the result channel of the request.
func (r *Runner) addSocket(ctx context.Context, m message.Message) {
	switch {
	case m.AddSocketRequest == nil:
		r.log.Printf("no AddSocketRequest on message: %v", m)
		return
	case m.AddSocketRequest.Result == nil:
		r.log.Printf("no AddSocketRequest Result channel on message: %v", m)
		return
	}
	s, err := r.handleAddSocket(ctx, m.PlayerName, m.AddSocketRequest.ResponseWriter, m.AddSocketRequest.Request)
	m2 := message.Message{
		PlayerName: m.PlayerName,
	}
	switch {
	case err != nil:
		m2.Type = message.SocketError
		m2.Info = err.Error()
	default:
		m2.Type = message.GameInfos
		m2.Addr = s.Addr
	}
	message.Send(m2, m.AddSocketRequest.Result, r.Debug, r.log)
}

// handleAddSocket runs and adds a socket for the player to the runner.
func (r *Runner) handleAddSocket(ctx context.Context, pn player.Name, w http.ResponseWriter, req *http.Request) (*Socket, error) {
	if r.numSockets() >= r.MaxSockets {
		return nil, fmt.Errorf("no room for another socket")
	}
	if len(pn) == 0 {
		return nil, fmt.Errorf("player name required")
	}
	if len(r.playerSockets[pn]) >= r.MaxPlayerSockets {
		return nil, fmt.Errorf("player has reached quota of sockets, close an existing one")
	}
	conn, err := r.upgrader.Upgrade(w, req)
	if err != nil {
		return nil, fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s, err := r.SocketConfig.NewSocket(r.log, pn, conn)
	if err != nil {
		return nil, fmt.Errorf("creating socket in runner: %v", err)
	}
	socketIn := make(chan message.Message)
	s.Run(socketIn, r.socketOut)
	if r.hasSocket(s.Addr) {
		return nil, fmt.Errorf("socket already exists with address of %v", s.Addr)
	}
	playerSockets, ok := r.playerSockets[pn]
	switch {
	case ok:
		playerSockets[s.Addr] = socketIn
	default:
		r.playerSockets[pn] = map[net.Addr]chan<- message.Message{
			s.Addr: socketIn,
		}
	}
	return s, nil
}

// numSockets sums the number of sockets for each player.  Not thread safe.
func (r *Runner) numSockets() int {
	numSockets := 0
	for _, sockets := range r.playerSockets {
		numSockets += len(sockets)
	}
	return numSockets
}

// hasSocket determines if a socket exists in the runner with the same address.  Not thread safe.
func (r *Runner) hasSocket(a net.Addr) bool {
	for _, sockets := range r.playerSockets {
		for a0 := range sockets {
			if a0 == a {
				return true
			}
		}
	}
	return false
}

// handleLobbyMessage writes the message to the appropriate sockets in the runner.
func (r *Runner) handleLobbyMessage(ctx context.Context, m message.Message) {
	switch m.Type {
	case message.GameInfos:
		r.sendGameInfos(ctx, m)
	case message.SocketError:
		r.sendSocketError(ctx, m)
	case message.PlayerRemove:
		r.removePlayer(ctx, m)
	case message.SocketAdd:
		r.addSocket(ctx, m)
	default:
		r.sendMessageForGame(ctx, m)
	}
}

// validateSocketMessage returns an error if the message from a socket is invalid
func (r *Runner) validateSocketMessage(m message.Message) error {
	if len(m.PlayerName) == 0 {
		return fmt.Errorf("no player name, cannot send message back")
	}
	if m.Addr == nil {
		return fmt.Errorf("no player address, cannot send message back")
	}
	socketAddrs, ok := r.playerSockets[m.PlayerName]
	if !ok {
		return fmt.Errorf("unknown player (%v), cannot send message back", m.PlayerName)
	}
	_, ok = socketAddrs[m.Addr]
	if !ok {
		return fmt.Errorf("unknown address: %v, cannot send message back", m.Addr)
	}
	if m.Game == nil && m.Type != message.SocketClose {
		return fmt.Errorf("received message without game")
	}
	switch m.Type {
	case message.CreateGame, message.JoinGame, message.SocketClose:
		// NOOP
	default:
		games, ok := r.playerGames[m.PlayerName]
		switch {
		case !ok:
			return fmt.Errorf("player %v at %v not playing any game,", m.PlayerName, m.Addr)
		default:
			addr, ok := games[m.Game.ID]
			if !ok {
				return fmt.Errorf("player %v at %v not in game %v", m.PlayerName, m.Addr, m.Game.ID)
			}
			if addr != m.Addr {
				return fmt.Errorf("player %v at %v playing game %v on a different socket (%v)", m.PlayerName, m.Addr, m.Game.ID, addr)
			}
		}
	}
	return nil
}

// handleSocketMessage writes the socket message to to the out channel, possibly taking action.
func (r *Runner) handleSocketMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	if err := r.validateSocketMessage(m); err != nil {
		r.log.Printf("invalid message from socket: %v: %v", err, m)
		return
	}
	switch m.Type {
	case message.SocketClose:
		r.removeSocket(ctx, m)
	case message.LeaveGame:
		r.leaveGame(ctx, m)
	default:
		message.Send(m, out, r.Debug, r.log)
	}
}

// sendGameInfos sends the game message with infos to the single socket or all.
// When a socket is added, only it immediately needs game infos.  Otherwise, when any game info changes, all sockets must be notified.
func (r *Runner) sendGameInfos(ctx context.Context, m message.Message) {
	switch {
	case m.Addr != nil:
		addrs, ok := r.playerSockets[m.PlayerName]
		if !ok {
			r.log.Printf("no player to send infos to for %v", m)
			return
		}
		socketIn, ok := addrs[m.Addr]
		if !ok {
			r.log.Printf("no socket for %v at %v", m.PlayerName, m.Addr)
			return
		}
		message.Send(m, socketIn, r.Debug, r.log)
	default:
		// send to all sockets (likely game info change)
		for _, addrs := range r.playerSockets {
			for _, socketIn := range addrs {
				message.Send(m, socketIn, r.Debug, r.log)
			}
		}
	}
}

// sendSocketError sends the game socket message to a specific socket if possible or all sockets for the player
func (r *Runner) sendSocketError(ctx context.Context, m message.Message) {
	r.log.Printf("socket error: %v", m)
	switch {
	case m.Game != nil:
		r.sendMessageForGame(ctx, m)
	default:
		socketAddrs := r.playerSockets[m.PlayerName]
		for _, socketIn := range socketAddrs {
			message.Send(m, socketIn, r.Debug, r.log)
		}
	}
}

// sendMessageForGame sends the game message to the player at the address, if possible.
func (r *Runner) sendMessageForGame(ctx context.Context, m message.Message) {
	if m.Game == nil {
		r.log.Printf("no 'game' to send game message for in %v", m)
		return
	}
	socketAddrs, ok := r.playerSockets[m.PlayerName]
	if !ok {
		r.log.Printf("could not send game message to %v, socket addrs not found - message: (%v)", m.PlayerName, m)
		return
	}
	var addr net.Addr
	switch m.Type {
	case message.JoinGame:
		addr = m.Addr
	case message.LeaveGame:
		defer r.leaveGame(ctx, m)
		fallthrough
	default:
		games, gOk := r.playerGames[m.PlayerName]
		if !gOk {
			return // don't worry if player not connected
		}
		var aOk bool
		addr, aOk = games[m.Game.ID]
		if !aOk {
			return // don't worry if player not observing game
		}
	}
	socketIn, ok := socketAddrs[addr]
	if !ok {
		r.log.Printf("could not send game message to %v at %v - message: (%v)", m.PlayerName, addr, m)
		return
	}
	switch m.Type {
	case message.JoinGame:
		r.joinGame(ctx, m, socketIn)
	default:
		message.Send(m, socketIn, r.Debug, r.log)
	}
}

// joinGame adds the socket to the game.
// If the socket is in a different game, that game is left.
// If a different socket is in the game for the player, that socket leaves the game.
func (r *Runner) joinGame(ctx context.Context, m message.Message, out chan<- message.Message) {
	games, ok := r.playerGames[m.PlayerName]
	switch {
	case !ok:
		games = make(map[game.ID]net.Addr, 1)
		r.playerGames[m.PlayerName] = games
	default:
		// remove other addr from the game
		addr2, ok := games[m.Game.ID]
		if ok {
			if m.Addr == addr2 {
				return // do not rejoin the game if already joined
			}
			m2 := message.Message{
				Type: message.LeaveGame,
				Info: "leaving game because it is being played on a different socket",
			}
			socketIn2 := r.playerSockets[m.PlayerName][addr2]
			message.Send(m2, socketIn2, r.Debug, r.log)
		}
		// remove the addr from its previously joined game if it is different
		for id, addr := range games {
			if addr == m.Addr {
				delete(games, id)
				break
			}
		}
	}
	games[m.Game.ID] = m.Addr
	message.Send(m, out, r.Debug, r.log)
}

// removeSocket removes the socket from the runner.
func (r *Runner) removeSocket(ctx context.Context, m message.Message) {
	if socketIn, ok := r.playerSockets[m.PlayerName][m.Addr]; ok {
		close(socketIn)
	}
	delete(r.playerSockets[m.PlayerName], m.Addr)
	if len(r.playerSockets[m.PlayerName]) == 0 {
		delete(r.playerSockets, m.PlayerName)
	}
	r.leaveGame(ctx, m)
}

// leaveGame removes the socket from any game it is in.
func (r *Runner) leaveGame(ctx context.Context, m message.Message) {
	playerGames, ok := r.playerGames[m.PlayerName]
	if !ok {
		return
	}
	switch {
	case m.Game != nil:
		delete(playerGames, m.Game.ID)
	case m.Addr != nil:
		for gID, addr := range playerGames {
			if addr == m.Addr {
				delete(playerGames, gID)
				break
			}
		}
	}
	if len(playerGames) == 0 {
		delete(r.playerGames, m.PlayerName)
	}
}

// removePlayer removes the player's sockets and games.
func (r *Runner) removePlayer(ctx context.Context, m message.Message) {
	if addrs, ok := r.playerSockets[m.PlayerName]; ok {
		for addr := range addrs {
			m2 := message.Message{
				PlayerName: m.PlayerName,
				Addr:       addr,
			}
			r.removeSocket(ctx, m2)
		}
	}
}
