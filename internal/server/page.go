package server

import (
	//"bytes"
	"fmt"
	"go/build"
	"io"
	"net/http"
	"strings"
	"sync"

	"go101.org/golds/internal/server/translations"
	"go101.org/golds/internal/util"
)

type PageOutputOptions struct {
	GoldsVersion string

	PreferredLang string

	NoStatistics           bool
	NoIdentifierUsesPages  bool
	NotCollectUnexporteds  bool
	AllowNetworkConnection bool
	VerboseLogs            bool
	RenderDocLinks         bool
	UnfoldAllInitially     bool
	SourceReadingStyle     string
	WdPkgsListingManner    string
	FooterShowingManner    string

	// ToDo:
	//ListUnexportedRes   bool
}

var (
	testingMode = false
	genDocsMode = false // static docs generation mode
	//footerHTML  string  // for static docs generation mode only (bad idea, for image relation urls are different on different pages)

	goldsVersion string

	enabledPageCache = true // for web serving mode only

	showStatistics   = true
	buildIdUsesPages = true // might be false in gen mode
	//enableSoruceNavigation = true // false to disable method implementation pages and some code reading features
	sourceReadingStyle = SourceReadingStyle_rich
	collectUnexporteds = true // false to not collect package-level resources
	//emphasizeWDPackages    = false // list packages in the current directory before other packages
	allowNetworkConnection = false
	wdPkgsListingManner    = WdPkgsListingManner_general
	footerShowingManner    = FooterShowingManner_none

	renderDocLinks     = false
	unfoldAllInitially = false

	verboseLogs = false

	// ToDo: use this one to replace the above ones, and put it in docServer (good or bad?).
	pageOutputOptions PageOutputOptions

	writeExternalSourceCodeLink func(w writer, pkgFile, line, endLine string) (handled bool, err error)
)

// This function should be called at prgram startup phase once.
func setPageOutputOptions(options PageOutputOptions, forTesting bool) {
	goldsVersion = options.GoldsVersion
	showStatistics = !options.NoStatistics || forTesting
	buildIdUsesPages = !options.NoIdentifierUsesPages || forTesting
	sourceReadingStyle = options.SourceReadingStyle
	collectUnexporteds = !options.NotCollectUnexporteds || forTesting
	allowNetworkConnection = options.AllowNetworkConnection && !forTesting
	renderDocLinks = options.RenderDocLinks || forTesting
	unfoldAllInitially = options.UnfoldAllInitially && !forTesting
	wdPkgsListingManner = options.WdPkgsListingManner
	footerShowingManner = options.FooterShowingManner
	verboseLogs = options.VerboseLogs
}

const (
	WdPkgsListingManner_general  = "general"
	WdPkgsListingManner_promoted = "promoted"
	WdPkgsListingManner_solo     = "solo"

	FooterShowingManner_none               = "none"
	FooterShowingManner_simple             = "simple"
	FooterShowingManner_verbose            = "verbose"
	FooterShowingManner_verbose_and_qrcode = "verbose+qrcode"

	SourceReadingStyle_plain     = "plain"
	SourceReadingStyle_highlight = "highlight"
	SourceReadingStyle_rich      = "rich"
	SourceReadingStyle_external  = "external" // auto detect project hosting URL
)

type pageResType string

const (
	ResTypeNone           pageResType = ""
	ResTypeAPI            pageResType = "api"
	ResTypeModule         pageResType = "mod"
	ResTypePackage        pageResType = "pkg"
	ResTypeDependency     pageResType = "dep"
	ResTypeImplementation pageResType = "imp"
	ResTypeSource         pageResType = "src"
	ResTypeReference      pageResType = "use"
	ResTypeCSS            pageResType = "css"
	ResTypeJS             pageResType = "jvs"
	ResTypeSVG            pageResType = "svg"
	ResTypePNG            pageResType = "png"
)

func isHTMLPage(res pageResType) bool {
	switch res {
	default:
		panic("unknown resource type: " + res)
	case ResTypeAPI, ResTypeCSS, ResTypeJS, ResTypeSVG, ResTypePNG:
		return false
	case ResTypeNone:
	case ResTypeModule:
	case ResTypePackage:
	case ResTypeDependency:
	case ResTypeImplementation:
	case ResTypeSource:
	case ResTypeReference:
	}
	return true
}

type pageCacheKey struct {
	resType pageResType
	res     interface{}
	options interface{}
}

type pageCacheValue struct {
	data    []byte
	options interface{}
}

func (ds *docServer) cachePage(key pageCacheKey, data []byte) {
	if enabledPageCache && ds.cachedPages != nil {
		if data == nil {
			delete(ds.cachedPages, key)
		} else {
			ds.cachedPages[key] = data
		}
	}
}

func (ds *docServer) cachedPage(key pageCacheKey) (data []byte, ok bool) {
	if enabledPageCache && ds.cachedPages != nil {
		data, ok = ds.cachedPages[key]
	}
	return
}

//func (ds *docServer) cachePageOptions(key pageCacheKey, options interface{}) {
//	if genDocsMode {
//	} else {
//		key.options = nil
//		ds.cachedPagesOptions[key] = options
//	}
//}

//func (ds *docServer) cachedPageOptions(key pageCacheKey) (options interface{}) {
//	if genDocsMode {
//	} else {
//		key.options = nil
//		options = ds.cachedPagesOptions[key]
//	}
//	return
//}

func addVersionToFilename(filename string, version string) string {
	return filename + "-" + version
}

func removeVersionFromFilename(filename string, version string) string {
	return strings.TrimSuffix(filename, "-"+version)
}

type htmlPage struct {
	//bytes.Buffer
	content Content

	//goldsVersion string
	PathInfo pagePathInfo

	// ToDo: use the two instead of server.currentXXXs.
	//theme Theme
	translation Translation

	isHTML bool

	htmlEscapeWriter *util.HTMLEscapeWriter
}

func (page *htmlPage) Translation() Translation {
	return page.translation
}

func NewHtmlPage(goldsVersion, title string, theme Theme, translation Translation, currentPageInfo pagePathInfo) *htmlPage {
	page := htmlPage{
		PathInfo: currentPageInfo,
		//goldsVersion: goldsVersion,
		translation: translation,
		isHTML:      isHTMLPage(currentPageInfo.resType),
	}
	//page.Grow(4 * 1024 * 1024)

	page.htmlEscapeWriter = util.NewHTMLEscapeWriter(&page)

	if page.isHTML {
		fmt.Fprintf(&page, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<link href="%s" rel="stylesheet">
<script src="%s"></script>
<body onload="onPageLoad()"><div>
`,
			title,
			buildPageHref(currentPageInfo, createPagePathInfo(ResTypeCSS, addVersionToFilename(theme.Name(), goldsVersion)), nil, ""),
			buildPageHref(currentPageInfo, createPagePathInfo(ResTypeJS, addVersionToFilename("golds", goldsVersion)), nil, ""),
		)
	}

	return &page
}

// ToDo: w is not used now. It will be used if the page cache feature is remvoed later.s
func (page *htmlPage) Done(w io.Writer) []byte {
	if page.isHTML {
		if footerShowingManner == FooterShowingManner_none {
			//} else if genDocsMode && footerHTML != "" {
			//	page.WriteString(footerHTML)
		} else {
			var footer string
			page.WriteString(`<pre id="footer">`)
			page.WriteByte('\n')
			if footerShowingManner == FooterShowingManner_simple {
				footer = page.translation.Text_GeneratedPageFooterSimple(goldsVersion, build.Default.GOOS, build.Default.GOARCH)
			} else { // FooterShowingManner_verbose, FooterShowingManner_verbose_and_qrcode
				var qrImgLink string
				if footerShowingManner == FooterShowingManner_verbose_and_qrcode {
					switch page.translation.(type) {
					case *translations.Chinese:
						qrImgLink = buildPageHref(page.PathInfo, createPagePathInfo(ResTypePNG, "go101-wechat"), nil, "")
					case *translations.English:
						qrImgLink = buildPageHref(page.PathInfo, createPagePathInfo(ResTypePNG, "go101-twitter"), nil, "")
					}
				}
				footer = page.translation.Text_GeneratedPageFooter(goldsVersion, qrImgLink, build.Default.GOOS, build.Default.GOARCH)
			}
			page.WriteString(footer)
			page.WriteString(`</pre>`)

			//	if genDocsMode {
			//		footerHTML = `<pre id="footer">` + "\n" + footer + `</pre>`
			//	}
		}
	}

	//return append([]byte(nil), page.Bytes()...)
	var data []byte
	if genDocsMode {
		w.(*docGenResponseWriter).content = page.content
		page.content = nil
		// The GenDocs function is in charge of collect page.content.
	} else {
		// w is a standard ResponseWriter.
		data = make([]byte, 0, page.content.DataLength())
		for _, bs := range page.content {
			//w.Write(bs)
			data = append(data, bs...)
		}
		contentPool.collect(page.content)
	}

	return data
}

func (page *htmlPage) Write(data []byte) (int, error) {
	dataLen := len(data)
	if dataLen != 0 {
		var bs []byte
		if page.content == nil {
			bs = contentPool.apply()[:0]
			page.content = [][]byte{bs, nil, nil}[:1]
		} else {
			bs = page.content[len(page.content)-1]
		}
		for len(data) > 0 {
			if len(bs) == cap(bs) {
				page.content[len(page.content)-1] = bs
				bs = contentPool.apply()[:0]
				page.content = append(page.content, bs)
			}

			n := copy(bs[len(bs):cap(bs)], data)
			bs = bs[:len(bs)+n]
			data = data[n:]
		}
		page.content[len(page.content)-1] = bs
	}

	return dataLen, nil
}

func (page *htmlPage) WriteString(data string) (int, error) {
	dataLen := len(data)
	if dataLen != 0 {
		var bs []byte
		if page.content == nil {
			bs = contentPool.apply()[:0]
			page.content = [][]byte{bs, nil, nil}[:1]
		} else {
			bs = page.content[len(page.content)-1]
		}
		for len(data) > 0 {
			if len(bs) == cap(bs) {
				page.content[len(page.content)-1] = bs
				bs = contentPool.apply()[:0]
				page.content = append(page.content, bs)
			}

			n := copy(bs[len(bs):cap(bs)], data)
			bs = bs[:len(bs)+n]
			data = data[n:]
		}
		page.content[len(page.content)-1] = bs
	}

	return dataLen, nil
}

func (page *htmlPage) WriteByte(c byte) error {
	var bs []byte
	if page.content == nil {
		bs = contentPool.apply()[:0]
		page.content = [][]byte{bs, nil, nil}[:1]
	} else {
		bs = page.content[len(page.content)-1]
	}
	if len(bs) == cap(bs) {
		bs = contentPool.apply()[:0]
		page.content = append(page.content, bs)
	}
	n := len(bs)
	bs = bs[:n+1]
	bs[n] = c
	page.content[len(page.content)-1] = bs
	return nil
}

func (page *htmlPage) AsHTMLEscapeWriter() *util.HTMLEscapeWriter {
	return page.htmlEscapeWriter
}

//func (page *htmlPage) writePageLink(writeHref func(), linkText string, fragments ...string) {
//	if linkText != "" {
//		page.WriteString(`<a href="`)
//	}
//	writeHref()
//	if len(fragments) > 0 {
//		page.WriteByte('#')
//		for _, fm := range fragments {
//			page.WriteString(fm)
//		}
//	}
//	if linkText != "" {
//		page.WriteString(`">`)
//		page.WriteString(linkText)
//		page.WriteString(`</a>`)
//	}
//}

type pagePathInfo struct {
	resType pageResType
	resPath string
}

func createPagePathInfo(resType pageResType, resPath string) pagePathInfo {
	return pagePathInfo{resType, resPath}
}

// scope should be an import path.
func createPagePathInfo1(resType pageResType, scope string) pagePathInfo {
	if genDocsMode {
		scope = hashedScope(scope)
	}

	return pagePathInfo{resType, scope}
}

// scope should be an import path.
func createPagePathInfo2(resType pageResType, scope, sep, resPath string) pagePathInfo {
	if genDocsMode {
		scope = hashedScope(scope)
		resPath = hashedIdentifier(resPath)
	}

	return pagePathInfo{resType, scope + sep + resPath}
}

// scope should be an import path.
func createPagePathInfo2b(resType pageResType, scope, sep, resPath string) pagePathInfo {
	if genDocsMode {
		scope = hashedScope(scope)
		resPath = hashedFilename(resPath)
	}

	return pagePathInfo{resType, scope + sep + resPath}
}

// scope should be an import path.
func createPagePathInfo3(resType pageResType, scope, sep, resPath, selector string) pagePathInfo {
	if genDocsMode {
		scope = hashedScope(scope)
		resPath = hashedIdentifier(resPath)
		selector = hashedIdentifier(selector)
	}

	return pagePathInfo{resType, scope + sep + resPath + "." + selector}
}

type writer interface {
	Write([]byte) (int, error)
	WriteString(string) (int, error)
	WriteByte(byte) error
}

func _() {
	var _ writer = &htmlPage{}
	var _ writer = &lengthCounter{}
	var _ writer = &strings.Builder{}
}

type lengthCounter struct {
	n int
}

func (c *lengthCounter) Write(s []byte) (int, error) {
	c.n += len(s)
	return len(s), nil
}
func (c *lengthCounter) WriteString(s string) (int, error) {
	c.n += len(s)
	return len(s), nil
}
func (c *lengthCounter) WriteByte(byte) error {
	c.n++
	return nil
}

// writeHref should also write into w.
func writePageLink(writeHref func(), w writer, linkText string, fragments ...string) {
	if linkText != "" {
		w.WriteString(`<a href="`)
	}
	writeHref()
	if len(fragments) > 0 {
		w.WriteByte('#')
		for _, fm := range fragments {
			w.WriteString(fm)
		}
	}
	if linkText != "" {
		w.WriteString(`">`)
		w.WriteString(linkText)
		w.WriteString(`</a>`)
	}
}

func buildString(f func(w writer)) string {
	var lc lengthCounter
	f(&lc)
	var b strings.Builder
	b.Grow(lc.n)
	f(&b)
	return b.String()
}

//========================================
// Content is used to save memory allocations
//========================================

const Size = 1024 * 1024

type Content [][]byte // all []byte with capacity Size
type ContentPool struct {
	frees         Content
	mu            sync.Mutex
	numByteSlices int
}

var contentPool ContentPool

func (c Content) DataLength() int {
	if len(c) == 0 {
		return 0
	}
	return (len(c)-1)*Size + len(c[(len(c)-1)])
}

func (pool *ContentPool) apply() []byte {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if n := len(pool.frees); n == 0 {
		pool.numByteSlices++
		return make([]byte, Size)
	} else {
		n--
		bs := pool.frees[n]
		pool.frees = pool.frees[:n]
		return bs[:cap(bs)]
	}
}

func (pool *ContentPool) collect(c Content) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.frees == nil {
		pool.frees = make(Content, 0, 32)
	}
	pool.frees = append(pool.frees, c...)
}

//func readAll func(r io.Reader) (c Content, e error) {
//	for done := false; !done; {
//		bs, off := apply(), 0
//		for {
//			n, err := r.Read(bs[off:])
//			if err != nil {
//				if errors.Is(err, io.EOF) {
//					done = true
//				} else {
//					e = err
//					return
//				}
//			}
//			off += n
//			if done || off == cap(bs) {
//				c = append(c, bs[:off])
//				break
//			}
//		}
//	}
//	return
//}
//
//func writeFile func(path string, c Content) error {
//	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
//	if err != nil {
//		return err
//	}
//	defer func() {
//		//release(c) // should not put here.
//		err = f.Close()
//	}()
//
//	for _, bs := range c {
//		_, err := f.Write(bs)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//fakeServer := httptest.NewServer(http.HandlerFunc(ds.ServeHTTP))
//defer fakeServer.Close()
//
//buildPageContentFromFakeServer := func(path string) (Content, error) {
//	res, err := http.Get(fakeServer.URL + path)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if res.StatusCode != http.StatusOK {
//		log.Fatalf("Visit %s, get non-ok status code: %d", path, res.StatusCode)
//	}
//
//	var content Content
//	if useCustomReadAllWriteFile {
//		content, err = readAll(res.Body)
//	} else {
//		var data []byte
//		data, err = ioutil.ReadAll(res.Body)
//		if err != nil {
//			content = append(content, data)
//		}
//	}
//	res.Body.Close()
//	return content, err
//}

type docGenResponseWriter struct {
	statusCode int
	header     http.Header
	content    Content
}

func (dw *docGenResponseWriter) reset() {
	dw.statusCode = http.StatusOK
	dw.content = nil
	for k := range dw.header {
		delete(dw.header, k)
	}
}

func (dw *docGenResponseWriter) Header() http.Header {
	if dw.header == nil {
		dw.header = make(http.Header, 3)
	}
	return dw.header
}

func (dw *docGenResponseWriter) WriteHeader(statusCode int) {
	dw.statusCode = statusCode
}

func (dw *docGenResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

//header := make(http.Header, 3)
//makeHeader := func() http.Header {
//	for k := range header {
//		header[k] = nil
//	}
//	return header
//}
//var responseWriter *docGenResponseWriter
//newResponseWriter := func(writeData func([]byte) (int, error)) *docGenResponseWriter {
//	if responseWriter == nil {
//		responseWriter = &docGenResponseWriter{}
//	}
//	responseWriter.statusCode = http.StatusOK
//	responseWriter.header = makeHeader()
//	responseWriter.writeData = writeData
//	return responseWriter
//}
//var fakeRequest *http.Request
//newRequest := func(path string) *http.Request {
//	if fakeRequest == nil {
//		req, err := http.NewRequest(http.MethodGet, "http://locahost", nil)
//		if err != nil {
//			log.Fatalln("Construct fake request error:", err)
//		}
//		fakeRequest = req
//	}
//	fakeRequest.URL.Path = path
//	return fakeRequest
//}
//buildPageContentUsingCustomWriter := func(path string) (Content, error) {
//	var content Content
//	var buf, off = apply(), 0
//	writeData := func(data []byte) (int, error) {
//		dataLen := len(data)
//		for len(data) > 0 {
//			if off == cap(buf) {
//				content = append(content, buf)
//				buf, off = apply(), 0
//			}
//			n := copy(buf[off:], data)
//			//log.Println("222", len(data), n, off, cap(buf))
//			data = data[n:]
//			off += n
//		}
//		return dataLen, nil
//	}
//
//	w := newResponseWriter(writeData)
//	r := newRequest(path)
//	ds.ServeHTTP(w, r)
//	buf = buf[:off]
//	content = append(content, buf)
//
//	if w.statusCode != http.StatusOK {
//		log.Fatalf("Build %s, get non-ok status code: %d", path, w.statusCode)
//	}
//
//	return content, nil
//}
