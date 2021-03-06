// +build tools

// Package tools imports commands used to generate code.
package tools

import (
	_ "golang.org/x/tools/cmd/stringer" // generate String() func for integer type constants.
)
