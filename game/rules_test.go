package game

import (
	"strings"
	"testing"
)

func TestRules(t *testing.T) {
	configs := []Config{
		{},
		{
			CheckOnSnag: true,
		},
		{
			Penalize: true,
		},
		{
			MinLength: 3,
		},
		{
			AllowDuplicates: true,
		},
		{
			CheckOnSnag: true,
			Penalize: true,
			MinLength: 2,
			AllowDuplicates: true,
		},
	}
	uniqueRules := make(map[string]struct{}, len(configs))
	for _, cfg := range configs {
		r := cfg.Rules()
		longRules := strings.Join(r, "")
		uniqueRules[longRules] = struct{}{}
	}
	if len(configs) != len(uniqueRules) {
		t.Errorf("wanted %v unique rule lists for the configs, got %v", len(configs), len(uniqueRules))
	}
}
