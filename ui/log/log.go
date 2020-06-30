// +build js,wasm

// Package log contains shared logging code
package log

import (
	"context"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

// InitDom regesters log dom functions.
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	clearJsFunc := dom.NewJsFunc(Clear)
	dom.RegisterFunc("log", "clear", clearJsFunc)
	go func() {
		<-ctx.Done()
		clearJsFunc.Release()
		wg.Done()
	}()
}

// Info logs an info-styled message.
func Info(text string) {
	log("info", text)
}

// Warning logs an warning-styled message.
func Warning(text string) {
	log("warning", text)
}

// Error logs an error-styled message.
func Error(text string) {
	log("error", text)
}

// Chat logs an chat-styled message.
func Chat(text string) {
	log("chat", text)
}

// Clear clears the log.
func Clear() {
	dom.SetCheckedQuery(".has-log", false)
	logScrollElement := dom.QuerySelector(".log>.scroll")
	logScrollElement.Set("innerHTML", "")
}

// log writes a log item with the specified class.
func log(class, text string) {
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
