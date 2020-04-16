package main

import "fmt"

func main() {
	fmt.Println("Hello, WebAssembly!")

	blocker := make(chan int, 0)
	<-blocker
}
