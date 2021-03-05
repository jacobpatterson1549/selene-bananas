package tile

import (
	"encoding/json"
	"testing"
)

func TestMarshalLetter(t *testing.T) {
	marshalLetterTests := []struct {
		Letter
		want string
	}{
		{
			want: `"\u0000"`,
		},
		{
			Letter: 'X',
			want:   `"X"`,
		},
	}
	for i, test := range marshalLetterTests {
		got, err := json.Marshal(test.Letter)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != string(got):
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, string(got))
		}
	}
}

func TestUnmarshalLetter(t *testing.T) {
	unmarshalLetterTests := []struct {
		json   string
		want   Letter
		wantOk bool
	}{
		{
			json: `"XYZ"`,
		},
		{
			json: `X`,
		},
		{
			json: `1`,
		},
		{
			json: `"@"`,
		},
		{
			json:   `"A"`,
			want:   'A',
			wantOk: true,
		},
		{
			json:   `"Z"`,
			want:   'Z',
			wantOk: true,
		},
		{
			json:   `"X"`,
			want:   'X',
			wantOk: true,
		},
	}
	for i, test := range unmarshalLetterTests {
		var got Letter
		err := json.Unmarshal([]byte(test.json), &got)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}
