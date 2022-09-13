package main

import (
	"bytes"
	"flag"
	"reflect"
	"strings"
	"testing"
)

func TestNewFlags(t *testing.T) {
	newFlagsTests := []struct {
		osArgs  []string
		envVars map[string]string
		want    *Flags
	}{
		{ // defaults
			want: &Flags{
				CacheSec:     defaultCacheSec,
				DBTimeoutSec: defaultDBTimeoutSec,
			},
		},
		{ // all command line
			osArgs: []string{
				"ignored-binary-name",
				"-http-port=1",
				"-https-port=2",
				"-data-source=3",
				"-debug-game",
				"-cache-sec=6",
				"-acme-challenge-token=7",
				"-acme-challenge-key=8",
				"-no-tls-redirect",
				"-db-timeout-sec=30",
			},
			want: &Flags{
				HTTPPort:       1,
				HTTPSPort:      2,
				DatabaseURL:    "3",
				DebugGame:      true,
				CacheSec:       6,
				ChallengeToken: "7",
				ChallengeKey:   "8",
				NoTLSRedirect:  true,
				DBTimeoutSec:   30,
			},
		},
		{ // all environment variables
			envVars: map[string]string{
				"HTTP_PORT":            "1",
				"HTTPS_PORT":           "2",
				"DATABASE_URL":         "3",
				"DEBUG_MESSAGES":       "",
				"CACHE_SECONDS":        "6",
				"ACME_CHALLENGE_TOKEN": "7",
				"ACME_CHALLENGE_KEY":   "8",
				"NO_TLS_REDIRECT":      "",
				"DB_TIMEOUT_SEC":       "9",
			},
			want: &Flags{
				HTTPPort:       1,
				HTTPSPort:      2,
				DatabaseURL:    "3",
				DebugGame:      true,
				CacheSec:       6,
				ChallengeToken: "7",
				ChallengeKey:   "8",
				NoTLSRedirect:  true,
				DBTimeoutSec:   9,
			},
		},
	}
	for i, test := range newFlagsTests {
		osLookupEnvFunc := func(key string) (string, bool) {
			v, ok := test.envVars[key]
			return v, ok
		}
		got := newFlags(test.osArgs, osLookupEnvFunc)
		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestNewFlagsPortOverride(t *testing.T) {
	envVars := map[string]string{
		"HTTP_PORT":      "1",
		"HTTPS_PORT":     "2",
		"PORT":           "3",
		"CACHE_SECONDS":  "0", // override default value
		"DB_TIMEOUT_SEC": "0", // override default value
	}
	osLookupEnvFunc := func(key string) (string, bool) {
		v, ok := envVars[key]
		return v, ok
	}
	var osArgs []string
	want := &Flags{
		HTTPPort:  -1,
		HTTPSPort: 3,
	}
	got := newFlags(osArgs, osLookupEnvFunc)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("port should override httpsPort and return -1 for http port\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestUsage(t *testing.T) {
	osLookupEnvFunc := func(key string) (string, bool) {
		return "", false
	}
	var f Flags
	var portOverride int
	fs := f.newFlagSet(osLookupEnvFunc, &portOverride)
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Init("", flag.ContinueOnError) // override ErrorHandling
	err := fs.Parse([]string{"-h"})
	if err != flag.ErrHelp {
		t.Errorf("wanted ErrHelp, got %v", err)
	}
	got := buf.String()
	totalCommas := strings.Count(got, ",")
	buf.Reset()
	fs.PrintDefaults()
	defaults := buf.String()
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
