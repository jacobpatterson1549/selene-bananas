// +build js,wasm

// Package log contains shared logging code
package log

import (
	"context"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
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
	dom.SetCheckedQuery(".log-visible", false)
	dom.ClearLog()
}

// log writes a log item with the specified class.
func log(class, text string) {
	dom.SetCheckedQuery(".log-visible", true)
	dom.AddLog(class, text)
}
