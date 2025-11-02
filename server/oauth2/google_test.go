package oauth2

import (
	"fmt"
	"testing"
)

func TestNewEndpoint(t *testing.T) {
	tests := []struct {
		clientID string
		secret   string
		wantCfg  bool
	}{
		{},
		{clientID: "my_client"},
		{secret: "my_secret"},
		{clientID: "my_client", secret: "my_secret", wantCfg: true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%v", i), func(t *testing.T) {
			cfg := GoogleConfig{
				ClientID:     test.clientID,
				ClientSecret: test.secret,
			}
			got, err := cfg.NewEndpoint()
			if err != nil {
				t.Errorf("unwanted error: %v", err)
			}
			if want, got := test.wantCfg, got != nil; want != got {
				t.Error()
			}
		})
	}
}
