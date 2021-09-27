//go:build js && wasm

package ui

import (
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

func TestQuerySelector(t *testing.T) {
	var dom DOM
	wantQuery := "some sort of a query"
	wantValue := js.ValueOf(1337)
	querySelector := MockQuerySelector(t, wantQuery, wantValue)
	gotValue := dom.QuerySelector(wantQuery)
	if !wantValue.Equal(gotValue) {
		t.Errorf("wanted different value")
	}
	querySelector.Release()
}

func TestQuerySelectorAll(t *testing.T) {
	var dom DOM
	wantQuery := "some sort of a query ALL"
	one, two, three := js.ValueOf(1), js.ValueOf("two"), js.ValueOf(true)
	all := js.ValueOf([]interface{}{one, two, three})
	querySelectorAll := MockQuery(t, wantQuery, all)
	target := js.ValueOf(map[string]interface{}{
		"querySelectorAll": querySelectorAll,
	})
	wantValue := []js.Value{one, two, three}
	gotValue := dom.QuerySelectorAll(target, wantQuery)
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Errorf("wanted different value")
	}
	querySelectorAll.Release()
}

func TestChecked(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		var dom DOM
		wantQuery := "some sort of a query checked"
		value := js.ValueOf(map[string]interface{}{
			"checked": want,
		})
		querySelector := MockQuerySelector(t, wantQuery, value)
		got := dom.Checked(wantQuery)
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
		var dom DOM
		wantQuery := "some sort of a query setChecked"
		gotValue := js.ValueOf(map[string]interface{}{})
		querySelector := MockQuerySelector(t, wantQuery, gotValue)
		dom.SetChecked(wantQuery, test.checked)
		querySelector.Release()
		if want, got := test.wantValue.Get("checked").Bool(), gotValue.Get("checked").Bool(); want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestValue(t *testing.T) {
	var dom DOM
	wantQuery := "some sort of a query value"
	want := "[[TOP SECRET VALUE]]"
	value := js.ValueOf(map[string]interface{}{
		"value": want,
	})
	querySelector := MockQuerySelector(t, wantQuery, value)
	got := dom.Value(wantQuery)
	querySelector.Release()
	if want != got {
		t.Errorf("values not equal: wanted %v, got %v", want, got)
	}
}

func TestSetValue(t *testing.T) {
	var dom DOM
	wantQuery := "some sort of a query SETvalue"
	value := "[[TOP SECRET VALUE]]"
	wantValue := js.ValueOf(map[string]interface{}{
		"value": value,
	})
	gotValue := js.ValueOf(map[string]interface{}{})
	querySelector := MockQuerySelector(t, wantQuery, gotValue)
	dom.SetValue(wantQuery, value)
	querySelector.Release()
	if want, got := wantValue.Get("value").String(), gotValue.Get("value").String(); want != got {
		t.Errorf("set value not expected: wanted %v, got %v", want, got)
	}
}

func TestSetButtonDisabled(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		var dom DOM
		wantQuery := "some sort of a query setChecked"
		value := js.ValueOf(map[string]interface{}{})
		querySelector := MockQuerySelector(t, wantQuery, value)
		dom.SetButtonDisabled(wantQuery, want)
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
	var dom DOM
	utcSeconds := int64(1632161703)
	want := "11:15:03"
	got := dom.FormatTime(utcSeconds)
	if want != got {
		t.Errorf("formatted time not expected: wanted %v, got %v", want, got)
	}
}

func TestCloneElement(t *testing.T) {
	var dom DOM
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
	gotValue := dom.CloneElement(query)
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
		var dom DOM
		message := "some sort of a confirm message"
		confirmFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			gotMessage := args[0].String()
			if message != gotMessage {
				t.Errorf("Test %v: messages not equal: wanted %v, got %v", i, message, gotMessage)
			}
			return want
		})
		js.Global().Set("confirm", confirmFn)
		got := dom.Confirm(message)
		confirmFn.Release()
		if want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestAlert(t *testing.T) {
	var dom DOM
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
	dom.alert(message)
	alertFn.Release()
	if !alerted {
		t.Errorf("wanted alert")
	}
}

func TestColor(t *testing.T) {
	var dom DOM
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
	got := dom.Color(wantElement)
	getComputedStyle.Release()
	if want != got {
		t.Errorf("colors not equal: wanted %v, got %v", want, got)
	}
}

func TestNewWebSocket(t *testing.T) {
	var dom DOM
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
	got := dom.NewWebSocket(wantURL)
	websocketFn.Release()
	if !want.Equal(got) {
		wantS, gotS := want.Get("key").String(), got.Get("key").String()
		t.Errorf("mock websocket objects not equal: wanted %q, got %q", wantS, gotS)
	}
}

func TestNewXHR(t *testing.T) {
	var dom DOM
	want := js.ValueOf(map[string]interface{}{
		"key": "the new xhr",
	})
	xhrFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return want
	})
	js.Global().Set("XMLHttpRequest", xhrFn)
	got := dom.NewXHR()
	xhrFn.Release()
	if !want.Equal(got) {
		wantS, gotS := want.Get("key").String(), got.Get("key").String()
		t.Errorf("mock xhr objects not equal: wanted %q, got %q", wantS, gotS)
	}
}

func TestBase64Decode(t *testing.T) {
	var dom DOM
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
	got := dom.Base64Decode(a)
	atobFn.Release()
	if want != string(got) {
		t.Errorf("decoded strings not equal: wanted %q, got %q", want, got)
	}
}

func TestStoreFormCredentials(t *testing.T) {
	t.Run("no PasswordCredential", func(t *testing.T) {
		dom := new(DOM)
		var form js.Value
		dom.StoreCredentials(form)
		// [ should not cause error ]
	})
	t.Run("with PasswordCredential", func(t *testing.T) {
		dom := new(DOM)
		form := js.ValueOf(map[string]interface{}{"a": 1})
		cred := js.ValueOf(map[string]interface{}{"b": 2})
		credentialsStored := false
		store := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			want := cred
			got := args[0]
			if !want.Equal(got) {
				t.Errorf("wanted %v to be stored as credentials, got %v", cred, got)
			}
			credentialsStored = true
			return nil
		})
		passwordCredential := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			want := form
			got := args[0]
			if !want.Equal(got) {
				t.Errorf("wanted new password credential to be created with %v, got %v", want, got)
			}
			return cred
		})
		navigator := js.ValueOf(map[string]interface{}{
			"credentials": js.ValueOf(map[string]interface{}{
				"store": store,
			}),
		})
		js.Global().Set("PasswordCredential", passwordCredential)
		js.Global().Set("navigator", navigator)
		dom.StoreCredentials(form)
		passwordCredential.Release()
		store.Release()
		if !credentialsStored {
			t.Error("wanted credentials to be stored")
		}
	})
}

// TestEncodeURIComponent ensures encodeURIComponent is called.
// tests copied from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent
func TestEncodeURIComponent(t *testing.T) {
	encodeEscapeTests := map[string]string{
		";,/?:@&=+$":  "%3B%2C%2F%3F%3A%40%26%3D%2B%24",
		"-_.!~*'()":   "-_.!~*'()",
		"#":           "%23",
		"ABC abc 123": "ABC%20abc%20123",
	}
	for text, want := range encodeEscapeTests {
		var dom DOM
		got := dom.EncodeURIComponent(text)
		if want != got {
			t.Errorf("did not encode properly\nwanted: %v\ngot:    %v", want, got)
		}
	}
}
