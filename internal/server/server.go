package server

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/language"

	"go101.org/golds/code"
	"go101.org/golds/internal/util"
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

	appPkgPath string
	//goldsVersion string

	workingDirectory string
	//emphasizeWDPkgs  bool

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
	//theCSSFile                cssFile
	//theOverviewPage           *overviewPage
	//theStatisticsPage         []byte
	//packagePages              map[string]packagePage
	//implPages                 map[implPageKey][]byte
	//identifierReferencesPages map[usePageKey][]byte
	//sourcePages               map[sourcePageKey][]byte
	//dependencyPages           map[string][]byte
	cachedPages map[pageCacheKey][]byte
	//cachedPagesOptions map[pageCacheKey]interface{} // key.options must be nil in this map

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

func Run(options PageOutputOptions, args []string, recommendedPort string, silentMode bool, printUsage func(io.Writer), appPkgPath string, roughBuildTime func() time.Time) {
	setPageOutputOptions(options, false)

	ds := &docServer{
		appPkgPath: appPkgPath,
		//goldsVersion: options.GoldsVersion,

		//emphasizeWDPkgs: options.EmphasizeWDPkgs,

		phase:           Phase_Unprepared,
		analyzer:        &code.CodeAnalyzer{},
		analyzingLogger: log.New(os.Stdout, "[Analyzing] ", 0),
		analyzingLogs:   make([]LoadingLogMessage, 0, 64),

		updateLogger:   log.New(os.Stdout, "[Update] ", 0),
		roughBuildTime: roughBuildTime,
	}

	if options.PreferredLang != "" {
		ds.visited = 1
		//ds.changeTranslationByAcceptLanguage(options.PreferredLang)
		ds.initSettings(options.PreferredLang)
	} else {
		ds.initSettings(os.Getenv("LANG"))
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%v", recommendedPort))
	if err != nil {
		log.Fatal(err)
	}
	delta := -1
	if addr.Port < 1234 {
		delta = 1
	}

NextTry:
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		if strings.Index(err.Error(), "bind: address already in use") >= 0 {
			addr.Port += delta
			goto NextTry
		}
		log.Fatal(err)
	}

	args, err = ds.validateArguments(args)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		ds.analyze(args, printUsage)
		ds.analyzingLogger.SetPrefix("")
		serverStarted := ds.currentTranslationSafely().Text_Server_Started()
		ds.analyzingLogger.Printf("%s http://localhost:%v\n", serverStarted, addr.Port)
	}()

	if !silentMode {
		err = util.OpenBrowser(fmt.Sprintf("http://localhost:%v", addr.Port))
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

var sem = make(chan struct{}, 10)

func (ds *docServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// To avoid too hight peak memory use cause by DDOS attack.
	// Rate Limiting is not very essential, for page content are cached.
	sem <- struct{}{}
	defer func() { <-sem }()

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
		ds.cssFile(w, r, removeVersionFromFilename(resPath, goldsVersion))
	case ResTypeJS: // "jvs"
		ds.javascriptFile(w, r, removeVersionFromFilename(resPath, goldsVersion))
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
		const sep = "/"
		index := strings.LastIndex(resPath, sep)
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Source file containing package is not specified")
		} else {
			ds.sourceCodePage(w, r, resPath[:index], resPath[index+len(sep):])
		}
	case ResTypeImplementation: // "imp"
		const sep = "."
		index := strings.LastIndex(resPath, sep)
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Interface type containing package is not specified")
		} else {
			ds.methodImplementationPage(w, r, resPath[:index], resPath[index+len(sep):])
		}
	case ResTypeReference: // "ref"
		// resPath doesn't contian unexported selectors with their package path prefixes for sure.
		// Two forms: pkg..id or pkg..type.selector.
		// As pkg might contains ".", so here we use ".." the seperator.
		// ToDo: // is better than ..?
		const sep = ".."
		index := strings.LastIndex(resPath, sep)
		if index < 0 {
			//ds.sourceCodePage(w, r, "", resPath)
			fmt.Fprint(w, "Identifer containing package is not specified")
		} else {
			ds.identifierReferencePage(w, r, resPath[:index], resPath[index+len(sep):])
		}
	}
}

func (ds *docServer) validateArguments(args []string) ([]string, error) {
	if len(args) == 0 {
		//args = []string{"."}
		panic("should not")
	}

	if len(args) == 1 && args[0] == "std" {
		os.Setenv("GO111MODULE", "off")
		os.Setenv("CGO_ENABLED", "0")
	} else {
		toolchainPath, dotPath := "", ""
		oldArgs := args
		args = args[:0]
		for _, p := range oldArgs {
			if p == "toolchain" {
				toolchainPath = filepath.Join(build.Default.GOROOT, "src", "cmd")
				if _, err := os.Stat(toolchainPath); errors.Is(err, os.ErrNotExist) {
					log.Printf("the toolchain argument is ignored for the assumed source directory (%s) doesn not exist", toolchainPath)
					continue
				}
				// looks both are ok.
				//args = append(args, toolchainPath + string(filepath.Separator) + "..."
				args = append(args, toolchainPath+"/...")
			} else {
				if p == "." || strings.HasPrefix(p, "./") || strings.HasPrefix(p, ".\\") {
					dotPath = p
				}
				args = append(args, p)
			}
		}
		if toolchainPath != "" {
			if dotPath != "" {
				return nil, fmt.Errorf("the toolchain argument conflicts with %s\n", dotPath)
			}
			os.Chdir(toolchainPath)
		}
	}

	return args, nil
}

func (ds *docServer) analyze(args []string, printUsage func(io.Writer)) {
	ds.workingDirectory, _ = os.Getwd()

	var stopWatch = util.NewStopWatch()
	defer func() {
		d := stopWatch.Duration(false)
		memUsed := util.MemoryUse()
		ds.registerAnalyzingLogMessage(func() string {
			return ds.currentTranslation.Text_Analyzing_Done(d, memUsed)
		})
		ds.registerAnalyzingLogMessage(func() string { return "" })
	}()

	ds.registerAnalyzingLogMessage(func() string {
		return ds.currentTranslationSafely().Text_Analyzing_Start()
	})

	if !ds.analyzer.ParsePackages(ds.onAnalyzingSubTaskDone, args...) {
		//if printUsage != nil {
		printUsage(os.Stdout)
		//}
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
		//ds.packagePages = make(map[string]packagePage, ds.analyzer.NumPackages())
		//ds.implPages = make(map[implPageKey][]byte, ds.analyzer.RoughTypeNameCount())
		//ds.identifierReferencesPages = make(map[usePageKey][]byte, ds.analyzer.RoughExportedIdentifierCount())
		//ds.sourcePages = make(map[sourcePageKey][]byte, ds.analyzer.NumSourceFiles())
		//ds.dependencyPages = make(map[string][]byte, ds.analyzer.NumPackages())

		n := ds.analyzer.NumPackages() +
			ds.analyzer.NumPackages() +
			ds.analyzer.NumSourceFiles() +
			int(ds.analyzer.RoughTypeNameCount()) +
			int(ds.analyzer.RoughExportedIdentifierCount())
		ds.cachedPages = make(map[pageCacheKey][]byte, int(n))
		//ds.cachedPagesOptions = make(map[pageCacheKey]interface{}, ds.analyzer.NumPackages())
		ds.mutex.Unlock()
	}
}
