package main

import (
	"io"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestUnembedFS(t *testing.T) {
	version := []byte("v")
	words := []byte("a\nb\nc")
	tlsCert := []byte("C")
	tlsKey := []byte("K")
	unembedOrFail := func(fsys fs.FS, subdirectory string) fs.FS {
		embedFS, err := fs.Sub(fsys, "embed")
		if err != nil {
			t.Fatal(err)
		}
		dir, err := fs.Sub(embedFS, subdirectory)
		if err != nil {
			t.Fatal(err)
		}
		return dir
	}
	unembedFSTests := []struct {
		fs.FS
		wantOk bool
		want   *EmbeddedData
	}{
		{ // no embed subdirectory
			FS: fstest.MapFS{},
		},
		{ // no version
			FS: fstest.MapFS{
				"embed/": &fstest.MapFile{},
			},
		},
		{ // no words
			FS: fstest.MapFS{
				"embed/version.txt": &fstest.MapFile{Data: version},
			},
		},
		{ // no tls cert
			FS: fstest.MapFS{
				"embed/version.txt": &fstest.MapFile{Data: version},
				"embed/words.txt":   &fstest.MapFile{Data: words},
			},
		},
		{ // no tls key
			FS: fstest.MapFS{
				"embed/version.txt":  &fstest.MapFile{Data: version},
				"embed/words.txt":    &fstest.MapFile{Data: words},
				"embed/tls-cert.pem": &fstest.MapFile{Data: tlsCert},
			},
		},
		{ // no static fs
			FS: fstest.MapFS{
				"embed/version.txt":  &fstest.MapFile{Data: version},
				"embed/words.txt":    &fstest.MapFile{Data: words},
				"embed/tls-cert.pem": &fstest.MapFile{Data: tlsCert},
				"embed/tls-key.pem":  &fstest.MapFile{Data: tlsKey},
			},
		},
		{ // no template fs
			FS: fstest.MapFS{
				"embed/version.txt":  &fstest.MapFile{Data: version},
				"embed/words.txt":    &fstest.MapFile{Data: words},
				"embed/tls-cert.pem": &fstest.MapFile{Data: tlsCert},
				"embed/tls-key.pem":  &fstest.MapFile{Data: tlsKey},
				"embed/static":       &fstest.MapFile{},
			},
		},
		{ // no SQL fs
			FS: fstest.MapFS{
				"embed/version.txt":  &fstest.MapFile{Data: version},
				"embed/words.txt":    &fstest.MapFile{Data: words},
				"embed/tls-cert.pem": &fstest.MapFile{Data: tlsCert},
				"embed/tls-key.pem":  &fstest.MapFile{Data: tlsKey},
				"embed/static":       &fstest.MapFile{},
				"embed/template":     &fstest.MapFile{},
			},
		},
		{ // happy path
			FS: fstest.MapFS{
				"embed/version.txt":  &fstest.MapFile{Data: version},
				"embed/words.txt":    &fstest.MapFile{Data: words},
				"embed/tls-cert.pem": &fstest.MapFile{Data: tlsCert},
				"embed/tls-key.pem":  &fstest.MapFile{Data: tlsKey},
				"embed/static":       &fstest.MapFile{},
				"embed/template":     &fstest.MapFile{},
				"embed/sql":          &fstest.MapFile{},
			},
			wantOk: true,
			want: &EmbeddedData{
				Version:    version,
				Words:      words,
				TLSCertPEM: tlsCert,
				TLSKeyPEM:  tlsKey,
			},
		},
	}
	for i, test := range unembedFSTests {
		got, err := unembedFS(test.FS)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			// create the embedded file systems from the parent embedded file system for deep equal to work
			test.want.StaticFS = unembedOrFail(test.FS, "static")
			test.want.TemplateFS = unembedOrFail(test.FS, "template")
			test.want.SQLFS = unembedOrFail(test.FS, "sql")
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v: not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
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
			var got []byte
			for j, f := range gotFiles {
				b, err := io.ReadAll(f)
				if err != nil {
					t.Errorf("could not read file %v: %v", j, err)
				}
				got = append(got, b...)
			}
			want := "123456"
			if want != string(got) {
				t.Errorf("concatenation of files not equal: wanted %v, got %s", want, got)
			}
		}
	})
}
