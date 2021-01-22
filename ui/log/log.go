// +build js,wasm

// Package log contains shared logging code
package log

import (
	"context"
	"sync"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

// Log manages messages for the log div.
type Log struct{}

// InitDom registers log dom functions.
func (l *Log) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"clear": dom.NewJsFunc(l.Clear),
	}
	dom.RegisterFuncs(ctx, wg, "log", jsFuncs)
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
	dom.SetChecked("#has-log", false)
	logScrollElement := dom.QuerySelector(".log>.scroll")
	logScrollElement.Set("innerHTML", "")
}

// add writes a log item with the specified class.
func (l *Log) add(class, text string) {
	dom.SetChecked("#has-log", true)
	clone := dom.CloneElement(".log>template")
	cloneChildren := clone.Get("children")
	logItemElement := cloneChildren.Index(0)
	time := dom.FormatTime(time.Now().UTC().Unix())
	textContent := time + " : " + text
	logItemElement.Set("textContent", textContent)
	logItemElement.Set("className", class)
	logScrollElement := dom.QuerySelector(".log>.scroll")
	logScrollElement.Call("appendChild", logItemElement)
	scrollHeight := logScrollElement.Get("scrollHeight")
	clientHeight := logScrollElement.Get("clientHeight")
	scrollTop := scrollHeight.Int() - clientHeight.Int()
	logScrollElement.Set("scrollTop", scrollTop)
}
