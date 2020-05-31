package foo

type runtimeObject interface {
	Object
	Object
	Object() int
	GetNamespace() string
}

type Object interface {
	GetNamespace() string
}
