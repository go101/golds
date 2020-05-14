package code

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Module struct {
	Dir     string
	Root    string // root import path
	Version string
}

type Package struct {
	Index int
	PPkg  *packages.Package

	Mod      *Module
	Deps     []*Package
	DepLevel int // 0 means the level is not determined yet
	DepedBys []*Package

	// This field might be shared with PackageForDisplay
	// for concurrenct reads.
	*PackageAnalyzeResult
}

func (p *Package) Path() string {
	return p.PPkg.PkgPath // might be prefixed with "vendor/", which is different from import path.
}

type PackageAnalyzeResult struct {
	AllTypeNames []*TypeName
	AllFunctions []*Function
	AllVariables []*Variable
	AllConstants []*Constant
	AllImports   []*Import
	SourceFiles  []SourceFileInfo
}

func NewPackageAnalyzeResult() *PackageAnalyzeResult {
	// ToDo: maybe it is better to run a statistic phase firstly,
	// so that the length of each slice will get knowledged.
	return &PackageAnalyzeResult{
		AllTypeNames: make([]*TypeName, 0, 64),
		AllFunctions: make([]*Function, 0, 64),
		AllVariables: make([]*Variable, 0, 64),
		AllConstants: make([]*Constant, 0, 64),
		AllImports:   make([]*Import, 0, 64),
	}
}

func (r *PackageAnalyzeResult) SourceFileInfoByBareFilename(bareFilename string) *SourceFileInfo {
	for _, info := range r.SourceFiles {
		//if info.OriginalGoFile == srcPath {
		//	return &info
		//}
		//if info.GeneratedFile == srcPath {
		//	return &info
		//}
		if info.BareFilename == bareFilename {
			return &info
		}
		if info.BareGeneratedFilename == bareFilename {
			return &info
		}
	}
	return nil
}

// ToDo: better to maintain a global sourceFilePath => SourceFileInfo table?
//func (r *PackageAnalyzeResult) SourceFileInfo(srcPath string) *SourceFileInfo {
func (r *PackageAnalyzeResult) SourceFileInfoByFilePath(srcPath string) *SourceFileInfo {
	for _, info := range r.SourceFiles {
		if info.OriginalFile == srcPath {
			return &info
		}
		if info.GeneratedFile == srcPath {
			return &info
		}
	}
	return nil
}

type RefPos struct {
	Pkg *Package
	Pos token.Pos
}

type AstNode struct {
	Pkg  *Package
	Node ast.Node
}

type Resource interface {
	Name() string
	Exported() bool
	//IndexString() string
	Documentation() string
	Comment() string
	Position() token.Position
	Package() *Package
}

type ValueResource interface {
	Resource
	TType() types.Type // The result should not be used in comparisons.
	TypeInfo(d *CodeAnalyzer) *TypeInfo
}

type Attribute uint32

const (
	// Runtime only flags.
	analyseCompleted Attribute = 1 << (31 - iota)
	directSelectorsCollected
	promotedSelectorsCollected

	// Higher bits are for runtime-only flags.
	AtributesPersistentMask Attribute = (1 << 25) - 1

	// Caching individual packages seperately might be not a good idea.
	// There are many complexities here.
	// * implementation relations become larger along with more packages are involved.
	// Caching by arguments starting packages, as one file, is simpler.

	// For functions, type aliases and named types.
	Builtin Attribute = 1 << 0

	// For type aliases and named types.
	Embeddable    Attribute = 1 << 1
	PtrEmbeddable Attribute = 1 << 2

	// For unnamed struct and interface types.
	HasUnexporteds Attribute = 1 << 3

	// For all types.
	Defined    Attribute = 1 << 4
	Comparable Attribute = 1 << 5

	// For channel types.
	Sendable   Attribute = 1 << 6
	Receivable Attribute = 1 << 7

	// For funcitons.
	Variadic Attribute = 1 << 8
)

type TypeSource struct {
	TypeName    *TypeName
	UnnamedType *TypeInfo
}

//func (ts *TypeSource) Denoting(d *CodeAnalyzer) *TypeInfo {
//	if ts.UnnamedType != nil {
//		return ts.UnnamedType
//	}
//	return ts.TypeName.Denoting(d)
//}

type TypeName struct {
	// One and only one of the two is nil.
	Alias *TypeAlias
	Named *TypeInfo

	//index uint32 // the global index

	// ToDo: simplify the source defintion.
	// Four kinds of sources to affect promoted selectors:
	// 1. typename
	// 2. *typename
	// 3. unnamed type
	// 4. *unname type
	Source     TypeSource
	StarSource *TypeSource

	UsePositions []token.Position

	*types.TypeName

	index uint32 // ToDo: any useful?

	Pkg     *Package // some duplicated with types.TypeName.Pkg(), except builtin types
	AstDecl *ast.GenDecl
	AstSpec *ast.TypeSpec
}

//func (tn *TypeName) IndexString() string {
//	var b strings.Builder
//
//	b.WriteString(tn.Name())
//	if tn.Alias != nil {
//		b.WriteString(" = ")
//	} else {
//		b.WriteString(" ")
//	}
//	WriteType(&b, tn.AstSpec.Type, tn.Pkg.PPkg.TypesInfo, true)
//
//	return b.String()
//}

//func (tn *TypeName) Id() string {
//	return tn.obj.Id()
//}

//func (tn *TypeName) Name() string {
//	return tn.obj.Name()
//}

func (tn *TypeName) Exported() bool {
	if tn.Pkg.Path() == "builtin" {
		return !token.IsExported(tn.Name())
	}
	return tn.TypeName.Exported()
}

func (tn *TypeName) Position() token.Position {
	return tn.Pkg.PPkg.Fset.PositionFor(tn.AstSpec.Name.Pos(), false)
}

func (tn *TypeName) Documentation() string {
	//doc := tn.AstDecl.Doc.Text()
	//if t := tn.AstSpec.Doc.Text(); t != "" {
	//	doc = doc + "\n\n" + t
	//}
	//return doc
	doc := tn.AstSpec.Doc.Text()
	if doc == "" {
		doc = tn.AstDecl.Doc.Text()
	}
	return doc
}

func (tn *TypeName) Comment() string {
	return tn.AstSpec.Comment.Text()
}

func (tn *TypeName) Package() *Package {
	return tn.Pkg
}

//func (tn *TypeName) Comment() string {
//	return tn.AstSpec.Comment.Text()
//}

//func (tn *TypeName) Denoting(d *CodeAnalyzer) *TypeInfo {
//	if tn.Named != nil {
//		return tn.Named
//	}
//
//	if tn.StarSource != nil {
//		return d.RegisterType(types.NewPointer(tn.StarSource.Denoting(d).TT))
//	}
//
//	return tn.Source.Denoting(d)
//}

func (tn *TypeName) Denoting() *TypeInfo {
	if tn.Named != nil {
		return tn.Named
	}

	return tn.Alias.Denoting
}

//func (tn *TypeName) Underlying(d *CodeAnalyzer) *TypeInfo {
//	if tn.StarSource != nil || tn.Source.UnnamedType != nil {
//		return tn.Denoting(d)
//	}
//	return tn.Source.TypeName.Underlying(d)
//}

type TypeAlias struct {
	Denoting *TypeInfo

	// For named and basic types.
	TypeName *TypeName

	// Builtin, Embeddable.
	attributes Attribute
}

//func (a *TypeAlias) Embeddable() bool {
//	var tc = a.Denoting.Common()
//	if tc.Attributes&Embeddable != 0 {
//		return true
//	}
//	if tc.Kind != Pointer {
//		return false
//	}
//	if _, ok := a.Denoting.(*Type_Named); !ok {
//		return false
//	}
//	tc = a.Denoting.(*Type_Pointer).Common()
//	return tc.Kind&(Ptr|Interface) == 0
//}

type TypeInfo struct {
	TT types.Type

	Underlying *TypeInfo

	//Implements     []*TypeInfo
	///StarImplements []*TypeInfo // if TT is neither pointer nor interface.
	Implements []Implementation

	// For interface types.
	ImplementedBys []*TypeInfo

	// For builtin and unnamed types only.
	Aliases []*TypeAlias

	// For named and basic types.
	TypeName *TypeName

	// For unnamed types.
	UsePositions []token.Position

	// For unnamed interfaces and structs, this field must be nil.
	//Pkg *Package // Looks this field is never used. (It really should not exist in this type.)

	// Including promoted ones. For struct types only.
	// * For named types, only explicitly declared methods are included.
	//   The field is only built for T. (*T).DirectSelectors is always nil.
	// * For named interface types, all explicitly specified methods and embedded types (as fields).
	// * For unnamed struct types, only direct fields. Only built for strct{...}, not for *struct{...}.
	DirectSelectors []*Selector

	// All methods, including extended/promoted ones.
	AllMethods []*Selector

	// All fields, including promoted ones.
	AllFields []*Selector

	// Including promoted ones. For both T and *T.
	//Methods []*Method

	// For .TypeName != nil
	AsTypesOf   []ValueResource // variables and constants
	AsInputsOf  []ValueResource // variables and functions
	AsOutputsOf []ValueResource // variables and functions
	// ToDo: register variables (of function types) for AsInputsOf and AsOutputsOf

	attributes Attribute // ToDo: fill the bits

	// The global type index. It will be
	// used in calculating method signatures.
	// ToDo: check if it is problematic to allow index == 0.
	index uint32

	// Used in several scenarios.
	counter uint32
	//counter2 int32
}

func (t *TypeInfo) Kind() reflect.Kind {
	switch tt := t.TT.Underlying().(type) {
	default:
		log.Printf("unknown kind of type: %T", tt)
		return reflect.Invalid
	case *types.Basic:
		switch bt := tt.Kind(); bt {
		default: // t.TT: builtin.Type, unsafe.ArbitraryType, etc.
			//log.Printf("bad basic kind: %v, %v", bt, t.TT)

			return reflect.Invalid
		case types.Bool:
			return reflect.Bool
		case types.Int:
			return reflect.Int
		case types.Int8:
			return reflect.Int8
		case types.Int16:
			return reflect.Int16
		case types.Int32:
			return reflect.Int32
		case types.Int64:
			return reflect.Int64
		case types.Uint:
			return reflect.Uint
		case types.Uint8:
			return reflect.Uint8
		case types.Uint16:
			return reflect.Uint16
		case types.Uint32:
			return reflect.Uint32
		case types.Uint64:
			return reflect.Uint64
		case types.Uintptr:
			return reflect.Uintptr
		case types.Float32:
			return reflect.Float32
		case types.Float64:
			return reflect.Float64
		case types.Complex64:
			return reflect.Complex64
		case types.Complex128:
			return reflect.Complex128
		case types.String:
			return reflect.String
		case types.UnsafePointer:
			return reflect.UnsafePointer
		}
	case *types.Pointer:
		return reflect.Ptr
	case *types.Struct:
		return reflect.Struct
	case *types.Array:
		return reflect.Array
	case *types.Slice:
		return reflect.Slice
	case *types.Map:
		return reflect.Map
	case *types.Chan:
		return reflect.Chan
	case *types.Signature:
		return reflect.Func
	case *types.Interface:
		return reflect.Interface
	}
}

type Implementation struct {
	Impler    *TypeInfo // a struct or named type (same as the owner), or a pointer to such a type
	Interface *TypeInfo // an interface type
}

type Import struct {
	*types.PkgName

	Pkg     *Package // some duplicated with types.PkgName.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ImportSpec
}

type Constant struct {
	*types.Const

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Const.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ValueSpec
}

func (c *Constant) Position() token.Position {
	for _, n := range c.AstSpec.Names {
		if n.Name == c.Name() {
			return c.Pkg.PPkg.Fset.PositionFor(n.Pos(), false)
		}
	}
	panic("should not")
}

func (c *Constant) Documentation() string {
	doc := c.AstSpec.Doc.Text()
	if doc == "" {
		doc = c.AstDecl.Doc.Text()
	}
	return doc
}

func (c *Constant) Comment() string {
	return c.AstSpec.Comment.Text()
}

func (c *Constant) Package() *Package {
	return c.Pkg
}

func (c *Constant) Exported() bool {
	if c.Pkg.Path() == "builtin" {
		return !token.IsExported(c.Name())
	}
	return c.Const.Exported()
}

func (c *Constant) TType() types.Type {
	return c.Const.Type()
}

func (c *Constant) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if c.Type == nil {
		c.Type = d.RegisterType(c.TType())
	}
	return c.Type
}

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

type Variable struct {
	*types.Var

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Var.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ValueSpec
}

func (v *Variable) Position() token.Position {
	for _, n := range v.AstSpec.Names {
		if n.Name == v.Name() {
			return v.Pkg.PPkg.Fset.PositionFor(n.Pos(), false)
		}
	}
	panic("should not")
}

func (v *Variable) Documentation() string {
	doc := v.AstSpec.Doc.Text()
	if doc == "" {
		doc = v.AstDecl.Doc.Text()
	}
	return doc
}

func (v *Variable) Comment() string {
	return v.AstSpec.Comment.Text()
}

func (v *Variable) Package() *Package {
	return v.Pkg
}

func (v *Variable) Exported() bool {
	if v.Pkg.Path() == "builtin" {
		return !token.IsExported(v.Name())
	}
	return v.Var.Exported()
}

func (v *Variable) TType() types.Type {
	return v.Var.Type()
}

func (v *Variable) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if v.Type == nil {
		v.Type = d.RegisterType(v.TType())
	}
	return v.Type
}

//func (v *Variable) IndexString() string {
//	var b strings.Builder
//
//	b.WriteString(v.Name())
//	b.WriteByte(' ')
//	b.WriteString(v.Type().String())
//
//	s := b.String()
//	println(s)
//	return s
//}

type Function struct {
	*types.Func
	*types.Builtin // for builtin functions

	// Builtin, Variadic.
	attributes Attribute

	// ToDo: maintain parameter and result TypeInfos, for performance.

	// ToDo
	fSigIndex uint32 // as package function
	mSigIndex uint32 // as method, (ToDo: make 0 as invalid function index)

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Func.Pkg(), except builtin functions
	AstDecl *ast.FuncDecl
}

func (f *Function) Name() string {
	if f.Func != nil {
		return f.Func.Name()
	}
	return f.Builtin.Name()
}

func (f *Function) Exported() bool {
	if f.Builtin != nil {
		return true
	}
	if f.Pkg.Path() == "builtin" {
		return !token.IsExported(f.Name())
	}
	return f.Func.Exported()
}

func (f *Function) Position() token.Position {
	return f.Pkg.PPkg.Fset.PositionFor(f.AstDecl.Name.Pos(), false)
}

func (f *Function) Documentation() string {
	// ToDo: html escape
	return f.AstDecl.Doc.Text()
}

func (f *Function) Comment() string {
	return ""
}

func (f *Function) Package() *Package {
	return f.Pkg
}

func (f *Function) TType() types.Type {
	if f.Func != nil {
		return f.Func.Type()
	}
	return f.Builtin.Type()
}

func (f *Function) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if f.Type == nil {
		f.Type = d.RegisterType(f.TType())
	}
	return f.Type
}

func (f *Function) IsMethod() bool {
	return f.Func != nil && f.Func.Type().(*types.Signature).Recv() != nil
}

func (f *Function) String() string {
	if f.Func != nil {
		return f.Func.String()
	}
	return f.Builtin.String()
}

//func (f *Function) IndexString() string {
//	var b strings.Builder
//	b.WriteString(f.Name())
//	b.WriteByte(' ')
//	WriteType(&b, f.AstDecl.Type, f.Pkg.PPkg.TypesInfo, true)
//	return b.String()
//}

// Please make sure the Funciton is a method when calling this method.
func (f *Function) ReceiverTypeName() (paramField *ast.Field, typeIdent *ast.Ident, isStar bool) {
	if f.AstDecl.Recv == nil {
		panic("should not")
	}
	if len(f.AstDecl.Recv.List) != 1 {
		panic("should not")
	}

	paramField = f.AstDecl.Recv.List[0]
	switch expr := paramField.Type.(type) {
	default:
		panic("should not")
	case *ast.Ident:
		typeIdent = expr
		isStar = false
		return
	case *ast.StarExpr:
		tid, ok := expr.X.(*ast.Ident)
		if !ok {
			panic("should not")
		}
		typeIdent = tid
		isStar = true
		return
	}
}

// ToDo: not use types.NewMethodSet or typesutil.MethodSet().
//       Implement it from scratch instead.
//type Method struct {
//	*types.Func // receiver is ignored
//
//	SignatureIndex uint32
//
//	PointerReceiverOnly bool
//
//	// The embedded type names in full form.
//	// Nil means this method is not obtained through embedding.
//	SelectorChain []Embedded
//
//	astFunc *ast.FuncDecl
//}

type MethodSignature struct {
	Name string // must be an identifier other than "_"
	Pkg  string // the import path, for unepxorted method names only

	//InOutTypes []int32 // global type indexes
	InOutTypes string

	NumInOutAndVariadic int
}

//// The lower bits of each Embedded is an index to the global TypeName table.
//// The global TypeName table comtains all type aliases and defined types.
//// The highest bit indicates whether or not the embedding for is *T or not.
//type Embedded uint32
//
//type Field struct {
//	*types.Var
//
//	// The info is contained in the above types.Var field.
//	//Owner *TypeInfo // must be a (non-defined) struct type
//
//	// The embedded type names in full form.
//	// Nil means this is a non-embedded field.
//	SelectorChain []Embedded
//
//	astList  *ast.FieldList
//	astField *ast.Field
//}
//
//type Method struct {
//	*types.Func // object denoted by x.f
//
//	SelectorChain []Embedded
//
//	astFunc *ast.FuncDecl
//}

type EmbedMode uint8

const (
	EmbedMode_None EmbedMode = iota
	EmbedMode_Direct
	EmbedMode_Indirect
)

type Field struct {
	astStruct    *ast.StructType
	AstField     *ast.Field
	astInterface *ast.InterfaceType // for embedding interface in interface

	Pkg  *Package // nil for exported
	Name string
	Type *TypeInfo

	Tag  string
	Mode EmbedMode
}

func (fld *Field) Position() token.Position {
	return fld.Pkg.PPkg.Fset.PositionFor(fld.AstField.Pos(), false)
}

type Method struct {
	AstFunc      *ast.FuncDecl      // for concrete methods
	astInterface *ast.InterfaceType // for interface methods
	AstField     *ast.Field         // for interface methods

	Pkg  *Package // nil for exported
	Name string
	Type *TypeInfo // ToDo: use custom struct including PointerRecv instead.

	PointerRecv bool // duplicated info, for faster access
}

func (mthd *Method) Position() token.Position {
	return mthd.Pkg.PPkg.Fset.PositionFor(mthd.AstFunc.Pos(), false)
}

type EmbeddedField struct {
	*Field
	Prev *EmbeddedField
}

type SelectorCond uint8

const (
	SelectorCond_Normal SelectorCond = iota
	SelectorCond_Hidden
)

type Selector struct {
	Id string

	// One and only one of the two is nil.
	*Field
	*Method

	// EmbeddedField is nil means this is not an promoted selector.
	//EmbeddedFields []*Field

	EmbeddingChain *EmbeddedField // in the inverse order
	Depth          uint16         // the chain length
	Indirect       bool           // whether the chain contains indirects or not

	// colliding or shadowed susposed promoted selector?
	//shadowed bool // used in collecting phase.
	cond SelectorCond
}

func (s *Selector) Reset() {
	*s = Selector{}
}

func (s *Selector) Position() token.Position {
	if s.Field != nil {
		return s.Field.Pkg.PPkg.Fset.PositionFor(s.Field.AstField.Pos(), false)
	} else if s.Method.AstFunc != nil { // method declaration
		return s.Method.Pkg.PPkg.Fset.PositionFor(s.Method.AstFunc.Pos(), false)
	} else { // if s.Method.AstField != nil //initerface method specification
		return s.Method.Pkg.PPkg.Fset.PositionFor(s.Method.AstField.Pos(), false)
	}
}

func (s *Selector) Name() string {
	if s.Field != nil {
		return s.Field.Name
	} else {
		return s.Method.Name
	}
}

func (s *Selector) Pkg() *Package {
	if s.Field != nil {
		return s.Field.Pkg
	} else {
		return s.Method.Pkg
	}
}

//func (s *Selector) Depth() int {
//	return len(s.EmbeddedFields)
//}

func (s *Selector) PointerReceiverOnly() bool {
	if s.Method == nil {
		panic("not a method selector")
	}

	return !s.Indirect && s.Method.PointerRecv
}

func (s *Selector) String() string {
	return EmbededFieldsPath(s.EmbeddingChain, nil, s.Name(), s.Field != nil)
}

func EmbededFieldsPath(embedding *EmbeddedField, b *strings.Builder, selName string, isField bool) (r string) {
	if embedding == nil {
		if isField {
			return "[field] " + selName
		} else {
			return "[method] " + selName
		}
	}
	if b == nil {
		b = &strings.Builder{}
		if isField {
			b.WriteString("[field] ")
		} else {
			b.WriteString("[method] ")
		}
		defer func() {
			b.WriteString(selName)
			r = b.String()
		}()
	}
	if p := embedding.Prev; p != nil {
		EmbededFieldsPath(p, b, "", isField)
	}
	if embedding.Field.Mode == EmbedMode_Indirect {
		b.WriteByte('*')
	}
	b.WriteString(embedding.Field.Name)
	b.WriteByte('.')
	return
}

func PrintSelectors(title string, selectors []*Selector) {
	log.Printf("%s (%d)\n", title, len(selectors))
	for _, sel := range selectors {
		log.Println("  ", sel)
	}
}

// ToDo: use go/doc package
