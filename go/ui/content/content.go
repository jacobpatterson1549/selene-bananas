// +build js

// Package content contains shared functions
// TODO: move to user package
package content

import (
	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
)

// SetLoggedIn sets the checked property of the has-login input.
func SetLoggedIn(loggedIn bool) {
	js.SetChecked("has-login", loggedIn)
}

// IsLoggedIn returns whether the has-login input is checked.
func IsLoggedIn() bool {
	return js.GetChecked("has-login")
}

// SetErrorMessage sets the inner html of the error-message div
// TODO: replace error-message div with calls to log.error()
func SetErrorMessage(text string) {
	js.SetInnerHTML("error-message", text)
}

// GetJWT gets the value of the jwt input.
func GetJWT() string {
	return js.GetValue("jwt")
}

// SetJWT sets the value of the jwt input.
func SetJWT(jwt string) {
	js.SetValue("jwt", jwt)
}
