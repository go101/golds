package promoteds

type aaaaaa struct {
	Foo int
	Bar func()
}

func (aaaaaa) Method() {
}

type B struct {
	X int
	aaaaaa
}
