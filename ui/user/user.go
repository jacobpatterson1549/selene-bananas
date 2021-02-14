// +build js,wasm

// Package user contains code to create and edit users that can play games in the lobby.
package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/http"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// User is a http/login helper.
	User struct {
		log        *log.Log
		httpClient http.Client
		escapeRE   regexp.Regexp
		Socket     Socket
	}

	// userInfo contains a user's username and points.
	userInfo struct {
		Name   string `json:"sub"`    // the JWT subject
		Points int    `json:"points"` // custom JWT field
	}

	// Socket is a structure that the user interacts with for the lobby and game.
	Socket interface {
		Close()
	}
)

// New creates a http/login helper struct.
func New(log *log.Log, httpClient http.Client) *User {
	escapeRE := regexp.MustCompile("([" + regexp.QuoteMeta(`\^$*+?.()|[]{}`) + "])")
	u := User{
		log:        log,
		httpClient: httpClient,
		escapeRE:   *escapeRE,
	}
	return &u
}

// InitDom registers user dom functions.
func (u *User) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"logout":               dom.NewJsEventFunc(u.logoutButtonClick),
		"request":              dom.NewJsEventFuncAsync(u.request, true),
		"updateConfirmPattern": dom.NewJsEventFunc(u.updateConfirmPassword),
	}
	dom.RegisterFuncs(ctx, wg, "user", jsFuncs)
}

// login handles requesting a login when the login button is clicked.
func (u *User) login(jwt string) {
	dom.SetValue(".jwt", jwt)
	ui, err := u.info(jwt)
	if err != nil {
		u.log.Error("getting user from jwt: " + err.Error())
		return
	}
	u.setUsernamesReadOnly(string(ui.Name))
	dom.SetValue("input.points", strconv.Itoa(ui.Points))
	dom.SetChecked("#tab-lobby", true)
	dom.SetChecked("#has-login", true)
}

// logoutButtonClick handles logging out the user when the button has been clicked.
func (u *User) logoutButtonClick(event js.Value) {
	u.Logout()
	u.log.Clear()
}

// Logout logs out the user.
func (u *User) Logout() {
	u.Socket.Close()
	dom.SetChecked("#has-login", false)
	u.setUsernamesReadOnly("")
	dom.SetChecked("#tab-login-user", true)
}

// info retrieves the user information from the token.
func (User) info(jwt string) (*userInfo, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, errors.New("wanted 3 jwt parts, got " + strconv.Itoa(len(parts)))
	}
	payload := parts[1]
	jwtUserClaims, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("decoding user info: " + err.Error())
	}
	var ui userInfo
	if err := json.Unmarshal(jwtUserClaims, &ui); err != nil {
		return nil, errors.New("parsing json: " + err.Error())
	}
	return &ui, nil
}

// JWT gets the value of the jwt input.
func (u User) JWT() string {
	return dom.Value(".jwt")
}

// Username returns the username of the logged in user.
// If any problem occurs, an empty string is returned.
func (u User) Username() string {
	jwt := u.JWT()
	ui, err := u.info(jwt)
	if err != nil {
		return ""
	}
	return ui.Name
}

// updateConfirmPassword updates the pattern of the password confirm input to expect the content of the password input.
func (u *User) updateConfirmPassword(event js.Value) {
	password1InputElement := event.Get("target")
	password2InputElement := password1InputElement.
		Get("parentElement").
		Get("nextElementSibling").
		Get("lastElementChild")
	password1Value := password1InputElement.Get("value").String()
	passwordRegex := u.escapePassword(password1Value)
	password2InputElement.Set("pattern", passwordRegex)
}

// escapePassword escapes the password for html dom input pattern matching using Regexp.
func (u User) escapePassword(p string) string {
	return string(u.escapeRE.ReplaceAll([]byte(p), []byte(`\$1`)))
}

// setUsernamesReadOnly sets all of the username inputs to readonly with the specified username if it is not empty, otherwise, it removes the readonly attribute.
func (u *User) setUsernamesReadOnly(username string) {
	body := dom.QuerySelector("body")
	usernameElements := dom.QuerySelectorAll(body, "input.username")
	for _, usernameElement := range usernameElements {
		switch {
		case len(username) == 0:
			usernameElement.Call("removeAttribute", "readonly")
		default:
			usernameElement.Set("value", username)
			usernameElement.Call("setAttribute", "readonly", "readonly")
		}
	}
}
