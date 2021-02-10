package game

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Runner runs games.
	Runner struct {
		// games maps game ids to the channel each games listens to for incoming messages
		// OutChannels are stored here because the Runner writes to the game, which in turn reads from the Runner's channel as an InChannel
		games map[game.ID]chan<- message.Message
		// lastID is the ID of themost recently created game.  The next new game should get a larger ID.
		lastID game.ID
		// the UserDao increments user points when a game is finished.
		UserDao
		// RunnerConfig contains configruation properties of the Runner.
		RunnerConfig
	}

	// RunnerConfig is used to create a game Runner.
	RunnerConfig struct {
		// Log is used to log errors and other information
		Log *log.Logger
		// The maximum number of games.
		MaxGames int
		// The config for creating new games.
		GameConfig Config
	}
)

// NewRunner creates a new game runner from the config.
func (cfg RunnerConfig) NewRunner(ud UserDao) (*Runner, error) {
	if err := cfg.validate(ud); err != nil {
		return nil, fmt.Errorf("creating game runner: validation: %w", err)
	}
	m := Runner{
		games:        make(map[game.ID]chan<- message.Message, cfg.MaxGames),
		RunnerConfig: cfg,
		UserDao:      ud,
	}
	return &m, nil
}

// Run consumes messages from the "in" channel, processing them on a new goroutine until the "in" channel closes.
// The results of messages are sent on the "out" channel to be read by the subscriber.
func (r *Runner) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	out := make(chan message.Message)
	go func() {
		defer close(out)
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				r.handleMessage(ctx, m, out)
			}
		}
	}()
	return out
}

// validate ensures the configuration has no errors.
func (cfg RunnerConfig) validate(ud UserDao) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case ud == nil:
		return fmt.Errorf("user dao required")
	case cfg.MaxGames < 1:
		return fmt.Errorf("must be able to create at least one game")
	}
	return nil
}

// handleMessage takes appropriate actions for different message types.
func (r *Runner) handleMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	switch m.Type {
	case message.CreateGame:
		r.createGame(ctx, m, out)
	case message.DeleteGame:
		r.deleteGame(ctx, m, out)
	default:
		r.handleGameMessage(ctx, m, out)
	}
}

// createGame allocates a new game, adding it to the open games.
func (r *Runner) createGame(ctx context.Context, m message.Message, out chan<- message.Message) {
	switch {
	case len(r.games) >= r.MaxGames:
		err := fmt.Errorf("the maximum number of games have already been created (%v)", r.MaxGames)
		r.sendError(err, m.PlayerName, out)
		return
	case m.Game == nil, m.Game.Board == nil:
		err := fmt.Errorf("board config required when creating game")
		r.sendError(err, m.PlayerName, out)
		return
	}
	id := r.lastID + 1
	g, err := r.GameConfig.NewGame(id, r.UserDao)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	r.lastID = id
	in := make(chan message.Message)
	g.Run(ctx, in, out) // all games publish to the same "out" channel
	r.games[id] = in
	m.Type = message.JoinGame
	in <- m
}

// deleteGame removes a game from the runner, notifying the game that it is being deleted so it can notify users.
func (r *Runner) deleteGame(ctx context.Context, m message.Message, out chan<- message.Message) {
	gIn, err := r.getGame(m)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	delete(r.games, m.Game.ID)
	gIn <- m
}

// handleGameMessage passes an error to the game the message is for.
func (r *Runner) handleGameMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	gIn, err := r.getGame(m)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	gIn <- m
}

// getGame retrieves the game from the runner for the message, if the runner has a game for the message's game ID.
func (r *Runner) getGame(m message.Message) (chan<- message.Message, error) {
	if m.Game == nil {
		return nil, errors.New("no game for runner to handle in message")
	}
	gIn, ok := r.games[m.Game.ID]
	if !ok {
		return nil, errors.New("no game ID for runner to handle in message")
	}
	return gIn, nil
}

// sendError adds a message for the player on the channel
func (r *Runner) sendError(err error, pn player.Name, out chan<- message.Message) {
	err = fmt.Errorf("player %v: %w", pn, err)
	r.Log.Print(err)
	m := message.Message{
		Type:       message.SocketError,
		Info:       err.Error(),
		PlayerName: pn,
	}
	out <- m
}
