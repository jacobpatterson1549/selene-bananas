// +build js,wasm

// Package user contains code to create and edit users that can play games in the lobby.
package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// jwt is a java-web token.
	jwt string
	// User is a http/login helper.
	User struct {
		log        *log.Log
		httpClient *http.Client
		escapeRE   regexp.Regexp
		Socket     Socket
	}

	// Config contains the parameters to create a User.
	Config struct {
		Log *log.Log
	}

	// userInfo contains a user's username and points.
	userInfo struct {
		username string
		points   int
	}

	// Socket is a structure that the user interacts with for the lobby and game.
	Socket interface {
		Close()
	}
)

// New creates a http/login helper struct.
func (cfg Config) New(httpClient *http.Client) *User {
	escapeRE := regexp.MustCompile("([" + regexp.QuoteMeta(`\^$*+?.()|[]{}`) + "])")
	u := User{
		log:        cfg.Log,
		httpClient: httpClient,
		escapeRE:   *escapeRE,
	}
	return &u
}

// InitDom regesters user dom functions.
func (u *User) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	logout := dom.NewJsEventFunc(u.logoutButton)
	request := dom.NewJsEventFuncAsync(u.request, true)
	updateConfirmPassword := dom.NewJsEventFunc(u.updateConfirmPassword)
	dom.RegisterFunc("user", "logout", logout)
	dom.RegisterFunc("user", "request", request)
	dom.RegisterFunc("user", "updateConfirmPattern", updateConfirmPassword)
	go dom.ReleaseJsFuncsOnDone(ctx, wg, logout, request, updateConfirmPassword)
}

func (u *User) login(token string) {
	dom.SetValue(".jwt", token)
	j := jwt(token)
	ui, err := j.getUser()
	if err != nil {
		u.log.Error("getting user from jwt: " + err.Error())
		return
	}
	u.setUsernamesReadOnly(string(ui.username))
	dom.SetValue("input.points", strconv.Itoa(ui.points))
	dom.SetCheckedQuery("#tab-lobby", true)
	dom.SetCheckedQuery(".has-login", true)
}

func (u *User) logoutButton(event js.Value) {
	u.Logout()
	u.log.Clear()
}

// Logout logs out the user
func (u *User) Logout() {
	u.Socket.Close()
	dom.SetCheckedQuery(".has-login", false)
	u.setUsernamesReadOnly("")
	dom.SetCheckedQuery("#tab-login-user", true)
}

func (j jwt) getUser() (*userInfo, error) {
	parts := strings.Split(string(j), ".")
	if len(parts) != 3 {
		return nil, errors.New("expected 3 jwt parts")
	}
	payload := parts[1]
	jwtUsernameClaims, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("decoding user info: " + err.Error())
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(jwtUsernameClaims, &claims); err != nil {
		return nil, errors.New("destructuring user info" + err.Error())
	}
	sub, ok := claims["sub"]
	if !ok {
		return nil, errors.New("no 'sub' field in user claims")
	}
	username, ok := sub.(string)
	if !ok {
		return nil, errors.New("sub is not a string")
	}
	points, ok := claims["points"]
	if !ok {
		return nil, errors.New("no 'points' field in user claims")
	}
	pointsF, ok := points.(float64)
	if !ok {
		return nil, errors.New("points is not a number")
	}
	u := userInfo{
		username: username,
		points:   int(pointsF),
	}
	return &u, nil
}

// JWT gets the value of the jwt input.
func (u User) JWT() string {
	return dom.GetValue(".jwt")
}

// Username returns the username of the logged in user.
// If any problem occurs, an empty string is returned.
func (u User) Username() string {
	j := jwt(u.JWT())
	ui, err := j.getUser()
	if err != nil {
		return ""
	}
	return ui.username
}

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
