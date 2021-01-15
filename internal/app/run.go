package app

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go101.org/golds/internal/server"
	"go101.org/golds/internal/util"
)

func init() {
	// Leave some power for others.
	var numProcessors = runtime.NumCPU() * 3 / 4
	if numProcessors < 1 {
		numProcessors = 1
	}
	runtime.GOMAXPROCS(numProcessors)
}

func Run() {
	// This is used for updating Golds. It is invisible to users.
	var roughBuildTimeFlag = flag.Bool("rough-build-time", false, "show rough build time")

	flag.CommandLine.Usage = func() {
		printUsage(os.Stdout)
	}

	flag.Parse()
	if *roughBuildTimeFlag {
		fmt.Print(RoughBuildTime)
		return
	}
	if *hFlag || *helpFlag {
		printUsage(os.Stdout)
		return
	}
	if *versionFlag {
		printVersion(os.Stdout)
		return
	}

	var getRoughBuildTime = func() time.Time {
		output, err := util.RunShellCommand(time.Second*5, "", nil, os.Args[0], "-rough-build-time")
		if err != nil {
			log.Printf("Run: %s -rough-build-time error: %s", os.Args[0], err)
			return time.Now()
		}

		t, err := time.Parse("2006-01-02", string(output))
		if err != nil {
			log.Printf("! parse rough build time (%s) error: %s", output, err)
			return time.Now()
		}

		return t
	}

	// ...
	log.SetFlags(log.Lshortfile)

	var validateDir = func(dir string) string {
		if dir == "" {
			dir = "."
		} else if dir == "memory" {
			dir = "" // not to save, for testing purpose.
		} else {
			dir = strings.TrimRight(dir, "\\/")
			dir = strings.Replace(dir, "/", string(filepath.Separator), -1)
			dir = strings.Replace(dir, "\\", string(filepath.Separator), -1)
		}
		return dir
	}

	silentMode := *silentFlag || *sFlag

	// files serving mode
	if flag.NArg() == 0 {
		log.SetFlags(0)

		if *dirFlag == "" {
			log.Printf(`Running in directory serving mode. If docs serving mode is intended, please run the following command instead:
   %s .

`,
				strings.Join(os.Args, " ")[len(os.Args[0])-len(filepath.Base(os.Args[0])):],
			)
		}
		if *portFlag == "" {
			*portFlag = "9999" // to be consistent with the one used in the old golf program.
		}

		util.ServeFiles(validateDir(*dirFlag), *portFlag, silentMode, Version)
		return
	}

	emphasizeWDPkgs := *emphasizeWorkingDirectoryPackages
	if *compact {
		*nouses = true
		*plainsrc = true
		// ToDo: exit on conflicts
	}
	options := server.PageOutputOptions{
		GoldsVersion:          Version,
		PreferredLang:         *langFlag,
		NoIdentifierUsesPages: *nouses,
		PlainSourceCodePages:  *plainsrc,
		EmphasizeWDPkgs:       emphasizeWDPkgs,
	}

	// docs generating mode
	if gen := *genFlag; gen {
		outputDir := validateDir(*dirFlag)
		switch intent := *genIntentFlag; intent {
		default:
			log.Println("Unknown gen intent:", intent)
			printUsage(os.Stdout)
		case "testdata":
			server.GenTestData(flag.Args(), outputDir, silentMode, printUsage)
		case "docs":
			viewDocsCommand := func(docsDir string) string {
				return os.Args[0] + " -dir=" + docsDir
			}
			// ToDo: also support json format output
			server.GenDocs(options, flag.Args(), outputDir, silentMode, printUsage, *moregcFlag, viewDocsCommand)
		}

		return
	}

	// docs serving mode

	//appPkgPath := "go101.org/gold" // changed to "golds" now.
	appPkgPath := "go101.org/golds" // for updating Golds
	switch appName := filepath.Base(os.Args[0]); appName {
	case "gold", "godoge", "gocore":
		appPkgPath += "/" + appName
	default:
	case "golds":
	}

	server.Run(options, flag.Args(), *portFlag, silentMode, printUsage, appPkgPath, getRoughBuildTime)
}

var hFlag = flag.Bool("h", false, "show help")
var helpFlag = flag.Bool("help", false, "show help")

//var uFlag = flag.Bool("u", false, "update self")
//var updateFlag = flag.Bool("update", false, "update self")
var versionFlag = flag.Bool("version", false, "show version info")
var genFlag = flag.Bool("gen", false, "HTML generation mode")
var genIntentFlag = flag.String("gen-intent", "docs", "docs | testdata")
var langFlag = flag.String("lang", "", "docs generation language tag")
var dirFlag = flag.String("dir", "", "directory for file serving or HTML generation")
var portFlag = flag.String("port", "", "preferred server port [1024, 65536]. Default: 56789 or 9999")
var sFlag = flag.Bool("s", false, "not open a browser automatically")
var silentFlag = flag.Bool("silent", false, "not open a browser automatically")
var moregcFlag = flag.Bool("moregc", false, "increase garbage collection frequency")
var emphasizeWorkingDirectoryPackages = flag.Bool("emphasize-wdpkgs", false, "disable the source navigation feature")
var nouses = flag.Bool("nouses", false, "disable the identifier uses feature")
var plainsrc = flag.Bool("plainsrc", false, "disable the source navigation feature")
var compact = flag.Bool("compact", false, "sacrifice some disk-consuming features in generation")

func printVersion(out io.Writer) {
	fmt.Fprintf(out, "Golds %s\n", Version)
}

// Cancelled options:
//	-u/-update
//		Update Golds itself.

// Hidden options:
//	-moregc
//		Increase garbage collection frequency
//		to lower peak memroy use. For HTML docs
//		generation mode only. Enabling it will
//		slow down the docs generation speed.

// Planning options:
//	-generated-packages=wd|all
//		Determine which packages the docs will be generated for.

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Golds - a Go local docs server (%[2]s).

Usage:
	%[1]v [options] [arguments]

Options:
	-h/-help
		Show help information. When the flags
		present, others will be ignored.
	-port=<ServicePort>
		Service port, defaults to 56789 in docs
		mode and 9999 in files serving mode.
		If the specified or default port is not
		availabe, another availabe port will be
		selected automatically.
	-s/-silent
		Don't open a browser automatically
		or don't show HTML file generation
		logs in docs generation mode.
	-gen
		Static HTML docs generation mode.
		"memory" means not to save (for testing).
	-dir=<ContentDirectory>|memory
		Specifiy the docs generation or file
		serving diretory. Current directory
		will be used if no arguments specified.
		"memory" means not to save (for testing).
	-nouses
		Disable the identifier uses feature.
		For HTML docs generation mode only.
	-plainsrc
		Disable the source navigation feature.
		For HTML docs generation mode only.
	-compact
		This is a shortcut of the combination
		of several other options, including
		-nouses and -plainsrc now.
	-emphasize-wdpkgs
		List the packages under the current
		directory before other pacakges.
		For HTML docs generation mode only.

Examples:
	%[1]v std
		Show docs of standard packages.
	%[1]v x.y.z/myapp
		Show docs of package x.y.z/myapp.
	%[1]v .
		Show docs of the package in the
		current directory.
	%[1]v ./...
		Show docs of all the packages
		within the current directory.
	%[1]v -gen -dir=./generated ./...
		Generate HTML docs pages into the path
		specified by the -dir flag for the
		packages under the current directory
		and their dependency packages.
	%[1]v -dir=. -s
		Serve the files in working directory
		without opening a browser window.
	%[1]v
		Serve the files in working directory and
		open a browser window to list items. If
		there is a file named "index.html", then
		it will be rendered in the browser window.
`,
		filepath.Base(os.Args[0]),
		Version,
	)
}
