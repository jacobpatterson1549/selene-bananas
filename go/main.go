package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/game/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/go/server"
	_ "github.com/lib/pq"
)

const (
	environmentVariableApplicationName = "APPLICATION_NAME"
	environmentVariableServerPort      = "PORT"
	environmentVariableDatabaseURL     = "DATABASE_URL"
	environmentVariableWordsFile       = "WORDS_FILE"
	environmentVariableDebugGame       = "DEBUG_GAME_MESSAGES"
)

type mainFlags struct {
	applicationName string
	serverPort      string
	databaseURL     string
	wordsFile       string
	debugGame       bool
}

func main() {
	fs, m := initFlags(os.Args[0])
	flag.CommandLine = fs
	flag.Parse()

	var buf bytes.Buffer
	log := log.New(&buf, m.applicationName+" ", log.LstdFlags)
	log.SetOutput(os.Stdout)

	cfg, err := serverConfig(m, log)
	if err != nil {
		log.Fatalf("configuring server: %v", err)
	}

	server, err := cfg.NewServer()
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	err = server.Run()
	if err != nil {
		log.Fatalf("running server: %v", err)
	}
}

func flagUsage(fs *flag.FlagSet) {
	envVars := []string{
		environmentVariableApplicationName,
		environmentVariableServerPort,
		environmentVariableDatabaseURL,
	}
	fmt.Fprintln(fs.Output(), "Starts the server")
	fmt.Fprintln(fs.Output(), "Reads environment variables when possible:", fmt.Sprintf("[%s]", strings.Join(envVars, ",")))
	fmt.Fprintln(fs.Output(), fmt.Sprintf("Usage of %s:", fs.Name()))
	fs.PrintDefaults()
}

func initFlags(programName string) (*flag.FlagSet, mainFlags) {
	fs := flag.NewFlagSet(programName, flag.ExitOnError)
	fs.Usage = func() { flagUsage(fs) }
	var m mainFlags
	envOrDefault := func(envKey, defaultValue string) string {
		if envValue, ok := os.LookupEnv(envKey); ok {
			return envValue
		}
		return defaultValue
	}
	defaultDebugGame := func() bool {
		_, ok := os.LookupEnv(environmentVariableDebugGame)
		return ok
	}
	fs.StringVar(&m.applicationName, "app-name", envOrDefault(environmentVariableApplicationName, programName), "The name of the application.")
	fs.StringVar(&m.databaseURL, "data-source", os.Getenv(environmentVariableDatabaseURL), "The data source to the PostgreSQL database (connection URI).")
	fs.StringVar(&m.serverPort, "port", os.Getenv(environmentVariableServerPort), "The port number to run the server on.")
	fs.StringVar(&m.wordsFile, "words-file", envOrDefault(environmentVariableWordsFile, "/usr/share/dict/american-english-small"), "The list of valid lower-case words that can be used.")
	fs.BoolVar(&m.debugGame, "debug-game", defaultDebugGame(), "Logs game message types in the console if present.")
	return fs, m
}

func serverConfig(m mainFlags, log *log.Logger) (*server.Config, error) {
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	tokenizer, err := server.NewTokenizer(rand)
	if err != nil {
		return nil, err
	}
	d, err := db.NewPostgresDatabase(m.databaseURL)
	if err != nil {
		return nil, err
	}
	ud := db.NewUserDao(d)
	err = ud.Setup()
	if err != nil {
		return nil, err
	}
	lobbyCfg, err := lobbyConfig(m, log, rand, ud)
	if err != nil {
		return nil, err
	}
	cfg := server.Config{
		AppName:   m.applicationName,
		Port:      m.serverPort,
		Log:       log,
		Tokenizer: tokenizer,
		UserDao:   ud,
		LobbyCfg:  *lobbyCfg,
	}
	return &cfg, nil
}

func lobbyConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud db.UserDao) (*lobby.Config, error) {
	gameCfg, err := gameConfig(m, log, rand, ud)
	socketCfg := socketConfig(m, log)
	if err != nil {
		return nil, err
	}
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

func gameConfig(m mainFlags, log *log.Logger, rand *rand.Rand, ud db.UserDao) (*controller.Config, error) {
	wordsFile, err := os.Open(m.wordsFile)
	if err != nil {
		return nil, err
	}
	wc, err := game.NewWordChecker(wordsFile)
	if err != nil {
		return nil, err
	}
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
		UserDao:                ud,
		MaxPlayers:             8,
		NumNewTiles:            21,
		TileLetters:            "",
		Words:                  *wc,
		IdlePeriod:             60 * time.Minute,
		ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
		ShufflePlayersFunc:     shufflePlayersFunc,
	}
	return &cfg, nil
}

func socketConfig(m mainFlags, log *log.Logger) socket.Config {
	cfg := socket.Config{
		Debug:          m.debugGame,
		Log:            log,
		PongPeriod:     20 * time.Second,
		PingPeriod:     15 * time.Second,
		IdlePeriod:     15 * time.Minute,
		HTTPPingPeriod: 10 * time.Minute,
	}
	return cfg
}
