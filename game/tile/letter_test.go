package tile

import (
	"testing"
)

func TestNewLetter(t *testing.T) {
	newLetterTests := []struct {
		ch     rune
		want   Letter
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
		case test.want != *got:
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}
