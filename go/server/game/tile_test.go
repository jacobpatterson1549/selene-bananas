package game

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLetterString(t *testing.T) {
	l := letter('X')
	want := "X"
	got := fmt.Sprintf("%v", l)
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
		t.Errorf("unexpected error: %v", err)
	case want != string(got):
		t.Errorf("wanted %v, got %v", want, string(got))
	}
}

func TestUnmarshalLetter(t *testing.T) {
	unmarshalLetterTests := []struct {
		json    string
		want    letter
		wantErr bool
	}{
		// {`"A"`, letter('A'), false},
		// {`"Z"`, letter('Z'), false},
		// {`"X"`, letter('X'), false},
		// {`"XYZ"`, letter('X'), true},
		{`X`, 0, true},
		// {`"@"`, 0, true},
	}
	for i, test := range unmarshalLetterTests {
		var got letter
		err := json.Unmarshal([]byte(test.json), &got)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}
