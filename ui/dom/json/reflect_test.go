package json

import (
	"reflect"
	"testing"
)

func TestFromObjectString(t *testing.T) {
	var got string
	want := "abc"
	err := fromObject(&got, want)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectStringCustom(t *testing.T) {
	type myString string
	var got myString
	want := myString("abc")
	err := fromObject(&got, want)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectInt(t *testing.T) {
	var got int
	want := 17
	err := fromObject(&got, want)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectInt64(t *testing.T) {
	var got int64
	want := int64(1597076190)
	err := fromObject(&got, want)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectSlice(t *testing.T) {
	var got []int
	source := []interface{}{6, 4, 3, 5}
	want := []int{6, 4, 3, 5}
	err := fromObject(&got, source)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectSlice2D(t *testing.T) {
	var got [][]int
	source := []interface{}{
		[]interface{}{1, 2},
		[]interface{}{3, 4},
	}
	want := [][]int{
		{1, 2},
		{3, 4},
	}
	err := fromObject(&got, source)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestFromObjectStruct(t *testing.T) {
	type flag struct {
		Color string `json:"color"`
		Time  int64  `json:"time"`
	}
	type myStruct struct {
		Name     string `json:"name"`
		Password string
		Day      int      `json:"d,omitempty"`
		Month    int      `json:"m,omitEmpty"`
		Year     int      `json:"y"`
		Pets     []string `json:"pets"`
		Flag     flag     `json:"flag"`
		Flags    []flag   `json:"flags"`
	}
	source := map[string]interface{}{
		"name": "selene",
		"d":    28,
		"pets": []string{"fred"},
		"flag": map[string]interface{}{
			"color": "blue",
		},
		"flags": []interface{}{
			map[string]interface{}{
				"color": "red",
				"time":  1597076190,
			},
		},
	}
	var got myStruct
	want := myStruct{
		Name: "selene",
		Day:  28,
		Pets: []string{"fred"},
		Flag: flag{
			Color: "blue",
		},
		Flags: []flag{
			{
				Color: "red",
				Time:  1597076190,
			},
		},
	}
	err := fromObject(&got, source)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %#v\ngot:    %#v", want, got)
	}
}

func TestToObject(t *testing.T) {
	type myInt int
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
			src:  myInt(7),
			want: int64(7),
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
				EmptyStruct struct{} `json:"EmptyStruct,omitempty"`
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
