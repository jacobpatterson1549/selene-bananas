//go:build js && wasm

// Package log contains shared logging code
package log

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

// Log manages messages for the log div.
type Log struct {
	// DOM contains utilities for logging  messages
	dom *ui.DOM
	// TimeFunc is a function which should supply the current time since the unix epoch.
	// This is used for logging message timestamps
	TimeFunc func() int64
}

// New creates a new socket.
func New(dom *ui.DOM, timeFunc func() int64) *Log {
	l := Log{
		dom: dom,
		TimeFunc: timeFunc,
	}
	return &l
}

// InitDom registers log dom functions.
func (l *Log) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"clear": l.dom.NewJsFunc(l.Clear),
	}
	l.dom.RegisterFuncs(ctx, wg, "log", jsFuncs)
}

// Info logs an info-styled message.
func (l *Log) Info(text string) {
	l.add("info", text)
}

// Warning logs an warning-styled message.
func (l *Log) Warning(text string) {
	l.add("warning", text)
}

// Error logs an error-styled message.
func (l *Log) Error(text string) {
	l.add("error", text)
}

// Chat logs an chat-styled message.
func (l *Log) Chat(text string) {
	l.add("chat", text)
}

// Clear clears the log.
func (l *Log) Clear() {
	l.dom.SetChecked("#hide-log", true)
	logScrollElement := l.dom.QuerySelector(".log>.scroll")
	logScrollElement.Set("innerHTML", "")
}

// add writes a log item with the specified class.
func (l *Log) add(class, text string) {
	l.dom.SetChecked("#hide-log", false)
	clone := l.dom.CloneElement(".log>template")
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := l.dom.FormatTime(l.TimeFunc())
	textContent := time + " : " + text
	logItemElement.Set("textContent", textContent)
	logItemElement.Set("className", class)
	logScrollElement := l.dom.QuerySelector(".log>.scroll")
	logScrollElement.Call("appendChild", logItemElement)
	scrollHeight := logScrollElement.Get("scrollHeight")
	clientHeight := logScrollElement.Get("clientHeight")
	scrollTop := scrollHeight.Int() - clientHeight.Int()
	logScrollElement.Set("scrollTop", scrollTop)
}
