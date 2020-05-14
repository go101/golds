package code

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"

	"go101.org/gold/util"
)

func avoidCheckFuncBody(fset *token.FileSet, parseFilename string, _ []byte) (*ast.File, error) {
	var src interface{}
	mode := parser.ParseComments // | parser.AllErrors
	file, err := parser.ParseFile(fset, parseFilename, src, mode)
	if file == nil {
		return nil, err
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fd.Body = nil
		}
	}
	return file, nil
}

func collectPPackages(ppkgs []*packages.Package) map[string]*packages.Package {
	var allPPkgs = make(map[string]*packages.Package, 1000)
	var regPkgs func(ppkg *packages.Package)
	regPkgs = func(ppkg *packages.Package) {
		if _, present := allPPkgs[ppkg.PkgPath]; present {
			return
		}
		allPPkgs[ppkg.PkgPath] = ppkg
		for _, p := range ppkg.Imports {
			regPkgs(p)
		}
	}

	for _, ppkg := range ppkgs {
		regPkgs(ppkg)
	}

	return allPPkgs
}

func collectStdPackages() ([]*packages.Package, error) {
	//log.Println("[collect std packages ...]")
	//defer log.Println("[collect std packages done]")

	var configForCollectStdPkgs = &packages.Config{
		Tests: false,
	}

	return packages.Load(configForCollectStdPkgs, "std")
}

func (d *CodeAnalyzer) ParsePackages(args ...string) bool {
	var stopWatch = util.NewStopWatch()

	//log.Println("[parse packages ...], args:", args)

	// ToDo: check cache to avoid parsing again.

	downloading := true

	var configForParsing = &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedTypes | packages.NeedExportsFile | packages.NeedFiles |
			packages.NeedCompiledGoFiles | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false, // ToDo: parse tests

		//Logf: func(format string, args ...interface{}) {
		//	log.Println("================================================\n", args)
		//},

		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			if downloading {
				log.Println("Prepare packages:", stopWatch.Duration())
				downloading = false
			}

			//defer log.Println("parsed", filename)
			const mode = parser.AllErrors | parser.ParseComments
			return parser.ParseFile(fset, filename, src, mode)
		},

		// Reasons to disable this:
		// 1. to surpress "imported but not used" errors
		// 2. to implemente "voew code" and "jump to definition" features.
		// It looks the memory comsumed will be doubled.
		//ParseFile: avoidCheckFuncBody,

		//       NeedTypes: NeedTypes adds Types, Fset, and IllTyped.
		//       Why can't only Fset be got?
		// ToDo: modify go/packages code to not use go/types.
		//       use ast only and build type info tailored for docs and code reading.
		//       But it looks NeedTypes doesn't consume much more memory, so ...
		//       And, go/types can be used to verify the correctness of the custom implementaion.
	}

	ppkgs, err := packages.Load(configForParsing, args...)
	if err != nil {
		log.Println("packages.Load (parse packages):", err)
		return false
	}

	log.Println("Load packages:", stopWatch.Duration())

	stdPPkgs, err := collectStdPackages()
	if err != nil {
		log.Fatal("failed to collect std packages: ", err)
	}

	log.Println("CollectStdPackages:", stopWatch.Duration())

	defer func() {
		log.Println("Collect packages:", stopWatch.Duration())
	}()

	var hasErrors bool
	for _, ppkg := range ppkgs {
		switch ppkg.PkgPath {
		case "builtin":
			// skip "illegal cycle in declaration of int" alike errors.
		//case "unsafe":
		default:
			// ToDo: how to judge "imported but not used" errors?

			if packages.PrintErrors([]*packages.Package{ppkg}) > 0 {
				hasErrors = true
			}
		}
	}
	if hasErrors {
		log.Fatal("exit for above errors")
	}

	var allPPkgs = collectPPackages(ppkgs)
	d.packageList = make([]*Package, 0, len(allPPkgs))
	d.packageTable = make(map[string]*Package, len(allPPkgs))

	//if len(d.packageList) != len(allPPkgs) {
	//	//panic("package counts not match! " + strconv.Itoa(len(d.packageList)) + " != " + strconv.Itoa(len(allPPkgs)))
	//}

	//if len(d.packageTable) != len(allPPkgs) {
	//	//panic("package counts not match! " + strconv.Itoa(len(d.packageTable)) + " != " + strconv.Itoa(len(allPPkgs)))
	//}

	// It looks the AST info of the parsed "unsafe" package is blank.
	// So we fill the info manually to simplify some implementations later.
	if unsafePPkg, builtinPPkg := allPPkgs["unsafe"], allPPkgs["builtin"]; unsafePPkg != nil && builtinPPkg != nil {
		//log.Println("====== 111", unsafePPkg.Fset.Base(), builtinPPkg.Fset.Base(), allPPkgs["bytes"].Fset.Base())
		fillUnsafePackage(unsafePPkg, builtinPPkg)
	}

	//var packageListChanged = false
	for path, ppkg := range allPPkgs {
		pkg := d.packageTable[ppkg.PkgPath]
		if pkg == nil {
			//packageListChanged = true

			pkg := &Package{PPkg: ppkg}
			d.packageTable[path] = pkg
			d.packageList = append(d.packageList, pkg)

			//log.Println("     [parsed]", path)
		} else {
			pkg.PPkg = ppkg

			log.Println("     [parsed]", path, "(duplicated?)")
		}
		//if len(ppkg.Errors) > 0 {
		//	for _, err := range ppkg.Errors {
		//		log.Printf("          error: %#v", err)
		//	}
		//}
	}
	d.builtinPkg = d.packageTable["builtin"]

	var pkgNumDepedBys = make(map[*Package]uint32, len(allPPkgs))
	for _, pkg := range d.packageList {
		pkg.Deps = make([]*Package, 0, len(pkg.PPkg.Imports))
		//for path := range pkg.PPkg.Imports // the path never starts with "vendor/"
		for _, ppkg := range pkg.PPkg.Imports {
			path := ppkg.PkgPath // may start with "vendor/"
			depPkg := d.packageTable[path]
			if depPkg == nil {
				panic("ParsePackages: dependency package " + path + " not found")
			}
			pkg.Deps = append(pkg.Deps, depPkg)
			pkgNumDepedBys[depPkg]++
		}
	}
	for _, pkg := range d.packageList {
		pkg.DepedBys = make([]*Package, 0, pkgNumDepedBys[pkg])
	}
	for _, pkg := range d.packageList {
		for _, dep := range pkg.Deps {
			dep.DepedBys = append(dep.DepedBys, pkg)
		}
	}

	//log.Println("[parse packages done]")

	// Confirm std packages.
	d.stdModule = &Module{
		Dir:     "",
		Root:    "",
		Version: "", // ToDo
	}
	estimatedNumMods := 1 + len(d.packageList)/3
	d.allModules = make([]*Module, estimatedNumMods)
	d.allModules = append(d.allModules, d.stdModule)

	for _, ppkg := range stdPPkgs {
		pkg := d.packageTable[ppkg.PkgPath]
		if pkg != nil {
			pkg.Mod = d.stdModule
		}
	}

	// ...

	return true
}

// Go 1.14: added go/build.Context.Dir, ..., some convienient to not use go/types and go/packages?

// ToDo: use go/doc to retrieve package docs

// ToDo: don't load builtin pacakge, construct a custom builtin package manually instead.

// ToDo: make some special handling in unsafe package page creation.

func fillUnsafePackage(unsafePPkg *packages.Package, builtinPPkg *packages.Package) {
	intType := builtinPPkg.Types.Scope().Lookup("int").Type()

	//log.Println("====== 000", unsafePPkg.PkgPath)
	//log.Println("====== 222", unsafePPkg.Fset.PositionFor(token.Pos(0), false))

	buildPkg, err := build.Import("unsafe", "", build.FindOnly)
	if err != nil {
		log.Fatal(fmt.Errorf("build.Import: %w", err))
	}

	filter := func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
	}

	//log.Println("====== 333", buildPkg.Dir)
	fset := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fset, buildPkg.Dir, filter, parser.ParseComments)
	if err != nil {
		log.Fatal(fmt.Errorf("parser.ParseDir: %w", err))
	}

	astPkg := astPkgs["unsafe"]
	if astPkg == nil {
		log.Fatal("ast package for unsafe is not found")
	}

	// It is strange that unsafePPkg.Fset is not blank
	// (it looks all parsed packages (by go/Packages.Load) share the same FileSet)
	// even if unsafePPkg.GoFiles and unsafePPkg.Syntax (and more) are both blank.
	// This is why the current function tries to fill them.
	unsafePPkg.TypesInfo.Defs = make(map[*ast.Ident]types.Object)
	unsafePPkg.TypesInfo.Types = make(map[ast.Expr]types.TypeAndValue)
	unsafePPkg.Fset = fset

	var artitraryExpr, intExpr ast.Expr
	var artitraryType types.Type

	for filename, astFile := range astPkg.Files {
		unsafePPkg.GoFiles = append(unsafePPkg.GoFiles, filename)
		unsafePPkg.CompiledGoFiles = append(unsafePPkg.CompiledGoFiles, filename)
		//log.Println("unsafe filename:", filename)
		unsafePPkg.Syntax = append(unsafePPkg.Syntax, astFile)
		//unsafePPkg.Fset.AddFile(filename, unsafePPkg.Fset.Base(), int(astFile.End()-astFile.Pos()))
		//log.Println("====== 444", filename, unsafePPkg.Fset.Base(), int(astFile.End()-astFile.Pos()))

		for _, decl := range astFile.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				//if fd.Name.IsExported() {
				//	log.Printf("     func declaration: %s", fd.Name.Name)
				//}

				obj := types.Unsafe.Scope().Lookup(fd.Name.Name)
				if obj == nil {
					panic(fd.Name.Name + " is not found in unsafe scope")
				}

				unsafePPkg.TypesInfo.Defs[fd.Name] = obj
				continue
			}

			gn, ok := decl.(*ast.GenDecl)
			if !ok || gn.Tok != token.TYPE {
				continue
			}

			for _, spec := range gn.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				//if typeSpec.Name.IsExported() {
				//	log.Printf("     type declaration: %s", typeSpec.Name.Name)
				//}

				obj := types.Unsafe.Scope().Lookup(typeSpec.Name.Name)
				typeObj, _ := obj.(*types.TypeName)
				switch typeSpec.Name.Name {
				default:
					panic("Unexpected type name in unsafe: " + typeSpec.Name.Name)
				case "Pointer":
					if typeObj == nil {
						panic("a non-nil type object for unsafe.Pointer is expected")
					}
					artitraryExpr = typeSpec.Type
				case "ArbitraryType":
					intExpr = typeSpec.Type
					if typeObj != nil {
						panic("a nil type object for ArbitraryType is expected")
					}
					//log.Println("    ", typeSpec.Name.Name, "is not found in unsafe scope. Create one manually.")

					//log.Printf("%T %T\n", intType, intType.Underlying())

					// ToDo:
					// The last argument is nil is because the creations of
					// the following two objects depend on each other.
					// ;(, Maybe I have not found the right solution.
					typeObj = types.NewTypeName(typeSpec.Pos(), types.Unsafe, typeSpec.Name.Name, nil)
					unsafePPkg.Types.Scope().Insert(typeObj)
					artitraryType = types.NewNamed(typeObj, intType.Underlying(), nil)
				}

				// new declared type (source type will be set below)
				unsafePPkg.TypesInfo.Defs[typeSpec.Name] = typeObj
			}
		}

		// ToDo: how init and _ functions are parsed.
	}

	if artitraryExpr == nil {
		panic("artitraryExpr is nil")
	}

	if intExpr == nil {
		panic("intExpr is nil")
	}

	if artitraryType == nil {
		panic("artitraryType is nil")
	}

	// source types
	unsafePPkg.TypesInfo.Types[intExpr] = types.TypeAndValue{Type: intType}
	unsafePPkg.TypesInfo.Types[artitraryExpr] = types.TypeAndValue{Type: artitraryType}
}
