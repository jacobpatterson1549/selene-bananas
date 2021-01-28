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
	// Manager runs games.
	Manager struct {
		// games maps game ids to the channel each games listens to for incoming messages
		// OutChannels are stored here because the Manager writes to the game, which in turn reads from the Manager's channel as an InChannel
		games map[game.ID]message.OutChannel
		// lastID is the ID of themost recently created game.  The next new game should get a larger ID.
		lastID game.ID
		// ManagerConfig contains configruation properties of the Manager.
		ManagerConfig
	}

	// ManagerConfig is used to create a game Manager.
	ManagerConfig struct {
		// Log is used to log errors and other information
		Log *log.Logger
		// The maximum number of games.
		MaxGames int
		// The config for creating new games.
		GameConfig Config
	}
)

// NewManager creates a new game manager from the config.
func (cfg ManagerConfig) NewManager() (*Manager, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating game manager: validation: %w", err)
	}
	m := Manager{
		games:         make(map[game.ID]message.OutChannel, cfg.MaxGames),
		ManagerConfig: cfg,
	}
	return &m, nil
}

// Run consumes messages from the "in" channel, processing them on a new goroutine until the "in" channel closes.
// The results of messages are sent on the "out" channel to be read by the subscriber.
func (gm *Manager) Run(ctx context.Context, in message.InChannel) (out message.InChannel) {
	outC := make(chan message.Message)
	go func() {
		defer close(outC)
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				gm.handleMessage(ctx, m, outC)
			}
		}
	}()
	return outC
}

// validate ensures the configuration has no errors.
func (cfg ManagerConfig) validate() error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case cfg.MaxGames < 1:
		return fmt.Errorf("must be able to create at least one game")
	}
	return nil
}

// handleMessage takes appropriate actions for different message types.
func (gm *Manager) handleMessage(ctx context.Context, m message.Message, out message.OutChannel) {
	switch m.Type {
	case message.Create:
		gm.createGame(ctx, m, out)
	case message.Delete:
		gm.deleteGame(ctx, m, out)
	default:
		gm.handleGameMessage(ctx, m, out)
	}
}

// createGame allocates a new game, adding it to the open games.
func (gm *Manager) createGame(ctx context.Context, m message.Message, out message.OutChannel) {
	if len(gm.games) >= gm.MaxGames {
		err := fmt.Errorf("the maximum number of games have already been created (%v)", gm.MaxGames)
		gm.sendError(err, m.PlayerName, out)
		return

	}
	id := gm.lastID + 1
	g, err := gm.GameConfig.NewGame(id)
	if err != nil {
		gm.sendError(err, m.PlayerName, out)
		return
	}
	gm.lastID = id
	in := make(chan message.Message)
	go g.Run(ctx, in, out) // all games publish to the same "out" channel
	gm.games[id] = in
	m.Type = message.Create
	in <- m
}

// deleteGame removes a game from the manager, notifying the game that it is being deleted so it can notify users.
func (gm *Manager) deleteGame(ctx context.Context, m message.Message, out message.OutChannel) {
	gIn, err := gm.getGame(m)
	if err != nil {
		gm.sendError(err, m.PlayerName, out)
		return
	}
	delete(gm.games, m.Game.ID)
	gIn <- m
}

// handleGameMessage passes an error to the game the message is for.
func (gm *Manager) handleGameMessage(ctx context.Context, m message.Message, out message.OutChannel) {
	gIn, err := gm.getGame(m)
	if err != nil {
		gm.sendError(err, m.PlayerName, out)
		return
	}
	gIn <- m
}

// getGame retrieves the game from the manager for the message, if the manager has a game for the message's game ID.
func (gm Manager) getGame(m message.Message) (message.OutChannel, error) {
	if m.Game == nil {
		return nil, errors.New("no game for manager to handle in message")
	}
	gIn, ok := gm.games[m.Game.ID]
	if !ok {
		return nil, errors.New("no game ID for manager to handle in message")
	}
	return gIn, nil
}

// sendError adds a message for the player on the channel
func (gm *Manager) sendError(err error, pn player.Name, out message.OutChannel) {
	err = fmt.Errorf("player %v: %w", pn, err)
	gm.Log.Print(err)
	m := message.Message{
		Type:       message.SocketError,
		Info:       err.Error(),
		PlayerName: pn,
	}
	out <- m
}
