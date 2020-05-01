package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/server"
	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
	_ "github.com/lib/pq"
)

const (
	environmentVariableApplicationName = "APPLICATION_NAME"
	environmentVariableServerPort      = "PORT"
	environmentVariableDatabaseURL     = "DATABASE_URL"
	environmentVariableHTTPSCertFile   = "HTTPS_CERT_FILE"
	environmentVariableHTTPSKeyFile    = "HTTPS_KEY_FILE"
)

type mainFlags struct {
	applicationName string
	serverPort      string
	httpsCertFile   string
	httpsKeyFile    string
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
		HTTPSCertFile: mainFlags.httpsCertFile,
		HTTPSKeyFile:  mainFlags.httpsKeyFile,
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
		environmentVariableHTTPSCertFile,
		environmentVariableHTTPSKeyFile,
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
	fs.StringVar(&mainFlags.httpsCertFile, "tls-cert", os.Getenv(environmentVariableHTTPSCertFile), "The absolute path of the certificate file to use for TLS")
	fs.StringVar(&mainFlags.httpsKeyFile, "tls-key", os.Getenv(environmentVariableHTTPSKeyFile), "The absolute path of the key file to use for TLS.")
	return fs, mainFlags
}
