// +build js,wasm

package json

import (
	"reflect"
	"syscall/js"
	"testing"
)

func TestStringify(t *testing.T) {
	stringifyTests := []struct {
		value interface{}
		want  string
	}{
		{
			value: "A",
			want:  `"A"`,
		},
		{
			value: 3,
			want:  `3`,
		},
		{
			value: []string{"green", "day"},
			want:  `["green","day"]`,
		},
		{
			value: struct {
				Letter string `json:"ch"`
			}{
				Letter: "B",
			},
			want: `{"ch":"B"}`,
		},
		{
			value: true,
			want:  "true",
		},
		{
			value: false,
			want:  "false",
		},
	}
	for i, test := range stringifyTests {
		got, err := Stringify(test.value)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != got:
			t.Errorf("Test %v: wanted: %v, got %v", i, test.want, got)
		}
	}
}

func TestToInterface(t *testing.T) {
	interfaces := []interface{}{
		nil,
		"C",
		9,
		true,
		false,
		[]interface{}{"nine", "blind", "mice"},
		map[string]interface{}{
			"I": 1,
			"S": "2",
			"A": []interface{}{44, 57},
			"O": map[string]interface{}{
				"fn": "harry",
				"ln": "potter",
			},
		},
	}
	for i, want := range interfaces {
		jsValue := js.ValueOf(want)
		got, err := toInterface(jsValue)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(want, got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, want, got)
		}
	}
}
