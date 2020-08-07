package tile

import (
	"encoding/json"
	"testing"
)

func TestNewLetter(t *testing.T) {
	newLetterTests := []struct {
		ch     rune
		want   letter
		wantOk bool
	}{
		{},
		{
			ch: 'a',
		},
		{
			ch: '_',
		},
		{
			ch:     'A',
			want:   'A',
			wantOk: true,
		},
		{
			ch:     'Z',
			want:   'Z',
			wantOk: true,
		},
		{
			ch:     'L',
			want:   'L',
			wantOk: true,
		},
	}
	for i, test := range newLetterTests {
		got, err := newLetter(test.ch)
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

func TestLetterString(t *testing.T) {
	l := letter('X')
	want := "X"
	got := l.String()
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMarshalLetter(t *testing.T) {
	l := letter('X')
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
		want   letter
		wantOk bool
	}{
		{
			json: `"XYZ"`,
		},
		{
			json: `X`,
		},
		{
			json: `"@"`,
		},
		{
			json:   `"A"`,
			want:   letter('A'),
			wantOk: true,
		},
		{
			json:   `"Z"`,
			want:   letter('Z'),
			wantOk: true,
		},
		{
			json:   `"X"`,
			want:   letter('X'),
			wantOk: true,
		},
	}
	for i, test := range unmarshalLetterTests {
		var got letter
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

func TestUnmarshalLetterDirectError(t *testing.T) {
	var l letter
	err := l.UnmarshalJSON([]byte(`X`))
	if err == nil {
		t.Errorf("wanted error when unmarshalling unquoted letter")
	}
}
