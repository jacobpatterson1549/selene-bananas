package json

import (
	"reflect"
	"testing"
)

func TestFromMap(t *testing.T) {
	fromMapTests := []struct {
		want interface{}
		got  interface{}
	}{
		// TODO
	}
	for i, test := range fromMapTests {
		err := fromObject(&test.got, test.want)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, test.got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, test.got)
		}
	}
}

func TestToObject(t *testing.T) {
	parseUserInfoJSONTests := []struct {
		src     interface{}
		wantErr bool
		want    interface{}
	}{
		{},
		{
			src:  int(7),
			want: int64(7),
		},
		{
			src:  int64(1597076190),
			want: int64(1597076190),
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
				Name        string   `json:"id"`
				DoNotEncode string   `json:"-"`
				EmptyInt    int      `json:"EmptyInt,omitempty"`
				EmptySlice  []string `json:"EmptySlice,omitempty"`
				EmptyStruct struct {
					E int `json:"e,omitempty"`
				} `json:"EmptyStruct,omitempty"`
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
		got, err := toObject(test.src)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestToObjectCustomType(t *testing.T) {
	type myInt int
	var x myInt = 7
	var want interface{} = int64(7)
	got, err := toObject(x)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("wanted: %v, got: %v", want, got)
	}
}
