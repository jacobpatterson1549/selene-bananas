package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

func flagUsage(fs *flag.FlagSet) {
	envVars := []string{
		environmentVariableApplicationName,
		environmentVariableServerPort,
		environmentVariableDatabaseURL,
		environmentVariableWordsFile,
		environmentVariableDebugGame,
	}
	fmt.Fprintln(fs.Output(), "Starts the server")
	fmt.Fprintln(fs.Output(), "Reads environment variables when possible:", fmt.Sprintf("[%s]", strings.Join(envVars, ",")))
	fmt.Fprintln(fs.Output(), fmt.Sprintf("Usage of %s:", fs.Name()))
	fs.PrintDefaults()
}

func initFlags(programName string) (*flag.FlagSet, *mainFlags) {
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
	return fs, &m
}
