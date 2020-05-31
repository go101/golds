package code

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"strings"

	"golang.org/x/tools/go/types/typeutil"
)

const (
	SubTask_PreparationDone = iota
	SubTask_NFilesParsed
	SubTask_ParsePackagesDone
	SubTask_CollectPackages
	SubTask_SortPackagesByDependencies
	SubTask_CollectDeclarations
	SubTask_CollectRuntimeFunctionPositions
	SubTask_FindTypeSources
	SubTask_CollectSelectors
	SubTask_CheckCollectedSelectors
	SubTask_FindImplementations
	SubTask_RegisterInterfaceMethodsForTypes
	SubTask_MakeStatistics
	SubTask_CollectSourceFiles
)

type CodeAnalyzer struct {
	allModules []*Module
	stdModule  *Module

	//stdPackages  map[string]struct{}
	packageTable map[string]*Package
	packageList  []*Package
	builtinPkg   *Package

	// This one is removed now. We should use FileSet.PositionFor.
	//sourceFileLineOffsetTable map[string]int32

	//=== Will be fulfilled in analyse phase ===

	//sourceFile2PackageTable  map[string]SourceFile
	sourceFile2PackageTable         map[string]*Package
	generatedFile2OriginalFileTable map[string]string

	// *types.Type -> *TypeInfo
	lastTypeIndex       uint32
	ttype2TypeInfoTable typeutil.Map
	allTypeInfos        []*TypeInfo

	// Package-level declared type names.
	lastTypeNameIndex uint32
	allTypeNameTable  map[string]*TypeName

	// Position info of some runtime functions.
	runtimeFuncPositions map[string]token.Position

	// Refs of unnamed types, type names, variables, functions, ...
	// Why not put []RefPos in TypeInfo, Variable, ...?
	refPositions map[interface{}][]RefPos

	// Not concurrent safe.
	tempTypeLookup map[uint32]struct{}

	forbidRegisterTypes bool // for debug

	stats Stats
}

const KindCount = reflect.UnsafePointer + 1

type Stats struct {
	Packages,
	AllPackageDeps int32
	PackagesByDeps [64]int32
	//PackagesByImportBys [1024]int32 // use sorting packages by importBys instead.

	FilesWithoutGenerateds, // without generated ones
	FilesWithGenerateds, // with generated ones
	//ToDo: stat code lines. Use the info in AstFile?
	CodeLines,
	BlankCodeLines,
	CommentCodeLines,

	// To calculate imports per file.
	// Deps per packages are available in other ways.
	AstFiles,
	Imports int32
	FilesByImportCount [64]int32

	// Types
	//NamedStructTypesWithEmbeddingField,
	//NamedStructTypeFields
	ExportedTypeNames,
	ExportedTypeAliases int32

	//ExportedTypeAliasesByKind
	ExportedTypeNamesByKind [KindCount]int32
	ExportedNamedIntergerTypes,
	ExportedNamedUnsignedIntergerTypes,
	ExportedNamedNumericTypes int32

	NamedStructsByFieldCount, // including promoteds and non-exporteds
	NamedStructsByExplicitFieldCount, // including non-exporteds but not including promoted
	NamedStructsByExportedFieldCount, // including promoteds
	NamedStructsByExportedExplicitFieldCount, // not including promoteds
	ExportedNamedNonInterfaceTypesByMethodCount, // T and * T combined
	ExportedNamedNonInterfaceTypesByExportedMethodCount, // T and * T combined
	ExportedNamedInterfacesByMethodCount,
	ExportedNamedInterfacesByExportedMethodCount [64]int32 // the last element means (N-1)+

	// Values
	ExportedVariables,
	ExportedConstants,
	Functions,
	ExportedFunctions,
	Methods,
	ExportedMethods int32

	FunctionsByParameterCount,
	MethodsByParameterCount [32]int32 // the last element means (N-1)+
	FunctionsByResultCount,
	MethodsByResultCount [16]int32 // the last element means (N-1)+
}

func incSliceStat(stats []int32, index int) {
	if index >= len(stats) {
		stats[len(stats)-1]++
	} else {
		stats[index]++
	}
}

func (d *CodeAnalyzer) Statistics() Stats {
	return d.stats
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

func (d *CodeAnalyzer) NumPackages() int {
	return len(d.packageList)
}

func (d *CodeAnalyzer) PackageAt(i int) *Package {
	return d.packageList[i]
}

// ToDo: remove the second result
func (d *CodeAnalyzer) PackageByPath(path string) *Package {
	return d.packageTable[path]
}

func (d *CodeAnalyzer) IsStandardPackage(pkg *Package) bool {
	return pkg.Mod == d.stdModule
}

// ToDo: add Standard field.
func (d *CodeAnalyzer) IsStandardPackageByPath(path string) bool {
	pkg, ok := d.packageTable[path]
	return ok && d.IsStandardPackage(pkg)
}

func (d *CodeAnalyzer) BuiltinPackge() *Package {
	return d.builtinPkg
}

func (d *CodeAnalyzer) NumSourceFiles() int {
	return len(d.sourceFile2PackageTable) // including generated files and non-go files
}

func (d *CodeAnalyzer) SourceFile2Package(path string) (*Package, bool) {
	srcFile, ok := d.sourceFile2PackageTable[path]
	return srcFile, ok
}

//func (d *CodeAnalyzer) SourceFileLineOffset(path string) int {
//	return int(d.sourceFileLineOffsetTable[path])
//}

func (d *CodeAnalyzer) OriginalGoSourceFile(filename string) string {
	if f, ok := d.generatedFile2OriginalFileTable[filename]; ok {
		return f
	}
	return filename
}

func (d *CodeAnalyzer) RuntimeFunctionCodePosition(f string) token.Position {
	return d.runtimeFuncPositions[f]
}

func (d *CodeAnalyzer) RuntimePackage() *Package {
	return d.PackageByPath("runtime")
}

func (d *CodeAnalyzer) Id1(p *types.Package, name string) string {
	if p == nil {
		p = d.builtinPkg.PPkg.Types
	}

	return types.Id(p, name)
}

func (d *CodeAnalyzer) Id1b(pkg *Package, name string) string {
	if pkg == nil {
		return d.Id1(nil, name)
	}

	return d.Id1(pkg.PPkg.Types, name)
}

func (d *CodeAnalyzer) Id2(p *types.Package, name string) string {
	if p == nil {
		p = d.builtinPkg.PPkg.Types
	}

	return p.Path() + "." + name
}

func (d *CodeAnalyzer) Id2b(pkg *Package, name string) string {
	if pkg == nil {
		return d.Id2(nil, name)
	}

	return d.Id2(pkg.PPkg.Types, name)
}

// Declared functions.
func (d *CodeAnalyzer) RegisterFunction(f *Function) {
	// meaningful?
}

// The registered name must be a package-level name.
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

func (d *CodeAnalyzer) RegisterType(t types.Type) *TypeInfo {
	return d.TryRegisteringType(t, true)
}

func (d *CodeAnalyzer) TryRegisteringType(t types.Type, createOnNonexist bool) *TypeInfo {
	typeInfo, _ := d.ttype2TypeInfoTable.At(t).(*TypeInfo)
	if typeInfo == nil && createOnNonexist {
		if d.forbidRegisterTypes {
			log.Println("=================================", t)
		}

		//d.lastTypeIndex++ // the old design
		typeInfo = &TypeInfo{TT: t, index: d.lastTypeIndex}
		d.ttype2TypeInfoTable.Set(t, typeInfo)
		if d.allTypeInfos == nil {
			d.allTypeInfos = make([]*TypeInfo, 0, 8192)

			// The old design ensure all type index > 0,
			// which maight be an unnecesary design.
			//d.allTypeInfos = append(d.allTypeInfos, nil)
		}
		d.lastTypeIndex++ // the new design
		d.allTypeInfos = append(d.allTypeInfos, typeInfo)

		switch t := t.(type) {
		case *types.Named:
			//typeInfo.Name = t.Obj().Name()

			underlying := d.RegisterType(t.Underlying())
			typeInfo.Underlying = underlying
			//underlying.Underlying = underlying // already done
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
	case *ast.Ident:
		tt := pkg.PPkg.TypesInfo.TypeOf(node)
		if tt == nil {
			if pkg.Path() == "unsafe" {
				return
			}
			log.Println("node:", node.Name)
		}
		typeInfo := d.RegisterType(tt)
		if typeInfo.TypeName == nil {
			//panic("not a named type")
			return // it might be an alias to an unnamed type
			// ToDo: also collect functions for such aliases.
		}
		onTypeName(typeInfo)
	case *ast.SelectorExpr:
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
	// To avoid return too much weak-related results, the following types are ignored now.
	case *ast.FuncType:
	case *ast.StructType:
	case *ast.InterfaceType:
	}
}

func (d *CodeAnalyzer) lookForAndRegisterUnnamedInterfaceAndStructTypes(typeLiteral ast.Node, pkg *Package) {
	switch node := typeLiteral.(type) {
	default:
		panic(fmt.Sprintf("unexpected ast expression. %T : %v", node, node))
	case *ast.BadExpr:
		log.Println("encounter BadExpr:", node)
	case *ast.Ident, *ast.SelectorExpr:
		// named types and basic types will be registered from other routes.
	case *ast.ParenExpr:
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.X, pkg)
	case *ast.StarExpr:
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.X, pkg)
	case *ast.ArrayType:
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Elt, pkg)
	case *ast.Ellipsis: // ...Ele
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Elt, pkg)
	case *ast.ChanType:
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Value, pkg)
	case *ast.FuncType:
		for _, field := range node.Params.List {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)
		}
		if node.Results != nil {
			for _, field := range node.Results.List {
				d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)
			}
		}
	case *ast.MapType:
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Key, pkg)
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(node.Value, pkg)
	case *ast.StructType:
		tv := pkg.PPkg.TypesInfo.Types[node]
		typeInfo := d.RegisterType(tv.Type)
		d.registerDirectFields(typeInfo, node, pkg)
	case *ast.InterfaceType:
		tv := pkg.PPkg.TypesInfo.Types[node]
		typeInfo := d.RegisterType(tv.Type)
		d.registerExplicitlySpecifiedMethods(typeInfo, node, pkg)
	}

	// ToDo: don't use the std go/types and go/pacakges packages.
	//       Now, uint8 and byte are treated as two types by go/types.
	//       Write a custom one tailored for docs and code analyzing.
	//       Run "go mod tidy" before running gold using the custom packages
	//       to ensure all modules are cached locally.
}

func (d *CodeAnalyzer) registerDirectFields(typeInfo *TypeInfo, astStructNode *ast.StructType, pkg *Package) {
	if (typeInfo.attributes & directSelectorsCollected) != 0 {
		return
	}
	typeInfo.attributes |= directSelectorsCollected

	register := func(field *Field) {
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
		sel := &Selector{
			Id:    d.Id1b(field.Pkg, field.Name),
			Field: field,
		}
		typeInfo.DirectSelectors = append(typeInfo.DirectSelectors, sel)
	}

	for _, field := range astStructNode.Fields.List {
		if len(field.Names) == 0 {
			var id string

			var isStar = false
			for ok, node := true, field.Type; ok; ok = isStar {
				switch expr := node.(type) {
				default:
					panic("not an embedded field but should be. type: " + fmt.Sprintf("%T", expr))
				case *ast.Ident:
					//id = d.Id1b(pkg, expr.Name) // incorrect for builtin typenames

					tn := pkg.PPkg.TypesInfo.Uses[expr]
					id = d.Id2(tn.Pkg(), expr.Name)

				case *ast.SelectorExpr:
					srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr.X.(*ast.Ident))
					srcPkg := srcObj.(*types.PkgName)
					id = d.Id2(srcPkg.Imported(), expr.Sel.Name)
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

			tn := d.allTypeNameTable[id]
			if tn == nil {
				panic("TypeName for " + id + " not found")
			}

			//if tn.Name() == "_" {
			//	continue
			//}

			fieldTypeInfo := tn.Named
			if fieldTypeInfo == nil {
				fieldTypeInfo = tn.Alias.Denoting
			}
			embedMode := EmbedMode_Direct
			if isStar {
				fieldTypeInfo = d.RegisterType(types.NewPointer(fieldTypeInfo.TT))
				embedMode = EmbedMode_Indirect
			}

			var tag string
			if field.Tag != nil {
				tag = field.Tag.Value
			}

			register(&Field{
				Pkg:  pkg,
				Name: tn.Name(),
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

		//todo: if field.Type is an interface or struct, or pointer to interface or struct, collect direct selectors.
		//or even disassenble any complex types and look for struct and interface types.

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
	//log.Println("=========================", f.Pkg.Path(), f.Name())

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
// This funciton is to ensure that the selectors of unnamed types are all confirmed before comfirming selectors for all types.
func (d *CodeAnalyzer) registerUnnamedInterfaceAndStructTypesFromParametersAndResults(astFunc *ast.FuncType, pkg *Package) {
	//log.Println("=========================", f.Pkg.Path(), f.Name())

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

// ToDo: now interface{Error() string} and interface{error} will be viewed as one TypeInfo
func (d *CodeAnalyzer) registerExplicitlySpecifiedMethods(typeInfo *TypeInfo, astInterfaceNode *ast.InterfaceType, pkg *Package) {
	//if (typeInfo.attributes & directSelectorsCollected) != 0 {
	//	return
	//}

	// The logic of the above three lines is not right as it looks.
	// In the current go/* packages implementation, "interface{A}" and "type A interface {M{}}"
	// will be viewed as identicial types. If they are passed by the above order to this function, ...
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
		if len(method.Names) == 0 {
			//log.Println("   embed")
			//continue // embed interface type. ToDo

			var id string
			switch expr := method.Type.(type) {
			default:
				panic("not a valid embedding interface type name")
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

				astInterface: astInterfaceNode,
				AstField:     method,
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
				methodTypeInfo = d.RegisterType(errorUnderlyingType.Method(0).Type())
			}

			registerMethod(&Method{
				Pkg:  pkg,
				Name: ident.Name,
				Type: methodTypeInfo,

				PointerRecv: false,

				astInterface: astInterfaceNode,
				AstField:     method,
			})

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
func (d *CodeAnalyzer) registerFunctionForInvolvedTypeNames(f *Function) (ins, outs int) {
	// ToDo: unepxorted function should also reged,
	//       but then they should be filtered out when in listing.
	notToReg := !f.Exported()
	fType := f.AstDecl.Type

	//log.Println("=========================", f.Pkg.Path(), f.Name())

	if fType.Params != nil {
		for _, fld := range fType.Params.List {
			if n := len(fld.Names); n == 0 {
				ins++
			} else {
				ins += n
			}
			if notToReg {
				continue
			}
			d.iterateTypenames(fld.Type, f.Pkg, func(t *TypeInfo) {
				if t.TypeName == nil {
					panic("shoud not")
				}
				//if t.Pkg == nil {
				//	log.Println("================", f.Pkg.Path(), t.TT)
				//}
				if t.TypeName.Pkg.Path() == "builtin" {
					return
				}
				if t.AsInputsOf == nil {
					t.AsInputsOf = make([]ValueResource, 0, 4)
				}
				t.AsInputsOf = append(t.AsInputsOf, f)
			})
		}
	}

	if fType.Results != nil {
		for _, fld := range fType.Results.List {
			if n := len(fld.Names); n == 0 {
				outs++
			} else {
				outs += n
			}
			if notToReg {
				continue
			}
			d.iterateTypenames(fld.Type, f.Pkg, func(t *TypeInfo) {
				if t.TypeName == nil {
					panic("shoud not")
				}
				//if t.Pkg == nil {
				//	log.Println("================", f.Pkg.Path(), t.TT)
				//}
				if t.TypeName.Pkg.Path() == "builtin" {
					return
				}
				if t.AsOutputsOf == nil {
					t.AsOutputsOf = make([]ValueResource, 0, 4)
				}
				t.AsOutputsOf = append(t.AsOutputsOf, f)
			})
		}
	}

	return
}

func (d *CodeAnalyzer) registerValueForItsTypeName(res ValueResource) {
	t := res.TypeInfo(d)
	toRegsiter := t.TypeName != nil
	if !toRegsiter {
		// ToDo: also for []T, [N]T, chan T, etc.
		switch tt := t.TT.(type) {
		case *types.Pointer:
			bt := d.RegisterType(tt.Elem())
			// ToDo: also register if an unnamed type has some aliases
			if bt.TypeName != nil {
				//log.Println("========= t=", t)
				toRegsiter = true
			}
		}
	}

	if toRegsiter {
		if t.AsTypesOf == nil {
			t.AsTypesOf = make([]ValueResource, 0, 4)
		}
		t.AsTypesOf = append(t.AsTypesOf, res)
	}
}

func (d *CodeAnalyzer) BuildMethodSignatureFromFuncObject(funcObj *types.Func) MethodSignature {
	funcSig, ok := funcObj.Type().(*types.Signature)
	if !ok {
		panic(funcObj.Id() + "'s type is not types.Signature")
	}

	methodName, pkgImportPath := funcObj.Id(), ""
	if !token.IsExported(methodName) {
		pkgImportPath = funcObj.Pkg().Path()
	}

	return d.BuildMethodSignatureFromFunctionSignature(funcSig, methodName, pkgImportPath)
}

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
