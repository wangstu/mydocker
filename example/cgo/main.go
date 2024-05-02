package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
)

func main() {
	Seed(123)
	fmt.Println("Random: ", Random())
}

func Seed(i int) {
	C.srandom(C.uint(i))
}

// Random 产生一个随机数
func Random() int {
	return int(C.random())
}
