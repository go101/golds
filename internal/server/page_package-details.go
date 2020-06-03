package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"go101.org/gold/code"
)

var _ = log.Print

func (ds *docServer) packageDetailsPage(w http.ResponseWriter, r *http.Request, pkgPath string) {
	w.Header().Set("Content-Type", "text/html")

	// ToDo: create a custom "builtin" package page.

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	if ds.packagePages[pkgPath] == nil {
		// ToDo: not found

		//details := ds.buildPackageDetailsData(pkgPath)
		details := buildPackageDetailsData(ds.analyzer, pkgPath)
		if details == nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Package (%s) not found", pkgPath)
			return
		}

		ds.packagePages[pkgPath] = ds.buildPackageDetailsPage(details)
	}
	w.Write(ds.packagePages[pkgPath])
}

func (ds *docServer) buildPackageDetailsPage(pkg *PackageDetails) []byte {
	page := NewHtmlPage(ds.goldVersion, ds.currentTranslation.Text_Package(pkg.ImportPath), ds.currentTheme.Name(), pagePathInfo{ResTypePackage, pkg.ImportPath})

	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">package <b>%s</b></span>
`,
		pkg.Name,
	)

	fmt.Fprintf(page, `
<span class="title">%s</span>
	<a href="%s#pkg-%s">%s</a>%s`,
		ds.currentTranslation.Text_ImportPath(),
		buildPageHref(page.PathInfo, pagePathInfo{ResTypeNone, ""}, nil, ""),
		pkg.ImportPath,
		pkg.ImportPath,
		ds.currentTranslation.Text_PackageDocsLinksOnOtherWebsites(pkg.ImportPath, pkg.IsStandard),
	)

	if pkg.ImportPath != "builtin" {
		fmt.Fprintf(page, `

<span class="title">%s</span>
	%s`,
			ds.currentTranslation.Text_DependencyRelations(""),
			//ds.currentTranslation.Text_ImportStat(int(pkg.NumDeps), int(pkg.NumDepedBys), "/dep:"+pkg.ImportPath),
			ds.currentTranslation.Text_ImportStat(int(pkg.NumDeps), int(pkg.NumDepedBys), buildPageHref(page.PathInfo, pagePathInfo{ResTypeDependency, pkg.ImportPath}, nil, "")),
		)
	}

	if len(pkg.Files) > 0 {
		fmt.Fprint(page, "\n\n", `<span class="title">`, ds.currentTranslation.Text_InvolvedFiles(len(pkg.Files)), `</span>`)

		for _, info := range pkg.Files {
			page.WriteString("\n\t")
			if info.HasDocs {
				ds.writeSourceCodeDocLink(page, pkg.Package, info.Filename)
			} else {
				page.WriteString("    ")
			}
			ds.writeSrouceCodeFileLink(page, pkg.Package, info.Filename)
		}
	}

	needOneMoreLine := false
	if len(pkg.ExportedTypeNames) == 0 {
		needOneMoreLine = true
		goto WriteValues
	}

	fmt.Fprint(page, "\n\n", `<span class="title">`, ds.currentTranslation.Text_ExportedTypeNames(len(pkg.ExportedTypeNames)), `</span>`)
	page.WriteByte('\n')
	for _, et := range pkg.ExportedTypeNames {
		page.WriteString("\n")
		fmt.Fprintf(page, `<div class="anchor" id="name-%s">`, et.TypeName.Name())
		page.WriteByte('\t')
		ds.writeResourceIndexHTML(page, et.TypeName, true, false)
		if doc := et.TypeName.Documentation(); doc != "" {
			page.WriteString("\n")
			writePageText(page, "\t\t", doc, true)
		}

		// ToDo: for alias, if its denoting type is an exported named type, then stop here.
		//       (might be not a good idea. 1. such cases are rare. 2. if they happen, it does need to list ...)

		page.WriteString("\n")
		if count := len(et.Fields); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "fields",
				ds.currentTranslation.Text_Fields(count),
				func() {
					fields := ds.sortFieldList(et.Fields)
					for _, fld := range fields {
						page.WriteString("\n\t\t\t")
						ds.writeFieldForListing(page, pkg.Package, fld, et.TypeName)
					}
				})
		}
		if count := len(et.Methods); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "methods",
				ds.currentTranslation.Text_Methods(count),
				func() {
					methods := ds.sortMethodList(et.Methods)
					for _, mthd := range methods {
						page.WriteString("\n\t\t\t")
						ds.writeMethodForListing(page, pkg.Package, mthd, et.TypeName)
					}
				})
		}
		if count := len(et.ImplementedBys); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "impledby",
				ds.currentTranslation.Text_ImplementedBy(count),
				func() {
					// ToDo: why not "pkg.ImportPath" instead of "et.TypeName"
					impledLys := ds.sortTypeList(et.ImplementedBys, pkg.Package)
					for _, by := range impledLys {
						page.WriteString("\n\t\t\t")
						writeTypeForListing(page, by, pkg.Package, "")
					}
				})
		}
		if count := len(et.Implements); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "impls",
				ds.currentTranslation.Text_Implements(count),
				func() {
					// ToDo: why not "pkg.ImportPath" instead of "et.TypeName"
					impls := ds.sortTypeList(et.Implements, pkg.Package)
					for _, impl := range impls {
						page.WriteString("\n\t\t\t")
						writeTypeForListing(page, impl, pkg.Package, et.TypeName.Name())
					}
				})
		}
		if count := len(et.AsOutputsOf); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "results",
				ds.currentTranslation.Text_AsOutputsOf(count),
				func() {
					values := ds.sortValueList(et.AsOutputsOf, pkg.Package)
					for _, v := range values {
						page.WriteString("\n\t\t\t")
						ds.writeValueForListing(page, v, pkg.Package, et.TypeName)
					}
				})
		}
		if count := len(et.AsInputsOf); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "params",
				ds.currentTranslation.Text_AsInputsOf(count),
				func() {
					values := ds.sortValueList(et.AsInputsOf, pkg.Package)
					for _, v := range values {
						page.WriteString("\n\t\t\t")
						ds.writeValueForListing(page, v, pkg.Package, et.TypeName)
					}
				})
		}
		if count := len(et.Values); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "values",
				ds.currentTranslation.Text_AsTypesOf(count),
				func() {
					values := ds.sortValueList(et.Values, pkg.Package)
					for _, v := range values {
						page.WriteString("\n\t\t\t")
						ds.writeValueForListing(page, v, pkg.Package, et.TypeName)
					}
				})
		}

		page.WriteString("</div>")
	}

WriteValues:
	if len(pkg.ValueResources) == 0 {
		goto Done
	}

	if needOneMoreLine {
		page.WriteByte('\n')
	}

	fmt.Fprint(page, "\n", `<span class="title">`, ds.currentTranslation.Text_ExportedValues(len(pkg.ValueResources)), `</span>`)
	page.WriteByte('\n')
	//fmt.Fprint(page, ` <input type="checkbox" id="consts" name="consts" value="constants"><label for="constants">const</label>`)
	//fmt.Fprint(page, `<input type="checkbox" id="vars" name="vars" value="variables"><label for="vars">var</label>`)
	//fmt.Fprint(page, `<input type="checkbox" id="funcs" name="funcs" value="functions"><label for="funcs">func</label>`)
	for _, v := range pkg.ValueResources {
		page.WriteByte('\n')
		fmt.Fprintf(page, `<div class="anchor" id="name-%s">`, v.Name())
		page.WriteByte('\t')
		ds.writeResourceIndexHTML(page, v, true, false)
		if doc := v.Documentation(); doc != "" {
			page.WriteString("\n")
			writePageText(page, "\t\t", doc, true)
		}
		page.WriteString("</div>")
	}

Done:
	page.WriteString("</code></pre>")
	return page.Done(ds.currentTranslation)
}

type FileInfo struct {
	Filename string
	HasDocs  bool
}

type PackageDetails struct {
	//PPkg *packages.Package
	//Mod  *Module
	//Info *PackageAnalyzeResult

	Package *code.Package

	IsStandard     bool
	Index          int
	Name           string
	ImportPath     string
	Files          []FileInfo
	ValueResources []code.ValueResource
	//ExportedTypeNames []*code.TypeName
	//UnexportedTypeNames []*code.TypeName
	ExportedTypeNames []*ExportedType

	// Line dismatches exist in some cgo generated files.
	//FileLineNumberOffsets map[string][]int

	NumDeps     uint32
	NumDepedBys uint32

	// ToDo: use go/doc
	//IntroductionCode template.HTML
}

type ExportedType struct {
	TypeName *code.TypeName
	Fields   []*code.Selector
	Methods  []*code.Selector
	//ImplementedBys []*code.TypeInfo
	//Implements     []code.Implementation
	ImplementedBys []TypeForListing
	Implements     []TypeForListing

	// ToDo: Now both implements and implementebys miss aliases to unnamed types.
	//       (And miss many unnamed types. Maybe it is good to automatically
	//       create some aliases for the unnamed types without explicit aliases)

	// All are in the current package.
	// (Nearby packages should also be checked? Module scope is better!)
	//Values []code.ValueResource
	Values []ValueForListing

	// Including functions/methods, and variables.
	// At present, only the values in the current package will be collected.
	// (Nearby packages should also be checked.)
	//
	// For non-interface types, all functions are declared in the current package.
	// For interface types (except error), may include functions in outside packages.
	// ToDo: collect outside ones at analyzing phase, or at page generation phase.
	//       Only the packages imported this package need to be checked.
	//       Packages importing the packages containing any alias of this type
	//       also need to be checked. (Also any types depending on this type?)
	//AsInputsOf  []code.ValueResource
	//AsOutputsOf []code.ValueResource
	AsInputsOf  []ValueForListing
	AsOutputsOf []ValueForListing
}

// ds should be locked before calling this method.
//func (ds *docServer) buildPackageDetailsData(pkgPath string) *PackageDetails {
func buildPackageDetailsData(analyzer *code.CodeAnalyzer, pkgPath string) *PackageDetails {
	pkg := analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil
	}

	//analyzer.BuildCgoFileMappings(pkg)

	isBuiltin := pkgPath == "builtin"

	// ...
	files := make([]FileInfo, 0, len(pkg.PPkg.GoFiles)+len(pkg.PPkg.OtherFiles))
	//lineStartOffsets := make(map[string][]int, len(pkg.PPkg.GoFiles))

	for i := range pkg.SourceFiles {
		info := &pkg.SourceFiles[i]
		if info.OriginalFile != "" {
			files = append(files, FileInfo{
				Filename: info.BareFilename,
				HasDocs:  info.AstFile != nil && info.AstFile.Doc != nil,
			})
		}

		//filePath := info.OriginalGoFile
		//if info.GeneratedFile != "" {
		//	filePath = info.GeneratedFile
		//}
		//content, err := ioutil.ReadFile(filePath)
		//if err != nil {
		//	log.Printf("read file (%s) error: %s", filePath, err)
		//} else {
		//	_, lineStartOffsets[info.OriginalGoFile] = BuildLineOffsets(content, false)
		//}
	}

	// Now, these file are also put into pkg.SourceFiles.
	//for _, path := range pkg.PPkg.OtherFiles {
	//	files = append(files, FileInfo{FilePath: path})
	//}

	// ...
	var valueResources = make([]code.ValueResource, 0,
		len(pkg.PackageAnalyzeResult.AllConstants)+
			len(pkg.PackageAnalyzeResult.AllVariables)+
			len(pkg.PackageAnalyzeResult.AllFunctions))
	for _, c := range pkg.PackageAnalyzeResult.AllConstants {
		if c.Exported() {
			valueResources = append(valueResources, c)
		}
	}
	for _, v := range pkg.PackageAnalyzeResult.AllVariables {
		if v.Exported() {
			valueResources = append(valueResources, v)
		}
	}
	for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
		if f.Exported() && !f.IsMethod() {
			valueResources = append(valueResources, f)
		}
	}
	sort.Slice(valueResources, func(i, j int) bool {
		// ToDo: cache lower names?
		return strings.ToLower(valueResources[i].Name()) < strings.ToLower(valueResources[j].Name())
	})

	//asTypesOf := make([]code.ValueResource, 256)
	//asParamsOf := make([]code.ValueResource, 256)
	//asResultsOf := make([]code.ValueResource, 256)
	//isType := func(tt types.Type, comparer *code.TypeInfo) bool {
	//	// only check T and *T
	//	t := analyzer.RegisterType(tt)
	//	if t == comparer {
	//		return true
	//	}
	//	if ptt, ok := tt.(*types.Pointer); ok {
	//		return analyzer.RegisterType(ptt.Elem()) == comparer
	//	}
	//	return false
	//}

	var exportedTypesResources = make([]*ExportedType, 0, len(pkg.PackageAnalyzeResult.AllTypeNames))
	//var unexportedTypesResources = make([]*code.TypeName, 0, len(pkg.PackageAnalyzeResult.AllTypeNames))
	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		if tn.Exported() {
			denoting := tn.Denoting()
			et := &ExportedType{TypeName: tn}
			exportedTypesResources = append(exportedTypesResources, et)

			// Generally, we don't collect info for a type alias, execpt it denotes an unnamed or unexported type.
			if tn.Alias != nil && tn.Alias.Denoting.TypeName != nil && tn.Alias.Denoting.TypeName.Exported() {
				continue
			}

			et.Fields = make([]*code.Selector, 0, len(denoting.AllFields))
			et.Methods = make([]*code.Selector, 0, len(denoting.AllMethods))
			//et.ImplementedBys = make([]*code.TypeInfo, 0, len(denoting.ImplementedBys))
			et.ImplementedBys = make([]TypeForListing, 0, len(denoting.ImplementedBys))
			//et.Implements = make([]code.Implementation, 0, len(denoting.Implements))
			et.Implements = make([]TypeForListing, 0, len(denoting.Implements))

			for _, mthd := range denoting.AllMethods {
				if token.IsExported(mthd.Name()) {
					et.Methods = append(et.Methods, mthd)
				}
			}
			for _, fld := range denoting.AllFields {
				if token.IsExported(fld.Name()) {
					et.Fields = append(et.Fields, fld)
				}
			}
			for _, impledBy := range denoting.ImplementedBys {
				bytn, isPointer := analyzer.RetrieveTypeName(impledBy)
				if bytn != nil && bytn != tn && bytn.Exported() {
					et.ImplementedBys = append(et.ImplementedBys, TypeForListing{
						TypeName:  bytn,
						IsPointer: isPointer,
					})
				}
			}
			for _, impl := range analyzer.CleanImplements(denoting) {
				//if impl.Interface.TypeName == nil || token.IsExported(impl.Interface.TypeName.Name()) {
				//	et.Implements = append(et.Implements, impl)
				//}
				// Might miss: interface {Unwrap() error}
				if itn := impl.Interface.TypeName; itn != nil && itn.Exported() {
					_, isPointer := impl.Impler.TT.(*types.Pointer)
					et.Implements = append(et.Implements, TypeForListing{
						TypeName:  itn,
						IsPointer: isPointer,
					})
				}
			}

			if isBuiltin {
				continue
			}

			// unexportedTypesResources = append(unexportedTypesResources, tn)

			/*
				asTypesOf, asParamsOf, asResultsOf = asTypesOf[:0], asParamsOf[:0], asResultsOf[:0]

				for _, res := range valueResources {
					if isType(res.TType(), denoting) {
						asTypesOf = append(asTypesOf, res)
					}
				}
				collectAsParamsAndAsResults := func(res code.ValueResource) {
					resTT := res.TType()
					if sig, ok := resTT.Underlying().(*types.Signature); ok {
						params, results := sig.Params(), sig.Results()
						for i := 0; i < params.Len(); i++ {
							param := params.At(i)
							if isType(param.Type(), denoting) {
								asParamsOf = append(asParamsOf, res)
								break
							}
						}
						for i := 0; i < results.Len(); i++ {
							result := results.At(i)
							if isType(result.Type(), denoting) {
								asResultsOf = append(asResultsOf, res)
								break
							}
						}
					}
				}
				for _, v := range pkg.PackageAnalyzeResult.AllVariables {
					if v.Exported() {
						collectAsParamsAndAsResults(v)
					}
				}
				for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
					if f.Exported() { //} && !f.IsMethod() {
						collectAsParamsAndAsResults(f)
					}
				}

				var nil []code.ValueResource
				et.Values = append(nil, asTypesOf...)
				et.AsInputsOf = append(nil, asParamsOf...)
				et.AsOutputsOf = append(nil, asResultsOf...)

				sort.Slice(et.AsInputsOf, func(i, j int) bool {
					// ToDo: cache lower names?
					return strings.ToLower(et.AsInputsOf[i].Name()) < strings.ToLower(et.AsInputsOf[j].Name())
				})

				sort.Slice(et.AsOutputsOf, func(i, j int) bool {
					// ToDo: cache lower names?
					return strings.ToLower(et.AsOutputsOf[i].Name()) < strings.ToLower(et.AsOutputsOf[j].Name())
				})
			*/

			//var nil []code.ValueResource
			//et.Values = append(nil, denoting.AsTypesOf...)
			//et.AsInputsOf = append(nil, denoting.AsInputsOf...)
			//et.AsOutputsOf = append(nil, denoting.AsOutputsOf...)

			//et.Values = buildValueList(denoting.AsTypesOf)
			et.AsInputsOf = buildValueList(denoting.AsInputsOf)
			et.AsOutputsOf = buildValueList(denoting.AsOutputsOf)

			var values []code.ValueResource
			values = append(values, denoting.AsTypesOf...)
			// ToDo: also combine values of []T, chan T, ...
			if t := analyzer.TryRegisteringType(types.NewPointer(denoting.TT), false); t != nil {
				values = append(values, t.AsTypesOf...)
			}
			et.Values = buildValueList(values)
		}
	}
	sort.Slice(exportedTypesResources, func(i, j int) bool {
		// ToDo: cache lower names?
		return strings.ToLower(exportedTypesResources[i].TypeName.Name()) < strings.ToLower(exportedTypesResources[j].TypeName.Name())
	})
	//sort.Slice(unexportedTypesResources, func(i, j int) bool {
	//	// ToDo: cache lower names?
	//	return strings.ToLower(unexportedTypesResources[i].Name()) < strings.ToLower(unexportedTypesResources[j].Name())
	//})

	// ...
	return &PackageDetails{
		//PPkg: pkg.PPkg,
		//Mod:  pkg.Mod,
		//Info: pkg.PackageAnalyzeResult,

		Package: pkg,

		IsStandard:        analyzer.IsStandardPackage(pkg),
		Index:             pkg.Index,
		Name:              pkg.PPkg.Name,
		ImportPath:        pkg.PPkg.PkgPath,
		Files:             files,
		ValueResources:    valueResources,
		ExportedTypeNames: exportedTypesResources,
		//UnexportedTypeNames: unexportedTypesResources,

		//FileLineNumberOffsets: lineStartOffsets,

		NumDeps:     uint32(len(pkg.Deps)),
		NumDepedBys: uint32(len(pkg.DepedBys)),
	}
}

type ValueForListing struct {
	code.ValueResource
	InCurrentPkg bool
	CommonPath   string
}

func buildValueList(values []code.ValueResource) []ValueForListing {
	listedValues := make([]ValueForListing, len(values))
	for i := range listedValues {
		lv := &listedValues[i]
		lv.ValueResource = values[i]
	}
	return listedValues
}

// The implementations sortValueList and sortTypeList are some reapetitive.
// Need generic.? (Or let ValueForListing and TypeForListing implement the same interface)
func (ds *docServer) sortValueList(valueList []ValueForListing, pkg *code.Package) []*ValueForListing {
	result := make([]*ValueForListing, len(valueList))

	pkgPath := pkg.Path()
	for i := range valueList {
		v := &valueList[i]
		result[i] = v
		v.InCurrentPkg = v.Package() == pkg
		if !v.InCurrentPkg {
			v.CommonPath = FindPackageCommonPrefixPaths(v.Package().Path(), pkgPath)
		}
	}

	compareWithoutPackges := func(a, b *ValueForListing) bool {
		fa, oka := a.ValueResource.(*code.Function)
		fb, okb := b.ValueResource.(*code.Function)
		if oka && okb {
			if p, q := fa.IsMethod(), fb.IsMethod(); p && q {
				_, ida, _ := fa.ReceiverTypeName()
				_, idb, _ := fb.ReceiverTypeName()
				if r := strings.Compare(strings.ToLower(ida.Name), strings.ToLower(idb.Name)); r != 0 {
					return r < 0
				}
			} else if p != q {
				return q
			}
		}
		return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
	}

	sort.Slice(result, func(a, b int) bool {
		if x, y := result[a].InCurrentPkg, result[b].InCurrentPkg; x || y {
			if x && y {
				return compareWithoutPackges(result[a], result[b])
			}
			return x
		}
		commonA, commonB := result[a].CommonPath, result[b].CommonPath
		if len(commonA) != len(commonB) {
			if len(commonA) == len(pkgPath) {
				return true
			}
			if len(commonB) == len(pkgPath) {
				return false
			}
			if len(commonA) > 0 || len(commonB) > 0 {
				return len(commonA) > len(commonB)
			}
		}
		r := strings.Compare(strings.ToLower(result[a].Package().Path()), strings.ToLower(result[b].Package().Path()))
		if r == 0 {
			return compareWithoutPackges(result[a], result[b])
		}
		if result[a].Package().Path() == "builtin" {
			return true
		}
		if result[b].Package().Path() == "builtin" {
			return false
		}
		return r < 0
	})

	return result
}

// The funciton is some repeatitive with writeResourceIndexHTML.
//func (ds *docServer) writeValueForListing(page *htmlPage, v *ValueForListing, pkg *code.Package, fileLineOffsets map[string][]int, forTypeName *code.TypeName) {
func (ds *docServer) writeValueForListing(page *htmlPage, v *ValueForListing, pkg *code.Package, forTypeName *code.TypeName) {
	pos := v.Position()
	//if lineOffsets, ok := fileLineOffsets[pos.Filename]; ok {
	//	correctPosition(lineOffsets, &pos)
	//} else {
	//	pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	//}

	//log.Println("   :", pos)

	switch res := v.ValueResource.(type) {
	default:
		panic("should not")
	case *code.Constant, *code.Variable:
		if _, ok := v.ValueResource.(*code.Constant); ok {
			fmt.Fprint(page, `const `)
		} else {
			fmt.Fprint(page, `  var `)
		}

		if v.Package() != pkg {
			//if v.Package().Path() != "builtin" {
			page.WriteString(v.Package().Path())
			page.WriteByte('.')
			//}
			fmt.Fprintf(page, `<a href="`)
			//page.WriteString("/pkg:")
			//page.WriteString(v.Package().Path())
			buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, "")
		} else {
			fmt.Fprintf(page, `<a href="`)
		}
		page.WriteString("#name-")
		page.WriteString(v.Name())
		fmt.Fprintf(page, `">`)
		page.WriteString(v.Name())
		page.WriteString("</a>")

		if t := res.TypeInfo(ds.analyzer); t != forTypeName.Denoting() {
			page.WriteByte(' ')
			//page.WriteString(res.TType().String())
			specOwner := res.(code.AstValueSpecOwner)
			if astType := specOwner.AstValueSpec().Type; astType != nil {
				ds.WriteAstType(page, astType, specOwner.Package(), specOwner.Package(), false, nil, forTypeName)
			} else {
				// ToDo: track to get the AstType and use WriteAstType instead.
				ds.writeValueTType(page, res.TType(), specOwner.Package(), true, forTypeName)
			}
		}
	case *code.Function:

		page.WriteString("func ")
		if vpkg := v.Package(); vpkg != pkg {
			if vpkg != nil {
				page.WriteString(v.Package().Path())
				page.WriteString(".")
			}
		}

		if res.IsMethod() {
			recvParam, typeId, isStar := res.ReceiverTypeName()
			if isStar {
				if v.Package() != pkg {
					//fmt.Fprintf(page, `(*<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>).`, v.Package().Path(), typeId.Name)
					page.WriteString("(*")
					buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, typeId.Name, "name-", typeId.Name)
					page.WriteString(").")
				} else {
					fmt.Fprintf(page, `(*<a href="#name-%[1]s">%[1]s</a>).`, typeId.Name)
				}
				//fmt.Fprintf(page, "(*%s) ", typeId.Name)
			} else {
				if v.Package() != pkg {
					//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>.`, v.Package().Path(), typeId.Name)
					buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, typeId.Name, "name-", typeId.Name)
				} else {
					fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>.`, typeId.Name)
				}
				//fmt.Fprintf(page, "(%s) ", typeId.Name)
			}

			ds.writeSrouceCodeLineLink(page, v.Package(), pos, v.Name(), "", false)

			//ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, pkg, false, recvParam, forTypeName)
			ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, pkg, false, nil, forTypeName)
			_ = recvParam
		} else {
			if v.Package() != pkg {
				//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>`, v.Package().Path(), v.Name())
				buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, v.Name(), "name-", v.Name())
			} else {
				fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>`, v.Name())
			}

			ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, pkg, false, nil, forTypeName)
		}
	}
}

type TypeForListing struct {
	*code.TypeName
	IsPointer    bool
	InCurrentPkg bool
	CommonPath   string
}

// Assume all types are named or pointer to named.
func (ds *docServer) sortTypeList(typeList []TypeForListing, pkg *code.Package) []*TypeForListing {
	result := make([]*TypeForListing, len(typeList))

	pkgPath := pkg.Path()
	for i := range typeList {
		t := &typeList[i]
		result[i] = t
		t.InCurrentPkg = t.Package() == pkg
		if !t.InCurrentPkg {
			t.CommonPath = FindPackageCommonPrefixPaths(t.Package().Path(), pkgPath)
		}
	}

	sort.Slice(result, func(a, b int) bool {
		if x, y := result[a].InCurrentPkg, result[b].InCurrentPkg; x || y {
			if x && y {
				return strings.ToLower(result[a].Name()) < strings.ToLower(result[b].Name())
			}
			return x
		}
		commonA, commonB := result[a].CommonPath, result[b].CommonPath
		if len(commonA) != len(commonB) {
			if len(commonA) == len(pkgPath) {
				return true
			}
			if len(commonB) == len(pkgPath) {
				return false
			}
			if len(commonA) > 0 || len(commonB) > 0 {
				return len(commonA) > len(commonB)
			}
		}
		pathA, pathB := strings.ToLower(result[a].Pkg.Path()), strings.ToLower(result[b].Pkg.Path())
		r := strings.Compare(pathA, pathB)
		if r == 0 {
			return strings.ToLower(result[a].Name()) < strings.ToLower(result[b].Name())
		}
		if pathA == "builtin" {
			return true
		}
		if pathB == "builtin" {
			return false
		}
		return r < 0
	})

	return result
}

func writeTypeForListing(page *htmlPage, t *TypeForListing, pkg *code.Package, implerName string) {
	if implerName == "" {
		if t.IsPointer {
			page.WriteByte('*')
		} else {
			page.WriteByte(' ')
		}
	} else {
		if t.IsPointer {
			page.WriteString("*T : ")
			//fmt.Fprintf(page, "*%s : ", implerName)
		} else {
			page.WriteString(" T : ")
			//fmt.Fprintf(page, " %s : ", implerName)
		}
	}
	if t.Package() != pkg {
		if t.Pkg.Path() != "builtin" {
			page.WriteString(t.Pkg.Path())
			page.WriteByte('.')
		}
		fmt.Fprintf(page, `<a href="`)
		//page.WriteString("/pkg:")
		//page.WriteString(t.Pkg.Path())
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, t.Pkg.Path()}, page, "")
	} else {
		fmt.Fprintf(page, `<a href="`)
	}
	page.WriteString("#name-")
	page.WriteString(t.Name())
	fmt.Fprintf(page, `">`)
	page.WriteString(t.Name())
	page.WriteString("</a>")
}

type FieldForListing struct {
	*code.Selector
	Middles []*code.Field

	numDuplicatedMiddlesWithLast int
}

func (ds *docServer) sortFieldList(selectors []*code.Selector) []*FieldForListing {
	selList := make([]FieldForListing, len(selectors))
	result := make([]*FieldForListing, len(selectors))
	for i, sel := range selectors {
		selForListing := &selList[i]
		result[i] = selForListing
		selForListing.Selector = sel
		if sel.Depth > 0 {
			selForListing.Middles = make([]*code.Field, sel.Depth)
			chain := sel.EmbeddingChain
			for k := int(sel.Depth) - 1; k >= 0; k-- {
				//log.Println(sel.Depth, k, chain)
				selForListing.Middles[k] = chain.Field
				chain = chain.Prev
			}
		}
	}

	sort.Slice(result, func(a, b int) bool {
		sa, sb := result[a], result[b]
		k := len(sa.Middles)
		if k > len(sb.Middles) {
			k = len(sb.Middles)
		}
		for i := 0; i < k; i++ {
			switch strings.Compare(strings.ToLower(sa.Middles[i].Name), strings.ToLower(sb.Middles[i].Name)) {
			case -1:
				return true
			case 1:
				return false
			}
		}
		if len(sa.Middles) < len(sb.Middles) {
			switch strings.Compare(strings.ToLower(sa.Name()), strings.ToLower(sb.Middles[k].Name)) {
			case 0, -1:
				return true
			case 1:
				return false
			}
		}
		if len(sa.Middles) > len(sb.Middles) {
			switch strings.Compare(strings.ToLower(sa.Middles[k].Name), strings.ToLower(sb.Name())) {
			case 0, 1:
				return false
			case -1:
				return true
			}
		}
		return sa.Name() < sb.Name()
	})

	for i := 1; i < len(result); i++ {
		last := result[i-1]
		sel := result[i]
		i, k := 0, len(last.Middles)
		if k > len(sel.Middles) {
			k = len(sel.Middles)
		}
		for ; i < k; i++ {
			if last.Middles[i].Name != sel.Middles[i].Name {
				break
			}
		}
		if len(last.Middles) < len(sel.Middles) {
			if last.Name() == sel.Middles[i].Name {
				i++
			}
		}
		sel.numDuplicatedMiddlesWithLast = i
	}

	return result
}

func (ds *docServer) sortMethodList(selectors []*code.Selector) []*code.Selector {
	sort.Slice(selectors, func(a, b int) bool {
		return selectors[a].Name() < selectors[b].Name()
	})
	return selectors
}

func (ds *docServer) writeFieldForListing(page *htmlPage, pkg *code.Package, sel *FieldForListing, forTypeName *code.TypeName) {
	for i, fld := range sel.Middles {
		pos := fld.Position()
		//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
		class := ""
		if i < sel.numDuplicatedMiddlesWithLast {
			class = "path-duplicate"
		}
		if token.IsExported(fld.Name) {
			ds.writeSrouceCodeLineLink(page, fld.Pkg, pos, fld.Name, class, false)
			page.WriteByte('.')
		} else {
			//ds.writeSrouceCodeLineLink(page, fld.Pkg, pos, "<strike>"+fld.Name+"</strike>", class, false)
			//page.WriteString("<strike>.</strike>")
			page.WriteString("<i>")
			ds.writeSrouceCodeLineLink(page, fld.Pkg, pos, fld.Name, class, false)
			page.WriteString(".</i>")
		}
	}
	selField := sel.Field
	if selField == nil {
		panic("should not")
	}
	pos := sel.Position()
	//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	ds.writeSrouceCodeLineLink(page, sel.Pkg(), pos, selField.Name, "", false)
	page.WriteByte(' ')
	ds.WriteAstType(page, selField.AstField.Type, selField.Pkg, pkg, true, nil, forTypeName)
}

func (ds *docServer) writeMethodForListing(page *htmlPage, pkg *code.Package, sel *code.Selector, forTypeName *code.TypeName) {
	setMethod := sel.Method
	if setMethod == nil {
		panic("should not")
	}
	if sel.PointerReceiverOnly() {
		page.WriteString("(*T) ")
	} else {
		page.WriteString(" (T) ")
	}
	pos := sel.Position()
	//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	ds.writeSrouceCodeLineLink(page, sel.Pkg(), pos, setMethod.Name, "", false)
	if setMethod.AstFunc != nil {
		ds.WriteAstType(page, setMethod.AstFunc.Type, setMethod.Pkg, pkg, false, nil, forTypeName)
	} else {
		ds.WriteAstType(page, setMethod.AstField.Type, setMethod.Pkg, pkg, false, nil, forTypeName)
	}
}

func writeKindText(page *htmlPage, tt types.Type) {
	var kind string
	var bold = false

	switch tt.Underlying().(type) {
	default:
		return
	case *types.Pointer:
		kind = "*Type"
	case *types.Struct:
		kind = reflect.Struct.String()
	case *types.Array:
		kind = "[...]"
	case *types.Slice:
		kind = "[]"
	case *types.Map:
		kind = reflect.Map.String()
	case *types.Chan:
		kind = reflect.Chan.String()
	case *types.Signature:
		kind = reflect.Func.String()
	case *types.Interface:
		kind = reflect.Interface.String()
		bold = true
	}

	if bold {
		fmt.Fprintf(page, ` <b><i>(%s)</i></b>`, kind)
	} else {
		fmt.Fprintf(page, ` <i>(%s)</i>`, kind)
	}
}

//func (ds *docServer) writeResourceIndexHTML(page *htmlPage, res code.Resource, fileLineOffsets map[string][]int, writeType, writeReceiver bool) {
func (ds *docServer) writeResourceIndexHTML(page *htmlPage, res code.Resource, writeType, writeReceiver bool) {
	pos := res.Position()
	//if lineOffsets, ok := fileLineOffsets[pos.Filename]; ok {
	//	correctPosition(lineOffsets, &pos)
	//} else {
	//	// For many reasons, line offset tables for the files
	//	// outside of the current package are not avaliable at this time.
	//	// * link to methods or fields which are obtained through embedding
	//	// * link to items in asTypesOf/asInputsOf/asOutputsOf lists.
	//	//
	//	// The way might cause line number inaccuracy.
	//	pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	//	// ToDo: maybe it is acceptable to eventually load involved files
	//	//       and calculate/cache their line offset tables.
	//	//       (It would be best if the std ast parser could support an option
	//	//       to turn off line-repositions.)
	//}

	//log.Println("   :", pos)

	isBuiltin := res.Package().Path() == "builtin"

	switch res := res.(type) {
	default:
		panic("should not")
	case *code.TypeName:
		page.WriteString(" type ")
		if isBuiltin {
			page.WriteString(res.Name())
		} else {
			ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		}

		if writeType {
			showSource := false
			if isBuiltin {
				// builtin package source code are fake.
				showSource = res.Alias != nil
			} else {
				allowStar := res.Alias != nil
				for t, done := res.AstSpec.Type, false; !done; {
					switch e := t.(type) {
					case *ast.Ident, *ast.SelectorExpr:
						showSource = true
						done = true
					case *ast.ParenExpr:
						t = e.X
					case *ast.StarExpr:
						// type A = *T
						if allowStar {
							t = e.X
							allowStar = false
						} else {
							done = true
						}
					default:
						done = true
					}
				}
			}

			if res.Alias != nil {
				page.WriteByte(' ')
				page.WriteByte('=')
			}

			//page.WriteString(types.TypeString(res.Denoting().TT, types.RelativeTo(res.Package().PPkg.Types)))
			if res.AstSpec.Type == nil {
				panic("res.Alias != nil, but res.AstSpec.Type == nil, ???")
			}

			if showSource {
				page.WriteByte(' ')
				ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, true, nil, nil)
				//ds.writeValueTType(page, res.Denoting().TT, res.Pkg, true, nil)
			}
			writeKindText(page, res.Denoting().TT)
		}
	case *code.Constant:
		page.WriteString("const ")
		if isBuiltin {
			page.WriteString(res.Name())
		} else {
			ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		}

		btt, ok := res.TType().Underlying().(*types.Basic)
		if !ok {
			panic("constants should be always of basic types, but " + res.String() + " : " + res.TType().String())
		}
		if writeType && btt.Info()&types.IsUntyped == 0 {
			page.WriteByte(' ')
			//page.WriteString(types.TypeString(res.TType(), types.RelativeTo(res.Package().PPkg.Types)))
			if res.AstSpec.Type != nil {
				ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, false, nil, nil)
			} else {
				ds.writeValueTType(page, res.TType(), res.Pkg, true, nil)
			}
		}
		if !isBuiltin {
			page.WriteString(" = ")
			page.WriteString(res.Val().String())
		}
	case *code.Variable:
		page.WriteString("  var ")
		if isBuiltin {
			page.WriteString(res.Name())
		} else {
			ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		}
		if writeType {
			page.WriteByte(' ')
			//page.WriteString(res.TType().String())
			if res.AstSpec.Type != nil {
				ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, false, nil, nil)
			} else {
				// ToDo: track to get the AstType and use WriteAstType instead.
				ds.writeValueTType(page, res.TType(), res.Pkg, true, nil)
			}
		}
	case *code.Function:
		var recv *types.Var
		if writeReceiver && res.Func != nil {
			sig := res.Func.Type().(*types.Signature)
			recv = sig.Recv()
		}
		page.WriteString(" func ")
		// This if-block will be never entered now.
		if recv != nil {
			switch tt := recv.Type().(type) {
			case *types.Named:
				fmt.Fprintf(page, `(%s) `, tt.Obj().Name())
			case *types.Pointer:
				if named, ok := tt.Elem().(*types.Named); ok {
					fmt.Fprintf(page, `(*%s) `, named.Obj().Name())
				} else {
					panic("should not")
				}
			default:
				panic("should not")
			}
		}
		if isBuiltin {
			page.WriteString(res.Name())
			// ToDo: link panic/recover/... to their implementation positions.
		} else {
			ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		}
		if writeType {
			ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, res.Pkg, false, nil, nil)
			//ds.writeValueTType(page, res.TType(), res.Pkg, false)
		}
	}

	if comment := res.Comment(); comment != "" {
		page.WriteString(" // ")
		writePageText(page, "", comment, true)
	}

	//fmt.Fprint(page, ` <a href="#">{/}</a>`)
}

func (ds *docServer) writeTypeName(page *htmlPage, tt *types.Named, docPkg *code.Package, alternativeTypeName string) {
	objpkg := tt.Obj().Pkg()
	isBuiltin := objpkg == nil
	if isBuiltin {
		objpkg = ds.analyzer.BuiltinPackge().PPkg.Types
	} else if objpkg != docPkg.PPkg.Types {
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, objpkg.Path()}, page, objpkg.Name())
		page.Write(period)
	}
	ttName := alternativeTypeName
	if ttName == "" {
		ttName = tt.Obj().Name()
	}
	//page.WriteString(tt.Obj().Name())
	if isBuiltin || tt.Obj().Exported() {
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, objpkg.Path()}, page, ttName, "name-", tt.Obj().Name())
	} else {
		p := ds.analyzer.PackageByPath(objpkg.Path())
		if p == nil {
			panic("should not")
		}
		ttPos := p.PPkg.Fset.PositionFor(tt.Obj().Pos(), false)
		//log.Printf("============ %v, %v, %v", tt, pkg.Path(), ttPos)
		ds.writeSrouceCodeLineLink(page, p, ttPos, ttName, "", false)
	}

}

func (ds *docServer) writeValueTType(page *htmlPage, tt types.Type, docPkg *code.Package, writeFuncKeyword bool, forTypeName *code.TypeName) {
	switch tt := tt.(type) {
	default:
		panic("should not")
	case *types.Named:
		if forTypeName != nil && tt == forTypeName.Denoting().TT {
			page.WriteString(tt.Obj().Name())
		} else {
			ds.writeTypeName(page, tt, docPkg, "")
		}
	case *types.Basic:
		if forTypeName != nil && tt == forTypeName.Denoting().TT {
			page.WriteString(tt.Name())
		} else {
			buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, "builtin"}, page, tt.Name(), "name-", tt.Name())
		}
	case *types.Pointer:
		page.Write(star)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Array:
		page.Write(leftSquare)
		fmt.Fprintf(page, "%d", tt.Len())
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Slice:
		page.Write(leftSquare)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Map:
		page.Write(mapKeyword)
		page.Write(leftSquare)
		ds.writeValueTType(page, tt.Key(), docPkg, true, forTypeName)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Chan:
		if tt.Dir() == types.RecvOnly {
			page.Write(chanDir)
			page.Write(chanKeyword)
		} else if tt.Dir() == types.SendOnly {
			page.Write(chanKeyword)
			page.Write(chanDir)
		} else {
			page.Write(chanKeyword)
		}
		page.Write(space)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Signature:
		if writeFuncKeyword {
			page.Write(funcKeyword)
			//page.Write(space)
		}
		page.Write(leftParen)
		ds.writeTuple(page, tt.Params(), docPkg, tt.Variadic(), forTypeName)
		page.Write(rightParen)
		if rs := tt.Results(); rs != nil && rs.Len() > 0 {
			page.Write(space)
			if rs.Len() == 1 && rs.At(0).Name() == "" {
				ds.writeTuple(page, rs, docPkg, false, forTypeName)
			} else {
				page.Write(leftParen)
				ds.writeTuple(page, rs, docPkg, false, forTypeName)
				page.Write(rightParen)
			}
		}
	case *types.Struct:
		page.Write(structKeyword)
		//page.Write(space)
		page.Write(leftBrace)
		ds.writeStructFields(page, tt, docPkg, forTypeName)
		page.Write(rightBrace)
	case *types.Interface:
		page.Write(interfaceKeyword)
		//page.Write(space)
		page.Write(leftBrace)
		ds.writeInterfaceMethods(page, tt, docPkg, forTypeName)
		page.Write(rightBrace)
	}
}

func (ds *docServer) writeTuple(page *htmlPage, tuple *types.Tuple, docPkg *code.Package, variadic bool, forTypeName *code.TypeName) {
	n := tuple.Len()
	for i := 0; i < n; i++ {
		v := tuple.At(i)
		if v.Name() != "" {
			page.WriteString(v.Name())
			page.WriteByte(' ')
		}
		if i == n-1 {
			if variadic {
				st, ok := v.Type().(*types.Slice)
				if !ok {
					panic("should not")
				}
				page.WriteString("...")
				ds.writeValueTType(page, st.Elem(), docPkg, true, forTypeName)
			} else {
				ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
			}
		} else {
			ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
			page.WriteString(", ")
		}
	}
}

func (ds *docServer) writeStructFields(page *htmlPage, st *types.Struct, docPkg *code.Package, forTypeName *code.TypeName) {
	n := st.NumFields()
	for i := 0; i < n; i++ {
		v := st.Field(i)
		if v.Embedded() {
			// ToDo: try to find ast representation of the types of all variables.
			//       Otherwise, the embedded interface type aliases info are lost.
			// This is a suboptimal implementaiuon.
			if tn, ok := v.Type().(*types.Named); ok {
				ds.writeTypeName(page, tn, docPkg, v.Name())
			} else {
				page.WriteString(v.Name())
			}
		} else {
			page.WriteString(v.Name())
			page.WriteByte(' ')
			ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
		}
		if i < n-1 {
			page.WriteString("; ")
		}
	}
}

func (ds *docServer) writeInterfaceMethods(page *htmlPage, it *types.Interface, docPkg *code.Package, forTypeName *code.TypeName) {
	//n, m := it.NumEmbeddeds(), it.NumExplicitMethods()
	//
	//for i := 0; i < m; i++ {
	//	f := it.ExplicitMethod(i)
	//	page.WriteString(f.Name())
	//	//page.WriteByte(' ')
	//	ds.writeValueTType(page, f.Type(), docPkg, false)
	//	if i < m-1 {
	//		page.WriteString("; ")
	//	}
	//}
	//if n > 0 && m > 0 {
	//	page.WriteString("; ")
	//}
	//
	//for i := 0; i < n; i++ {
	//	named := it.Embedded(i)
	//	ds.writeValueTType(page, named.Obj().Type(), docPkg, false)
	//	if i < n-1 {
	//		page.WriteString("; ")
	//	}
	//}

	// ToDo: try to find ast representation of the types of all variables.
	//       Otherwise, the embedded interface type aliases info are lost.
	// This is a suboptimal implementaiuon.
	var k = it.NumMethods()
	for i := 0; i < k; i++ {
		f := it.Method(i)
		page.WriteString(f.Name())
		//page.WriteByte(' ')
		ds.writeValueTType(page, f.Type(), docPkg, false, forTypeName)
		if i < k-1 {
			page.WriteString("; ")
		}
	}
}

var (
	blankID          = []byte("_")
	space            = []byte(" ")
	leftParen        = []byte("(")
	rightParen       = []byte(")")
	period           = []byte(".")
	comma            = []byte(", ")
	semicoloon       = []byte("; ")
	ellipsis         = []byte("...")
	star             = []byte("*")
	leftSquare       = []byte("[")
	rightSquare      = []byte("]")
	leftBrace        = []byte("{")
	rightBrace       = []byte("}")
	mapKeyword       = []byte("map")
	chanKeyword      = []byte("chan")
	chanDir          = []byte("&lt;-")
	funcKeyword      = []byte("func")
	structKeyword    = []byte("struct")
	interfaceKeyword = []byte("interface")

	BoldTagStart = []byte("<b>")
	BoldTagEnd   = []byte("</b>")
)

// This is a rewritten of WriteTypeEx.
// Please make sure w.Write never makes errors.
// "forTypeName", if it is not blank, should be declared in docPkg.
// ToDo: "too many fields/methods/params/results" is replaced with ".....".
func (ds *docServer) WriteAstType(w *htmlPage, typeLit ast.Expr, codePkg, docPkg *code.Package, funcKeywordNeeded bool, recvParam *ast.Field, forTypeName *code.TypeName) {
	switch node := typeLit.(type) {
	default:
		panic(fmt.Sprint("WriteType, unknown node: ", node))
	case *ast.ParenExpr:
		w.Write(leftParen)
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightParen)
	case *ast.Ident:
		// obj := codePkg.PPkg.TypesInfo.ObjectOf(node)
		// The above one might return a *types.Var object for embedding field.
		// So us the following one instead, to make sure it is a *types.TypeName.
		obj := codePkg.PPkg.Types.Scope().Lookup(node.Name)
		if obj == nil {
			obj = types.Universe.Lookup(node.Name)
		}
		if obj == nil {
			//log.Printf("%s, %s: %s", docPkg.Path(), codePkg.Path(), node.Name)
			//panic("should not")

			// It really should panic here, but to make it tolerable,

			w.WriteString(node.Name)

			return
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			panic("object should be a TypeName")
		}
		objpkg := obj.Pkg()
		isBuiltin := objpkg == nil
		if isBuiltin {
			objpkg = ds.analyzer.BuiltinPackge().PPkg.Types
		} else if objpkg != docPkg.PPkg.Types {
			buildPageHref(w.PathInfo, pagePathInfo{ResTypePackage, objpkg.Path()}, w, objpkg.Name())
			w.Write(period)
		}

		if forTypeName != nil && types.Identical(tn.Type(), forTypeName.Denoting().TT) {
			w.Write(BoldTagStart)
			defer w.Write(BoldTagEnd)
		}

		if objpkg == docPkg.PPkg.Types && forTypeName != nil && node.Name == forTypeName.Name() {
			w.WriteString(node.Name)
		} else if docPkg.Path() == "builtin" {
			if obj.Exported() { // like Type
				w.WriteString(node.Name)
			} else { // like int
				buildPageHref(w.PathInfo, pagePathInfo{ResTypePackage, objpkg.Path()}, w, node.Name, "name-", node.Name)
			}
		} else if isBuiltin || obj.Exported() {
			buildPageHref(w.PathInfo, pagePathInfo{ResTypePackage, objpkg.Path()}, w, node.Name, "name-", node.Name)
		} else {
			p := ds.analyzer.PackageByPath(objpkg.Path())
			if p == nil {
				panic("should not")
			}
			ttPos := p.PPkg.Fset.PositionFor(obj.Pos(), false)
			//log.Printf("============ %v, %v, %v", tt, pkg.Path(), ttPos)
			ds.writeSrouceCodeLineLink(w, p, ttPos, node.Name, "", false)
		}
	case *ast.SelectorExpr:
		pkgId, ok := node.X.(*ast.Ident)
		if !ok {
			panic("should not")
		}
		importobj := codePkg.PPkg.TypesInfo.ObjectOf(pkgId)
		if importobj == nil {
			panic("should not")
		}
		pkgobj := importobj.(*types.PkgName)
		if pkgobj == nil {
			panic("should not")
		}
		pkgpkg := pkgobj.Imported()
		if pkgpkg == nil {
			panic("should not")
		}
		if pkgpkg != docPkg.PPkg.Types {
			//w.WriteString(pkgpkg.Name())
			buildPageHref(w.PathInfo, pagePathInfo{ResTypePackage, pkgpkg.Path()}, w, pkgId.Name)
			w.Write(period)
		}

		//log.Println(pkgId.Name, node.Sel.Name, pkgpkg.Path(), codePkg.Path())
		obj := pkgpkg.Scope().Lookup(node.Sel.Name)
		if obj.Pkg() != pkgpkg {
			//panic("should not")

			// It really should panic here, but to make it tolerable,

			w.WriteString(node.Sel.Name)

			return
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			panic(fmt.Sprintf("%v is a %T, not a type name", obj, obj))
		}

		if forTypeName != nil && types.Identical(tn.Type(), forTypeName.Denoting().TT) {
			w.Write(BoldTagStart)
			defer w.Write(BoldTagEnd)
		}

		if pkgpkg == docPkg.PPkg.Types && forTypeName != nil && node.Sel.Name == forTypeName.Name() {
			w.WriteString(node.Sel.Name)
		} else if obj.Exported() { // || isBuiltin { // must not be builtin
			buildPageHref(w.PathInfo, pagePathInfo{ResTypePackage, pkgpkg.Path()}, w, node.Sel.Name, "name-", node.Sel.Name)
		} else {
			//w.WriteString(node.Sel.Name)
			p := ds.analyzer.PackageByPath(pkgpkg.Path())
			if p == nil {
				panic("should not")
			}
			ttPos := p.PPkg.Fset.PositionFor(obj.Pos(), false)
			//log.Printf("============ %v, %v, %v", tt, pkg.Path(), ttPos)
			ds.writeSrouceCodeLineLink(w, p, ttPos, node.Sel.Name, "", false)
		}
	case *ast.StarExpr:
		w.Write(star)
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
	case *ast.Ellipsis: // possible? (yes, variadic parameters)
		//panic("[...] should be impossible") // ToDo: go/types package has a case.
		//w.Write(leftSquare)
		w.Write(ellipsis)
		//w.Write(rightSquare)
		ds.WriteAstType(w, node.Elt, codePkg, docPkg, true, nil, forTypeName)
	case *ast.ArrayType:
		w.Write(leftSquare)
		if node.Len != nil {
			tv, ok := codePkg.PPkg.TypesInfo.Types[node.Len]
			if !ok {
				panic(fmt.Sprint("no values found for ", node.Len))
			}
			w.WriteString(tv.Value.String())
		}
		w.Write(rightSquare)
		ds.WriteAstType(w, node.Elt, codePkg, docPkg, true, nil, forTypeName)
	case *ast.MapType:
		w.Write(mapKeyword)
		w.Write(leftSquare)
		ds.WriteAstType(w, node.Key, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightSquare)
		ds.WriteAstType(w, node.Value, codePkg, docPkg, true, nil, forTypeName)
	case *ast.ChanType:
		if node.Dir == ast.RECV {
			w.Write(chanDir)
			w.Write(chanKeyword)
		} else if node.Dir == ast.SEND {
			w.Write(chanKeyword)
			w.Write(chanDir)
		} else {
			w.Write(chanKeyword)
		}
		w.Write(space)
		ds.WriteAstType(w, node.Value, codePkg, docPkg, true, nil, forTypeName)
	case *ast.FuncType:
		if funcKeywordNeeded {
			w.Write(funcKeyword)
			//w.Write(space)
		}
		w.Write(leftParen)
		ds.WriteAstFieldList(w, node.Params, true, comma, codePkg, docPkg, true, recvParam, forTypeName)
		w.Write(rightParen)
		if node.Results != nil && len(node.Results.List) > 0 {
			w.Write(space)
			if len(node.Results.List) == 1 && len(node.Results.List[0].Names) == 0 {
				ds.WriteAstFieldList(w, node.Results, true, comma, codePkg, docPkg, true, nil, forTypeName)
			} else {
				w.Write(leftParen)
				ds.WriteAstFieldList(w, node.Results, true, comma, codePkg, docPkg, true, nil, forTypeName)
				w.Write(rightParen)
			}
		}
	case *ast.StructType:
		w.Write(structKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		ds.WriteAstFieldList(w, node.Fields, false, semicoloon, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightBrace)
	case *ast.InterfaceType:
		w.Write(interfaceKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		ds.WriteAstFieldList(w, node.Methods, false, semicoloon, codePkg, docPkg, false, nil, forTypeName)
		w.Write(rightBrace)
	}
}

func (ds *docServer) WriteAstFieldList(w *htmlPage, fieldList *ast.FieldList, isParamOrResultList bool, sep []byte, codePkg, docPkg *code.Package, funcKeywordNeeded bool, recvParam *ast.Field, forTypeName *code.TypeName) {
	if fieldList == nil {
		return
	}
	showRecvName := recvParam != nil && len(recvParam.Names) > 0
	showParamsNames := isParamOrResultList && len(fieldList.List) > 0 && len(fieldList.List[0].Names) > 0
	showParamsNames = showParamsNames || showRecvName

	fields := fieldList.List
	if recvParam != nil {
		fields = append([]*ast.Field{recvParam}, fields...)
	}

	for i, fld := range fields {
		if len(fld.Names) > 0 {
			for k, n := range fld.Names {
				w.Write([]byte(n.Name))
				if k+1 < len(fld.Names) {
					w.Write(comma)
				}
			}
			if funcKeywordNeeded {
				w.Write(space)
			} // else for interface methods
		} else if showParamsNames {
			w.Write(blankID)
			if funcKeywordNeeded {
				w.Write(space)
			} // else for interface methods
		}
		ds.WriteAstType(w, fld.Type, codePkg, docPkg, funcKeywordNeeded, nil, forTypeName)
		if i+1 < len(fields) {
			w.Write(sep)
		}
	}
}

//var basicKind2ReflectKind = [...]reflect.Kind{
//	types.Bool:          reflect.Bool,
//	types.Int:           reflect.Int,
//	types.Int8:          reflect.Int8,
//	types.Int16:         reflect.Int16,
//	types.Int32:         reflect.Int32,
//	types.Int64:         reflect.Int64,
//	types.Uint:          reflect.Uint,
//	types.Uint8:         reflect.Uint8,
//	types.Uint16:        reflect.Uint16,
//	types.Uint32:        reflect.Uint32,
//	types.Uint64:        reflect.Uint64,
//	types.Uintptr:       reflect.Uintptr,
//	types.Float32:       reflect.Float32,
//	types.Float64:       reflect.Float64,
//	types.Complex64:     reflect.Complex64,
//	types.Complex128:    reflect.Complex128,
//	types.String:        reflect.String,
//	types.UnsafePointer: reflect.UnsafePointer,
//}

// Only write interface
//func writeTypeKind(page *htmlPage, tt types.Type) {
//	switch tt := tt.Underlying().(type) {
//	default:
//		panic(fmt.Sprintf("should not: %T", tt))
//	case *types.Named:
//		panic("should not")
//	case *types.Basic:
//		page.WriteString(basicKind2ReflectKind[tt.Kind()].String())
//	case *types.Pointer:
//		page.WriteString(reflect.Ptr.String())
//	case *types.Struct:
//		page.WriteString(reflect.Struct.String())
//	case *types.Array:
//		page.WriteString(reflect.Array.String())
//	case *types.Slice:
//		page.WriteString(reflect.Slice.String())
//	case *types.Map:
//		page.WriteString(reflect.Map.String())
//	case *types.Chan:
//		page.WriteString(reflect.Chan.String())
//	case *types.Signature:
//		page.WriteString(reflect.Func.String())
//	case *types.Interface:
//		page.WriteString(reflect.Interface.String())
//	}
//}

//func (c *Constant) IndexString() string {
//	btt, ok := c.Type().(*types.Basic)
//	if !ok {
//		panic("constant should be always of basic type")
//	}
//	isTyped := btt.Info()&types.IsUntyped == 0
//
//	var b strings.Builder
//
//	b.WriteString(c.Name())
//	if isTyped {
//		b.WriteByte(' ')
//		b.WriteString(c.Type().String())
//	}
//	b.WriteString(" = ")
//	b.WriteString(c.Val().String())
//
//	return b.String()
//}

//func writeTypeNameIndexHTML(page *htmlPage, tn *code.TypeName)  {
//	fmt.Fprintf(page, ` type <a href="#name-%[1]s">%[1]s</a>`, tn.Name())
//}

// onclickCode should use single quotes in it.
func writeNamedStatTitle(page *htmlPage, resName, statName, statTitle string, listStatContent func()) {
	fmt.Fprintf(page, `<input type='checkbox' class="stat" id="%[1]s-stat-%[2]s"><label for="%[1]s-stat-%[2]s">%[3]s</label><span id='%[1]s-stat-%[2]s-content' class="stat-content">`,
		resName, statName, statTitle)
	listStatContent()
	page.WriteString("</span>")
}

func writePageText(page *htmlPage, indent, text string, htmlEscape bool) {
	buffer := bytes.NewBufferString(text)
	reader := bufio.NewReader(buffer)
	notFirstLine, needAddMissingNewLine := false, false
	for {
		if needAddMissingNewLine {
			page.WriteByte('\n')
		}
		line, isPrefix, err := reader.ReadLine()
		if len(line) > 0 {
			if notFirstLine {
				page.WriteByte('\n')
			}
			page.WriteString(indent)
			needAddMissingNewLine = false
		} else {
			needAddMissingNewLine = true
		}
		if htmlEscape {
			WriteHtmlEscapedBytes(page, line)
		} else {
			page.Write(line)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if !isPrefix {
			notFirstLine = true
		}
	}
}
