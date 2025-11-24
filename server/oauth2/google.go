package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleConfig struct {
	// ClientID is the username for Google Oauth2 user logins.
	ClientID string
	// ClientSecret is the username for Google Oauth2 user logins.
	ClientSecret string
	// CSRFToken should be a randomly generated token to prevent CSRF attacks.
	CSRFToken string
	// RedirectURLScheme should be either https or http.  Defaults to HTTPS.
	RedirectURL string
}

const (
	GoogleLoginURL    = "/oauth2_google_login"
	GoogleCallbackURL = "/oauth2_google_callback"
)

// NewEndpoint creates a Google Oauth2 config.
// Nil is returned if the ClientID or SecretID are not set.
func (cfg GoogleConfig) NewEndpoint() (*Endpoint, error) {
	switch {
	case len(cfg.ClientID) == 0,
		len(cfg.ClientSecret) == 0:
		return nil, nil
	}

	// set redirect urls in gConsole Credentials' "Authorized redirect URIs" page.
	callbackURL, err := url.Parse(cfg.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("creating RedirectURL")
	}
	callbackURL.Path = GoogleCallbackURL
	redirectURL := callbackURL.String()
	scopes := []string{
		// See https://developers.google.com/identity/protocols/oauth2/scopes
		// "https://www.googleapis.com/auth/userinfo.email", // { "sub", "picture", "email", "email_verified" }
		// "https://www.googleapis.com/auth/userinfo.profile", // { "sub", "picture", "name", "given_name", "family_name" }
		"openid", // { "sub", "picture" }
	}
	conf := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOnline,
	}

	e := Endpoint{
		conf:      conf,
		opts:      opts,
		csrfToken: "selene_bananas_google_csrf_token_" + cfg.CSRFToken,
	}
	return &e, nil
}

func (e *Endpoint) authenticate(r *http.Request) (*auth, error) {
	csrf := r.FormValue("state")
	if csrf != e.csrfToken {
		return nil, fmt.Errorf("invalid csrf token: %q %q", e.csrfToken, csrf)
	}

	code := r.FormValue("code")
	if len(code) == 0 {
		return nil, nil // cancel button click
	}

	token, err := e.conf.Exchange(r.Context(), code, e.opts...)
	if err != nil {
		return nil, fmt.Errorf("exchanging Google token code: %w", err)
	}
	client := e.conf.Client(r.Context(), token)
	response, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("requesting userinfo: %w", err)
	}
	decoder := json.NewDecoder(response.Body)
	var rj tokenResponse
	if err := decoder.Decode(&rj); err != nil {
		return nil, fmt.Errorf("reading userinfo: %w", err)
	}

	a := auth{
		ID:          rj.Sub,
		AccessToken: token.AccessToken,
	}
	return &a, nil
}

func (e *Endpoint) revokeAccess(accessToken string) error {
	data := make(url.Values)
	data.Set("token", accessToken)
	_, err := http.PostForm("https://oauth2.googleapis.com/revoke", data)
	// or manually revoke at https://myaccount.google.com/connections
	if err != nil {
		return fmt.Errorf("revoking access: %w", err)
	}
	return nil
}
