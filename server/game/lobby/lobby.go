// Package lobby handles players connecting to games and communication between games and players
package lobby

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

type (
	// Lobby is the place users can create, join, and participate in games.
	// It is a middleman between the socket runner and game runner.  It also handles communication from the server to add and remove sockets.
	Lobby struct {
		log          log.Logger
		socketRunner SocketRunner
		gameRunner   GameRunner
		// socketMessages is used for sending messages to the socket runner that stem from HTTP requests to add and remove sockets.
		socketMessages chan message.Socket
		// games is a cache of game infos.  This is useful so all can be easily sent out if the info for one game changes.
		games map[game.ID]game.Info
		Config
	}

	// Config contiains the properties to create a lobby
	Config struct {
		// Debug is a flag that causes the game to log the types messages that are read.
		Debug bool
	}

	// SocketRunner handles running and managing sockets.
	SocketRunner interface {
		Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message
	}

	// GameRunner handles running and managing games.
	GameRunner interface {
		Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message
	}
)

// NewLobby creates a new game lobby.
func (cfg Config) NewLobby(log log.Logger, socketRunner SocketRunner, gameRunner GameRunner) (*Lobby, error) {
	if err := cfg.validate(log, socketRunner, gameRunner); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
	}
	l := Lobby{
		log:            log,
		socketRunner:   socketRunner,
		gameRunner:     gameRunner,
		socketMessages: make(chan message.Socket),
		games:          make(map[game.ID]game.Info),
		Config:         cfg,
	}
	return &l, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(log log.Logger, socketRunner SocketRunner, gameRunner GameRunner) error {
	switch {
	case log == nil:
		return fmt.Errorf("log required")
	case socketRunner == nil:
		return fmt.Errorf("socket runner required")
	case gameRunner == nil:
		return fmt.Errorf("game runner required")
	}
	return nil
}

// Run runs the lobby until the context is closed.
func (l *Lobby) Run(ctx context.Context, wg *sync.WaitGroup) {
	gameRunnerIn := make(chan message.Message)
	socketRunnerIn := make(chan message.Message)
	socketRunnerOut := l.socketRunner.Run(ctx, wg, socketRunnerIn, l.socketMessages)
	gameRunnerOut := l.gameRunner.Run(ctx, wg, gameRunnerIn)
	wg.Add(1)
	run := func() {
		defer wg.Done()
		defer l.log.Printf("lobby stopped")
		defer close(socketRunnerIn)
		defer close(gameRunnerIn)
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m, ok := <-socketRunnerOut:
				if !ok {
					return
				}
				l.handleSocketMessage(m, gameRunnerIn, socketRunnerIn)
			case m, ok := <-gameRunnerOut:
				if !ok {
					return
				}
				l.handleGameMessage(m, socketRunnerIn)
			}
		}
	}
	go run()
}

// AddUser adds a user to the lobby, it opens a new websocket (player) for the username.
func (l *Lobby) AddUser(username string, w http.ResponseWriter, r *http.Request) error {
	result := make(chan error)
	pn := player.Name(username)
	sm := message.Socket{
		Type:           message.SocketAdd,
		PlayerName:     pn,
		ResponseWriter: w,
		Request:        r,
		Result:         result,
	}
	l.socketMessages <- sm
	err := <-result
	if err != nil {
		return err
	}
	return nil
}

// RemoveUser removes all sockets for the user from the lobby.
func (l *Lobby) RemoveUser(username string) {
	sm := message.Socket{
		Type:       message.PlayerRemove,
		PlayerName: player.Name(username),
	}
	l.socketMessages <- sm
}

// handleSocketMessage writes a socket message to the gameRunnerIn channel unless it is a gameInfos request, in which case it is sent back with infos.
func (l *Lobby) handleSocketMessage(m message.Message, gameRunnerIn, socketRunnerIn chan<- message.Message) {
	switch m.Type {
	case message.GameInfos:
		m.Games = l.gameInfos()
		message.Send(m, socketRunnerIn, l.Debug, l.log)
	default:
		message.Send(m, gameRunnerIn, l.Debug, l.log)
	}
}

// handleGameMessage writes a game message to the socketMessages channel, possibly modifying it.
func (l *Lobby) handleGameMessage(m message.Message, socketRunnerIn chan<- message.Message) {
	switch m.Type {
	case message.GameInfos:
		l.handleGameInfoChanged(m, socketRunnerIn)
	default:
		message.Send(m, socketRunnerIn, l.Debug, l.log)
	}
}

// handleGameInfo updates the game info for the game.
func (l *Lobby) handleGameInfoChanged(m message.Message, socketRunnerIn chan<- message.Message) {
	if m.Game == nil {
		m2 := message.Message{
			Type:       message.SocketError,
			Info:       "cannot update game info when no game is provided",
			PlayerName: m.PlayerName,
		}
		message.Send(m2, socketRunnerIn, l.Debug, l.log)
		l.log.Printf(m2.Info)
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
	message.Send(m2, socketRunnerIn, l.Debug, l.log)
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
