package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"go101.org/gold/server"
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
	flag.Parse()
	if *hFlag || *helpFlag {
		printUsage(os.Stdout)
		return
	}

	if o := *oFlag; o != "" {
		server.Gen(o, flag.Args(), printUsage)
		return
	}

	server.Run(*portFlag, flag.Args(), printUsage)
}

var portFlag = flag.String("port", "56789", "preferred server port")
var oFlag = flag.String("o", "", "html generation output folder")
var hFlag = flag.Bool("h", false, "show help")
var helpFlag = flag.Bool("help", false, "show help")

// var versionFlag = flag.String("version", "", "show version info")

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Usage:
	%[1]v [options] [arguments]

Options:
	-port ServicePort
		Service port, default to 56789 (preferred) or a random one.
		This option will be ignored if the "-o" option presents.
	-o OutputFolder
		Generate all pages in the specified folder.

Examples:
	%[1]v std
		Show docs of standard packages.
	%[1]v x.y.z/myapp
		Show docs of package x.y.z/myapp.
	%[1]v
		Show docs of the package in the current directory.
	%[1]v ./...
		Show docs of the package and sub-packages in the current directory.
`, os.Args[0])
}
