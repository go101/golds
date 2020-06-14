package translation

import (
	"fmt"
	"time"

	"go101.org/gold/code"
)

type English struct{}

func (*English) Name() string { return "English" }

func (*English) LangTag() string { return "en-US" }

///////////////////////////////////////////////////////////////////
// server
///////////////////////////////////////////////////////////////////

func (*English) Text_Server_Started() string {
	return "Server started:"
}

///////////////////////////////////////////////////////////////////
// analyzing
///////////////////////////////////////////////////////////////////

func (*English) Text_Analyzing() string { return "Analyzing ..." }

func (*English) Text_AnalyzingRefresh(currentPageURL string) string {
	return fmt.Sprintf(`Please wait a moment ... (<a href="%s">refresh</a>)`, currentPageURL)
}

func (*English) Text_Analyzing_Start() string {
	return "Start analyzing ..."
}

func (*English) Text_Analyzing_PreparationDone(d time.Duration) string {
	return fmt.Sprintf("Preparation done: %s", d)
}

func (*English) Text_Analyzing_NFilesParsed(numFiles int, d time.Duration) string {
	if numFiles == 1 {
		return fmt.Sprintf("One file parsed: %s", d)
	}
	return fmt.Sprintf("%d files parsed: %s", numFiles, d)
}

func (*English) Text_Analyzing_ParsePackagesDone(numFiles int, d time.Duration) string {
	if numFiles == 1 {
		return fmt.Sprintf("one file parsed: %s", d)
	}
	return fmt.Sprintf("All %d files are parsed: %s", numFiles, d)
}

func (*English) Text_Analyzing_CollectPackages(numPkgs int, d time.Duration) string {
	if numPkgs == 1 {
		return fmt.Sprintf("Collect one package: %s", d)
	}
	return fmt.Sprintf("Collect %d packages: %s", numPkgs, d)
}

func (*English) Text_Analyzing_SortPackagesByDependencies(d time.Duration) string {
	return fmt.Sprintf("Sort packages by dependency relations: %s", d)
}

func (*English) Text_Analyzing_CollectDeclarations(d time.Duration) string {
	return fmt.Sprintf("Collect declarations: %s", d)
}

func (*English) Text_Analyzing_CollectRuntimeFunctionPositions(d time.Duration) string {
	return fmt.Sprintf("Collect some runtime function positions: %s", d)
}

func (*English) Text_Analyzing_FindTypeSources(d time.Duration) string {
	return fmt.Sprintf("Find type sources: %s", d)
}

func (*English) Text_Analyzing_CollectSelectors(d time.Duration) string {
	return fmt.Sprintf("Collect selectors: %s", d)
}

func (*English) Text_Analyzing_FindImplementations(d time.Duration) string {
	return fmt.Sprintf("Find implementations: %s", d)
}

func (*English) Text_Analyzing_RegisterInterfaceMethodsForTypes(d time.Duration) string {
	return fmt.Sprintf("Register interface methods: %s", d)
}

func (*English) Text_Analyzing_MakeStatistics(d time.Duration) string {
	return fmt.Sprintf("Make statistics: %s", d)
}

func (*English) Text_Analyzing_CollectSourceFiles(d time.Duration) string {
	return fmt.Sprintf("Collect Source Files: %s", d)
}

func (*English) Text_Analyzing_Done(d time.Duration, memoryUse string) string {
	return fmt.Sprintf("Done. (Total time: %s, used memory: %s)", d, memoryUse)
}

///////////////////////////////////////////////////////////////////
// overview page
///////////////////////////////////////////////////////////////////

func (*English) Text_Overview() string { return "Overview" }

func (*English) Text_PackageList() string {
	return "All Packages"
}

func (*English) Text_StatisticsWithMoreLink(detailedStatsLink string) string {
	return fmt.Sprintf(`Statistics (<a href="%s">detailed ones</a>)`, detailedStatsLink)
}

func (*English) Text_SimpleStats(stats *code.Stats) string {
	return fmt.Sprintf(`Total %d packages analyzed and %d Go files parsed.
On average,
* each Go source file imports %.2f packages,
* each package depends on %.2f other packages,
  contains %.2f source code files, and exports
  - %.2f type names,
  - %.2f variables,
  - %.2f constants,
  - %.2f functions.`,
		stats.Packages, stats.AstFiles,
		float64(stats.Imports)/float64(stats.AstFiles),
		float64(stats.AllPackageDeps)/float64(stats.Packages),
		float64(stats.FilesWithoutGenerateds)/float64(stats.Packages),
		float64(stats.ExportedTypeNames)/float64(stats.Packages),
		float64(stats.ExportedVariables)/float64(stats.Packages),
		float64(stats.ExportedConstants)/float64(stats.Packages),
		float64(stats.ExportedFunctions)/float64(stats.Packages),
	)

}

func (*English) Text_Modules() string { return "Modules" }

func (*English) Text_BelongingModule() string { return "Belonging Module" }

func (*English) Text_RequireStat(numRequires, numRequiredBys int) string {
	return fmt.Sprintf("requires %d modules, and required by %d.", numRequires, numRequiredBys)
}

func (*English) Text_UpdateTip(tipName string) string {
	switch tipName {
	case "ToUpdate":
		return `<b>Gold</b> has not been updated for more than one month. You may run <b>go get -u go101.org/gold</b> or <b><a href="/update">click here</a></b> to update it.`
	case "Updating":
		return `<b>Gold</b> is being updated.`
	case "Updated":
		return `<b>Gold</b> has been updated. You may restart the server to see the latest effect.`
	}
	return ""
}

///////////////////////////////////////////////////////////////////
// package details page: type details
///////////////////////////////////////////////////////////////////

func (*English) Text_Package(pkgPath string) string {
	return fmt.Sprintf("Package: %s", pkgPath)
}

func (*English) Text_BelongingPackage() string { return "Belonging Package" }

func (*English) Text_PackageDocsLinksOnOtherWebsites(pkgPath string, isStdPkg bool) string {
	if isStdPkg {
		return fmt.Sprintf(`<i> (on <a href="https://golang.org/pkg/%[1]s/" target="_blank">golang.org</a> and <a href="https://pkg.go.dev/%[1]s" target="_blank">go.dev</a>)</i>`, pkgPath)
	} else {
		return fmt.Sprintf(`<i> (on <a href="https://pkg.go.dev/%s" target="_blank">go.dev</a>)</i>`, pkgPath)
	}
}

func (*English) Text_ImportPath() string { return "Import Path" }

func (*English) Text_ImportStat(numImports, numImportedBys int, depPageURL string) string {
	var importsStr, importedBysStr string

	if numImports == 1 {
		importsStr = "one package"
	} else {
		importsStr = fmt.Sprintf("%d packages", numImports)
	}
	if numImports > 0 {
		importsStr = fmt.Sprintf(`<a href="%s">%s</a>`, depPageURL, importsStr)
	}

	if numImportedBys == 1 {
		importedBysStr = "one package"
	} else {
		importedBysStr = fmt.Sprintf("%d packages", numImportedBys)
	}
	if numImportedBys > 0 {
		importedBysStr = fmt.Sprintf(`<a href="%s#imported-by">%s</a>`, depPageURL, importedBysStr)
	}

	return fmt.Sprintf(`imports %s, and imported by %s`, importsStr, importedBysStr)
}

func (*English) Text_InvolvedFiles(num int) string { return "Involved Source Files" }

func (*English) Text_ExportedValues(num int) string {
	return "Exported Values"
}

func (*English) Text_ExportedTypeNames(num int) string {
	return "Exported Type Names"
}

func (*English) Text_UnexportedTypeNames(num int) string {
	return "Unexported Type Names"
}

///////////////////////////////////////////////////////////////////
// package details page: type details
///////////////////////////////////////////////////////////////////

func (*English) Text_Fields(num int) string {
	if num == 1 {
		return "One Exported Field"
	}
	return fmt.Sprintf("Exported Fields (%d)", num)
}

func (*English) Text_Methods(num int) string {
	if num == 1 {
		return "One Exported Method"
	}
	return fmt.Sprintf("Exported Methods (%d)", num)
}

func (*English) Text_ImplementedBy(num int) string {
	return fmt.Sprintf("Implemented By (%d+)", num)
}

func (*English) Text_Implements(num int) string {
	return fmt.Sprintf("Implements (%d+)", num)
}

func (*English) Text_AsOutputsOf(num int) string {
	return fmt.Sprintf("As Outputs Of (%d+)", num)
}

func (*English) Text_AsInputsOf(num int) string {
	return fmt.Sprintf("As Inputs Of (%d+)", num)
}

func (*English) Text_AsTypesOf(num int) string {
	return fmt.Sprintf("As Types Of (%d+)", num)
}

func (*English) Text_References(num int) string {
	return fmt.Sprintf("References (%d+)", num)
}

///////////////////////////////////////////////////////////////////
// package dependencies page
///////////////////////////////////////////////////////////////////

func (*English) Text_DependencyRelations(pkgPath string) string {
	if pkgPath == "" {
		return "Dependency Relation" // used in package details page
	} else {
		return fmt.Sprintf("Dependency Relation: %s", pkgPath)
	}
}

func (*English) Text_Imports() string { return "Imports" }

func (*English) Text_ImportedBy() string { return "Imported By" }

///////////////////////////////////////////////////////////////////
// source code page
///////////////////////////////////////////////////////////////////

func (*English) Text_SourceCode(pkgPath, bareFilename string) string {
	return fmt.Sprintf("Source: %s in package %s", bareFilename, pkgPath)
}

func (*English) Text_SourceFilePath() string { return "Source File" }

func (*English) Text_GeneratedFrom() string { return "Generated From" }

///////////////////////////////////////////////////////////////////
// statistics
///////////////////////////////////////////////////////////////////

func (*English) Text_Statistics() string {
	return "Statistics"
}

func (*English) Text_ChartTitle(chartName string) string {
	switch chartName {
	case "gosourcefiles-by-imports":
		return "Numbers of Go Source Files by Import Counts"
	case "packages-by-dependencies":
		return "Numbers of Packages by Dependency Counts"
	case "exportedtypenames-by-kinds":
		return "Numbers of Exported Type Names by Kinds"
	case "exportedstructtypes-by-embeddingfields":
		return "Numbers of Exported Struct Types by Embedding Field Counts"
	//case "exportedstructtypes-by-allfields":
	//	return "Numbers of Exported Struct Types by All Field Counts"
	case "exportedstructtypes-by-explicitfields":
		return "Numbers of Exported Struct Types by Explicit Field Counts"
	case "exportedstructtypes-by-exportedfields":
		return "Numbers of Exported Struct Types by Exported (incl. Promoted) Field Counts"
	case "exportedstructtypes-by-exportedexplicitfields":
		return "Numbers of Exported Struct Types by Exported Explicit Field Counts"
	//case "exportedstructtypes-by-exportedpromotedfields":
	//	return "Numbers of Exported Struct Types by Exported Promoted Field Counts"
	case "exportedfunctions-by-parameters":
		return "Numbers of Exported Functions/Methods by Parameter Counts"
	case "exportedfunctions-by-results":
		return "Numbers of Exported Functions/Methods by Result Counts"
	case "exportedidentifiers-by-lengths":
		return "Number of Exported Identifiers by Lengths"
	case "exportedvariables-by-typekinds":
		return "Numbers of Exported Variables by Type Kinds"
	case "exportedconstants-by-typekinds":
		return "Numbers of Exported Constants by Type (or Default Type) Kinds"
	case "exportednoninterfacetypes-by-exportedmethods":
		return "Numbers of Exported Non-Interface Types by Exported Method Counts"
	case "exportedinterfacetypes-by-exportedmethods":
		return "Numbers of Exported Interface Types by Exported Method Counts"
	default:
		panic("unknown char name: " + chartName)
	}
}

func (*English) Text_StatisticsTitle(titleName string) string {
	switch titleName {
	case "packages":
		return "Packages"
	case "types":
		return "Types"
	case "values":
		return "Values"
	case "others":
		return "Others"
	default:
		panic("unknown statistics tile: " + titleName)
	}
}

func (*English) Text_PackageStatistics(values map[string]interface{}) string {
	return fmt.Sprintf(`
	Total <a href="%s">%d packages</a>, %d of them are standard packages.
	Total %d source files, %d of them are Go source files.
	Averagely,
	- each package contains %.2f source files,
	- each Go source file imports %.2f packages,
	- each package depends %.2f other packages.

	<img src="%s"></image>
	<img src="%s"></image>
`,
		values["overviewPageURL"],
		values["packageCount"],
		values["standardPackageCount"],
		values["sourceFileCount"], 	
		values["goSourceFileCount"],
		values["averageSourceFileCountPerPackage"], 
		values["averageImportCountPerFile"], 	
		values["averageDependencyCountPerPackage"],

		values["gosourcefilesByImportsChartURL"], 
		values["packagesByDependenciesChartURL"], 
	)
}

func (*English) Text_TypeStatistics(values map[string]interface{}) string {
	return fmt.Sprintf(`
	Total %d exported type names, %d of them are aliases.
	In them, %d are composite types and %d are basic types.
	In the basic types, %d are integers (%d are unsigneds).

	<img src="%s"></image>

	In %d exported struct types, %d have embedding fields,
	and %d have promoted fields.

	<img src="%s"></image>

	On average, each exported struct type has
	* %.2f fields (including promoteds and unexporteds),
	* %.2f explicit fields (including unexporteds),
	* %.2f exported fields (including promoteds),
	* %.2f exported explicit fields.

	<img src="%s"></image>
	<img src="%s"></image>
	<img src="%s"></image>

	Averagely,
	- for exported non-interface types with at least one exported
	  method, each of them has %.2f exported methods.
	- each exported interface type specified %.2f exported methods.

	<img src="%s"></image>
	<img src="%s"></image>
`,
		values["exportedTypeNameCount"],
		values["exportedTypeAliases"],
		values["exportedCompositeTypeNames"],
		values["exportedBasicTypeNames"],
		values["exportedIntergerTypeNames"],
		values["exportedUnsignedTypeNames"],

		values["exportedtypenamesByKindsChartURL"],

		values["exportedStructTypeNames"],
		values["exportedNamedStructTypesWithEmbeddingFields"],
		values["exportedNamedStructTypesWithPromotedFields"],

		values["exportedstructtypesByEmbeddingfieldsChartURL"],

		values["exportedNamedStructTypeFieldsPerExportedStruct"],
		values["exportedNamedStructTypeExplicitFieldsPerExportedStruct"],
		values["exportedNamedStructTypeExportedFieldsPerExportedStruct"],
		values["exportedNamedStructTypeExportedExplicitFieldsPerExportedStruct"],

		values["exportedstructtypesByExplicitfieldsChartURL"],
		values["exportedstructtypesByExportedexplicitfieldsChartURL"],
		values["exportedstructtypesByExportedfieldsChartURL"],

		values["exportedNamedNonInterfacesExportedMethodsPerExportedNonInterfaceType"],
		values["exportedNamedInterfacesExportedMethodsPerExportedInterfaceType"],

		values["exportednoninterfacetypesByExportedmethodsChartURL"],
		values["exportedinterfacetypesByExportedmethodsChartURL"],
	)
}

func (*English) Text_ValueStatistics(values map[string]interface{}) string {
	return fmt.Sprintf(`
	Total %d exported variables and %d exported constants.

	<img src="%s"></image>
	<img src="%s"></image>

	Total %d exported functions and %d exported explicit methods.
	On average, each of these functions and methods has
	%.2f parameters and %.2f results. For %d (%d%%) of these
	functions and methods, the last result types are <code>error</code>.

	<img src="%s"></image>
	<img src="%s"></image>
`,
		values["exportedVariables"],
		values["exportedConstants"],

		values["exportedvariablesByTypekindsChartURL"],
		values["exportedconstantsByTypekindsChartURL"],

		values["exportedFunctions"],
		values["exportedMethods"],
		values["averageParameterCountPerExportedFunction"],
		values["averageResultCountPerExportedFunction"],
		values["exportedFunctionWithLastErrorResult"],
		values["exportedFunctionWithLastErrorResultPercentage"],

		values["exportedfunctionsByParametersChartURL"],
		values["exportedfunctionsByResultsChartURL"],
	)
}

func (*English) Text_Othertatistics(values map[string]interface{}) string {
	return fmt.Sprintf(`
	The average length of exported identifiers is %.2f.

	<img src="%s"></image>
`,
		values["averageIdentiferLength"],

		values["exportedidentifiersByLengthsChartURL"],
	)
}

///////////////////////////////////////////////////////////////////
// footer
///////////////////////////////////////////////////////////////////

func (*English) Text_GeneratedPageFooter(goldVersion string) string {
	return fmt.Sprintf(`Generated with <a href="https://go101.org/article/tool-gold.html"><b>Gold</b></a> <i>%s</i>.
<b>Gold</b> is a <a href="https://go101.org">Go 101</a> project started by <a href="https://tapirgames.com">TapirLiu</a>.
Please follow <a href="https://twitter.com/go100and1">@Go100and1</a> to get the latest news of <b>Gold</b>.
PR and bug reports are welcomed and can be submitted <a href="https://github.com/go101/gold">here</a>.`,
		goldVersion,
	)
}


