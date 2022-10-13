package server

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"net/http"
	"strconv"

	"go101.org/golds/code"
	"go101.org/golds/internal/util"
)

//type sourcePageKey struct {
//	pkg string
//	src string
//}

func (ds *docServer) sourceCodePage(w http.ResponseWriter, r *http.Request, pkgPath, bareFilename string) {
	w.Header().Set("Content-Type", "text/html")

	//log.Println(pkgPath, bareFilename)

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	if genDocsMode {
		pkgPath = deHashScope(pkgPath)
		bareFilename = deHashFilename(bareFilename)
	}

	// Browers will replace all \ in url to / automatically, so we need convert them back.
	// Otherwise, the file will not be found on Windows.
	//srcPath = strings.Replace(srcPath, "/", string(filepath.Separator), -1)
	//if ds.sourcePages[srcPath] == nil {
	//	result, err := ds.analyzeSoureCode(srcPath)
	//	if err != nil {
	//		// ToDo: not found
	//		fmt.Fprint(w, "Load file (", srcPath, ") error: ", err)
	//		return
	//	}
	//	ds.sourcePages[srcPath] = ds.buildSourceCodePage(result)
	//}
	//w.Write(ds.sourcePages[srcPath])

	//pageKey := sourcePageKey{pkg: pkgPath, src: bareFilename}
	//if ds.sourcePages[pageKey] == nil {
	//	result, err := ds.analyzeSoureCode(pkgPath, bareFilename)
	//	if err != nil {
	//		w.WriteHeader(http.StatusNotFound)
	//		fmt.Fprint(w, "Load file (", bareFilename, ") in ", pkgPath, " error: ", err)
	//		return
	//	}
	//	ds.sourcePages[pageKey] = ds.buildSourceCodePage(result)
	//}
	//w.Write(ds.sourcePages[pageKey])

	pageKey := pageCacheKey{
		resType: ResTypeSource,
		res:     [...]string{pkgPath, bareFilename},
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		result, err := ds.analyzeSoureCode(pkgPath, bareFilename)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Load file (", bareFilename, ") in ", pkgPath, " error: ", err)
			return
		}

		data = ds.buildSourceCodePage(w, result)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildSourceCodePage(w http.ResponseWriter, result *SourceFileAnalyzeResult) []byte {
	page := NewHtmlPage(goldsVersion, ds.currentTranslation.Text_SourceCode(result.PkgPath, result.BareFilename), ds.currentTheme, ds.currentTranslation, createPagePathInfo2b(ResTypeSource, result.PkgPath, "/", result.BareFilename))

	realFilePath := result.OriginalPath
	if result.GeneratedPath != "" {
		realFilePath = result.GeneratedPath
	}

	if genDocsMode {
		fmt.Fprintf(page, `
<pre id="header"><code><span class="title">%s</span>
	%s`,
			page.Translation().Text_SourceFilePath(),
			result.BareFilename,
		)
	} else {
		fmt.Fprintf(page, `
<pre id="header"><code><span class="title">%s</span>
	%s`,
			page.Translation().Text_SourceFilePath(),
			realFilePath,
		)

		if result.OriginalPath != "" && result.OriginalPath != realFilePath {
			fmt.Fprintf(page, `

<span class="title">%s</span>
	%s`,
				page.Translation().Text_GeneratedFrom(),
				result.OriginalPath,
			)
		}
	}

	fmt.Fprintf(page, `

<span class="title">%s</span>
	<a href="%s">%s</a>
</code></pre>
`,
		page.Translation().Text_BelongingPackage(),
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, result.PkgPath), nil, ""),
		result.PkgPath,
	)

	if result.NumRatios > 0 || result.NumImportRatios > 0 {
		page.WriteString("<style>")
		page.WriteString("input[type=radio] {display: none;}\n")
		for i := int32(0); i < result.NumRatios; i++ {
			fmt.Fprintf(page, `input[id=r%[1]d]:checked ~pre label[for=r%[1]d]`, i)
			if i < result.NumRatios-1 {
				page.WriteByte(',')
			}
			page.WriteByte('\n')
		}
		page.WriteString("{background: #226; color: #ff8;}\n")
		for i := int32(0); i < result.NumImportRatios; i++ {
			fmt.Fprintf(page, `input[id=i%[1]d]:checked ~pre .i%[1]d`, i)
			if i < result.NumImportRatios-1 {
				page.WriteByte(',')
			}
			page.WriteByte('\n')
		}
		page.WriteString("{background: brown; color: #eed;}\n")
		page.WriteString("</style>")

		for i := int32(0); i < result.NumRatios; i++ {
			fmt.Fprintf(page, `<input id="r%d" type="radio" name="g"/>`, i)
			page.WriteByte('\n')
		}
		for i := int32(0); i < result.NumImportRatios; i++ {
			fmt.Fprintf(page, `<input id="i%d" type="radio" name="i"/>`, i)
			page.WriteByte('\n')
		}
	}

	page.WriteString(`
<pre class="line-numbers">`)

	var outputNewLine = true
	for i, line := range result.Lines {
		//		fmt.Fprintf(page, `
		//<span class="anchor" id="line-%d"><code>%s</code></span>`,
		//			i+1, line)
		lineNumber := i + 1
		if outputNewLine {
			page.WriteByte('\n')
		}
		if lineNumber == result.DocStartLine {
			page.WriteString(`<div class="anchor" id="doc">`)
		}
		fmt.Fprintf(page, `<span class="codeline" id="line-%d"><code>%s</code></span>`, lineNumber, line)
		if lineNumber == result.DocEndLine {
			page.WriteString(`</div>`)
			outputNewLine = false
		} else {
			outputNewLine = true
		}
	}

	page.WriteString(`
</pre>`)

	return page.Done(w)
}

type SourceFileAnalyzeResult struct {
	PkgPath         string
	BareFilename    string
	OriginalPath    string
	GeneratedPath   string
	Lines           []string
	NumRatios       int32 // not including import idendifiers
	NumImportRatios int32
	DocStartLine    int
	DocEndLine      int
}

/*
var (
	blankID          = []byte("_")
	space            = []byte(" ")
	leftParen        = []byte("(")
	rightParen       = []byte(")")
	period           = []byte(".")
	comma            = []byte(", ")
	semicoloon       = []byte("; ")
	ellipsis         = []byte("...")
	star             = []byte("*")
	leftSquare       = []byte("[")
	rightSquare      = []byte("]")
	leftBrace        = []byte("{")
	rightBrace       = []byte("}")
	mapKeyword       = []byte("map")
	chanKeyword      = []byte("chan")
	chanDir          = []byte("&lt;-")
	funcKeyword      = []byte("func")
	structKeyword    = []byte("struct")
	interfaceKeyword = []byte("interface")

	BoldTagStart = []byte("<b>")
	BoldTagEnd   = []byte("</b>")
)

func WriteFieldList(w io.Writer, fieldList *ast.FieldList, sep []byte, info *types.Info, funcKeywordNeeded bool) {
	WriteFieldListEx(w, fieldList, sep, info, funcKeywordNeeded, nil, nil)
}

func WriteFieldListEx(w io.Writer, fieldList *ast.FieldList, sep []byte, info *types.Info, funcKeywordNeeded bool, recvParam *ast.Field, lvi *ListedValueInfo) {
	if fieldList == nil {
		return
	}
	showRecvName := recvParam != nil && len(recvParam.Names) > 0
	showParamsNames := len(fieldList.List) > 0 && len(fieldList.List[0].Names) > 0
	showParamsNames = showParamsNames || showRecvName

	fields := fieldList.List
	if recvParam != nil {
		fields = append([]*ast.Field{recvParam}, fields...)
	}

	for i, fld := range fields {
		if len(fld.Names) > 0 {
			for k, n := range fld.Names {
				w.Write([]byte(n.Name))
				if k+1 < len(fld.Names) {
					w.Write(comma)
				}
			}
			w.Write(space)
		} else if showParamsNames {
			w.Write(blankID)
			w.Write(space)
		}
		WriteTypeEx(w, fld.Type, info, funcKeywordNeeded, nil, lvi)
		if i+1 < len(fields) {
			w.Write(sep)
		}
	}
}

func WriteType(w io.Writer, typeLit ast.Expr, info *types.Info, funcKeywordNeeded bool) {
	WriteTypeEx(w, typeLit, info, funcKeywordNeeded, nil, nil)
}

type ListedValueInfo struct {
	codePkg *code.Package // the package in which the value is declared
	docPkg  *code.Package // the package in which "forType" is declared

	forTypeName string
}

// For texts in the Index section. Note,
// 1. struct tags are ignored.
// 2. ToDo: "too many fields/methods/params/results" is replaced with ".....".
// Please make sure w.Write never makes errors.
func WriteTypeEx(w io.Writer, typeLit ast.Expr, info *types.Info, funcKeywordNeeded bool, recvParam *ast.Field, lvi *ListedValueInfo) {
	switch node := typeLit.(type) {
	default:
		panic(fmt.Sprint("WriteType, unknown node: ", node))
	case *ast.ParenExpr:
		w.Write(leftParen)
		WriteTypeEx(w, node.X, info, true, nil, lvi)
		w.Write(rightParen)
	case *ast.Ident:
		if lvi != nil {
			// forTypeName should be in lvi.docPkg.
			// lvi.forTypeName should never be builtin types.
			isForTypeName := node.Name == lvi.forTypeName
			obj := lvi.codePkg.PPkg.TypesInfo.ObjectOf(node)
			_, ok := obj.(*types.TypeName)
			// obj.Pkg() might be nil for builtin types.
			if ok && obj.Pkg() != nil && obj.Pkg() != lvi.docPkg.PPkg.Types {
				isForTypeName = false
				w.Write([]byte(obj.Pkg().Name()))
				w.Write(period)
			}

			if isForTypeName {
				w.Write(BoldTagStart)
				w.Write([]byte(node.Name))
				w.Write(BoldTagEnd)
			} else {
				w.Write([]byte(node.Name))
			}
		} else {
			w.Write([]byte(node.Name))
		}
	case *ast.SelectorExpr:
		if lvi != nil {
			isForTypeName := node.Sel.Name == lvi.forTypeName
			obj := lvi.codePkg.PPkg.TypesInfo.ObjectOf(node.Sel)
			// obj.Pkg() might be nil for builtin types.
			if obj.Pkg() != nil && obj.Pkg() != lvi.docPkg.PPkg.Types {
				isForTypeName = false
				w.Write([]byte(obj.Pkg().Name()))
				w.Write(period)
			}

			if isForTypeName {
				w.Write(BoldTagStart)
			}
			w.Write([]byte(node.Sel.Name))
			if isForTypeName {
				w.Write(BoldTagEnd)
			}
		} else {
			//WriteTypeEx(w, node.X, info, true, nil, lvi)
			pkgId, ok := node.X.(*ast.Ident)
			if !ok {
				panic("should not")
			}
			w.Write([]byte(pkgId.Name))
			w.Write(period)
			w.Write([]byte(node.Sel.Name))
		}
	case *ast.StarExpr:
		w.Write(star)
		WriteTypeEx(w, node.X, info, true, nil, lvi)
	case *ast.Ellipsis: // possible? (yes, variadic parameters)
		//panic("[...] should be impossible") // ToDo: go/types package has a case.
		//w.Write(leftSquare)
		w.Write(ellipsis)
		//w.Write(rightSquare)
		WriteTypeEx(w, node.Elt, info, true, nil, lvi)
	case *ast.ArrayType:
		w.Write(leftSquare)
		if node.Len != nil {
			tv, ok := info.Types[node.Len]
			if !ok {
				panic(fmt.Sprint("no values found for ", node.Len))
			}
			w.Write([]byte(tv.Value.String()))
		}
		w.Write(rightSquare)
		WriteTypeEx(w, node.Elt, info, true, nil, lvi)
	case *ast.MapType:
		w.Write(mapKeyword)
		w.Write(leftSquare)
		WriteTypeEx(w, node.Key, info, true, nil, lvi)
		w.Write(rightSquare)
		WriteTypeEx(w, node.Value, info, true, nil, lvi)
	case *ast.ChanType:
		if node.Dir == ast.RECV {
			w.Write(chanDir)
			w.Write(chanKeyword)
		} else if node.Dir == ast.SEND {
			w.Write(chanKeyword)
			w.Write(chanDir)
		} else {
			w.Write(chanKeyword)
		}
		w.Write(space)
		WriteTypeEx(w, node.Value, info, true, nil, lvi)
	case *ast.FuncType:
		if funcKeywordNeeded {
			w.Write(funcKeyword)
			//w.Write(space)
		}
		w.Write(leftParen)
		WriteFieldListEx(w, node.Params, comma, info, true, recvParam, lvi)
		w.Write(rightParen)
		if node.Results != nil && len(node.Results.List) > 0 {
			w.Write(space)
			if len(node.Results.List) == 1 && len(node.Results.List[0].Names) == 0 {
				WriteFieldListEx(w, node.Results, comma, info, true, nil, lvi)
			} else {
				w.Write(leftParen)
				WriteFieldListEx(w, node.Results, comma, info, true, nil, lvi)
				w.Write(rightParen)
			}
		}
	case *ast.StructType:
		w.Write(structKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		WriteFieldListEx(w, node.Fields, semicoloon, info, true, nil, lvi)
		w.Write(rightBrace)
	case *ast.InterfaceType:
		w.Write(interfaceKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		WriteFieldListEx(w, node.Methods, semicoloon, info, false, nil, lvi)
		w.Write(rightBrace)
	}
}
*/

// should be faster than strings.Compare for comparing non-equal package paths.
//func CompareStringsInversely(a, b string) (r int) {
//	//defer func(x, y string) {
//	//	println("Compare ", x, " and ", y, ": ", r)
//	//}(a, b)
//
//	pos, neg := 1, -1
//	if len(a) > len(b) {
//		a, b = b, a
//		pos, neg = neg, pos
//	}
//
//	i, j := len(a)-1, len(b)-1
//	for i >= 0 {
//		if a[i] < b[j] {
//			return neg
//		} else if a[i] > b[j] {
//			return pos
//		}
//		i--
//		j--
//	}
//	if j >= 0 {
//		return neg
//	}
//	return 0
//}

// ToDo: write to page directly.
type astVisitor struct {
	currentPathInfo pagePathInfo

	dataAnalyzer *code.CodeAnalyzer
	pkg          *code.Package
	fset         *token.FileSet
	file         *token.File
	info         *types.Info
	content      []byte

	// ToDo: Some Go files might contains line-repositions.
	//       The current implementation only handles the cgo generated content.
	goFilePath string
	//goFileContentOffset int32
	//goFileLineOffset    int32

	result *SourceFileAnalyzeResult

	// temp vars
	lineNumber int // 1-based
	offset     int
	//lineStartOffsets []int
	//lineBuilder strings.Builder // slower in fact for the specified case
	lineBuilder bytes.Buffer

	//docCommentGroup *ast.CommentGroup

	specialAstNodes *list.List // elements: ast.Node
	// The following old two are merged into the above one.
	//comments          []*ast.Comment
	//pendingTokenPoses []KeywordToken

	sameFileObjects map[types.Object]int32

	astNodeDepth int32

	topLevelFuncNodeDepth int32
	topLevelFuncInfo      *astFunctionInfo

	// ToDo: maybe these top-level things could be merged into one.
	topLevelTypeSpecInfo *ast.TypeSpec

	// ToDo: also support implementation page for local interface types (including unnamed ones).
	//       Local interface types should get IDs like Name-1234.
	topLevelInterfaceTypeNodeDepth int32
	topLevelInterfaceTypeInfo      *astInterfaceTypeInfo

	topLevelStructTypeNodeDepth int32
	topLevelStructTypeSpec      *ast.TypeSpec

	pkgPath2RatioID map[string]int32
}

type astFunctionInfo struct {
	Node         ast.Node
	Name         *ast.Ident
	RecvTypeName string
}

type astInterfaceTypeInfo struct {
	TypeName string
	Methods  []*ast.Field
}

// see https://groups.google.com/forum/#!topic/golang-tools/PaJBT2WjEPQ
type KeywordToken struct {
	keyword string // "range" or "else" or "<-"
	pos     token.Pos
}

func (kw *KeywordToken) Pos() token.Pos {
	return kw.pos
}

func (kw *KeywordToken) End() token.Pos {
	return kw.pos + token.Pos(len(kw.keyword))
}

type ChanCommOprator struct {
	send  bool
	hasOK bool
	pos   token.Pos
}

func (ccp *ChanCommOprator) Pos() token.Pos {
	return ccp.pos
}

func (ccp *ChanCommOprator) End() token.Pos {
	return ccp.pos + token.Pos(len("<-"))
}

func (v *astVisitor) addSpecialNode(n ast.Node) {
	for e := v.specialAstNodes.Front(); e != nil; e = e.Next() {
		en := e.Value.(ast.Node)
		if en.Pos() > n.Pos() {
			v.specialAstNodes.InsertBefore(n, e)
			return
		}
	}
	v.specialAstNodes.PushBack(n)
}

// Output
// * comments,
// * "else" and "range" keywords.
// * "<-" channel receive and send (todo)
func (v *astVisitor) tryToHandleSomeSpecialNodes(beforeNode ast.Node) {
	for e := v.specialAstNodes.Front(); e != nil; {
		next := e.Next()

		en := e.Value.(ast.Node)
		if beforeNode != nil && en.Pos() > beforeNode.Pos() {
			break
		}

		switch node := en.(type) {
		default:
			panic("should not")
		case *ast.CommentGroup:
			v.handleNode(node, "comment", "")
		case *KeywordToken:
			v.handleKeywordToken(node.pos, node.keyword)
		case *ChanCommOprator:
			f := "chansend"
			if !node.send {
				if node.hasOK {
					f = "chanrecv2"
				} else {
					f = "chanrecv1"
				}
			}
			fPosition := v.dataAnalyzer.RuntimeFunctionCodePosition(f)
			if fPosition.IsValid() {
				start := v.pkg.PPkg.Fset.PositionFor(node.Pos(), false)
				end := v.pkg.PPkg.Fset.PositionFor(node.End(), false)
				v.buildText(start, end, "", buildSrouceCodeLineLink(v.currentPathInfo, v.dataAnalyzer, v.dataAnalyzer.RuntimePackage(), fPosition), "")
			}
		}

		// This line will clear the the prev and next elements of e.
		// This is why we cached the next at the loop beginning.
		v.specialAstNodes.Remove(e)
		e = next
	}
}

//func (v *astVisitor) nextComment() *ast.Comment {
//	if len(v.comments) > 0 {
//		return v.comments[0]
//	}
//	return nil
//}

//func (v *astVisitor) removeNextComment() {
//	if len(v.comments) <= 0 {
//		panic("no more comments")
//	}
//	v.comments = v.comments[1:]
//	return
//}

//func (v *astVisitor) lastTokenPos() (KeywordToken, bool) {
//	if n := len(v.pendingTokenPoses); n > 0 {
//		return v.pendingTokenPoses[n-1], true
//	}
//	return KeywordToken{}, false
//}

//func (v *astVisitor) removeLastTokenPos() {
//	if n := len(v.pendingTokenPoses); n <= 0 {
//		panic("no more else statements")
//	} else {
//		v.pendingTokenPoses = v.pendingTokenPoses[:n-1]
//	}
//	return
//}

//func (v *astVisitor) correctPosition(pos *token.Position) {
//	// ToDo: to remove
//	b1 := CompareStringsInversely(pos.Filename, v.goFilePath)
//	b2 := pos.Filename == v.goFilePath
//	if (b1 == 0) != b2 {
//		panic("b1 != b2")
//	}
//
//	if pos.Filename != v.goFilePath {
//		// ToDo: maybe it is needed to cache line offsets of the files
//		//       which contain line re-position directives.
//		//       This has two benefits:
//		//       1. to correct line information
//		//       2. avoid the calculation and memory used in the below part of this function.
//		pos.Line += v.dataAnalyzer.SourceFileLineOffset(pos.Filename)
//		return
//	}
//
//	correctPosition(v.lineStartOffsets, pos)
//}

//func correctPosition(lineOffsets []int, pos *token.Position) {
//	// Find the real line of pos.
//	if len(lineOffsets) == 0 || pos.Offset < 0 {
//		return
//	}
//
//	i, j := 0, len(lineOffsets)
//	for i+1 < j {
//		k := (i + j) / 2
//		if lineOffsets[k] <= pos.Offset {
//			i = k
//		} else {
//			j = k
//		}
//	}
//
//	pos.Line = i + 1 // 1 based
//	if lineOffsets[i+1] <= pos.Offset {
//		pos.Line++
//	}
//}

func (v *astVisitor) writeEscapedHTML(data []byte, class string) {
	if len(data) == 0 {
		return
	}
	if class != "" {
		fmt.Fprintf(&v.lineBuilder, `<span class="%s">`, class)
	}
	util.WriteHtmlEscapedBytes(&v.lineBuilder, data)
	if class != "" {
		v.lineBuilder.WriteString("</span>")
	}
}

func (v *astVisitor) buildConfirmedLines(toLine int, class string) {
	//log.Println("=================== buildConfirmedLines:", v.lineNumber, toLine, v.file.Name())
	for range [1024 * 256]struct{}{} {
		if v.lineNumber >= toLine {
			break
		}
		v.lineNumber++
		//log.Println("v.lineNumber=", v.lineNumber)
		lineStart := v.file.Offset(v.file.LineStart(v.lineNumber))
		lastLineEnd := lineStart
		//log.Println("+++", v.offset, lastLineEnd, lineStart)
		if lastLineEnd > 0 && v.content[lastLineEnd-1] == '\n' {
			lastLineEnd--
		}
		if lastLineEnd > 0 && v.content[lastLineEnd-1] == '\r' {
			lastLineEnd--
		}
		//log.Println("---", v.offset, lastLineEnd, lineStart)
		//if lastLineEnd < v.offset {
		//	// https://github.com/go101/golds/issues/18
		//	bs := v.content[v.offset:]
		//	if len(bs) > 100 {
		//		bs = bs[:100]
		//	}
		//	log.Printf(">>>>>>> debug: %s\n%s\n<<<<<<<<", v.goFilePath, bs)
		//} else {
		v.writeEscapedHTML(v.content[v.offset:lastLineEnd], class)
		//}
		v.buildLine()

		//log.Println("buildConfirmedLines v.offset = lineStart :", lineStart)
		v.offset = lineStart
	}
}

func (v *astVisitor) buildLine() {
	v.result.Lines = append(v.result.Lines, v.lineBuilder.String())
	v.lineBuilder.Reset()
}

func (v *astVisitor) buildText(litStart, litEnd token.Position, class, link, labelForId string) {
	if litStart.Offset < v.offset {
		//log.Printf("already handled: %s", v.content[litStart.Offset:litEnd.Offset])
		// Posible cases:
		// 1. the "func" keyword has been handled in FuncDecl, but re-handled now in FuncType.
		return
	}
	v.buildConfirmedLines(litStart.Line, "")
	v.writeEscapedHTML(v.content[v.offset:litStart.Offset], "")
	v.offset = litStart.Offset

	if labelForId != "" {
		fmt.Fprintf(&v.lineBuilder, `<label for="%s">`, labelForId)
		defer fmt.Fprintf(&v.lineBuilder, `</label>`)
	}
	if link != "" {
		fmt.Fprintf(&v.lineBuilder, `<a href="%s">`, link)
		defer fmt.Fprintf(&v.lineBuilder, `</a>`)
	}
	if litStart.Line != litEnd.Line {
		//log.Println("=============================", litStart.Line, litEnd.Line)
		v.buildConfirmedLines(litEnd.Line, class)
	}
	// This segment will not cross lines for sure.
	v.writeEscapedHTML(v.content[v.offset:litEnd.Offset], class)
	v.offset = litEnd.Offset
}

func (v *astVisitor) buildLink(idStart, idEnd token.Position, link, extraClass string) {
	if idStart.Offset < v.offset {
		//log.Printf("already handled: %s", v.content[litStart.Offset:litEnd.Offset])
		// Posible cases:
		// 1. import spec is handled, but the import name is handled subsequently.
		return
	}
	class := "ident"
	if extraClass != "" {
		class += " " + extraClass
	}
	v.buildConfirmedLines(idStart.Line, "")
	v.writeEscapedHTML(v.content[v.offset:idStart.Offset], "")
	fmt.Fprintf(&v.lineBuilder, `<a href="%s" class="%s">`, link, class)
	defer v.lineBuilder.WriteString(`</a>`)
	v.writeEscapedHTML(v.content[idStart.Offset:idEnd.Offset], "")
	v.offset = idEnd.Offset
}

// func (v *astVisitor) buildIdentifier(idStart, idEnd token.Position, ratioId int32, link, id string) {
func (v *astVisitor) buildIdentifier(idStart, idEnd token.Position, ratioId int32, link string) {
	if idStart.Offset < v.offset {
		//log.Printf("already handled: %s", v.content[litStart.Offset:litEnd.Offset])
		// Posible cases:
		// 1.
		return
	}

	var class = "ident"

	//startOffset := idStart.Offset
	//endOffset := idEnd.Offset
	//log.Println("idStart:", idStart, startOffset)
	//log.Println("idEnd:", idEnd, endOffset)

	//log.Println("@@@ [startOffset, endOffset):", startOffset, endOffset, v.offset)
	//log.Println("@@@ idStart.Line:", idStart.Line, string(v.content[startOffset:endOffset]))
	v.buildConfirmedLines(idStart.Line, "")

	//log.Println("!!!!!!!!!!! @@@ v.offset:", v.offset)

	//v.lineBuilder.Write(v.content[v.offset:startOffsett])
	//v.writeEscapedHTML(v.content[v.offset:idStart.Offset], class)
	v.writeEscapedHTML(v.content[v.offset:idStart.Offset], "")

	if ratioId >= 0 {
		fmt.Fprintf(&v.lineBuilder, `<label for="r%d" class="%s">`, ratioId, class)
		defer v.lineBuilder.WriteString(`</label>`)
	}

	if link != "" {
		if ratioId >= 0 {
		}
		//if id == "" {
		fmt.Fprintf(&v.lineBuilder, `<a href="%s" class="%s">`, link, class)
		//} else {
		//	v.lineBuilder.WriteString(`<a href="` + link + `" class="` + class + `" id="` + id + `">`)
		//}
		defer v.lineBuilder.WriteString(`</a>`)
	}

	//v.lineBuilder.Write(v.content[startOffset:endOffset])
	v.writeEscapedHTML(v.content[idStart.Offset:idEnd.Offset], "")

	//log.Println("buildIdentifier v.offset = endOffset :", endOffset)

	v.offset = idEnd.Offset
}

func (v *astVisitor) finish() {
	v.tryToHandleSomeSpecialNodes(nil)

	//log.Println("v.file.LineCount()=", v.file.LineCount())
	v.buildConfirmedLines(v.file.LineCount(), "")
	endOffset := v.file.Size()
	if endOffset > 0 && v.content[endOffset-1] == '\n' {
		endOffset--
	}
	if endOffset > 0 && v.content[endOffset-1] == '\r' {
		endOffset--
	}

	//log.Println("v.offset < ", v.offset, endOffset, v.offset < endOffset, v.file.Size())
	if v.offset < endOffset {
		//v.lineBuilder.Write(v.content[v.offset:endOffset])
		v.writeEscapedHTML(v.content[v.offset:endOffset], "")
	}
	if v.lineBuilder.Len() > 0 {
		v.buildLine()
	}
}

var (
	StarSlash = []byte("*/")
)

func (v *astVisitor) findTokenBetween(start, maxPos token.Pos, token string, returnFirst bool) *KeywordToken {
	offset := v.file.Offset(start)
	max := v.file.Offset(maxPos)

	var min = offset
	var lastMatchOffset = -1
Loop:
	for ; offset < max; offset++ {
		//log.Println("#", offset, max)
		switch v.content[offset] {
		case '/':
			if offset-1 >= min && v.content[offset-1] == '/' {
				index := bytes.IndexByte(v.content[offset+1:], '\n')
				if index < 0 {
					break Loop
				}
				//offset = (offset + 1) + index + 1 - 1
				min = offset + 1 + index + len("\n")
				offset = min - 1
				//log.Println(" 111: ", offset)
			}
		case '*':
			if v.content[offset-1] == '/' {
				index := bytes.Index(v.content[offset+1:], StarSlash)
				if index < 0 {
					break Loop
				}
				//log.Println(" 222: ", offset, index, index+len(StarSlash)-1)
				//offset = (offset+1) + index + len(StarSlash) - 1
				min = offset + 1 + index + len(StarSlash)
				offset = min - 1
			}
		case token[0]:
			if offset+len(token) > max {
				break Loop
			}

			if string(v.content[offset:offset+len(token)]) == token {
				lastMatchOffset = offset
				if returnFirst {
					break Loop
				}

				min = offset + len(token)
				offset = min - 1
			}
		}
	}

	if lastMatchOffset >= 0 {
		return &KeywordToken{
			keyword: token,
			pos:     v.file.Pos(lastMatchOffset),
		}
	}

	panic("token " + token + " is not found")
}

func (v *astVisitor) findElseToken(ifstmt *ast.IfStmt) *KeywordToken {
	// There might be some comments between ...
	return v.findTokenBetween(ifstmt.Body.End(), ifstmt.Else.Pos(), "else", true)
}

func (v *astVisitor) findRangeToken(rangeStmt *ast.RangeStmt) *KeywordToken {
	pos := rangeStmt.For + token.Pos(len(token.FOR.String()))
	if rangeStmt.Key != nil {
		pos = rangeStmt.TokPos + token.Pos(len(rangeStmt.Tok.String()))
	}
	// There might be some comments between ...
	return v.findTokenBetween(pos, rangeStmt.X.Pos(), "range", true)
}

func (v *astVisitor) findTypeToken(typeswitchStmt *ast.TypeSwitchStmt) *KeywordToken {
	// There might be some comments before ...
	return v.findTokenBetween(typeswitchStmt.Assign.Pos(), typeswitchStmt.Assign.End(), "type", false)
}

func (v *astVisitor) Visit(n ast.Node) (w ast.Visitor) {
	w = v
	//log.Println(">>>>>>>>>>> node:", n)
	//log.Printf(">>>>>>>>>>> node type: %T", n)
	if n == nil {
		v.astNodeDepth--
		if v.astNodeDepth < 0 {
			panic("should not")
		}
		if v.topLevelFuncInfo != nil && v.astNodeDepth == v.topLevelFuncNodeDepth {
			v.topLevelFuncInfo = nil
		}
		if v.topLevelInterfaceTypeInfo != nil && v.astNodeDepth == v.topLevelInterfaceTypeNodeDepth {
			v.topLevelInterfaceTypeInfo = nil
		}
		return
	} else {
		// ToDo: also replace topLevelFuncNodeDepth and topLevelInterfaceTypeNodeDepth in this way?
		if v.topLevelStructTypeSpec != nil && n.Pos() > v.topLevelStructTypeSpec.End() {
			v.topLevelStructTypeSpec = nil
		}
		if v.topLevelTypeSpecInfo != nil && n.Pos() > v.topLevelTypeSpecInfo.End() {
			v.topLevelTypeSpecInfo = nil
		}
	}

	if v.topLevelFuncInfo == nil {
		switch f := n.(type) {
		case *ast.FuncDecl:
			v.topLevelFuncNodeDepth = v.astNodeDepth

			var recvTypeName string
			if f.Recv != nil {
				recvTypeName = func(typeExpr ast.Expr) string {
					for {
						switch e := typeExpr.(type) {
						case *ast.Ident:
							// ToDo: what if this ident is an alias to a pointer type?
							return e.Name
						case *ast.ParenExpr:
							typeExpr = e.X
						case *ast.StarExpr:
							typeExpr = e.X
						//>> 1.18
						case *astIndexExpr:
							typeExpr = e.X
						case *astIndexListExpr:
							typeExpr = e.X
						//<<
						default:
							panic(fmt.Sprintf("impossible type: %T", e))
						}
					}
				}(f.Recv.List[0].Type)
			}

			v.topLevelFuncInfo = &astFunctionInfo{
				Node:         n,
				Name:         f.Name,
				RecvTypeName: recvTypeName,
			}
		case *ast.FuncLit:
			v.topLevelFuncNodeDepth = v.astNodeDepth
			v.topLevelFuncInfo = &astFunctionInfo{
				Node: n,
			}
		}
	}

	if v.topLevelFuncInfo == nil {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if v.topLevelInterfaceTypeInfo == nil {
				it, ok1 := ts.Type.(*ast.InterfaceType)
				tn, ok2 := v.info.ObjectOf(ts.Name).(*types.TypeName)
				if ok1 && ok2 && ts.Name.Name != "_" {
					v.topLevelInterfaceTypeNodeDepth = v.astNodeDepth
					v.topLevelInterfaceTypeInfo = &astInterfaceTypeInfo{
						TypeName: tn.Name(),
						Methods:  it.Methods.List,
					}
				}
			}

			if v.topLevelStructTypeSpec == nil {
				_, ok := ts.Type.(*ast.StructType)
				if ok && ts.Name.Name != "_" {
					v.topLevelStructTypeSpec = ts
					v.topLevelStructTypeNodeDepth = v.astNodeDepth
				}
			}

			if v.topLevelTypeSpecInfo == nil {
				v.topLevelTypeSpecInfo = ts
			}
		}
	}

	// ...
	v.astNodeDepth++

	// ...
	//for {
	//	tokenpos, present := v.lastTokenPos()
	//	if present && tokenpos.Pos > n.Pos() {
	//		present = false
	//	}
	//
	//	comment := v.nextComment()
	//	if comment != nil && comment.Pos() <= n.Pos() {
	//		if present && tokenpos.Pos < comment.Pos() {
	//			v.handleKeywordToken(tokenpos.Pos, tokenpos.Tok)
	//			v.removeLastTokenPos()
	//		}
	//
	//		//log.Println("=== write comment")
	//
	//		v.handleNode(comment, "comment")
	//		v.removeNextComment()
	//		continue
	//	}
	//
	//	if present {
	//		v.handleKeywordToken(tokenpos.Pos, tokenpos.Tok)
	//		v.removeLastTokenPos()
	//		continue
	//	}
	//
	//	break
	//}
	//log.Println(">>>>>>>>>>>>>>>>>>>>>")
	v.tryToHandleSomeSpecialNodes(n)

	//log.Printf("%T", n)

	switch node := n.(type) {
	default:
		//log.Printf("node type: %T", node)

	//case *ast.Comment:
	//	//v.handleNode(node, "comment")
	//	return
	//case *ast.CommentGroup:
	//	//v.handleNode(node, "comment")
	//	return

	// keywords
	case *ast.File:
		v.handleKeyword(node.Package, token.PACKAGE)
	case *ast.SwitchStmt:
		v.handleKeyword(node.Switch, token.SWITCH)
	case *ast.TypeSwitchStmt:
		v.handleKeyword(node.Switch, token.SWITCH)
		v.addSpecialNode(v.findTypeToken(node))
	case *ast.SelectStmt:
		//v.handleKeyword(node.Select, token.SELECT)

		numDefaults, numCases := 0, 0
		var caseComm ast.Stmt
		for _, stmt := range node.Body.List {
			commClause, ok := stmt.(*ast.CommClause)
			if !ok {
				panic("should not")
			}
			if commClause.Comm == nil {
				numDefaults++
				if numDefaults > 1 {
					panic("should not")
				}
			} else {
				numCases++
				if numDefaults > 1 {
					break
				}
				caseComm = commClause.Comm
			}
		}

		f := "selectgo"
		if numDefaults == 1 && numCases == 1 {
			//switch caseStmt := caseComm.(type) {
			switch caseComm.(type) {
			case *ast.SendStmt:
				f = "selectnbsend"
			case *ast.ExprStmt: // <-c
				f = "selectnbrecv"
			case *ast.AssignStmt:
				//if len(caseStmt.Lhs) < 2 {
				f = "selectnbrecv"
				//} else {
				//	f = "selectnbrecv2" // removed since Go 1.17
				//}
			}
		}

		fPosition := v.dataAnalyzer.RuntimeFunctionCodePosition(f)
		if fPosition.IsValid() {
			v.handleSelectKeyword(node.Select, fPosition)
		}
	case *ast.CommClause:
		if node.Comm == nil {
			v.handleKeyword(node.Case, token.DEFAULT)
		} else {
			v.handleKeyword(node.Case, token.CASE)
		}

		switch caseStmt := node.Comm.(type) {
		case *ast.SendStmt:
			v.addSpecialNode(&ChanCommOprator{
				send: true,
				pos:  caseStmt.Arrow,
			})
		case *ast.ExprStmt: // <-c
			unaryExpr, ok := caseStmt.X.(*ast.UnaryExpr)
			if !ok {
				panic("possible?")
			}
			if unaryExpr.Op != token.ARROW {
				panic("possible?")
			}
			v.addSpecialNode(&ChanCommOprator{
				send: false,
				pos:  unaryExpr.OpPos,
			})
		case *ast.AssignStmt:
			if len(caseStmt.Rhs) != 1 {
				panic("possible?")
			}
			unaryExpr, ok := caseStmt.Rhs[0].(*ast.UnaryExpr)
			if !ok {
				panic("possible?")
			}
			if unaryExpr.Op != token.ARROW {
				panic("possible?")
			}
			v.addSpecialNode(&ChanCommOprator{
				send:  false,
				hasOK: len(caseStmt.Lhs) > 1,
				pos:   unaryExpr.OpPos,
			})
		}
	case *ast.CaseClause:
		if node.List == nil {
			v.handleKeyword(node.Case, token.DEFAULT)
		} else {
			v.handleKeyword(node.Case, token.CASE)
		}
	case *ast.BranchStmt:
		v.handleKeyword(node.TokPos, node.Tok)
	case *ast.ReturnStmt:
		v.handleKeyword(node.Return, token.RETURN)
	case *ast.IfStmt:
		v.handleKeyword(node.If, token.IF)
		if node.Else != nil {
			//v.pendingTokenPoses = append(v.pendingTokenPoses, v.findElseToken(node))
			v.addSpecialNode(v.findElseToken(node))
		}
	case *ast.ForStmt:
		v.handleKeyword(node.For, token.FOR)
	case *ast.RangeStmt:
		v.handleKeyword(node.For, token.FOR)
		//v.pendingTokenPoses = append(v.pendingTokenPoses, v.findRangeToken(node))
		v.addSpecialNode(v.findRangeToken(node))
	case *ast.DeferStmt:
		v.handleKeyword(node.Defer, token.DEFER)
	case *ast.GoStmt:
		v.handleKeyword(node.Go, token.GO)
	// ...
	case *ast.ImportSpec:
		// a package might be imported sevral times.
		importRatioId, ok := v.pkgPath2RatioID[node.Path.Value]
		if !ok {
			path, err := strconv.Unquote(node.Path.Value)
			if err != nil {
				//continue
				panic(node.Path.Value + " can't be unqoted")
			}
			//log.Println(v.result.NumImportRatios, path)
			importRatioId = v.result.NumImportRatios
			v.pkgPath2RatioID[path] = importRatioId
			v.result.NumImportRatios++
		}
		if node.Name != nil {
			v.handleIdent(node.Name)
		}
		importClass := fmt.Sprintf("i%d", importRatioId)
		v.handleBasicLit(node.Path, importClass, importClass)

	case *ast.GenDecl:
		v.handleKeyword(node.TokPos, node.Tok)
	case *ast.FuncDecl:
		v.handleKeyword(node.Type.Func, token.FUNC)
	case *ast.FuncType:
		// The "func" kwyword might have already been handled
		// if this FuncType is part of a FuncDecl.
		// See the start of buildText() for details.
		if node.Func != token.NoPos {
			v.handleKeyword(node.Func, token.FUNC)
		}
	case *ast.InterfaceType:
		v.handleKeyword(node.Interface, token.INTERFACE)
	case *ast.MapType:
		v.handleKeyword(node.Map, token.MAP)
	case *ast.StructType:
		v.handleKeyword(node.Struct, token.STRUCT)
	case *ast.ChanType:
		//v.handleKeyword(node.Begin, token.CHAN)
		chanPos := node.Begin
		if chanPos == node.Arrow {
			chanPos = v.findTokenBetween(node.Arrow, node.End(), "chan", true).pos
		}
		v.handleKeyword(chanPos, token.CHAN)
	//...
	case *ast.BasicLit:
		v.handleBasicLit(node, "", "")
	case *ast.Ident:
		v.handleIdent(node)
	}

	return
}

func (v *astVisitor) handleNode(node ast.Node, class, labelForId string) {
	start := v.fset.PositionFor(node.Pos(), false)
	end := v.fset.PositionFor(node.End(), false)
	//log.Println("=============================", start.Line, start.Offset, end.Line, end.Offset)
	//v.correctPosition(&start)
	//v.correctPosition(&end)
	//log.Println("                             ", start.Line, start.Offset, end.Line, end.Offset)

	v.buildText(start, end, class, "", labelForId)
}

func (v *astVisitor) handleBasicLit(basicLit *ast.BasicLit, extraClass, labelForId string) {
	class := "lit-number"
	if basicLit.Kind == token.STRING {
		class = "lit-string"
	}
	if extraClass != "" {
		class += " " + extraClass
	}

	v.handleNode(basicLit, class, labelForId)
}

func (v *astVisitor) handleSelectKeyword(selectPos token.Pos, fPosition token.Position) {
	v.handleToken(selectPos, token.SELECT.String(), "keyword", buildSrouceCodeLineLink(v.currentPathInfo, v.dataAnalyzer, v.dataAnalyzer.RuntimePackage(), fPosition))
}

func (v *astVisitor) handleKeyword(pos token.Pos, tok token.Token) {
	v.handleKeywordToken(pos, tok.String())
}

func (v *astVisitor) handleKeywordToken(pos token.Pos, token string) {
	v.handleToken(pos, token, "keyword", "")
}

func (v *astVisitor) handleToken(pos token.Pos, token, class, link string) {
	length := len(token)
	start := v.fset.PositionFor(pos, false)

	//v.correctPosition(&start)
	end := start
	end.Column += length
	end.Offset += length
	v.buildText(start, end, class, link, "")
}

func (v *astVisitor) handleIdent(ident *ast.Ident) {
	if sourceReadingStyle != SourceReadingStyle_rich {
		return
	}

	start := v.fset.PositionFor(ident.Pos(), false)
	end := v.fset.PositionFor(ident.End(), false)

	if start.Line != end.Line {
		panic(fmt.Sprintf("start.Line != end.Line. %d : %d", start.Line, end.Line))
	}

	var obj types.Object
	// ToDo: why not just call ObjectOf?
	if use, ok := v.info.Uses[ident]; ok {
		obj = use
	} else {
		obj = v.info.ObjectOf(ident)
	}

	if obj == nil {
		//log.Println(fmt.Sprintf("object for identifier %s (%v) is not found", ident.Name, ident.Pos()))
		return
	}

	//log.Printf("==== %s: %T\n", ident.Name, obj)

	if pkgName, ok := obj.(*types.PkgName); ok {
		//v.buildIdentifier(start, end, -1, "/pkg:"+pkgName.Imported().Path())
		importRatioId := v.pkgPath2RatioID[pkgName.Imported().Path()]
		importClass := fmt.Sprintf("i%d", importRatioId)
		//v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, pkgName.Imported().Path()), nil, ""))
		v.buildLink(start, end, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, pkgName.Imported().Path()), nil, ""), importClass)
		return
	}

	objPPkg := obj.Pkg()
	if objPPkg == nil {
		if obj.Parent() == types.Universe {
			for obj.Name() == "make" {
				tv, ok := v.info.Types[ident]
				if !ok {
					break
				}
				sig, ok := tv.Type.Underlying().(*types.Signature)
				if !ok {
					break
				}
				_, ok = sig.Params().At(0).Type().Underlying().(*types.Chan)
				if !ok {
					break
				}
				fPosition := v.dataAnalyzer.RuntimeFunctionCodePosition("makechan")
				if !fPosition.IsValid() {
					break
				}
				v.buildText(start, end, "", buildSrouceCodeLineLink(v.currentPathInfo, v.dataAnalyzer, v.dataAnalyzer.RuntimePackage(), fPosition), "")
				return
			}

			//log.Println(fmt.Sprintf("ppkg for identifier %s (%v) is not found", ident.Name, obj))
			//v.buildIdentifier(start, end, -1, "/pkg:builtin#name-"+obj.Name())
			v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, "builtin"), nil, "")+"#name-"+obj.Name())

			// ToDo: link to runtime.panic/recover/...
			return
		}

		// labels
		// todo: new ratio

		return
	}

	objPkgPath := objPPkg.Path()
	// ToDo: remove (objPkgPath == ""), already handled above. Also (objPkgPath == "builtin")?
	//if objPkgPath == "" || objPkgPath == "unsafe" || objPkgPath == "builtin" {
	//	//log.Println("============== objPkgPath=", objPkgPath)
	// Yes, it is ok to check "unsafe" only here.
	if objPkgPath == "unsafe" {
		//v.buildIdentifier(start, end, -1, "/pkg:"+objPkgPath+"#name-"+obj.Name())
		v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, objPkgPath), nil, "")+"#name-"+obj.Name())
		return
	}

	objPkg := v.dataAnalyzer.PackageByPath(objPkgPath)
	if objPkg == nil {
		panic(fmt.Sprintf("package for object (%v) is not found", obj))
	}

	objPos := objPkg.PPkg.Fset.PositionFor(obj.Pos(), false)

	var inTopFuncRange = v.topLevelFuncInfo != nil &&
		obj.Pos() > v.topLevelFuncInfo.Node.Pos() &&
		obj.Pos() < v.topLevelFuncInfo.Node.End()
	var inTopTypeSpecRange = v.topLevelTypeSpecInfo != nil &&
		obj.Pos() > v.topLevelTypeSpecInfo.Pos() &&
		obj.Pos() < v.topLevelTypeSpecInfo.End()
	for inTopTypeSpecRange {
		if tn, ok := obj.(*types.TypeName); ok {
			if _, ok = tn.Type().(*typesTypeParam); ok {
				break
			}
		}
		inTopTypeSpecRange = false
		break
	}

	var sameFileObjOrderId int32 = -1

	if (inTopFuncRange || inTopTypeSpecRange) && objPos.Filename == v.goFilePath {
		n, ok := v.sameFileObjects[obj]
		if ok {
			sameFileObjOrderId = n
		} else {
			sameFileObjOrderId = v.result.NumRatios // len(v.sameFileObjects)
			v.sameFileObjects[obj] = sameFileObjOrderId
			v.result.NumRatios++
		}
	}
	// ToDo: also link non-exported function names to their references.

	// The declaration of the id is locally, certainly for its uses.
	if sameFileObjOrderId >= 0 {
		var link string
		if inTopFuncRange && v.topLevelFuncInfo.Name == ident {
			funcName := v.topLevelFuncInfo.Name.Name
			if v.topLevelFuncInfo.RecvTypeName != "" {
				// The handling for unexported fileds should be unnecessary here.
				// For a directly declared method has no duplicated methods for sure.
				// Duplicated methods come only through embedding.
				// ToDo: need think more.

				if buildIdUsesPages {
					//if collectUnexporteds || token.IsExported(v.topLevelFuncInfo.RecvTypeName) && token.IsExported(funcName) || v.pkg.Path == "builtin" {
					link = buildPageHref(v.currentPathInfo, createPagePathInfo3(ResTypeReference, v.pkg.Path, "..", v.topLevelFuncInfo.RecvTypeName, funcName), nil, "")
					//}
				} else {
					var methodPkgPath string
					if !token.IsExported(funcName) {
						methodPkgPath = v.pkg.Path
					}
					if sourceReadingStyle == SourceReadingStyle_rich && v.dataAnalyzer.CheckTypeMethodContributingToTypeImplementations(v.pkg.Path, v.topLevelFuncInfo.RecvTypeName, methodPkgPath, funcName) {
						methodIsExported := !token.IsExported(funcName)
						if collectUnexporteds || methodIsExported && token.IsExported(v.topLevelFuncInfo.RecvTypeName) || v.pkg.Path == "builtin" {
							anchorName := funcName
							if !methodIsExported {
								anchorName = methodPkgPath + "." + anchorName
							}
							link = buildPageHref(v.currentPathInfo, createPagePathInfo2(ResTypeImplementation, v.pkg.Path, ".", v.topLevelFuncInfo.RecvTypeName), nil, "", "name-", anchorName)
						}
					}
				}
			} else if collectUnexporteds || token.IsExported(funcName) {
				link = buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, v.pkg.Path), nil, "", "name-", funcName)
			} else {
				goto GoOn // see below "case scp.Parent() == types.Universe:"
			}
		}

		//v.buildIdentifier(start, end, sameFileObjOrderId, "#line-"+strconv.Itoa(objPos.Line), "")
		v.buildIdentifier(start, end, sameFileObjOrderId, link)
		return
	}

GoOn:

	//fmt.Println("========= obj=", obj)
	//fmt.Println("========= objPos=", objPos)
	//fmt.Println("========= objPkgPath=", objPkgPath

	//if !alreadyCheckedEmbeddingType {
	//	if embeddingType, ok := objPkg.PPkg.TypesInfo.Uses[ident]; ok {
	//		log.Printf("=========== %T, %v, %s", embeddingType, ident, start)
	/*
		if field, ok := embeddingType.(*types.Var); ok {
			// obj = v.info.TypeOf(ident) // not good if the type is an unnamed type

			obj = nil
			expr := field.Type
			for {
				switch e := expr.(type) {
				default:
					log.Println("possible?")
				case *ast.StarExpr:
					expr = e.X
				case *ast.Ident:
					obj = v.info.TypeOf(e)
					break
				case *ast.SelectExpr:
					obj = v.info.TypeOf(e)
					break
				}
			}

			alreadyCheckedEmbeddingType = true
			goto AgainForEmbeddingType
		} else {
			log.Println("possible?")
		}
	*/
	//	}
	//}

	// This judegement missses "metav1.ObjectMeta" and "*Name" embedding cases captured in the last if-block.
	if objPos == start {
		// This is an identifier which is just declared.

		// The "if objPos == start" is not correct here,
		// it misses the following embedding cases:
		// . metav1.ObjectMeta
		// . *Ident

		// Local identifiers.
		// ToDo: builtin package is an exception?
		//if obj.Parent() != obj.Pkg().Scope() {
		//	// ToDo: click to highlight all occurrences.
		//}

		switch scp := obj.Parent(); {
		case scp == nil: // fields or interface methods

			// For embedded ones, click to type declarations.
			// For non-embedded ones, click to show reference list.

			// ToDo: if isMethod: click to show all implemented methods.
			//       or click to open a new page which list all implemented methods.

			switch o := obj.(type) {
			case *types.Func: // interface method

				//log.Printf("   parent: %v\n", o.Parent())
				//log.Printf("   scope : %v\n", o.Scope())
				//ot := o.Type().(*types.Signature)
				_ = o
				//log.Printf("   reciver: %v\n", ot.Recv())

				if v.topLevelInterfaceTypeInfo != nil && v.topLevelInterfaceTypeInfo.TypeName != "_" && len(v.topLevelInterfaceTypeInfo.Methods) > 0 {
					if sourceReadingStyle == SourceReadingStyle_rich && ident.Pos() == v.topLevelInterfaceTypeInfo.Methods[0].Pos() {
						methodIsExported := token.IsExported(obj.Name())
						if collectUnexporteds || methodIsExported && token.IsExported(v.topLevelInterfaceTypeInfo.TypeName) || objPkgPath == "builtin" {
							anchorName := obj.Name()
							if !methodIsExported {
								anchorName = objPkgPath + "." + anchorName
							}
							v.buildLink(start, end, buildPageHref(v.currentPathInfo, createPagePathInfo2(ResTypeImplementation, objPkgPath, ".", v.topLevelInterfaceTypeInfo.TypeName), nil, "")+"#name-"+anchorName, "")
							v.topLevelInterfaceTypeInfo.Methods = v.topLevelInterfaceTypeInfo.Methods[1:]
							return
						}
					}
				}
			case *types.Var: // struct field
				// ToDo: generate fake IDs for unnamed types.
				// 5 depth distance from struct type spec to field ident.
				if v.topLevelStructTypeSpec != nil && v.astNodeDepth-v.topLevelStructTypeNodeDepth == 5 {
					enclosingTypeName := v.topLevelStructTypeSpec.Name.Name
					fieldName := obj.Name()
					if fieldName != "_" && buildIdUsesPages {
						//if collectUnexporteds || token.IsExported(enclosingTypeName) && token.IsExported(obj.Name()) || objPkgPath == "builtin" {
						v.buildLink(start, end, buildPageHref(v.currentPathInfo, createPagePathInfo3(ResTypeReference, objPkgPath, "..", enclosingTypeName, fieldName), nil, ""), "")
						return
						//}
					}
				}
				// ToDo: the above code works for the "bar" and "baz" fields, but not for the "X" field.
				//
				// type Foo struct {
				// 	bar Type
				//	baz struct {
				//		X int
				//	}
				//}
				//
				// There are two ways to solve this problem:
				// 1. create a fake type name "unamed-12345" and use "unamed-12345.X" to denote the X field.
				// 2. modify reference/use page implementation to support "Foo.baz.X" (not recommended, may have loop problem).
			}

			goto End

		case scp.Parent() == types.Universe: // package-level elements
			// ToDo:
			// * CTRL to pkg details page.
			// * Click + click to show reference list.

			//v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, objPkgPath), nil, "")+"#name-"+obj.Name())
			// now all unexporteds are listed in package details pages (?show=all is depreciated).
			// All id-ref pages are entered from package details pages now.
			//return

			if collectUnexporteds || obj.Exported() {
				//v.buildIdentifier(start, end, -1, "/pkg:"+objPkgPath+"#name-"+obj.Name())
				v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, objPkgPath), nil, "")+"#name-"+obj.Name())
				return
			} else {
				switch obj.(type) {
				case *types.TypeName:
					if collectUnexporteds {
						//v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, objPkgPath), nil, "")+"?show=all#name-"+obj.Name())
						v.buildIdentifier(start, end, -1, buildPageHref(v.currentPathInfo, createPagePathInfo1(ResTypePackage, objPkgPath), nil, "")+"#name-"+obj.Name())
						return
					} else if buildIdUsesPages {
						//if collectUnexporteds || token.IsExported(obj.Name()) || objPkgPath == "builtin" {
						v.buildLink(start, end, buildPageHref(v.currentPathInfo, createPagePathInfo2(ResTypeReference, objPkgPath, "..", obj.Name()), nil, ""), "")
						return
						//}
					}
				case *types.Func, *types.Var, *types.Const:
					if buildIdUsesPages {
						//if collectUnexporteds || token.IsExported(obj.Name()) || objPkgPath == "builtin" {
						v.buildLink(start, end, buildPageHref(v.currentPathInfo, createPagePathInfo2(ResTypeReference, objPkgPath, "..", obj.Name()), nil, ""), "")
						return
						//}
					}

				}

				// ToDo: open reference list page
			}
		}

		return
	}

	v.buildIdentifier(start, end, -1, buildSrouceCodeLineLink(v.currentPathInfo, v.dataAnalyzer, objPkg, objPos))

End:
	// Handle interface embedding interface cases.
	if v.topLevelInterfaceTypeInfo != nil && len(v.topLevelInterfaceTypeInfo.Methods) > 0 {
		// == for {Writer; MyMethod()}
		// >  for {io.Writer, MyMethod()}
		if ident.Pos() >= v.topLevelInterfaceTypeInfo.Methods[0].Pos() {
			v.topLevelInterfaceTypeInfo.Methods = v.topLevelInterfaceTypeInfo.Methods[1:]
		}
	}

	return
}

func buildSrouceCodeLineLink(currentPathInfo pagePathInfo, analyzer *code.CodeAnalyzer, pkg *code.Package, p token.Position) string {
	//if p.Filename == "" {
	//	panic(fmt.Sprint(pkg.Path, p))
	//}

	var sourceFilename string
	fileInfo := pkg.SourceFileInfoByFilePath(p.Filename)
	if fileInfo == nil {
		log.Printf("! file info for %s in package %s is not found", p.Filename, pkg.Path)
	} else {
		//sourceFilename = fileInfo.BareFilename
		//if sourceFilename == "" {
		//	sourceFilename = fileInfo.BareGeneratedFilename
		//}
		sourceFilename = fileInfo.AstBareFileName()
	}

	return buildPageHref(currentPathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), nil, "", "line-", strconv.Itoa(p.Line))
}

func writeSrouceCodeLineLink(page *htmlPage, pkg *code.Package, p token.Position, text, class string) {
	if class != "" {
		class = fmt.Sprintf(` class="%s"`, class)
	}

	var sourceFilename string
	fileInfo := pkg.SourceFileInfoByFilePath(p.Filename)
	if fileInfo == nil {
		panic(fmt.Sprintf("! file info for %s in package %s is not found", p.Filename, pkg.Path))
	} else {
		//sourceFilename = fileInfo.BareFilename
		//if sourceFilename == "" {
		//	sourceFilename = fileInfo.BareGeneratedFilename
		//}
		sourceFilename = fileInfo.AstBareFileName()
	}

	fmt.Fprintf(page, `<a href="`)
	buildPageHref(page.PathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), page, "", "line-", strconv.Itoa(p.Line))
	fmt.Fprintf(page, `"%s>%s</a>`, class, text)
}

func writeSrouceCodeFileLink(page *htmlPage, pkg *code.Package, sourceFilename string) {
	buildPageHref(page.PathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), page, sourceFilename)
}

func writeSourceCodeDocLink(page *htmlPage, pkg *code.Package, sourceFilename string, startLine, endLine int32) {
	if sourceReadingStyle == SourceReadingStyle_external {
		start, end := strconv.Itoa(int(startLine)), ""
		if endLine == startLine {
			buildPageHref(page.PathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), page, "#d", "doc", "line-", start)
		} else {
			end = strconv.Itoa(int(endLine))
			buildPageHref(page.PathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), page, "#d", "doc", "line-", start, ":", end)
		}
	} else {
		buildPageHref(page.PathInfo, createPagePathInfo2b(ResTypeSource, pkg.Path, "/", sourceFilename), page, "#d", "doc")
	}
	page.WriteByte(' ')
}

func writeMainFunctionArrow(page *htmlPage, pkg *code.Package, mainPos token.Position) {
	if mainPos.IsValid() {
		//mainPos.Line += ds.analyzer.SourceFileLineOffset(mainPos.Filename)
		//writeSrouceCodeLineLink(page, pkg, mainPos, "m-&gt;", "")
		writeSrouceCodeLineLink(page, pkg, mainPos, "#m", "")
	} else {
		//page.WriteString("m-&gt;")
		page.WriteString("#m")
	}
	page.WriteByte(' ')
}

func BuildLineOffsets(content []byte, onlyStatLineCount bool) (int, []int) {
	lineCount := 0
	for data := content; len(data) >= 0; {
		lineCount++
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		data = data[i+1:]
	}

	if onlyStatLineCount {
		return lineCount, nil
	}

	//lineStartOffsets := make([]int, lineCount+1)
	//lineNumber := 0
	//lineStartOffsets[lineNumber] = 0
	//for data := content; len(data) >= 0; {
	//	lineNumber++
	//	i := bytes.IndexByte(data, '\n')
	//	if i < 0 {
	//		break
	//	}
	//	data = data[i+1:]
	//	lineStartOffsets[lineNumber] = lineStartOffsets[lineNumber-1] + i + 1
	//}
	//lineStartOffsets[lineCount] = len(content)
	//return lineCount, lineStartOffsets
	return lineCount, nil
}

// Need locking before calling this function.
func (ds *docServer) analyzeSoureCode(pkgPath, bareFilename string) (*SourceFileAnalyzeResult, error) {
	pkg := ds.analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil, errors.New("package not found")
	}

	//ds.analyzer.loadSourceFiles(pkg)

	var fileInfo = pkg.SourceFileInfoByBareFilename(bareFilename)
	if fileInfo == nil {
		return nil, errors.New("file not found")
	}

	//log.Printf("%#v", fileInfo)

	////generatedFilePath := srcPath
	////filePath := srcPath
	//generatedFilePath := fileInfo.GeneratedFile
	//filePath := fileInfo.OriginalFile
	//if generatedFilePath != "" {
	//	filePath = generatedFilePath
	//	if generatedFilePath == fileInfo.OriginalFile {
	//		generatedFilePath = ""
	//	}
	//}
	generatedFilePath := fileInfo.GeneratedFile
	filePath := fileInfo.OriginalFile
	if filePath == generatedFilePath {
		generatedFilePath = ""
	}
	astFilePath := generatedFilePath
	if astFilePath == "" {
		astFilePath = filePath
	}

	//content, err := ioutil.ReadFile(filePath)
	//if err != nil {
	//	return nil, err
	//}
	content := fileInfo.Content
	if content == nil {
		return nil, errors.New("source code content not available: " + filePath)
	}

	//log.Println("===================== goFilePath=", srcPath)
	//log.Println("===================== filePath=", filePath)

	var docStartLine, docEndLine int
	if fileInfo.AstFile != nil && fileInfo.AstFile.Doc != nil {
		start := pkg.PPkg.Fset.PositionFor(fileInfo.AstFile.Doc.Pos(), false)
		end := pkg.PPkg.Fset.PositionFor(fileInfo.AstFile.Doc.End(), false)
		docStartLine = start.Line
		docEndLine = end.Line
	}

	var result *SourceFileAnalyzeResult
	//if !enableSoruceNavigation || fileInfo.AstFile == nil {
	if sourceReadingStyle == SourceReadingStyle_plain || fileInfo.AstFile == nil {

		//log.Println("fileInfo == nil")

		lineCount, _ := BuildLineOffsets(content, true)

		result = &SourceFileAnalyzeResult{
			PkgPath:       pkg.Path,
			BareFilename:  bareFilename,
			OriginalPath:  fileInfo.OriginalFile,
			GeneratedPath: generatedFilePath,
			Lines:         make([]string, 0, lineCount),
			DocStartLine:  docStartLine,
			DocEndLine:    docEndLine,
		}
		var buf bytes.Buffer
		buf.Grow(1024)
		for data := content; len(data) > 0; {
			i := bytes.IndexByte(data, '\n')
			k := i
			if k < 0 {
				k = len(data)
			}
			if k > 0 && data[k-1] == '\r' {
				k--
			}
			util.WriteHtmlEscapedBytes(&buf, data[:k])
			result.Lines = append(result.Lines, buf.String())
			buf.Reset()

			if i < 0 {
				break
			}
			data = data[i+1:]
		}
	} else {

		//_, lineStartOffsets := BuildLineOffsets(content, false)

		fset := pkg.PPkg.Fset
		file := fset.File(fileInfo.AstFile.Pos())

		if file.Size() != len(content) {
			panic(fmt.Sprintf("file sizes not match. %d : %d. %s. %s", file.Size(), len(content), file.Name(), filePath))
		}

		//log.Println("===================== GoFileContentOffset=", fileInfo.GoFileContentOffset)
		//log.Println("===================== GoFileLineOffset=", fileInfo.GoFileLineOffset)

		specialAstNodes := list.New()
		for _, cg := range fileInfo.AstFile.Comments {
			specialAstNodes.PushBack(cg)
		}

		av := &astVisitor{
			currentPathInfo: createPagePathInfo2b(ResTypeSource, pkg.Path, "/", bareFilename),

			dataAnalyzer: ds.analyzer,
			pkg:          pkg,
			fset:         pkg.PPkg.Fset,
			file:         file,
			info:         pkg.PPkg.TypesInfo,
			content:      content,

			//goFilePath: filePath, // fileInfo.OriginalFile?
			goFilePath: astFilePath,

			//goFileContentOffset: fileInfo.GoFileContentOffset,
			//goFileLineOffset:    fileInfo.GoFileLineOffset,

			result: &SourceFileAnalyzeResult{
				PkgPath:       pkg.Path,
				BareFilename:  bareFilename,
				OriginalPath:  fileInfo.OriginalFile,
				GeneratedPath: generatedFilePath,
				Lines:         make([]string, 0, file.LineCount()),
				DocStartLine:  docStartLine,
				DocEndLine:    docEndLine,
			},

			lineNumber: 1,
			offset:     0,
			//lineStartOffsets: lineStartOffsets,

			//docCommentGroup: fileInfo.AstFile.Doc,

			specialAstNodes: specialAstNodes,
			//comments:          comments,
			//pendingTokenPoses: make([]TokenPos, 0, 10),

			sameFileObjects: make(map[types.Object]int32, 256),
		}
		av.lineBuilder.Grow(1024)
		av.pkgPath2RatioID = make(map[string]int32, len(fileInfo.AstFile.Imports))
		// ToDo: construct pkgPath2RatioID here?

		//if fileInfo.GoFileContentOffset > 0 {
		//	ab.buildConfirmedLines(int(fileInfo.GoFileLineOffset+1), "")
		//}
		ast.Walk(av, fileInfo.AstFile)
		av.finish()

		if n := av.specialAstNodes.Len(); n > 0 {
			log.Println("!!!", filePath, "has still", n, "special ast node(s) not handled yet.")
		}

		result = av.result
	}

	return result, nil
}
