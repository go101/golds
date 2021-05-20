package code

import (
	"reflect"
)

const KindCount = reflect.UnsafePointer + 1

// A Stats hold all the analysis statistics data.
type Stats struct {
	Packages            int32
	StdPackages         int32
	AllPackageDeps      int32
	PackagesByDeps      [100]int32
	PackagesDepsTopList TopList

	FilesWithoutGenerateds int32 // without generated ones
	FilesWithGenerateds    int32 // with generated ones

	// To calculate imports per file.
	// Deps per packages are available in other ways.
	AstFiles                int32
	Imports                 int32
	FilesByImportCount      [100]int32
	FilesImportCountTopList TopList

	CodeLinesWithBlankLines           int32
	FilesByCodeLinesWithBlankLines    [100]int32
	FilesCodeLineTopList              TopList
	PackagesByCodeLinesWithBlankLines [100]int32
	PackagesCodeLineTopList           TopList

	// Types
	ExportedTypeNamesByKind    [KindCount]int32
	ExportedTypeNames          int32
	ExportedTypeAliases        int32
	ExportedCompositeTypeNames int32
	ExportedBasicTypeNames     int32
	ExportedNumericTypeNames   int32
	ExportedIntergerTypeNames  int32
	ExportedUnsignedTypeNames  int32

	// This records the count of all package-level declared types.
	// However,
	// * types without methods should not be counted.
	// * some local declared types with methods should also be counted.
	roughTypeNameCount int32
	// This recoreds the count of all package-level declared exported identifiers,
	// exported field/method names, and fake identifiers (#HashOrderID) for unnamed types.
	roughExportedIdentifierCount int32

	//ExportedNamedStructTypeNames                        int32 // should be equal to ExportedTypeNamesByKind[reflect.Struct]
	ExportedNamedStructTypesWithEmbeddingFields   int32
	ExportedNamedStructTypesWithPromotedFields    int32
	ExportedNamedStructTypeFields                 int32
	ExportedNamedStructTypeExplicitFields         int32
	ExportedNamedStructTypeExportedFields         int32
	ExportedNamedStructTypeExportedExplicitFields int32

	ExportedNamedStructsByEmbeddingFieldCount        [100]int32
	ExportedNamedStructsEmbeddingFieldCountTopList   TopList
	ExportedNamedStructsByFieldCount                 [100]int32 // including promoteds and non-exporteds
	ExportedNamedStructsFieldCountTopList            TopList
	ExportedNamedStructsByExplicitFieldCount         [100]int32 // including non-exporteds but not including promoted
	ExportedNamedStructsExplicitFieldCountTopList    TopList
	ExportedNamedStructsByExportedFieldCount         [100]int32 // including promoteds
	ExportedNamedStructsExportedFieldCountTopList    TopList
	ExportedNamedStructsByExportedExplicitFieldCount [100]int32 // not including promoteds
	ExportedNamedStructsExportedExplicitFieldCount   TopList
	ExportedNamedStructsByExportedPromotedFieldCount [100]int32
	ExportedNamedStructsExportedPromotedFieldCount   TopList

	ExportedNamedNonInterfaceTypesByMethodCount              [100]int32 // T and *T combined
	ExportedNamedNonInterfaceTypesByExportedMethodCount      [100]int32 // T and *T combined
	ExportedNamedNonInterfaceTypesExportedMethodCountTopList TopList
	ExportedNamedNonInterfacesExportedMethods                int32
	ExportedNamedNonInterfacesWithExportedMethods            int32

	ExportedNamedInterfacesByMethodCount              [100]int32
	ExportedNamedInterfacesByExportedMethodCount      [100]int32 // the last element means (N-1)+
	ExportedNamedInterfacesExportedMethodCountTopList TopList
	ExportedNamedInterfacesExportedMethods            int32

	// Values
	ExportedVariables int32
	ExportedConstants int32
	ExportedFunctions int32
	ExportedMethods   int32 // non-interface methods

	ExportedVariablesByTypeKind [KindCount]int32
	ExportedConstantsByTypeKind [KindCount]int32

	// ToDo: Methods corresponding the same interface method should be viewed as one method in counting.
	ExportedFunctionParameters             int32      // including methods
	ExportedFunctionResults                int32      // including methods
	ExportedFunctionWithLastErrorResult    int32      // including methods
	ExportedFunctionsByParameterCount      [100]int32 // including methods
	ExportedFunctionsParameterCountTopList TopList
	ExportedFunctionsByResultCount         [100]int32 // including methods
	ExportedFunctionsResultCountTopList    TopList

	// Others.
	ExportedIdentifers             int32
	ExportedIdentifersSumLength    int32
	ExportedIdentifiersByLength    [100]int32
	ExportedIdentiferLengthTopList TopList
}

// A TopList specifies the minimu criteria for a top list
// and hold the top list items.
type TopList struct {
	Criteria int
	Items    []interface{}
}

// TryToInit inits a TopList if it has not been.
func (tl *TopList) TryToInit(n int) {
	if tl.Criteria == 0 {
		if n <= 0 {
			panic("shoould not")
		}
		tl.Criteria = n
		tl.Items = make([]interface{}, 0, 4)
	}
}

// Push trys to add a new top item.
func (tl *TopList) Push(n int, obj interface{}) {
	if n < tl.Criteria {
		return
	}
	if n > tl.Criteria {
		tl.Criteria = n
		tl.Items = tl.Items[:0]
	}
	tl.Items = append(tl.Items, obj)
}

func incSliceStat(stats []int32, index int) {
	if index >= len(stats) {
		stats[len(stats)-1]++
	} else {
		stats[index]++
	}
}

// Statistics returns the analysis statistics data.
func (d *CodeAnalyzer) Statistics() Stats {
	return d.stats
}

// RoughTypeNameCount returns a rough number of all type names.
func (d *CodeAnalyzer) RoughTypeNameCount() int32 {
	return d.stats.roughTypeNameCount
}

// RoughExportedIdentifierCount returns a rough number of exported identifiers.
func (d *CodeAnalyzer) RoughExportedIdentifierCount() int32 {
	return d.stats.roughExportedIdentifierCount
}

func (d *CodeAnalyzer) stat_OnNewPackage(std bool, numSrcFiles, numDeps int, pkgPath string) {
	if std {
		d.stats.StdPackages++
	}
	d.stats.FilesWithGenerateds += int32(numSrcFiles)
	d.stats.AllPackageDeps += int32(numDeps)
	incSliceStat(d.stats.PackagesByDeps[:], numDeps)

	d.stats.PackagesDepsTopList.TryToInit(28)
	d.stats.PackagesDepsTopList.Push(numDeps, &pkgPath)
}

func (d *CodeAnalyzer) stat_OnNewAstFile(numImports, linesWithBlanks int, bareFileName string, pkg *Package) {
	d.stats.AstFiles++
	d.stats.Imports += int32(numImports)
	incSliceStat(d.stats.FilesByImportCount[:], numImports)

	pkgFile := &struct {
		*Package
		Filename string
	}{pkg, bareFileName}
	d.stats.FilesImportCountTopList.TryToInit(16)
	d.stats.FilesImportCountTopList.Push(numImports, pkgFile)

	// ...
	numHundreds := linesWithBlanks / 100
	incSliceStat(d.stats.FilesByCodeLinesWithBlankLines[:], numHundreds)
	d.stats.FilesCodeLineTopList.TryToInit(20) // 2,000 lines
	d.stats.FilesCodeLineTopList.Push(numHundreds, pkgFile)
}

func (d *CodeAnalyzer) stat_OnPackageCodeLineCount(linesWithBlanks int, pkg *Package) {
	pkg.CodeLinesWithBlankLines += int32(linesWithBlanks)
	d.stats.CodeLinesWithBlankLines += int32(linesWithBlanks)
	numThousands := linesWithBlanks / 1000
	incSliceStat(d.stats.PackagesByCodeLinesWithBlankLines[:], numThousands)
	d.stats.PackagesCodeLineTopList.TryToInit(20) // 20,000 lines
	d.stats.PackagesCodeLineTopList.Push(numThousands, pkg)
}

func (d *CodeAnalyzer) stat_OnNewExportedNonInterfaceTypeNames(numAllMethods, numExportedMethods int, tn interface{}) {
	incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByMethodCount[:], numAllMethods)
	incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByExportedMethodCount[:], numExportedMethods)

	d.stats.ExportedNamedNonInterfaceTypesExportedMethodCountTopList.TryToInit(26)
	d.stats.ExportedNamedNonInterfaceTypesExportedMethodCountTopList.Push(numExportedMethods, tn)
}

func (d *CodeAnalyzer) stat_OnNewExportedInterfaceTypeNames(numAllMethods, numExportedMethods int, tn interface{}) {
	incSliceStat(d.stats.ExportedNamedInterfacesByMethodCount[:], numAllMethods)
	incSliceStat(d.stats.ExportedNamedInterfacesByExportedMethodCount[:], numExportedMethods)
	d.stats.ExportedNamedInterfacesExportedMethods += int32(numExportedMethods)

	d.stats.ExportedNamedInterfacesExportedMethodCountTopList.TryToInit(9)
	d.stats.ExportedNamedInterfacesExportedMethodCountTopList.Push(numExportedMethods, tn)
}

func (d *CodeAnalyzer) stat_OnNewExportedStructTypeName(hasEmbeddeds bool, numAllFields, numEmbeddingFields, numExpliciteds, numExporteds, numExportedExpliciteds, numExportedPromoteds int, tn interface{}) {
	if hasEmbeddeds {
		d.stats.ExportedNamedStructTypesWithPromotedFields++
	}
	if numEmbeddingFields > 0 {
		d.stats.ExportedNamedStructTypesWithEmbeddingFields++
	}
	incSliceStat(d.stats.ExportedNamedStructsByEmbeddingFieldCount[:], numEmbeddingFields)

	incSliceStat(d.stats.ExportedNamedStructsByExplicitFieldCount[:], numExpliciteds)
	d.stats.ExportedNamedStructTypeExplicitFields += int32(numExpliciteds)
	incSliceStat(d.stats.ExportedNamedStructsByExportedFieldCount[:], numExporteds)
	d.stats.ExportedNamedStructTypeExportedFields += int32(numExporteds)
	incSliceStat(d.stats.ExportedNamedStructsByExportedExplicitFieldCount[:], numExportedExpliciteds)
	d.stats.ExportedNamedStructTypeExportedExplicitFields += int32(numExportedExpliciteds)
	d.stats.roughExportedIdentifierCount += int32(numExportedExpliciteds)

	incSliceStat(d.stats.ExportedNamedStructsByExportedPromotedFieldCount[:], numExportedPromoteds)

	incSliceStat(d.stats.ExportedNamedStructsByFieldCount[:], numAllFields)
	d.stats.ExportedNamedStructTypeFields += int32(numAllFields)

	d.stats.ExportedNamedStructsEmbeddingFieldCountTopList.TryToInit(3)
	d.stats.ExportedNamedStructsEmbeddingFieldCountTopList.Push(numEmbeddingFields, tn)
	d.stats.ExportedNamedStructsFieldCountTopList.TryToInit(32)
	d.stats.ExportedNamedStructsFieldCountTopList.Push(numAllFields, tn)
	d.stats.ExportedNamedStructsExplicitFieldCountTopList.TryToInit(32)
	d.stats.ExportedNamedStructsExplicitFieldCountTopList.Push(numExpliciteds, tn)
	d.stats.ExportedNamedStructsExportedFieldCountTopList.TryToInit(32)
	d.stats.ExportedNamedStructsExportedFieldCountTopList.Push(numExporteds, tn)
	d.stats.ExportedNamedStructsExportedExplicitFieldCount.TryToInit(30)
	d.stats.ExportedNamedStructsExportedExplicitFieldCount.Push(numExportedExpliciteds, tn)
	d.stats.ExportedNamedStructsExportedPromotedFieldCount.TryToInit(16)
	d.stats.ExportedNamedStructsExportedPromotedFieldCount.Push(numExportedPromoteds, tn)
}

func (d *CodeAnalyzer) stat_OnNewExportedFunction(numInputs, numOutputs int, f *Function) {
	d.stats.ExportedFunctionParameters += int32(numInputs)
	d.stats.ExportedFunctionResults += int32(numOutputs)
	incSliceStat(d.stats.ExportedFunctionsByParameterCount[:], numInputs)
	incSliceStat(d.stats.ExportedFunctionsByResultCount[:], numOutputs)

	d.stats.ExportedFunctionsParameterCountTopList.TryToInit(9)
	d.stats.ExportedFunctionsParameterCountTopList.Push(numInputs, f)
	d.stats.ExportedFunctionsResultCountTopList.TryToInit(4)
	d.stats.ExportedFunctionsResultCountTopList.Push(numOutputs, f)
}

func (d *CodeAnalyzer) stat_OnNewExportedIdentifer(length int, obj interface{}) {
	incSliceStat(d.stats.ExportedIdentifiersByLength[:], length)
	d.stats.ExportedIdentifersSumLength += int32(length)
	d.stats.ExportedIdentifers++

	d.stats.ExportedIdentiferLengthTopList.TryToInit(32)
	d.stats.ExportedIdentiferLengthTopList.Push(length, obj)
}
