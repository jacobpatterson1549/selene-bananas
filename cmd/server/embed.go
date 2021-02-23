package main

import (
	"embed"
	"io/fs"
)

//go:embed embed/version.txt
var embedVersion string

//go:embed embed/sql/users.sql
//go:embed embed/sql/user_create.sql
//go:embed embed/sql/user_read.sql
//go:embed embed/sql/user_update_password.sql
//go:embed embed/sql/user_update_points_increment.sql
//go:embed embed/sql/user_delete.sql
var embeddedSQLFS embed.FS

//go:embed embed/html
//go:embed embed/fa
//go:embed embed/favicon.svg
//go:embed embed/index.css
//go:embed embed/*.js
//go:embed embed/manifest.json
var embeddedTemplateFS embed.FS

//go:embed embed/main.wasm
//go:embed embed/wasm_exec.js
//go:embed embed/robots.txt
//go:embed embed/favicon.png
//go:embed embed/LICENSE
var embeddedStaticFS embed.FS

// unembedFS returns the embed/ subdirectory of the file system.
func unembedFS(fsys fs.FS) (fs.FS, error) {
	return fs.Sub(fsys, "embed")
}
