//go:build js && wasm

package ui

import (
	"reflect"
	"strings"
	"syscall/js"
	"testing"
)

func TestNewForm(t *testing.T) {
	t.Run("no querier", func(t *testing.T) {
		event := js.ValueOf(map[string]interface{}{
			"target": map[string]interface{}{
				"method": "POST",
				"action": "http://example.com",
			},
		})
		_, err := NewForm(nil, event)
		if err == nil {
			t.Error("wanted error when creating form with bad url")
		}
	})
	t.Run("bad action", func(t *testing.T) {
		querier := func(form js.Value, query string) (noValues []js.Value) {
			return noValues
		}
		event := js.ValueOf(map[string]interface{}{
			"target": map[string]interface{}{
				"method": "POST",
				"action": "bad_url",
			},
		})
		_, err := NewForm(querier, event)
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
		all := []js.Value{Param1, param2}
		querier := func(form js.Value, query string) []js.Value {
			if form.Get("id").String() != "form_value" {
				t.Errorf("wanted form element to be queried for inputs")
			}
			if !strings.HasPrefix(query, "input") {
				t.Errorf("wanted query to be string for inputs: got %v", query)
			}
			return all
		}
		formValue := js.ValueOf(map[string]interface{}{ // form
			"method": "POST",
			"action": "https://example.com/hello?wasm=true",
			"id":     "form_value",
		})
		event := js.ValueOf(map[string]interface{}{
			"target": formValue,
		})
		want := Form{
			Element: js.Undefined(),
			Method:  "POST",
			URL: URL{
				Scheme:    "https",
				Authority: "example.com",
				Path:      "/hello",
				RawQuery:  "wasm=true",
			},
			Params: Values{
				"A": "first param",
				"B": "2",
			},
		}
		got, err := NewForm(querier, event)
		switch {
		case err != nil:
			t.Errorf("unwanted error: %v", err)
		case got == nil, !formValue.Equal(got.Element):
			t.Error("form values not equal")
		case got.querier == nil:
			t.Error("querier not set")
		default:
			got.Element = js.Undefined()
			got.querier = nil
			if !reflect.DeepEqual(want, *got) {
				t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, *got)
			}
		}
	})
}

func TestResetForm(t *testing.T) {
	one := js.ValueOf(map[string]interface{}{"value": "first value"})
	two := js.ValueOf(map[string]interface{}{"value": 2})
	three := js.ValueOf(map[string]interface{}{"value": true})
	all := []js.Value{one, two, three}
	querier := func(form js.Value, query string) []js.Value {
		if want, got := "form_element", form.Get("id").String(); want != got {
			t.Errorf("wanted reset form to be called on form element (%v), got %v", want, got)
		}
		if !strings.HasPrefix(query, "input") {
			t.Errorf("wanted query to be string for inputs: got %v", query)
		}
		return all
	}
	element := js.ValueOf(map[string]interface{}{
		"id": "form_element",
	})
	f := Form{
		Element: element,
		querier: querier,
	}
	f.Reset()
	for i, v := range all {
		got := v.Get("value").String()
		if len(got) != 0 {
			t.Errorf("wanted value %v to have empty value, got %q", i, got)
		}
	}
}
