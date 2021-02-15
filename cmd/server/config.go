package main

import (
	"context"
	crypto_rand "crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	"github.com/jacobpatterson1549/selene-bananas/server"
	"github.com/jacobpatterson1549/selene-bananas/server/auth"
	"github.com/jacobpatterson1549/selene-bananas/server/certificate"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
	"github.com/jacobpatterson1549/selene-bananas/server/game/lobby"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/game/socket"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

// newServer creates the server.
func newServer(ctx context.Context, m mainFlags, log *log.Logger) (*server.Server, error) {
	timeFunc := func() int64 {
		return time.Now().UTC().Unix()
	}
	key := make([]byte, 64)
	if _, err := crypto_rand.Reader.Read(key); err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	tokenizerCfg := tokenizerConfig(timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer(key)
	if err != nil {
		return nil, fmt.Errorf("creating authentication tokenizer: %w", err)
	}
	if len(m.databaseURL) == 0 {
		return nil, fmt.Errorf("missing data-source uri")
	}
	sqlDB, err := sqlDatabase(m)
	if err != nil {
		return nil, fmt.Errorf("creating SQL database: %w", err)
	}
	sqlFiles, err := userSQLFiles()
	if err != nil {
		return nil, fmt.Errorf("loading SQL files to manage user data: %w", err)
	}
	userDao, err := user.NewDao(ctx, sqlDB, sqlFiles)
	if err != nil {
		return nil, fmt.Errorf("creating user dao: %w", err)
	}
	socketRunnerCfg := socketRunnerConfig(m, timeFunc)
	socketRunner, err := socketRunnerCfg.NewRunner(log)
	if err != nil {
		return nil, fmt.Errorf("creating socket runner: %w", err)
	}
	wordChecker, err := wordChecker(m)
	if err != nil {
		return nil, fmt.Errorf("creating word checker: %w", err)
	}
	gameRunnerCfg := gameRunnerConfig(m, timeFunc)
	gameRunner, err := gameRunnerCfg.NewRunner(log, wordChecker, userDao)
	if err != nil {
		return nil, fmt.Errorf("creating game runner: %w", err)
	}
	lobbyCfg := lobbyConfig(m)
	lobby, err := lobbyCfg.NewLobby(log, socketRunner, gameRunner)
	if err != nil {
		return nil, fmt.Errorf("creating lobby: %w", err)
	}
	version, err := version(m)
	if err != nil {
		return nil, fmt.Errorf("creating build version: %w", err)
	}
	challenge := certificate.Challenge{
		Token: m.challengeToken,
		Key:   m.challengeKey,
	}
	colorConfig := colorConfig()
	cfg := server.Config{
		HTTPPort:      m.httpPort,
		HTTPSPort:     m.httpsPort,
		StopDur:       time.Second,
		CacheSec:      m.cacheSec,
		Version:       version,
		Challenge:     challenge,
		TLSCertFile:   m.tlsCertFile,
		TLSKeyFile:    m.tlsKeyFile,
		ColorConfig:   colorConfig,
		NoTLSRedirect: m.noTLSRedirect,
	}
	return cfg.NewServer(log, tokenizer, userDao, lobby)
}

// colorConfig creates the color config for the css.
func colorConfig() server.ColorConfig {
	cc := server.ColorConfig{
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
	return cc
}

// tokenizerConfig creates the configuration for authentication token reader/writer.
func tokenizerConfig(timeFunc func() int64) auth.TokenizerConfig {
	oneDay := 24 * time.Hour.Seconds()
	cfg := auth.TokenizerConfig{
		TimeFunc: timeFunc,
		ValidSec: int64(oneDay),
	}
	return cfg
}

// sqlDatabase creates a SQL database to persist user information.
func sqlDatabase(m mainFlags) (db.Database, error) {
	cfg := sql.DatabaseConfig{
		DriverName:  "postgres",
		DatabaseURL: m.databaseURL,
		QueryPeriod: 5 * time.Second,
	}
	return cfg.NewDatabase()
}

// userSQLFiles loads the SQL files needed to manage user data.
func userSQLFiles() ([][]byte, error) {
	// SQLFiles are the SQL files that are used for and by the dao.
	userSQLFileNames := []string{
		"users",
		"user_create",
		"user_read",
		"user_update_password",
		"user_update_points_increment",
		"user_delete",
	}
	userSQLFiles := make([][]byte, len(userSQLFileNames))
	for i, n := range userSQLFileNames {
		n = fmt.Sprintf("resources/sql/%s.sql", n)
		b, err := ioutil.ReadFile(n)
		if err != nil {
			return nil, fmt.Errorf("reading setup file %v: %w", n, err)
		}
		userSQLFiles[i] = b
	}
	return userSQLFiles, nil
}

// lobbyConfig creates the configuration for running and managing players of games.
func lobbyConfig(m mainFlags) lobby.Config {
	cfg := lobby.Config{
		Debug: m.debugGame,
	}
	return cfg
}

// gameRunnerConfig creates the configuration for running and managing games.
func gameRunnerConfig(m mainFlags, timeFunc func() int64) gameController.RunnerConfig {
	gameCfg := gameConfig(m, timeFunc)
	cfg := gameController.RunnerConfig{
		Debug:      m.debugGame,
		MaxGames:   4,
		GameConfig: gameCfg,
	}
	return cfg
}

// wordChecker creates the word checker.
func wordChecker(m mainFlags) (*word.Checker, error) {
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		return nil, fmt.Errorf("trying to open words file: %w", err)
	}
	wc := word.NewChecker(wordsFile)
	return wc, nil
}

// gameConfig creates the base configuration for all games.
func gameConfig(m mainFlags, timeFunc func() int64) gameController.Config {
	playerCfg := playerController.Config{
		WinPoints: 10,
	}
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
		Debug:                  m.debugGame,
		TimeFunc:               timeFunc,
		MaxPlayers:             8,
		PlayerCfg:              playerCfg,
		NumNewTiles:            21,
		TileLetters:            "",
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return cfg
}

// socketRunnerConfig creates the configuration for creating new sockets (each tab that is connected to the lobby).
func socketRunnerConfig(m mainFlags, timeFunc func() int64) socket.RunnerConfig {
	socketCfg := socket.Config{
		Debug:          m.debugGame,
		TimeFunc:       timeFunc,
		ReadWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		PingPeriod:     15 * time.Second,
		HTTPPingPeriod: 10 * time.Minute,
	}
	cfg := socket.RunnerConfig{
		Debug:            m.debugGame,
		MaxSockets:       32,
		MaxPlayerSockets: 5,
		SocketConfig:     socketCfg,
	}
	return cfg
}

// version reads the first word of the versionFile to use as the version.
func version(m mainFlags) (string, error) {
	b, err := ioutil.ReadFile(m.versionFile)
	if err != nil {
		return "", err
	}
	version := string(b)
	version = strings.TrimSpace(version) // version often ends in newline
	return version, nil
}
