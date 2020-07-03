package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

const (
	environmentVariableHTTPPort        = "HTTP_PORT"
	environmentVariableHTTPSPort       = "HTTPS_PORT"
	environmentVariableDatabaseURL     = "DATABASE_URL"
	environmentVariableWordsFile       = "WORDS_FILE"
	environmentVariableVersionFile     = "VERSION_FILE"
	environmentVariableDebugGame       = "DEBUG_GAME_MESSAGES"
	environmentVariableCacheSec        = "CACHE_SECONDS"
	environmentVariableChallengeToken  = "ACME_CHALLENGE_TOKEN"
	environmentVariableChallengeKey    = "ACME_CHALLENGE_KEY"
	environmentVariableTLSCertFile     = "TLS_CERT_FILE"
	environmentVariableTLSKeyFile      = "TLS_KEY_FILE"
)

type mainFlags struct {
	httpPort        int
	httpsPort       int
	databaseURL     string
	wordsFile       string
	versionFile     string
	challengeToken  string
	challengeKey    string
	tlsCertFile     string
	tlsKeyFile      string
	debugGame       bool
	cacheSec        int
}

const (
	defaultCacheSec int = 60 * 60 * 24 * 365 // 1 year
)

func usage(fs *flag.FlagSet) {
	envVars := []string{
		environmentVariableHTTPPort,
		environmentVariableHTTPSPort,
		environmentVariableDatabaseURL,
		environmentVariableWordsFile,
		environmentVariableVersionFile,
		environmentVariableDebugGame,
		environmentVariableCacheSec,
		environmentVariableChallengeToken,
		environmentVariableChallengeKey,
		environmentVariableTLSCertFile,
		environmentVariableTLSKeyFile,
	}
	fmt.Fprintln(fs.Output(), "Starts the server")
	fmt.Fprintln(fs.Output(), "Reads environment variables when possible:", fmt.Sprintf("[%s]", strings.Join(envVars, ",")))
	fmt.Fprintln(fs.Output(), fmt.Sprintf("Usage of %s:", fs.Name()))
	fs.PrintDefaults()
}

// newFlagSet creates a flagSet that populates the specified mainFlags.
func (m *mainFlags) newFlagSet(osLookupEnvFunc func(string) (string, bool)) *flag.FlagSet {
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	fs.Usage = func() { usage(fs) }

	envOrDefault := func(key, defaultValue string) string {
		if envValue, ok := osLookupEnvFunc(key); ok {
			return envValue
		}
		return defaultValue
	}
	envOrDefaultInt := func(key string, defaultValue int) int {
		v1 := envOrDefault(key, string(defaultValue))
		if v2, err := strconv.Atoi(v1); err == nil {
			return v2
		}
		return defaultValue
	}
	envPresent := func(key string) bool {
		_, ok := osLookupEnvFunc(key)
		return ok
	}
	fs.StringVar(&m.databaseURL, "data-source", envOrDefault(environmentVariableDatabaseURL, ""), "The data source to the PostgreSQL database (connection URI).")
	fs.IntVar(&m.httpPort, "http-port", envOrDefaultInt(environmentVariableHTTPPort, 80), "The TCP port for server http requests.  All traffic is redirected to the https port.")
	fs.IntVar(&m.httpsPort, "https-port", envOrDefaultInt(environmentVariableHTTPSPort, 443), "The TCP port for server https requests.")
	fs.StringVar(&m.wordsFile, "words-file", envOrDefault(environmentVariableWordsFile, ""), "The list of valid lower-case words that can be used.")
	fs.StringVar(&m.versionFile, "version-file", envOrDefault(environmentVariableVersionFile, ""), "A file containing the version key (the first word).  Used to bust previously cached files.  Change each time a new version of the server is run.")
	fs.StringVar(&m.challengeToken, "acme-challenge-token", envOrDefault(environmentVariableChallengeToken, ""), "The ACME HTTP-01 Challenge token used to get a certificate.")
	fs.StringVar(&m.challengeKey, "acme-challenge-key", envOrDefault(environmentVariableChallengeKey, ""), "The ACME HTTP-01 Challenge key used to get a certificate.")
	fs.StringVar(&m.tlsCertFile, "tls-cert", envOrDefault(environmentVariableTLSCertFile, ""), "The absolute path of the certificate file to use for TLS.")
	fs.StringVar(&m.tlsKeyFile, "tls-key", envOrDefault(environmentVariableTLSKeyFile, ""), "The absolute path of the key file to use for TLS.")
	fs.BoolVar(&m.debugGame, "debug-game", envPresent(environmentVariableDebugGame), "Logs game message types in the console if present.")
	fs.IntVar(&m.cacheSec, "cache-sec", envOrDefaultInt(environmentVariableCacheSec, defaultCacheSec), "The number of seconds static assets are cached, such as javascript files.")
	return fs
}

// newMainFlags creates a new, populated mainFlags structure
// Fields are populated from command line arguments.
// If fields are not specified on the command line, environment variable values are used before defaulting to other defaults.
func newMainFlags(osArgs []string, osLookupEnvFunc func(string) (string, bool)) mainFlags {
	if len(osArgs) == 0 {
		osArgs = []string{""}
	}
	programArgs := osArgs[1:]
	var m mainFlags
	fs := m.newFlagSet(osLookupEnvFunc)
	fs.Parse(programArgs)
	return m
}
