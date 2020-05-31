package a

import "x.y/a/b"
import "x.y/a/c"

// methodCount: 2
type Ia interface {
	b.I
	c.I
}

// fieldCount: 4
type Sa struct {
	b.Sb
	c.Sc
}
