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
	"sync/atomic"
	"time"

	"golang.org/x/text/language"

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

	//
	allThemes                  []Theme
	allTranslations            []Translation
	langMatcher                language.Matcher
	translationsByLangTagIndex []Translation

	//
	phase           int
	analyzer        *code.CodeAnalyzer
	analyzingLogger *log.Logger
	analyzingLogs   []LoadingLogMessage

	// Cached pages
	theCSSFile         cssFile
	theOverviewPage    *overviewPage
	theStatisticsPage  []byte
	packagePages       map[string]packagePage
	implPages          map[implPageKey][]byte
	identifierUsePages map[usePageKey][]byte
	sourcePages        map[sourcePageKey][]byte
	dependencyPages    map[string][]byte

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
	visited       int32
}

func Run(port, lang string, args []string, silentMode bool, goldVersion string, printUsage func(io.Writer), roughBuildTime func() time.Time) {
	ds := &docServer{
		goldVersion: goldVersion,

		phase:           Phase_Unprepared,
		analyzer:        &code.CodeAnalyzer{},
		analyzingLogger: log.New(os.Stdout, "[Analyzing] ", 0),
		analyzingLogs:   make([]LoadingLogMessage, 0, 64),

		updateLogger:   log.New(os.Stdout, "[Update] ", 0),
		roughBuildTime: roughBuildTime,
	}

	ds.initSettings(os.Getenv("LANG"))

	if lang != "" {
		ds.visited = 1
		ds.changeTranslationByAcceptLanguage(lang)
	}

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
		serverStarted := ds.currentTranslationSafely().Text_Server_Started()
		ds.analyzingLogger.Printf("%s http://localhost:%v\n", serverStarted, port)
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
	if atomic.SwapInt32(&ds.visited, 1) == 0 {
		ds.changeTranslationByAcceptLanguage(r.Header.Get("Accept-Language"))
	}

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
			ds.statisticsPage(w, r)
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
	case ResTypePNG: // "png"
		ds.pngFile(w, r, resPath)
	//case "mod:": // module
	//	ds.modulePage(w, r, path)
	case ResTypePackage: // "pkg"
		ds.packageDetailsPage(w, r, resPath)
	case ResTypeDependency: // "dep"
		ds.packageDependenciesPage(w, r, resPath)
	case ResTypeSource: // "src"
		index := strings.LastIndex(resPath, "/")
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Source file containing package is not specified")
		} else {
			ds.sourceCodePage(w, r, resPath[:index], resPath[index+1:])
		}
	case ResTypeImplementation: // "imp"
		index := strings.LastIndex(resPath, ".")
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Interface type containing package is not specified")
		} else {
			ds.methodImplementationPage(w, r, resPath[:index], resPath[index+1:])
		}
	case ResTypeUse: // "use"
		index := strings.LastIndex(resPath, ".")
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Identifer containing package is not specified")
		} else {
			ds.identifierUsesPage(w, r, resPath[:index], resPath[index+1:])
		}
	}
}

func (ds *docServer) analyze(args []string, printUsage func(io.Writer)) {
	var stopWatch = util.NewStopWatch()
	defer func() {
		d := stopWatch.Duration(false)
		memUsed := util.MemoryUse()
		ds.registerAnalyzingLogMessage(func() string {
			return ds.currentTranslation.Text_Analyzing_Done(d, memUsed)
		})
		ds.registerAnalyzingLogMessage(func() string { return "" })
	}()

	if len(args) == 0 {
		args = []string{"."}
	} else if len(args) == 1 && args[0] == "std" {
		os.Setenv("CGO_ENABLED", "0")
	}

	ds.registerAnalyzingLogMessage(func() string {
		return ds.currentTranslationSafely().Text_Analyzing_Start()
	})

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
		ds.packagePages = make(map[string]packagePage, ds.analyzer.NumPackages())
		ds.implPages = make(map[implPageKey][]byte, ds.analyzer.RoughTypeNameCount())
		ds.identifierUsePages = make(map[usePageKey][]byte, ds.analyzer.RoughExportedIdentifierCount())
		ds.sourcePages = make(map[sourcePageKey][]byte, ds.analyzer.NumSourceFiles())
		ds.dependencyPages = make(map[string][]byte, ds.analyzer.NumPackages())
		ds.mutex.Unlock()
	}
}
