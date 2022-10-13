// Package code is used to analyse Go code packages.
// It can find out all the implementation relations between package-level type.
package code

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"strings"
	//"runtime/debug"

	"golang.org/x/tools/go/types/typeutil"
)

// The analysis steps.
const (
	SubTask_PreparationDone = iota
	SubTask_NFilesParsed
	SubTask_ParsePackagesDone
	SubTask_CollectPackages
	SubTask_CollectModules
	SubTask_CollectExamples
	SubTask_SortPackagesByDependencies
	SubTask_CollectDeclarations
	SubTask_CollectRuntimeFunctionPositions
	SubTask_ConfirmTypeSources
	SubTask_CollectSelectors
	SubTask_CheckCollectedSelectors
	SubTask_FindImplementations
	SubTask_RegisterInterfaceMethodsForTypes
	SubTask_MakeStatistics
	SubTask_CollectSourceFiles
	SubTask_CollectObjectReferences
	SubTask_CacheSourceFiles
)

type ToolchainInfo struct {
	// Three paths
	Root, Src, Cmd string

	// A commit hash or something like "go1.16".
	// ToDo: now lso might be blank, but need some handling ...
	Version string
}

// CodeAnalyzer holds all the analysis results and functionalities.
type CodeAnalyzer struct {
	modulesByPath       map[string]*Module // including stdModule
	nonToolchainModules []Module           // not including stdModule and std/cmd module
	stdModule           *Module
	wdModule            *Module // working diretory module. It might be the cmd toolchain module, or nil if modules feature is off.

	//stdPackages  map[string]struct{}
	packageTable map[string]*Package
	packageList  []*Package
	builtinPkg   *Package

	// This one is removed now. We should use FileSet.PositionFor.
	//sourceFileLineOffsetTable map[string]int32

	//=== Will be fulfilled in analyse phase ===

	allSourceFiles map[string]*SourceFileInfo
	//sourceFile2PackageTable  map[string]SourceFile
	//sourceFile2PackageTable         map[string]*Package
	//generatedFile2OriginalFileTable map[string]string

	exampleFileSet *token.FileSet

	// *types.Type -> *TypeInfo
	lastTypeIndex       uint32
	ttype2TypeInfoTable typeutil.Map
	allTypeInfos        []*TypeInfo

	//>> 1.18, fake underlying for instantiated types
	// Always nil for 1.17-.
	blankInterface *TypeInfo
	//<<

	// Package-level declared type names.
	lastTypeNameIndex uint32
	allTypeNameTable  map[string]*TypeName

	// ToDo: need a better implementation.
	typeMethodsContributingToTypeImplementations map[[4]string]struct{}

	// Position info of some runtime functions.
	runtimeFuncPositions map[string]token.Position

	// Refs of unnamed types, type names, variables, functions, ...
	// Why not put []RefPos in TypeInfo, Variable, ...?
	//refPositions map[interface{}][]RefPos

	// ToDo: some TopN lists
	stats Stats

	// Identifer references (ToDo: need optimizations)
	objectRefs map[types.Object][]Identifier

	// Not concurrent safe.
	tempTypeLookup map[uint32]struct{}

	//
	forbidRegisterTypes bool // for debug

	debug bool
}

// WorkingDirectoryModule returns the module at the working directory.
// It might be nil.
func (d *CodeAnalyzer) WorkingDirectoryModule() *Module {
	return d.wdModule
}

// ModuleByPath returns the module corresponding the specified path.
func (d *CodeAnalyzer) ModuleByPath(path string) *Module {
	return d.modulesByPath[path]
}

// IterateModule iterates all modules and passes them to the specified callback f.
func (d *CodeAnalyzer) IterateModule(f func(*Module)) {
	if d.stdModule != nil {
		f(d.stdModule)
	}
	if d.wdModule != nil {
		f(d.wdModule)
	}
	for i := range d.nonToolchainModules {
		m := &d.nonToolchainModules[i]
		if m != d.wdModule {
			f(m)
		}
	}
}

func (d *CodeAnalyzer) regObjectReference(obj types.Object, fileInfo *SourceFileInfo, id *ast.Ident) {
	if d.objectRefs == nil {
		d.objectRefs = map[types.Object][]Identifier{} // ToDo: estimate an initial minimum capacity
	}

	ids := d.objectRefs[obj]
	if ids == nil {
		ids = make([]Identifier, 0, 2) // ToDo: estimate an initial minimum capacity. How?
	}
	ids = append(ids, Identifier{FileInfo: fileInfo, AstIdent: id})
	d.objectRefs[obj] = ids
}

// ObjectReferences returns all the references to the given object.
func (d *CodeAnalyzer) ObjectReferences(obj types.Object) []Identifier {
	ids := d.objectRefs[obj]
	if ids == nil {
		return nil
	}
	dups := make([]Identifier, len(ids))
	copy(dups, ids)
	return dups
}

// Please reset it after using.
func (d *CodeAnalyzer) tempTypeLookupTable() map[uint32]struct{} {
	if d.tempTypeLookup == nil {
		d.tempTypeLookup = make(map[uint32]struct{}, 1024)
	}
	return d.tempTypeLookup
}

func (d *CodeAnalyzer) resetTempTypeLookupTable() {
	// the gc compiler optimize this
	for k := range d.tempTypeLookup {
		delete(d.tempTypeLookup, k)
	}
}

// NumPackages returns packages count.
func (d *CodeAnalyzer) NumPackages() int {
	return len(d.packageList)
}

// PackageAt returns the packages at specified index i.
func (d *CodeAnalyzer) PackageAt(i int) *Package {
	return d.packageList[i]
}

// PackageByPath returns the packages corresponding the specified path.
func (d *CodeAnalyzer) PackageByPath(path string) *Package {
	return d.packageTable[path]
}

// IsStandardPackage returns whether or not the given package is a standard package.
func (d *CodeAnalyzer) IsStandardPackage(pkg *Package) bool {
	return pkg.module == d.stdModule
}

// IsStandardPackageByPath returns whether or not the package specified by the path is a standard package.
func (d *CodeAnalyzer) IsStandardPackageByPath(path string) bool {
	pkg, ok := d.packageTable[path]
	return ok && d.IsStandardPackage(pkg)
}

// BuiltinPackge returns the builtin package.
func (d *CodeAnalyzer) BuiltinPackge() *Package {
	return d.builtinPkg
}

// NumSourceFiles returns the source files count.
func (d *CodeAnalyzer) NumSourceFiles() int {
	//return len(d.sourceFile2PackageTable) // including generated files and non-go files
	return len(d.allSourceFiles) // including generated files and non-go files
}

// Must be callsed after stats.FilesWithGenerateds is confirmed.
func (d *CodeAnalyzer) buildSourceFileTable() {
	d.allSourceFiles = make(map[string]*SourceFileInfo, d.stats.FilesWithGenerateds)
	for _, pkg := range d.packageList {
		for i := range pkg.SourceFiles {
			f := &pkg.SourceFiles[i]
			d.allSourceFiles[pkg.Path+"/"+f.AstBareFileName()] = f
		}
	}
}

// SourceFile returns the source file coresponding the specified file name.
// pkgFile: pkg.Path + "/" + file.BareFilename
func (d *CodeAnalyzer) SourceFile(pkgFile string) *SourceFileInfo {
	return d.allSourceFiles[pkgFile]
}

func (d *CodeAnalyzer) ExampleFileSet() *token.FileSet {
	return d.exampleFileSet
}

//func (d *CodeAnalyzer) SourceFile2Package(path string) (*Package, bool) {
//	srcFile, ok := d.sourceFile2PackageTable[path]
//	return srcFile, ok
//}

//func (d *CodeAnalyzer) SourceFileLineOffset(path string) int {
//	return int(d.sourceFileLineOffsetTable[path])
//}

//func (d *CodeAnalyzer) OriginalGoSourceFile(filename string) string {
//	if f, ok := d.generatedFile2OriginalFileTable[filename]; ok {
//		return f
//	}
//	return filename
//}

// RuntimeFunctionCodePosition returns the position of the specified runtime function f.
func (d *CodeAnalyzer) RuntimeFunctionCodePosition(f string) token.Position {
	return d.runtimeFuncPositions[f]
}

// RuntimePackage returns the runtime package.
func (d *CodeAnalyzer) RuntimePackage() *Package {
	return d.PackageByPath("runtime")
}

// Id1 builds an id from the specified package and identifier name.
// The result is the same as go/types.Id.
func (d *CodeAnalyzer) Id1(p *types.Package, name string) string {
	if p == nil {
		p = d.builtinPkg.PPkg.Types
	}

	return types.Id(p, name)
}

// Id1b builds an id from the specified package and identifier name.
// The result is almost the same as go/types.Id.
func (d *CodeAnalyzer) Id1b(pkg *Package, name string) string {
	if pkg == nil {
		return d.Id1(nil, name)
	}

	return d.Id1(pkg.PPkg.Types, name)
}

// ToDo: avoid string concating by using a struct{pkg; name} for Id and Id2 functions.

// Id2 builds an id from the specified package and identifier name.
func (d *CodeAnalyzer) Id2(p *types.Package, name string) string {
	if p == nil {
		p = d.builtinPkg.PPkg.Types
	}

	return p.Path() + "." + name
}

// Id2b builds an id from the specified package and identifier name.
func (d *CodeAnalyzer) Id2b(pkg *Package, name string) string {
	if pkg == nil {
		return d.Id2(nil, name)
	}

	return d.Id2(pkg.PPkg.Types, name)
}

// Declared functions.
//func (d *CodeAnalyzer) RegisterFunction(f *Function) {
//	// meaningful?
//}

// RegisterTypeName registers a TypeName.
func (d *CodeAnalyzer) RegisterTypeName(tn *TypeName) {
	if d.allTypeNameTable == nil {
		d.allTypeNameTable = make(map[string]*TypeName, 4096)
	}
	tn.index = d.lastTypeNameIndex
	d.lastTypeNameIndex++
	if name := tn.Name(); name != "_" {
		//d.allTypeNameTable[tn.Id()] = tn
		// Unify the id generations.
		// !!! For builtin types, tn.TypeName.Pkg() == nil
		//d.allTypeNameTable[types.Id(tn.TypeName.Pkg(), tn.TypeName.Name())] = tn

		//d.allTypeNameTable[types.Id(tn.Pkg.PPkg.Types, tn.TypeName.Name())] = tn
		d.allTypeNameTable[d.Id2b(tn.Pkg, tn.TypeName.Name())] = tn

		//log.Println(">>>", types.Id(tn.Pkg.PPkg.Types, tn.TypeName.Name()))
	}
}

// RegisterType registers a go/types.Type as TypeInfo.
func (d *CodeAnalyzer) RegisterType(t types.Type) *TypeInfo {
	return d.registeringType(t, true)
}

// LookForType trys to find out the TypeInfo registered for the spefified types.Type.
func (d *CodeAnalyzer) LookForType(t types.Type) *TypeInfo {
	return d.registeringType(t, false)
}

func (d *CodeAnalyzer) registeringType(t types.Type, createOnNonexist bool) *TypeInfo {
	if t == nil {
		t = types.Typ[types.Invalid]
	}

	typeInfo, _ := d.ttype2TypeInfoTable.At(t).(*TypeInfo)
	if typeInfo == nil && createOnNonexist {
		if d.forbidRegisterTypes {
			if d.debug {
				log.Printf("================================= %v, %T", t, t)
			}
		}

		//d.lastTypeIndex++ // the old design (1-based)
		typeInfo = &TypeInfo{TT: t, index: d.lastTypeIndex}
		//if st, ok := t.(*types.Struct); ok {
		//	if st.NumFields() == 1 {
		//		ft := st.Field(0).Type()
		//		if nt, ok := ft.(*types.Named); ok {
		//			if true &&
		//				nt.Obj().Name() == "Type" &&
		//				nt.Obj().Pkg().Name() == "types" &&
		//				st.Field(0).Name() == "Type" {
		//				log.Println("=====================", nt.Obj().Pkg())
		//				debug.PrintStack()
		//				//d.debug = true
		//				iiiii = d.lastTypeIndex
		//			}
		//		}
		//	}
		//}
		d.ttype2TypeInfoTable.Set(t, typeInfo)
		if d.allTypeInfos == nil {
			d.allTypeInfos = make([]*TypeInfo, 0, 8192)

			// The old design ensure all type index > 0,
			// which maight be an unnecessary design.
			//d.allTypeInfos = append(d.allTypeInfos, nil)
		}
		d.lastTypeIndex++ // the new design (0-based)
		d.allTypeInfos = append(d.allTypeInfos, typeInfo)

		switch t := t.(type) {
		case *types.Named:
			//>> 1.18, todo
			// Fake underlying for instantiated types.
			// A temp handling to avoid high code complexity.
			if originType(t) != t {
				typeInfo.Underlying = d.blankInterface
				break
			}
			//<<

			//typeInfo.Name = t.Obj().Name()

			underlying := d.RegisterType(t.Underlying())
			typeInfo.Underlying = underlying
			//underlying.Underlying = underlying // already done

			//numNameds++
			//if _, ok := t.Underlying().(*types.Interface); ok {
			//	numNamedInterfaces++
			//}
		default:
			typeInfo.Underlying = typeInfo
		}

		switch t.Underlying().(type) {
		case *types.Pointer:
			// The exception is to avoid infinite pointer depth.
		case *types.Interface:
			// Pointers of interfaces are not important.
		default:
			// *T might have methods if T is neigher an interface nor pointer type.
			d.RegisterType(types.NewPointer(t))
		}
	}
	return typeInfo
}

// RetrieveTypeName trys to retrieve the TypeName from a TypeInfo.
func (d *CodeAnalyzer) RetrieveTypeName(t *TypeInfo) (*TypeName, bool) {
	if tn := t.TypeName; tn != nil {
		return tn, false
	}

	if ptt, ok := t.TT.(*types.Pointer); ok {
		bt := d.RegisterType(ptt.Elem())
		if btn := bt.TypeName; btn != nil {
			return btn, true
		}

		if _, ok := bt.TT.(*types.Named); ok {
			panic("the base type must have been registered before calling this function")
		}
	}

	return nil, false
}

// Methods contribute to type implementations.
// The key is typeIndex << 32 | methodIndex.
// typeIndex must be the index of a non-interface type.
//
// typeMethodsContributingToTypeImplementations map[uint64]]struct{}
//
//func (d *CodeAnalyzer) registerTypeMethodContributingToTypeImplementations(typeIndex, methodIndex uint32) {
//	if d.typeMethodsContributingToTypeImplementations == nil {
//		d.typeMethodsContributingToTypeImplementations = make(map[int64]struct{}, d.lastTypeIndex*3)
//	}
//	key := uint64(typeIndex)<<32 | uint64(methodIndex)
//	d.typeMethodsContributingToTypeImplementations[key] = struct{}{}
//}
//
//// typeIndex must be the index of a non-interface type.
//func (d *CodeAnalyzer) CheckTypeMethodContributingToTypeImplementations(typeIndex, methodIndex uint32) bool {
//	key := uint64(typeIndex)<<32 | uint64(methodIndex)
//	_, ok := d.typeMethodsContributingToTypeImplementations[key]
//	return ok
//}

func (d *CodeAnalyzer) registerTypeMethodContributingToTypeImplementations(pkg, typ, methodPkg, method string) {
	if d.typeMethodsContributingToTypeImplementations == nil {
		d.typeMethodsContributingToTypeImplementations = make(map[[4]string]struct{}, d.lastTypeIndex*3)
	}
	d.typeMethodsContributingToTypeImplementations[[4]string{pkg, typ, methodPkg, method}] = struct{}{}
}

// CheckTypeMethodContributingToTypeImplementations checks whether or not a method implements some interface methods.
func (d *CodeAnalyzer) CheckTypeMethodContributingToTypeImplementations(pkg, typ, methodPkg, method string) bool {
	_, ok := d.typeMethodsContributingToTypeImplementations[[4]string{pkg, typ, methodPkg, method}]
	return ok
}

// CleanImplements returns a clean list of the implementions for a TypeInfo.
func (d *CodeAnalyzer) CleanImplements(self *TypeInfo) []Implementation {
	// remove:
	// * self
	// * unnameds whose underlied names are also in the list (or are self)
	// The ones in internal packages are kept.

	typeLookupTable := d.tempTypeLookupTable()
	defer d.resetTempTypeLookupTable()

	if itt, ok := self.TT.Underlying().(*types.Interface); ok {
		typeLookupTable[self.index] = struct{}{}
		ut := d.RegisterType(itt)
		typeLookupTable[ut.index] = struct{}{}
	}

	implements := make([]Implementation, 0, len(self.Implements))
	for _, impl := range self.Implements {
		it := impl.Interface
		if it.TypeName == nil {
			continue
		}
		if _, ok := typeLookupTable[it.index]; ok {
			continue
		}
		typeLookupTable[it.index] = struct{}{}
		ut := d.RegisterType(it.TT.Underlying())
		typeLookupTable[ut.index] = struct{}{}
		implements = append(implements, impl)
	}
	for _, impl := range self.Implements {
		it := impl.Interface
		if it.TypeName != nil {
			continue
		}
		if _, ok := typeLookupTable[it.index]; ok {
			continue
		}
		implements = append(implements, impl)
	}

	return implements
}

func (d *CodeAnalyzer) iterateTypenames(typeLiteral ast.Expr, pkg *Package, onTypeName func(*TypeInfo)) {
	switch node := typeLiteral.(type) {
	default:
		panic(fmt.Sprintf("unexpected ast expression. %T : %v", node, node))
	// 1.18, ToDo
	//>> it is actually a fallthrough to the next branch.
	case *astIndexExpr, *astIndexListExpr:
		tt := pkg.PPkg.TypesInfo.TypeOf(node)
		if tt == nil {
			// ToDo: good?
			if pkg.Path == "unsafe" {
				return
			}
			//log.Println("??? type of node is nil:", node.Name, pkg.Path)
			return
		}
		switch t := tt.(type) {
		default:
			// not interested

			// log.Printf("%T, %v", tt, tt)

			// ToDo: it might be an alias to an unnamed type
			//       To also collect functions for such aliases.

			return // not interested
		case *typesTypeParam: // 1.18
			//log.Printf("%T, %v", tt, tt)
			return // not interested
		case *types.Basic:
		case *types.Named:
			//>> 1.18
			tt = originType(t)
			//<<
		}
		typeInfo := d.RegisterType(tt)
		if typeInfo.TypeName == nil {
			//panic("not a named type")
			return
		}
		onTypeName(typeInfo)
	//<<
	case *ast.Ident:
		tt := pkg.PPkg.TypesInfo.TypeOf(node)
		if tt == nil {
			// ToDo: good?
			if pkg.Path == "unsafe" {
				return
			}
			log.Println("??? type of node is nil:", node.Name, pkg.Path)
			return
		}
		switch t := tt.(type) {
		default:
			// not interested

			// log.Printf("%T, %v", tt, tt)

			// ToDo: it might be an alias to an unnamed type
			//       To also collect functions for such aliases.

			return // not interested
		case *typesTypeParam: // 1.18
			//log.Printf("%T, %v", tt, tt)
			return // not interested
		case *types.Basic:
		case *types.Named:
			//>> 1.18
			tt = originType(t)
			//<<
		}
		typeInfo := d.RegisterType(tt)
		if typeInfo.TypeName == nil {
			//panic("not a named type")
			return
		}
		onTypeName(typeInfo)
	case *ast.SelectorExpr: // ToDo: merge with *ast.IndexExpr, *ast.IndexListExpr branch?
		d.iterateTypenames(node.Sel, pkg, onTypeName)

	case *ast.ParenExpr:
		d.iterateTypenames(node.X, pkg, onTypeName)
	case *ast.StarExpr:
		d.iterateTypenames(node.X, pkg, onTypeName)
	case *ast.ArrayType:
		d.iterateTypenames(node.Elt, pkg, onTypeName)
	case *ast.Ellipsis: // ...Ele
		d.iterateTypenames(node.Elt, pkg, onTypeName)
	case *ast.MapType:
		d.iterateTypenames(node.Key, pkg, onTypeName)
		d.iterateTypenames(node.Value, pkg, onTypeName)
	case *ast.ChanType:
		d.iterateTypenames(node.Value, pkg, onTypeName)
	// To avoid returning too much weak-related results, the following types are ignored now.
	case *ast.FuncType:
	case *ast.StructType:
	case *ast.InterfaceType:
	}
}

// I some forget the meaningfulness of this method.
// It looks this method is to collect complete method lists for types,
// which is important to calculate implementation relations.
//
// As Go 1.18 introduces custom generics, two new possible type exprssion nodes
// (ast.IndexExpr and ast.IndexListExpr, both denote named types) are added,
// so the "unnamed" word in the method name is not accurate now.
func (d *CodeAnalyzer) lookForAndRegisterUnnamedInterfaceAndStructTypes(typeLiteral ast.Node, pkg *Package) {
	// ToDo: move this func to package level?
	var reg = func(n ast.Expr) *TypeInfo {
		tv := pkg.PPkg.TypesInfo.Types[n]
		return d.RegisterType(tv.Type)
	}

	switch node := typeLiteral.(type) {

	//>> 1.18, ToDo
	// If the underlying type of reg(node) is an interfae type,
	// need to find a way to get the ast.Node for the interface type.
	// Need an inverse-lookup from the IndexExpr/IndexListExpr to the generics declaration
	// so that the ast.Node could be found.
	// So, need one more pass:
	// 1. collect all the instances of each generic type, (in a loop "for range allPkgs")
	// 2. in another loop "for range allPkgs", handle all variable types:
	//    "for range pkg.PackageAnalyzeResult.AllVariables"
	// The WriteAstType function might also need some tweaks,
	// by replace the type parameters with type arguments,
	// but might also not (maybe WriteAstType doesn't require to do this).
	//
	// Maybe, at least for some situations,
	// registerDirectFields and registerExplicitlySpecifiedMethods don't need
	// set the ast.Node for Field and Method structs, but the need to handle the cases
	// in which AstFunc or AstField is nil. The ast nodes are used in fields/method listing.

	case *astIndexExpr:
		//log.Printf("======== node = %v, %#v", node, node)
		t := reg(node)
		_ = t

		//>> 1.18, ToDo
		// How to get the instanced ast.Node, with the specified TypeParams?
		//if t.Underlying is interface or struct {
		//	d.registerDirectFields needs ast info
		//    d.registerExplicitlySpecifiedMethods needs ast info
		//}
		//<<

		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Index, pkg)
		return
	case *astIndexListExpr:
		//log.Printf("======== node = %v, %#v", node, node)
		t := reg(node)
		_ = t

		//>> 1.18, ToDo
		// How to get the instanced ast.Node, with the specified TypeParams?
		//if t.Underlying is interface or struct {
		//	d.registerDirectFields needs ast info
		//    d.registerExplicitlySpecifiedMethods needs ast info
		//}
		//<<

		for _, index := range node.Indices {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(index, pkg)
		}
	//<<
	default:
		panic(fmt.Sprintf("unexpected ast expression. %T : %v", node, node))
	case *ast.BadExpr:
		log.Println("encounter BadExpr:", node)
		return
	case *ast.Ident, *ast.SelectorExpr:
		// named types and basic types will be registered from other routes.
		return
	case *ast.ParenExpr:
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.X, pkg)
	case *ast.StarExpr:
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.X, pkg)
	case *ast.ArrayType:
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Elt, pkg)
	case *ast.Ellipsis: // ...Ele
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Elt, pkg)
	case *ast.ChanType:
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Value, pkg)
	case *ast.FuncType:
		reg(node)
		for _, field := range node.Params.List {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)
		}
		if node.Results != nil {
			for _, field := range node.Results.List {
				d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)
			}
		}
	case *ast.MapType:
		reg(node)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Key, pkg)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Value, pkg)
	case *ast.StructType:
		d.registerDirectFields(reg(node), node, pkg)
		return
	case *ast.InterfaceType:
		d.registerExplicitlySpecifiedMethods(reg(node), node, pkg)
		return
	}
}

func (d *CodeAnalyzer) registerDirectFields(typeInfo *TypeInfo, astStructNode *ast.StructType, pkg *Package) {
	if (typeInfo.attributes & directSelectorsCollected) != 0 {
		return
	}
	typeInfo.attributes |= directSelectorsCollected

	register := func(field *Field) {
		if field.Name == "-" { // ??? ToDo: forget what does this mean?
			panic("impossible")
		}
		// ToDo, ToDo2: the handling of ".Pkg" here might be some problematic.
		// 1. The ".Pkg" field is always set in the callers of this function.
		// 2. Is the "sel.Id" calculation for builtinPkg+unexportedField right?
		if !token.IsExported(field.Name) {
			field.Pkg = pkg
			if pkg == d.builtinPkg {
				field.Pkg = nil
			}
		}
		if typeInfo.DirectSelectors == nil {
			typeInfo.DirectSelectors = make([]*Selector, 0, 16)
		}
		sel := &Selector{
			Id:    d.Id1b(field.Pkg, field.Name),
			Field: field,
		}
		typeInfo.DirectSelectors = append(typeInfo.DirectSelectors, sel)
	}

	for _, field := range astStructNode.Fields.List {
		if len(field.Names) == 0 {
			//var id string
			var fieldName string

			var isStar = false
			for ok, node := true, field.Type; ok; ok = isStar {
				switch expr := node.(type) {
				default:
					panic("not an embedded field but should be. type: " + fmt.Sprintf("%T", expr))
				//>> ToDo: Go 1.18
				case *astIndexExpr:
					node = expr.X
					continue
				case *astIndexListExpr:
					node = expr.X
					continue
				//<<
				case *ast.Ident:
					//id = d.Id1b(pkg, expr.Name) // incorrect for builtin typenames

					//tn := pkg.PPkg.TypesInfo.Uses[expr]
					//id = d.Id2(tn.Pkg(), expr.Name)
					fieldName = expr.Name
				case *ast.SelectorExpr:
					//srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr.X.(*ast.Ident))
					//srcPkg := srcObj.(*types.PkgName)
					//id = d.Id2(srcPkg.Imported(), expr.Sel.Name)
					fieldName = expr.Sel.Name
				case *ast.StarExpr:
					if isStar {
						panic("bad embedded field **T.")
					}

					node = expr.X
					isStar = true
					continue
				}

				break
			}

			//tn := d.allTypeNameTable[id]
			//if tn == nil {
			//	panic("TypeName for " + id + " not found")
			//}
			//
			//fieldTypeInfo := tn.Named
			//if fieldTypeInfo == nil {
			//	fieldTypeInfo = tn.Alias.Denoting
			//}
			//
			// fieldName := tn.Name()

			tv := pkg.PPkg.TypesInfo.Types[field.Type]

			// ToDo: if field.Type is an interface or struct, or pointer to interface or struct, collect direct selectors.
			// or even disassemble any complex types and look for struct and interface types.

			fieldTypeInfo := d.RegisterType(tv.Type)

			embedMode := EmbedMode_Direct
			if isStar {
				//fieldTypeInfo = d.RegisterType(types.NewPointer(fieldTypeInfo.TT))
				embedMode = EmbedMode_Indirect
			}

			var tag string
			if field.Tag != nil {
				tag = field.Tag.Value
			}

			typeInfo.EmbeddingFields++

			register(&Field{
				Pkg:  pkg,
				Name: fieldName,
				Type: fieldTypeInfo,
				Mode: embedMode,
				Tag:  tag,

				astStruct: astStructNode,
				AstField:  field,
			})

			continue
		}

		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)

		tv := pkg.PPkg.TypesInfo.Types[field.Type]

		// ToDo: if field.Type is an interface or struct, or pointer to interface or struct, collect direct selectors.
		// or even disassemble any complex types and look for struct and interface types.

		fieldTypeInfo := d.RegisterType(tv.Type)

		var tag string
		if field.Tag != nil {
			tag = field.Tag.Value
		}

		for _, ident := range field.Names {
			if ident.Name == "_" {
				continue
			}

			register(&Field{
				Pkg:  pkg,
				Name: ident.Name,
				Type: fieldTypeInfo,
				Mode: EmbedMode_None,
				Tag:  tag,

				astStruct: astStructNode,
				AstField:  field,
			})
		}
	}
}

func (d *CodeAnalyzer) registerParameterAndResultTypes(astFunc *ast.FuncType, pkg *Package) {
	//log.Println("=========================", f.Pkg.Path, f.Name())

	if astFunc.Params != nil {
		for _, fld := range astFunc.Params.List {
			tt := pkg.PPkg.TypesInfo.TypeOf(fld.Type)
			if tt == nil {
				log.Println("tt is nil!")
				continue
			}
			d.RegisterType(tt)
		}
	}

	if astFunc.Results != nil {
		for _, fld := range astFunc.Results.List {
			tt := pkg.PPkg.TypesInfo.TypeOf(fld.Type)
			if tt == nil {
				log.Println("tt is nil!")
				continue
			}
			d.RegisterType(tt)
		}
	}
}

// ToDo: also register function variables?
// This function is to ensure that the selectors of unnamed types are all confirmed before comfirming selectors for all types.
func (d *CodeAnalyzer) registerUnnamedInterfaceAndStructTypesFromParametersAndResults(astFunc *ast.FuncType, pkg *Package) {
	//log.Println("=========================", f.Pkg.Path, f.Name())

	if astFunc.Params != nil {
		for _, fld := range astFunc.Params.List {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(fld.Type, pkg)
		}
	}

	if astFunc.Results != nil {
		for _, fld := range astFunc.Results.List {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(fld.Type, pkg)
		}
	}
}

// ToDo: now interface{Error() string} and interface{error} will be viewed as one TypeInfo,
//
//	which is true from Go senmatics view, but might be not good from code analysis view.
func (d *CodeAnalyzer) registerExplicitlySpecifiedMethods(typeInfo *TypeInfo, astInterfaceNode *ast.InterfaceType, pkg *Package) {
	//if (typeInfo.attributes & directSelectorsCollected) != 0 {
	//	return
	//}

	// The logic of the above three lines is not right as it looks.
	// In the current go/* packages implementation, "interface{A}" and "type A interface {M{}}"
	// will be viewed as identical types. If they are passed by the above order to this function, ...
	// So now the above three lines are disabled.

	// Another detail is that, unlike embedded fields, the embedded type names in an interface type can be identical.

	typeInfo.attributes |= directSelectorsCollected

	registerMethod := func(method *Method) {
		if method.Name == "-" {
			panic("impossible")
		}
		if !token.IsExported(method.Name) {
			method.Pkg = pkg
			if pkg == d.builtinPkg {
				method.Pkg = nil
			}
		}

		if typeInfo.DirectSelectors == nil {
			typeInfo.DirectSelectors = make([]*Selector, 0, 16)
		}

		newId := d.Id1b(method.Pkg, method.Name)

		for _, old := range typeInfo.DirectSelectors {
			// A method and some field names can be the same.
			if old.Id == newId && old.Method != nil {
				// See the comment at the starting of the function.
				return
			}
		}

		sel := &Selector{
			Id:     newId,
			Method: method,
		}
		typeInfo.DirectSelectors = append(typeInfo.DirectSelectors, sel)
	}

	// ToDo: since Go 1.14, two same name interface names can be embedded together.
	registerField := func(field *Field) {
		if field.Name == "-" {
			panic("impossible")
		}
		if !token.IsExported(field.Name) {
			field.Pkg = pkg
			if pkg == d.builtinPkg {
				field.Pkg = nil
			}
		}

		if typeInfo.DirectSelectors == nil {
			typeInfo.DirectSelectors = make([]*Selector, 0, 16)
		}

		newId := d.Id1b(field.Pkg, field.Name)

		// Two embedded types in an interface type can have an identical name.
		//for _, old := range typeInfo.DirectSelectors {
		//	if old.Id == newId {
		//		// See the comment at the starting of the function.
		//		return
		//	}
		//}

		sel := &Selector{
			Id:    newId,
			Field: field,
		}
		typeInfo.DirectSelectors = append(typeInfo.DirectSelectors, sel)
	}

	//log.Println("!!!!!! registerExplicitlySpecifiedMethods:", typeInfo)

	for _, method := range astInterfaceNode.Methods.List {
		// method is a *ast.Field.

		if len(method.Names) == 0 {
			var id string
			switch expr := method.Type.(type) {
			default:
				// embed interface type (anonymous field)

				//>> 1.18
				//
				// Now, it is also possible
				// * an unnamed type
				// * an instantiated type
				// * a type union (ast.BinaryExpr)
				// * ~aType (ast.UnaryExpr)
				continue
				//<<

				//panic(fmt.Sprintf("not a valid embedding interface type name: %#v", method))
			case *ast.Ident:
				ttn := pkg.PPkg.TypesInfo.Uses[expr]
				id = d.Id2(ttn.Pkg(), ttn.Name())
			case *ast.SelectorExpr:
				srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr.X.(*ast.Ident))
				srcPkg := srcObj.(*types.PkgName)
				id = d.Id2(srcPkg.Imported(), expr.Sel.Name)
			}

			tn := d.allTypeNameTable[id]
			if tn == nil {
				panic("TypeName for " + id + " not found")
			}

			fieldTypeInfo := tn.Named
			if fieldTypeInfo == nil {
				fieldTypeInfo = tn.Alias.Denoting
			}
			embedMode := EmbedMode_Direct

			//if strings.Index(id, "image") >= 0 {
			//log.Println("!!!!!!!!!!! ", id, fieldTypeInfo)
			//}

			registerField(&Field{
				Pkg:  pkg,
				Name: tn.Name(),
				Type: fieldTypeInfo,
				Mode: embedMode,

				//AstInterface: astInterfaceNode,
				AstField: method,
			})

		} else {
			//log.Println("   method")
			if len(method.Names) != 1 {
				panic(fmt.Sprint("number of method names is not 1: ", len(method.Names), method.Names))
			}

			ident := method.Names[0]
			if ident.Name == "_" {
				continue
			}

			tv := pkg.PPkg.TypesInfo.Types[method.Type]
			methodTypeInfo := d.RegisterType(tv.Type)
			if pkg == d.builtinPkg && ident.Name == "Error" {
				// The special handling is to correctly find all implementations of the builtin "error" type.
				errorUnderlyingType := types.Universe.Lookup("error").(*types.TypeName).Type().Underlying().(*types.Interface)
				// ToDo: optimization: the result should be cached
				methodTypeInfo = d.RegisterType(errorUnderlyingType.Method(0).Type())
			}

			m := &Method{
				Pkg:  pkg,
				Name: ident.Name,
				Type: methodTypeInfo,

				PointerRecv: false,

				//AstInterface: astInterfaceNode,
				AstField: method,
			}
			//>> 1.18, ToDo
			//m.Parameterized = checkParameterized(m.Type)
			//<<

			registerMethod(m)

			astFunc, ok := method.Type.(*ast.FuncType)
			if !ok {
				panic("should not")
			}
			d.registerUnnamedInterfaceAndStructTypesFromParametersAndResults(astFunc, pkg)
			d.registerParameterAndResultTypes(astFunc, pkg)
		}
	}
	//log.Println("       registerExplicitlySpecifiedMethods:", len(typeInfo.DirectSelectors))
}

// ToDo: to loop parameter and result lists and use AST to constract custom methods.
func (d *CodeAnalyzer) registerExplicitlyDeclaredMethod(f *Function) {
	funcObj, funcDecl, pkg := f.Func, f.AstDecl, f.Pkg

	funcName := funcObj.Name()
	if funcName == "-" {
		return
	}

	sig := funcObj.Type().(*types.Signature)
	recv := sig.Recv()
	if recv == nil {
		return
	}

	d.registerUnnamedInterfaceAndStructTypesFromParametersAndResults(f.AstDecl.Type, f.Pkg)
	d.registerParameterAndResultTypes(f.AstDecl.Type, f.Pkg)

	recvTT := recv.Type()
	var baseTT *types.Named
	var ptrRecv bool
	switch tt := recvTT.(type) {
	case *types.Named:
		baseTT = tt
		ptrRecv = false
	case *types.Pointer:
		baseTT = tt.Elem().(*types.Named)
		ptrRecv = true
	default:
		panic("impossible")
	}

	// ToDo: using sig.Params() and sig.Results() instead of funcObj.Type()

	//>> 1.18
	baseTT = originType(baseTT)
	//<<

	typeInfo := d.RegisterType(baseTT)
	if typeInfo.DirectSelectors == nil {
		selectors := make([]*Selector, 0, 16)
		typeInfo.DirectSelectors = selectors
	}
	method := &Method{
		Pkg:         pkg, // ToDo: research why must set it?
		Name:        funcName,
		Type:        d.RegisterType(funcObj.Type()),
		PointerRecv: ptrRecv,
		AstFunc:     funcDecl,
	}
	//>> 1.18, ToDo
	//method.Parameterized = checkParameterized(method.Type)
	//<<
	if !token.IsExported(funcName) {
		method.Pkg = pkg
		if pkg == d.builtinPkg {
			method.Pkg = nil
		}
	}
	sel := &Selector{
		Id:     d.Id1b(method.Pkg, method.Name),
		Method: method,
	}
	typeInfo.DirectSelectors = append(typeInfo.DirectSelectors, sel)
}

// ToDo: also register function variables?
// Return parameter and result counts.
// func (d *CodeAnalyzer) registerFunctionForInvolvedTypeNames(f *Function) (ins, outs int, lastResultIsError bool) {
func (d *CodeAnalyzer) registerFunctionForInvolvedTypeNames(f FunctionResource) (ins, outs int, lastResultIsError bool) {
	// ToDo: unepxorted function should also reged,
	//       but then they should be filtered out when in listing.
	//notToReg := !f.Exported()

	//fType := f.AstDecl.Type
	fType := f.AstFuncType()

	//log.Println("=========================", f.Pkg.Path, f.Name())

	if fType.Params != nil {
		for _, fld := range fType.Params.List {
			if n := len(fld.Names); n == 0 {
				ins++
			} else {
				ins += n
			}
			//if notToReg {
			//	continue
			//}
			//d.iterateTypenames(fld.Type, f.Package(), func(t *TypeInfo) {
			d.iterateTypenames(fld.Type, f.AstPackage(), func(t *TypeInfo) {
				if t.TypeName == nil {
					panic("shoud not")
				}
				//if t.Pkg == nil {
				//	log.Println("================", f.Pkg.Path, t.TT)
				//}
				if t.TypeName.Pkg.Path == "builtin" {
					return
				}
				if t.AsInputsOf == nil {
					t.AsInputsOf = make([]ValueResource, 0, 4)
				}
				t.AsInputsOf = append(t.AsInputsOf, f.(ValueResource))
			})
		}
	}

	if fType.Results != nil {
		for _, fld := range fType.Results.List {
			lastResultIsError = false

			if n := len(fld.Names); n == 0 {
				outs++
			} else {
				outs += n
			}
			//if notToReg {
			//	continue
			//}
			//d.iterateTypenames(fld.Type, f.Package(), func(t *TypeInfo) {
			d.iterateTypenames(fld.Type, f.AstPackage(), func(t *TypeInfo) {
				if t.TypeName == nil {
					panic("shoud not")
				}
				//if t.Pkg == nil {
				//	log.Println("================", f.Pkg.Path, t.TT)
				//}
				if t.TypeName.Pkg.Path == "builtin" {
					if t.TypeName.Name() == "error" {
						lastResultIsError = true
					}
					return
				}
				if t.AsOutputsOf == nil {
					t.AsOutputsOf = make([]ValueResource, 0, 4)
				}
				t.AsOutputsOf = append(t.AsOutputsOf, f.(ValueResource))
			})
		}
	}

	return
}

func (d *CodeAnalyzer) registerValueForItsTypeName(res ValueResource) {

	//>> 1.18, ToDo
	// Now, for an instantiated type, t.TypeName is nil.
	// ToDo: in d.registeringType, if t.TT is found a *types.Named,
	//       then find the Origin type, and register the value on that origin type.
	t := res.TypeInfo(d)
	//<<
	toRegsiter := t.TypeName != nil

	//if d.debug {
	//	log.Printf("======= toRegsiter %v, === %v, === %v, ", toRegsiter, t.TypeName, res)
	//}

	if !toRegsiter {
		//>> 1.18
		originNamedType := func(t *TypeInfo) *TypeInfo {
			if ntt, ok := t.TT.(*types.Named); ok {
				if ott := originType(ntt); ntt != ott {
					ot := d.RegisterType(ott)
					if ot.TypeName != nil {
						return ot
					}
				}
			}
			return nil
		}

		if ot := originNamedType(t); ot != nil {
			toRegsiter = true
			t = ot
			goto Done
		}
		//<<

		// ToDo: also for []T, [N]T, chan T, etc.
		switch tt := t.TT.(type) {
		// ToDo: also consider []T, [..]T, chan T, map[K]T, ... ?
		case *types.Pointer:
			bt := d.RegisterType(tt.Elem())
			// ToDo: also register if an unnamed type has some aliases
			if bt.TypeName != nil {
				//log.Println("========= t=", t)
				toRegsiter = true
				goto Done
			}

			//>> 1.18
			if ot := originNamedType(bt); ot != nil {
				toRegsiter = true
				t = d.RegisterType(types.NewPointer(ot.TT))
				goto Done
			}
			//<<
		}
	}

Done:
	if toRegsiter {

		if t.AsTypesOf == nil {
			t.AsTypesOf = make([]ValueResource, 0, 4)
		}
		t.AsTypesOf = append(t.AsTypesOf, res)
	}
}

// BuildMethodSignatureFromFuncObject builds the signature for function object.
//func (d *CodeAnalyzer) BuildMethodSignatureFromFuncObject(funcObj *types.Func) MethodSignature {
//	funcSig, ok := funcObj.Type().(*types.Signature)
//	if !ok {
//		panic(funcObj.Id() + "'s type is not types.Signature")
//	}
//
//	methodName, pkgImportPath := funcObj.Id(), ""
//	if !token.IsExported(methodName) {
//		pkgImportPath = funcObj.Pkg().Path
//	}
//
//	return d.BuildMethodSignatureFromFunctionSignature(funcSig, methodName, pkgImportPath)
//}

// BuildMethodSignatureFromFunctionSignature  builds the signature for method function object.
// pkgImportPath should be only passed for unexported method names.
func (d *CodeAnalyzer) BuildMethodSignatureFromFunctionSignature(funcSig *types.Signature, methodName string, pkgImportPath string) MethodSignature {
	if pkgImportPath != "" {
		if token.IsExported(methodName) {
			//panic("bad argument: " + pkgImportPath + "." + methodName)
			// ToDo: handle this case gracefully.
			//log.Println("bad argument: " + pkgImportPath + "." + methodName)
			pkgImportPath = ""
		} else if pkgImportPath == "builtin" {
			log.Println("bad argument: " + pkgImportPath + "." + methodName)
			pkgImportPath = ""
		}
	}

	var b strings.Builder
	var writeTypeIndex = func(index uint32) {
		b.WriteByte(byte((index >> 24) & 0xFF))
		b.WriteByte(byte((index >> 16) & 0xFF))
		b.WriteByte(byte((index >> 8) & 0xFF))
		b.WriteByte(byte((index >> 0) & 0xFF))
	}

	params, results := funcSig.Params(), funcSig.Results()

	n := 4 * (params.Len() + results.Len())
	b.Grow(n)

	//inouts := make([]byte, n)
	//cursor := 0
	for i := params.Len() - 1; i >= 0; i-- {
		typeIndex := d.RegisterType(params.At(i).Type()).index
		//binary.LittleEndian.PutUint32(inouts[cursor:], typeIndex)
		//cursor += 4
		writeTypeIndex(typeIndex)
	}
	for i := results.Len() - 1; i >= 0; i-- {
		typeIndex := d.RegisterType(results.At(i).Type()).index
		//binary.LittleEndian.PutUint32(inouts[cursor:], typeIndex)
		//cursor += 4
		writeTypeIndex(typeIndex)
	}

	counts := params.Len()<<16 | results.Len()
	if funcSig.Variadic() {
		counts = -counts
	}

	//log.Println("?? full name:", funcObj.Id())
	//log.Println("   counts:   ", counts)
	//log.Println("   inouts:   ", inouts)

	return MethodSignature{
		//InOutTypes:          string(inouts),
		InOutTypes:          b.String(),
		NumInOutAndVariadic: counts,
		Name:                methodName,
		Pkg:                 pkgImportPath,
	}
}
