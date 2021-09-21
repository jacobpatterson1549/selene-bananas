//go:build js && wasm

package http

import (
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
		mockXMLHttpRequest
		eventType string
		want      Response
		wantOk    bool
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
			mockXMLHttpRequest: mockXMLHttpRequest{
				status:            200,
				response:          "successful body 16",
				wantOpenMethod:    "GET",
				wantOpenURL:       "https://example.com",
				gotRequestHeaders: make(map[string]string, 2),
				wantRequestHeaders: map[string]string{
					"Content-Type":  "text/plain",
					"Authorization": "Bearer s3cr3t",
				},
				wantBody: "[[request body]]",
			},
			eventType: "load",
			want: Response{
				Code: 200,
				Body: "successful body 16",
			},
			wantOk: true,
		},
	}
	for i, test := range tests {
		var jsFuncs []js.Func
		dom := mockDOM{
			NewXHRFunc: func() js.Value {
				return test.mockXMLHttpRequest.jsValue(test.eventType)
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
		test.mockXMLHttpRequest.Release()
		test.mockXMLHttpRequest.checkCalls(t)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error when eventType is %v, got %#v", i, test.eventType, *got)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}
