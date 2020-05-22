package main

import (
	"flag"
	"fmt"
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

func usage(fs *flag.FlagSet) {
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

// newMainFlags creates a new, populated mainFlags structure
// Fields are populated from command line arguments.
// If fields are not specified on the command line, environment variable values are used before defaulting to other defaults.
func newMainFlags(osArgs []string, osLookupEnvFunc func(string) (string, bool)) mainFlags {
	if len(osArgs) == 0 {
		osArgs = []string{""}
	}
	programName, programArgs := osArgs[0], osArgs[1:]
	fs := flag.NewFlagSet(programName, flag.ExitOnError)
	fs.Usage = func() { usage(fs) }
	var m mainFlags
	envOrDefault := func(key, defaultValue string) string {
		if envValue, ok := osLookupEnvFunc(key); ok {
			return envValue
		}
		return defaultValue
	}
	envPresent := func(key string) bool {
		_, ok := osLookupEnvFunc(key)
		return ok
	}
	fs.StringVar(&m.applicationName, "app-name", envOrDefault(environmentVariableApplicationName, programName), "The name of the application.")
	fs.StringVar(&m.databaseURL, "data-source", envOrDefault(environmentVariableDatabaseURL, ""), "The data source to the PostgreSQL database (connection URI).")
	fs.StringVar(&m.serverPort, "port", envOrDefault(environmentVariableServerPort, ""), "The port number to run the server on.")
	fs.StringVar(&m.wordsFile, "words-file", envOrDefault(environmentVariableWordsFile, ""), "The list of valid lower-case words that can be used.")
	fs.BoolVar(&m.debugGame, "debug-game", envPresent(environmentVariableDebugGame), "Logs game message types in the console if present.")
	fs.Parse(programArgs)
	return m
}
