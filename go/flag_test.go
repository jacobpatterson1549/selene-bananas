package main

import "testing"

func TestNewMainFlags(t *testing.T) {
	newMainFlagsTests := []struct {
		osArgs  []string
		envVars map[string]string
		want    mainFlags
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
			osArgs: []string{"", "port=8001"},
		},
		{
			osArgs: []string{"", "-port=8001"},
			want:   mainFlags{serverPort: "8001"},
		},
		{
			osArgs: []string{"", "--port=8001"},
			want:   mainFlags{serverPort: "8001"},
		},
		{
			envVars: map[string]string{"PORT": "8002"},
			want:    mainFlags{serverPort: "8002"},
		},
		{
			osArgs:  []string{"", "-port=8003"},
			envVars: map[string]string{"PORT": "8004"},
			want:    mainFlags{serverPort: "8003"},
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
				"-port=2",
				"-data-source=3",
				"-words-file=4",
				"-debug-game",
			},
			want: mainFlags{
				applicationName: "1",
				serverPort:      "2",
				databaseURL:     "3",
				wordsFile:       "4",
				debugGame:       true,
			},
		},
		{ // all environment variables
			envVars: map[string]string{
				"APPLICATION_NAME":    "1",
				"PORT":                "2",
				"DATABASE_URL":        "3",
				"WORDS_FILE":          "4",
				"DEBUG_GAME_MESSAGES": "",
			},
			want: mainFlags{
				applicationName: "1",
				serverPort:      "2",
				databaseURL:     "3",
				wordsFile:       "4",
				debugGame:       true,
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
