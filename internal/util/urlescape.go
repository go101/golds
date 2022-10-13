package util

import (
	"io"
)

var urlEscapeCharTable = [128]string{
	'"':  "%22",
	'\'': "%27",
}

type URLEscapeWriter struct {
	w io.StringWriter
}

func NewURLEscapeWriter(w io.StringWriter) *URLEscapeWriter {
	return &URLEscapeWriter{w: w}
}

func (ue *URLEscapeWriter) WriteString(data string) (n int, err error) {
	WriteUrlEscapedString(ue.w, data)
	return len(data), nil
}

// Please make sure w.Write never makes errors.
func WriteUrlEscapedString(w io.StringWriter, data string) {
	last := 0
	for i, b := range []byte(data) {
		if b >= 128 {
			continue
		}
		s := urlEscapeCharTable[b]
		if s != "" {
			w.WriteString(data[last:i])
			w.WriteString(s)
			last = i + 1
		}
	}
	w.WriteString(data[last:])
}
