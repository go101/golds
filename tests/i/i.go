package i

import "io"

type Conn struct {
	frameReader
	PayloadType byte
}

type frameReader interface {
	// Reader is to read payload of the frame.
	io.Reader

	// PayloadType returns payload type.
	PayloadType() byte

	// HeaderReader returns a reader to read header of the frame.
	HeaderReader() io.Reader

	// TrailerReader returns a reader to read trailer of the frame.
	// If it returns nil, there is no trailer in the frame.
	TrailerReader() io.Reader

	// Len returns total length of the frame, including header and trailer.
	Len() int
}

func f() {
	var c Conn
	//var x func() byte = c.PayloadType
	var x = c.PayloadType
	_ = x

	// typeutil.MethsetCache has bug.
}
