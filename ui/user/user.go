//go:build js && wasm

// Package user contains code to create and edit users that can play games in the lobby.
package user

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

type (
	// User is a http/login helper.
	User struct {
		dom        DOM
		log        Log
		httpClient http.Client
		escapeR    *strings.Replacer
		Socket     Socket
	}

	// userInfo contains a user's username and points.
	userInfo struct {
		Name   string `json:"sub"`    // the JWT subject
		Points int    `json:"points"` // custom JWT field
	}

	// DOM interacts with the page.
	DOM interface {
		QuerySelector(query string) js.Value
		QuerySelectorAll(document js.Value, query string) []js.Value
		Checked(query string) bool
		SetChecked(query string, checked bool)
		Value(query string) string
		SetValue(query, value string)
		Confirm(message string) bool
		NewXHR() js.Value
		Base64Decode(a string) []byte
		StoreCredentials(form js.Value)
		RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
		NewJsEventFunc(fn func(event js.Value)) js.Func
		NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func
	}

	// Log is used to store text about connection errors.
	Log interface {
		Error(text string)
		Warning(text string)
		Clear()
	}

	// Socket is a structure that the user interacts with for the lobby and game.
	Socket interface {
		Close()
	}
)

// New creates a http/login helper struct.
func New(dom DOM, log Log, httpClient http.Client) *User {
	quoteLetters := `\^$*+?.()|[]{}`
	escapePairs := make([]string, len(quoteLetters)*2)
	for i := range quoteLetters {
		letter := quoteLetters[i : i+1]
		escapePairs[i*2] = letter
		escapePairs[i*2+1] = `\` + letter
	}
	escapeR := strings.NewReplacer(escapePairs...)
	u := User{
		dom:        dom,
		log:        log,
		httpClient: httpClient,
		escapeR:    escapeR,
	}
	return &u
}

// InitDom registers user dom functions.
func (u *User) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"logout":               u.dom.NewJsEventFunc(u.logoutButtonClick),
		"request":              u.dom.NewJsEventFuncAsync(u.request, true),
		"updateConfirmPattern": u.dom.NewJsEventFunc(u.updateConfirmPassword),
	}
	u.dom.RegisterFuncs(ctx, wg, "user", jsFuncs)
}

// login handles requesting a login when the login button is clicked.
func (u *User) login(jwt string) {
	userInfo, err := u.setInfo(jwt)
	if err != nil {
		u.log.Error("getting user from jwt: " + err.Error())
		return
	}
	u.setUsernamesReadOnly(string(userInfo.Name))
	u.dom.SetValue("input.points", strconv.Itoa(userInfo.Points))
	u.dom.SetChecked("#tab-lobby", true)
	u.dom.SetChecked("#has-login", true)
}

// logoutButtonClick handles logging out the user when the button has been clicked.
func (u *User) logoutButtonClick(event js.Value) {
	u.Logout()
	u.log.Clear()
}

// Logout logs out the user.
func (u *User) Logout() {
	u.Socket.Close()
	u.dom.SetChecked("#has-login", false)
	u.setUsernamesReadOnly("")
	u.dom.SetChecked("#tab-login-user", true)
}

// setInfo retrieves the user information from the token.
func (u *User) setInfo(jwt string) (*userInfo, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, errors.New("wanted 3 jwt parts, got " + strconv.Itoa(len(parts)))
	}
	payload := parts[1]
	jwtUserClaims := u.dom.Base64Decode(payload)
	var ui userInfo
	if err := json.Unmarshal(jwtUserClaims, &ui); err != nil {
		return nil, errors.New("parsing json: " + err.Error())
	}
	u.dom.SetValue(".jwt", jwt)
	return &ui, nil
}

// JWT gets the value of the jwt input.
func (u User) JWT() string {
	return u.dom.Value(".jwt")
}

// Username returns the username of the logged in user.
// If any problem occurs, an empty string is returned.
func (u User) Username() string {
	jwt := u.JWT()
	ui, err := u.setInfo(jwt)
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
	return u.escapeR.Replace(p)
}

// setUsernamesReadOnly sets all of the username inputs to readonly with the specified username if it is not empty, otherwise, it removes the readonly attribute.
func (u *User) setUsernamesReadOnly(username string) {
	body := u.dom.QuerySelector("body")
	usernameElements := u.dom.QuerySelectorAll(body, "input.username")
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
