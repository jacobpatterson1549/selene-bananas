//go:build js && wasm

package ui

import (
	"errors"
	"fmt"
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

func TestQuerySelector(t *testing.T) {
	wantQuery := "some sort of a query"
	wantValue := js.ValueOf(1337)
	querySelector := MockQuerySelector(t, wantQuery, wantValue)
	gotValue := QuerySelector(wantQuery)
	if !wantValue.Equal(gotValue) {
		t.Errorf("wanted different value")
	}
	querySelector.Release()
}

func TestQuerySelectorAll(t *testing.T) {
	wantQuery := "some sort of a query ALL"
	one, two, three := js.ValueOf(1), js.ValueOf("two"), js.ValueOf(true)
	all := js.ValueOf([]interface{}{one, two, three})
	querySelectorAll := MockQuery(t, wantQuery, all)
	target := js.ValueOf(map[string]interface{}{
		"querySelectorAll": querySelectorAll,
	})
	wantValue := []js.Value{one, two, three}
	gotValue := QuerySelectorAll(target, wantQuery)
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Errorf("wanted different value")
	}
	querySelectorAll.Release()
}

func TestChecked(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		wantQuery := "some sort of a query checked"
		value := js.ValueOf(map[string]interface{}{
			"checked": want,
		})
		querySelector := MockQuerySelector(t, wantQuery, value)
		got := Checked(wantQuery)
		querySelector.Release()
		if want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestSetChecked(t *testing.T) {
	tests := []struct {
		checked   bool
		wantValue js.Value
	}{
		{
			checked: true,
			wantValue: js.ValueOf(map[string]interface{}{
				"checked": true,
			}),
		},
		{
			checked: false,
			wantValue: js.ValueOf(map[string]interface{}{
				"checked": false,
			}),
		},
	}
	for i, test := range tests {
		wantQuery := "some sort of a query setChecked"
		gotValue := js.ValueOf(map[string]interface{}{})
		querySelector := MockQuerySelector(t, wantQuery, gotValue)
		SetChecked(wantQuery, test.checked)
		querySelector.Release()
		if want, got := test.wantValue.Get("checked").Bool(), gotValue.Get("checked").Bool(); want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestValue(t *testing.T) {
	wantQuery := "some sort of a query value"
	want := "[[TOP SECRET VALUE]]"
	value := js.ValueOf(map[string]interface{}{
		"value": want,
	})
	querySelector := MockQuerySelector(t, wantQuery, value)
	got := Value(wantQuery)
	querySelector.Release()
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestSetValue(t *testing.T) {
	wantQuery := "some sort of a query SETvalue"
	value := "[[TOP SECRET VALUE]]"
	wantValue := js.ValueOf(map[string]interface{}{
		"value": value,
	})
	gotValue := js.ValueOf(map[string]interface{}{})
	querySelector := MockQuerySelector(t, wantQuery, gotValue)
	SetValue(wantQuery, value)
	querySelector.Release()
	if want, got := wantValue.Get("value").String(), gotValue.Get("value").String(); want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestSetButtonDisabled(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		wantQuery := "some sort of a query setChecked"
		value := js.ValueOf(map[string]interface{}{})
		querySelector := MockQuerySelector(t, wantQuery, value)
		SetButtonDisabled(wantQuery, want)
		querySelector.Release()
		got := value.Get("disabled").Bool()
		if want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestFormatTime(t *testing.T) {
	oldTZ := time.Local
	loc, _ := time.LoadLocation("America/Los_Angeles")
	time.Local = loc
	defer func() { time.Local = oldTZ }()
	// TODO: add helper func for above code, using t.Cleanup
	utcSeconds := int64(1632161703)
	want := "11:15:03"
	got := FormatTime(utcSeconds)
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestCloneElement(t *testing.T) {
	query := "some sort of a query cloneElement"
	want := "value from result of cloneNode"
	wantValue := js.ValueOf(want)
	cloneNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return wantValue
	})
	content := js.ValueOf(map[string]interface{}{
		"cloneNode": cloneNode,
	})
	value := js.ValueOf(map[string]interface{}{
		"content": content,
	})
	querySelector := MockQuerySelector(t, query, value)
	gotValue := CloneElement(query)
	querySelector.Release()
	cloneNode.Release()
	got := gotValue.String()
	if want != got {
		t.Errorf("not equal\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestConfirm(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		message := "some sort of a confirm message"
		confirmFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			gotMessage := args[0].String()
			if message != gotMessage {
				t.Errorf("Test %v: messages not equal: wanted %v, got %v", i, message, gotMessage)
			}
			return want
		})
		js.Global().Set("confirm", confirmFn)
		got := Confirm(message)
		confirmFn.Release()
		if want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestAlert(t *testing.T) {
	message := "some sort of a alert message"
	alerted := false
	alertFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		gotMessage := args[0].String()
		if message != gotMessage {
			t.Errorf("messages not equal: wanted %v, got %v", message, gotMessage)
		}
		alerted = true
		return nil
	})
	js.Global().Set("alert", alertFn)
	alert(message)
	alertFn.Release()
	if !alerted {
		t.Errorf("wanted alert")
	}
}

func TestColor(t *testing.T) {
	want := "cyan"
	computedStyle := js.ValueOf(map[string]interface{}{
		"color": want,
	})
	wantElement := js.ValueOf("wantColorElement")
	getComputedStyle := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		gotElement := args[0]
		if !wantElement.Equal(gotElement) {
			t.Errorf("elements not equal: wanted %q, got %q", wantElement.String(), gotElement.String())
		}
		return computedStyle
	})
	js.Global().Set("getComputedStyle", getComputedStyle)
	got := Color(wantElement)
	getComputedStyle.Release()
	if want != got {
		t.Errorf("colors not equal: wanted %v, got %v", want, got)
	}
}

func TestNewWebSocket(t *testing.T) {
	want := js.ValueOf(map[string]interface{}{
		"key": "the new websocket",
	})
	wantURL := "special_url"
	websocketFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		gotURL := args[0].String()
		if wantURL != gotURL {
			t.Errorf("urls not equal: wanted %v, got %v", wantURL, gotURL)
		}
		return want
	})
	js.Global().Set("WebSocket", websocketFn)
	got := NewWebSocket(wantURL)
	websocketFn.Release()
	if !want.Equal(got) {
		wantS, gotS := want.Get("key").String(), got.Get("key").String()
		t.Errorf("mock websocket objects not equal: wanted %q, got %q", wantS, gotS)
	}
}

func TestNewXHR(t *testing.T) {
	want := js.ValueOf(map[string]interface{}{
		"key": "the new xhr",
	})
	xhrFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return want
	})
	js.Global().Set("XMLHttpRequest", xhrFn)
	got := NewXHR()
	xhrFn.Release()
	if !want.Equal(got) {
		wantS, gotS := want.Get("key").String(), got.Get("key").String()
		t.Errorf("mock xhr objects not equal: wanted %q, got %q", wantS, gotS)
	}
}

func TestRecoverError(t *testing.T) {
	tests := []struct {
		r         interface{}
		want      error
		wantPanic bool
	}{
		{
			r:    errors.New("error 0"),
			want: errors.New("error 0"),
		},
		{
			r:    "error 1",
			want: errors.New("error 1"),
		},
		{
			r:         2,
			wantPanic: true,
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test %v", i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case r == nil && test.wantPanic:
					t.Errorf("wanted panic B")
				case r != nil && !test.wantPanic:
					t.Error("unwanted panic")
				}
			}()
			got := RecoverError(test.r)
			switch {
			case test.wantPanic:
				t.Error("wanted panic A")
			case test.want.Error() != got.Error():
				t.Errorf("errors not equal: wanted %v, got %v", test.want, got)
			}
		})
	}
}

func TestBase64Decode(t *testing.T) {
	a := `SGVsbG8gV29ybGQh`
	want := `Hello World!`
	atobFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wantA, gotA := a, args[0].String()
		if wantA != gotA {
			t.Errorf("input parameters to atob not equal: wanted %q, got %q", wantA, gotA)
		}
		return want
	})
	js.Global().Set("atob", atobFn)
	got := Base64Decode(a)
	atobFn.Release()
	if want != string(got) {
		t.Errorf("decoded strings not equal: wanted %q, got %q", want, got)
	}
}