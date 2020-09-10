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

	"go101.org/gold/internal/server"
	"go101.org/gold/internal/util"
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
	// This is used for updating Gold. It is invisible to users.
	var roughBuildTimeFlag = flag.Bool("rough-build-time", false, "show rough build time")

	flag.CommandLine.Usage = func() {
		printUsage(os.Stdout)
	}

	flag.Parse()
	if *hFlag || *helpFlag {
		printUsage(os.Stdout)
		return
	}

	if *versionFlag {
		printVersion(os.Stdout)
		return
	}

	if *roughBuildTimeFlag {
		fmt.Print(RoughBuildTime)
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

	var validateDiir = func(dir string) string {
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

	if gen := *genFlag; gen {
		viewDocsCommand := func(docsDir string) string {
			return os.Args[0] + " -dir=" + docsDir
		}
		options := server.DocsGenerationOptions{
			NoIdentifierUsesPages: *nouses,
			PlainSourceCodePages:  *plainsrc,
			SilentMode:            silentMode,
			IncreaseGCFrequency:   *moregcFlag,
			EmphasizeWdPkgs:       *emphasizeWorkingDirectoryPackages,
		}
		server.Gen(*genIntentFlag, validateDiir(*dirFlag), *langFlag, flag.Args(), options, Version, printUsage, viewDocsCommand)
		return
	}

	if flag.NArg() == 0 {
		log.SetFlags(0)

		if *dirFlag == "" {
			log.Printf(`Running in directory serving mode. If docs serving mode
is expected, please run the following command instead:
	%s .

`,
				strings.Join(os.Args, " ")[len(os.Args[0])-len(filepath.Base(os.Args[0])):],
			)
		}
		if *portFlag == "" {
			*portFlag = "9999" // to be consistent with the one used in the old golf program.
		}

		util.ServeFiles(validateDiir(*dirFlag), *portFlag, silentMode, Version)
		return
	}

	server.Run(*portFlag, *langFlag, flag.Args(), silentMode, Version, printUsage, getRoughBuildTime)
}

var hFlag = flag.Bool("h", false, "show help")
var helpFlag = flag.Bool("help", false, "show help")
var versionFlag = flag.Bool("version", false, "show version info")
var genFlag = flag.Bool("gen", false, "HTML generation mode")
var genIntentFlag = flag.String("gen-intent", "docs", "docs | testdata")
var langFlag = flag.String("lang", "", "docs generation language tag")
var dirFlag = flag.String("dir", "", "directory for file serving or HTML generation")
var portFlag = flag.String("port", "", "preferred server port [1024, 65536]. Default: 56789")
var sFlag = flag.Bool("s", false, "not open a browser automatically")
var silentFlag = flag.Bool("silent", false, "not open a browser automatically")
var moregcFlag = flag.Bool("moregc", false, "increase garbage collection frequency")
var nouses = flag.Bool("nouses", false, "disable the identifier uses feature")
var plainsrc = flag.Bool("plainsrc", false, "disable the source navigation feature")
var emphasizeWorkingDirectoryPackages = flag.Bool("emphasize-wdpkgs", false, "disable the source navigation feature")

func printVersion(out io.Writer) {
	fmt.Fprintf(out, "Gold %s\n", Version)
}

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Gold - a Go local docs server (%[2]s).

Usage:
	%[1]v [options] [arguments]

Options:
	-h/-help
		Show help information. When the flags
		present, others will be ignored.
	-port=ServicePort
		Service port, default to 56789. If
		the specified or default port is not
		availabe, a random port will be used.
	-s/-silent
		Don't open a browser automatically
		or don't show HTML file generation
		logs in docs generation mode.
	-gen
		Static HTML docs generation mode.
	-dir=ContentDirectory
		Specifiy the docs generation or file
		serving diretory. Current directory
		will be used if no arguments specified.
		"memory" means not to save (for testing).
	-moregc
		Increase garbage collection frequency
		to lower peak memroy use. For HTML docs
		generation mode only. Enabling it will
		slow down the docs generation speed.
	-nouses
		Disable the identifier uses feature.
		For HTML docs generation mode only.
	-plainsrc
		Disable the source navigation feature.
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
