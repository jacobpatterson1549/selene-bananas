//go:build js && wasm

package http

import (
	"reflect"
	"syscall/js"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

func TestDoRequest(t *testing.T) {
	t.Run("nil DOM", func(t *testing.T) {
		var dom *ui.DOM
		var r Request
		var c Client
		_, err := c.Do(dom, r)
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
		dom := new(ui.DOM) // TODO: use mock
		xhr := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return test.mockXMLHttpRequest.jsValue(test.eventType)
		})
		js.Global().Set("XMLHttpRequest", xhr)
		got, err := test.Client.Do(dom, test.Request)
		xhr.Release()
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

type mockXMLHttpRequest struct {
	status             int
	response           string
	wantOpenMethod     string
	wantOpenURL        string
	wantTimeout        int
	wantRequestHeaders map[string]string
	wantBody           string
	gotOpenMethod      string
	gotOpenURL         string
	gotTimeout         int
	gotRequestHeaders  map[string]string
	gotBody            string
	valueFuncs         []js.Func
}

func (xhr *mockXMLHttpRequest) jsValue(eventType string) js.Value {
	eventListeners := make(map[string]js.Value, 4)
	open := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		xhr.gotOpenMethod = args[0].String()
		xhr.gotOpenURL = args[1].String()
		return nil
	})
	setRequestHeader := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		k, v := args[0].String(), args[1].String()
		xhr.gotRequestHeaders[k] = v
		return nil
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventType, handler := args[0].String(), args[1]
		eventListeners[eventType] = handler
		return nil
	})
	send := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		xhr.gotBody = args[0].String()
		handler := eventListeners[eventType]
		preventDefault := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return nil
		})
		event := js.ValueOf(map[string]interface{}{
			"preventDefault": preventDefault,
			"type":           eventType,
		})
		handler.Invoke(event)
		preventDefault.Release()
		return nil
	})
	xhr.valueFuncs = []js.Func{open, setRequestHeader, addEventListener, send}
	return js.ValueOf(map[string]interface{}{
		"open":             open,
		"setRequestHeader": setRequestHeader,
		"addEventListener": addEventListener,
		"send":             send,
		"status":           xhr.status,
		"response":         xhr.response,
	})
}

func (xhr *mockXMLHttpRequest) Release() {
	for _, f := range xhr.valueFuncs {
		f.Release()
	}
}

func (xhr mockXMLHttpRequest) checkCalls(t *testing.T) {
	t.Helper()
	if want, got := xhr.wantOpenMethod, xhr.gotOpenMethod; want != got {
		t.Errorf("xhr open methods not equal:\nwanted: %v\ngot:    %v", want, got)
	}
	if want, got := xhr.wantTimeout, xhr.gotTimeout; want != got {
		t.Errorf("xhr timeouts equal:\nwanted: %v\ngot:    %v", want, got)
	}
	if want, got := xhr.wantOpenURL, xhr.gotOpenURL; want != got {
		t.Errorf("xhr open urls not equal:\nwanted: %v\ngot:    %v", want, got)
	}
	if want, got := xhr.wantRequestHeaders, xhr.gotRequestHeaders; !reflect.DeepEqual(want, got) {
		t.Errorf("xhr open args not equal:\nwanted: %v\ngot:    %v", want, got)
	}
	if want, got := xhr.wantBody, xhr.gotBody; want != got {
		t.Errorf("xhr bodies not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}
