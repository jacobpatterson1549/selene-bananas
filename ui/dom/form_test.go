//go:build js && wasm

package dom

import (
	"reflect"
	"strings"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom/url"
)

func TestNewForm(t *testing.T) {
	t.Run("bad action", func(t *testing.T) {
		event := js.ValueOf(map[string]interface{}{
			"target": map[string]interface{}{
				"method": "POST",
				"action": "bad_url",
			},
		})
		_, err := NewForm(event)
		if err == nil {
			t.Error("wanted error when creating form with bad url")
		}
	})
	t.Run("happy path", func(t *testing.T) {
		Param1 := js.ValueOf(map[string]interface{}{
			"name":  "A",
			"value": "first param",
		})
		param2 := js.ValueOf(map[string]interface{}{
			"name":  "B",
			"value": "2",
		})
		all := js.ValueOf([]interface{}{Param1, param2})
		querySelectorAll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			query := args[0] // query
			if this.Get("id").String() != "event" {
				t.Errorf("wanted event to be queried for inputs")
			}
			if query.Type() != js.TypeString || !strings.HasPrefix(query.String(), "input") {
				t.Errorf("wanted query to be string for inputs: got %v, (%v)", query.String(), query.Type())
			}
			return all
		})
		formValue := js.ValueOf(map[string]interface{}{ // form
			"method":           "POST",
			"action":           "https://example.com/hello?wasm=true",
			"id":               "event",
			"querySelectorAll": querySelectorAll,
		})
		event := js.ValueOf(map[string]interface{}{
			"target": formValue,
		})
		want := Form{
			v:      formValue,
			Method: "POST",
			URL: url.URL{
				Scheme:    "https",
				Authority: "example.com",
				Path:      "/hello",
				RawQuery:  "wasm=true",
			},
			Params: url.Values{
				"A": "first param",
				"B": "2",
			},
		}
		got, err := NewForm(event)
		querySelectorAll.Release()
		switch {
		case err != nil:
			t.Errorf("unwanted error: %v", err)
		case !reflect.DeepEqual(want, *got):
			t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, *got)
		}
	})
}

func TestResetForm(t *testing.T) {
	one := js.ValueOf(map[string]interface{}{"value": "first value"})
	two := js.ValueOf(map[string]interface{}{"value": 2})
	three := js.ValueOf(map[string]interface{}{"value": true})
	all := js.ValueOf([]interface{}{one, two, three})
	querySelectorAll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		query := args[0] // query
		if this.Get("id").String() != "form_element" {
			t.Errorf("wanted reset form to be called on form element")
		}
		if query.Type() != js.TypeString || len(query.String()) == 0 {
			t.Errorf("wanted query to be string with length: got %v, (%v)", query.String(), query.Type())
		}
		return all
	})
	element := js.ValueOf(map[string]interface{}{
		"id":               "form_element",
		"querySelectorAll": querySelectorAll,
	})
	f := Form{v: element}
	f.Reset()
	querySelectorAll.Release()
	for i := 0; i < 3; i++ {
		v := all.Index(i)
		got := v.Get("value").String()
		if len(got) != 0 {
			t.Errorf("wanted value %v to have empty value, got %q", i, got)
		}
	}
}

func TestStoreFormCredentials(t *testing.T) {
	t.Run("no PasswordCredential", func(t *testing.T) {
		var f Form
		f.StoreCredentials()
	})
	t.Run("with PasswordCredential", func(t *testing.T) {
		element := js.ValueOf(map[string]interface{}{"a": 1})
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
			want := element
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
		f := Form{v: element}
		f.StoreCredentials()
		passwordCredential.Release()
		store.Release()
		if !credentialsStored {
			t.Error("wanted credentials to be stored")
		}
	})
}
