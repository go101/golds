package translation

import "fmt"

type English struct{}

func (*English) Name() string { return "English" }

func (*English) LangTag() string { return "en" }

///////////////////////////////////////////////////////////////////
// overview page
///////////////////////////////////////////////////////////////////

func (*English) Text_Overview() string { return "Overview" }

func (*English) Text_PackageList(num int) string {
	return fmt.Sprintf("All Packages (%d)", num)
}

func (*English) Text_AllStats(stats map[string]int) string {
	return "ToDo" // how many interfaces, ...
}

func (*English) Text_Modules() string { return "Modules" }

func (*English) Text_BelongingModule() string { return "Belonging Module" }

func (*English) Text_RequireStat(numRequires, numRequiredBys int) string {
	return fmt.Sprintf("requires %d modules, and required by %d.", numRequires, numRequiredBys)
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
		return fmt.Sprintf(` (on <a href="https://pkg.go.dev/%[1]s" target="_blank">go.dev</a> and <a href="https://golang.org/pkg/%[1]s/" target="_blank">golang.org</a>)`, pkgPath)
	} else {
		return fmt.Sprintf(` (on <a href="https://pkg.go.dev/%s" target="_blank">go.dev</a>)`, pkgPath)
	}
}

func (*English) Text_ImportPath() string { return "Import Path" }

func (*English) Text_ImportStat(numImports, numImportedBys int, depPageURL string) string {
	var importsStr, importedBysStr string

	if numImports == 1 {
		importsStr = "1 package"
	} else {
		importsStr = fmt.Sprintf("%d packages", numImports)
	}
	if numImports > 0 {
		importsStr = fmt.Sprintf(`<a href="%s">%s</a>`, depPageURL, importsStr)
	}

	if numImportedBys == 1 {
		importedBysStr = "1 package"
	} else {
		importedBysStr = fmt.Sprintf("%d packages", numImportedBys)
	}
	if numImportedBys > 0 {
		importedBysStr = fmt.Sprintf(`<a href="%s#imported-by">%s</a>`, depPageURL, importedBysStr)
	}

	return fmt.Sprintf(`imports %s, and imported by %s`, importsStr, importedBysStr)
}

func (*English) Text_InvolvedFiles(num int) string { return "Involved Files" }

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
