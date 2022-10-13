package server

import (
	"fmt"
	"math"
	"net/http"
	"reflect"

	"go101.org/golds/code"
)

func (ds *docServer) statisticsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	//if ds.theStatisticsPage == nil {
	//	ds.theStatisticsPage = ds.buildStatisticsData()
	//}
	//w.Write(ds.theStatisticsPage)

	pageKey := pageCacheKey{
		resType: ResTypeNone,
		res:     "statistics",
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		data = ds.buildStatisticsPage(w)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildStatisticsPage(w http.ResponseWriter) []byte {
	page := NewHtmlPage(goldsVersion, ds.currentTranslation.Text_Statistics(), ds.currentTheme, ds.currentTranslation, createPagePathInfo(ResTypeNone, "statistics"))
	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">%s</span></code></pre>
`,
		page.Translation().Text_Statistics(),
	)

	//svgData := func(svgFile string) []byte {
	//	return ds.buildSVG(svgFile, page.Translation().Text_ChartTitle(svgFile))
	//}

	writeSVG := func(svgFile string) {
		page.WriteString("\t")
		// ToDo: pass page as io.Writer argument
		page.Write(ds.buildSVG(svgFile, page.Translation().Text_ChartTitle(svgFile)))
		page.WriteString("\n")
	}

	writeSVGwithFolding := func(svgFile string, topsItems []interface{}, listTops func([]interface{})) {
		if len(topsItems) == 0 {
			writeSVG(svgFile)
			return
		}

		writeFoldingBlock(page, svgFile, "stats", "docs", false,
			func() {
				writeSVG(svgFile)
			},
			func() {
				listTops(topsItems)
			},
		)
	}

	stats := ds.analyzer.Statistics()

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, page.Translation().Text_StatisticsTitle("packages"))
	textSegments := page.Translation().Text_PackageStatistics(map[string]interface{}{
		"overviewPageURL":                  buildPageHref(page.PathInfo, createPagePathInfo(ResTypeNone, ""), nil, ""),
		"packageCount":                     stats.Packages,
		"standardPackageCount":             stats.StdPackages,
		"sourceFileCount":                  stats.FilesWithoutGenerateds,
		"goSourceFileCount":                stats.AstFiles,
		"goSourceLineCount":                stats.CodeLinesWithBlankLines,
		"averageImportCountPerFile":        float64(stats.Imports) / float64(stats.AstFiles),
		"averageCodeLineCountPerFile":      math.Round(float64(stats.CodeLinesWithBlankLines) / float64(stats.AstFiles)),
		"averageDependencyCountPerPackage": float64(stats.AllPackageDeps) / float64(stats.Packages),
		"averageSourceFileCountPerPackage": float64(stats.FilesWithGenerateds) / float64(stats.Packages),
		"averageCodeLineCountPerPackage":   math.Round(float64(stats.CodeLinesWithBlankLines) / float64(stats.Packages)),

		//"gosourcefilesByImportsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "gosourcefiles-by-imports"), nil, ""),
		//"packagesByDependenciesChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "packages-by-dependencies"), nil, ""),
		//"gosourcefilesByImportsChartSVG": svgData("gosourcefiles-by-imports"),
		//"packagesByDependenciesChartSVG": svgData("packages-by-dependencies"),
	})
	page.WriteString(textSegments[0])

	writeSVGwithFolding("gosourcefiles-by-imports", stats.FilesImportCountTopList.Items, func(items []interface{}) {
		defer page.WriteString("\n")
		for _, t := range items {
			v, ok := t.(*struct {
				*code.Package
				Filename string
			})
			if !ok {
				continue
			}
			func() {
				page.WriteString("\t\t\t")
				defer page.WriteString("\n")
				page.WriteString(v.Package.Path)
				page.WriteByte('/')
				writeSrouceCodeFileLink(page, v.Package, v.Filename)
			}()
		}
	})

	writeSVGwithFolding("packages-by-dependencies", stats.PackagesDepsTopList.Items, func(items []interface{}) {
		defer page.WriteString("\n")
		for _, t := range items {
			pkgPath, ok := t.(*string)
			if !ok {
				continue
			}
			func() {
				page.WriteString("\t\t\t")
				defer page.WriteString("\n")
				fmt.Fprintf(page,
					`<a href="%s">%s</a>`,
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypeDependency, *pkgPath), nil, ""),
					*pkgPath,
				)
			}()
		}
	})

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, page.Translation().Text_StatisticsTitle("types"))
	textSegments = page.Translation().Text_TypeStatistics(map[string]interface{}{
		"exportedTypeNameCount":      stats.ExportedTypeNames,
		"exportedTypeAliases":        stats.ExportedTypeAliases,
		"exportedCompositeTypeNames": stats.ExportedCompositeTypeNames,
		"exportedBasicTypeNames":     stats.ExportedBasicTypeNames,
		"exportedIntergerTypeNames":  stats.ExportedIntergerTypeNames,
		"exportedUnsignedTypeNames":  stats.ExportedUnsignedTypeNames,

		//"exportedtypenamesByKindsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedtypenames-by-kinds"), nil, ""),
		//"exportedtypenamesByKindsChartSVG": svgData("exportedtypenames-by-kinds"),

		"exportedStructTypeNames":                     stats.ExportedTypeNamesByKind[reflect.Struct],
		"exportedNamedStructTypesWithEmbeddingFields": stats.ExportedNamedStructTypesWithEmbeddingFields,
		"exportedNamedStructTypesWithPromotedFields":  stats.ExportedNamedStructTypesWithPromotedFields,

		//"exportedstructtypesByEmbeddingfieldsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedstructtypes-by-embeddingfields"), nil, ""),
		//"exportedstructtypesByEmbeddingfieldsChartSVG": svgData("exportedstructtypes-by-embeddingfields"),

		"exportedNamedStructTypeFieldsPerExportedStruct":                 float64(stats.ExportedNamedStructTypeFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExplicitFieldsPerExportedStruct":         float64(stats.ExportedNamedStructTypeExplicitFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExportedFieldsPerExportedStruct":         float64(stats.ExportedNamedStructTypeExportedFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExportedExplicitFieldsPerExportedStruct": float64(stats.ExportedNamedStructTypeExportedExplicitFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),

		//"exportedstructtypesByExplicitfieldsChartURL":         buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedstructtypes-by-explicitfields"), nil, ""),
		//"exportedstructtypesByExportedexplicitfieldsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedstructtypes-by-exportedexplicitfields"), nil, ""),
		////"exportedstructtypesByExportedfieldsChartURL":         buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedstructtypes-by-exportedfields"), nil, ""),
		//"exportedstructtypesByExportedpromotedfieldsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedstructtypes-by-exportedpromotedfields"), nil, ""),
		//"exportedstructtypesByExplicitfieldsChartSVG":         svgData("exportedstructtypes-by-explicitfields"),
		//"exportedstructtypesByExportedexplicitfieldsChartSVG": svgData("exportedstructtypes-by-exportedexplicitfields"),
		////"exportedstructtypesByExportedfieldsChartSVG":         svgData("exportedstructtypes-by-exportedfields"),
		//"exportedstructtypesByExportedpromotedfieldsChartSVG": svgData("exportedstructtypes-by-exportedpromotedfields"),

		"exportedNamedNonInterfacesExportedMethodsPerExportedNonInterfaceType": float64(stats.ExportedNamedNonInterfacesExportedMethods) / float64(stats.ExportedNamedNonInterfacesWithExportedMethods),
		"exportedNamedInterfacesExportedMethodsPerExportedInterfaceType":       float64(stats.ExportedNamedInterfacesExportedMethods) / float64(stats.ExportedTypeNamesByKind[reflect.Interface]),

		//"exportednoninterfacetypesByExportedmethodsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportednoninterfacetypes-by-exportedmethods"), nil, ""),
		//"exportedinterfacetypesByExportedmethodsChartURL":    buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedinterfacetypes-by-exportedmethods"), nil, ""),
		//"exportednoninterfacetypesByExportedmethodsChartSVG": svgData("exportednoninterfacetypes-by-exportedmethods"),
		//"exportedinterfacetypesByExportedmethodsChartSVG":    svgData("exportedinterfacetypes-by-exportedmethods"),
	})
	page.WriteString(textSegments[0])
	writeSVG("exportedtypenames-by-kinds")

	page.WriteString(textSegments[1])

	// All are linked to package details page.
	writeTypes := func(items []interface{}) {
		defer page.WriteString("\n")
		for _, t := range items {
			if tn, ok := t.(*code.TypeName); ok {
				func() {
					page.WriteString("\t\t\t")
					defer page.WriteString("\n")
					page.WriteString(tn.Package().Path)
					page.WriteByte('.')
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, tn.Package().Path), page, tn.Name(), "name-", tn.Name())
				}()
			}
		}
	}

	writeSVGwithFolding("exportedstructtypes-by-embeddingfields", stats.ExportedNamedStructsEmbeddingFieldCountTopList.Items, func(items []interface{}) {
		writeTypes(items)
	})

	page.WriteString(textSegments[2])

	writeSVGwithFolding("exportedstructtypes-by-explicitfields", stats.ExportedNamedStructsExplicitFieldCountTopList.Items, func(items []interface{}) {
		writeTypes(items)
	})

	writeSVGwithFolding("exportedstructtypes-by-exportedexplicitfields", stats.ExportedNamedStructsExportedExplicitFieldCount.Items, func(items []interface{}) {
		writeTypes(items)
	})

	//writeSVG("exportedstructtypes-by-exportedfields")

	writeSVGwithFolding("exportedstructtypes-by-exportedpromotedfields", stats.ExportedNamedStructsExportedPromotedFieldCount.Items, func(items []interface{}) {
		writeTypes(items)
	})

	page.WriteString(textSegments[3])

	writeSVGwithFolding("exportednoninterfacetypes-by-exportedmethods", stats.ExportedNamedNonInterfaceTypesExportedMethodCountTopList.Items, func(items []interface{}) {
		writeTypes(items)
	})

	writeSVGwithFolding("exportedinterfacetypes-by-exportedmethods", stats.ExportedNamedInterfacesExportedMethodCountTopList.Items, func(items []interface{}) {
		writeTypes(items)
	})

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, page.Translation().Text_StatisticsTitle("values"))
	textSegments = page.Translation().Text_ValueStatistics(map[string]interface{}{
		"exportedVariables": stats.ExportedVariables,
		"exportedConstants": stats.ExportedConstants,

		//"exportedvariablesByTypekindsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedvariables-by-typekinds"), nil, ""),
		//"exportedconstantsByTypekindsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedconstants-by-typekinds"), nil, ""),
		//"exportedvariablesByTypekindsChartSVG": svgData("exportedvariables-by-typekinds"),
		//"exportedconstantsByTypekindsChartSVG": svgData("exportedconstants-by-typekinds"),

		"exportedFunctions":                             stats.ExportedFunctions,
		"exportedMethods":                               stats.ExportedMethods,
		"averageParameterCountPerExportedFunction":      float64(stats.ExportedFunctionParameters) / float64(stats.ExportedFunctions+stats.ExportedMethods),
		"averageResultCountPerExportedFunction":         float64(stats.ExportedFunctionResults) / float64(stats.ExportedFunctions+stats.ExportedMethods),
		"exportedFunctionWithLastErrorResult":           stats.ExportedFunctionWithLastErrorResult,
		"exportedFunctionWithLastErrorResultPercentage": int(math.Round(100 * float64(stats.ExportedFunctionWithLastErrorResult) / float64(stats.ExportedFunctions+stats.ExportedMethods))),

		//"exportedfunctionsByParametersChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedfunctions-by-parameters"), nil, ""),
		//"exportedfunctionsByResultsChartURL":    buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedfunctions-by-results"), nil, ""),
		//"exportedfunctionsByParametersChartSVG": svgData("exportedfunctions-by-parameters"),
		//"exportedfunctionsByResultsChartSVG":    svgData("exportedfunctions-by-results"),
	})
	page.WriteString(textSegments[0])
	writeSVG("exportedvariables-by-typekinds")
	writeSVG("exportedconstants-by-typekinds")

	page.WriteString(textSegments[1])

	// All are linked to source code, for some are method functions.
	writeFunctions := func(items []interface{}) {
		defer page.WriteString("\n")
		for _, t := range items {
			if f, ok := t.(*code.Function); ok {
				func() {
					page.WriteString("\t\t\t")
					defer page.WriteString("\n")
					page.WriteString(f.Package().Path)
					page.WriteByte('.')
					ds.writeResourceIndexHTML(page, f.Package(), f, false, false, false)
				}()
			}
		}
	}

	writeSVGwithFolding("exportedfunctions-by-parameters", stats.ExportedFunctionsParameterCountTopList.Items, func(items []interface{}) {
		writeFunctions(items)
	})

	writeSVGwithFolding("exportedfunctions-by-results", stats.ExportedFunctionsResultCountTopList.Items, func(items []interface{}) {
		writeFunctions(items)
	})

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, page.Translation().Text_StatisticsTitle("others"))
	textSegments = page.Translation().Text_Othertatistics(map[string]interface{}{
		"averageIdentiferLength": float64(stats.ExportedIdentifersSumLength) / float64(stats.ExportedIdentifers),

		//"exportedidentifiersByLengthsChartURL": buildPageHref(page.PathInfo, createPagePathInfo(ResTypeSVG, "exportedidentifiers-by-lengths"), nil, ""),
		//"exportedidentifiersByLengthsChartSVG": svgData("exportedidentifiers-by-lengths"),
	})
	page.WriteString(textSegments[0])

	// All are linked to source code, for some are fields/methods.
	writeSVGwithFolding("exportedidentifiers-by-lengths", stats.ExportedIdentiferLengthTopList.Items, func(items []interface{}) {
		defer page.WriteString("\n")
		for _, t := range items {
			func() {
				page.WriteString("\t\t\t")
				defer page.WriteString("\n")
				switch t := t.(type) {
				case *struct {
					*code.TypeName
					*code.Selector
				}:
					page.WriteString(t.TypeName.Package().Path)
					page.WriteByte('.')
					page.WriteString(t.TypeName.Name())
					page.WriteByte('.')
					ds.writeFieldCodeLink(page, t.Selector)
					return
				case *code.TypeName, *code.Function, *code.Constant, *code.Variable:
					res := t.(code.Resource)
					page.WriteString(res.Package().Path)
					page.WriteByte('.')
					ds.writeResourceIndexHTML(page, res.Package(), res, false, false, false)
				}
			}()
		}
	})

	return page.Done(w)
}
