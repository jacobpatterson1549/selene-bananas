package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestUnembedFS(t *testing.T) {
	unembedFSTests := []struct {
		fs.FS
		subdirectory  string
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
		{ // empty embedded file system, no subdirectory
			FS: fstest.MapFS{
				"embed": &fstest.MapFile{},
			},
		},
		{ // empty embedded file system
			FS: fstest.MapFS{
				"embed/d1": &fstest.MapFile{},
			},
			subdirectory: "d1",
		},
		{
			FS: fstest.MapFS{
				"f0":                &fstest.MapFile{},
				"embed/f1":          &fstest.MapFile{},
				"embed/d1/f2":       &fstest.MapFile{},
				"embed/d1/d2/d3/f3": &fstest.MapFile{},
				"embed/d1/d4/f4":    &fstest.MapFile{},
				"embed/d1/d4/f5":    &fstest.MapFile{},
				"embed/d2/f6":       &fstest.MapFile{},
			},
			subdirectory: "d1",
			wantFileNames: []string{
				"f2",
				"d2/d3/f3",
				"d4/f4",
				"d4/f5",
			},
		},
	}
	for i, test := range unembedFSTests {
		got, err := unembedFS(test.FS, test.subdirectory)
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
