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
		cache   bool // cache is specified
		httpPort bool // httpPort is specified
		httpsPort bool // httpsPort is specified
	}{
		{
			osArgs: []string{"name"},
			want:   mainFlags{applicationName: "name"},
		},
		{
			osArgs: []string{"name1", "-app-name=name2"},
			want:   mainFlags{applicationName: "name2"},
		},
		{
			osArgs: []string{"", "https-port=8001"},
		},
		{
			osArgs: []string{"", "-https-port=8001"},
			want:   mainFlags{httpsPort: 8001},
			httpsPort: true,
		},
		{
			osArgs: []string{"", "--https-port=8001"},
			want:   mainFlags{httpsPort: 8001},
			httpsPort: true,
		},
		{
			envVars: map[string]string{"HTTPS_PORT": "8002"},
			want:    mainFlags{httpsPort: 8002},
			httpsPort: true,
		},
		{
			osArgs:  []string{"", "-https-port=8003"},
			envVars: map[string]string{"HTTPS_PORT": "8004"},
			want:    mainFlags{httpsPort: 8003},
			httpsPort: true,
		},
		{
			osArgs: []string{"", "-debug-game"},
			want:   mainFlags{debugGame: true},
		},
		{
			envVars: map[string]string{"DEBUG_GAME_MESSAGES": ""},
			want:    mainFlags{debugGame: true},
		},
		{
			// 	osArgs: []string{"", "-h"}, // should print usage to console
		},
		{ // all command line
			osArgs: []string{
				"",
				"-app-name=1",
				"-https-port=2",
				"-data-source=3",
				"-words-file=4",
				"-debug-game",
				"-cache-sec=467",
			},
			want: mainFlags{
				applicationName: "1",
				httpsPort:       2,
				databaseURL:     "3",
				wordsFile:       "4",
				debugGame:       true,
				cacheSec:        467,
			},
			cache: true,
			httpsPort: true,
		},
		{ // all environment variables
			envVars: map[string]string{
				"APPLICATION_NAME":    "1",
				"HTTPS_PORT":          "2",
				"DATABASE_URL":        "3",
				"WORDS_FILE":          "4",
				"DEBUG_GAME_MESSAGES": "",
				"CACHE_SECONDS":       "113",
			},
			want: mainFlags{
				applicationName: "1",
				httpsPort:       2,
				databaseURL:     "3",
				wordsFile:       "4",
				debugGame:       true,
				cacheSec:        113,
			},
			cache: true,
			httpsPort: true,
		},
	}
	for i, test := range newMainFlagsTests {
		osLookupEnvFunc := func(key string) (string, bool) {
			v, ok := test.envVars[key]
			return v, ok
		}
		got := newMainFlags(test.osArgs, osLookupEnvFunc)
		if !test.httpPort {
			test.want.httpPort = 80
		}
		if !test.httpsPort {
			test.want.httpsPort = 443
		}
		if !test.cache {
			test.want.cacheSec = defaultCacheSec
		}
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestUsage(t *testing.T) {
	programName := "mockProgramName"
	osLookupEnvFunc := func(key string) (string, bool) {
		return "", false
	}
	var m mainFlags
	fs := m.newFlagSet(programName, osLookupEnvFunc)
	var b bytes.Buffer
	fs.SetOutput(&b)
	fs.Init(programName, flag.ContinueOnError) // override ErrorHandling
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
	wantEnvVarCount := envCommas + 1       // n+1 vars are joined with n commas
	wantLineCount := 3 + wantEnvVarCount*2 // 3 initial lines, 2 lines per env var
	gotLineCount := strings.Count(got, "\n")
	note := "NOTE: this might be flaky, but it helps ensure that each environment variable is in the usage text"
	if wantLineCount != gotLineCount {
		t.Errorf("wanted usage to have %v lines, but got %v. %v, got:\n%v", wantLineCount, gotLineCount, note, got)
	}
}
