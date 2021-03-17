package game

import (
	"strings"
	"testing"
)

func TestConfigRules(t *testing.T) {
	t.Run("testDifferent", func(t *testing.T) {
		simpleConfig := Config{
			AllowDuplicates: true, // TODO: reword variable to make allowing duplicates default, but be false in the config
		}
		defaultRules := simpleConfig.Rules()
		defaultRulesM := make(map[string]struct{}, len(defaultRules))
		for _, r := range defaultRules {
			defaultRulesM[r] = struct{}{}
		}
		singleChangeConfigs := []Config{
			{
				CheckOnSnag:     true,
				AllowDuplicates: true,
			},
			{
				Penalize:        true,
				AllowDuplicates: true,
			},
			{
				MinLength:       7,
				AllowDuplicates: true,
			},
			{
				AllowDuplicates: false,
			},
		}
		for i, cfg := range singleChangeConfigs {
			rules := cfg.Rules()
			differentRuleCount := 0
			for _, r := range rules {
				if _, ok := defaultRulesM[r]; !ok {
					differentRuleCount++
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
			AllowDuplicates: true,
			MinLength:       1337,
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
