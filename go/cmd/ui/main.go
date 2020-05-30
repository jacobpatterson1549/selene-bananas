// +build js

package main

import (
	"github.com/jacobpatterson1549/selene-bananas/go/ui"
)

func main() {
	ui.Init()

	// TODO: regester a context with the init shutdown
	done := make(chan struct{})
	<-done
}
