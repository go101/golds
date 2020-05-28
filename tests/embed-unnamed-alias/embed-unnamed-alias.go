package foo

type D map[int]int

type A = D
type B = struct{x int} // xxxx ssss
type C = map[B]A
type F = int
type K []int// xxxx ssss
type Q [5]bool// xxxx ssss

func f() (r C) {return}

var L = f()

func g() (r struct{A;B;C}) {return}

var P = g()
