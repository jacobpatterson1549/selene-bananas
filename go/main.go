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
)

type mainFlags struct {
	applicationName string
	serverPort      string
	databaseURL     string
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
	wordsFileName := "/usr/share/dict/american-english" // TODO: add env variable

	cfg := server.Config{
		AppName:       mainFlags.applicationName,
		Port:          mainFlags.serverPort,
		Database:      db,
		Log:           log,
		WordsFileName: wordsFileName,
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
	defaultApplicationName := func() string {
		if applicationName, ok := os.LookupEnv(environmentVariableApplicationName); ok {
			return applicationName
		}
		return programName
	}
	fs.StringVar(&mainFlags.applicationName, "n", defaultApplicationName(), "The name of the application.")
	fs.StringVar(&mainFlags.databaseURL, "ds", os.Getenv(environmentVariableDatabaseURL), "The data source to the PostgreSQL database (connection URI).")
	fs.StringVar(&mainFlags.serverPort, "p", os.Getenv(environmentVariableServerPort), "The port number to run the server on.")
	return fs, mainFlags
}
