package game

import (
	"strings"
	"testing"
)

func TestConfigRules(t *testing.T) {
	t.Run("testDifferent", func(t *testing.T) {
		var defaultConfig Config
		defaultRules := defaultConfig.Rules()
		defaultRulesM := make(map[string]struct{}, len(defaultRules))
		for _, r := range defaultRules {
			if _, ok := defaultRulesM[r]; ok {
				t.Errorf("default rule occurred multiple times: '%v'", r)
			}
			defaultRulesM[r] = struct{}{}
		}
		singleChangeConfigs := []Config{
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
				MinLength: 7,
			},
			{
				ProhibitDuplicates: true,
			},
		}
		differentRules := make(map[string]struct{}, len(singleChangeConfigs))
		for i, cfg := range singleChangeConfigs {
			rules := cfg.Rules()
			differentRuleCount := 0
			for _, r := range rules {
				if _, ok := defaultRulesM[r]; !ok {
					differentRuleCount++
					if _, ok := differentRules[r]; ok {
						t.Errorf("Test %v: rule that should be different occurred more than once between configs: '%v'", i, r)
					}
					differentRules[r] = struct{}{}
				}
			}
			switch {
			case differentRuleCount != 1:
				t.Errorf("Test %v: wanted only 1 different rule, got %v", i, differentRuleCount)
			case len(defaultRules)+1 != len(rules):
				t.Errorf("Test %v: wanted %v rules, got %v", i, len(defaultRules)+1, len(rules))
			}
		}
	})
	t.Run("TestMinLengthNumber", func(t *testing.T) {
		cfg := Config{
			MinLength: 1337,
		}
		rules := cfg.Rules()
		for _, r := range rules {
			if strings.Contains(r, "1337") {
				return
			}
		}
		t.Errorf("abnormal minlength not included in rules")
	})
}
