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

// formatDate formats a datetime to HH:MM:SS.
func formatTime(utcSeconds int64) string {
	t := time.Unix(utcSeconds, 0) // uses local timezone
	return t.Format("15:04:05")
}

// cloneElement creates a close of the element, which should be a template.
func cloneElement(query string) js.Value {
	templateElement := QuerySelector(query)
	contentElement := templateElement.Get("content")
	clone := contentElement.Call("cloneNode", true)
	return clone
}

//ClearLog removes all log messages.
func ClearLog() {
	logScrollElement := QuerySelector(".log>.scroll")
	logScrollElement.Set("innerHTML", "")
}

// AddLog adds a log message with the specified class
func AddLog(class, text string) {
	clone := cloneElement(".log>template")
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := formatTime(time.Now().UTC().Unix())
	textContent := time + " : " + text
	logItemElement.Set("textContent", textContent)
	logItemElement.Set("className", class)
	logScrollElement := QuerySelector(".log>.scroll")
	logScrollElement.Call("appendChild", logItemElement)
	scrollHeight := logScrollElement.Get("scrollHeight")
	clientHeight := logScrollElement.Get("clientHeight")
	scrollTop := scrollHeight.Int() - clientHeight.Int()
	logScrollElement.Set("scrollTop", scrollTop)
}

// SetGameInfos updates the game-infos table with the specified game infos.
func SetGameInfos(gameInfos []game.Info, username string) {
	tbodyElement := QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
	if len(gameInfos) == 0 {
		emptyGameInfoElement := cloneElement(".no-game-info-row")
		tbodyElement.Call("appendChild", emptyGameInfoElement)
		return
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := cloneElement(".game-info-row")
		rowElement := gameInfoElement.Get("children").Index(0)
		createdAtTimeText := formatTime(gameInfo.CreatedAt)
		rowElement.Get("children").Index(0).Set("innerHTML", createdAtTimeText)
		players := strings.Join(gameInfo.Players, ", ")
		rowElement.Get("children").Index(1).Set("innerHTML", players)
		status := gameInfo.Status.String()
		rowElement.Get("children").Index(2).Set("innerHTML", status)
		if gameInfo.CanJoin(username) {
			joinGameButtonElement := cloneElement(".join-game-button")
			joinGameButtonElement.Get("children").Index(0).Set("value", int(gameInfo.ID))
			rowElement.Get("children").Index(2).Call("appendChild", joinGameButtonElement)
		}
		tbodyElement.Call("appendChild", gameInfoElement)
	}
}

// SetUsernamesReadOnly sets all of the username inputs to readonly with the specified username if it is not empty, otherwise, it removes the readonly attribute.
func SetUsernamesReadOnly(username string) {
	usernameElements := document.Call("querySelectorAll", "input.username")
	for i := 0; i < usernameElements.Length(); i++ {
		usernameElement := usernameElements.Index(i)
		switch {
		case len(username) == 0:
			usernameElement.Call("removeAttribute", "readonly")
		default:
			usernameElement.Set("value", username)
			usernameElement.Call("setAttribute", "readonly", "readonly")
		}
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

// SocketHTTPPing submits the small ping form to keep the server's http handling active.
func SocketHTTPPing() {
	pingFormElement := QuerySelector(".ping-form>form")
	var preventDefaultFunc js.Func
	preventDefaultFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		preventDefaultFunc.Release()
		return nil
	})
	pingEvent := map[string]interface{}{
		"preventDefault": preventDefaultFunc,
		"target":         pingFormElement,
	}
	pingFormElement.Call("onsubmit", js.ValueOf(pingEvent))
}

// NewWebSocket creates a new WebSocket with the specified url.
func NewWebSocket(url string) js.Value {
	return global.Get("WebSocket").New(url)
}

// Send delivers a message to the sever.
// TODO: rename to SendWebSocketMessage
func Send(m game.Message) {
	WebSocket.Send(m)
}
