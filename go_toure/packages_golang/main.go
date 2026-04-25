package main

import "fmt"

func main() {

	var list []int = []int{}
	list = append(list, 23)
	list = append(list, 22)
	list = append(list, -9)
	list = append(list, -90)
	list = append(list, 1293)
	list = append(list, 10)

	var count []int = make([]int, 3)

	count = append(count, 123)
	count = append(count, -123)
	count = append(count, 11)
	count = append(count, -12)
	count = append(count, 91)

	for i := 0; i < len(count); i++ {
		fmt.Println(i)

	}

}
