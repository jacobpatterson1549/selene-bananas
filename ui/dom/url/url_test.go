package url

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	parseTests := []struct {
		text   string
		wantOk bool
		want   URL
	}{
		{},
		{
			text: "noscheme.com",
		},
		{
			text: "http:/no_authority",
		},
		{
			text: "wss://selene-bananas.herokuapp.com/lobby?access_token=INVALID#PLEASE",
		},
		{
			text:   "https://example.com",
			wantOk: true,
			want: URL{
				Scheme:    "https",
				Authority: "example.com",
			},
		},
		{
			text:   "https://example.com/",
			wantOk: true,
			want: URL{
				Scheme:    "https",
				Authority: "example.com",
				Path:      "/",
			},
		},
		{
			text:   "http://127.0.0.1:8000/user_join_lobby",
			wantOk: true,
			want: URL{
				Scheme:    "http",
				Authority: "127.0.0.1:8000",
				Path:      "/user_join_lobby",
			},
		},
		{
			text:   "wss://selene-bananas.herokuapp.com/lobby?access_token=INVALID",
			wantOk: true,
			want: URL{
				Scheme:    "wss",
				Authority: "selene-bananas.herokuapp.com",
				Path:      "/lobby",
				RawQuery:  "access_token=INVALID",
			},
		},
		{
			text:   "http://example.com?hello=world",
			wantOk: true,
			want: URL{
				Scheme:    "http",
				Authority: "example.com",
				RawQuery:  "hello=world",
			},
		},
	}
	for i, test := range parseTests {
		got, err := Parse(test.text)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(test.want, *got):
			t.Errorf("Test %v:\nwanted: %#v\ngot:    %#v", i, test.want, *got)
		case test.text != got.String(): // TestString
			t.Errorf("Test %v: String not equal:\nwanted: %v\ngot:    %v", i, test.text, got.String())
		}
	}
}

func TestGet(t *testing.T) {
	v := make(Values)
	if got := v.Get("a"); "" != got {
		t.Errorf("wanted empty string when value not present, got %v", got)
	}
	v.Add("a", "34")
	if got := v.Get("a"); "34" != got {
		t.Errorf("wanted 34, got %v", got)
	}
}

func TestEncode(t *testing.T) {
	v := make(Values)
	v.Add("a", "34")
	v.Add("b", "cat")
	v.Add("a", "340")
	v.Add("b", "cat")
	got := v.Encode()
	switch got {
	case "a=340&b=cat", "b=cat&a=340": // nondeterminstic ordering
	default:
		t.Errorf("did not encode properly, got %v", got)
	}
}
