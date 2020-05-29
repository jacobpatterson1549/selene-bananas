// +build js

// Package js contains the javascript bindings for the site
package js

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

var document js.Value = js.Global().Get("document")

func getElementById(id string) js.Value {
	return document.Call("getElementById", id)
}

// SetChecked sets the checked property of the element with the specified element id
func SetChecked(id string, checked bool) {
	element := getElementById(id)
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

// DormatDate formats a datetime to HH:MM:SS.
func FormatDate(time time.Time) string {
	return time.Format("15:04:05")
}

// AddLog adds a log message with the specified class
func AddLog(class, text string) {
	logItemTemplate := getElementById("log-item")
	logItemTemplateContent := logItemTemplate.Get("content")
	clone := logItemTemplateContent.Call("cloneNode", true)
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := FormatDate(time.Now())
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
		log := document.Get("log")
		log.Call("error", "unknown gameStatus: "+string(i.Status))
		return "?"
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := gameInfoTemplateContent.Call("cloneNode", true)
		rowElement := gameInfoElement.Get("children").Index(0)
		createdAt := gameInfo.CreatedAt + int64(timezoneOffsetSeconds)
		createdAtTime := time.Unix(createdAt, 0)
		createdAtTimeText := FormatDate(createdAtTime)
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
