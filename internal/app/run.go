package app

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
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

func Main() {
	const ballastSize = 128 << 19
	ballast := make([]byte, ballastSize)

	run() // never exit

	runtime.KeepAlive(&ballast)
}

func run() {
	// This is used for updating Golds. It is invisible to users.
	var roughBuildTimeFlag = flag.Bool("rough-build-time", false, "show rough build time")

	flag.Parse()

	if *roughBuildTimeFlag {
		fmt.Print(RoughBuildTime)
		return
	}
	if *hFlag || *helpFlag {
		printUsage(os.Stdout)
		return
	}
	if *versionFlag || flag.NArg() == 1 && flag.Arg(0) == "version" {
		printVersion(os.Stdout)
		return
	}
	if flag.NArg() == 1 && flag.Arg(0) == "release" {
		releaseGolds()
		return
	}

	flag.CommandLine.Usage = func() {
		printUsage(os.Stdout)
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

	var validateDir = func(dir string, forGenerating bool) string {
		if dir == "" {
			if forGenerating {
				dir = "generated-" + time.Now().Format("20060102150405")
			} else {
				dir = "."
			}
		} else if dir == "memory" {
			if forGenerating {
				dir = "" // not to save, for testing purpose.
			}
		} else {
			dir = strings.TrimRight(dir, "\\/")
			dir = strings.Replace(dir, "/", string(filepath.Separator), -1)
			dir = strings.Replace(dir, "\\", string(filepath.Separator), -1)
		}
		return dir
	}

	silentMode := *silentFlag || *sFlag
	verboseMode := *verboseFlag || *vFlag

	// files serving mode
	if flag.NArg() == 0 && !*genFlag {
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

		util.ServeFiles(validateDir(*dirFlag, false), *portFlag, silentMode, Version)
		return
	}

	// Use user GOROOT instead binary releaser GOROOT.
	output, err := util.RunShellCommand(time.Second*5, "", nil, "go", "env", "GOROOT")
	if err != nil {
		log.Fatalf("Run: go env GOROOT error: %s", err)
		//return
	}
	if gr := string(bytes.TrimSpace(output)); gr != "" {
		build.Default.GOROOT = gr // the initial value is the value of releaser machine
	}

	// docs generating

	wdPkgsListingManner := *wdPkgsListingMannerFlag
	switch wdPkgsListingManner {
	default:
		log.Fatalln("Unknown wdpkgs-listing option:", wdPkgsListingManner)
		//return
	case "":
		if *emphasizeWdPackagesFlag {
			wdPkgsListingManner = server.WdPkgsListingManner_promoted
		} else {
			wdPkgsListingManner = server.WdPkgsListingManner_general
		}
	case server.WdPkgsListingManner_promoted:
	case server.WdPkgsListingManner_general:
		fallthrough
	case server.WdPkgsListingManner_solo:
		if *emphasizeWdPackagesFlag {
			log.Fatalln("emphasize-wdpkgs and wdpkgs-listing options conflict")
			//return
		}
	}
	if wdPkgsListingManner == server.WdPkgsListingManner_promoted || *emphasizeWdPackagesFlag {
		log.Println("Note: The -emphasize-wdpkgs option has been depreciated by -wdpkgs-listing=promoted")
		log.Println()
	}

	footerShowingManner := *footerShowingMannerFlag
	switch footerShowingManner {
	default:
		log.Fatalln("Unknown footer-showing option:", footerShowingManner)
		//return
	case "":
		footerShowingManner = server.FooterShowingManner_verbose_and_qrcode
	case server.FooterShowingManner_none:
	case server.FooterShowingManner_simple:
	case server.FooterShowingManner_verbose:
	case server.FooterShowingManner_verbose_and_qrcode:
	}

	srcReadingStyle := *srcReadingStyleFlag
	switch {
	default:
		log.Fatalln("Unknown source-code-reading option:", srcReadingStyle)
		//return
	case srcReadingStyle == "":
		if *plainsrc || *compact {
			srcReadingStyle = server.SourceReadingStyle_plain
		} else {
			srcReadingStyle = server.SourceReadingStyle_rich
		}
	case srcReadingStyle == server.SourceReadingStyle_plain:
		if *plainsrc {
			log.Println("Note: The -plainsrc option has been depreciated by -source-code-reading=plain")
			log.Println()
		}
	case srcReadingStyle == server.SourceReadingStyle_highlight:
	case srcReadingStyle == server.SourceReadingStyle_rich:
	case strings.HasPrefix(srcReadingStyle, server.SourceReadingStyle_external):
	}
	if srcReadingStyle != server.SourceReadingStyle_plain && *plainsrc {
		log.Printf("Note: The -plainsrc option suppressed by -source-code-reading=%s", srcReadingStyle)
		log.Println()
	}

	if *compact {
		*nouses = true
		//*plainsrc = true
		*nounexporteds = true
	}

	options := server.PageOutputOptions{
		GoldsVersion:           Version,
		PreferredLang:          *langFlag,
		NoStatistics:           *nostats,
		NoIdentifierUsesPages:  *nouses,
		SourceReadingStyle:     srcReadingStyle,
		AllowNetworkConnection: *allowNetworkConnection,
		NotCollectUnexporteds:  *nounexporteds,
		WdPkgsListingManner:    wdPkgsListingManner,
		FooterShowingManner:    footerShowingManner,
		RenderDocLinks:         *renderDocLinksFlag,
		UnfoldAllInitially:     *unfoldAllInitiallyFlag,
		VerboseLogs:            verboseMode,
	}

	// static docs generating mode
	if gen := *genFlag; gen {
		outputDir := validateDir(*dirFlag, true)
		switch intent := *genIntentFlag; intent {
		default:
			log.Println("Unknown gen intent:", intent)
			//printUsage(os.Stdout)
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

	// dynamic docs serving mode

	//appPkgPath := "go101.org/gold" // changed to "golds" now.
	appPkgPath := "go101.org/golds" // for updating Golds
	switch appName := filepath.Base(os.Args[0]); appName {
	case "gold", "godoge", "gocore":
		appPkgPath += "/" + appName
	default:
	case "golds":
	}

	if *portFlag == "" {
		*portFlag = "56789"
	}

	server.Run(options, flag.Args(), *portFlag, silentMode, printUsage, appPkgPath, getRoughBuildTime)
}

var hFlag = flag.Bool("h", false, "show help")
var helpFlag = flag.Bool("help", false, "show help")
var vFlag = flag.Bool("v", false, "verbose mode")
var verboseFlag = flag.Bool("verbose", false, "verbose mode")

// var uFlag = flag.Bool("u", false, "update self")
// var updateFlag = flag.Bool("update", false, "update self")
var versionFlag = flag.Bool("version", false, "show version info")
var genFlag = flag.Bool("gen", false, "HTML generation mode")
var genIntentFlag = flag.String("gen-intent", "docs", "docs | testdata")
var langFlag = flag.String("lang", "", "docs generation language tag")
var dirFlag = flag.String("dir", "", "directory for file serving or HTML generation")
var portFlag = flag.String("port", "", "preferred server port [1024, 65536]. Default: 56789 or 9999")
var sFlag = flag.Bool("s", false, "not open a browser automatically")
var silentFlag = flag.Bool("silent", false, "not open a browser automatically")
var moregcFlag = flag.Bool("moregc", false, "increase garbage collection frequency")

var nostats = flag.Bool("nostats", false, "disable the statistics feature")
var nouses = flag.Bool("nouses", false, "disable the identifier uses feature")
var nounexporteds = flag.Bool("only-list-exporteds", false, "don't collect unexported package-level resources")
var compact = flag.Bool("compact", false, "sacrifice some disk-consuming features in generation")

var plainsrc = flag.Bool("plainsrc", false, "disable the source navigation feature")
var srcReadingStyleFlag = flag.String("source-code-reading", "", "specify how and where to show source code")

var allowNetworkConnection = flag.Bool("allow-network-connection", false, "specify whether or not network connections are allowed")

var footerShowingMannerFlag = flag.String("footer", "verbose+qrcode", "verbose+qrcode | verbose | simple | none")

var renderDocLinksFlag = flag.Bool("render-doclinks", false, "render links in doc comments")
var unfoldAllInitiallyFlag = flag.Bool("unfold-all-initially", false, "unfold all foldables initially")

// depreciated by "-wdpkgs-listing=promoted" since v0.1.8
var emphasizeWdPackagesFlag = flag.Bool("emphasize-wdpkgs", false, "promote working directory packages")
var wdPkgsListingMannerFlag = flag.String("wdpkgs-listing", "", "specify how to list working directory packages")

func printVersion(out io.Writer) {
	fmt.Fprintf(out, "Golds %s\n", Version)
}

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
		available, another available port will be
		selected automatically.
	-s/-silent
		Don't open a browser automatically
		or don't show HTML file generation
		logs in docs generation mode.
	-gen
		Static HTML docs generation mode.
	-dir=<ContentDirectory>|memory
		Specify the docs generation or file
		serving diretory. A new created subfolder
		with a random name under the current directory
		will be used if this option is not specified.
		"memory" means not to save (for testing).
	-nostats
		Disable the statistics feature.
	-nouses
		Disable the identifier uses feature.
	-plainsrc (depreciated)
		Disable the source navigation feature.
		Depreciated by "-source-code-reading=plain".
	-source-code-reading=plain|highlight|rich|external
		How and where to read source code
		(default is rich):
		* plain: plain experience.
		* highlight: highlight keywords only.
		* rich: rich experience.
		* external: read code on external code hosting
		  websites. Do its best, use highlight on fails.
	-allow-network-connection
		When enabled,
		* source files of the packages which external
		  host URLs couldn't be determined locally
		  will be found out by sending a HTTPS query.
		* Possible more cases needing net connections.
	-only-list-exporteds
		Not to list unexported resources
		in package-details pages.
	-compact
		This is a shortcut of the combination
		of several other options, including
		-nouses, -only-list-exporteds and
		-source-code-reading=plain|cvs now.
	-emphasize-wdpkgs (depreciated)
		List the packages under the current
		directory before other pacakges.
		Depreciated by "-wdpkgs-listing=promoted".
	-wdpkgs-listing=promoted|solo|general
		Specify how to list the packages in the
		working directory (default is general):
		* promoted: list them before others.
		* solo: list them without others.
		* general: list them with others by
		  alphabetical order.
	-footer-showing=verbose+qrcode|verbose|simple|none
		Specify how page footers should be shown.
		Available values (default is verbose+qrcode):
		* none: show nothing.
		* simple: show Golds version only.
		* verbose: also show Golds author.
		  promotion info.
		* verbose+qrcode: include verbose content
		  and a qr-code.
	-render-doclinks
		Whether or not to render the links in docs.
	-unfold-all-initially
		Unfold all foldables initially.

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
