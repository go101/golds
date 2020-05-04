package server

import (
	"container/list"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go101.org/gold/code"
)

/*
To simplify the implementation, in generation mode, page hrefs are like
	index.html
	pages/pkg-12345.html
	pages/dep-12345.html
	pages/src-123456.html
*/

func writePageGenerationInfo(page *htmlPage) {
	if !genMode {
		return
	}

	page.WriteString(`
<pre id="gen-footer">
(Generated with <a href="https://github.com/go101/gold/">Gold</a>)
</pre>
`)
}

func init() {
	//enabledHtmlGenerationMod() // debug
}

var (
	genMode        bool
	pageHrefList   *list.List // elements are *string
	resHrefs       map[pageResType]map[string]int
	pageHrefsMutex sync.Mutex // in fact, for the current implementation, the lock is not essential
)

func enabledHtmlGenerationMod() {
	genMode = true
	pageHrefList = list.New()
	resHrefs = make(map[pageResType]map[string]int, 8)

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

var resType2ExtTable = map[pageResType]string{
	"":    ".html",
	"pkg": ".html",
	"dep": ".html",
	"src": ".html",
	"mod": ".html",
	"css": ".css",
	"jvs": ".js",
}

var dotdotslashes = strings.Repeat("../", 256)

func DotDotSlashes(count int) string {
	return dotdotslashes[:count*3]
}

func RelativePath(a, b string) string {
	var c = FindPackageCommonPrefixPaths(a, b)
	a = a[len(c):]
	n := strings.Count(a, "/")
	return DotDotSlashes(n) + b[len(c):]
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
// Otherwise, build the href as a string and return it (only the href part).
// inRootPage is for generation mode only. inRootPage==false means in "pages/xxx" pages.
// Note: fragments is only meaningful when page != nil.
//
// ToDo: improve the design.
func buildPageHref(currentPageInfo, linkedPageInfo pagePathInfo, page *htmlPage, linkText string, fragments ...string) (r string) {
	if genMode {
		goto Generate
	}

	if linkedPageInfo.resType == ResTypeNone {
		if linkedPageInfo.resPath != "" {
			panic("should not now")
		}
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

	var makeHref = func(pathInfo pagePathInfo) string {
		if pathInfo.resType == ResTypeNone { // homepages
			if pathInfo.resPath != "" {
				panic("should not now")
			}
			return "index" + resType2ExtTable[pathInfo.resType]
		} else {

			return string(pathInfo.resType) + "/" + pathInfo.resPath + resType2ExtTable[pathInfo.resType]
		}
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
	}

	return
}

func Gen(outputDir string, args []string, printUsage func(io.Writer), roughBuildTime string) {
	log.SetFlags(log.Lshortfile)

	// ...

	enabledHtmlGenerationMod()

	outputDir = strings.TrimRight(outputDir, "\\/")
	outputDir = strings.Replace(outputDir, "/", string(filepath.Separator), -1)
	outputDir = strings.Replace(outputDir, "\\", string(filepath.Separator), -1)
	outputDir = filepath.Join(outputDir, "generated-"+time.Now().UTC().Format("20060102150405"))

	// ...
	ds := &docServer{
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}
	ds.parseRoughBuildTime(roughBuildTime)
	ds.changeSettings("", "")
	ds.analyze(args, printUsage)

	// ...
	fakeServer := httptest.NewServer(http.HandlerFunc(ds.ServeHTTP))
	defer fakeServer.Close()

	// ...
	type Page struct {
		FilePath string
		Content  []byte
	}

	var pages = make(chan Page, 8)

	buildPageHref(pagePathInfo{ResTypeNone, ""}, pagePathInfo{ResTypeNone, ""}, nil, "") // the overview page

	// stat orso number of pages, print about progress in writing.

	// page loader
	go func() {
		for {
			info := nextPageToLoad()
			if info == nil {
				break
			}

			res, err := http.Get(fakeServer.URL + info.HrefPath)
			if err != nil {
				log.Fatal(err)
			}

			content, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Fatal(err)
			}

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
		numPages++
		numBytes += len(pg.Content)

		path := filepath.Join(outputDir, pg.FilePath)
		path = strings.Replace(path, "/", string(filepath.Separator), -1)
		path = strings.Replace(path, "\\", string(filepath.Separator), -1)

		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			log.Fatalln("Mkdir error:", err)
		}

		if err := ioutil.WriteFile(path, pg.Content, 0644); err != nil {
			log.Fatalln("Write file error:", err)
		}

		log.Printf("Generated %s (size: %d).", pg.FilePath, len(pg.Content))
	}

	log.Printf("Done (%d pages are generated and %d bytes are written).", numPages, numBytes)
}
