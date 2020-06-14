// +build js,wasm

// Package user contains code to view available games and to close the websocket.
package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

type (
	jwt  string
	user struct {
		username string
		points   int
	}
)

// InitDom regesters user dom functions.
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	logoutJsFunc := dom.NewJsFunc(Logout)
	requestJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		f := dom.NewForm(event)
		go request(f)
	})
	updateConfirmPasswordJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		password1InputElement := event.Get("target")
		password2InputElement := password1InputElement.
			Get("parentElement").
			Get("nextElementSibling").
			Call("querySelector", ".password2")
		password1Value := password1InputElement.Get("value").String()
		passwordRegex := escapePassword(password1Value)
		password2InputElement.Set("pattern", passwordRegex)
	})
	dom.RegisterFunc("user", "logout", logoutJsFunc)
	dom.RegisterFunc("user", "request", requestJsFunc)
	dom.RegisterFunc("user", "updateConfirmPattern", updateConfirmPasswordJsFunc)
	go func() {
		<-ctx.Done()
		logoutJsFunc.Release()
		requestJsFunc.Release()
		updateConfirmPasswordJsFunc.Release()
		wg.Done()
	}()
}

func login(token string) {
	dom.SetValue("jwt", token)
	j := jwt(token)
	u, err := j.getUser()
	if err != nil {
		log.Error("getting user from jwt: " + err.Error())
		return
	}
	dom.SetUsernamesReadOnly(u.username)
	dom.SetPoints(u.points)
	dom.SetChecked("tab-4", true) // lobby tab
	hasLogin(true)
}

// Logout logs out the user
func Logout() {
	hasLogin(false)
	dom.SetChecked("has-game", false)
	dom.SetUsernamesReadOnly("")
	dom.SetChecked("tab-1", true) // login tab
}

func (j jwt) getUser() (*user, error) {
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
	pointsI, ok := points.(float64)
	if !ok {
		return nil, errors.New("points is not a number")
	}
	u := user{
		username: username,
		points:   int(pointsI),
	}
	return &u, nil
	return &user{}, nil
}

// hasLogin sets the checked property of the has-login input.
func hasLogin(loggedIn bool) {
	dom.SetChecked("has-login", loggedIn)
}

// JWT gets the value of the jwt input.
func JWT() string {
	return dom.GetValue("jwt")
}

// escapePassword escapes the password for html dom input pattern matching using Regexp.
func escapePassword(p string) string {
	escapeRE := regexp.MustCompile("([" + regexp.QuoteMeta(`\^$*+?.()|[]{}`) + "])")
	return string(escapeRE.ReplaceAll([]byte(p), []byte(`\$1`)))
}
