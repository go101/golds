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

func init() {
	//enabledHtmlGenerationMod() // debug
}

var (
	genDocsMode    bool
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

func GenDocs(outputDir string, args []string, goldVersion string, silent bool, printUsage func(io.Writer), viewDocsCommand func(string) string) {
	enabledHtmlGenerationMod()
	//

	ds := &docServer{
		goldVersion: goldVersion,
		phase:       Phase_Unprepared,
		analyzer:    &code.CodeAnalyzer{},
	}
	ds.changeSettings("", "")
	ds.analyze(args, printUsage)

	// ...
	outputDir = filepath.Join(outputDir, "generated-"+time.Now().Format("20060102150405"))

	fakeServer := httptest.NewServer(http.HandlerFunc(ds.ServeHTTP))
	defer fakeServer.Close()

	// ...
	type Page struct {
		FilePath string
		Content  []byte
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

			res, err := http.Get(fakeServer.URL + info.HrefPath)
			if err != nil {
				log.Fatal(err)
			}

			if res.StatusCode != http.StatusOK {
				log.Fatalf("visit %s, get non-ok status code: %d", info.HrefPath, res.StatusCode)
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

		if !silent {
			log.Printf("Generated %s (size: %d).", pg.FilePath, len(pg.Content))
		}
	}

	log.Printf("Done (%d pages are generated and %d bytes are written).", numPages, numBytes)
	//outputDir, _ = filepath.Abs(outputDir)
	log.Printf("Docs are generated in %s.", outputDir)
	log.Println("Run the following command to view the docs:")
	log.Printf("\t%s", viewDocsCommand(outputDir))
}
