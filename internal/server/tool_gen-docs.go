package server

import (
	"container/list"
	"fmt"
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
	"time"

	"go101.org/golds/code"
)

var _ = runtime.GC

func init() {
	//enabledHtmlGenerationMod() // debug
}

var (
	pageHrefList   *list.List // elements are *string
	resHrefs       map[pageResType]map[string]int
	pageHrefsMutex sync.Mutex // in fact, for the current implementation, the lock is not essential
)

func enabledHtmlGenerationMod() {
	genDocsMode = true
	pageHrefList = list.New()
	resHrefs = make(map[pageResType]map[string]int, 8)
}

//func disabledHtmlGenerationMod() {
//	genDocsMode = false
//	pageHrefList = nil
//	resHrefs = nil
//}

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
//buildPageHref(v.currentPathInfo, pagePathInfo{ResTypePackage, "builtin"}, nil, "")+"#name-"+obj.Name()
//=>
//buildPageHref(v.currentPathInfo, pagePathInfo{ResTypePackage, "builtin"}, nil, "#", name-"+obj.Name())
//
//buildPageHref(v.currentPathInfo, pagePathInfo{ResTypePackage, "builtin"}, nil, "?", "key=value", "#")

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
// Otherwise, build the href as a string and return it (only the href part).
// inRootPage is for generation mode only. inRootPage==false means in "pages/xxx" pages.
// Note: fragments is only meaningful when page != nil.
//
// ToDo: improve the design.
func buildPageHref(currentPageInfo, linkedPageInfo pagePathInfo, page *htmlPage, linkText string, fragments ...string) (r string) {
	if genDocsMode {
		goto Generate
	}

	if linkedPageInfo.resType == ResTypeNone {
		if page != nil {
			page.writePageLink(func() {
				page.WriteByte('/')
				page.WriteString(linkedPageInfo.resPath)
			}, linkText, fragments...)
		} else {
			r = "/" + linkedPageInfo.resPath
		}
	} else {
		if page != nil {
			page.writePageLink(func() {
				page.WriteByte('/')
				page.WriteString(string(linkedPageInfo.resType))
				page.WriteByte(':')
				page.WriteString(linkedPageInfo.resPath)
			}, linkText, fragments...)
		} else {
			r = "/" + string(linkedPageInfo.resType) + ":" + linkedPageInfo.resPath
		}
	}

	return

Generate:

	if !buildIdUsesPages && linkedPageInfo.resType == ResTypeReference {
		panic("identifer-uses page (" + linkedPageInfo.resPath + ") should not be build")
	}
	if !enableSoruceNavigation && linkedPageInfo.resType == ResTypeImplementation {
		panic("method-implementation page (" + linkedPageInfo.resPath + ") should not be build")
	}

	var makeHref = func(pathInfo pagePathInfo) string {
		switch pathInfo.resType {
		case ResTypeNone: // top-level pages
			switch pathInfo.resPath {
			case "":
				return "index" + resType2ExtTable(pathInfo.resType)
			default:
				return pathInfo.resPath + resType2ExtTable(pathInfo.resType)
			}
		case ResTypeReference:
			//pathInfo.resPath = strings.ReplaceAll(pathInfo.resPath, "..", "/") // no need to convert
		}

		return string(pathInfo.resType) + "/" + pathInfo.resPath + resType2ExtTable(pathInfo.resType)
	}

	var _, needRegisterHref = resHrefID(linkedPageInfo.resType, linkedPageInfo.resPath)
	var currentHref = makeHref(currentPageInfo)
	var generatedHref = makeHref(linkedPageInfo)
	var relativeHref = RelativePath(currentHref, generatedHref)

	if page != nil {
		page.writePageLink(func() {
			page.WriteString(relativeHref)
		}, linkText, fragments...)
	} else {
		r = relativeHref
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
	enabledHtmlGenerationMod()

	forTesting := outputDir == ""
	silent := silentMode || forTesting
	if increaseGCFrequency {
		debug.SetGCPercent(75)
	}

	setPageOutputOptions(options, forTesting)

	// ...
	ds := &docServer{
		//goldsVersion: options.GoldsVersion,
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}
	ds.initSettings(options.PreferredLang)
	ds.analyze(args, printUsage)

	// ...
	genOutputDir := outputDir
	if genOutputDir == "." {
		genOutputDir = ds.workingDirectory
	}
	defer os.Chdir(ds.workingDirectory)
	genOutputDir = filepath.Join(genOutputDir, "generated-"+time.Now().Format("20060102150405"))

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

	writeFile := func(path string, c Content) error {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer func() {
			//release(c) // should not put here.
			err = f.Close()
		}()

		for _, bs := range c {
			_, err := f.Write(bs)
			if err != nil {
				return err
			}
		}

		return nil
	}

	type Page struct {
		FilePath string
		//Content  []byte
		Content Content
	}

	var pages = make(chan Page, 8)

	buildPageHref(pagePathInfo{ResTypeNone, ""}, pagePathInfo{ResTypeNone, ""}, nil, "") // the overview page

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

			numPages++
			numBytes += len(pg.Content)

			path := filepath.Join(genOutputDir, pg.FilePath)
			path = strings.Replace(path, "/", string(filepath.Separator), -1)
			path = strings.Replace(path, "\\", string(filepath.Separator), -1)

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				log.Fatalln("Mkdir error:", err)
			}

			//if err := ioutil.WriteFile(path, pg.Content, 0644); err != nil {
			//	log.Fatalln("Write file error:", err)
			//}
			if err := writeFile(path, pg.Content); err != nil {
				log.Fatalln("Write file error:", err)
			}

			if !silent {
				log.Printf("Generated %s (size: %d).", pg.FilePath, pg.Content.DataLength())
			}
		}(pg)
	}

	if forTesting {
		return
	}

	log.Printf("Done (%d pages are generated and %d bytes are written).", numPages, numBytes)
	log.Printf("Docs are generated in %s.", outputDir) // genOutputDir)
	log.Println("Run the following command to view the docs:")
	log.Printf("\t%s", viewDocsCommand(outputDir)) // genOutputDir))
}
