package server

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"net/http"
	"sort"
	"strings"

	"go101.org/golds/code"
)

//type usePageKey struct {
//	pkg string
//
//	// ToDo: Generally, this is a package-level identifer and selector identifier.
//	// It might be extended to fake identiers for unnamed types later.
//	// It should be nerver a local identifer.
//	id string
//}

// ToDo: for types, also list its values, including locals

// identifier might be a package-level declared identifier,
// or a selector which represents a field or method.
func (ds *docServer) identifierReferencePage(w http.ResponseWriter, r *http.Request, pkgPath, identifier string) {
	w.Header().Set("Content-Type", "text/html")

	//log.Println(pkgPath, bareFilename)

	// To avoid some too time-comsuming cases,
	// now only supporting unexported identfiers, which
	// don't need page caching.

	// Add query parameter: scope=a/b/pkg, default is the id containing package.
	// If the id is exported, list the pacakges importing the containing package
	// by use each of them as the scope parameter value.
	// Only search one package for each page show.

	// The search result should be be cached.
	// Use don't care most id uses.
	// Cache the ever searcheds is ok.
	//    map[*ast.Ident][]token.Pos

	tokens := strings.Split(identifier, ".")
	if genDocsMode {
		pkgPath = deHashScope(pkgPath)
		for i, t := range tokens {
			tokens[i] = deHashIdentifier(t)
		}
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	// Pages for non-exported identifiers will not be cached.

	//useKey := usePageKey{pkg: pkgPath, id: identifier}
	//if ds.identifierReferencesPages[useKey] == nil {
	//	result, err := ds.buildReferencesData(pkgPath, identifier)
	//	if err != nil {
	//		w.WriteHeader(http.StatusNotFound)
	//		fmt.Fprint(w, "error: ", err)
	//		return
	//	}
	//	ds.identifierReferencesPages[useKey] = ds.buildReferencesPage(result)
	//}
	//w.Write(ds.identifierReferencesPages[useKey])

	pageKey := pageCacheKey{
		resType: ResTypeReference,
		res:     [...]string{pkgPath, identifier},
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		result, err := ds.buildReferencesData(pkgPath, tokens...)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "error: ", err)
			return
		}

		data = ds.buildReferencesPage(w, result)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildReferencesPage(w http.ResponseWriter, result *ReferencesResult) []byte {
	title := ds.currentTranslation.Text_ReferenceList() + ds.currentTranslation.Text_Colon(false) + result.Package.Path + "." + result.Identifier
	page := NewHtmlPage(goldsVersion, title, ds.currentTheme, ds.currentTranslation, createPagePathInfo2(ResTypeReference, result.Package.Path, "..", result.Identifier))

	var prefix string
	if result.Selector == nil {
		switch result.Resource.(type) {
		case *code.Variable:
			prefix = "var "
		case *code.Constant:
			prefix = "const "
		case *code.Function:
			prefix = "func "
		case *code.TypeName:
			prefix = "type "
		}
	}
	fmt.Fprintf(page, `
<pre><code><span style="font-size:x-large;">%s<b><a href="%s">%s</a>.`,
		prefix,
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, result.Package.Path), nil, ""),
		result.Package.Path,
	)

	ds.writeResourceIndexHTML(page, result.Package, result.Resource, false, false, false)

	if result.Selector != nil {
		page.WriteByte('.')
		if result.Selector.Field != nil {
			ds.writeFieldCodeLink(page, result.Selector)
		} else {
			ds.writeMethodForListing(page, result.Package, result.Selector, nil, false, true)
		}
	}
	page.WriteString(`</b></span>`)

	if result.Selector != nil {
		page.WriteString(`<span style="font-size: large;"><i>`)
		page.WriteString(page.Translation().Text_Parenthesis(false))
		if result.Selector.Field != nil {
			page.WriteString(page.Translation().Text_ObjectKind("field"))
		} else {
			page.WriteString(page.Translation().Text_ObjectKind("method"))

			methodName := result.Selector.Method.Name
			var methodPkgPath string
			if !token.IsExported(methodName) {
				methodPkgPath = result.Selector.Method.Pkg.Path
			}
			var link string
			if ds.analyzer.CheckTypeMethodContributingToTypeImplementations(result.Package.Path, result.Resource.Name(), methodPkgPath, methodName) {
				// entering here meaning this must be a non-interface method.

				anchorName := methodName
				if !token.IsExported(methodName) {
					anchorName = methodPkgPath + "." + anchorName
				}
				if sourceReadingStyle == SourceReadingStyle_rich { // enableSoruceNavigation {
					if collectUnexporteds || result.Resource.Exported() || result.Package.Path == "builtin" {
						link = buildPageHref(page.PathInfo, createPagePathInfo2(ResTypeImplementation, result.Package.Path, ".", result.Resource.Name()), nil, "", "name-", anchorName)
					}
				}
			}

			if link != "" {
				page.WriteString(page.Translation().Text_Comma())
				fmt.Fprintf(page, `<a href="%s">%s</a>`, link, page.Translation().Text_ViewMethodImplementations())
			}
		}
		page.WriteString(page.Translation().Text_Parenthesis(true))
		page.WriteString(`</i></span>`)
	}

	page.WriteString("\n\n")
	page.WriteString(`<span class="title">`)
	page.WriteString(page.Translation().Text_ObjectUses(result.UsesCount))
	page.WriteString(`</span>`)
	page.WriteString("\n")

	type idpos struct {
		id  *ast.Ident
		pos token.Position
	}
	stack := make([]idpos, 0, 8)

	excerptCode := func(fileInfo *code.SourceFileInfo) {
		n := len(stack)
		if n == 0 {
			panic("should not")
		}
		n--

		// ToDo: maybe ast.File.LineStart is better to do this job.
		start := bytes.LastIndexByte(fileInfo.Content[:stack[0].pos.Offset], '\n')
		start++
		end := bytes.IndexByte(fileInfo.Content[stack[n].pos.Offset:], '\n')
		if end < 0 {
			end = len(fileInfo.Content)
		} else {
			end += stack[n].pos.Offset
		}

		for i := range stack {
			endOffset := stack[i].pos.Offset + len(stack[i].id.Name)

			//page.Write(fileInfo.Content[start:stack[i].pos.Offset])
			page.AsHTMLEscapeWriter().Write(fileInfo.Content[start:stack[i].pos.Offset])
			page.WriteString("<b>")
			//page.Write(fileInfo.Content[stack[i].pos.Offset:endOffset])
			page.AsHTMLEscapeWriter().Write(fileInfo.Content[stack[i].pos.Offset:endOffset])
			page.WriteString("</b>")

			start = endOffset
		}
		page.AsHTMLEscapeWriter().Write(fileInfo.Content[start:end])
		page.WriteByte('\n')
		stack = stack[:0]
	}

	for _, refGroup := range result.References {
		page.WriteString("\n\t")
		if refGroup.Pkg.Path == result.Package.Path {
			page.WriteString(refGroup.Pkg.Path)
			page.WriteString(page.Translation().Text_CurrentPackage())
		} else {
			buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, refGroup.Pkg.Path), page, refGroup.Pkg.Path)
		}
		page.WriteByte('\n')

		var fileInfo *code.SourceFileInfo
		var lineNumber int
		stack = stack[:0]
		for i := range refGroup.Identifiers {
			id := &refGroup.Identifiers[i]
			if fileInfo != id.FileInfo {
				if fileInfo != nil {
					excerptCode(fileInfo)
				}
				lineNumber = 0
				fileInfo = id.FileInfo
				//page.WriteString("\t\t")
				//writeSrouceCodeFileLink(page, refGroup.Pkg, fileInfo.AstBareFileName())
				//page.WriteByte('\n')
			}

			pos := refGroup.Pkg.PPkg.Fset.PositionFor(id.AstIdent.NamePos, false)
			if lineNumber != pos.Line {
				if lineNumber > 0 {
					// ExcerptNearbyCode(page, id.FileInfo, id.AstIdent, pos)
					excerptCode(fileInfo)
				}
				//page.WriteString("\t\t\t")
				page.WriteString("\t\t")
				if lineNumber > 0 {
					linkText := fmt.Sprintf("%s", fileInfo.AstBareFileName())
					writeSrouceCodeLineLink(page, refGroup.Pkg, pos, linkText, "path-duplicate")
					linkText = fmt.Sprintf("#L%d", pos.Line)
					writeSrouceCodeLineLink(page, refGroup.Pkg, pos, linkText, "")
				} else {
					linkText := fmt.Sprintf("%s#L%d", fileInfo.AstBareFileName(), pos.Line)
					writeSrouceCodeLineLink(page, refGroup.Pkg, pos, linkText, "")
				}
				page.WriteString(": ")
				lineNumber = pos.Line
			}
			stack = append(stack, idpos{id: id.AstIdent, pos: pos})
		}
		excerptCode(fileInfo)
	}

	page.WriteString("</code></pre>")
	return page.Done(w)
}

//func ExcerptNearbyCode(page *htmlPage, fileInfo *code.SourceFileInfo, astIdent *ast.Ident, pos token.Position) {
//	// ToDo: maybe ast.File.LineStart is better to do this job.
//	start := bytes.LastIndexByte(fileInfo.Content[:pos.Offset], '\n')
//	start++
//	end := bytes.IndexByte(fileInfo.Content[pos.Offset:], '\n')
//	if end < 0 {
//		end = len(fileInfo.Content)
//	} else {
//		end += pos.Offset
//	}
//	endOffset := pos.Offset + len(astIdent.Name)
//	page.Write(fileInfo.Content[start:pos.Offset])
//	page.WriteString("<b>")
//	page.Write(fileInfo.Content[pos.Offset:endOffset])
//	page.WriteString("</b>")
//	page.Write(fileInfo.Content[endOffset:end])
//}

type ReferencesResult struct {
	Package    *code.Package
	Identifier string
	Resource   code.Resource
	Selector   *code.Selector // non-nil for fields and methods
	References []*ObjectReferences
	UsesCount  int
}

type ObjectReferences struct {
	Pkg          *code.Package
	CommonPath   string // relative to the current package
	InCurrentPkg bool
	Identifiers  []code.Identifier
}

func (ds *docServer) buildReferencesData(pkgPath string, tokens ...string) (*ReferencesResult, error) {
	pkg := ds.analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil, fmt.Errorf("package %s is not found", pkgPath)
	}

	//isBuiltin := pkgPath == "builtin"

	//if len(identifier) == 0 {
	//	return nil, errors.New("identifier is not specified")
	//}
	//
	//tokens := strings.Split(identifier, ".")
	if len(tokens) > 2 {
		return nil, errors.New("invalid identifier (must be a pure identifer or a selector).")
	}

	var identifier string
	var res code.Resource
	var sel *code.Selector
	var obj types.Object
	if len(tokens) == 1 {
		if tokens[0] == "" {
			return nil, errors.New("identifier is not specified")
		}
		//if !collectUnexporteds && !isBuiltin && !token.IsExported(tokens[0]) {
		//	panic("should not go here (use): " + pkgPath + "." + tokens[0])
		//}
		identifier = tokens[0]

		for _, tn := range pkg.AllTypeNames {
			if tn.Name() == tokens[0] {
				res, obj = tn, tn.TypeName
				goto ResFound
			}
		}
		for _, f := range pkg.AllFunctions {
			if !f.IsMethod() && f.Name() == tokens[0] {
				res = f
				if f.Func != nil {
					obj = f.Func
				} else { // f.Builtin != nil
					obj = f.Builtin
				}
				goto ResFound
			}
		}
		for _, v := range pkg.AllVariables {
			if v.Name() == tokens[0] {
				res, obj = v, v.Var
				goto ResFound
			}
		}
		for _, c := range pkg.AllConstants {
			if c.Name() == tokens[0] {
				res, obj = c, c.Const
				goto ResFound
			}
		}

		// ToDo: add a ImportName Resource. As the references of an import name
		// are all in one source file, it would be good to use an alternative way
		// to list these references.

		return nil, fmt.Errorf("type %s is not found in package %s", tokens[0], pkgPath)
	} else { // len(tokens) == 2
		//if !collectUnexporteds && !isBuiltin && !token.IsExported(tokens[0]) {
		//	panic("should not go here (use): " + pkgPath + ".." + tokens[0])
		//}
		//if !collectUnexporteds && !token.IsExported(tokens[1]) {
		//	panic("should not go here (use): " + pkgPath + ".." + tokens[0] + "." + tokens[1])
		//}
		identifier = tokens[0] + "." + tokens[1]

		for _, tn := range pkg.AllTypeNames {
			if tn.Name() == tokens[0] {
				t := tn.Denoting()
				for _, field := range t.AllFields {
					if field.Name() == tokens[1] {
						sel = field
						goto SelFound
					}
				}
				for _, method := range t.AllMethods {
					if method.Name() == tokens[1] {
						sel = method
						goto SelFound
					}
				}
				return nil, fmt.Errorf("selector %s is not found for type %s in package %s", tokens[1], tokens[0], pkgPath)

			SelFound:

				res, obj = tn, sel.Object()
				goto ResFound
			}
		}
		return nil, fmt.Errorf("type %s is not found in package %s", tokens[0], pkgPath)
	}

ResFound:

	var refs []*ObjectReferences
	var usesCount int
	if obj != nil {
		ids := ds.analyzer.ObjectReferences(obj)
		usesCount = len(ids)

		numPkgs := 0
		var lastPkg *code.Package
		for _, id := range ids {
			if id.FileInfo.Pkg != lastPkg {
				lastPkg = id.FileInfo.Pkg
				numPkgs++
			}
		}

		allocatedRefs := make([]ObjectReferences, numPkgs)
		refs = make([]*ObjectReferences, numPkgs)
		var refIndex = numPkgs - 1
		var endIndex = len(ids) - 1
		var register = func(startIndex int) {
			ref := &allocatedRefs[refIndex]
			refs[refIndex] = ref
			ref.Identifiers = ids[startIndex : endIndex+1]
			ref.Pkg = lastPkg
			ref.InCurrentPkg = lastPkg.Path == pkgPath
			if ref.InCurrentPkg {
				ref.CommonPath = pkgPath
			} else {
				ref.CommonPath = FindPackageCommonPrefixPaths(lastPkg.Path, pkgPath)
			}
			refIndex--
		}

		for i := endIndex; i >= 0; i-- {
			id := &ids[i]
			if id.FileInfo.Pkg != lastPkg {
				register(i + 1)
				lastPkg = id.FileInfo.Pkg
				endIndex = i
			}
		}
		register(0)

		/*
			refsByPkg := make(map[*code.Package][]*ast.Ident, numPkgs)
			for _, id := range ids {
				dups := refsByPkg[id.FileInfo.Pkg]
				if dups == nil {
					dups = make([]*ast.Ident, 0, 4)
				}
				dups = append(dups, id.AstIdent)
				refsByPkg[id.FileInfo.Pkg] = dups
			}

			allocatedRefs := make([]ObjectReferences, len(refsByPkg))
			refs = make([]*ObjectReferences, len(refsByPkg))
			i := 0
			for pkg, ids := range refsByPkg {
				refs[i] = &allocatedRefs[i]
				refs[i].AstIdents = ids
				refs[i].Pkg = pkg
				refs[i].InCurrentPkg = pkg.Path == pkgPath
				if refs[i].InCurrentPkg {
					refs[i].CommonPath = pkgPath
				} else {
					refs[i].CommonPath = FindPackageCommonPrefixPaths(pkg.Path, pkgPath)
				}
				i++
			}
		*/

		sort.Slice(refs, func(a, b int) bool {
			commonA, commonB := refs[a].CommonPath, refs[b].CommonPath
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
			pathA, pathB := strings.ToLower(refs[a].Pkg.Path), strings.ToLower(refs[b].Pkg.Path)
			r := strings.Compare(pathA, pathB)
			if pathA == "builtin" {
				return true
			}
			if pathB == "builtin" {
				return false
			}
			return r < 0
		})
	}

	return &ReferencesResult{
		Package:    pkg,
		Identifier: identifier,
		Resource:   res,
		Selector:   sel,
		References: refs,
		UsesCount:  usesCount,
	}, nil
}
