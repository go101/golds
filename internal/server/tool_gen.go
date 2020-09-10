package server

import (
	"io"
	"log"
	"os"
)

type DocsGenerationOptions struct {
	NoIdentifierUsesPages bool
	PlainSourceCodePages  bool
	SilentMode            bool
	IncreaseGCFrequency   bool
	EmphasizeWdPkgs       bool
}

func Gen(intent, outputDir, lang string, args []string, options DocsGenerationOptions, goldVersion string, printUsage func(io.Writer), viewDocsCommand func(string) string) {
	log.SetFlags(0)

	// ...

	// ...

	switch intent {
	default:
		log.Println("Unknown gen intent:", intent)
		printUsage(os.Stdout)
	case "docs":
		GenDocs(outputDir, args, lang, options, goldVersion, printUsage, viewDocsCommand)
	case "testdata":
		GenTestData(outputDir, args, options.SilentMode, goldVersion, printUsage)
	}

	// ...
}
