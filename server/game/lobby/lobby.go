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
)

type (
	// Lobby is the place users can create, join, and participate in games
	Lobby struct {
		socketRunner Runner
		gameRunner   Runner
		// socketRunnerIn is the channel for sending messages to the socket runner.  Used to add and remove sockets
		socketRunnerIn chan message.Message
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

	// Runner handles running and managing games or sockets.
	Runner interface {
		Run(ctx context.Context, in <-chan message.Message) <-chan message.Message
	}
)

// NewLobby creates a new game lobby.
func (cfg Config) NewLobby(socketRunner, gameRunner Runner) (*Lobby, error) {
	if err := cfg.validate(socketRunner, gameRunner); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
	}
	l := Lobby{
		socketRunner:   socketRunner,
		gameRunner:     gameRunner,
		socketRunnerIn: make(chan message.Message),
		games:          make(map[game.ID]game.Info),
		Config:         cfg,
	}
	return &l, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(socketRunner, gameRunner Runner) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case socketRunner == nil:
		return fmt.Errorf("socket runner required")
	case gameRunner == nil:
		return fmt.Errorf("game runner required")
	}
	return nil
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context) {
	gameRunnerIn := make(chan message.Message)
	socketRunnerOut := l.socketRunner.Run(ctx, l.socketRunnerIn)
	gameRunnerOut := l.gameRunner.Run(ctx, gameRunnerIn)
	go func() {
		defer close(l.socketRunnerIn)
		defer close(gameRunnerIn)
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m := <-socketRunnerOut:
				gameRunnerIn <- m
			case m := <-gameRunnerOut:
				l.handleGameMessage(m)
			}
		}
	}()
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username.
func (l *Lobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	result := make(chan message.Message)
	pn := player.Name(username)
	m := message.Message{
		Type:       message.SocketAdd,
		PlayerName: pn,
		AddSocketRequest: &message.AddSocketRequest{
			ResponseWriter: w,
			Request:        r,
			Result:         result,
		},
	}
	l.socketRunnerIn <- m
	// The result contains the address of the new socket to get the game infos of the lobby for.
	m2 := <-result
	if m2.Type == message.SocketError {
		return fmt.Errorf(m2.Info)
	}
	m2.Type = message.GameInfos
	m2.Games = l.gameInfos()
	l.socketRunnerIn <- m2
	return nil
}

// RemoveUser removes all sockets for the user from the lobby.
func (l *Lobby) RemoveUser(username string) {
	m := message.Message{
		Type:       message.PlayerRemove,
		PlayerName: player.Name(username),
	}
	l.socketRunnerIn <- m
}

// handleGameMessage writes a game message to the socketMessages channel, possibly modifying it.
func (l *Lobby) handleGameMessage(m message.Message) {
	switch m.Type {
	case message.GameInfos:
		l.handleGameInfoChanged(m)
	default:
		l.socketRunnerIn <- m
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
		l.socketRunnerIn <- m2
		l.Log.Print(m2.Info)
		return
	}
	switch m.Game.Status {
	case game.Deleted:
		delete(l.games, m.Game.ID)
	default:
		l.games[m.Game.ID] = *m.Game
	}
	infos := l.gameInfos()
	m2 := message.Message{
		Type:  message.GameInfos,
		Games: infos,
	}
	l.socketRunnerIn <- m2
}

// game infos gets the sorted game infos for the Lobby.
func (l *Lobby) gameInfos() []game.Info {
	infos := make([]game.Info, 0, len(l.games))
	for _, info := range l.games {
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}
