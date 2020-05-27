package variadic

import "io"

var X func(a int, b ...int)

func Foo(int, ...int) {}

var Y struct {
	AAA int
	io.Reader
	int
	_ bool
}

var Z interface {
	error
	io.Reader
	Foo(...int) func() int
}
