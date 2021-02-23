package main

import (
	"context"
	crypto_rand "crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

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
	setupSQL, err := setupSQL(embeddedSQLFS)
	if err != nil {
		return nil, fmt.Errorf("loading SQL files to manage user data: %w", err)
	}
	userDao, err := user.NewDao(ctx, sqlDB, setupSQL)
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
	version, err := cleanVersion(embedVersion)
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
		StopDur:       20 * time.Second, // should be longer than the PingPeriod of sockets so they can close gracefully
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

// setupSQL loads the SQL files needed to manage user data.
func setupSQL(fsys fs.FS) ([][]byte, error) {
	files, err := Files(fsys)
	if err != nil {
		return nil, err
	}
	setupSQL := make([][]byte, 0, len(files))
	for n, f := range files {
		b, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("reading setup query %v: %v", n, err)
		}
		setupSQL = append(setupSQL, b)
	}
	return setupSQL, nil
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

// cleanVersion returns the version, but cleaned up to only be letters and numbers.
// Spaces on each end are trimmed, but spaces in the middle of the version or special characters cause an error to be returned.
func cleanVersion(v string) (string, error) {
	cleanV := strings.TrimSpace(v)
	switch {
	case len(cleanV) == 0:
		return "", fmt.Errorf("empty")
	default:
		for i, r := range cleanV {
			if !unicode.In(r, unicode.Letter, unicode.Digit) {
				return "", fmt.Errorf("only letters and digits are allowed: invalid rune at index %v of '%v': '%v'", i, cleanV, string(r))
			}
		}
	}
	return cleanV, nil
}

// Files collects the files in the file system by name.
func Files(fsys fs.FS) (map[string]fs.File, error) {
	if fsys == nil {
		return nil, nil
	}
	files := make(map[string]fs.File)
	var walkDir fs.WalkDirFunc
	walkDir = func(path string, d fs.DirEntry, err error) error {
		switch {
		case err == fs.SkipDir:
			// NOOP
		case d.IsDir():
			dirEntries, err := fs.ReadDir(fsys, path)
			if err != nil {
				return err
			}
			for _, de := range dirEntries {
				p := filepath.Join(path, de.Name())
				if err := walkDir(p, de, nil); err != nil {
					return err
				}
			}
		default:
			f, err := fsys.Open(path)
			if err != nil {
				return err
			}
			files[path] = f
		}
		return nil
	}
	err := fs.WalkDir(fsys, ".", walkDir)
	if err != nil {
		return nil, fmt.Errorf("collecting files: %w", err)
	}
	return files, nil
}
