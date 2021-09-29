//go:build js && wasm

package http

import (
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

func TestDoRequest(t *testing.T) {
	t.Run("nil DOM", func(t *testing.T) {
		var r Request
		var c Client
		_, err := c.Do(nil, r)
		if err == nil {
			t.Error("wanted error when dom is nil")
		}
	})
	tests := []struct {
		Client
		Request
		eventType          string
		want               Response
		wantOk             bool
		responseStatus     int
		responseBody       string
		wantOpenMethod     string
		wantOpenURL        string
		wantTimeout        int
		wantRequestHeaders map[string]string
		wantBody           string
	}{
		{
			eventType: "timeout",
		},
		{
			eventType: "abort",
		},
		{
			eventType: "error",
		},
		{
			eventType: "load",
			want: Response{
				Code: 200,
				Body: "successful body 16",
			},
			wantOk: true,
			Client: Client{
				Timeout: 15 * time.Second,
			},
			Request: Request{
				Method: "GET",
				URL:    "https://example.com",
				Headers: map[string]string{
					"Content-Type":  "text/plain",
					"Authorization": "Bearer s3cr3t",
				},
				Body: "[[request body]]",
			},
			responseStatus: 200,
			responseBody:   "successful body 16",
			wantOpenMethod: "GET",
			wantOpenURL:    "https://example.com",
			wantRequestHeaders: map[string]string{
				"Content-Type":  "text/plain",
				"Authorization": "Bearer s3cr3t",
			},
			wantBody: "[[request body]]",
		},
	}
	for i, test := range tests {
		var jsFuncs []js.Func
		gotOpenMethod := ""
		gotTimeout := 0
		gotOpenURL := ""
		gotRequestHeaders := make(map[string]string)
		gotBody := ""
		dom := mockDOM{
			NewXHRFunc: func() js.Value {
				eventListeners := make(map[string]js.Value, 4)
				open := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					gotOpenMethod = args[0].String()
					gotOpenURL = args[1].String()
					return nil
				})
				setRequestHeader := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					k, v := args[0].String(), args[1].String()
					gotRequestHeaders[k] = v
					return nil
				})
				addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					eventType, handler := args[0].String(), args[1]
					eventListeners[eventType] = handler
					return nil
				})
				send := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					gotBody = args[0].String()
					handler := eventListeners[test.eventType]
					event := js.ValueOf(map[string]interface{}{
						"type": test.eventType,
					})
					handler.Invoke(event)
					return nil
				})
				jsFuncs = append(jsFuncs, open, setRequestHeader, addEventListener, send)
				return js.ValueOf(map[string]interface{}{
					"open":             open,
					"setRequestHeader": setRequestHeader,
					"addEventListener": addEventListener,
					"send":             send,
					"status":           test.responseStatus,
					"response":         test.responseBody,
				})
			},
			NewJsEventFuncFunc: func(fn func(event js.Value)) js.Func {
				f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					event := args[0]
					fn(event)
					return nil
				})
				jsFuncs = append(jsFuncs, f)
				return f
			},
		}
		got, err := test.Client.Do(&dom, test.Request)
		for _, f := range jsFuncs {
			f.Release()
		}
		switch {
		case test.wantOpenMethod != gotOpenMethod:
			t.Errorf("xhr open methods not equal:\nwanted: %v\ngot:    %v", test.wantOpenMethod, gotOpenMethod)
		case test.wantTimeout != gotTimeout:
			t.Errorf("xhr timeouts equal:\nwanted: %v\ngot:    %v", test.wantTimeout, gotTimeout)
		case test.wantOpenURL != gotOpenURL:
			t.Errorf("xhr open urls not equal:\nwanted: %v\ngot:    %v", test.wantOpenURL, gotOpenURL)
		case test.wantBody != gotBody:
			t.Errorf("xhr bodies not equal:\nwanted: %v\ngot:    %v", test.wantBody, gotBody)
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error when eventType is %v, got %#v", i, test.eventType, *got)
			}
		case !reflect.DeepEqual(test.wantRequestHeaders, gotRequestHeaders): // checking after wantOk because wantRequestHeaders is often nil
			t.Errorf("xhr open args not equal:\nwanted: %v\ngot:    %v", test.wantRequestHeaders, gotRequestHeaders)
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}
