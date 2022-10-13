package util

import (
	"bytes"
	"fmt"
	"testing"
)

var mdRender MarkdownRenderer

func makeURL_dummy(string) string {
	return ""
}

func Test_RenderCase1(t *testing.T) {
	var n = 10
Next:
	n--
	if n < 0 {
		return
	}

	var mdText = `
	https://go101.org end
	https://go101.org, end
	(https://go101.org) end
		"https://go101.org" end
	[aaa]
	[bbb]

	[aaa]: http://aaa.com
	[bbb]: http://bbb.com end
`
	var expected = trimLeadingBlankLines(trimEndingSpaces(`
	<a href="https://go101.org">https://go101.org</a> end
	<a href="https://go101.org">https://go101.org</a>, end
	(<a href="https://go101.org">https://go101.org</a>) end
		"https://go101.org" end
	<a href="http://aaa.com">[aaa]</a>
	[bbb]

	[bbb]: <a href="http://bbb.com">http://bbb.com</a> end
`))

	var buf bytes.Buffer
	mdRender.Render(&buf, mdText, "", false, makeURL_dummy)
	if out := buf.String(); out != expected {
		t.Errorf(`===== case 1 (a) =====
[input]
%s

[output]
%s

[expected]
%s
`,
			mdText, out, expected)
	}

	var expectedB = trimLeadingBlankLines(trimEndingSpaces(`
	:	:<a href="https://go101.org">https://go101.org</a> end
	:	:<a href="https://go101.org">https://go101.org</a>, end
	:	:(<a href="https://go101.org">https://go101.org</a>) end
	:	:	"https://go101.org" end
	:	:<a href="http://aaa.com">[aaa]</a>
	:	:[bbb]
	:	:
	:	:[bbb]: <a href="http://bbb.com">http://bbb.com</a> end
`))

	var bufB bytes.Buffer
	mdRender.Render(&bufB, mdText, "\t:\t:", true, makeURL_dummy)
	if out := bufB.String(); out != expectedB {
		t.Errorf(`===== case 1 (b) =====
[input]
%s

[output]
%s

[expected]
%s
`,
			mdText, out, expectedB)
	}

	goto Next
}

func Test_RenderCase2(t *testing.T) {
	var n = 10
Next:
	n--
	if n < 0 {
		return
	}

	var mdText = fmt.Sprintf(`
	%[1]shttps://go101.org end
	https://go101.org%[1]s, end
	(https://go101.org) end
	   "https://go101.org" end
	[aaa
	bbb]

	[aaa bbb]: http://aaa.com
`,
		"`")

	var expected = trimLeadingBlankLines(trimEndingSpaces(fmt.Sprintf(`
%[1]shttps://go101.org end
https://go101.org%[1]s, end
(<a href="https://go101.org">https://go101.org</a>) end
   "https://go101.org" end
<a href="http://aaa.com">[aaa</a>
<a href="http://aaa.com">bbb]</a>
`,
		"`")))

	var buf bytes.Buffer
	mdRender.Render(&buf, mdText, "", true, makeURL_dummy)
	if out := buf.String(); out != expected {
		t.Errorf(`===== case 2 (a) =====
[input]
%s

[output]
%s

[expected]
%s
`,
			mdText, out, expected)
	}

	var expectedB = trimLeadingBlankLines(trimEndingSpaces(fmt.Sprintf(`
>>> 	%[1]shttps://go101.org end
>>> 	https://go101.org%[1]s, end
>>> 	(<a href="https://go101.org">https://go101.org</a>) end
>>> 	   "https://go101.org" end
>>> 	<a href="http://aaa.com">[aaa</a>
>>> <a href="http://aaa.com">	bbb]</a>
`,
		"`")))

	var bufB bytes.Buffer
	mdRender.Render(&bufB, mdText, ">>> ", false, makeURL_dummy)
	if out := bufB.String(); out != expectedB {
		t.Errorf(`===== case 2 (b) =====
[input]
%s

[output]
%s

[expected]
%s
`,
			mdText, out, expectedB)
	}

	goto Next
}
