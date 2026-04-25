package main

import (
	"fmt"
)

func main() {
	var num int = 340
	var user_name string
	user_name = "Achyut Pal"

	var credit float64 = 3.5343

	fmt.Println(num)
	fmt.Println(user_name)
	fmt.Println(credit)

	fmt.Println("Enter your age:")

	var input int
	fmt.Scanln(&input)

	fmt.Printf("Entered age is %d\n", input)

}
