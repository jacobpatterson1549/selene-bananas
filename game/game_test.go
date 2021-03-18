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
				MinLength: 7,
			},
			{
				ProhibitDuplicates: true,
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
	t.Run("TestUniqueRules", func(t *testing.T) {
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
				ProhibitDuplicates: true,
			},
			{
				CheckOnSnag:        true,
				Penalize:           true,
				MinLength:          4,
				ProhibitDuplicates: true,
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
	})
}
