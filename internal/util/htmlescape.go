package util

import (
	"io"
)

// NOTE: the following write operations assume that the write target is memory based.
//       That means no errors should happen in the writing process.

var (
	andBytes     = []byte("&amp;")
	smallerBytes = []byte("&lt;")
	largerBytes  = []byte("&gt;")
)

// Please make sure w.Write never makes errors.
func WriteHtmlEscapedBytes(w io.Writer, data []byte) {
	last := 0
	for i, b := range data {
		switch b {
		case '&':
			w.Write(data[last:i])
			w.Write(andBytes)
			last = i + 1
		case '<':
			w.Write(data[last:i])
			w.Write(smallerBytes)
			last = i + 1
		case '>':
			w.Write(data[last:i])
			w.Write(largerBytes)
			last = i + 1
		}
	}
	w.Write(data[last:])
}

func WriteHtmlEscapedString(w io.Writer, data string) {
	var bs = make([]byte, 1)
	for _, b := range []byte(data) {
		switch b {
		default:
			bs[0] = b
			w.Write(bs)
		case '&':
			w.Write(andBytes)
		case '<':
			w.Write(smallerBytes)
		case '>':
			w.Write(largerBytes)
		}
	}
}

type HTMLEscapeWriter struct {
	w io.Writer
}

func MakeHTMLEscapeWriter(w io.Writer) HTMLEscapeWriter {
	return HTMLEscapeWriter{w}
}

func (he HTMLEscapeWriter) Write(data []byte) (n int, err error) {
	WriteHtmlEscapedBytes(he.w, data)
	return len(data), nil
}
