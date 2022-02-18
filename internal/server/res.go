package server

import (
	"time"

	"golang.org/x/text/language"

	"go101.org/golds/code"
	theme "go101.org/golds/internal/server/themes"
	translation "go101.org/golds/internal/server/translations"
)

type Theme interface {
	Name() string
	CSS() string
}

type Translation interface {
	Name() string
	LangTag() string

	// common
	Text_Space() string
	Text_Comma() string
	Text_Colon(atLineEnd bool) string
	Text_Period(paragraphEnd bool) string
	Text_Parenthesis(close bool) string
	Text_EnclosedInOarentheses(text string) string
	Text_PreferredFontList() string

	// server
	Text_Server_Started() string

	// analyzing
	Text_Analyzing() string
	Text_AnalyzingRefresh(currentPageURL string) string // also used in other pages
	Text_Analyzing_Start() string
	Text_Analyzing_Done(d time.Duration, memoryUse string) string

	Text_Analyzing_PreparationDone(d time.Duration) string // ToDo: merge these into one?
	Text_Analyzing_NFilesParsed(numFiles int, d time.Duration) string
	Text_Analyzing_ParsePackagesDone(numFiles int, d time.Duration) string
	Text_Analyzing_CollectPackages(numMods int, d time.Duration) string
	Text_Analyzing_CollectModules(numPkgs int, d time.Duration) string
	Text_Analyzing_CollectExamples(d time.Duration) string
	Text_Analyzing_SortPackagesByDependencies(d time.Duration) string
	Text_Analyzing_CollectDeclarations(d time.Duration) string
	Text_Analyzing_CollectRuntimeFunctionPositions(d time.Duration) string
	Text_Analyzing_ConfirmTypeSources(d time.Duration) string
	Text_Analyzing_CollectSelectors(d time.Duration) string
	Text_Analyzing_FindImplementations(d time.Duration) string
	Text_Analyzing_RegisterInterfaceMethodsForTypes(d time.Duration) string
	Text_Analyzing_MakeStatistics(d time.Duration) string
	Text_Analyzing_CollectSourceFiles(d time.Duration) string
	Text_Analyzing_CollectObjectReferences(d time.Duration) string
	Text_Analyzing_CacheSourceFiles(d time.Duration) string

	// overview page
	Text_Overview() string
	Text_PackageList() string
	Text_StatisticsWithMoreLink(detailedStatsLink string) string
	Text_SimpleStats(stats *code.Stats) string
	Text_Modules() string                                    // to use
	Text_BelongingModule() string                            // to use
	Text_RequireStat(numRequires, numRequiredBys int) string // to use
	Text_UpdateTip(tipName string) string                    // tip names: "ToUpdate", "Updating", "Updated"

	Text_SortBy(whatToSort string) string // also used in other pages
	Text_SortByItem(by string) string     // also used in other pages

	// package details page
	Text_Package(pkgPath string) string
	Text_BelongingPackage() string // also used in source code page
	Text_PackageDocsLinksOnOtherWebsites(pkgPath string, isStdPkg bool) string
	Text_ImportPath() string
	Text_ImportStat(numImports, numImportedBys int, depPageURL string) string
	Text_InvolvedFiles(num int) string
	Text_Examples(num int) string
	Text_PackageLevelTypeNames() string
	Text_TypeParameters() string
	//Text_AllPackageLevelValues(num int) string
	Text_PackageLevelFunctions() string
	Text_PackageLevelVariables() string
	Text_PackageLevelConstants() string
	Text_PackageLevelResourceSimpleStat(statsAreExact bool, num, numExporteds int, mentionExporteds bool) string
	Text_UnexportedResourcesHeader(show bool, numUnexporteds int, exact bool) string
	Text_ListUnexportes() string

	Text_BasicType() string
	Text_Fields() string // ToDo: merge these into one?
	Text_Methods() string
	Text_ImplementedBy() string
	Text_Implements() string
	Text_AsOutputsOf() string
	Text_AsInputsOf() string
	Text_AsTypesOf() string

	// package dependencies page
	Text_DependencyRelations(pkgPath string) string // also used in package details page with a blank argument.
	Text_Imports() string
	Text_ImportedBy() string

	// method implementation page
	Text_MethodImplementations() string
	Text_NumMethodsImplementingNothing(count int) string
	Text_ViewMethodImplementations() string

	// object references(uses) page
	Text_ReferenceList() string
	Text_CurrentPackage() string
	Text_ObjectKind(kind string) string
	Text_ObjectUses(num int) string // also used in other pages

	// source code page
	Text_SourceCode(pkgPath, bareFilename string) string
	Text_SourceFilePath() string
	Text_GeneratedFrom() string

	// statistics
	Text_Statistics() string
	Text_ChartTitle(chartName string) string
	Text_StatisticsTitle(titleName string) string
	Text_PackageStatistics(values map[string]interface{}) []string
	Text_TypeStatistics(values map[string]interface{}) []string
	Text_ValueStatistics(values map[string]interface{}) []string
	Text_Othertatistics(values map[string]interface{}) []string

	// Footer
	Text_GeneratedPageFooter(goldsVersion, qrCodeLink, goOS, goArch string) string
	Text_GeneratedPageFooterSimple(goldsVersion, goOS, goArch string) string
}

func (ds *docServer) currentSettings() (Theme, Translation) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	return ds.currentTheme, ds.currentTranslation
}

func (ds *docServer) changeSettings(themeName string, langTags ...string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if themeName != "" {
		ds.currentTheme = ds.themeByName(themeName)
	}
	if len(langTags) > 0 {
		ds.currentTranslation = ds.translationByLangs(langTags...)
	}
}

func (ds *docServer) changeTranslationByAcceptLanguage(acceptedLanguage string) {
	langTags, _, _ := language.ParseAcceptLanguage(acceptedLanguage)
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	ds.currentTranslation = ds.translationByLangTags(langTags...)
}

// All themes and translations must be registered at init phase,
// so that no syncrhomization is needed.
func (ds *docServer) initSettings(lang string) {
	var (
		themes        = make([]Theme, 0, 2)
		translations  = make([]Translation, 0, 6)
		langTags      = make([]language.Tag, 0, len(translations)*2)
		translations2 = make([]Translation, 0, len(translations)*2)
	)

	registerTheme := func(theme Theme) {
		themes = append(themes, theme)
	}
	registerTranslation := func(tr Translation) {
		translations = append(translations, tr)
		tag := language.Make(tr.LangTag())
		langTags = append(langTags, tag)
		translations2 = append(translations2, tr)
	}

	registerTheme(&theme.Light{})

	registerTranslation(&translation.English{})
	registerTranslation(&translation.Chinese{})

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.allThemes = themes
	ds.allTranslations = translations
	ds.langMatcher = language.NewMatcher(langTags)
	ds.translationsByLangTagIndex = translations2

	ds.currentTheme = ds.allThemes[0]
	ds.currentTranslation = ds.allTranslations[0]
	ds.currentTranslation = ds.translationByLangs(lang)
}

func (ds *docServer) currentTranslationSafely() Translation {
	return ds.currentTranslation
}

func (ds *docServer) themeByName(name string) Theme {
	theme := ds.allThemes[0]
	for _, t := range ds.allThemes[1:] {
		if t.Name() == name {
			theme = t
			break
		}
	}
	return theme
}

func (ds *docServer) translationByName(name string) Translation {
	trans := ds.allTranslations[0]
	for _, tr := range ds.allTranslations[1:] {
		if tr.Name() == name {
			trans = tr
			break
		}
	}
	return trans
}

func (ds *docServer) translationByLangs(langs ...string) Translation {
	userPrefs := make([]language.Tag, 0, len(langs))
	for _, l := range langs {
		if l != "" {
			userPrefs = append(userPrefs, language.Make(l))
		}
	}
	return ds.translationByLangTags(userPrefs...)
}

func (ds *docServer) translationByLangTags(userPrefs ...language.Tag) Translation {
	if len(userPrefs) == 0 {
		return ds.currentTranslation
	}

	_, index, confidence := ds.langMatcher.Match(userPrefs...)
	if confidence == language.No {
		return ds.allTranslations[0]
	}
	return ds.translationsByLangTagIndex[index]
}
