// +build js

// Package js contains the javascript bindings for the site
package js

import (
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

var document js.Value = js.Global().Get("document")

func getElementById(id string) js.Value {
	return document.Call("getElementById", id)
}

func querySelector(query string) js.Value {
	return document.Call("querySelector", query)
}

// SetChecked sets the checked property of the element with the specified element id
func SetChecked(id string, checked bool) {
	element := getElementById(id)
	element.Set("checked", checked)
}

// SetCheckedQuery sets the checked property of the element with the specified element query.
func SetCheckedQuery(query string, checked bool) {
	element := querySelector(query)
	element.Set("checked", checked)
}

// GetChecked returns whether the element has a checked value of true.
func GetChecked(id string) bool {
	element := getElementById(id)
	checked := element.Get("checked")
	return checked.Bool()
}

// SetInnerHTML sets the inner html of the element with the specified id.
// TODO: audit SetInnerHTML usage
func SetInnerHTML(id string, innerHTML string) {
	element := getElementById(id)
	element.Set("innerHTML", innerHTML)
}

// GetValue gets the value of the input element with the specified id.
func GetValue(id string) string {
	element := getElementById(id)
	value := element.Get("value")
	return value.String()

}

// SetValue sets the value of the input element with the specified id.
func SetValue(id, value string) {
	element := getElementById(id)
	element.Set("value", value)
}

// SetValueQuery sets the value of the input element with the specified element query..
func SetValueQuery(query, value string) {
	element := querySelector(query)
	element.Set("value", value)
}

// SetButtonDisabledQuery sets the button element with the id disabled or enabled.
func SetButtonDisabled(id string, disabled bool) {
	element := getElementById(id)
	element.Set("disabled", disabled)
}

// DormatDate formats a datetime to HH:MM:SS.
func FormatTime(time time.Time) string {
	return time.Format("15:04:05")
}

// AddLog adds a log message with the specified class
func AddLog(class, text string) {
	logItemTemplate := getElementById("log-item")
	logItemTemplateContent := logItemTemplate.Get("content")
	clone := logItemTemplateContent.Call("cloneNode", true)
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := FormatTime(time.Now())
	textContent := time + " : " + text
	logItemElement.Set("textContent", textContent)
	logItemElement.Set("className", class)
	logScrollElement := getElementById("log-scroll")
	logScrollElement.Call("appendChild", logItemElement)
	scrollHeight := logScrollElement.Get("scrollHeight")
	clientHeight := logScrollElement.Get("clientHeight")
	scrollTop := scrollHeight.Int() - clientHeight.Int()
	logScrollElement.Set("scrollTop", scrollTop)
}

// SetGameInfos updates the game-infos table with the specified game infos.
func SetGameInfos(gameInfos []game.Info) {
	tbodyElement := document.Call("querySelector", "table#game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
	if len(gameInfos) == 0 {
		// TODO: create clone helper function
		emptyGameInfoTemplate := getElementById("no-game-info-row")
		emptyGameInfoTemplateContent := emptyGameInfoTemplate.Get("content")
		emptyGameInfoElement := emptyGameInfoTemplateContent.Call("cloneNode", true)
		tbodyElement.Call("appendChild", emptyGameInfoElement)
		return
	}
	// println("TODO: setGameInfos (total=" + string(len(gameInfos)) + ")")
	// println(fmt.Sprintf("gameInfos: %v", gameInfos)) // DELETEME
	gameInfoTemplate := getElementById("game-info-row")
	gameInfoTemplateContent := gameInfoTemplate.Get("content")
	_, timezoneOffsetSeconds := time.Now().Zone()
	getStatus := func(i game.Info) string {
		switch i.Status {
		case game.NotStarted:
			return "Not Started"
		case game.InProgress:
			return "In Progress"
		case game.Finished:
			return "Finished"
		}
		return "?"
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := gameInfoTemplateContent.Call("cloneNode", true)
		rowElement := gameInfoElement.Get("children").Index(0)
		createdAt := gameInfo.CreatedAt + int64(timezoneOffsetSeconds)
		createdAtTime := time.Unix(createdAt, 0)
		createdAtTimeText := FormatTime(createdAtTime)
		rowElement.Get("children").Index(0).Set("innerHTML", createdAtTimeText)
		players := strings.Join(gameInfo.Players, ", ")
		rowElement.Get("children").Index(1).Set("innerHTML", players)
		status := getStatus(gameInfo)
		rowElement.Get("children").Index(2).Set("innerHTML", status)
		if gameInfo.CanJoin {
			joinGameButtonTemplate := getElementById("join-game-button")
			joinGameButtonElement := joinGameButtonTemplate.Get("content").Call("cloneNode", true)
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

// SetPoints sets the value of the points input
func SetPoints(points int) {
	pointsElement := document.Call("querySelector", "input.points")
	pointsElement.Set("value", points)
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
	result := js.Global().Call("confirm", message)
	return result.Bool()
}

// SocketHTTPPing submits the small ping form to keep the server's http handling active.
func SocketHTTPPing() {
	pingFormElement := getElementById("ping-form")
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

// Send delivers a message to the sever.
// TODO: near-duplicate code in ui.go
func Send(m game.Message) {
	_websocket := js.Global().Get("websocket").Get("_websocket")
	if !_websocket.IsUndefined() && !_websocket.IsNull() && _websocket.Get("readyState").Int() == 1 {
		messageJSON, err := json.Marshal(m)
		if err != nil {
			panic("marshalling socket message to send: " + err.Error())
			return
		}
		_websocket.Call("send", js.ValueOf(string(messageJSON)))
	}
}

// CloseWebsocket closes the websocket
// TODO: investigate more un-circular use
func CloseWebsocket() {
	websocket := js.Global().Get("websocket")
	websocket.Call("close")
}
