package main

import (
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestUnembedFS(t *testing.T) {
	unembedFSTests := []struct {
		fs.FS
		subdirectory  string
		wantFileNames []string
		wantOk        bool
	}{
		{
			FS: fstest.MapFS{},
		},
		{ // no embedded file system
			FS: fstest.MapFS{
				"f1":        &fstest.MapFile{},
				"subdir/f2": &fstest.MapFile{},
			},
		},
		{ // empty embedded file system, missing subdirectory
			FS: fstest.MapFS{
				"embed/d1": &fstest.MapFile{},
			},
			subdirectory: "d2",
		},
		{ // empty embedded file system, no subdirectory
			FS: fstest.MapFS{
				"embed":    &fstest.MapFile{},
				"embed/f1": &fstest.MapFile{},
			},
			wantOk: true,
			wantFileNames: []string{
				"f1",
			},
		},
		{ // empty embedded file system
			FS: fstest.MapFS{
				"embed/d1": &fstest.MapFile{},
			},
			subdirectory: "d1",
			wantOk:       true,
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
			wantOk: true,
		},
	}
	for i, test := range unembedFSTests {
		got, err := unembedFS(test.FS, test.subdirectory)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
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

func TestNewEmbedParameters(t *testing.T) {
	var emptyFS fstest.MapFS
	staticFS := fstest.MapFS{
		"embed/static": &fstest.MapFile{},
	}
	templateFS := fstest.MapFS{
		"embed/template": &fstest.MapFile{},
	}
	sqlFS := fstest.MapFS{
		"embed/sql": &fstest.MapFile{},
	}
	okVersion := "ok"
	words := "apple\nbanana\ncarrot"
	wantVersion := okVersion
	wantWords := words
	newEmbedParametersTests := []struct {
		embeddedData
		words  string
		wantOk bool
	}{
		{}, // bad version
		{ // missing words
			embeddedData: embeddedData{
				Version: okVersion,
			},
		},
		{ // missing static fs
			embeddedData: embeddedData{
				Version: okVersion,
			},
			words: words,
		},
		{ // missing template fs
			embeddedData: embeddedData{
				Version:  okVersion,
				StaticFS: emptyFS,
			},
			words: words,
		},
		{ // missing SQL fs
			embeddedData: embeddedData{
				Version:    okVersion,
				StaticFS:   emptyFS,
				TemplateFS: emptyFS,
			},
			words: words,
		},
		{ // bad static fs
			embeddedData: embeddedData{
				Version:    okVersion,
				StaticFS:   emptyFS,
				TemplateFS: emptyFS,
				SQLFS:      emptyFS,
			},
			words: words,
		},
		{ // bad template fs
			embeddedData: embeddedData{
				Version:    okVersion,
				StaticFS:   staticFS,
				TemplateFS: emptyFS,
				SQLFS:      emptyFS,
			},
			words: words,
		},
		{ // bad SQL fs
			embeddedData: embeddedData{
				Version:    okVersion,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
				SQLFS:      emptyFS,
			},
			words: words,
		},
		{ // happy path
			embeddedData: embeddedData{
				Version:    okVersion,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
				SQLFS:      sqlFS,
			},
			words:  words,
			wantOk: true,
		},
	}
	for i, test := range newEmbedParametersTests {
		got, err := newEmbedParameters(test.Version, test.words, test.StaticFS, test.TemplateFS, test.SQLFS)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case wantVersion != got.Version:
			t.Errorf("Test %v: not equal:\nwanted: %v\ngot:    %v", i, wantVersion, got.Version)
		default:
			gotWords, err := io.ReadAll(got.WordsReader)
			switch {
			case err != nil:
				t.Errorf("Test %v: unwanted error reading embedded words: %v", i, err)
			case wantWords != string(gotWords):
				t.Errorf("Test %v: words from WordsReader not equal:\nwanted: %v\ngot:    %v", i, wantWords, string(gotWords))
			}
		}
	}
}

// TestSQLFiles ensures the files are en order before being sent to the database.
// The table creation scripts must be run before other scripts reference the tables.
func TestSQLFiles(t *testing.T) {
	t.Run("BadFS", func(t *testing.T) {
		e := embeddedData{
			SQLFS: fstest.MapFS{},
		}
		_, err := e.sqlFiles()
		if err == nil {
			t.Error("wanted error with file system without desired sql files")
		}
	})
	t.Run("OkFS", func(t *testing.T) {
		e := embeddedData{
			SQLFS: fstest.MapFS{
				"user_create.sql":                  &fstest.MapFile{Data: []byte("2")},
				"user_delete.sql":                  &fstest.MapFile{Data: []byte("6")},
				"user_read.sql":                    &fstest.MapFile{Data: []byte("3")},
				"users.sql":                        &fstest.MapFile{Data: []byte("1")},
				"user_update_password.sql":         &fstest.MapFile{Data: []byte("4")},
				"user_update_points_increment.sql": &fstest.MapFile{Data: []byte("5")},
			},
		}
		gotFiles, err := e.sqlFiles()
		switch {
		case err != nil:
			t.Errorf("unwanted error: %v", err)
		default:
			got := ""
			for j, f := range gotFiles {
				b, err := io.ReadAll(f)
				if err != nil {
					t.Errorf("could not read file %v: %v", j, err)
				}
				got += string(b)
			}
			want := "123456"
			if want != got {
				t.Errorf("concatenation of files not equal: wanted %v, got %v", want, got)
			}
		}
	})
}
