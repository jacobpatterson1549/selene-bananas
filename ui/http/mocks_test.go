package http

import (
	"reflect"
	"syscall/js"
	"testing"
)

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
		event := js.ValueOf(map[string]interface{}{
			"type": eventType,
		})
		handler.Invoke(event)
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

type mockDOM struct {
	NewXHRFunc         func() js.Value
	NewJsEventFuncFunc func(fn func(event js.Value)) js.Func
}

func (m *mockDOM) NewXHR() js.Value {
	return m.NewXHRFunc()
}

func (m *mockDOM) NewJsEventFunc(fn func(event js.Value)) js.Func {
	return m.NewJsEventFuncFunc(fn)
}
