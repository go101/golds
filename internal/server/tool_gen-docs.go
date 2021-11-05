package server

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/token"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

var _ = runtime.GC

func init() {
	//enabledHtmlGenerationMod() // debug
}

var (
	pageHrefList *list.List // elements are *string
	resHrefs     map[pageResType]map[string]int
	pageHrefs    map[pagePathInfo]string

	hashedScopes map[string]string
	hashedTokens map[string]string

	pageHrefsMutex sync.Mutex // in fact, for the current implementation, the lock is not essential
)

func enabledHtmlGenerationMod() {
	genDocsMode = true
	enabledPageCache = false

	pageHrefList = list.New()
	resHrefs = make(map[pageResType]map[string]int, 16)
	pageHrefs = make(map[pagePathInfo]string, 65536)

	hashedScopes = make(map[string]string, 128)
	hashedTokens = make(map[string]string, 128*1024)
}

//func disabledHtmlGenerationMod() {
//	genDocsMode = false
//	pageHrefList = nil
//	resHrefs = nil
//}

func hashedScope(scope string) string {
	if strings.ToLower(scope) == scope {
		return scope
	}

	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()

	if hs := hashedScopes[scope]; hs != "" {
		return hs
	}

	var b strings.Builder
	b.Grow(len(scope))
	tokens := strings.Split(scope, "/")
	for i, t := range tokens {
		tokens[i] = hashedFilename(t)
	}
	hs := strings.Join(tokens, "/")
	hashedScopes[scope] = hs
	return hs
}

func deHashScope(scope string) string {
	if strings.ToLower(scope) == scope {
		return scope
	}

	tokens := strings.Split(scope, "/")
	for i, t := range tokens {
		tokens[i] = deHashFilename(t)
	}
	return strings.Join(tokens, "/")
}

func hashedFilename(t string) string {
	if strings.ToLower(t) == t {
		return t
	}

	return t + "^" + string(hashHexHead([]byte(t)))
}

func deHashFilename(t string) string {
	for i := len(t) - 1; i >= 0; i-- {
		if t[i] == '^' {
			return t[:i]
		}
		if '0' <= t[i] && t[i] <= '9' {
			continue
		}
		if 'a' <= t[i] && t[i] <= 'f' {
			continue
		}
		break
	}
	return t
}

func hashedIdentifier(t string) string {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()

	if token.IsExported(t) {
		lt := strings.ToLower(t)
		if et := hashedTokens[lt]; et == "" {
			hashedTokens[lt] = t
			return t
		} else if et == t {
			return t
		}
	}

	return t + "^" + string(hashHexHead([]byte(t)))
}

func deHashIdentifier(t string) string {
	if i := strings.LastIndexByte(t, '^'); i >= 0 {
		return t[:i]
	}
	return t
}

func hashHexHead(data []byte) []byte {
	sum := sha256.Sum256([]byte(data)) // [32]byte
	hexes := sum[10:30]
	hex.Encode(hexes, sum[:10])

	const abbrLen = 5 // max is 20, recommended min is 4
	return hexes[:abbrLen]
}

type genPageInfo struct {
	HrefPath string
	FilePath string
}

func registerPageHref(info genPageInfo) {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	pageHrefList.PushBack(&info)
}

func nextPageToLoad() (info *genPageInfo) {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	if front := pageHrefList.Front(); front != nil {
		info = front.Value.(*genPageInfo)
		pageHrefList.Remove(front)
	}
	return
}

func cachePageHref(pathInfo pagePathInfo, href string) {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	pageHrefs[pathInfo] = href
}

func cachedPageHref(pathInfo pagePathInfo) string {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	return pageHrefs[pathInfo]
}

// Return the id and whether or not the id is just registered.
func resHrefID(resType pageResType, resPath string) (int, bool) {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	hrefs := resHrefs[resType]
	if hrefs == nil {
		hrefs = make(map[string]int, 1024*10)
		resHrefs[resType] = hrefs
	}
	id, ok := hrefs[resPath]
	if !ok {
		id = len(hrefs)
		hrefs[resPath] = id
	}
	return id, !ok
}

var _resType2ExtTable = map[pageResType]string{
	ResTypeAPI: ".json",
	ResTypeCSS: ".css",
	ResTypeJS:  ".js",
	ResTypeSVG: ".svg",
	ResTypePNG: ".png",
	//ResTypeAPI
}

func resType2ExtTable(res pageResType) string {
	ext, ok := _resType2ExtTable[res]
	if testingMode && isHTMLPage(res) == ok {
		panic("isHTMLPage not match: " + res)
	}
	if !ok {
		ext = ".html"
	}
	return ext
}

var dotdotslashes = strings.Repeat("../", 256)

func DotDotSlashes(count int) string {
	if count > 256 {
		return "" // panic is better?
	}
	return dotdotslashes[:count*3]
}

func RelativePath(a, b string) string {
	var c = FindPackageCommonPrefixPaths(a, b)
	if len(c) > len(a) {
		if len(c) != len(a)+1 {
			panic("what a?")
		}
		if c[len(a)] != '/' {
			panic("what a?!")
		}
	} else {
		a = a[len(c):]
	}
	n := strings.Count(a, "/")
	if len(c) > len(b) {
		if len(c) != len(b)+1 {
			panic("what b?")
		}
		if c[len(b)] != '/' {
			panic("what b?!")
		}
		return DotDotSlashes(n)
	}
	return DotDotSlashes(n) + b[len(c):]
}

// Return "" for invalid.
// Assume the digits of major/minor/patch are all from 0 to 9.
func PreviousVersion(version string) string {
	vs := strings.SplitN(version, ".", 3)
	if len(vs) < 3 {
		return ""
	}
	if i := strings.Index(vs[2], "-"); i >= 0 {
		vs[2] = vs[2][:i]
	}
	patch, err := strconv.Atoi(vs[2])
	if err != nil {
		return ""
	}
	if patch > 0 {
		vs[2] = strconv.Itoa(patch - 1)
		return strings.Join(vs, ".")
	}
	minor, err := strconv.Atoi(vs[1])
	if err != nil {
		return ""
	}
	vs[2] = "9"
	if minor > 0 {
		vs[1] = strconv.Itoa(minor - 1)
		return strings.Join(vs, ".")
	}
	prefix := ""
	for i := len(vs[0]) - 1; i >= 0; i-- {
		if vs[0][i] < '0' || vs[0][i] > '9' {
			prefix, vs[0] = vs[0][:i+1], vs[0][i+1:]
			break
		}
	}
	major, err := strconv.Atoi(vs[0])
	if err != nil {
		return ""
	}
	vs[1] = "9"
	if major > 0 {
		vs[0] = strconv.Itoa(major - 1)
		return prefix + vs[0] + "." + vs[1] + "." + vs[2]
	}
	return ""
}

// ToDo:
// path prefixes should be removed from srouce file paths.
// * project root path
// * module cache root path
// * GOPATH src roots
// * GOROOT src root
// This results handledPath.
//
// src:handledPath will be hashed as the generated path, or not.

// If page is not nil, write the href directly into page (write the full <a...</a> if linkText is not blank).
// The fragments arguments don't include "#".
//
// ToDo: improve the design.
func buildPageHref(currentPageInfo, linkedPageInfo pagePathInfo, page *htmlPage, linkText string, fragments ...string) (r string) {
	if linkedPageInfo.resType == ResTypeSource && sourceReadingStyle == SourceReadingStyle_external {
		if writeExternalSourceCodeLink == nil {
			panic("writeExternalSourceCodeLink == nil")
		}

		var docFragment []string
		if len(fragments) > 1 && fragments[0] == "doc" {
			docFragment = fragments[:1]
			fragments = fragments[1:]
		}

		var line, endLine string
		if n := len(fragments); n > 0 {
			if fragments[0] != "line-" {
				panic("unexpected fragments[0]: " + fragments[0])
			}
			switch {
			case n > 4:
				log.Println("warning: fragments[4:] are ignored: " + strings.Join(fragments[4:], ", "))
				fallthrough
			case n == 4:
				if fragments[2] != ":" {
					panic("unexpected fragments[2]: " + fragments[2])
				}
				line = fragments[1]
				endLine = fragments[3]
			case n == 3:
				log.Println("warning: fragments[2] is ignored: " + fragments[2])
				fallthrough
			case n == 2:
				line = fragments[1]
			case n == 1:
				panic("too few fragments: " + strconv.Itoa(n))
			}
		}

		var err error
		var handled bool
		writeHref := func(w writer) {
			handled, err = writeExternalSourceCodeLink(w, deHashScope(linkedPageInfo.resPath), line, endLine)
		}
		link := buildString(writeHref)
		if err != nil {
			panic("writeExternalSourceCodeLink error: " + err.Error())
		}
		if handled {
			if page != nil {
				writePageLink(func() {
					page.WriteString(link)
				}, page, linkText)
			} else {
				r = link
			}
			return
		}

		// Use default way.
		if docFragment != nil {
			fragments = docFragment
		}
	}

	if genDocsMode {
		goto Generate
	}

	{
		writeLink := func(w writer) {
			if linkedPageInfo.resType == ResTypeNone {
				writePageLink(func() {
					w.WriteByte('/')
					w.WriteString(linkedPageInfo.resPath)
				}, w, linkText, fragments...)
			} else {
				writePageLink(func() {
					w.WriteByte('/')
					w.WriteString(string(linkedPageInfo.resType))
					w.WriteByte(':')
					w.WriteString(linkedPageInfo.resPath)
				}, w, linkText, fragments...)
			}
		}

		if page != nil {
			writeLink(page)
		} else {
			r = buildString(writeLink)
		}
	}

	return

Generate:

	if !buildIdUsesPages && linkedPageInfo.resType == ResTypeReference {
		panic("identifer-uses page (" + linkedPageInfo.resPath + ") should not be build")
	}
	//if !enableSoruceNavigation && linkedPageInfo.resType == ResTypeImplementation {
	//	panic("method-implementation page (" + linkedPageInfo.resPath + ") should not be build")
	//}
	if sourceReadingStyle != SourceReadingStyle_rich && linkedPageInfo.resType == ResTypeImplementation {
		panic("method-implementation page (" + linkedPageInfo.resPath + ") should not be build")
	}

	var makeHref = func(pathInfo pagePathInfo) (href string) {
		href = cachedPageHref(pathInfo)
		if href != "" {
			return
		}

		if pathInfo.resType == ResTypeNone {
			if pathInfo.resPath == "" {
				href = "index" + resType2ExtTable(pathInfo.resType)
			} else {
				href = pathInfo.resPath + resType2ExtTable(pathInfo.resType)
			}
		} else {
			href = string(pathInfo.resType) + "/" + pathInfo.resPath + resType2ExtTable(pathInfo.resType)
		}
		cachePageHref(pathInfo, href)
		return
	}

	var _, needRegisterHref = resHrefID(linkedPageInfo.resType, linkedPageInfo.resPath)
	var currentHref = makeHref(currentPageInfo)
	var generatedHref = makeHref(linkedPageInfo)
	var relativeHref = RelativePath(currentHref, generatedHref)

	writeLink := func(w writer) {
		writePageLink(func() {
			w.WriteString(relativeHref)
		}, w, linkText, fragments...)
	}

	if page != nil {
		writeLink(page)
	} else {
		r = buildString(writeLink)
	}

	if needRegisterHref {
		var hrefNotForGenerating string
		if linkedPageInfo.resType == ResTypeNone {
			hrefNotForGenerating = "/" + linkedPageInfo.resPath
		} else {
			hrefNotForGenerating = "/" + string(linkedPageInfo.resType) + ":" + linkedPageInfo.resPath
		}

		registerPageHref(genPageInfo{
			HrefPath: hrefNotForGenerating,
			FilePath: generatedHref,
		})

		if ext := filepath.Ext(generatedHref); ext != ".html" {
			//dir, file := filepath.Split(generatedHref)
			dir, file := path.Split(generatedHref)
			if i := strings.LastIndex(file, goldsVersion); i >= 0 {
				version := goldsVersion
				for range [5]struct{}{} {
					version = PreviousVersion(version)
					if version == "" {
						break
					}

					registerPageHref(genPageInfo{
						HrefPath: hrefNotForGenerating,
						FilePath: dir + file[:i] + version + file[i+len(goldsVersion):] + ext,
					})
				}
			}
		}
	}

	return
}

func GenDocs(options PageOutputOptions, args []string, outputDir string, silentMode bool, printUsage func(io.Writer), increaseGCFrequency bool, viewDocsCommand func(string) string) {
	toolchain, err := findToolchainInfo()
	if err != nil {
		log.Fatal(err)
	}

	enabledHtmlGenerationMod()

	forTesting := outputDir == ""
	silent := silentMode || forTesting
	if increaseGCFrequency {
		debug.SetGCPercent(75)
	}
	// ...
	ds := &docServer{}
	ds.analyze(args, options, toolchain, forTesting, printUsage)

	// ...
	genOutputDir := outputDir
	if genOutputDir == "." {
		genOutputDir = ds.initialWorkingDirectory
	}
	defer os.Chdir(ds.initialWorkingDirectory)
	//genOutputDir = filepath.Join(genOutputDir, "generated-"+time.Now().Format("20060102150405"))

	// ...
	//defer func() { log.Println("============== contentPool.numByteSlices:", contentPool.numByteSlices) }() // 10 for std
	w := &docGenResponseWriter{}
	r := &http.Request{URL: &url.URL{}}
	buildPageContent := func(path string) (Content, error) {
		w.reset()
		r.URL.Path = path
		ds.ServeHTTP(w, r)
		if w.statusCode != http.StatusOK {
			contentPool.collect(w.content)
			return nil, fmt.Errorf("build %s, get non-ok status code: %d", path, w.statusCode)
		}
		return w.content, nil
	}

	// ...

	writeFile := func(path string, c Content) (n int, err error) {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return
		}
		defer func() {
			//release(c) // should not put here.
			err = f.Close()
		}()

		for _, bs := range c {
			x, err := f.Write(bs)
			n += x
			if err != nil {
				return n, err
			}
		}

		return
	}

	type Page struct {
		FilePath string
		//Content  []byte
		Content Content
	}

	var pages = make(chan Page, 8)

	buildPageHref(createPagePathInfo(ResTypeNone, ""), createPagePathInfo(ResTypeNone, ""), nil, "") // the overview page

	// page loader
	go func() {
		for {
			info := nextPageToLoad()
			if info == nil {
				break
			}

			content, err := buildPageContent(info.HrefPath)
			if err != nil {
				log.Fatalln("Read page data error:", err)
			}

			//log.Println(count, count&2048, info.FilePath)
			pages <- Page{
				FilePath: info.FilePath,
				Content:  content,
			}
		}
		close(pages)
	}()

	// page saver
	numPages, numBytes := 0, 0
	for pg := range pages {
		func(pg Page) {
			defer contentPool.collect(pg.Content)

			if forTesting {
				return
			}

			path := filepath.Join(genOutputDir, pg.FilePath)
			path = strings.Replace(path, "/", string(filepath.Separator), -1)
			path = strings.Replace(path, "\\", string(filepath.Separator), -1)

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				log.Fatalln("Mkdir error:", err)
			}

			//if err := ioutil.WriteFile(path, pg.Content, 0644); err != nil {
			//	log.Fatalln("Write file error:", err)
			//}
			if n, err := writeFile(path, pg.Content); err != nil {
				log.Fatalln("Write file error:", err)
			} else {
				numPages++
				numBytes += n
			}

			//if verboseLogs || !silent {
			if !silent {
				log.Printf("Generated %s (size: %d).", pg.FilePath, pg.Content.DataLength())
			}
		}(pg)
	}

	if forTesting {
		return
	}

	//if verboseLogs || !silent {
	if !silent {
		log.Printf("Done (%d pages are generated and %d bytes are written).", numPages, numBytes)
	}

	log.Printf("Docs are generated in %s.", outputDir) // genOutputDir)
	if sourceReadingStyle == SourceReadingStyle_external {
		for _, w := range ds.wdRepositoryWarnings {
			log.Println("!!! Warning:", w)
		}
		if len(ds.wdRepositoryWarnings) > 0 {
			log.Println()
		}
	}
	log.Println("Run the following command to view the docs:")
	log.Printf("\t%s", viewDocsCommand(outputDir)) // genOutputDir))
}
