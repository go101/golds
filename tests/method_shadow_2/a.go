package a

import "x.y/a/b"
import "x.y/a/c"

type Ia interface {
	b.I
	c.I
}

type Sa struct {
	b.Sb
	c.Sc
}
