package main

import (
	"fmt"
)

func init() {
	fmt.Println("First init")
}

func init() {
	fmt.Println("Second init")
}

func main() {
	fmt.Println("Hello from Golang!!!")
}
