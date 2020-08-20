// Package main initializes interactive frontend elements and runs as long as the webpage is open.
package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMainImports ensures encoding/json, net/*, and fmt are not used.
// This keeps the wasm file small and allows it to work with tinygo.
func TestMainImports(t *testing.T) {
	imports, err := packageNames()
	if err != nil {
		t.Fatalf("getting package names: %v", err)
	}
	contains := func(pkg string, isSuffix bool) bool {
		for _, i := range imports {
			if i == pkg || (isSuffix && strings.HasSuffix(i, pkg)) {
				return true
			}
		}
		return false
	}
	mainImportsTests := []struct {
		name     string
		isSuffix bool
		wantOk   bool
	}{
		{
			name:   "syscall/js",
			wantOk: true,
		},
		{
			name:     "ui/dom",
			isSuffix: true,
			wantOk:   true,
		},
		{
			name:     "ui/dom/url",
			isSuffix: true,
			wantOk:   true,
		},
		{
			name:     "server",
			isSuffix: true,
		},
		{
			name: "net/http",
		},
		{
			name: "net/url",
		},
		{
			name: "net/fmt",
		},
		{
			name: "encoding/json",
		},
	}
	for _, test := range mainImportsTests {
		got := contains(test.name, test.isSuffix)
		if test.wantOk != got {
			message := "imports to "
			if !test.wantOk {
				message += "NOT "
			}
			message += "contain " + test.name
			if test.isSuffix {
				message += " (as a suffix)"
			}
			t.Errorf("wanted %v, got: %v", message, imports)
		}
	}
}

func packageNames() ([]string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return nil, fmt.Errorf("could not get caller of function")
	}
	dir := filepath.Dir(filename)
	dirs := []string{dir}
	imports := make(map[string]struct{})
	for len(dirs) > 0 {
		dir, dirs = dirs[0], dirs[1:]
		pkg, err := build.ImportDir(dir, 0)
		if err != nil {
			return nil, fmt.Errorf("importing %v: %w", dir, err)
		}
		imports2 := pkg.Imports
		for _, i := range imports2 {
			if _, ok := imports[i]; !ok {
				imports[i] = struct{}{}
				dir2 := filepath.Join(pkg.SrcRoot, i)
				fi, err := os.Stat(dir2)
				switch {
				case err != nil:
					if !os.IsNotExist(err) {
						return nil, fmt.Errorf("checking if import %v in %v is a file: %w", i, dir, err)
					}
				case fi.IsDir():
					dirs = append(dirs, dir2)
				}
			}
		}
	}
	packageNames := make([]string, 0, len(imports))
	for pn := range imports {
		packageNames = append(packageNames, pn)
	}
	return packageNames, nil
}
