package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestUnembedFS(t *testing.T) {
	unembedFSTests := []struct {
		fs.FS
		wantFileNames []string
	}{
		{},
		{
			FS: fstest.MapFS{},
		},
		{ // no embedded file system
			FS: fstest.MapFS{
				"f1":        &fstest.MapFile{},
				"subdir/f2": &fstest.MapFile{},
			},
		},
		{ // empty embedded file system
			FS: fstest.MapFS{
				"embed": &fstest.MapFile{},
			},
		},
		{
			FS: fstest.MapFS{
				"f0":                &fstest.MapFile{}, // should be ignored
				"embed/f1":          &fstest.MapFile{},
				"embed/d1/d2/d3/f2": &fstest.MapFile{},
			},
			wantFileNames: []string{
				"f1",
				"d1/d2/d3/f2",
			},
		},
	}
	for i, test := range unembedFSTests {
		got, err := unembedFS(test.FS)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case got == nil:
			t.Errorf("Test %v: wanted unembedded file system to not be nil", i)
		default:
			for _, n := range test.wantFileNames {
				if _, err := got.Open(n); err != nil {
					t.Errorf("Test %v: wanted file named '%v' in unembedded file stystem", i, n)
				}
			}
		}
	}
}
