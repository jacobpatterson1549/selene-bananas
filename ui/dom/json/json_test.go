// +build js,wasm

package json

import "testing"

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
