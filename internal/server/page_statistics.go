package server

import (
	"fmt"
	"math"
	"net/http"
	"reflect"
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

	if ds.theStatisticsPage == nil {
		ds.theStatisticsPage = ds.buildStatisticsData()
	}
	w.Write(ds.theStatisticsPage)
}

func (ds *docServer) buildStatisticsData() []byte {
	page := NewHtmlPage(ds.goldVersion, ds.currentTranslation.Text_Statistics(), ds.currentTheme.Name(), pagePathInfo{ResTypeNone, "statistics"})
	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">%s</span></code></pre>
`,
		ds.currentTranslation.Text_Statistics(),
	)

	stats := ds.analyzer.Statistics()

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, ds.currentTranslation.Text_StatisticsTitle("packages"))
	page.WriteString(ds.currentTranslation.Text_PackageStatistics(map[string]interface{}{
		"overviewPageURL":                  buildPageHref(page.PathInfo, pagePathInfo{ResTypeNone, ""}, nil, ""),
		"packageCount":                     stats.Packages,
		"standardPackageCount":             stats.StdPackages,
		"sourceFileCount":                  stats.FilesWithoutGenerateds,
		"goSourceFileCount":                stats.AstFiles,
		"averageSourceFileCountPerPackage": float64(stats.FilesWithGenerateds) / float64(stats.Packages),
		"averageImportCountPerFile":        float64(stats.Imports) / float64(stats.AstFiles),
		"averageDependencyCountPerPackage": float64(stats.AllPackageDeps) / float64(stats.Packages),

		"gosourcefilesByImportsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "gosourcefiles-by-imports"}, nil, ""),
		"packagesByDependenciesChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "packages-by-dependencies"}, nil, ""),
	}))

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, ds.currentTranslation.Text_StatisticsTitle("types"))
	page.WriteString(ds.currentTranslation.Text_TypeStatistics(map[string]interface{}{
		"exportedTypeNameCount":      stats.ExportedTypeNames,
		"exportedTypeAliases":        stats.ExportedTypeAliases,
		"exportedCompositeTypeNames": stats.ExportedCompositeTypeNames,
		"exportedBasicTypeNames":     stats.ExportedBasicTypeNames,
		"exportedIntergerTypeNames":  stats.ExportedIntergerTypeNames,
		"exportedUnsignedTypeNames":  stats.ExportedUnsignedTypeNames,

		"exportedtypenamesByKindsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedtypenames-by-kinds"}, nil, ""),

		"exportedStructTypeNames":                     stats.ExportedTypeNamesByKind[reflect.Struct],
		"exportedNamedStructTypesWithEmbeddingFields": stats.ExportedNamedStructTypesWithEmbeddingFields,
		"exportedNamedStructTypesWithPromotedFields":  stats.ExportedNamedStructTypesWithPromotedFields,

		"exportedstructtypesByEmbeddingfieldsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedstructtypes-by-embeddingfields"}, nil, ""),

		"exportedNamedStructTypeFieldsPerExportedStruct":                 float64(stats.ExportedNamedStructTypeFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExplicitFieldsPerExportedStruct":         float64(stats.ExportedNamedStructTypeExplicitFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExportedFieldsPerExportedStruct":         float64(stats.ExportedNamedStructTypeExportedFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),
		"exportedNamedStructTypeExportedExplicitFieldsPerExportedStruct": float64(stats.ExportedNamedStructTypeExportedExplicitFields) / float64(stats.ExportedTypeNamesByKind[reflect.Struct]),

		"exportedstructtypesByExplicitfieldsChartURL":         buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedstructtypes-by-explicitfields"}, nil, ""),
		"exportedstructtypesByExportedexplicitfieldsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedstructtypes-by-exportedexplicitfields"}, nil, ""),
		//"exportedstructtypesByExportedfieldsChartURL":         buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedstructtypes-by-exportedfields"}, nil, ""),
		"exportedstructtypesByExportedpromotedfieldsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedstructtypes-by-exportedpromotedfields"}, nil, ""),

		"exportedNamedNonInterfacesExportedMethodsPerExportedNonInterfaceType": float64(stats.ExportedNamedNonInterfacesExportedMethods) / float64(stats.ExportedNamedNonInterfacesWithExportedMethods),
		"exportedNamedInterfacesExportedMethodsPerExportedInterfaceType":       float64(stats.ExportedNamedInterfacesExportedMethods) / float64(stats.ExportedTypeNamesByKind[reflect.Interface]),

		"exportednoninterfacetypesByExportedmethodsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportednoninterfacetypes-by-exportedmethods"}, nil, ""),
		"exportedinterfacetypesByExportedmethodsChartURL":    buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedinterfacetypes-by-exportedmethods"}, nil, ""),
	}))

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, ds.currentTranslation.Text_StatisticsTitle("values"))
	page.WriteString(ds.currentTranslation.Text_ValueStatistics(map[string]interface{}{
		"exportedVariables": stats.ExportedVariables,
		"exportedConstants": stats.ExportedConstants,

		"exportedvariablesByTypekindsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedvariables-by-typekinds"}, nil, ""),
		"exportedconstantsByTypekindsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedconstants-by-typekinds"}, nil, ""),

		"exportedFunctions":                             stats.ExportedFunctions,
		"exportedMethods":                               stats.ExportedMethods,
		"averageParameterCountPerExportedFunction":      float64(stats.ExportedFunctionParameters) / float64(stats.ExportedFunctions+stats.ExportedMethods),
		"averageResultCountPerExportedFunction":         float64(stats.ExportedFunctionResults) / float64(stats.ExportedFunctions+stats.ExportedMethods),
		"exportedFunctionWithLastErrorResult":           stats.ExportedFunctionWithLastErrorResult,
		"exportedFunctionWithLastErrorResultPercentage": int(math.Round(100 * float64(stats.ExportedFunctionWithLastErrorResult) / float64(stats.ExportedFunctions+stats.ExportedMethods))),

		"exportedfunctionsByParametersChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedfunctions-by-parameters"}, nil, ""),
		"exportedfunctionsByResultsChartURL":    buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedfunctions-by-results"}, nil, ""),
	}))

	fmt.Fprintf(page, `<pre><code><span class="title">%s</span></code>`, ds.currentTranslation.Text_StatisticsTitle("others"))
	page.WriteString(ds.currentTranslation.Text_Othertatistics(map[string]interface{}{
		"averageIdentiferLength": float64(stats.ExportedIdentifersSumLength) / float64(stats.ExportedIdentifers),

		"exportedidentifiersByLengthsChartURL": buildPageHref(page.PathInfo, pagePathInfo{ResTypeSVG, "exportedidentifiers-by-lengths"}, nil, ""),
	}))

	return page.Done(ds.currentTranslation)
}
