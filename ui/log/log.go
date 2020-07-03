// +build js,wasm

// Package log contains shared logging code
package log

import (
	"context"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

type (
	// Log manages messages for the log div.
	Log struct{}
)

// InitDom regesters log dom functions.
func (l *Log) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	clearJsFunc := dom.NewJsFunc(l.Clear)
	dom.RegisterFunc("log", "clear", clearJsFunc)
	go func() {
		<-ctx.Done()
		clearJsFunc.Release()
		wg.Done()
	}()
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
	dom.SetCheckedQuery(".has-log", false)
	logScrollElement := dom.QuerySelector(".log>.scroll")
	logScrollElement.Set("innerHTML", "")
}

// add writes a log item with the specified class.
func (l *Log) add(class, text string) {
	dom.SetCheckedQuery(".has-log", true)
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
