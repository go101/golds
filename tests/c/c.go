package c

type A interface {
	m1()
}

type B interface {
	m1()
}

type C interface {
	A
}

type D interface {
	A
	B
}

type T struct{}

func (T) m1() {}
func (T) M2() {}
