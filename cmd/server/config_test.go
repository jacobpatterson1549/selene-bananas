package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
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

func TestFiles(t *testing.T) {
	t.Run("names", func(t *testing.T) {
		var fA, fB fstest.MapFile
		filesTests := []struct {
			fs.FS
			want []string
		}{
			{}, // nil FS, will casue panic in io/fs, but this expects an nil map
			{ // empty FS
				FS: fstest.MapFS{},
			},
			{ // files all in root
				FS: fstest.MapFS{
					"a": &fA,
					"b": &fB,
				},
				want: []string{
					"a",
					"b",
				},
			},
			{ // sub-directory
				FS: fstest.MapFS{
					"d1/a": &fA,
				},
				want: []string{
					"d1/a",
				},
			},
			{ // duplicate names, buf in different directories
				FS: fstest.MapFS{
					"a":    &fA,
					"d1/a": &fB,
				},
				want: []string{
					"a",
					"d1/a",
				},
			},
			{ // nested in directory
				FS: fstest.MapFS{
					"d1/d2/a": &fA,
				},
				want: []string{
					"d1/d2/a",
				},
			},
		}
		for i, test := range filesTests {
			got, err := Files(test.FS)
			switch {
			case err != nil:
				t.Errorf("Test %v: unwanted error: %v", i, err)
			default:
				// ensure the keys are equal
				switch {
				case len(test.want) != len(got):
					t.Errorf("Test %v: wanted %v keys, got %v", i, len(test.want), len(got))
				default:
					for _, n := range test.want {
						if _, ok := got[n]; !ok {
							t.Errorf("Test %v: wanted file name %v", i, n)
						}
					}
				}
				// case !reflect.DeepEqual(test.want, got):
				// 	t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	})
}
