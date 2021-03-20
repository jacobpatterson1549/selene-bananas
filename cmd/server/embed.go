package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed embed/version.txt
var embedVersion string

//go:embed embed/words.txt
var embeddedWords string

//go:embed embed/tls-cert.pem
var embeddedTLSCertPEM string

//go:embed embed/tls-key.pem
var embeddedTLSKeyPEM string

//go:embed embed/sql
var embeddedSQLFS embed.FS

//go:embed embed/template
var embeddedTemplateFS embed.FS

//go:embed embed/static
var embeddedStaticFS embed.FS

// unembedFS returns the embed/subdirectory subdirectory of the file system.
func unembedFS(fsys fs.FS, subdirectory string) (fs.FS, error) {
	dir := filepath.Join("embed", subdirectory)
	// Open the directory to ensure it exists.
	// This ensures the file systems passed to newEmbeddedData are not out of order.
	_, err := fsys.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("unembedding file system to %v: %w", dir, err)
	}
	return fs.Sub(fsys, dir)
}

// EmbeddedData is used to retrieve files embedded in the server.
type EmbeddedData struct {
	Version    string
	Words      string
	TLSCertPEM string
	TLSKeyPEM  string
	StaticFS   fs.FS
	TemplateFS fs.FS
	SQLFS      fs.FS
}

// unEmbed validates, unembeds, and returns the parameters in a structure.
// Version and words are required, file systems are unembedded
func (e EmbeddedData) unEmbed() (*EmbeddedData, error) {
	switch {
	case len(e.Version) == 0:
		return nil, fmt.Errorf("version required")
	case len(e.Words) == 0:
		return nil, fmt.Errorf("empty words file provided")
	case e.StaticFS == nil:
		return nil, fmt.Errorf("embedded static file system required")
	case e.TemplateFS == nil:
		return nil, fmt.Errorf("embedded template file system required")
	case e.SQLFS == nil:
		return nil, fmt.Errorf("embedded sql file system required")
	}
	trimmedVersion := strings.TrimSpace(e.Version)
	staticFS, err := unembedFS(e.StaticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("unembedding static file system: %w", err)
	}
	templateFS, err := unembedFS(e.TemplateFS, "template")
	if err != nil {
		return nil, fmt.Errorf("unembedding template file system: %w", err)
	}
	sqlFS, err := unembedFS(e.SQLFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("unembedding sql file system: %w", err)
	}
	e2 := EmbeddedData{
		Version:    trimmedVersion,
		Words:      e.Words,
		TLSCertPEM: e.TLSCertPEM,
		TLSKeyPEM:  e.TLSKeyPEM,
		StaticFS:   staticFS,
		TemplateFS: templateFS,
		SQLFS:      sqlFS,
	}
	return &e2, nil
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
