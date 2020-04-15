package server

import (
	"io"

	"go101.org/gold/code"
)

func Gen(port string, args []string, printUsage func(io.Writer)) {
	ds := &docServer{
		phase:    Phase_Unprepared,
		analyzer: &code.CodeAnalyzer{},
	}

	ds.analyze(args, printUsage)

}
