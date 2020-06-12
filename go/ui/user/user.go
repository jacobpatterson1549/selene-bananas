// +build js

// Package user contains code to view available games and to close the websocket.
package user

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

type (
	jwt  string
	user struct {
		username string
		points   int
	}
)

func login(token string) {
	js.SetValue("jwt", token)
	j := jwt(token)
	u, err := j.getUser()
	if err != nil {
		log.Error("getting user from jwt: " + err.Error())
		return
	}
	js.SetUsernamesReadOnly(u.username)
	js.SetPoints(u.points)
	js.SetChecked("tab-4", true) // lobby tab
	hasLogin(true)
}

// Logout logs out the user
func Logout() {
	hasLogin(false)
	js.SetChecked("has-game", false)
	js.SetUsernamesReadOnly("")
	js.SetChecked("tab-1", true) // login tab
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
	js.SetChecked("has-login", loggedIn)
}

// JWT gets the value of the jwt input.
func JWT() string {
	return js.GetValue("jwt")
}
