package server

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"

	"go101.org/gold/code"
)

var (
	// These variables are not guarded by mutex.
	// They must be set at initialization phase.
	collectPagePaths = false
	useRelativePaths = false // true for not browsing from browsers

	pagePaths chan string
)

func registerPagePath(path string) string {
	if collectPagePaths {
		pagePaths <- path
	}
}

func Gen(port string, args []string, printUsage func(io.Writer)) {
	collectPagePaths = true
	pagePaths = make(chan string, 1024)

	ds := &docServer{
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}
	ds.changeSettings("", "")
	ds.analyze(args, printUsage)

	fakeServer := httptest.NewServer(http.HandlerFunc(ds.ServeHTTP))
	defer fakeServer.Close()
	//log.Println(fakeServer.URL)
	root := fakeServer.URL

	type Page struct {
		Path    string
		Content []byte
	}
	pages := make(chan Page, 32)
	registerPagePath("/")
	const NumClients = 5
	idleClients := make(chan struct{}, NumClients)

	for range [NumClients]struct{}{} {
		go func() {
			for {
				select {
				default:
					idleClients <- struct{}{}
					if len(idleClients) == cap(idleClients) {
						return
					}

				case path := <-pagePaths:
					res, err := http.Get(root + path)
					if err != nil {
						log.Fatal(err)
					}

					content, err := ioutil.ReadAll(res.Body)
					res.Body.Close()
					if err != nil {
						log.Fatal(err)
					}

					pages <- Page{
						Path:    path,
						Content: content,
					}
				}
			}
		}()
	}

	for pg := range pages {
		_ = pg
	}
}
