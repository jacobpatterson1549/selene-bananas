package main

import (
	"embed"
	"io/fs"
	"path/filepath"
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
	return fs.Sub(fsys, dir)
}
