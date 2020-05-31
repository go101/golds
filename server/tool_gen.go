package server

import (
	"io"
	"log"
	"os"
)

func Gen(intent, outputDir string, args []string, silent bool, goldVersion string, printUsage func(io.Writer), viewDocsCommand func(string) string) {
	log.SetFlags(0)

	// ...

	// ...

	switch intent {
	default:
		log.Println("Unknown gen intent:", intent)
		printUsage(os.Stdout)
	case "docs":
		GenDocs(outputDir, args, goldVersion, silent, printUsage, viewDocsCommand)
	case "testdata":
		GenTestData(outputDir, args, goldVersion, silent, printUsage)
	}

	// ...
}
