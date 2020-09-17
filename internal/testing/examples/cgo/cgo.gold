// exportedFunctionCount: 2
package cgo

/*
#include <stdlib.h>
*/
import "C"

func Random() int {
	return int(C.random())
}

func Seed(i int) {
	C.srandom(C.uint(i))
}
