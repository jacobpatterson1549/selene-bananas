// +build js

// Package log contains shared logging code
package log

import (
	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
)

// Info logs an info-styled message.
func Info(text string) {
	log("info", text)
}

// Warning logs an warning-styled message.
func Warning(text string) {
	log("warning", text)
}

// Info logs an error-styled message.
func Error(text string) {
	log("error", text)
}

// Info logs an chat-styled message.
func Chat(text string) {
	log("chat", text)
}

// Clear clears the log.
func Clear() {
	js.SetChecked("has-log", false)
	js.SetInnerHTML("log-scroll", "")
}

// log writes a log item with the specified class.
func log(class, text string) {
	js.SetChecked("has-log", true)
	js.AddLog(class, text)
}
