package tile

import (
	"testing"
)

func TestNew(t *testing.T) {
	newTests := []struct {
		id      ID
		r       rune
		wantErr bool
		want    Tile
	}{
		{
			id:      3,
			r:       'a',
			wantErr: true,
		},
		{
			id:   3,
			r:    'A',
			want: Tile{ID: 3, Ch: 'A'},
		},
	}
	for i, test := range newTests {
		got, err := New(test.id, test.r)
		switch {
		case err != nil:
			if got != nil || !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case got == nil || test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}
