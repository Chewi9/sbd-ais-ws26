package main

import "fmt"

type Greeting struct {
	Message string
}


func main() {
	hello := Greeting {
		Message: "Hello, World!",
	}

	fmt.Println(hello.Message)
}