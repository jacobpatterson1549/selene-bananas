package main

import "fmt"

func main() {
	fmt.Println("Hello, WebAssembly!") // TODO

	blocker := make(chan int, 0)
	<-blocker
}
