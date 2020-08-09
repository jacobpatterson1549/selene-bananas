package json

import (
	"reflect"
	"testing"
)

func TestToMap(t *testing.T) {
	parseUserInfoJSONTests := []struct {
		src     interface{}
		wantErr bool
		want    interface{}
	}{
		{},
		{
			src:  7,
			want: 7,
		},
		{
			src:  9223372036854775807,
			want: 9223372036854775807,
		},
		{
			src:  "some text",
			want: "some text",
		},
		{
			src:  []string{"a", "b", "c"},
			want: []interface{}{"a", "b", "c"},
		},
		{
			src: struct {
				Name string
			}{
				Name: "selene",
			},
			wantErr: true, // no json tag for struct field
		},
		{
			src: struct {
				Name        string `json:"id"`
				DoNotEncode string `json:"-"`
			}{
				Name:        "selene",
				DoNotEncode: "secret",
			},
			want: map[string]interface{}{
				"id": "selene",
			},
		},
	}
	for i, test := range parseUserInfoJSONTests {
		got, err := toMap(test.src)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, got)
		}
	}
}
