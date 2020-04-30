package server

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go101.org/gold/code"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	Phase_Unprepared = iota
	Phase_Parsed
	Phase_Analyzed
)

type docServer struct {
	mutex    sync.Mutex
	phase    int
	analyzer *code.CodeAnalyzer

	// Cached pages
	// ToDo: cache body parts only.
	//       Best not to cache the header parts:
	//       * almost the same for all pages.
	//       * theme may change.
	packageListPage []byte
	packagePages    map[string][]byte
	sourcePages     map[string][]byte
	dependencyPages map[string][]byte

	//
	currentTheme       Theme
	currentTranslation Translation
}

func Run(port string, args []string, printUsage func(io.Writer)) {
	log.SetFlags(log.Lshortfile)

	ds := &docServer{
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}

	ds.changeSettings("", "")

NextTry:
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		if strings.Index(err.Error(), "bind: address already in use") >= 0 {
			port = strconv.Itoa(50000 + 1 + rand.Int()%9999)
			goto NextTry
		}
		// ToDo: random port
		log.Fatal(err)
	}

	go ds.analyze(args, printUsage)

	err = OpenBrowser(fmt.Sprintf("http://localhost:%v", port))
	if err != nil {
		log.Println(err)
	}

	log.Printf("Server started: http://localhost:%v\n", port)
	(&http.Server{
		Handler:      ds,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}).Serve(l)
}

func (ds *docServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Query strings might contain setting change parameters,
	// such as "?theme=dark&lang=fr".
	// ToDo, if query string is not blank, change settings,
	//       then redirect to the url without query string.

	var path = r.URL.Path[1:]
	if path == "" {
		ds.overviewPage(w, r)
		return
	}

	// Min valid path length is 5.
	if len(path) < 5 || path[3] != ':' {
		fmt.Fprint(w, "Invalid url")
		return
	}

	switch resType, resPath := pageResType(path[:3]), path[4:]; resType {
	default: // ResTypeNone
		w.WriteHeader(http.StatusNotFound)
	case ResTypeCSS: // "css"
		ds.cssFile(w, r, resPath)
	case ResTypeJS: // "jvs"
		ds.javascriptFile(w, r, resPath)
	//case "mod:": // module
	//	ds.modulePage(w, r, path)
	case ResTypePackage: // "pkg"
		ds.packageDetailsPage(w, r, resPath)
	case ResTypeDependency: // "dep"
		ds.packageDependenciesPage(w, r, resPath)
	case ResTypeSource: // "src"
		index := strings.LastIndex(resPath, "/")
		if index < 0 {
			ds.sourceCodePage(w, r, "", resPath)
		} else {
			ds.sourceCodePage(w, r, resPath[:index], resPath[index+1:])
		}
	}
}

func (ds *docServer) analyze(args []string, printUsage func(io.Writer)) {
	if len(args) == 0 {
		args = []string{"."}
	} else if len(args) == 1 && args[0] == "std" {
		os.Setenv("CGO_ENABLED", "0")
	}

	for _, arg := range args {
		if arg == "builtin" {
			goto Start
		}
	}

	// "builtin" package is always needed.
	// ToDo: remove this line, use a custom builtin page.
	args = append(args, "builtin")

Start:
	//if !ds.collectImports(args...) {
	//	printUsage()
	//}

	if !ds.analyzer.ParsePackages(args...) {
		printUsage(os.Stdout)
		return
	}

	{
		ds.mutex.Lock()
		ds.phase = Phase_Parsed
		ds.mutex.Unlock()
	}

	ds.analyzer.AnalyzePackages()
	ds.analyzer.CollectSourceFiles()

	{
		ds.mutex.Lock()
		ds.phase = Phase_Analyzed
		ds.packagePages = make(map[string][]byte, ds.analyzer.NumPackages())
		ds.sourcePages = make(map[string][]byte, ds.analyzer.NumSourceFiles())
		ds.dependencyPages = make(map[string][]byte, ds.analyzer.NumPackages())
		ds.mutex.Unlock()
	}
}
