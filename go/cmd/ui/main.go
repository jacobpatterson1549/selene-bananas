// +build js, wasm
package main

import (
	"fmt"
	"syscall/js"
)

func main() {
	fmt.Println("Hello, WebAssembly!") // TODO
	js.Global().Call("alert", "Hello, JavaScript")
}
