package util

import (
	"io"
)

// NOTE: the following write operations assume that the write target is memory based.
//       That means no errors should happen in the writing process.

var htmlEscapeTable = [128][]byte{
	'&': []byte("&amp;"),
	'<': []byte("&lt;"),
	'>': []byte("&gt;"),

	//'\'': []byte("&#39;"), // "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	//'"':  []byte("&#34;"), // "&#34;" is shorter than "&quot;"
}

// Please make sure w.Write never makes errors.
func WriteHtmlEscapedBytes(w io.Writer, data []byte) {
	last := 0
	for i, b := range data {
		if b >= 128 {
			continue
		}
		s := htmlEscapeTable[b]
		if s != nil {
			w.Write(data[last:i])
			w.Write(s)
			last = i + 1
		}
	}
	w.Write(data[last:])
}

type HTMLEscapeWriter struct {
	w io.Writer
	s [1]byte
}

func NewHTMLEscapeWriter(w io.Writer) *HTMLEscapeWriter {
	return &HTMLEscapeWriter{w: w}
}

func (he *HTMLEscapeWriter) Write(data []byte) (n int, err error) {
	WriteHtmlEscapedBytes(he.w, data)
	return len(data), nil
}

// Please make sure w.Write never makes errors.
func (he *HTMLEscapeWriter) WriteString(data string) (n int, err error) {
	var s, w = he.s[:], he.w
	for _, b := range []byte(data) {
		if b < 128 {
			s := htmlEscapeTable[b]
			if s != nil {
				w.Write(s)
				continue
			}
		}
		s[0] = b
		w.Write(s)
	}
	return len(data), nil
}
