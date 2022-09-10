package main

import (
	"bytes"
	"context"
	crypto_rand "crypto/rand"
	database_sql "database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/sql/postgres"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	"github.com/jacobpatterson1549/selene-bananas/server"
	"github.com/jacobpatterson1549/selene-bananas/server/auth"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/lobby"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

// CreateUserBackend creates and sets up the database to back the user DAO.
func (f Flags) CreateUserBackend(ctx context.Context, e EmbeddedData) (user.Backend, error) {
	cfg := f.sqlDatabaseConfig()
	u, err := url.Parse(f.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}
	driverName := u.Scheme
	switch driverName {
	case "postgres":
		database, err := f.CreateSQLDatabase(ctx, cfg, driverName, e)
		if err != nil {
			return nil, fmt.Errorf("creating SQL database: %w", err)
		}
		ub := postgres.UserBackend{
			Database: database,
		}
		return &ub, nil
	}
	return nil, fmt.Errorf("unsupported DATABASE_URL: %q", f.DatabaseURL)
}

func (f Flags) CreateSQLDatabase(ctx context.Context, cfg db.Config, driverName string, e EmbeddedData) (*sql.Database, error) {
	sqlDB, err := database_sql.Open(driverName, f.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	database := sql.Database{
		DB:     sqlDB,
		Config: cfg,
	}
	setupSQL, err := e.sqlFiles()
	if err != nil {
		return nil, err
	}
	if err := database.Setup(ctx, setupSQL); err != nil {
		return nil, fmt.Errorf("setting up SQL database: %w", err)
	}
	return &database, nil
}

// CreateServer creates the server.
func (f Flags) CreateServer(ctx context.Context, log log.Logger, ub user.Backend, e EmbeddedData) (*server.Server, error) {
	timeFunc := func() int64 {
		return time.Now().Unix()
	}
	key := make([]byte, 64)
	if _, err := crypto_rand.Reader.Read(key); err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	tokenizerCfg := f.tokenizerConfig(timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer(key)
	if err != nil {
		return nil, fmt.Errorf("creating authentication tokenizer: %w", err)
	}
	userDao, err := user.NewDao(ub)
	if err != nil {
		return nil, fmt.Errorf("creating user dao: %w", err)
	}
	socketRunnerCfg := f.socketRunnerConfig(timeFunc)
	socketRunner, err := socketRunnerCfg.NewRunner(log)
	if err != nil {
		return nil, fmt.Errorf("creating socket runner: %w", err)
	}
	wordsReader := bytes.NewReader(e.Words)
	wordValidator, err := word.NewValidator(wordsReader)
	if err != nil {
		return nil, fmt.Errorf("creating word validator: %v", err)
	}
	gameRunnerCfg := f.gameRunnerConfig(timeFunc)
	gameRunner, err := gameRunnerCfg.NewRunner(log, wordValidator, userDao)
	if err != nil {
		return nil, fmt.Errorf("creating game runner: %w", err)
	}
	lobbyCfg := f.lobbyConfig()
	lobby, err := lobbyCfg.NewLobby(log, socketRunner, gameRunner)
	if err != nil {
		return nil, fmt.Errorf("creating lobby: %w", err)
	}
	challenge := server.Challenge{
		Token: f.ChallengeToken,
		Key:   f.ChallengeKey,
	}
	colorCfg := f.colorConfig()
	cfg := server.Config{
		HTTPPort:      f.HTTPPort,
		HTTPSPort:     f.HTTPSPort,
		StopDur:       20 * time.Second, // should be longer than the PingPeriod of sockets so they can close gracefully
		CacheSec:      f.CacheSec,
		Version:       strings.TrimSpace(string(e.Version)),
		TLSCertPEM:    string(e.TLSCertPEM),
		TLSKeyPEM:     string(e.TLSKeyPEM),
		Challenge:     challenge,
		ColorConfig:   colorCfg,
		NoTLSRedirect: f.NoTLSRedirect,
	}
	p := server.Parameters{
		Logger:     log,
		Tokenizer:  tokenizer,
		UserDao:    userDao,
		Lobby:      lobby,
		StaticFS:   e.StaticFS,
		TemplateFS: e.TemplateFS,
	}
	return cfg.NewServer(p)
}

// colorConfig creates the color config for the css.
func (Flags) colorConfig() server.ColorConfig {
	cfg := server.ColorConfig{
		CanvasPrimary: "#000000",
		CanvasDrag:    "#0000ff",
		CanvasTile:    "#f0d0b5",
		LogError:      "#ff0000",
		LogWarning:    "#ff8000",
		LogChat:       "#008000",
		TabBackground: "#ffffc2",
		TableStripe:   "#d9da9c",
		Button:        "#eeeeee",
		ButtonHover:   "#dddddd",
		ButtonActive:  "#cccccc",
	}
	return cfg
}

// tokenizerConfig creates the configuration for authentication token reader/writer.
func (Flags) tokenizerConfig(timeFunc func() int64) auth.TokenizerConfig {
	oneDay := 24 * time.Hour.Seconds()
	cfg := auth.TokenizerConfig{
		TimeFunc: timeFunc,
		ValidSec: int64(oneDay),
	}
	return cfg
}

// sqlDatabase creates the configuration for a SQL database to persist user information.
func (f Flags) sqlDatabaseConfig() db.Config {
	cfg := db.Config{
		QueryPeriod: 5 * time.Second,
	}
	return cfg
}

// lobbyConfig creates the configuration for running and managing players of games.
func (f Flags) lobbyConfig() lobby.Config {
	cfg := lobby.Config{
		Debug: f.DebugGame,
	}
	return cfg
}

// gameRunnerConfig creates the configuration for running and managing games.
func (f Flags) gameRunnerConfig(timeFunc func() int64) gameController.RunnerConfig {
	gameCfg := f.gameConfig(timeFunc)
	cfg := gameController.RunnerConfig{
		Debug:      f.DebugGame,
		MaxGames:   4,
		GameConfig: gameCfg,
	}
	return cfg
}

// gameConfig creates the base configuration for all games.
func (f Flags) gameConfig(timeFunc func() int64) gameController.Config {
	playerCfg := playerController.Config{
		WinPoints: 10,
	}
	rand.Seed(timeFunc())
	shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
		rand.Shuffle(len(tiles), func(i, j int) {
			tiles[i], tiles[j] = tiles[j], tiles[i]
		})
	}
	shufflePlayersFunc := func(sockets []player.Name) {
		rand.Shuffle(len(sockets), func(i, j int) {
			sockets[i], sockets[j] = sockets[j], sockets[i]
		})
	}
	cfg := gameController.Config{
		Debug:                  f.DebugGame,
		TimeFunc:               timeFunc,
		MaxPlayers:             6,
		PlayerCfg:              playerCfg,
		NumNewTiles:            21,
		TileLetters:            "", // 144 default tiles = 144-6*21 = 18 tiles left, which leaves a maximum of 3 snags
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return cfg
}

// socketRunnerConfig creates the configuration for creating new sockets (each tab that is connected to the lobby).
func (f Flags) socketRunnerConfig(timeFunc func() int64) socket.RunnerConfig {
	socketCfg := socket.Config{
		Debug:          f.DebugGame,
		TimeFunc:       timeFunc,
		ReadWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		PingPeriod:     15 * time.Second,
		HTTPPingPeriod: 10 * time.Minute,
	}
	cfg := socket.RunnerConfig{
		Debug:            f.DebugGame,
		MaxSockets:       32,
		MaxPlayerSockets: 5,
		SocketConfig:     socketCfg,
	}
	return cfg
}
