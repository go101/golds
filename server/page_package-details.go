package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"

	//"go/ast"
	"go/token"
	"go/types"
	"io"

	"log"
	"net/http"
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
		ds.loadingPage(w, r)
		return
	}

	if ds.packagePages[pkgPath] == nil {
		// ToDo: not found

		details := ds.buildPackageDetailsData(pkgPath)
		if details == nil {
			fmt.Fprintf(w, "Package (%s) not found", pkgPath)
			return
		}

		ds.packagePages[pkgPath] = ds.buildPackageDetailsPage(details)
	}
	w.Write(ds.packagePages[pkgPath])
}

func (ds *docServer) buildPackageDetailsPage(pkg *PackageDetails) []byte {
	page := NewHtmlPage(ds.currentTranslation.Text_Package(pkg.ImportPath), ds.currentTheme.Name(), pagePathInfo{ResTypePackage, pkg.ImportPath})

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
		ds.writeResourceIndexHTML(page, et.TypeName, pkg.FileLineNumberOffsets, true, false)
		if doc := et.TypeName.Documentation(); doc != "" {
			page.WriteString("\n")
			writePageText(page, "\t\t", doc, true)
		}
		page.WriteString("\n")
		if count := len(et.Fields); count > 0 {
			page.WriteString("\n\t\t")
			writeNamedStatTitle(page, et.TypeName.Name(), "fields",
				ds.currentTranslation.Text_Fields(count),
				func() {
					fields := ds.sortFieldList(et.Fields)
					for _, fld := range fields {
						page.WriteString("\n\t\t\t")
						ds.writeFieldForListing(page, fld)
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
						ds.writeMethodForListing(page, mthd)
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
						writeTypeForListing(page, by, pkg.Package, true)
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
						writeTypeForListing(page, impl, pkg.Package, false)
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
						ds.writeValueForListing(page, v, pkg.Package, pkg.FileLineNumberOffsets, et.TypeName.Name())
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
						ds.writeValueForListing(page, v, pkg.Package, pkg.FileLineNumberOffsets, et.TypeName.Name())
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
						ds.writeValueForListing(page, v, pkg.Package, pkg.FileLineNumberOffsets, et.TypeName.Name())
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
		ds.writeResourceIndexHTML(page, v, pkg.FileLineNumberOffsets, true, false)
		if doc := v.Documentation(); doc != "" {
			page.WriteString("\n")
			writePageText(page, "\t\t", doc, true)
		}
		page.WriteString("</div>")
	}

Done:
	page.WriteString("</code></pre>")
	return page.Done()
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
	FileLineNumberOffsets map[string][]int

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
func (ds *docServer) buildPackageDetailsData(pkgPath string) *PackageDetails {
	pkg, ok := ds.analyzer.PackageByPath(pkgPath)
	if !ok || pkg == nil {
		return nil
	}

	//ds.analyzer.BuildCgoFileMappings(pkg)

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
	//	t := ds.analyzer.RegisterType(tt)
	//	if t == comparer {
	//		return true
	//	}
	//	if ptt, ok := tt.(*types.Pointer); ok {
	//		return ds.analyzer.RegisterType(ptt.Elem()) == comparer
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
				bytn, isPointer := ds.analyzer.RetrieveTypeName(impledBy)
				if bytn != nil && bytn != tn && bytn.Exported() {
					et.ImplementedBys = append(et.ImplementedBys, TypeForListing{
						TypeName:  bytn,
						IsPointer: isPointer,
					})
				}
			}
			for _, impl := range ds.analyzer.CleanImplements(denoting) {
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
			et.Values = buildValueList(denoting.AsTypesOf)
			et.AsInputsOf = buildValueList(denoting.AsInputsOf)
			et.AsOutputsOf = buildValueList(denoting.AsOutputsOf)

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

		IsStandard:        ds.analyzer.IsStandardPackage(pkg),
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
func (ds *docServer) writeValueForListing(page *htmlPage, v *ValueForListing, pkg *code.Package, fileLineOffsets map[string][]int, forTypeName string) {
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
	case *code.Function:
		page.WriteString(" func ")

		if v.Package() != pkg {
			//if v.Package().Path() != "builtin" {
			page.WriteString(v.Package().Path())
			page.WriteByte('.')
		}
		lvi := &ListedValueInfo{
			codePkg:     v.Package(),
			docPkg:      pkg,
			forTypeName: forTypeName,
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
			} else {
				if v.Package() != pkg {
					//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>.`, v.Package().Path(), typeId.Name)
					buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, typeId.Name, "name-", typeId.Name)
				} else {
					fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>.`, typeId.Name)
				}
			}

			ds.writeSrouceCodeLineLink(page, v.Package(), pos, v.Name(), "", false)

			WriteTypeEx(page, res.AstDecl.Type, res.Pkg.PPkg.TypesInfo, false, recvParam, lvi)
		} else {
			if v.Package() != pkg {
				//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>`, v.Package().Path(), v.Name())
				buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, v.Package().Path()}, page, v.Name(), "name-", v.Name())
			} else {
				fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>`, v.Name())
			}

			WriteTypeEx(page, res.AstDecl.Type, res.Pkg.PPkg.TypesInfo, false, nil, lvi)
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

func writeTypeForListing(page *htmlPage, t *TypeForListing, pkg *code.Package, isImpler bool) {
	if isImpler {
		if t.IsPointer {
			page.WriteByte('*')
		} else {
			page.WriteByte(' ')
		}
	} else { // is an interface being implemented
		if t.IsPointer {
			page.WriteString("*T : ")
		} else {
			page.WriteString(" T : ")
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

func (ds *docServer) writeFieldForListing(page *htmlPage, sel *FieldForListing) {
	for i, fld := range sel.Middles {
		pos := fld.Position()
		//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
		class := ""
		if i < sel.numDuplicatedMiddlesWithLast {
			class = "path-duplicate"
		}
		ds.writeSrouceCodeLineLink(page, fld.Pkg, pos, fld.Name, class, false)
		page.WriteByte('.')
	}
	selField := sel.Field
	if selField == nil {
		panic("should not")
	}
	pos := sel.Position()
	//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	ds.writeSrouceCodeLineLink(page, sel.Pkg(), pos, selField.Name, "", false)
	page.WriteByte(' ')
	WriteType(page, selField.AstField.Type, selField.Pkg.PPkg.TypesInfo, true)
}

func (ds *docServer) writeMethodForListing(page *htmlPage, sel *code.Selector) {
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
		WriteType(page, setMethod.AstFunc.Type, setMethod.Pkg.PPkg.TypesInfo, false)
	} else {
		WriteType(page, setMethod.AstField.Type, setMethod.Pkg.PPkg.TypesInfo, false)
	}
}

func (ds *docServer) writeResourceIndexHTML(page *htmlPage, res code.Resource, fileLineOffsets map[string][]int, writeType, writeReceiver bool) {
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

	switch res := res.(type) {
	default:
		panic("should not")
	case *code.TypeName:
		page.WriteString(" type ")
		ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		// ToDo: also wirte non-alais src type.
		// ToDo: use a custom formatter to avoid multiple line and too long src type strings.
		if writeType {
			writeInterfaceText := func(tt types.Type) {
				if _, ok := tt.Underlying().(*types.Interface); ok {
					page.WriteString(` <i>(interface)</i>`)
				}
			}

			if res.Alias != nil {
				page.WriteString(" = ")
				page.WriteString(types.TypeString(res.Denoting().TT, types.RelativeTo(res.Package().PPkg.Types)))
				if ttn, ok := res.Denoting().TT.(*types.Named); ok {
					writeInterfaceText(ttn)
				}
			} else {
				writeInterfaceText(res.Named.TT)
			}
		}
	case *code.Constant:
		page.WriteString("const ")
		ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)

		btt, ok := res.TType().Underlying().(*types.Basic)
		if !ok {
			panic("constants should be always of basic types, but " + res.String() + " : " + res.TType().String())
		}
		if writeType && btt.Info()&types.IsUntyped == 0 {
			page.WriteByte(' ')
			//page.WriteString(types.TypeString(res.TType(), types.RelativeTo(res.Package().PPkg.Types)))
			ds.writeValueTType(page, res.TType(), true)
			// ToDo: use the ast exp instead (if avaliable)
		}
		page.WriteString(" = ")
		page.WriteString(res.Val().String())
	case *code.Variable:
		page.WriteString("  var ")
		ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		if writeType {
			page.WriteByte(' ')
			//page.WriteString(res.TType().String())
			ds.writeValueTType(page, res.TType(), true)
			// ToDo: use the ast exp instead (if avaliable)
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
		ds.writeSrouceCodeLineLink(page, res.Package(), pos, res.Name(), "", false)
		if writeType {
			//WriteType(page, res.AstDecl.Type, res.Pkg.PPkg.TypesInfo, false)
			ds.writeValueTType(page, res.TType(), false)
			// ToDo: use the ast exp instead
		}
	}

	if comment := res.Comment(); comment != "" {
		page.WriteString(" // ")
		writePageText(page, "", comment, true)
	}

	//fmt.Fprint(page, ` <a href="#">{/}</a>`)
}

// ToDo: need to be a method?
func (ds *docServer) writeValueTType(page *htmlPage, tt types.Type, writeFuncKeyword bool) {
	switch tt := tt.(type) {
	default:
		panic("should not")
	case *types.Named:
		pkg := tt.Obj().Pkg()
		isBuiltin := false
		if pkg == nil {
			//log.Printf("======================= %v", tt)
			// must be the builtin error type
			p, _ := ds.analyzer.PackageByPath("builtin")
			if p == nil {
				panic("builtin package not found")
			}
			pkg = p.PPkg.Types
			isBuiltin = true
		}
		if page.PathInfo.resType != ResTypePackage || pkg.Path() != page.PathInfo.resPath {
			buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, pkg.Path()}, page, pkg.Name())
			page.Write(period)
		}
		//page.WriteString(tt.Obj().Name())
		if isBuiltin || tt.Obj().Exported() {
			var pkgPath string
			if isBuiltin {
				pkgPath = "builtin"
			} else {
				pkgPath = tt.Obj().Pkg().Path()
			}
			buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, pkgPath}, page, tt.Obj().Name(), "name-", tt.Obj().Name())
		} else {
			p, _ := ds.analyzer.PackageByPath(pkg.Path())
			if p == nil {
				panic("should not")
			}
			ttPos := p.PPkg.Fset.PositionFor(tt.Obj().Pos(), false)
			//log.Printf("============ %v, %v, %v", tt, pkg.Path(), ttPos)
			ds.writeSrouceCodeLineLink(page, p, ttPos, tt.Obj().Name(), "", false)
		}
	case *types.Basic:
		// if unsafe
		//page.WriteString(tt.Name())
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, "builtin"}, page, tt.Name(), "name-", tt.Name())
	case *types.Pointer:
		page.Write(star)
		ds.writeValueTType(page, tt.Elem(), true)
	case *types.Array:
		page.Write(leftSquare)
		fmt.Sprintf("%d", tt.Len())
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), true)
	case *types.Slice:
		page.Write(leftSquare)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), true)
	case *types.Map:
		page.Write(mapKeyword)
		page.Write(leftSquare)
		ds.writeValueTType(page, tt.Key(), true)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), true)
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
		ds.writeValueTType(page, tt.Elem(), true)
	case *types.Signature:
		if writeFuncKeyword {
			page.Write(funcKeyword)
			//page.Write(space)
		}
		page.Write(leftParen)
		ds.writeTuple(page, tt.Params(), tt.Variadic())
		page.Write(rightParen)
		if rs := tt.Results(); rs != nil && rs.Len() > 0 {
			page.Write(space)
			if rs.Len() == 1 && rs.At(0).Name() == "" {
				ds.writeTuple(page, rs, false)
			} else {
				page.Write(leftParen)
				ds.writeTuple(page, rs, false)
				page.Write(rightParen)
			}
		}
	case *types.Struct:
		page.Write(structKeyword)
		page.Write(space)
		page.Write(leftBrace)
		ds.writeStructFields(page, tt)
		page.Write(rightBrace)
	case *types.Interface:
		page.Write(interfaceKeyword)
		page.Write(space)
		page.Write(leftBrace)
		ds.writeInterfaceMethods(page, tt)
		page.Write(rightBrace)
	}
}

// ToDo: need to be a method?
func (ds *docServer) writeTuple(page *htmlPage, tuple *types.Tuple, variadic bool) {
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
				ds.writeValueTType(page, st.Elem(), true)
			} else {
				ds.writeValueTType(page, v.Type(), true)
			}
		} else {
			ds.writeValueTType(page, v.Type(), true)
			page.WriteString(", ")
		}
	}
}

// ToDo: need to be a method?
func (ds *docServer) writeStructFields(page *htmlPage, st *types.Struct) {
	n := st.NumFields()
	for i := 0; i < n; i++ {
		v := st.Field(i)
		if !v.Embedded() {
			page.WriteString(v.Name())
			page.WriteByte(' ')
		}
		ds.writeValueTType(page, v.Type(), true)
		if i < n-1 {
			page.WriteString("; ")
		}
	}
}

// ToDo: need to be a method?
func (ds *docServer) writeInterfaceMethods(page *htmlPage, it *types.Interface) {
	n := it.NumEmbeddeds()
	for i := 0; i < n; i++ {
		named := it.Embedded(i)
		ds.writeValueTType(page, named.Obj().Type(), false)
		if i < n-1 {
			page.WriteString("; ")
		}
	}
	n = it.NumExplicitMethods()
	if n > 0 {
		page.WriteString("; ")
	}
	for i := 0; i < n; i++ {
		f := it.ExplicitMethod(i)
		page.WriteString(f.Name())
		page.WriteByte(' ')
		ds.writeValueTType(page, f.Type(), false)
		if i < n-1 {
			page.WriteString("; ")
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

// Do not ue the offiical builtin package page? Too many quirks. Use a simple custom page instead?
// * make(Type ChannelKind|MapKind|SliceKind) Type
//   Type must denote a channel, map, or slice type.
//   make(Type ChannelKind|MapKind|SliceKind, size integer) Type
//   Type must denote a channel, map, or slice type.
//   size must be a non-negative integer value (of any integer type) or a literal denoting a non-negative integer value.
//   make(Type SliceKind, length integer, capacity interger) Type
//   Type must denote a slice type.
//   length and capacity must be both non-negative integer values (of any integer type) or literals denoting a non-negative integer values.
//   The types of length and capacity may be different.
// * new(Type AnyKind) *Type
// * each with simple examples
