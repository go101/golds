package util

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

func printLinkDefinition(bracketLeading, urlSeg *segment) {
	return
	fmt.Println(bracketLeading.text, urlSeg.text)
}

func IsAsciiLetter(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func CommonPrefix(x, y string) string {
	if len(x) > len(y) {
		x, y = y, x
	}
	if len(x) <= len(y) { // more code but more efficient
		for i := 0; i < len(x); i++ {
			if x[i] != y[i] { // bound check eliminated
				return x[:i]
			}
		}
	}
	return x
}

func unbracket(s string) string {
	var n = len(s)
	if n < 2 {
		panic("n < 2")
	}
	if s[0] != '[' || s[n-1] != ']' {
		panic("not bracketed")
	}
	return s[1 : n-1]
}

func trimLeadingBlankLines(s string) string {
	for len(s) > 0 {
		var k = 0
		for {
			var r, n = utf8.DecodeRuneInString(s[k:])
			if unicode.IsSpace(r) {
				k += n
				if r == '\n' {
					s = s[k:]
					break
				}
			} else if n == 0 {
				return ""
			} else {
				return s
			}
		}
	}
	return ""
}

func trimEndingSpaces(s string) string {
	for len(s) > 0 {
		var r, n = utf8.DecodeLastRuneInString(s)
		if unicode.IsSpace(r) {
			s = s[:len(s)-n]
		} else {
			break
		}
	}
	return s
}

func trimLeadingSpaces(s string) (string, string, bool) {
	for i, r := range s {
		if unicode.IsSpace(r) {
			if r == '\n' {
				return s[:i], s[i+1:], true
			}
			if r == '\r' {
				if len(s) > i+1 && s[i+1] == '\n' {
					return s[:i], s[i+2:], true
				}
			}
		} else {
			return s[:i], s[i:], false
		}
	}
	return s, "", true
}

func trimRunes(s string, rn rune) (string, string) {
	for i, r := range s {
		if r != rn {
			return s[:i], s[i:]
		}
	}
	return s, ""
}

func trimLine(s string) (string, string) {
	for i, r := range s {
		if unicode.IsSpace(r) {
			if r == '\n' {
				return s[:i], s[i+1:]
			}
			if r == '\r' {
				if len(s) > i+1 && s[i+1] == '\n' {
					return s[:i], s[i+2:]
				}
			}
		}
	}
	return s, ""
}

func trimURL(s string) (string, string) {
	if len(s) < 8 {
		return "", s
	}

	var isNeither = func(v, x, y byte) bool {
		return v != x && v != y
	}

	if isNeither(s[0], 'h', 'H') {
		return "", s
	}
	if isNeither(s[1], 't', 'Y') {
		return "", s[1:]
	}
	if isNeither(s[2], 't', 'Y') {
		return "", s[2:]
	}
	if isNeither(s[3], 'p', 'P') {
		return "", s[3:]
	}
	var s2 = s[4:]
	if s[4] == 's' {
		s2 = s2[1:]
	}
	if s2[0] != ':' {
		return "", s2
	}
	if s2[1] != '/' {
		return "", s2[1:]
	}
	if s2[2] != '/' {
		return "", s2[2:]
	}
	s2 = s2[3:]

	var i = 0
	for {
		var r, n = utf8.DecodeRuneInString(s2[i:])
		if n == 0 || unicode.IsSpace(r) {
			break
		}
		i += n
	}
	if i == 0 {
		return "", s2
	}
	s2 = s2[i:]

	return s[:len(s)-len(s2)], s2
}

type segmentKind int

const (
	Plain segmentKind = iota
	DirectLinkURL
	BracketText
)

type segment struct {
	atLineStart bool
	atLineEnd   bool
	isBlankLine bool

	kind segmentKind
	text string

	linkDefText string
	url         string

	next *segment
}

/*
A MarkdownRenderer is used to render docs and comments.
It only builds links. Even for building links,
it is not fully compatiable with https://go.dev/doc/comment.

## Test case 1

std lib type: [io.Reader]
std lib funciton: [fmt.Println]
current package type: [StopWatch]
current package funciton: [MemoryUse], [trimEndingSpaces]
current package selector: [MarkdownRenderer.freeSegments], [MarkdownRenderer.Render]
slibing package function: [app.Main], [app.run]
slibing package slector: [server.TypeForListing.Named], [server.TypeForListing.Position]
module top-level package type: [*code.Package],
module top-level package funciton: [code.ComparePackagePaths], [code.getMatchedPackages]

## Test case 2

The following texts are for testing purpose.

[Test 1] bla bla.
Ref: https://example.com/?a=b#something
[Test Test
Test
222]

[Test 1]: https://test1.example
[Test Test Test 222]: https://tes2.example
*/
type MarkdownRenderer struct {
	freeSegments *segment
	zeroSegment  segment
}

func (r *MarkdownRenderer) newSegment() (seg *segment) {
	if r.freeSegments != nil {
		seg = r.freeSegments
		r.freeSegments = seg.next
		return
	}
	return &segment{}
}

func (r *MarkdownRenderer) collectSegment(head *segment) {
	if head == nil {
		return
	}

	var seg = head
	for {
		*seg = r.zeroSegment
		if seg.next == nil {
			break
		}
		seg = seg.next
	}

	seg.next = r.freeSegments
	r.freeSegments = head
}

func (render *MarkdownRenderer) Render(w interface {
	io.StringWriter
	io.Writer
}, mdText string, indent string, removeLineCommonPrefix bool, makeURL func(string) string) {
	mdText = trimLeadingBlankLines(trimEndingSpaces(mdText))

	var headSegment *segment
	var lastSegment *segment
	var linkDefinitions = map[string]string{}

	defer func() {
		render.collectSegment(headSegment)
	}()

	var onNewSegment = func(seg *segment) {
		if headSegment == nil {
			headSegment = seg
		}
		if lastSegment != nil {
			lastSegment.next = seg
		}
		lastSegment = seg

		//printSegment(seg)
	}

	var newSegment = func(kind segmentKind, text string, lineStart, lineEnd, isBlankLine bool) *segment {
		var seg = render.newSegment()
		seg.atLineStart = lineStart
		seg.atLineEnd = lineEnd
		seg.isBlankLine = isBlankLine
		seg.kind = kind
		seg.text = text

		onNewSegment(seg)
		return seg
	}

	// Find [xxx] and http://xxx and https://xxx, ignoring the ones between
	// ` pairs and ^``` pairs.

	var commonIdent string
	var firstNonBlankLine = true

	var remaining = mdText
	var inCodeSpan bool
	var allowLinkDefinition = true

	var lastBracketSeg *segment

	for len(remaining) > 0 {
		var spaces, unchecked, reachLineEnd = trimLeadingSpaces(remaining)

		if reachLineEnd {
			newSegment(Plain, spaces, true, true, true)
			remaining = unchecked

			inCodeSpan = false
			allowLinkDefinition = true
			lastBracketSeg = nil
			continue
		}

		if firstNonBlankLine {
			commonIdent = spaces
			firstNonBlankLine = false
		} else {
			commonIdent = CommonPrefix(commonIdent, spaces)
		}

		if makeURL == nil {
			var line, theRest = trimLine(unchecked)
			newSegment(Plain, remaining[:len(spaces)+len(line)], true, true, false)
			unchecked = theRest
			remaining = unchecked
			continue
		}

		backticks, unchecked := trimRunes(unchecked, '`')
		if numBaclticks := len(backticks); numBaclticks == 0 {
			// the most common case
		} else if numBaclticks == 1 {
			inCodeSpan = !inCodeSpan
			allowLinkDefinition = false
		} else if numBaclticks >= 3 {
			inCodeSpan = false
			allowLinkDefinition = true
			lastBracketSeg = nil

			var line, theRest = trimLine(unchecked)
			newSegment(Plain, remaining[:len(spaces)+len(backticks)+len(line)], true, true, false)

			for len(theRest) > 0 {
				spaces, unchecked, reachLineEnd = trimLeadingSpaces(theRest)
				if reachLineEnd {
					newSegment(Plain, spaces, true, true, false)
					theRest = unchecked
					continue
				}

				backticks, unchecked = trimRunes(unchecked, '`')
				spaces2, unchecked, reachLineEnd := trimLeadingSpaces(unchecked)
				if reachLineEnd {
					newSegment(Plain, theRest[:len(spaces)+len(backticks)+len(spaces2)], true, true, false)
					if len(backticks) >= numBaclticks {
						theRest = unchecked
						break
					}
				}

				line, unchecked = trimLine(unchecked)
				newSegment(Plain, theRest[:len(spaces)+len(backticks)+len(spaces2)+len(line)], true, true, false)
				theRest = unchecked
			}

			remaining = theRest

			continue
		} else { // if numBaclticks == 2 {
			allowLinkDefinition = false
			// keep inCodeSpan status unchanged.
		}

		var lastIsAsciiLetter = false
		var isLineStart = true
		var bracketLeading = ""                   // in current line
		var firstBracketSeg, firstUrlSeg *segment // to determine link definitions
		var lastLineEndSeg = lastSegment

	NextRuneInLine:

		var r, n = utf8.DecodeRuneInString(unchecked)
		if n == 0 {
			goto LineEnd
		}

		if r == '`' {
			inCodeSpan = !inCodeSpan
		} else {
			var checkLineEnd = inCodeSpan
			if !inCodeSpan {
				checkLineEnd = false
				if r == '[' {
					lastBracketSeg = nil
					bracketLeading = unchecked
				} else if r == ']' {
					if lastBracketSeg != nil {
						var sb strings.Builder
						if s := strings.TrimSpace(lastBracketSeg.text[1:]); len(s) > 0 {
							sb.WriteString(s)
							sb.WriteByte(' ')
						}

						var bracketSeg = lastBracketSeg
						for bracketSeg != lastSegment {
							bracketSeg = bracketSeg.next
							if s := strings.TrimSpace(bracketSeg.text); len(s) > 0 {
								sb.WriteString(s)
								sb.WriteByte(' ')
							}
						}
						var linkDefText = ""
						unchecked = unchecked[n:]
						var seg = newSegment(Plain, remaining[:len(remaining)-len(unchecked)], isLineStart, false, false)
						if s := strings.TrimSpace(seg.text[:len(seg.text)-1]); len(s) > 0 {
							sb.WriteString(s)
							linkDefText = sb.String()
						} else {
							linkDefText = strings.TrimSpace(sb.String())
						}

						if len(linkDefText) > 0 {
							bracketSeg = lastBracketSeg
							for {
								bracketSeg.kind = BracketText
								bracketSeg.linkDefText = linkDefText

								if bracketSeg == lastSegment {
									break
								}
								bracketSeg = bracketSeg.next
							}
						}

						lastBracketSeg = nil

						bracketLeading = ""
						isLineStart = false
						lastIsAsciiLetter = false
						remaining = unchecked

						goto NextRuneInLine
					} else if bracketLeading != "" {
						if len(bracketLeading)-len(unchecked) == 1 {
							bracketLeading = ""
							goto NextRuneInLine
						}

						var firstInLine = isLineStart && allowLinkDefinition
						if k := len(remaining) - len(bracketLeading); k > 0 {
							var seg = newSegment(Plain, remaining[:k], isLineStart, false, false)
							if firstInLine {
								if len(seg.text) != len(spaces) {
									firstInLine = false
								}
							}
							isLineStart = false
						}

						unchecked = unchecked[n:]
						var seg = newSegment(BracketText, bracketLeading[:len(bracketLeading)-len(unchecked)], isLineStart, false, false)
						if firstInLine {
							firstBracketSeg = seg
						}

						bracketLeading = ""
						isLineStart = false
						lastIsAsciiLetter = false
						remaining = unchecked

						goto NextRuneInLine
					}
				} else {
					if lastBracketSeg == nil && bracketLeading == "" && !lastIsAsciiLetter {
						var urlText, theRest = trimURL(unchecked)
						if urlText != "" {
							if k := len(remaining) - len(unchecked); k > 0 {
								newSegment(Plain, remaining[:k], isLineStart, false, false)
								isLineStart = false
							}

							var seg = newSegment(DirectLinkURL, urlText, isLineStart, false, false)
							if firstBracketSeg != nil {
								if firstUrlSeg == nil &&
									firstBracketSeg.next != nil &&
									firstBracketSeg.next.kind == Plain &&
									firstBracketSeg.next.next == seg &&
									trimEndingSpaces(firstBracketSeg.next.text) == ":" {
									firstUrlSeg = seg
								} else {
									firstBracketSeg = nil
									firstUrlSeg = nil
								}
							}
							isLineStart = false

							// will enter the next if-block for sure

							unchecked = theRest
							remaining = unchecked
							goto NextRuneInLine
						} else if theRest != unchecked {
							unchecked = theRest
							goto NextRuneInLine
						}
					}
					checkLineEnd = true
				}
			}

			if checkLineEnd {
				var reachLineEnd = false
				if r == '\n' {
					reachLineEnd = true
				} else if r == '\r' {
					if len(unchecked) == 1 {
						reachLineEnd = true
					} else if unchecked[1] == '\n' {
						n++
						reachLineEnd = true
					}
				}

				if reachLineEnd {
					goto LineEnd
				}
			}
		}

		lastIsAsciiLetter = IsAsciiLetter(r)
		unchecked = unchecked[n:]
		goto NextRuneInLine

	LineEnd:

		if bracketLeading != "" {
			if k := len(remaining) - len(bracketLeading); k > 0 {
				newSegment(Plain, remaining[:k], isLineStart, false, false)
				isLineStart = false
			}
			remaining = bracketLeading

			var k = len(remaining) - len(unchecked)
			lastBracketSeg = newSegment(Plain, remaining[:k], isLineStart, true, false)
		} else if k := len(remaining) - len(unchecked); k > 0 {
			newSegment(Plain, remaining[:k], isLineStart, true, false)
		} else if lastSegment != nil {
			lastSegment.atLineEnd = true
		}

		unchecked = unchecked[n:]
		remaining = unchecked
		//bracketLeading = ""

		if firstUrlSeg == lastSegment ||
			firstUrlSeg != nil &&
				firstUrlSeg.next == lastSegment &&
				//lastSegment.text.kind == Plain && // surely
				strings.TrimSpace(lastSegment.text) == "" {
			printLinkDefinition(firstBracketSeg, firstUrlSeg)
			linkDefinitions[strings.TrimSpace(unbracket(firstBracketSeg.text))] = firstUrlSeg.text

			if lastLineEndSeg != nil {
				render.collectSegment(lastLineEndSeg.next)
				lastLineEndSeg.next = nil
			} else {
				headSegment = nil
			}
			lastSegment = lastLineEndSeg
		} else {
			allowLinkDefinition = false
		}
	}

	// trace (for debug)

	if false {
		var seg = headSegment
		for seg != nil {
			fmt.Println("============== ", seg.kind, seg.text, ":", seg.atLineStart, seg.atLineEnd, seg.isBlankLine)
			seg = seg.next
		}
	}

	if makeURL == nil {
		goto Render
	}

	// prune

	{
		for headSegment != nil {
			if headSegment.isBlankLine {
				headSegment = headSegment.next
			} else {
				break
			}
		}

		var isIdentedCodeLine = false
		var seg = headSegment
		for seg != nil {
			if seg.atLineStart {
				isIdentedCodeLine = false
				if len(seg.text) > len(commonIdent) {
					var r, _ = utf8.DecodeRuneInString(seg.text[len(commonIdent):])
					isIdentedCodeLine = unicode.IsSpace(r)
				}
			}

			if seg.isBlankLine {
				panic("seg should not be blank")
			}

			if isIdentedCodeLine {
				seg.kind = Plain
			} else if seg.kind == DirectLinkURL {
				var text = seg.text
				if strings.HasSuffix(text, ")") {
					var k = strings.Index(text, "://")
					if k < 0 {
						panic("should not")
					}
					var n = 0
					for _, r := range text[k+3:] {
						if r == '(' {
							n--
						} else if r == ')' {
							n++
						}
					}
					if n > 0 {
						text = text[:len(text)-n]
					}
				}
				{
					var n = 0
					for i := len(text) - 1; i >= 0; i-- {
						if text[i] == '"' {
							n++
						} else {
							break
						}
					}
					if n > 0 {
						text = text[:len(text)-n]
					} else {
						for _, s := range []string{",", ".", "!", "?", ":", ";"} {
							if strings.HasSuffix(seg.text, s) {
								text = text[:len(text)-1]
								break
							}
						}
					}
				}

				if len(seg.text) > len(text) {
					var urlText, punctText = text, seg.text[len(text):]
					if len(urlText) <= 8 && strings.HasSuffix(urlText, "://") {
						seg.kind = Plain
					} else {
						var punctSeg = &segment{
							atLineStart: false,
							atLineEnd:   seg.atLineEnd,
							isBlankLine: false,
							kind:        Plain,
							text:        punctText,
							next:        seg.next,
						}

						seg.text = urlText
						seg.url = urlText
						seg.atLineEnd = false

						seg.next = punctSeg
						seg = punctSeg
					}
				} else {
					seg.url = seg.text
				}
			} else if seg.kind == BracketText {
				var linkDefText = seg.linkDefText
				if linkDefText == "" {
					linkDefText = strings.TrimSpace(unbracket(seg.text))
					seg.linkDefText = linkDefText
				}
				seg.url = linkDefinitions[linkDefText]
			}

			var nextSeg = seg.next
			if nextSeg == nil {
				break
			}

			if nextSeg.isBlankLine {
			CheckMore:
				var nextnext = nextSeg.next
				if nextnext == nil {
					seg.next = nil
					break
				}

				if nextnext.isBlankLine {
					nextSeg.next = nextnext.next
					goto CheckMore
				}

				seg = nextnext
			} else {
				seg = nextSeg
			}
		}
	}

	// render
Render:

	{
		if makeURL == nil {
			makeURL = func(string) string {
				return ""
			}
		}

		var htmlEscapeWriter = NewHTMLEscapeWriter(w)
		var urlEscapeWriter = NewURLEscapeWriter(w)
		var writeLink = func(url, text string) {
			w.WriteString(`<a href="`)
			urlEscapeWriter.WriteString(url)
			w.WriteString(`">`)
			htmlEscapeWriter.WriteString(text)
			w.WriteString(`</a>`)
		}

		var seg = headSegment
		for seg != nil {
			var text = seg.text
			if seg.atLineStart {
				w.WriteString(indent)
			}

			if !seg.isBlankLine {
				if seg.atLineStart && removeLineCommonPrefix {
					text = text[len(commonIdent):]
				}

				switch seg.kind {
				default:
					panic(seg.kind)
				case Plain:
					htmlEscapeWriter.WriteString(text)
				case DirectLinkURL:
					writeLink(seg.url, text)
				case BracketText:
					if seg.url != "" {
						writeLink(seg.url, text)
					} else if url := makeURL(seg.linkDefText); url != "" {
						writeLink(url, text)
					} else {
						htmlEscapeWriter.WriteString(text)
					}
				}
			}

			if seg.atLineEnd && seg.next != nil {
				w.WriteString("\n")
			}

			seg = seg.next
		}
	}
}
