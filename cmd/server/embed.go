package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
)

// EmbeddedFS is the embedded "embed" subdirectory in this package with files for the server.
// !!! Run `make` to generate this directory. !!!
//
//go:embed embed
var EmbeddedFS embed.FS

// EmbeddedData is used to retrieve files embedded in the server.
type EmbeddedData struct {
	Version    []byte
	Words      []byte
	TLSCertPEM []byte
	TLSKeyPEM  []byte
	StaticFS   fs.FS
	TemplateFS fs.FS
	SQLFS      fs.FS
}

// UnembedFS validates, unembeds, and returns the files from the "embed" directory of the file system.
// Version and words are required, file systems are unembedded
func UnembedFS(fsys fs.FS) (*EmbeddedData, error) {
	unembedSubdirectory := func(fsys fs.FS, subdirectory string) (fs.FS, error) {
		if _, err := fsys.Open(subdirectory); err != nil {
			return nil, fmt.Errorf("checking embedded subdirectory existence: %w", err)
		}
		return fs.Sub(fsys, subdirectory)
	}
	embedFS, err := unembedSubdirectory(fsys, "embed")
	if err != nil {
		return nil, fmt.Errorf("unembedding embed director: %w", err)
	}
	version, err := fs.ReadFile(embedFS, "version.txt")
	if err != nil {
		return nil, fmt.Errorf("unembedding version: %w", err)
	}
	embeddedWords, err := fs.ReadFile(embedFS, "words.txt")
	if err != nil {
		return nil, fmt.Errorf("unembedding words file: %w", err)
	}
	tlsCertPEM, err := fs.ReadFile(embedFS, "tls-cert.pem")
	if err != nil {
		return nil, fmt.Errorf("unembedding TLS cert PEM: %w", err)
	}
	tlsKeyPEM, err := fs.ReadFile(embedFS, "tls-key.pem")
	if err != nil {
		return nil, fmt.Errorf("unembedding TLS key PEM: %w", err)
	}
	staticFS, err := unembedSubdirectory(embedFS, "static")
	if err != nil {
		return nil, fmt.Errorf("unembedding static file system: %w", err)
	}
	templateFS, err := unembedSubdirectory(embedFS, "template")
	if err != nil {
		return nil, fmt.Errorf("unembedding template file system: %w", err)
	}
	sqlFS, err := unembedSubdirectory(embedFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("unembedding sql file system: %w", err)
	}
	e := EmbeddedData{
		Version:    version,
		Words:      embeddedWords,
		TLSCertPEM: tlsCertPEM,
		TLSKeyPEM:  tlsKeyPEM,
		StaticFS:   staticFS,
		TemplateFS: templateFS,
		SQLFS:      sqlFS,
	}
	return &e, nil
}

// sqlFiles opens the SQL files needed to manage user data.
func (e EmbeddedData) sqlFiles() ([]io.Reader, error) {
	sqlFileNames := []string{
		"users",
		"user_create",
		"user_read",
		"user_update_password",
		"user_update_points_increment",
		"user_delete",
	}
	userSQLFiles := make([]io.Reader, len(sqlFileNames))
	for i, n := range sqlFileNames {
		n = fmt.Sprintf("%s.sql", n)
		f, err := e.SQLFS.Open(n)
		if err != nil {
			return nil, fmt.Errorf("opening setup file %v: %w", n, err)
		}
		userSQLFiles[i] = f
	}
	return userSQLFiles, nil
}
