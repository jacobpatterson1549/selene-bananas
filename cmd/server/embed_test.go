package main

import (
	"io"
	"io/fs"
	"reflect"
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
	okVersion := "9d2ffad8e5e5383569d37ec381147f2d"
	words := "apple\nbanana\ncarrot"
	unembedOrFail := func(fs fs.FS, subdirectory string) fs.FS {
		fs, err := unembedFS(fs, subdirectory)
		if err != nil {
			t.Errorf("unwanted error unembedding %v file system: %v", subdirectory, err)
			return &fstest.MapFS{}
		}
		return fs
	}
	newEmbedParametersTests := []struct {
		EmbeddedData
		wantOk bool
		want   *EmbeddedData
	}{
		{}, // bad version
		{ // missing words
			EmbeddedData: EmbeddedData{
				Version: okVersion,
			},
		},
		{ // missing static fs
			EmbeddedData: EmbeddedData{
				Version: okVersion,
				Words:   words,
			},
		},
		{ // missing template fs
			EmbeddedData: EmbeddedData{
				Version:  okVersion,
				Words:    words,
				StaticFS: emptyFS,
			},
		},
		{ // missing SQL fs
			EmbeddedData: EmbeddedData{
				Version:    okVersion,
				Words:      words,
				StaticFS:   emptyFS,
				TemplateFS: emptyFS,
			},
		},
		{ // bad static fs
			EmbeddedData: EmbeddedData{
				Version:    okVersion,
				Words:      words,
				StaticFS:   emptyFS,
				TemplateFS: emptyFS,
				SQLFS:      emptyFS,
			},
		},
		{ // bad template fs
			EmbeddedData: EmbeddedData{
				Version:    okVersion,
				Words:      words,
				StaticFS:   staticFS,
				TemplateFS: emptyFS,
				SQLFS:      emptyFS,
			},
		},
		{ // bad SQL fs
			EmbeddedData: EmbeddedData{
				Version:    okVersion,
				Words:      words,
				StaticFS:   staticFS,
				TemplateFS: templateFS,
				SQLFS:      emptyFS,
			},
		},
		{ // happy path
			EmbeddedData: EmbeddedData{
				Version:    okVersion + "\n",
				Words:      words,
				TLSCertPEM: "tls cert",
				TLSKeyPEM:  "tls key",
				StaticFS:   staticFS,
				TemplateFS: templateFS,
				SQLFS:      sqlFS,
			},
			wantOk: true,
			want: &EmbeddedData{
				Version:    "9d2ffad8e5e5383569d37ec381147f2d", // trimmed
				Words:      words,
				TLSCertPEM: "tls cert",
				TLSKeyPEM:  "tls key",
				StaticFS:   unembedOrFail(staticFS, "static"),
				TemplateFS: unembedOrFail(templateFS, "template"),
				SQLFS:      unembedOrFail(sqlFS, "sql"),
			},
		},
	}
	for i, test := range newEmbedParametersTests {
		got, err := test.EmbeddedData.unEmbed()
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v: not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

// TestSQLFiles ensures the files are en order before being sent to the database.
// The table creation scripts must be run before other scripts reference the tables.
func TestSQLFiles(t *testing.T) {
	t.Run("BadFS", func(t *testing.T) {
		e := EmbeddedData{
			SQLFS: fstest.MapFS{},
		}
		_, err := e.sqlFiles()
		if err == nil {
			t.Error("wanted error with file system without desired sql files")
		}
	})
	t.Run("OkFS", func(t *testing.T) {
		e := EmbeddedData{
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
