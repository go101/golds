package server

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"net/http"

	"go101.org/golds/code"
)

//type implPageKey struct {
//	pkg string
//	typ string
//}

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

	if genDocsMode {
		pkgPath = deHashScope(pkgPath)
		typeName = deHashIdentifier(typeName)
	}

	//pageKey := implPageKey{pkg: pkgPath, typ: typeName}
	//if ds.implPages[pageKey] == nil {
	//	result, err := ds.buildImplementationData(ds.analyzer, pkgPath, typeName)
	//	if err != nil {
	//		w.WriteHeader(http.StatusNotFound)
	//		fmt.Fprint(w, "Build implementation info for (", typeName, ") in ", pkgPath, " error: ", err)
	//		return
	//	}
	//	ds.implPages[pageKey] = ds.buildImplementationPage(result)
	//}
	//w.Write(ds.implPages[pageKey])

	pageKey := pageCacheKey{
		resType: ResTypeImplementation,
		res:     [...]string{pkgPath, typeName},
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		result, err := ds.buildImplementationData(ds.analyzer, pkgPath, typeName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Build implementation info for (", typeName, ") in ", pkgPath, " error: ", err)
			return
		}

		data = ds.buildImplementationPage(w, result)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildImplementationPage(w http.ResponseWriter, result *MethodImplementationResult) []byte {
	// some methods are born by embedding other types.
	// Use the same design for local id: click such methods to highlight all same-origin ones.

	title := ds.currentTranslation.Text_MethodImplementations() + ds.currentTranslation.Text_Colon(false) + result.Package.Path + "." + result.TypeName.Name()
	page := NewHtmlPage(goldsVersion, title, ds.currentTheme, ds.currentTranslation, createPagePathInfo2(ResTypeImplementation, result.Package.Path, ".", result.TypeName.Name()))

	fmt.Fprintf(page, `<pre><code><span style="font-size:x-large;">type <a href="%s">%s</a>.`,
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, result.Package.Path), nil, ""),
		result.Package.Path,
	)
	page.WriteString("<b>")
	//writeSrouceCodeLineLink(page, result.TypeName.Package(), result.TypeName.Position(), result.TypeName.Name(), "")
	ds.writeResourceIndexHTML(page, result.TypeName.Package(), result.TypeName, false, false, false)
	page.WriteString(`</b></span><span style="font-size:large;">`)
	writeKindText(page, result.TypeName.Denoting().TT)
	page.WriteString("</span>\n")

	nonImplementingMethodCountText := ""
	if !result.IsInterface {
		nonImplementingMethodCountText = page.Translation().Text_NumMethodsImplementingNothing(int(result.NonImplementingMethodCount))
	}

	fmt.Fprintf(page, `
<code><span class="title">%s<span class="title-stat"><i>%s</i></span></span>
`,
		page.Translation().Text_MethodImplementations(),
		nonImplementingMethodCountText,
	)

	for _, method := range result.Methods {
		methodName := method.Method.Name()
		dotMStyle := DotMStyle_Unexported
		if token.IsExported(methodName) {
			dotMStyle = DotMStyle_Exported
		}
		page.WriteString("\n")
		anchorName := methodName
		isExported := !token.IsExported(methodName)
		if isExported {
			anchorName = method.Method.Package().Path + "." + methodName
		}
		fmt.Fprintf(page, `<div class="anchor" id="name-%s">`, anchorName)
		page.WriteByte('\t')
		// ToDo: need to record which type the method is declared for.
		//       For some rare cases, two same unexported methods from two different packages ...
		//
		if buildIdUsesPages {
			buildPageHref(page.PathInfo, createPagePathInfo3(ResTypeReference, result.Package.Path, "..", result.TypeName.Name(), method.Method.Name()), page, method.Method.Name())
			ds.writeMethodType(page, result.Package, method.Method.Method, nil)
		} else {
			ds.writeMethodForListing(page, result.Package, method.Method, nil, false, false)
		}
		for _, imp := range method.Implementations {
			page.WriteString("\n\t\t")
			if result.IsInterface {
				ds.writeTypeForListing(page, imp.Receiver, result.Package, "", dotMStyle)
			} else {
				ds.writeTypeForListing(page, imp.Receiver, result.Package, result.TypeName.Name(), dotMStyle)
			}
			page.WriteByte('.')
			ds.WriteEmbeddingChain(page, imp.Method.EmbeddingChain)
			//writeSrouceCodeLineLink(page, imp.Method.Package(), imp.Method.Position(), methodName, "b")
			page.WriteString("<b>")
			ds.writeMethodForListing(page, result.Package, imp.Method, nil, false, true)
			page.WriteString("</b>")
		}
		page.WriteString("</div>")
	}

	page.WriteString("</code></pre>")
	return page.Done(w)
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
//
//	Because methods are registered on struct{...}.
func (ds *docServer) buildImplementationData(analyzer *code.CodeAnalyzer, pkgPath, typeName string) (*MethodImplementationResult, error) {
	if !collectUnexporteds && pkgPath != "builtin" && !token.IsExported(typeName) {
		panic("should not go here (imp): " + pkgPath + "." + typeName)
	}

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
					return nil, fmt.Errorf("%s.%s is an alias of %s.%s", pkgPath, typeName, typeInfo.TypeName.Pkg.Path, typeInfo.TypeName.Name())
				}
				//denotingTypeName = typeInfo.TypeName.Name()
				//denotingTypeNamePkgPath = typeInfo.TypeName.Pkg.Path
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
	methodSelectors, _ := buildTypeMethodsList(typeInfo, true)
	if isInterface {
		if len(typeInfo.ImplementedBys) == 0 {
			return nil, fmt.Errorf("no types implement %s.%s", pkgPath, typeName)
		}

		for _, sel := range methodSelectors {
			if !collectUnexporteds && !token.IsExported(sel.Name()) {
				continue
			}
			impls := make([]MethodInfo, 0, len(typeInfo.ImplementedBys))
			impBys, _ := buildTypeImplementedByList(analyzer, pkg, typeInfo, true, typeNameRes)
			selNameIsUnexported := !token.IsExported(sel.Name())
			for _, impBy := range impBys {
				if !collectUnexporteds && impBy.TypeName.Package().Path != "builtin" && !impBy.TypeName.Exported() {
					continue
				}
				impByDenoting := impBy.TypeName.Denoting()
				for _, m := range impByDenoting.AllMethods {
					if !collectUnexporteds && !token.IsExported(m.Name()) {
						continue
					}
					matched := sel.Name() == m.Name()
					if matched && selNameIsUnexported {
						matched = matched && m.Package().Path == sel.Package().Path
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
			if !collectUnexporteds && !token.IsExported(sel.Name()) {
				continue
			}
			impls := make([]MethodInfo, 0, len(typeInfo.Implements))
			imps, _ := buildTypeImplementsList(analyzer, pkg, typeInfo, true)
			selNameIsUnexported := !token.IsExported(sel.Name())
			for _, imp := range imps {
				if !collectUnexporteds && imp.TypeName.Package().Path != "builtin" && !imp.TypeName.Exported() {
					continue
				}
				impDenoting := imp.TypeName.Denoting()
				for _, m := range impDenoting.AllMethods {
					if !collectUnexporteds && !token.IsExported(m.Name()) {
						continue
					}
					matched := sel.Name() == m.Name()
					if matched && selNameIsUnexported {
						matched = matched && m.Package().Path == sel.Package().Path
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
