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
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Manager handles sending messages to different sockets.
	// The manager allows for players to open multiple sockets, but multiple sockets cannot play in the same game before first leaving.
	Manager struct {
		runner.Runner
		upgrader      Upgrader
		playerSockets map[player.Name]map[net.Addr]chan<- message.Message
		playerGames   map[player.Name]map[game.ID]net.Addr
		socketOut     chan message.Message
		ManagerConfig
	}

	// ManagerConfig is used to create a socket Manager.
	ManagerConfig struct {
		// Log is used to log errors and other information
		Log *log.Logger
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

// NewManager creates a new socket manager from the config.
func (cfg ManagerConfig) NewManager() (*Manager, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating socket manager: validation: %w", err)
	}
	u := newGorillaUpgrader()
	sm := Manager{
		upgrader:      u,
		playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message, cfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]net.Addr),
		ManagerConfig: cfg,
	}
	return &sm, nil
}

// validate ensures the configuration has no errors.
func (cfg ManagerConfig) validate() error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case cfg.MaxPlayerSockets < 1:
		return fmt.Errorf("each player must be able to open at least one socket")
	case cfg.MaxSockets < cfg.MaxPlayerSockets:
		return fmt.Errorf("players cannot create more sockets than the manager allows")
	}
	return nil
}

// Run consumes messages from the message channel.  This channel is used to create sockets and send messages from games to them.
// The messages recieved from sockets are sent on an "out" channel to be read by games.
func (sm *Manager) Run(ctx context.Context, in <-chan message.Message) (<-chan message.Message, error) {
	if err := sm.Runner.Run(); err != nil {
		return nil, fmt.Errorf("running socket manager: %v", err)
	}
	sm.socketOut = make(chan message.Message)
	out := make(chan message.Message)
	go func() {
		defer close(out)
		defer sm.Runner.Finish()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				sm.handleGameMessage(ctx, m)
			case m := <-sm.socketOut:
				sm.handleSocketMessage(ctx, m, out)
			}
		}
	}()
	return out, nil
}

// AddSocket runs and adds a socket for the player to the manager.
func (sm *Manager) AddSocket(ctx context.Context, pn player.Name, w http.ResponseWriter, r *http.Request) error {
	if !sm.Runner.IsRunning() {
		return fmt.Errorf("socket manager not running")
	}
	if sm.numSockets() >= sm.MaxSockets {
		return fmt.Errorf("no room for another socket")
	}
	if len(sm.playerSockets[pn]) >= sm.MaxPlayerSockets {
		return fmt.Errorf("player has reached quota of sockets, close an existing one")
	}
	conn, err := sm.upgrader.Upgrade(w, r)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s, err := sm.SocketConfig.NewSocket(pn, conn)
	if err != nil {
		return fmt.Errorf("creating socket in manager: %v", err)
	}
	socketIn := make(chan message.Message)
	if err := s.Run(ctx, socketIn, sm.socketOut); err != nil {
		return fmt.Errorf("running socket")
	}
	if sm.hasSocket(s.Addr) {
		return fmt.Errorf("socket already exists with address of %v", s.Addr)
	}
	playerSockets, ok := sm.playerSockets[pn]
	switch {
	case ok:
		playerSockets[s.Addr] = socketIn
	default:
		sm.playerSockets[pn] = map[net.Addr]chan<- message.Message{
			s.Addr: socketIn,
		}
	}
	return nil
}

// numSockets sums the number of sockets for each player.  Not thread safe.
func (sm *Manager) numSockets() int {
	numSockets := 0
	for _, sockets := range sm.playerSockets {
		numSockets += len(sockets)
	}
	return numSockets
}

// hasSocket determines if a socket exists in the manager with the same address.  Not thread safe.
func (sm *Manager) hasSocket(a net.Addr) bool {
	for _, sockets := range sm.playerSockets {
		for a0 := range sockets {
			if a0 == a {
				return true
			}
		}
	}
	return false
}

// handleGameMessage writes the message to the appropriate sockets in the manager.
func (sm *Manager) handleGameMessage(ctx context.Context, m message.Message) {
	switch m.Type {
	case message.Infos:
		sm.sendMessageInfos(ctx, m)
	case message.SocketError:
		sm.sendSocketError(ctx, m)
	case message.PlayerDelete:
		sm.leaveGame(ctx, m)
	default:
		sm.sendMessageForGame(ctx, m)
	}
}

// handleSocketMessage writes the socket message to to the out channel, possibly taking action.
func (sm *Manager) handleSocketMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	if len(m.PlayerName) == 0 {
		sm.Log.Printf("Received message without player name (%v).  Cannot send message back.", m.PlayerName)
		return
	}
	if m.Addr == nil {
		sm.Log.Printf("Received message without player address (%v) for %v.  Cannot send message back.", m.Addr, m.PlayerName)
		return
	}
	socketAddrs, ok := sm.playerSockets[m.PlayerName]
	if !ok {
		sm.Log.Printf("Received message from socket from unknown player (%v).  Cannot send message back.", m.PlayerName)
		return
	}
	_, ok = socketAddrs[m.Addr]
	if !ok {
		sm.Log.Printf("Received message from '%v' from unknown address: %v.  Cannot send message back.", m.PlayerName, m.Addr)
		return
	}
	if m.Game == nil {
		sm.Log.Printf("Received message without game: %v", m)
		return
	}
	switch m.Type {
	case message.Create, message.Join:
		// NOOP
	default:
		games, ok := sm.playerGames[m.PlayerName]
		switch {
		case !ok:
			sm.Log.Printf("player %v at %v not playering any game,", m.PlayerName, m.Addr)
			return
		default:
			addr, ok := games[m.Game.ID]
			if !ok {
				sm.Log.Printf("Player %v at %v not in game %v", m.PlayerName, m.Addr, m.Game.ID)
				return
			}
			if addr != m.Addr {
				sm.Log.Printf("Player %v at %v playing game %v on a different socket (%v)", m.PlayerName, m.Addr, m.Game.ID, addr)
				return
			}
		}
	}
	switch m.Type {
	case message.Join:
		sm.joinGame(ctx, m, out)
	case message.PlayerDelete:
		sm.removeSocket(ctx, m, out)
	case message.Leave:
		sm.leaveGame(ctx, m)
	default:
		out <- m
	}
}

// sendMessageInfos sends the game message with infos to sockets.
func (sm *Manager) sendMessageInfos(ctx context.Context, m message.Message) {
	switch len(m.PlayerName) {
	case 0:
		// send to all sockets (likely game info change)
		for _, addrs := range sm.playerSockets {
			for _, socketIn := range addrs {
				socketIn <- m
			}
		}
	default:
		// only send to the sockets for the player that are not in a game.
		addrsInGame := make(map[net.Addr]struct{}, 1)
		games := sm.playerGames[m.PlayerName]
		for _, addr := range games {
			addrsInGame[addr] = struct{}{}
		}
		addrs := sm.playerSockets[m.PlayerName]
		for addr, socketIn := range addrs {
			if _, ok := addrsInGame[addr]; !ok {
				socketIn <- m
			}
		}
	}
}

// sendSocketError sends the game socket message to a specific socket if possible or all sockets for the player
func (sm *Manager) sendSocketError(ctx context.Context, m message.Message) {
	switch {
	case m.Game != nil:
		sm.sendMessageForGame(ctx, m)
	default:
		// TODO: when does an error message need to be sent to all sockets?  Could the addr be preserved?
		socketAddrs, ok := sm.playerSockets[m.PlayerName]
		if !ok {
			return
		}
		for _, socketIn := range socketAddrs {
			socketIn <- m
		}
	}
}

// sendMessageForGame sends the game message to the player at the address, if possible.
func (sm *Manager) sendMessageForGame(ctx context.Context, m message.Message) {
	if m.Game == nil {
		sm.Log.Printf("no 'game' to send game message for in %v", m)
	}
	switch m.Type {
	case message.Leave:
		defer sm.leaveGame(ctx, m)
	}
	games, ok := sm.playerGames[m.PlayerName]
	if !ok {
		return
	}
	addr, ok := games[m.Game.ID]
	if !ok {
		return
	}
	socketAddrs, ok := sm.playerSockets[m.PlayerName]
	if !ok {
		sm.Log.Printf("could not send game socket error to %v, socket addrs not found - message: (%v)", m.PlayerName, m)
		return
	}
	socketIn, ok := socketAddrs[addr]
	if !ok {
		sm.Log.Printf("could not send game socket error to %v at %v - message: (%v)", m.PlayerName, addr, m)
		return
	}
	socketIn <- m
}

// joinGame adds the socket to the game.
// If the socket is in a different game, that game is left.
// If a different socket is in the game for the player, that socket leaves the game.
func (sm *Manager) joinGame(ctx context.Context, m message.Message, out chan<- message.Message) {
	games, ok := sm.playerGames[m.PlayerName]
	switch {
	case !ok:
		games = make(map[game.ID]net.Addr, 1)
		sm.playerGames[m.PlayerName] = games
	default:
		// remove other addr from the game
		addr2, ok := games[m.Game.ID]
		if ok {
			if m.Addr == addr2 {
				return // do not rejoin the game if already joined
			}
			m2 := message.Message{
				Type: message.Leave,
				Info: "leaving game because it is being played on a different socket",
			}
			socketIns := sm.playerSockets[m.PlayerName][addr2]
			socketIns <- m2
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
	out <- m
}

// removeSocket removes the socket from the manager.
func (sm *Manager) removeSocket(ctx context.Context, m message.Message, out chan<- message.Message) {
	delete(sm.playerSockets[m.PlayerName], m.Addr)
	if len(sm.playerSockets[m.PlayerName]) == 0 {
		delete(sm.playerSockets, m.PlayerName)
	}
	sm.leaveGame(ctx, m)
	out <- m
}

// leaveGame removes the socket from any game it is in.
func (sm *Manager) leaveGame(ctx context.Context, m message.Message) {
	delete(sm.playerGames[m.PlayerName], m.Game.ID)
	if len(sm.playerGames[m.PlayerName]) == 0 {
		delete(sm.playerGames, m.PlayerName)
	}
}
