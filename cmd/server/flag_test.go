package main

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestNewMainFlags(t *testing.T) {
	newMainFlagsTests := []struct {
		osArgs  []string
		envVars map[string]string
		want    mainFlags
	}{
		{ // defaults
			want: mainFlags{
				cacheSec: defaultCacheSec,
			},
		},
		{ // all command line
			osArgs: []string{
				"ignored-binary-name",
				"-http-port=1",
				"-https-port=2",
				"-data-source=3",
				"-words-file=4",
				"-debug-game",
				"-cache-sec=6",
				"-acme-challenge-token=7",
				"-acme-challenge-key=8",
				"-tls-cert-file=9",
				"-tls-key-file=a",
				"-no-tls-redirect",
				"-version-file=c",
			},
			want: mainFlags{
				httpPort:       1,
				httpsPort:      2,
				databaseURL:    "3",
				wordsFile:      "4",
				debugGame:      true,
				cacheSec:       6,
				challengeToken: "7",
				challengeKey:   "8",
				tlsCertFile:    "9",
				tlsKeyFile:     "a",
				noTLSRedirect:  true,
				versionFile:    "c",
			},
		},
		{ // all environment variables
			envVars: map[string]string{
				"HTTP_PORT":            "1",
				"HTTPS_PORT":           "2",
				"DATABASE_URL":         "3",
				"WORDS_FILE":           "4",
				"DEBUG_MESSAGES":       "",
				"CACHE_SECONDS":        "6",
				"ACME_CHALLENGE_TOKEN": "7",
				"ACME_CHALLENGE_KEY":   "8",
				"TLS_CERT_FILE":        "9",
				"TLS_KEY_FILE":         "a",
				"NO_TLS_REDIRECT":      "",
				"VERSION_FILE":         "c",
			},
			want: mainFlags{
				httpPort:       1,
				httpsPort:      2,
				databaseURL:    "3",
				wordsFile:      "4",
				debugGame:      true,
				cacheSec:       6,
				challengeToken: "7",
				challengeKey:   "8",
				tlsCertFile:    "9",
				tlsKeyFile:     "a",
				noTLSRedirect:  true,
				versionFile:    "c",
			},
		},
	}
	for i, test := range newMainFlagsTests {
		osLookupEnvFunc := func(key string) (string, bool) {
			v, ok := test.envVars[key]
			return v, ok
		}
		got := newMainFlags(test.osArgs, osLookupEnvFunc)
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestNewMainFlagsPortOverride(t *testing.T) {
	envVars := map[string]string{
		"HTTP_PORT":     "1",
		"HTTPS_PORT":    "2",
		"PORT":          "3",
		"CACHE_SECONDS": "0", // override default value
	}
	osLookupEnvFunc := func(key string) (string, bool) {
		v, ok := envVars[key]
		return v, ok
	}
	var osArgs []string
	want := mainFlags{
		httpPort:  -1,
		httpsPort: 3,
	}
	got := newMainFlags(osArgs, osLookupEnvFunc)
	if want != got {
		t.Errorf("port should override httpsPort and return -1 for http port\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestUsage(t *testing.T) {
	osLookupEnvFunc := func(key string) (string, bool) {
		return "", false
	}
	var m mainFlags
	var portOverride int
	fs := m.newFlagSet(osLookupEnvFunc, &portOverride)
	var b bytes.Buffer
	fs.SetOutput(&b)
	fs.Init("", flag.ContinueOnError) // override ErrorHandling
	err := fs.Parse([]string{"-h"})
	if err != flag.ErrHelp {
		t.Errorf("wanted ErrHelp, got %v", err)
	}
	got := b.String()
	totalCommas := strings.Count(got, ",")
	b.Reset()
	fs.PrintDefaults()
	defaults := b.String()
	descriptionCommas := strings.Count(defaults, ",")
	envCommas := totalCommas - descriptionCommas
	wantEnvVarCount := envCommas + 2       // n+1 vars are joined with n commas, add an extra 1 for the PORT variable
	wantLineCount := 3 + wantEnvVarCount*2 // 3 initial lines, 2 lines per env var
	gotLineCount := strings.Count(got, "\n")
	note := "NOTE: this might be flaky, but it helps ensure that each environment variable is in the usage text"
	if wantLineCount != gotLineCount {
		t.Errorf("wanted usage to have %v lines, but got %v. %v, got:\n%v", wantLineCount, gotLineCount, note, got)
	}
}
