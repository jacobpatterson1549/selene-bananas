package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

const (
	environmentVariableHTTPPort       = "HTTP_PORT"
	environmentVariableHTTPSPort      = "HTTPS_PORT"
	environmentVariablePort           = "PORT"
	environmentVariableDatabaseURL    = "DATABASE_URL"
	environmentVariableDebugGame      = "DEBUG_MESSAGES"
	environmentVariableNoTLSRedirect  = "NO_TLS_REDIRECT"
	environmentVariableCacheSec       = "CACHE_SECONDS"
	environmentVariableChallengeToken = "ACME_CHALLENGE_TOKEN"
	environmentVariableChallengeKey   = "ACME_CHALLENGE_KEY"
)

// Flags are the configuration options which can be easly configured at run startup for different environments.
type Flags struct {
	HTTPPort       int
	HTTPSPort      int
	DatabaseURL    string
	ChallengeToken string
	ChallengeKey   string
	DebugGame      bool
	NoTLSRedirect  bool
	CacheSec       int
}

const (
	defaultCacheSec int = 60 * 60 * 24 // 1 day
)

// usage prints how to run the server to the flagset's output.
func usage(fs *flag.FlagSet) {
	envVars := []string{
		environmentVariableHTTPPort,
		environmentVariableHTTPSPort,
		environmentVariableDatabaseURL,
		environmentVariableDebugGame,
		environmentVariableNoTLSRedirect,
		environmentVariableCacheSec,
		environmentVariableChallengeToken,
		environmentVariableChallengeKey,
	}
	fmt.Fprintf(fs.Output(), "Runs the server\n")
	fmt.Fprintf(fs.Output(), "Reads environment variables when possible: [%s]\n", strings.Join(envVars, ","))
	fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
	fs.PrintDefaults()
}

// newFlagSet creates a flagSet that populates the flags.
func (f *Flags) newFlagSet(osLookupEnvFunc func(string) (string, bool), portOverride *int) *flag.FlagSet {
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	fs.Usage = func() {
		usage(fs) // [lazy evaluation]
	}
	envValue := func(key string) string {
		if envValue, ok := osLookupEnvFunc(key); ok {
			return envValue
		}
		return ""
	}
	envValueInt := func(key string, defaultValue int) int {
		v1 := envValue(key)
		v2, err := strconv.Atoi(v1)
		if err != nil {
			return defaultValue
		}
		return v2
	}
	envPresent := func(key string) bool {
		_, ok := osLookupEnvFunc(key)
		return ok
	}
	fs.StringVar(&f.DatabaseURL, "data-source", envValue(environmentVariableDatabaseURL), "The data source to the PostgreSQL database (connection URI).")
	fs.IntVar(&f.HTTPPort, "http-port", envValueInt(environmentVariableHTTPPort, 0), "The TCP port for server http requests.  All traffic is redirected to the https port.")
	fs.IntVar(&f.HTTPSPort, "https-port", envValueInt(environmentVariableHTTPSPort, 0), "The TCP port for server https requests.")
	fs.IntVar(portOverride, "port", envValueInt(environmentVariablePort, 0), "The single port to run the server on.  Overrides the -https-port flag.  Causes the server to not handle http requests, ignoring -http-port.")
	fs.StringVar(&f.ChallengeToken, "acme-challenge-token", envValue(environmentVariableChallengeToken), "The ACME HTTP-01 Challenge token used to get a certificate.")
	fs.StringVar(&f.ChallengeKey, "acme-challenge-key", envValue(environmentVariableChallengeKey), "The ACME HTTP-01 Challenge key used to get a certificate.")
	fs.BoolVar(&f.DebugGame, "debug-game", envPresent(environmentVariableDebugGame), "Logs message types in the console when messages are passed between components.")
	fs.BoolVar(&f.NoTLSRedirect, "no-tls-redirect", envPresent(environmentVariableNoTLSRedirect), "Disables HTTPS redirection from http if present.")
	fs.IntVar(&f.CacheSec, "cache-sec", envValueInt(environmentVariableCacheSec, defaultCacheSec), "The number of seconds static assets are cached, such as javascript files.")
	return fs
}

// newFlags creates a new, populated flags structure.
// Fields are populated from command line arguments.
// If fields are not specified on the command line, environment variable values are used before defaulting to other defaults.
func newFlags(osArgs []string, osLookupEnvFunc func(string) (string, bool)) *Flags {
	if len(osArgs) == 0 {
		osArgs = []string{""}
	}
	programArgs := osArgs[1:]
	f := new(Flags)
	var portOverride int
	fs := f.newFlagSet(osLookupEnvFunc, &portOverride)
	fs.Parse(programArgs)
	if portOverride != 0 {
		f.HTTPSPort = portOverride
		f.HTTPPort = -1
	}
	return f
}
