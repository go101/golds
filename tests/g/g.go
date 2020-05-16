/*aaaa
bbbb
cccc*/
package g

import "go101.org/gold/tests/f"

type A struct {
	x int
}

type B struct {
	y bool
}

func (*B) M() {}

// doc doc xxxxx
type (
	// doc doc cccc
	C = struct {
		z string
		*A
		B
	} // comment cccc
)

type D struct {
	*C
	B int
}

func F() {
	type (
		// doc doc cccc
		C = struct {
			z string
			*A
			B
			f.AAA
		} // comment cccc
	)

	type D struct {
		*C
		B int
	}

	var c C
	_ = c
}

// dafafafadf
