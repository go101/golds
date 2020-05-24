package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"go101.org/gold/server"
	"go101.org/gold/util"
)

func init() {
	// Leave some power for others.
	var numProcessors = runtime.NumCPU() * 3 / 4
	if numProcessors < 1 {
		numProcessors = 1
	}
	runtime.GOMAXPROCS(numProcessors)
}

func main() {
	// This is used for updating Gold. It is invisible to users.
	var roughBuildTimeFlag = flag.Bool("rough-build-time", false, "show rough build time")

	flag.Parse()
	if *hFlag || *helpFlag {
		printUsage(os.Stdout)
		return
	}

	if *roughBuildTimeFlag {
		fmt.Print(RoughBuildTime)
		return
	}
	var getRoughBuildTime = func() time.Time {
		output, err := util.RunShellCommand(time.Second*3, "", os.Args[0], "-rough-build-time")
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

	if o := *genFlag; o != "" {
		server.Gen(o, flag.Args(), printUsage, getRoughBuildTime)
		return
	}

	silentMode := *silentFlag || *sFlag

	if d := *dirFlag; d != "" {
		util.ServeFiles(d, *portFlag, silentMode)
		return
	}

	server.Run(*portFlag, flag.Args(), silentMode, printUsage, getRoughBuildTime)
}

var hFlag = flag.Bool("h", false, "show help")
var helpFlag = flag.Bool("help", false, "show help")
var genFlag = flag.String("gen", "", "html generation output folder")
var dirFlag = flag.String("dir", "", "file serving folder")
var portFlag = flag.String("port", "56789", "preferred server port")
var sFlag = flag.Bool("s", false, "not open a browser automatically")
var silentFlag = flag.Bool("silent", false, "not open a browser automatically")

// var versionFlag = flag.String("version", "", "show version info")

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Usage:
	%[1]v [options] [arguments]

Options (by priority order):
	-h/-help
		Show help information.
		When the flags present, others will be ignored.
	-gen  OutputFolder
		Generate all pages in the specified folder.
		This flag will surpress "dir" and "port" flags.
	-dir  FileServingDirectory
		The directory in which the files are served.
	-port ServicePort
		Service port, default to 56789.
		If the specified or default port is not
		availabe, a random port will be used.
	-s/-silent
		Don't open a browser automatically.

Examples:
	%[1]v std
		Show docs of standard packages.
	%[1]v x.y.z/myapp
		Show docs of package x.y.z/myapp.
	%[1]v
		Show docs of the package in the current directory.
	%[1]v .
		Show docs of the package in the current directory.
	%[1]v ./...
		Show docs of the package and sub-packages in the
		current directory.
	%[1]v -gen=./generated ./...
		Generate HTML docs pages for the package and
		sub-packages in the current directory.
	%[1]v -dir=.
		Serving the files in the current directory.
`,
		os.Args[0])
}
