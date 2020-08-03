package server

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"net/http"

	"go101.org/gold/code"
)

type implPageKey struct {
	pkg string
	typ string
}

func (ds *docServer) methodImplementationPage(w http.ResponseWriter, r *http.Request, pkgPath, typeName string) {
	w.Header().Set("Content-Type", "text/html")

	//log.Println(pkgPath, bareFilename)

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	pageKey := implPageKey{pkg: pkgPath, typ: typeName}
	if ds.implPages[pageKey] == nil {
		result, err := ds.buildImplementationData(ds.analyzer, pkgPath, typeName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Build implementation info for (", typeName, ") in ", pkgPath, " error: ", err)
			return
		}
		ds.implPages[pageKey] = ds.buildImplementationPage(result)
	}
	w.Write(ds.implPages[pageKey])
}

func (ds *docServer) buildImplementationPage(result *MethodImplementationResult) []byte {
	// some methods are born by embedding other types.
	// Use the same design for local id: click such methods to highlight all same-origin ones.

	qualifiedTypeName := result.Package.Path() + "." + result.TypeName.Name()
	title := ds.currentTranslation.Text_MethodImplementation() + qualifiedTypeName
	page := NewHtmlPage(ds.goldVersion, title, ds.currentTheme.Name(), pagePathInfo{ResTypeImplementation, qualifiedTypeName})

	fmt.Fprintf(page, `<pre><code><span style="font-size:larger;">type <a href="%s">%s</a>.`,
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, result.Package.Path()}, nil, ""),
		result.Package.Path(),
	)
	ds.writeSrouceCodeLineLink(page, result.TypeName.Package(), result.TypeName.Position(), result.TypeName.Name(), "", false)
	writeKindText(page, result.TypeName.Denoting().TT)
	page.WriteString("</span>\n")

	nonImplementingMethodCountText := ""
	if !result.IsInterface {
		nonImplementingMethodCountText = fmt.Sprintf(" (%d other methods implement nothing)", result.NonImplementingMethodCount)
	}

	fmt.Fprintf(page, `
Method Implementations%s:
`,
		nonImplementingMethodCountText,
	)

	for _, method := range result.Methods {
		methodName := method.Method.Name()
		dotMStyle := DotMStyle_Unexported
		if token.IsExported(methodName) {
			dotMStyle = DotMStyle_Exported
		}
		page.WriteString("\n")
		fmt.Fprintf(page, `<div class="anchor" id="name-%s">`, methodName)
		page.WriteByte('\t')
		ds.writeMethodForListing(page, result.Package, method.Method, nil, false)
		for _, imp := range method.Implementations {
			page.WriteString("\n\t\t")
			if result.IsInterface {
				ds.writeTypeForListing(page, imp.Receiver, result.Package, "", dotMStyle)
			} else {
				ds.writeTypeForListing(page, imp.Receiver, result.Package, result.TypeName.Name(), dotMStyle)
			}
			page.WriteByte('.')
			ds.WriteEmbeddingChain(page, imp.Method.EmbeddingChain)
			ds.writeSrouceCodeLineLink(page, imp.Method.Pkg(), imp.Method.Position(), methodName, "b", false)
		}
		page.WriteString("</div>")
	}

	page.WriteString("</code></pre>")
	return page.Done(ds.currentTranslation)
}

type MethodImplementationResult struct {
	TypeName    *code.TypeName
	Package     *code.Package
	IsInterface bool
	//DenotingTypeName        string
	//DenotingTypeNamePkgPath string

	Methods []MethodImplementations

	NonImplementingMethodCount int32
}

type MethodImplementations struct {
	Method          *code.Selector
	Implementations []MethodInfo
}

type MethodInfo struct {
	Method    *code.Selector
	Receiver  *TypeForListing
	Explicit  bool // whether or not the method is explicit
	Interface bool // whether or not the Owner is an interface type
}

// ToDo: if typeName is like a (type T = *struct{...}, methods will not be listed.
//       Because methods are registered on struct{...}.
func (ds *docServer) buildImplementationData(analyzer *code.CodeAnalyzer, pkgPath, typeName string) (*MethodImplementationResult, error) {
	pkg := analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil, errors.New("package not found")
	}

	//var denotingTypeName, denotingTypeNamePkgPath string
	var typeNameRes *code.TypeName
	var typeInfo *code.TypeInfo
	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		if tn.Name() == typeName {
			typeNameRes = tn
			// tn might be an alias
			typeInfo = tn.Denoting()
			if tn.Alias != nil && typeInfo.TypeName != nil {
				if typeInfo.TypeName.Pkg == pkg {
					return nil, fmt.Errorf("%s.%s is an alias of %s", pkgPath, typeName, typeInfo.TypeName.Name())
				} else {
					return nil, fmt.Errorf("%s.%s is an alias of %s.%s", pkgPath, typeName, typeInfo.TypeName.Pkg.Path(), typeInfo.TypeName.Name())
				}
				//denotingTypeName = typeInfo.TypeName.Name()
				//denotingTypeNamePkgPath = typeInfo.TypeName.Pkg.Path()
			}
			break
		}
	}

	if typeInfo == nil {
		return nil, errors.New("typename not found")
	}
	if len(typeInfo.AllMethods) == 0 {
		return nil, fmt.Errorf("%s.%s has no methods", pkgPath, typeName)
	}

	_, isInterface := typeInfo.TT.Underlying().(*types.Interface)
	var nonImplementingMethodCount int32

	methodImplementations := make([]MethodImplementations, 0, len(typeInfo.AllMethods))
	methodSelectors := buildTypeMethodsList(typeInfo, true)
	if isInterface {
		if len(typeInfo.ImplementedBys) == 0 {
			return nil, fmt.Errorf("no types implement %s.%s", pkgPath, typeName)
		}

		for _, sel := range methodSelectors {
			impls := make([]MethodInfo, 0, len(typeInfo.ImplementedBys))
			impBys := ds.sortTypeList(buildTypeImplementedByList(analyzer, typeInfo, true, typeNameRes), pkg)
			selNameIsUnexported := !token.IsExported(sel.Name())
			for _, impBy := range impBys {
				impByDenoting := impBy.TypeName.Denoting()
				for _, m := range impByDenoting.AllMethods {
					matched := sel.Name() == m.Name()
					if matched && selNameIsUnexported {
						matched = matched && m.Pkg().Path() == sel.Pkg().Path()
					}
					if matched {
						explicit := sel.EmbeddingChain == nil
						_, inteface := impByDenoting.TT.Underlying().(*types.Interface)
						impls = append(impls, MethodInfo{
							Method:    m,
							Receiver:  impBy,
							Explicit:  explicit,
							Interface: inteface,
						})
						break
					}
				}
			}
			methodImplementations = append(methodImplementations, MethodImplementations{
				Method:          sel,
				Implementations: impls,
			})
		}
	} else {
		if len(typeInfo.Implements) == 0 {
			return nil, fmt.Errorf("%s.%s doesn't implement any interface types with at least one method", pkgPath, typeName)
		}

		for _, sel := range methodSelectors {
			impls := make([]MethodInfo, 0, len(typeInfo.Implements))
			imps := ds.sortTypeList(buildTypeImplementsList(analyzer, typeInfo, true), pkg)
			selNameIsUnexported := !token.IsExported(sel.Name())
			for _, imp := range imps {
				impDenoting := imp.TypeName.Denoting()
				for _, m := range impDenoting.AllMethods {
					matched := sel.Name() == m.Name()
					if matched && selNameIsUnexported {
						matched = matched && m.Pkg().Path() == sel.Pkg().Path()
					}
					if matched {
						explicit := sel.EmbeddingChain == nil
						inteface := true
						impls = append(impls, MethodInfo{
							Method:    m,
							Receiver:  imp,
							Explicit:  explicit,
							Interface: inteface,
						})
					}
				}
			}
			if len(impls) == 0 {
				nonImplementingMethodCount++
			} else {
				methodImplementations = append(methodImplementations, MethodImplementations{
					Method:          sel,
					Implementations: impls,
				})
			}
		}
	}

	return &MethodImplementationResult{
		TypeName:    typeNameRes,
		Package:     pkg,
		IsInterface: isInterface,
		//DenotingTypeName:        denotingTypeName,
		//DenotingTypeNamePkgPath: denotingTypeNamePkgPath,

		NonImplementingMethodCount: nonImplementingMethodCount,

		Methods: methodImplementations,
	}, nil
}
