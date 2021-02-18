package game

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Runner runs games.
	Runner struct {
		// log is used to log errors and other information
		log *log.Logger
		// games maps game ids to the channel each games listens to for incoming messages
		// OutChannels are stored here because the Runner writes to the game, which in turn reads from the Runner's channel as an InChannel
		games map[game.ID]chan<- message.Message
		// lastID is the ID of themost recently created game.  The next new game should get a larger ID.
		lastID game.ID
		// WordChecker is used to validate players' words when they try to finish the game.
		wordChecker WordChecker
		// UserDao increments user points when a game is finished.
		userDao UserDao
		// RunnerConfig contains configuration properties of the Runner.
		RunnerConfig
	}

	// RunnerConfig is used to create a game Runner.
	RunnerConfig struct {
		// Debug is a flag that causes the game to log the types messages that are read.
		Debug bool
		// The maximum number of games.
		MaxGames int
		// The config for creating new games.
		GameConfig Config
	}

	// WordChecker checks if words are valid.
	WordChecker interface {
		Check(word string) bool
	}

	// UserDao makes changes to the stored state of users in the game
	UserDao interface {
		// UpdatePointsIncrement increments points for the specified usernames based on the userPointsIncrementFunc
		UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error
	}
)

// NewRunner creates a new game runner from the config.
func (cfg RunnerConfig) NewRunner(log *log.Logger, wordChecker WordChecker, userDao UserDao) (*Runner, error) {
	if err := cfg.validate(log, wordChecker, userDao); err != nil {
		return nil, fmt.Errorf("creating game runner: validation: %w", err)
	}
	m := Runner{
		log:          log,
		games:        make(map[game.ID]chan<- message.Message, cfg.MaxGames),
		RunnerConfig: cfg,
		wordChecker:  wordChecker,
		userDao:      userDao,
	}
	return &m, nil
}

// Run consumes messages from the "in" channel, processing them on a new goroutine until the "in" channel closes.
// The results of messages are sent on the "out" channel to be read by the subscriber.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
	out := make(chan message.Message)
	wg.Add(1)
	go func() {
		ctx, cancelFunc := context.WithCancel(ctx)
		defer wg.Done()
		defer r.log.Println("game runner stopped")
		defer close(out)
		defer cancelFunc()
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				r.handleMessage(ctx, wg, m, out)
			}
		}
	}()
	return out
}

// validate ensures the configuration has no errors.
func (cfg RunnerConfig) validate(log *log.Logger, wordChecker WordChecker, userDao UserDao) error {
	switch {
	case log == nil:
		return fmt.Errorf("log required")
	case wordChecker == nil:
		return fmt.Errorf("word checker required")
	case userDao == nil:
		return fmt.Errorf("user dao required")
	case cfg.MaxGames < 1:
		return fmt.Errorf("must be able to create at least one game")
	}
	return nil
}

// handleMessage takes appropriate actions for different message types.
func (r *Runner) handleMessage(ctx context.Context, wg *sync.WaitGroup, m message.Message, out chan<- message.Message) {
	switch m.Type {
	case message.CreateGame:
		r.createGame(ctx, wg, m, out)
	case message.DeleteGame:
		r.deleteGame(ctx, m, out)
	default:
		r.handleGameMessage(ctx, m, out)
	}
}

// createGame allocates a new game, adding it to the open games.
func (r *Runner) createGame(ctx context.Context, wg *sync.WaitGroup, m message.Message, out chan<- message.Message) {
	if err := r.validateCreateGame(m); err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	id := r.lastID + 1
	gameCfg := r.GameConfig
	gameCfg.Config = *m.Game.Config
	g, err := gameCfg.NewGame(r.log, id, r.wordChecker, r.userDao)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	r.lastID = id
	gIn := make(chan message.Message)
	g.Run(ctx, wg, gIn, out) // all games publish to the same "out" channel
	r.games[id] = gIn
	m.Type = message.JoinGame
	message.Send(m, gIn, r.Debug, r.log)
}

// validateCreateGame returns an err if the runner cannot create a new game or the message to create one is invalid.
func (r *Runner) validateCreateGame(m message.Message) error {
	switch {
	case len(r.games) >= r.MaxGames:
		return fmt.Errorf("the maximum number of games have already been created (%v)", r.MaxGames)
	case m.Game == nil, m.Game.Board == nil:
		return fmt.Errorf("board config required when creating game")
	case m.Game.Config == nil:
		return fmt.Errorf("missing config for game properties")
	}
	return nil
}

// deleteGame removes a game from the runner, notifying the game that it is being deleted so it can notify users.
func (r *Runner) deleteGame(ctx context.Context, m message.Message, out chan<- message.Message) {
	gIn, err := r.getGame(m)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	delete(r.games, m.Game.ID)
	message.Send(m, gIn, r.Debug, r.log)
}

// handleGameMessage passes an error to the game the message is for.
func (r *Runner) handleGameMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	gIn, err := r.getGame(m)
	if err != nil {
		r.sendError(err, m.PlayerName, out)
		return
	}
	message.Send(m, gIn, r.Debug, r.log)
}

// getGame retrieves the game from the runner for the message, if the runner has a game for the message's game ID.
func (r *Runner) getGame(m message.Message) (chan<- message.Message, error) {
	if m.Game == nil {
		return nil, fmt.Errorf("no game for runner to handle in message: %v", m)
	}
	gIn, ok := r.games[m.Game.ID]
	if !ok {
		return nil, fmt.Errorf("no game ID for runner to handle in message: %v", m)
	}
	return gIn, nil
}

// sendError adds a message for the player on the channel
func (r *Runner) sendError(err error, pn player.Name, out chan<- message.Message) {
	err = fmt.Errorf("player %v: %w", pn, err)
	r.log.Printf("game runner error: %v", err)
	m := message.Message{
		Type:       message.SocketError,
		Info:       err.Error(),
		PlayerName: pn,
	}
	message.Send(m, out, r.Debug, r.log)
}
