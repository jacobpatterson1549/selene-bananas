//go:build js && wasm

package log

import (
	"context"
	"sync"
	"syscall/js"
	"testing"
	"time"
)

func TestInitDom(t *testing.T) {
	// TODO: test this in cmd/ui/config_test.go
	ctx := context.Background()
	var wg sync.WaitGroup
	log := new(Log)
	log.InitDom(ctx, &wg)
	if js.Global().Get("log").Get("clear").Type() != js.TypeFunction {
		t.Error("wanted clear func to be registered to dom")
	}
}

func TestClear(t *testing.T) {
	logFuncs := initLog(t)
	log := new(Log)
	log.Clear()
	hideLog := js.Global().Get("document").Call("querySelector", "#hide-log")
	if want, got := true, hideLog.Get("checked").Bool(); want != got {
		t.Errorf("wanted hide-log checked to be %v, got %v", want, got)
	}
	logScroll := js.Global().Get("document").Call("querySelector", ".log>.scroll")
	if want, got := "", logScroll.Get("innerHTML").String(); want != got {
		t.Errorf("wanted log scroll to be cleared, got %v", got)
	}
	releaseLog(logFuncs)
}

func TestLogClass(t *testing.T) {
	oldTZ := time.Local
	loc, _ := time.LoadLocation("America/Los_Angeles")
	time.Local = loc
	defer func() { time.Local = oldTZ }()
	log := Log{
		TimeFunc: func() int64 { return 1632078828 }, // 12:13:48 PDT 20121/09/19
	}
	tests := []struct {
		fn        func(string)
		wantClass string
	}{
		{
			log.Info,
			"info",
		},
		{
			log.Warning,
			"warning",
		},
		{
			log.Error,
			"error",
		},
		{
			log.Chat,
			"chat",
		},
	}
	for i, test := range tests {
		message := "log_message"
		wantMessage := "12:13:48 : " + message
		logFuncs := initLog(t)
		test.fn(message)
		hideLog := js.Global().Get("document").Call("querySelector", "#hide-log")
		if want, got := false, hideLog.Get("checked").Bool(); want != got {
			t.Errorf("wanted hide-log checked to be %v, got %v", want, got)
		}
		logItemElement := js.Global().Get("document").Call("querySelector", "TEST::APPENDED_LOG_ITEM")
		if want, got := wantMessage, logItemElement.Get("textContent").String(); want != got {
			t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, want, got)
		}
		if want, got := test.wantClass, logItemElement.Get("className").String(); want != got {
			t.Errorf("Test %v: classes not equal: wanted %v, got %v", i, want, got)
		}
		checkLogCalls(t, logItemElement)
		releaseLog(logFuncs)
	}
}

func initLog(t *testing.T) []js.Func {
	t.Helper()
	logItem := js.ValueOf(map[string]interface{}{})
	children := js.ValueOf([]interface{}{logItem})
	clone := js.ValueOf(map[string]interface{}{
		"children": children,
	})
	cloneNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		deep := args[0].Bool()
		if !deep {
			t.Errorf("cloneNode not called with param deep=true")
		}
		return clone
	})
	hideLog := js.ValueOf(map[string]interface{}{
		"checked": "",
	})
	logTemplateContent := js.ValueOf(map[string]interface{}{
		"cloneNode": cloneNode,
	})
	logTemplate := js.ValueOf(map[string]interface{}{
		"content": logTemplateContent,
	})
	appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logItemElement := args[0]
		this.Set("TEST::APPENDED_LOG_ITEM", logItemElement)
		return nil
	})
	logScroll := js.ValueOf(map[string]interface{}{
		"appendChild":  appendChild,
		"scrollHeight": 487,
		"clientHeight": 251,
		"innerHTML":    "SHOULD BE DELETED BY Clear()",
	})
	querySelector := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		query := args[0].String()
		switch query {
		case "#hide-log":
			return hideLog
		case ".log>template":
			return logTemplate
		case ".log>.scroll":
			return logScroll
		case "TEST::APPENDED_LOG_ITEM":
			return logItem
		}
		return nil
	})
	document := js.ValueOf(map[string]interface{}{
		"querySelector": querySelector,
	})
	js.Global().Set("document", document)
	return []js.Func{querySelector, cloneNode, appendChild}
}

func checkLogCalls(t *testing.T, recentLog js.Value) {
	t.Helper()
	logScroll := js.Global().Get("document").Call("querySelector", ".log>.scroll")
	if want, got := 236, logScroll.Get("scrollTop").Int(); want != got {
		t.Errorf("scrollTops not equal: wanted %v, got %v", want, got)
	}
	if !logScroll.Get("TEST::APPENDED_LOG_ITEM").Equal(recentLog) {
		t.Error("wanted recentLog item to be appended to log")
	}
}

func releaseLog(logFuncs []js.Func) {
	for _, f := range logFuncs {
		f.Release()
	}
}
