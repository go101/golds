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
	"go101.org/gold/internal/util"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	Phase_Unprepared = iota
	Phase_Analyzed
)

type docServer struct {
	mutex sync.Mutex

	goldVersion string

	phase           int
	analyzer        *code.CodeAnalyzer
	analyzingLogger *log.Logger
	analyzingLogs   []LoadingLogMessage

	// Cached pages
	packageListPage []byte
	packagePages    map[string][]byte
	sourcePages     map[string][]byte
	dependencyPages map[string][]byte

	//
	currentTheme       Theme
	currentTranslation Translation

	//
	updateLogger          *log.Logger
	roughBuildTime        func() time.Time
	updateTip             int
	cachedUpdateTip       int
	newerVersionInstalled bool

	//
	generalLogger *log.Logger
}

func Run(port string, args []string, silentMode bool, goldVersion string, printUsage func(io.Writer), roughBuildTime func() time.Time) {
	ds := &docServer{
		goldVersion: goldVersion,

		phase:           Phase_Unprepared,
		analyzer:        &code.CodeAnalyzer{},
		analyzingLogger: log.New(os.Stdout, "[Analyzing] ", 0),
		analyzingLogs:   make([]LoadingLogMessage, 0, 64),

		updateLogger:   log.New(os.Stdout, "[Update] ", 0),
		roughBuildTime: roughBuildTime,
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

	go func() {
		ds.analyze(args, printUsage)
		ds.analyzingLogger.SetPrefix("")
		ds.analyzingLogger.Printf("Server started: http://localhost:%v\n", port)
	}()

	if !silentMode {
		err = util.OpenBrowser(fmt.Sprintf("http://localhost:%v", port))
		if err != nil {
			log.Println(err)
		}
	}

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

	if len(path) < 5 || path[3] != ':' {
		switch path {
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Invalid url")
		case "update":
			ds.startUpdatingGold()
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		case "statistics":
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprint(w, "Not implemented yet")
		}
		return
	}

	switch resType, resPath := pageResType(path[:3]), path[4:]; resType {
	default: // ResTypeNone
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Invalid url")
	case ResTypeAPI: // "api"
		switch resPath {
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Invalid url")
		case "update":
			ds.updateAPI(w, r)
		case "load":
			ds.loadAPI(w, r)
		}
	case ResTypeCSS: // "css"
		ds.cssFile(w, r, removeVersionFromFilename(resPath, ds.goldVersion))
	case ResTypeJS: // "jvs"
		ds.javascriptFile(w, r, removeVersionFromFilename(resPath, ds.goldVersion))
	case ResTypeSVG: // "svg"
		ds.svgFile(w, r, resPath)
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
	var stopWatch = util.NewStopWatch()
	defer func() {
		d := stopWatch.Duration(false)
		ds.registerAnalyzingLogMessage(ds.currentTranslation.Text_Analyzing_Done(d, util.MemoryUse()))
		ds.registerAnalyzingLogMessage("")
	}()

	if len(args) == 0 {
		args = []string{"."}
	} else if len(args) == 1 && args[0] == "std" {
		os.Setenv("CGO_ENABLED", "0")
	}

	ds.registerAnalyzingLogMessage("Start analyzing ...")

	if !ds.analyzer.ParsePackages(ds.onAnalyzingSubTaskDone, args...) {
		if printUsage != nil {
			printUsage(os.Stdout)
		}
		os.Exit(1)
	}

	//{
	//	ds.mutex.Lock()
	//	ds.phase = Phase_Parsed
	//	ds.mutex.Unlock()
	//}

	ds.analyzer.AnalyzePackages(ds.onAnalyzingSubTaskDone)

	{
		ds.mutex.Lock()
		ds.phase = Phase_Analyzed
		ds.packagePages = make(map[string][]byte, ds.analyzer.NumPackages())
		ds.sourcePages = make(map[string][]byte, ds.analyzer.NumSourceFiles())
		ds.dependencyPages = make(map[string][]byte, ds.analyzer.NumPackages())
		ds.mutex.Unlock()
	}
}
