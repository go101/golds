package identifiers_on_one_line

import "bytes"

func foo(buf *bytes.Buffer) {
	_ = 1
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	buf.Write(nil)
	_ = 2
	buf.Write(nil)
	_ = 3
}
