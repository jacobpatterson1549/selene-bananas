package main

import (
	"testing"
)

func TestCleanVersion(t *testing.T) {
	cleanVersionTests := []struct {
		v      string
		wantOk bool
		want   string
	}{
		{},
		{
			v:      "9d2ffad8e5e5383569d37ec381147f2d\n",
			wantOk: true,
			want:   "9d2ffad8e5e5383569d37ec381147f2d",
		},
		{
			v: "adhoc version",
		},
	}
	for i, test := range cleanVersionTests {
		got, err := cleanVersion(test.v)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error when version is '%v'", i, test.v)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error when version is '%v': %v", i, test.v, err)
		case test.want != got:
			t.Errorf("Test %v: when version is '%v':\nwanted: '%v'\ngot:    '%v", i, test.v, test.want, got)
		}
	}
}
