package tile

import "testing"

func TestNew(t *testing.T) {
	newTests := []struct {
		id     ID
		r      rune
		wantOk bool
		want   Tile
	}{
		{
			id: 3,
			r:  'a',
		},
		{
			id: 3,
			r:  'A',
			want: Tile{
				ID: 3,
				Ch: 'A',
			},
			wantOk: true,
		},
	}
	for i, test := range newTests {
		got, err := New(test.id, test.r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}
