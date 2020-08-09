package tile

import (
	"encoding/json"
	"testing"
)

func TestMarshalLetter(t *testing.T) {
	l := Letter('X')
	want := `"X"`
	got, err := json.Marshal(l)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case want != string(got):
		t.Errorf("wanted %v, got %v", want, string(got))
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
			want:   "A",
			wantOk: true,
		},
		{
			json:   `"Z"`,
			want:   "Z",
			wantOk: true,
		},
		{
			json:   `"X"`,
			want:   "X",
			wantOk: true,
		},
	}
	for i, test := range unmarshalLetterTests {
		var got Letter
		err := json.Unmarshal([]byte(test.json), &got)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case test.want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}
