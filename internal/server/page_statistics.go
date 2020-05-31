package server

import (
	"fmt"
	"reflect"

	"go101.org/gold/code"
)

// ToDo:
func (ds *docServer) statisticsPage(page *htmlPage, stats *code.Stats) {
	// w.WriteHeader(http.StatusTooEarly)
	
	var sum = func(kinds ...reflect.Kind) (r int32) {
		for _, k := range kinds {
			r += stats.ExportedTypeNamesByKind[k]
		}
		return
	}

	fmt.Fprintf(page, `
<pre><code><span class="title">%s</span></code>
	Packages: %d, n of them are standard packages.
	- By Dependencies: %v
	- Dependencies/Package: %.2f
	- Source Files/Package: %.2f
	Source Files:          %d
	Parsed Go Files:       %d
	- By Imports:          %v
	- Imports/GoFile:        %.2f,
	Exported Named Types:  %d, Aliases: %d,
	- By Kind:             %v
	  - Numerics:          %d
	    - Integers:        %d (Unsigneds: %d)
	    - Floating-Points: %d (float64: %d)
	    - Complexs:        %d (complex128: %d)
	  - Structs By Fields:             %v
	    - By Explicit Fields:          %v
	    - By Exported Fields:          %v
	    - By Exported Explicit Fields: %v
	  - Interfaces By Methods:     %v
	    - By Exported Methods:     %v
	  - Non-Interfaces By Methods: %v
	    - By Exported Methods:     %v
	Exported Variables: %d
	Exported Constants: %d
	Functions:          %d (Exporteds: %d)
	- By Parameters:  %v
	- By Results:     %v
	Methods:          %d (Exporteds: %d)
	- By Parameters:  %v
	- By Results:     %v
	Identifer ave. length. Average funciton name length, ...
	- num camel styles
	- num a_b styles
	- IdentifiersByLength [64]int32
	- top N identifers of most length
	- top N functions with most parameters/results, ...
`,
		ds.currentTranslation.Text_Statistics(),
		stats.Packages,
		stats.PackagesByDeps,
		float64(stats.AllPackageDeps)/float64(stats.Packages),
		float64(stats.FilesWithGenerateds)/float64(stats.Packages),
		stats.FilesWithoutGenerateds,
		stats.AstFiles,
		stats.FilesByImportCount,
		float64(stats.Imports)/float64(stats.AstFiles),
		stats.ExportedTypeNames,
		stats.ExportedTypeAliases,
		stats.ExportedTypeNamesByKind,
		stats.ExportedNamedNumericTypes,
		stats.ExportedNamedIntergerTypes,
		stats.ExportedNamedUnsignedIntergerTypes,
		sum(reflect.Float32, reflect.Float64),
		sum(reflect.Float64),
		sum(reflect.Complex64, reflect.Complex128),
		sum(reflect.Complex128),
		stats.NamedStructsByFieldCount,
		stats.NamedStructsByExplicitFieldCount,
		stats.NamedStructsByExportedFieldCount,
		stats.NamedStructsByExportedExplicitFieldCount,
		stats.ExportedNamedInterfacesByMethodCount,
		stats.ExportedNamedInterfacesByExportedMethodCount,
		stats.ExportedNamedNonInterfaceTypesByMethodCount,
		stats.ExportedNamedNonInterfaceTypesByExportedMethodCount,
		stats.ExportedVariables,
		stats.ExportedConstants,
		stats.Functions,
		stats.ExportedFunctions,
		stats.FunctionsByParameterCount,
		stats.FunctionsByResultCount,
		stats.Methods,
		stats.ExportedMethods,
		stats.MethodsByParameterCount,
		stats.MethodsByResultCount,
	)

}
