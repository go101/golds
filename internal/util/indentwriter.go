package util

import (
	"io"
)

type IndentWriter struct {
	w      io.Writer
	indent []byte

	needWriteIndent bool
}

func NewIndentWriter(w io.Writer, indent []byte) *IndentWriter {
	return &IndentWriter{w, indent, true}

}

func (iw *IndentWriter) writeIndent() {
	iw.w.Write(iw.indent)
}

func (iw *IndentWriter) Write(s []byte) (int, error) {
	//return iw.w.Write(s)
	if len(s) == 0 {
		return 0, nil
	}
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if iw.needWriteIndent {
				iw.writeIndent()
			}

			n, err := iw.w.Write(s[last : i+1])
			if err != nil {
				//panic(err)
				return last + n, err
			}
			last = i + 1

			iw.needWriteIndent = true
		}
	}
	if last < len(s) {
		if iw.needWriteIndent {
			iw.writeIndent()
			iw.needWriteIndent = false
		}
		n, err := iw.w.Write(s[last:])
		if err != nil {
			//panic(err)
			return last + n, err
		}
	}
	return len(s), nil
}
