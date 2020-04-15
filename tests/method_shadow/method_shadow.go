package method_shadow

type Conn struct {
	FrameReader
	PayloadType int
}

type FrameReader interface {
	PayloadType() byte
}
