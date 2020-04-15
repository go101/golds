package j

type runtimeObject interface {
	Object
	Object
	Object() int
}

type Object interface {
	GetNamespace() string
}
