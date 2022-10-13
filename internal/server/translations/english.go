package translations

import (
	"fmt"
	"time"

	"go101.org/golds/code"
)

type English struct{}

func (*English) Name() string { return "English" }

func (*English) LangTag() string { return "en-US" }

///////////////////////////////////////////////////////////////////
// common
///////////////////////////////////////////////////////////////////

func (*English) Text_Space() string { return " " }

func (*English) Text_Comma() string { return ", " }

func (*English) Text_Colon(atLineEnd bool) string {
	if atLineEnd {
		return ":"
	} else {
		return ": "
	}
}

func (*English) Text_Period(paragraphEnd bool) string {
	if paragraphEnd {
		return "."
	} else {
		return ". "
	}
}

func (*English) Text_Parenthesis(close bool) string {
	if close {
		return ")"
	} else {
		return " ("
	}
}

func (*English) Text_EnclosedInOarentheses(text string) string {
	return " (" + text + ")"
}

func (*English) Text_PreferredFontList() string { return `"Courier New", Courier, monospace` }

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
		return fmt.Sprintf("Collected one package: %s", d)
	}
	return fmt.Sprintf("Collected %d packages: %s", numPkgs, d)
}

func (*English) Text_Analyzing_CollectModules(numMods int, d time.Duration) string {
	if numMods == 1 {
		return fmt.Sprintf("Collected one module: %s", d)
	}
	return fmt.Sprintf("Collected %d modules: %s", numMods, d)
}

func (*English) Text_Analyzing_CollectExamples(d time.Duration) string {
	return fmt.Sprintf("Collected code examples: %s", d)
}

func (*English) Text_Analyzing_SortPackagesByDependencies(d time.Duration) string {
	return fmt.Sprintf("Sorted packages by dependency relations: %s", d)
}

func (*English) Text_Analyzing_CollectDeclarations(d time.Duration) string {
	return fmt.Sprintf("Collected declarations: %s", d)
}

func (*English) Text_Analyzing_CollectRuntimeFunctionPositions(d time.Duration) string {
	return fmt.Sprintf("Collected some runtime function positions: %s", d)
}

func (*English) Text_Analyzing_ConfirmTypeSources(d time.Duration) string {
	return fmt.Sprintf("Confirmed type sources: %s", d)
}

func (*English) Text_Analyzing_CollectSelectors(d time.Duration) string {
	return fmt.Sprintf("Collected selectors: %s", d)
}

func (*English) Text_Analyzing_FindImplementations(d time.Duration) string {
	return fmt.Sprintf("Found implementations: %s", d)
}

func (*English) Text_Analyzing_RegisterInterfaceMethodsForTypes(d time.Duration) string {
	return fmt.Sprintf("Registered interface methods: %s", d)
}

func (*English) Text_Analyzing_MakeStatistics(d time.Duration) string {
	return fmt.Sprintf("Made statistics: %s", d)
}

func (*English) Text_Analyzing_CollectSourceFiles(d time.Duration) string {
	return fmt.Sprintf("Collected Source Files: %s", d)
}

func (*English) Text_Analyzing_CollectObjectReferences(d time.Duration) string {
	return fmt.Sprintf("Collected Object References: %s", d)
}

func (*English) Text_Analyzing_CacheSourceFiles(d time.Duration) string {
	return fmt.Sprintf("Cached Source Files: %s", d)
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
	return fmt.Sprintf(`Total %d packages analyzed and %d Go files
(%d lines of code) parsed. On average,
* each Go source file imports %.2f packages
  and contains %.0f lines of code.
* each package depends on %.2f other packages,
  contains %.2f source code files, and exports
  - %.2f type names,
  - %.2f variables,
  - %.2f constants,
  - %.2f functions.`,
		stats.Packages, stats.AstFiles, stats.CodeLinesWithBlankLines,
		float64(stats.Imports)/float64(stats.AstFiles),
		float64(stats.CodeLinesWithBlankLines)/float64(stats.AstFiles),
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
		return `<b>Golds</b> has not been updated for more than one month. You may run <b>go %s</b> or <b><a href="/update">click here</a></b> to update it.`
	case "Updating":
		return `<b>Golds</b> is being updated.`
	case "Updated":
		return `<b>Golds</b> has been updated. You may restart the server to see the latest effect.`
	}
	return ""
}

func (*English) Text_SortBy(whatToSort string) string {
	switch whatToSort {
	case "packages":
		return "sort packages by"
	case "exporteds-types":
		return "sort exporteds by"
	default:
		return "sort by"
	}
}

func (*English) Text_SortByItem(by string) string {
	switch by {
	case "alphabet":
		return "alphabet"
	case "popularity":
		return "popularity"
	case "importedbys":
		return "imported-by count"
	case "depdepth":
		return "dependency distance"
	case "codelines":
		return "lines of code"
	default:
		panic("unknown sort-by: " + by)
	}
}

///////////////////////////////////////////////////////////////////
// package details page: type details
///////////////////////////////////////////////////////////////////

func (*English) Text_Package(pkgPath string) string {
	return fmt.Sprintf("Package: %s", pkgPath)
}

func (*English) Text_BelongingPackage() string { return "Belonging Package" }

func (*English) Text_PackageDocsLinksOnOtherWebsites(pkgPath string, isStdPkg bool) string {
	// https://github.com/golang/go/issues/44356
	//if isStdPkg {
	//	return fmt.Sprintf(`<i> (on <a href="https://golang.org/pkg/%[1]s/" target="_blank">golang.org</a> and <a href="https://pkg.go.dev/%[1]s" target="_blank">go.dev</a>)</i>`, pkgPath)
	//} else {
	return fmt.Sprintf(`<i> (on <a href="https://pkg.go.dev/%s" target="_blank">go.dev</a>)</i>`, pkgPath)
	//}
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

func (*English) Text_Examples(num int) string { return "Code Examples" }

func (*English) Text_PackageLevelTypeNames() string {
	return "Package-Level Type Names"
}

func (*English) Text_TypeParameters() string {
	return "Type Parameters"
}

//func (*English) Text_AllPackageLevelValues(num int) string {
//	return "Package-Level Values"
//}

func (*English) Text_PackageLevelFunctions() string {
	return "Package-Level Functions"
}
func (*English) Text_PackageLevelVariables() string {
	return "Package-Level Variables"
}
func (*English) Text_PackageLevelConstants() string {
	return "Package-Level Constants"
}

func (e *English) Text_PackageLevelResourceSimpleStat(statsAreExact bool, num, numExporteds int, mentionExporteds bool) string {
	var total, exporteds string
	if num == 1 {
		if statsAreExact {
			total = "only one"
		} else {
			total = "at least one"
			if numExporteds == 0 {
				return "at least one unexported"
			} else {
				return "at least one exported"
			}
		}
		if mentionExporteds {
			if numExporteds == 0 {
				exporteds = "which is unexported"
			} else {
				exporteds = "which is exported"
			}
		}
	} else {
		if statsAreExact {
			total = fmt.Sprintf("total %d", num)
		} else {
			total = fmt.Sprintf("at least %d", num)
		}

		if mentionExporteds {
			if numExporteds == 0 {
				if num == 2 {
					exporteds = "neither is exported"
				} else {
					exporteds = "none are exported"
				}
			} else if numExporteds == 1 {
				exporteds = "in which 1 is exported"
			} else if numExporteds == num {
				if num == 2 {
					exporteds = "both are exported"
				} else {
					exporteds = "all are exported"
				}
				//} else if statsAreExact {
				//	exporteds = fmt.Sprintf("in which %d are exported", numExporteds)
			} else {
				exporteds = fmt.Sprintf("in which %d are exported", numExporteds)
			}
		}
	}

	if exporteds == "" {
		return total
	}
	return total + e.Text_Comma() + exporteds
}

func (*English) Text_UnexportedResourcesHeader(show bool, numUnexporteds int, exact bool) string {
	if show {
		if numUnexporteds == 1 {
			if exact {
				return "/* one unexported ... */"
			}
			return "/* at least one unexported ... */"
		}
		if exact {
			return fmt.Sprintf("/* %d unexporteds ... */", numUnexporteds)
		}
		return fmt.Sprintf("/* %d+ unexporteds ... */", numUnexporteds)
	} else {
		if numUnexporteds == 1 {
			if exact {
				return "/* one unexported: */"
			}
			return "/* at least one unexported: */"
		}
		if exact {
			return fmt.Sprintf("/* %d unexporteds: */", numUnexporteds)
		}
		return fmt.Sprintf("/* %d+ unexporteds: */", numUnexporteds)
	}
}

func (*English) Text_ListUnexportes() string {
	return "list unexporteds"
}

///////////////////////////////////////////////////////////////////
// package details page: type details
///////////////////////////////////////////////////////////////////

func (*English) Text_BasicType() string {
	return "basic type"
}

func (*English) Text_Fields() string {
	return "Fields"
}

func (*English) Text_Methods() string {
	return "Methods"
}

func (*English) Text_ImplementedBy() string {
	return "Implemented By"
}

func (*English) Text_Implements() string {
	return "Implements"
}

func (*English) Text_AsOutputsOf() string {
	return "As Outputs Of"
}

func (*English) Text_AsInputsOf() string {
	return "As Inputs Of"
}

func (*English) Text_AsTypesOf() string {
	return "As Types Of"
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
// method implementation page
///////////////////////////////////////////////////////////////////

func (*English) Text_MethodImplementations() string {
	return "Method Implmentations"
}

func (*English) Text_NumMethodsImplementingNothing(count int) string {
	if count == 0 {
		return ""
	}
	s1, s2 := "", "s"
	if count > 1 {
		s1, s2 = s2, s1
	}
	return fmt.Sprintf(" (%d other method%s implement%s nothing)", count, s1, s2)
}

func (*English) Text_ViewMethodImplementations() string {
	return "view implemented interface methods"
}

///////////////////////////////////////////////////////////////////
// object reference page
///////////////////////////////////////////////////////////////////

func (*English) Text_ReferenceList() string {
	return "References"
}

func (*English) Text_CurrentPackage() string {
	return " (current package)"
}

func (*English) Text_ObjectKind(kind string) string {
	switch kind {
	case "field":
		return "field"
	case "method":
		return "method"
	default:
		panic("unknown object kind name: " + kind)
	}
}

func (*English) Text_ObjectUses(num int) string {
	if num == 1 {
		return "one use"
	}
	return fmt.Sprintf("%d uses", num)
}

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
		return "Numbers of Exported Struct Types by Embedded Field Counts"
	//case "exportedstructtypes-by-allfields":
	//	return "Numbers of Exported Struct Types by All Field Counts"
	case "exportedstructtypes-by-explicitfields":
		return "Numbers of Exported Struct Types by Explicit Field Counts"
	//case "exportedstructtypes-by-exportedfields":
	//	return "Numbers of Exported Struct Types by Exported (incl. Promoted) Field Counts"
	case "exportedstructtypes-by-exportedexplicitfields":
		return "Numbers of Exported Struct Types by Exported Explicit Field Counts"
	case "exportedstructtypes-by-exportedpromotedfields":
		return "Numbers of Exported Struct Types by Exported Promoted Field Counts"
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

func (*English) Text_PackageStatistics(values map[string]interface{}) []string {
	return []string{
		fmt.Sprintf(`
	Total <a href="%s">%d packages</a>, %d of them are standard packages.
	Total %d source files, %d of them are Go source files.
	Total %d lines of Go code.
	Averagely,
	- each Go source file imports %.2f packages and contains %.0f lines of code.
	- each package depends %.2f other packages and contains %.2f source files.

`,

			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s

			values["overviewPageURL"],
			values["packageCount"],
			values["standardPackageCount"],
			values["sourceFileCount"],
			values["goSourceFileCount"],
			values["goSourceLineCount"],
			values["averageImportCountPerFile"],
			values["averageCodeLineCountPerFile"],
			values["averageDependencyCountPerPackage"],
			values["averageSourceFileCountPerPackage"],

			//values["gosourcefilesByImportsChartURL"],
			//values["packagesByDependenciesChartURL"],
			//values["gosourcefilesByImportsChartSVG"],
			//values["packagesByDependenciesChartSVG"],
		),
	}
}

func (*English) Text_TypeStatistics(values map[string]interface{}) []string {
	return []string{
		fmt.Sprintf(`
	Total %d exported type names, %d of them are aliases.
	In them, %d are composite types and %d are basic types.
	In the basic types, %d are integers (%d are unsigneds).

`,
			//<!--img src=""></image-->%s
			values["exportedTypeNameCount"],
			values["exportedTypeAliases"],
			values["exportedCompositeTypeNames"],
			values["exportedBasicTypeNames"],
			values["exportedIntergerTypeNames"],
			values["exportedUnsignedTypeNames"],

			//values["exportedtypenamesByKindsChartURL"],
			//values["exportedtypenamesByKindsChartSVG"],
		),

		fmt.Sprintf(`
	In %d exported struct types, %d have embedded fields,
	and %d have promoted fields.

`,
			//<!--img src=""></image-->%s

			values["exportedStructTypeNames"],
			values["exportedNamedStructTypesWithEmbeddingFields"],
			values["exportedNamedStructTypesWithPromotedFields"],

			//values["exportedstructtypesByEmbeddingfieldsChartURL"],
			//values["exportedstructtypesByEmbeddingfieldsChartSVG"],
		),

		fmt.Sprintf(`
	On average, each exported struct type has
	* %.2f fields (including promoteds and unexporteds),
	* %.2f explicit fields (including unexporteds),
	* %.2f exported fields (including promoteds),
	* %.2f exported explicit fields.

`,
			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s

			values["exportedNamedStructTypeFieldsPerExportedStruct"],
			values["exportedNamedStructTypeExplicitFieldsPerExportedStruct"],
			values["exportedNamedStructTypeExportedFieldsPerExportedStruct"],
			values["exportedNamedStructTypeExportedExplicitFieldsPerExportedStruct"],

			//values["exportedstructtypesByExplicitfieldsChartURL"],
			//values["exportedstructtypesByExportedexplicitfieldsChartURL"],
			////values["exportedstructtypesByExportedfieldsChartURL"],
			//values["exportedstructtypesByExportedpromotedfieldsChartURL"],
			//values["exportedstructtypesByExplicitfieldsChartSVG"],
			//values["exportedstructtypesByExportedexplicitfieldsChartSVG"],
			////values["exportedstructtypesByExportedfieldsChartSVG"],
			//values["exportedstructtypesByExportedpromotedfieldsChartSVG"],
		),

		fmt.Sprintf(`
	Averagely,
	- for exported non-interface types with at least one exported
	  method, each of them has %.2f exported methods.
	- each exported interface type specified %.2f exported methods.

`,
			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s

			values["exportedNamedNonInterfacesExportedMethodsPerExportedNonInterfaceType"],
			values["exportedNamedInterfacesExportedMethodsPerExportedInterfaceType"],

			//values["exportednoninterfacetypesByExportedmethodsChartURL"],
			//values["exportedinterfacetypesByExportedmethodsChartURL"],
			//values["exportednoninterfacetypesByExportedmethodsChartSVG"],
			//values["exportedinterfacetypesByExportedmethodsChartSVG"],
		),
	}
}

func (*English) Text_ValueStatistics(values map[string]interface{}) []string {
	return []string{
		fmt.Sprintf(`
	Total %d exported variables and %d exported constants.

`,
			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s

			values["exportedVariables"],
			values["exportedConstants"],

			//values["exportedvariablesByTypekindsChartURL"],
			//values["exportedconstantsByTypekindsChartURL"],
			//values["exportedvariablesByTypekindsChartSVG"],
			//values["exportedconstantsByTypekindsChartSVG"],
		),

		fmt.Sprintf(`
	Total %d exported functions and %d exported explicit methods.
	On average, each of these functions and methods has
	%.2f parameters and %.2f results. For %d (%d%%) of these
	functions and methods, the last result types are <code>error</code>.

`,
			//<!--img src=""></image-->%s
			//<!--img src=""></image-->%s

			values["exportedFunctions"],
			values["exportedMethods"],
			values["averageParameterCountPerExportedFunction"],
			values["averageResultCountPerExportedFunction"],
			values["exportedFunctionWithLastErrorResult"],
			values["exportedFunctionWithLastErrorResultPercentage"],

			//values["exportedfunctionsByParametersChartURL"],
			//values["exportedfunctionsByResultsChartURL"],
			//values["exportedfunctionsByParametersChartSVG"],
			//values["exportedfunctionsByResultsChartSVG"],
		),
	}
}

func (*English) Text_Othertatistics(values map[string]interface{}) []string {
	return []string{
		fmt.Sprintf(`
	The average length of exported identifiers is %.2f.

`,
			//<!--img src=""></image-->%s

			values["averageIdentiferLength"],

		//values["exportedidentifiersByLengthsChartURL"],
		//values["exportedidentifiersByLengthsChartSVG"],
		),
	}
}

///////////////////////////////////////////////////////////////////
// footer
///////////////////////////////////////////////////////////////////

func (*English) Text_GeneratedPageFooter(goldsVersion, qrCodeLink, goOS, goArch string) string {
	var qrImg, tip string
	if qrCodeLink != "" {
		qrImg = fmt.Sprintf(`<img src="%s">`, qrCodeLink)
		tip = " (reachable from the left QR code)"
	}
	return fmt.Sprintf(`<table><tr><td>%s</td>
<td>The pages are generated with <a href="https://go101.org/apps-and-libs/golds.html"><b>Golds</b></a> <i>%s</i>. (GOOS=%s GOARCH=%s)
<b>Golds</b> is a <a href="https://go101.org">Go 101</a> project developed by <a href="https://tapirgames.com">Tapir Liu</a>.
PR and bug reports are welcome and can be submitted to <a href="https://github.com/go101/golds">the issue list</a>.
Please follow <a href="https://twitter.com/go100and1">@Go100and1</a>%s to get the latest news of <b>Golds</b>.</td></tr></table>`,
		qrImg,
		goldsVersion,
		goOS,
		goArch,
		tip,
	)
}

func (*English) Text_GeneratedPageFooterSimple(goldsVersion, goOS, goArch string) string {
	return fmt.Sprintf(`The pages are generated with <a href="https://go101.org/apps-and-libs/golds.html"><b>Golds</b></a> <i>%s</i>. (GOOS=%s GOARCH=%s)`,
		goldsVersion,
		goOS,
		goArch,
	)
}
