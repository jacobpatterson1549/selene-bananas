package main

import (
	"bufio"
	"context"
	crypto_rand "crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/game/lobby"
	"github.com/jacobpatterson1549/selene-bananas/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	"github.com/jacobpatterson1549/selene-bananas/server"
	"github.com/jacobpatterson1549/selene-bananas/server/auth"
	"github.com/jacobpatterson1549/selene-bananas/server/certificate"
	_ "github.com/lib/pq" // register "postgres" database driver from package init() function
)

func serverConfig(ctx context.Context, m mainFlags, log *log.Logger) (*server.Config, error) {
	timeFunc := func() int64 {
		return time.Now().UTC().Unix()
	}
	keyReader := crypto_rand.Reader
	tokenizerCfg := tokenizerConfig(keyReader, timeFunc)
	tokenizer, err := tokenizerCfg.NewTokenizer()
	if err != nil {
		return nil, err
	}
	if len(m.databaseURL) == 0 {
		return nil, fmt.Errorf("missing data-source uri")
	}
	sqlDB, err := sqlDatabase(m)
	if err != nil {
		return nil, err
	}
	userDaoCfg := userDaoConfig(sqlDB)
	ud, err := userDaoCfg.NewDao()
	if err != nil {
		return nil, err
	}
	err = ud.Setup(ctx)
	if err != nil {
		return nil, err
	}
	lobbyCfg, err := lobbyConfig(m, log, ud, timeFunc)
	if err != nil {
		return nil, err
	}
	v := version(m, log)
	c := certificate.Challenge{
		Token: m.challengeToken,
		Key:   m.challengeKey,
	}
	cc := colorConfig()
	cfg := server.Config{
		HTTPPort:      m.httpPort,
		HTTPSPort:     m.httpsPort,
		Log:           log,
		Tokenizer:     tokenizer,
		UserDao:       ud,
		LobbyCfg:      *lobbyCfg,
		StopDur:       time.Second,
		CacheSec:      m.cacheSec,
		Version:       v,
		Challenge:     c,
		TLSCertFile:   m.tlsCertFile,
		TLSKeyFile:    m.tlsKeyFile,
		ColorConfig:   cc,
		NoTLSRedirect: m.noTLSRedirect,
	}
	return &cfg, nil
}

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

func tokenizerConfig(keyReader io.Reader, timeFunc func() int64) auth.TokenizerConfig {
	var tokenValidDurationSec int64 = int64((24 * time.Hour).Seconds()) // 1 day
	cfg := auth.TokenizerConfig{
		KeyReader: keyReader,
		TimeFunc:  timeFunc,
		ValidSec:  tokenValidDurationSec,
	}
	return cfg
}

func sqlDatabase(m mainFlags) (db.Database, error) {
	cfg := sql.DatabaseConfig{
		DriverName:  "postgres",
		DatabaseURL: m.databaseURL,
		QueryPeriod: 5 * time.Second,
	}
	db, err := cfg.NewDatabase()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func userDaoConfig(d db.Database) user.DaoConfig {
	cfg := user.DaoConfig{
		DB:           d,
		ReadFileFunc: ioutil.ReadFile,
	}
	return cfg
}

func lobbyConfig(m mainFlags, log *log.Logger, ud *user.Dao, timeFunc func() int64) (*lobby.Config, error) {
	gameCfg, err := gameConfig(m, log, ud, timeFunc)
	if err != nil {
		return nil, err
	}
	socketCfg := socketConfig(m, log, timeFunc)
	cfg := lobby.Config{
		Debug:      m.debugGame,
		Log:        log,
		MaxGames:   4,
		MaxSockets: 32,
		GameCfg:    *gameCfg,
		SocketCfg:  socketCfg,
	}
	return &cfg, nil
}

func gameConfig(m mainFlags, log *log.Logger, ud *user.Dao, timeFunc func() int64) (*controller.Config, error) {
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		return nil, err
	}
	wordChecker := word.NewChecker(wordsFile)
	shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
		rand.Shuffle(len(tiles), func(i, j int) {
			tiles[i], tiles[j] = tiles[j], tiles[i]
		})
	}
	shufflePlayersFunc := func(sockets []game.PlayerName) {
		rand.Shuffle(len(sockets), func(i, j int) {
			sockets[i], sockets[j] = sockets[j], sockets[i]
		})
	}
	cfg := controller.Config{
		Debug:                  m.debugGame,
		Log:                    log,
		TimeFunc:               timeFunc,
		UserDao:                ud,
		MaxPlayers:             8,
		NumNewTiles:            21,
		TileLetters:            "",
		WordChecker:            wordChecker,
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return &cfg, nil
}

func socketConfig(m mainFlags, log *log.Logger, timeFunc func() int64) socket.Config {
	cfg := socket.Config{
		Debug:          m.debugGame,
		Log:            log,
		TimeFunc:       timeFunc,
		ReadWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		IdlePeriod:     15 * time.Minute,
		HTTPPingPeriod: 10 * time.Minute,
	}
	return cfg
}

func version(m mainFlags, log *log.Logger) string {
	versionFile, err := os.Open(m.versionFile)
	if err != nil {
		log.Printf("trying to open version file: %v", err)
		return ""
	}
	scanner := bufio.NewScanner(versionFile)
	scanner.Split(bufio.ScanWords)
	if !scanner.Scan() {
		log.Print("no words in version file")
		return ""
	}
	return scanner.Text()
}
