package a

import "unsafe"

import _ "bytes"

import rand1 "math/rand"
import rand2 "crypto/rand"

var R1 = rand1.Read
var R2 = rand2.Read

type _ int

type _ bool

type X unsafe.Pointer

type P  interface {
	M1()
}

type T = interface {
	M1()
	M2()
}

type Q  interface {
	P
	M2()
}

type Sa struct {
	x int	
}
