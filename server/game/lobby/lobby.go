// Package lobby handles players connecting to games and communication between games and players
package lobby

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		runner.Runner
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
		Run(ctx context.Context, in <-chan message.Message) (<-chan message.Message, error)
	}

	// GameManager handles passing messages to multiple games.
	GameManager interface {
		Run(ctx context.Context, in <-chan message.Message) (<-chan message.Message, error)
	}
)

// NewLobby creates a new game lobby.
func (cfg Config) NewLobby(sm SocketManager, gm GameManager) (*Lobby, error) {
	if err := cfg.validate(sm, gm); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
	}
	l := Lobby{
		socketManager: sm,
		gameManager:   gm,
		games:         make(map[game.ID]game.Info),
		Config:        cfg,
	}
	return &l, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(sm SocketManager, gm GameManager) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case sm == nil:
		return fmt.Errorf("socket manager required")
	case gm == nil:
		return fmt.Errorf("game manager required")
	}
	return nil
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context) error {
	if err := l.Runner.Run(); err != nil {
		return fmt.Errorf("running lobby: %v", err)
	}
	l.socketMessages = make(chan message.Message)
	gameMessages := make(chan message.Message)
	socketMessagesOut, err := l.socketManager.Run(ctx, l.socketMessages)
	if err != nil {
		return fmt.Errorf("running socket manager: %w", err)
	}
	gameMessagesOut, err := l.gameManager.Run(ctx, gameMessages)
	if err != nil {
		return fmt.Errorf("game socket manager: %w", err)
	}
	go func() {
		defer close(l.socketMessages)
		defer close(gameMessages)
		defer l.Runner.Finish()
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
	if !l.Runner.IsRunning() {
		return fmt.Errorf("lobby not running")
	}
	result := make(chan error)
	m := message.Message{
		Type:       message.AddSocket,
		PlayerName: player.Name(username),
		AddSocketRequest: &message.AddSocketRequest{
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
	if !l.Runner.IsRunning() {
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
		m2 := message.Message{
			Type:       message.SocketError,
			Info:       "cannot update game info when no game is provided",
			PlayerName: m.PlayerName,
		}
		l.socketMessages <- m2
		l.Log.Print(m2.Info)
		return
	}
	switch m.Game.Status {
	case game.Deleted:
		delete(l.games, m.Game.ID)
	default:
		l.games[m.Game.ID] = *m.Game
	}
	infos := make([]game.Info, 0, len(l.games))
	for _, info := range l.games {
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	m2 := message.Message{
		Type:  message.Infos,
		Games: infos,
	}
	l.socketMessages <- m2
}
