package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode"
)

//go:embed embed/version.txt
var embedVersion string

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

// embeddedData is used to retrieve files embedded in the server.
type embeddedData struct {
	Version    string
	StaticFS   fs.FS
	TemplateFS fs.FS
	SQLFS      fs.FS
}

// newEmbedParameters validates, unembeds, and returns the parameters in a structure.
func newEmbedParameters(embedVersion string, embeddedStaticFS, embeddedTemplateFS, embeddedSQLFS fs.FS) (*embeddedData, error) {
	cleanVersion, err := cleanVersion(embedVersion)
	switch {
	case err != nil:
		return nil, err
	case embeddedStaticFS == nil:
		return nil, fmt.Errorf("embedded static file system required")
	case embeddedTemplateFS == nil:
		return nil, fmt.Errorf("embedded template file system required")
	case embeddedSQLFS == nil:
		return nil, fmt.Errorf("embedded sql file system required")
	}
	staticFS, err := unembedFS(embeddedStaticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("unembedding static file system: %w", err)
	}
	templateFS, err := unembedFS(embeddedTemplateFS, "template")
	if err != nil {
		return nil, fmt.Errorf("unembedding template file system: %w", err)
	}
	sqlFS, err := unembedFS(embeddedSQLFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("unembedding sql file system: %w", err)
	}
	e := embeddedData{
		Version:    cleanVersion,
		StaticFS:   staticFS,
		TemplateFS: templateFS,
		SQLFS:      sqlFS,
	}
	return &e, nil
}

// cleanVersion returns the version, but cleaned up to only be letters and numbers.
// Spaces on each end are trimmed, but spaces in the middle of the version or special characters cause an error to be returned.
func cleanVersion(v string) (string, error) {
	cleanV := strings.TrimSpace(v)
	switch {
	case len(cleanV) == 0:
		return "", fmt.Errorf("empty")
	default:
		for i, r := range cleanV {
			if !unicode.In(r, unicode.Letter, unicode.Digit) {
				return "", fmt.Errorf("only letters and digits are allowed: invalid rune at index %v of '%v': '%v'", i, cleanV, string(r))
			}
		}
	}
	return cleanV, nil
}

// sqlFiles opens the SQL files needed to manage user data.
func (e embeddedData) sqlFiles() ([]fs.File, error) {
	sqlFileNames := []string{
		"users",
		"user_create",
		"user_read",
		"user_update_password",
		"user_update_points_increment",
		"user_delete",
	}
	userSQLFiles := make([]fs.File, len(sqlFileNames))
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
