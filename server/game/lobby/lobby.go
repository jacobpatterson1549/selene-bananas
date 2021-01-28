// Package lobby handles players connecting to games and communication between games and players
package lobby

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		runMu         sync.Mutex
		running       bool
		socketManager SocketManager
		gameManager   GameManager
		// socketMessages is the channel for sending messages to the socket manager.  Used to add and remove sockets
		socketMessages chan message.Message
		// games is a cache of game infos.  This is useful so all can be easily sent out if the info for one game changes.
		games map[game.ID]game.Info
		Config
	}

	// Config contiains the properties to create a lobby
	Config struct {
		// Log is used to log errors and other information
		Log *log.Logger
		// SocketManagerConfig is used to create a socket manager
		SocketManagerConfig socket.ManagerConfig
		// GameManager is used to create a game Manager
		GameManagerConfig gameController.ManagerConfig
	}

	// messageHandler is a channel that can write message.Messages and be cancelled.
	messageHandler struct {
		writeMessages chan<- message.Message
		context.CancelFunc
	}

	// gameMessageHandler is a messageHandler with a game info field.
	gameMessageHandler struct {
		info game.Info
		messageHandler
	}

	// SocketManager handles connections from multiple players.
	SocketManager interface {
		Run(ctx context.Context, in <-chan message.Message) <-chan message.Message
	}

	// GameManager handles passing messages to multiple games.
	GameManager interface {
		Run(ctx context.Context, in <-chan message.Message) <-chan message.Message
	}
)

// NewLobby creates a new game lobby.
func (cfg Config) NewLobby() (*Lobby, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
	}
	socketManager, err := cfg.SocketManagerConfig.NewManager()
	if err != nil {
		return nil, fmt.Errorf("creating socket manager: %w", err)
	}
	gameManager, err := cfg.GameManagerConfig.NewManager()
	if err != nil {
		return nil, fmt.Errorf("creating game manager: %w", err)
	}
	l := Lobby{
		socketManager: socketManager,
		gameManager:   gameManager,
		Config:        cfg,
	}
	return &l, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate() error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	}
	return nil
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context) error {
	l.runMu.Lock()
	defer l.runMu.Unlock()
	if l.running {
		return fmt.Errorf("lobby already running or has finished running, it can only be run once")
	}
	l.running = true
	l.socketMessages = make(chan message.Message)
	gameMessages := make(chan message.Message)
	l.games = make(map[game.ID]game.Info) //  this be sized to GameManager.Config.MaxGames. oh well.
	defer close(l.socketMessages)
	defer close(gameMessages)
	socketMessagesOut := l.socketManager.Run(ctx, l.socketMessages)
	gameMessagesOut := l.gameManager.Run(ctx, gameMessages)
	go func() {
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m := <-socketMessagesOut:
				gameMessages <- m
			case m := <-gameMessagesOut:
				l.handleGameMessage(m)
			}
		}
	}()
	return nil
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username.
func (l *Lobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	if !l.running {
		return fmt.Errorf("lobby not running")
	}
	result := make(chan error)
	m := message.Message{
		Type:       message.AddSocket,
		PlayerName: player.Name(username),
		AddSocketRequest: message.AddSocketRequest{
			ResponseWriter: w,
			Request:        r,
			Result:         result,
		},
	}
	l.socketMessages <- m
	if err := <-result; err != nil {
		return err
	}
	return nil
}

// RemoveUser removes the user from the lobby and a game, if any.
func (l *Lobby) RemoveUser(username string) error {
	if !l.running {
		return fmt.Errorf("lobby not running")
	}
	m := message.Message{
		Type:       message.PlayerDelete,
		PlayerName: player.Name(username),
	}
	l.socketMessages <- m
	return nil
}

// handleGameMessage writes a game message to the socketMessages channel, possibly modifying it.
func (l *Lobby) handleGameMessage(m message.Message) {
	switch m.Type {
	case message.Infos:
		l.handleGameInfoChanged(m)
	default:
		l.socketMessages <- m
	}
}

// handleGameInfo updates the game info for the game.
func (l *Lobby) handleGameInfoChanged(m message.Message) {
	if m.Game == nil {
		l.Log.Print("cannot update game info when no game is provided")
		return
	}
	l.games[m.Game.ID] = *m.Game
	l.gameInfosChanged()
}

// gameInfos gets the gameInfos for the lobby.
func (l *Lobby) gameInfos() []game.Info {
	infos := make([]game.Info, 0, len(l.games))
	for _, info := range l.games {
		infos = append(infos, info)
	}
	// TODO: add test to ensure this is sorted
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

// gameInfosChanged notifies all sockets that the game infos have changed by sending them the new infos.
func (l *Lobby) gameInfosChanged() {
	infos := l.gameInfos()
	m := message.Message{
		Type:  message.Infos,
		Games: infos,
	}
	l.socketMessages <- m // TODO: ensure this gets sent to all sockets
}
