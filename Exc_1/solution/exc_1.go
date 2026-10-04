package main

import "fmt"

type Greeting struct {
	Message string
}

func (g Greeting) Print() {
	fmt.Println(g.Message)
}


func main() {
	hello := Greeting {
		Message: "Hello, World!",
	}

	hello.Print()
}