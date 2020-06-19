// +build js,wasm

// Package dom contains the javascript bindings for the site
package dom

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

type (
	// Socket represents a method set of functions to communicate directly with the server
	Socket interface {
		Send(m game.Message)
		Close()
	}
)

var (
	global   js.Value = js.Global()
	document js.Value = global.Get("document")
	// WebSocket is the Socket used for game Communication
	WebSocket Socket // TODO: HACK! (circular dependency)
)

// QuerySelector returns the first element returned by the query from root of the document.
func QuerySelector(query string) js.Value {
	return document.Call("querySelector", query)
}

// QuerySelectorAll returns an array of the elements returned by the query from root of the document.
func QuerySelectorAll(query string) js.Value {
	return document.Call("querySelectorAll", query)
}

// SetCheckedQuery sets the checked property of the element.
func SetCheckedQuery(query string, checked bool) {
	element := QuerySelector(query)
	element.Set("checked", checked)
}

// GetCheckedQuery returns whether the element has a checked value of true.
func GetCheckedQuery(query string) bool {
	element := QuerySelector(query)
	checked := element.Get("checked")
	return checked.Bool()
}

// GetValue gets the value of the input element.
func GetValue(query string) string {
	element := QuerySelector(query)
	value := element.Get("value")
	return value.String()
}

// SetValue sets the value of the input element.
func SetValue(query, value string) {
	element := QuerySelector(query)
	element.Set("value", value)
}

// FormatTime formats a datetime to HH:MM:SS.
func FormatTime(utcSeconds int64) string {
	t := time.Unix(utcSeconds, 0) // uses local timezone
	return t.Format("15:04:05")
}

// CloneElement creates a close of the element, which should be a template.
func CloneElement(query string) js.Value {
	templateElement := QuerySelector(query)
	contentElement := templateElement.Get("content")
	clone := contentElement.Call("cloneNode", true)
	return clone
}

// SetGameInfos updates the game-infos table with the specified game infos.
func SetGameInfos(gameInfos []game.Info, username string) {
	tbodyElement := QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
	if len(gameInfos) == 0 {
		emptyGameInfoElement := CloneElement(".no-game-info-row")
		tbodyElement.Call("appendChild", emptyGameInfoElement)
		return
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := CloneElement(".game-info-row")
		rowElement := gameInfoElement.Get("children").Index(0)
		createdAtTimeText := FormatTime(gameInfo.CreatedAt)
		rowElement.Get("children").Index(0).Set("innerHTML", createdAtTimeText)
		players := strings.Join(gameInfo.Players, ", ")
		rowElement.Get("children").Index(1).Set("innerHTML", players)
		status := gameInfo.Status.String()
		rowElement.Get("children").Index(2).Set("innerHTML", status)
		if gameInfo.CanJoin(username) {
			joinGameButtonElement := CloneElement(".join-game-button")
			joinGameButtonElement.Get("children").Index(0).Set("value", int(gameInfo.ID))
			rowElement.Get("children").Index(2).Call("appendChild", joinGameButtonElement)
		}
		tbodyElement.Call("appendChild", gameInfoElement)
	}
}

// EnableSubmitButtons removes the disabled attribute from all submit buttons
func EnableSubmitButtons() {
	disabledSubmitButtons := document.Call("querySelectorAll", `input[type="submit"]:disabled`)
	for i := 0; i < disabledSubmitButtons.Length(); i++ {
		submitButton := disabledSubmitButtons.Index(i)
		submitButton.Set("disabled", false)
	}
}

// StoreCredentials attempts to save the credentials for the login, if browser wants to
func StoreCredentials(username, password string) {
	passwordCredential := document.Get("PasswordCredential")
	if passwordCredential.Truthy() {
		c := map[string]string{
			"id":       username,
			"password": password,
		}
		document.Get("credentials").Call("store", c)
	}
}

// Confirm shows a popup asking the user a yes/no question.
// The true return value implies the "yes" choice.
func Confirm(message string) bool {
	result := global.Call("confirm", message)
	return result.Bool()
}

// NewWebSocket creates a new WebSocket with the specified url.
func NewWebSocket(url string) js.Value {
	return global.Get("WebSocket").New(url)
}

// SendWebSocketMessage delivers a message to the sever.
func SendWebSocketMessage(m game.Message) {
	WebSocket.Send(m)
}
