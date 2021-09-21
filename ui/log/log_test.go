//go:build js && wasm

package log

import (
	"syscall/js"
	"testing"
)

func TestNew(t *testing.T) {
	dom := new(mockDOM)
	timeFunc := func() int64 { return 0 }
	log := New(dom, timeFunc)
	if log.dom == nil {
		t.Errorf("wanted dom to be set")
	}
	if log.TimeFunc == nil {
		t.Error("wanted timeFunc to be set")
	}
}

func TestClear(t *testing.T) {
	hideLogChecked := false
	logScroll := js.ValueOf(map[string]interface{}{
		"innerHTML": "stuff",
	})
	dom := mockDOM{
		SetCheckedFunc: func(query string, checked bool) {
			hideLogChecked = checked
		},
		QuerySelectorFunc: func(query string) js.Value {
			return logScroll
		},
	}
	log := Log{
		dom: &dom,
	}
	log.Clear()
	if !hideLogChecked {
		t.Errorf("wanted hide-log checked to be checked")
	}
	if want, got := "", logScroll.Get("innerHTML").String(); want != got {
		t.Errorf("wanted log scroll to be cleared, got %v", got)
	}
}

func TestLogClass(t *testing.T) {
	tests := []struct {
		fn        func(log Log) func(string)
		wantClass string
	}{
		{
			fn:        func(log Log) func(string) { return log.Info },
			wantClass: "info",
		},
		{
			fn:        func(log Log) func(string) { return log.Warning },
			wantClass: "warning",
		},
		{
			fn:        func(log Log) func(string) { return log.Error },
			wantClass: "error",
		},
		{
			fn:        func(log Log) func(string) { return log.Chat },
			wantClass: "chat",
		},
	}
	for i, test := range tests {
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
		logItemElement := js.ValueOf(map[string]interface{}{})
		logTemplate := js.ValueOf(map[string]interface{}{
			"children": []interface{}{
				logItemElement,
			},
		})
		hideLogChecked := true
		dom := mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return logScroll
			},
			CloneElementFunc: func(query string) js.Value {
				return logTemplate
			},
			SetCheckedFunc: func(query string, checked bool) {
				hideLogChecked = checked
			},
			FormatTimeFunc: func(utcSeconds int64) string {
				return string(rune(utcSeconds))
			},
		}
		log := Log{
			dom:      &dom,
			TimeFunc: func() int64 { return 65 },
		}
		message := "log_message"
		wantMessage := "A : " + message
		logFn := test.fn(log)
		logFn(message)
		if hideLogChecked {
			t.Errorf("wanted hide-log not to be checked")
		}
		if want, got := wantMessage, logItemElement.Get("textContent").String(); want != got {
			t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, want, got)
		}
		if want, got := test.wantClass, logItemElement.Get("className").String(); want != got {
			t.Errorf("Test %v: classes not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := 236, logScroll.Get("scrollTop").Int(); want != got {
			t.Errorf("scrollTops not equal: wanted %v, got %v", want, got)
		}
		if !logScroll.Get("TEST::APPENDED_LOG_ITEM").Equal(logItemElement) {
			t.Error("wanted recentLog item to be appended to log")
		}
		appendChild.Release()
	}
}
