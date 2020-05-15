package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
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
	fs, mainFlags := initFlags(os.Args[0])
	flag.CommandLine = fs
	flag.Parse()

	var buf bytes.Buffer
	log := log.New(&buf, mainFlags.applicationName+" ", log.LstdFlags)
	log.SetOutput(os.Stdout)

	db, err := db.NewPostgresDatabase(mainFlags.databaseURL)
	if err != nil {
		log.Fatal(err)
	}

	cfg := server.Config{
		AppName:       mainFlags.applicationName,
		Port:          mainFlags.serverPort,
		Database:      db,
		Log:           log,
		WordsFileName: mainFlags.wordsFile,
		DebugGame:     mainFlags.debugGame,
	}
	server, err := cfg.NewServer()
	if err != nil {
		log.Fatal("creating server:", err)
	}

	err = server.Run()
	if err != nil {
		log.Fatal(err)
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

func initFlags(programName string) (*flag.FlagSet, *mainFlags) {
	fs := flag.NewFlagSet(programName, flag.ExitOnError)
	fs.Usage = func() { flagUsage(fs) }
	mainFlags := new(mainFlags)
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
	fs.StringVar(&mainFlags.applicationName, "n", envOrDefault(environmentVariableApplicationName, programName), "The name of the application.")
	fs.StringVar(&mainFlags.databaseURL, "ds", os.Getenv(environmentVariableDatabaseURL), "The data source to the PostgreSQL database (connection URI).")
	fs.StringVar(&mainFlags.serverPort, "p", os.Getenv(environmentVariableServerPort), "The port number to run the server on.")
	fs.StringVar(&mainFlags.wordsFile, "wf", envOrDefault(environmentVariableWordsFile, "/usr/share/dict/american-english"), "The list of valid lower-case words that can be used.")
	fs.BoolVar(&mainFlags.debugGame, "dg", defaultDebugGame(), "Logs game message types in the console if present.")
	return fs, mainFlags
}
