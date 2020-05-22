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
			osArgs: []string{"", "-port=8001"},
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
