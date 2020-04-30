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
	"strconv"
	"strings"
	"sync"

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
<div id="gen-footer">
(Generated with <a href="https://github.com/go101/gold/">Gold</a>)
</div>
`)
}

func init() {
	//enabledHtmlGenerationMod()
}

func enabledHtmlGenerationMod() {
	genMode = true
	pageHrefList = list.New()
	resHrefs = make(map[pageResType]map[string]int, 8)

}

var (
	genMode        bool
	pageHrefList   *list.List // elements are *string
	resHrefs       map[pageResType]map[string]int
	pageHrefsMutex sync.Mutex // in fact, for the current implementation, the lock is not essential
)

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
func resHrefID(resType pageResType, resName string) (int, bool) {
	pageHrefsMutex.Lock()
	defer pageHrefsMutex.Unlock()
	hrefs := resHrefs[resType]
	if hrefs == nil {
		hrefs = make(map[string]int, 1024*10)
		resHrefs[resType] = hrefs
	}
	id, ok := hrefs[resName]
	if !ok {
		id = len(hrefs)
		hrefs[resName] = id
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

type resPathInfo struct {
	resType    pageResType
	resPath    string
	subResType pageResType
	subResPath string
}

func (curRes *resPathInfo) buildPageHref(linkedRes resPathInfo, fragments ...string) string {
	// ToDo
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
// Otherwise, build the href as a string and return it (only the href part).
// inRootPage is for generation mode only. inRootPage==false means in "pages/xxx" pages.
// Note: fragments is only meaningful when page != nil.
//
// ToDo: improve the design.
func buildPageHref(resType pageResType, resName string, inRootPages bool, linkText string, page *htmlPage, fragments ...string) (r string) {

	writePageLink := func(writeHref func()) {
		if linkText != "" {
			page.WriteString(`<a href="`)
		}
		writeHref()
		if len(fragments) > 0 {
			page.WriteByte('#')
			for _, fm := range fragments {
				page.WriteString(fm)
			}
		}
		if linkText != "" {
			page.WriteString(`">`)
			page.WriteString(linkText)
			page.WriteString(`</a>`)
		}
	}

	if genMode {
		var needRegisterHref = false
		var id int

		if resType == "" { // homepages
			if resName != "" {
				panic("should not now")
			}
			_, needRegisterHref = resHrefID("", resName)
			if page != nil {
				writePageLink(func() {
					if !inRootPages {
						page.WriteString("index.html")
					} else {
						page.WriteString("../index.html")
					}
				})
			} else {
				if inRootPages {
					r = "index" + resType2ExtTable[resType]
				} else {
					r = "../index" + resType2ExtTable[resType]
				}
			}
		} else { // resource pages
			id, needRegisterHref = resHrefID(resType, resName)
			if page != nil {
				writePageLink(func() {
					if inRootPages {
						page.WriteString("pages/")
					}
					page.WriteString(string(resType))
					page.WriteByte('-')
					page.WriteString(strconv.Itoa(id))
					page.WriteString(".html")
				})
			} else {
				if inRootPages {
					r = "pages/" + string(resType) + "-" + strconv.Itoa(id) + resType2ExtTable[resType]
				} else {
					r = string(resType) + "-" + strconv.Itoa(id) + resType2ExtTable[resType]
				}
			}
		}

		if needRegisterHref {
			var hrefNotForGenerating, filePath string
			if resType == "" {
				hrefNotForGenerating = "/" + resName
				filePath = "index" + resType2ExtTable[resType]
			} else {
				hrefNotForGenerating = "/" + string(resType) + ":" + resName
				filePath = "pages/" + string(resType) + "-" + strconv.Itoa(id) + resType2ExtTable[resType]
			}
			registerPageHref(genPageInfo{
				HrefPath: hrefNotForGenerating,
				FilePath: filePath,
			})
		}
	} else {
		if resType == "" {
			if page != nil {
				writePageLink(func() {
					page.WriteByte('/')
					page.WriteString(resName)
				})
			} else {
				r = "/" + resName
			}
		} else {
			if page != nil {
				writePageLink(func() {
					page.WriteByte('/')
					page.WriteString(string(resType))
					page.WriteByte(':')
					page.WriteString(resName)
				})
			} else {
				r = "/" + string(resType) + ":" + resName
			}
		}
	}

	return
}

func Gen(outputDir string, args []string, printUsage func(io.Writer)) {
	log.SetFlags(log.Lshortfile)

	// ...

	enabledHtmlGenerationMod()

	outputDir = strings.TrimRight(outputDir, "\\/")
	outputDir = strings.Replace(outputDir, "/", string(filepath.Separator), -1)
	outputDir = strings.Replace(outputDir, "\\", string(filepath.Separator), -1)

	// ...
	ds := &docServer{
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}
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

	buildPageHref("", "", true, "", nil) // the overview page

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
