package foo

type Bar int

func (b *Bar) Baz() {
	*b++
	println(*b)
}
