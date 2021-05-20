package code

import (
	"go/ast"
	"go/doc"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// A Module holds the information for a Go module.
type Module struct {
	Index int // users might make some optimizations by using the index

	Dir string // might be blank for vendored modules

	Path    string
	Version string

	// ...
	Replace moduleReplacement

	// If replacement exists, the following info is for the replacement.

	RepositoryCommit string // might be the same as Version, or not.
	RepositoryURL    string
	RepositoryDir    string // no much useful

	// Generally blank. But
	// 1. "/src" for std and "/src/cmd" for toolchain.
	// 2. Some modules are not at the root of their repositories.
	//    (Often, such a repository contains multiple modules.
	//    ex., for github.com/aws/aws-sdk-go-v2/service/{ec2,s3},
	//    they are service/s3, service/ec2)
	ExtraPathInRepository string

	Pkgs []*Package // seen packages
}

// Note, for a module m with replacement r,
// * m.Version and r.Version could be both blank and non-blank or either blank.
type moduleReplacement struct {
	Dir     string
	Path    string
	Version string
}

func (m *Module) ActualPath() string {
	if m.Replace.Path != "" && m.Replace.Path[0] != '.' {
		return m.Replace.Path
	}
	return m.Path
}

func (m *Module) ActualVersion() string {
	if m.Replace.Version != "" {
		return m.Replace.Version
	}
	return m.Version
}

func (m *Module) ActualDir() string {
	if m.Replace.Dir != "" {
		return m.Replace.Dir
	}
	return m.Dir
}

// Package holds the information and the analysis result of a Go package.
type Package struct {
	Index int               // ToDo: use this to do some optimizations
	PPkg  *packages.Package // ToDo: renamed to PP to be consistent with TypeInfo.TT?

	Deps      []*Package
	DepedBys  []*Package
	DepHeight int32 // 0 means the height is not determined yet. The order determines the parse order.
	DepDepth  int32 // 0 means the depth is not determined yet. The value mains how close to main pacakges. (Moved to user space).

	// This field might be shared with PackageForDisplay
	// for concurrent reads.
	*PackageAnalyzeResult // ToDo: not as pointer?
	SourceFiles           []SourceFileInfo
	ExampleFiles          []*ast.File
	Examples              []*doc.Example

	Directory  string
	Module     *Module
	OneLineDoc string
}

// Path returns the import path of a Package.
func (p *Package) Path() string {
	return p.PPkg.PkgPath // might be prefixed with "vendor/", which is different from import path.
}

// PackageAnalyzeResult holds the analysis result of a Go package.
type PackageAnalyzeResult struct {
	AllTypeNames []*TypeName
	AllFunctions []*Function
	AllVariables []*Variable
	AllConstants []*Constant
	AllImports   []*Import

	CodeLinesWithBlankLines int32
}

// NewPackageAnalyzeResult returns a new initialized PackageAnalyzeResult.
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

// SourceFileInfoByBareFilename returns the SourceFileInfo corresponding the specified bare filename.
func (pkg *Package) SourceFileInfoByBareFilename(bareFilename string) *SourceFileInfo {
	for _, info := range pkg.SourceFiles {
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

// SourceFileInfoByFilePath return the SourceFileInfo corresponding the specified file path.
func (pkg *Package) SourceFileInfoByFilePath(srcPath string) *SourceFileInfo {
	for _, info := range pkg.SourceFiles {
		if info.OriginalFile == srcPath {
			return &info
		}
		if info.GeneratedFile == srcPath {
			return &info
		}
	}
	return nil
}

//type RefPos struct {
//	Pkg *Package
//	Pos token.Pos
//}

//type AstNode struct {
//	Pkg  *Package
//	Node ast.Node
//}

// Resource is an interface for Variable/Constant/TypeName/Function/InterfaceMethod.
type Resource interface {
	Name() string
	Exported() bool
	//IndexString() string
	Documentation() string
	Comment() string
	Position() token.Position
	Package() *Package
}

// ValueResource is an interface for Variable/Constant/Function/InterfaceMethod..
type ValueResource interface {
	Resource
	TType() types.Type // The result should not be used in comparisons.
	TypeInfo(d *CodeAnalyzer) *TypeInfo
}

// FunctionResource is an interface for Function/InterfaceMethod.
type FunctionResource interface {
	ValueResource
	IsMethod() bool
	//ReceiverTypeName() (paramField *ast.Field, typeIdent *ast.Ident, isStar bool)
	ReceiverTypeName() (paramField *ast.Field, typename *TypeName, isStar bool)
	AstFuncType() *ast.FuncType

	// For *Function, the result is the same as ValueResource.Package().
	// For *InterfaceMethod, this might be different (caused by embedding, or other reasons).
	AstPackage() *Package
}

var (
	_ FunctionResource = (*Function)(nil)
	_ FunctionResource = (*InterfaceMethod)(nil)
)

// AstValueSpecOwneris an interface for Variable/Constant.
type AstValueSpecOwner interface {
	AstValueSpec() *ast.ValueSpec
	Package() *Package
}

var (
	_ AstValueSpecOwner = (*Variable)(nil)
	_ AstValueSpecOwner = (*Constant)(nil)
)

// A Attribute records some imformations by using bits.
type Attribute uint32

const (
	// Runtime only flags.
	analyseCompleted Attribute = 1 << (31 - iota)
	directSelectorsCollected
	promotedSelectorsCollected

	// Higher bits are for runtime-only flags.
	AtributesPersistentMask Attribute = (1 << 25) - 1

	// Caching individual packages separately might be not a good idea.
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

	// For functions.
	Variadic Attribute = 1 << 8

	// For methods.
	StarReceiver Attribute = 1 << 9
)

// A TypeSource represents the source type in a type specification.
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

// EmbedInfo records the information for an embedded field.
type EmbedInfo struct {
	TypeName *TypeName
	IsStar   bool
}

// A TypeName represents a type name.
type TypeName struct {
	Examples []*Example

	Pkg     *Package // some duplicated with types.TypeName.Pkg(), except builtin types
	AstDecl *ast.GenDecl
	AstSpec *ast.TypeSpec

	*types.TypeName

	// One and only one of the two is nil.
	Alias *TypeAlias
	Named *TypeInfo

	//index uint32 // the global index

	// ToDo: simplify the source definition.
	// Four kinds of sources to affect promoted selectors:
	// 1. typename
	// 2. *typename
	// 3. unnamed type
	// 4. *unname type
	Source     TypeSource
	StarSource *TypeSource

	//UsePositions []token.Position

	// ToDo: maybe it is better to add some filters to id-use pages,
	//       * only show those in type specifications.
	//EmbeddedIn []EmbedInfo

	index uint32 // ToDo: any useful?
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

// Exported returns whether or not a TypeName is exported.
func (tn *TypeName) Exported() bool {
	if tn.Pkg.Path() == "builtin" {
		return !token.IsExported(tn.Name())
	}
	return tn.TypeName.Exported()
}

// Position returns the declaration position of a TypeName.
func (tn *TypeName) Position() token.Position {
	return tn.Pkg.PPkg.Fset.PositionFor(tn.AstSpec.Name.Pos(), false)
}

// Documentation returns the documents of a TypeName.
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

// Comment returns the comment of a TypeName.
func (tn *TypeName) Comment() string {
	return tn.AstSpec.Comment.Text()
}

// Package returns the owner Package of a TypeName.
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

// Denoting returns the denoting TypeInfo of a TypeName.
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

// TypeAlias represents a type alias,
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

// TypeInfo represent a type and records its analysis result.
type TypeInfo struct {
	TT types.Type

	Underlying *TypeInfo

	// For named and basic types.
	TypeName *TypeName

	//Implements     []*TypeInfo
	///StarImplements []*TypeInfo // if TT is neither pointer nor interface.
	Implements []Implementation

	// For interface types.
	ImplementedBys []*TypeInfo

	// For builtin and unnamed types only.
	Aliases []*TypeName

	// ToDo: For unnamed and builtin basic types.
	Underlieds []*TypeName

	// For unnamed types.
	//UsePositions []token.Position

	// For unnamed interfaces and structs, this field must be nil.
	//Pkg *Package // Looks this field is never used. (It really should not exist in this type.)

	// Explicit fields and methods.
	// * For named types, only explicitly declared methods are included.
	//   The field is only built for T. (*T).DirectSelectors is always nil.
	// * For named interface types, all explicitly specified methods and embedded types (as fields).
	// * For unnamed struct types, only direct fields. Only built for strct{...}, not for *struct{...}.
	DirectSelectors []*Selector
	EmbeddingFields int32 // for struct types only now. ToDo: also for interface types.

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

// Kind returns the kinds (as reflect.Kind) of a type.
func (t *TypeInfo) Kind() reflect.Kind {
	return Kind(t.TT)
}

// Kind rerurns the kinds (as reflect.Kind) for a go/types.Type.
func Kind(tt types.Type) reflect.Kind {
	switch tt := tt.Underlying().(type) {
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

// Implementation represents an implementation relation.
type Implementation struct {
	Impler    *TypeInfo // a struct or named type (same as the owner), or a pointer to such a type
	Interface *TypeInfo // an interface type
}

// Import represents an import.
type Import struct {
	*types.PkgName

	Pkg     *Package // some duplicated with types.PkgName.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ImportSpec
}

// Constant represents a constant.
type Constant struct {
	Examples []*Example

	*types.Const

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Const.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ValueSpec
}

// Position returns the declaration position of a Constant.
func (c *Constant) Position() token.Position {
	for _, n := range c.AstSpec.Names {
		if n.Name == c.Name() {
			return c.Pkg.PPkg.Fset.PositionFor(n.Pos(), false)
		}
	}
	panic("should not")
}

// Documentation returns the document of a Constant.
func (c *Constant) Documentation() string {
	doc := c.AstSpec.Doc.Text()
	if doc == "" {
		doc = c.AstDecl.Doc.Text()
	}
	return doc
}

// Comment returns the comment of a Constant.
func (c *Constant) Comment() string {
	return c.AstSpec.Comment.Text()
}

// Package returns the owner Package of a Constant.
func (c *Constant) Package() *Package {
	return c.Pkg
}

// Exported returns whether or not a Constant is exported.
func (c *Constant) Exported() bool {
	if c.Pkg.Path() == "builtin" {
		return !token.IsExported(c.Name())
	}
	return c.Const.Exported()
}

// TType returns the go/types.Type of a Constant.
func (c *Constant) TType() types.Type {
	return c.Const.Type()
}

// TypeInfo returns the type of a Constant.
func (c *Constant) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if c.Type == nil {
		c.Type = d.RegisterType(c.TType())
	}
	return c.Type
}

// AstValueSpec returns the go/ast.ValueSpec for a Constant.
func (c *Constant) AstValueSpec() *ast.ValueSpec {
	return c.AstSpec
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

// Variable represents a variable.
type Variable struct {
	Examples []*Example

	*types.Var

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Var.Pkg()
	AstDecl *ast.GenDecl
	AstSpec *ast.ValueSpec
}

// Position returns the position in code for a Variable.
func (v *Variable) Position() token.Position {
	for _, n := range v.AstSpec.Names {
		if n.Name == v.Name() {
			return v.Pkg.PPkg.Fset.PositionFor(n.Pos(), false)
		}
	}
	panic("should not")
}

// Documentation returns the document of a Variable.
func (v *Variable) Documentation() string {
	doc := v.AstSpec.Doc.Text()
	if doc == "" {
		doc = v.AstDecl.Doc.Text()
	}
	return doc
}

// Comment returns the comment of a Variable.
func (v *Variable) Comment() string {
	return v.AstSpec.Comment.Text()
}

// Package returns the owner package of a Variable.
func (v *Variable) Package() *Package {
	return v.Pkg
}

// Exported returns whether or not a Variable is exported.
func (v *Variable) Exported() bool {
	if v.Pkg.Path() == "builtin" {
		return !token.IsExported(v.Name())
	}
	return v.Var.Exported()
}

// TType returns the go/types.Type for a Variable.
func (v *Variable) TType() types.Type {
	return v.Var.Type()
}

// TypeInfo returns the type of a Variable.
func (v *Variable) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if v.Type == nil {
		v.Type = d.RegisterType(v.TType())
	}
	return v.Type
}

// AstValueSpec returns the go/ast.ValueSpec for a Variable.
func (v *Variable) AstValueSpec() *ast.ValueSpec {
	return v.AstSpec
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

// Function represents a function, including non-interface methods.
type Function struct {
	Examples []*Example

	*types.Func
	*types.Builtin // for builtin functions

	// isStarReceiver, ... ToDo: Builtin, Variadic.
	attributes Attribute

	// Non-nil for method functions.
	receiverTypeName *TypeName

	// ToDo: maintain parameter and result TypeInfos, for performance.

	// ToDo
	fSigIndex uint32 // as package function
	mSigIndex uint32 // as method, (ToDo: make 0 as invalid function index)

	Type    *TypeInfo
	Pkg     *Package // some duplicated with types.Func.Pkg(), except builtin functions
	AstDecl *ast.FuncDecl
}

// Names returns the name of a Function.
func (f *Function) Name() string {
	if f.Func != nil {
		return f.Func.Name()
	}
	return f.Builtin.Name()
}

// Exported returns whether or or a Function is exported.
func (f *Function) Exported() bool {
	if f.Builtin != nil {
		return true
	}
	if f.Pkg.Path() == "builtin" {
		return !token.IsExported(f.Name())
	}
	return f.Func.Exported()
}

// Position return the position of a Function.
func (f *Function) Position() token.Position {
	return f.Pkg.PPkg.Fset.PositionFor(f.AstDecl.Name.Pos(), false)
}

// Documentation return document of a Function.
func (f *Function) Documentation() string {
	// ToDo: html escape
	return f.AstDecl.Doc.Text()
}

// Comment always return "".
func (f *Function) Comment() string {
	return ""
}

// Package returns the owner of a Function.
func (f *Function) Package() *Package {
	return f.Pkg
}

// TType returns the go/types.Type for a Function.
func (f *Function) TType() types.Type {
	if f.Func != nil {
		return f.Func.Type()
	}
	return f.Builtin.Type()
}

// TypeInfo returns the tyoe of a Function.
func (f *Function) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	if f.Type == nil {
		f.Type = d.RegisterType(f.TType())
	}
	return f.Type
}

// IsMethod returns whether or not a Function is a method.
func (f *Function) IsMethod() bool {
	return f.Func != nil && f.Func.Type().(*types.Signature).Recv() != nil
}

// String returns the string representation of a Function.
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

// ReceiverTypeName returns the TypeName and whether or not the receiver is a pointer for a method function.
func (f *Function) ReceiverTypeName() (paramField *ast.Field, typename *TypeName, isStar bool) {
	if f.AstDecl.Recv == nil {
		panic("should not")
	}
	if len(f.AstDecl.Recv.List) != 1 {
		panic("should not")
	}

	paramField = f.AstDecl.Recv.List[0]

	//switch expr := paramField.Type.(type) {
	//default:
	//	panic("should not")
	//case *ast.Ident:
	//	typeIdent = expr
	//	isStar = false
	//case *ast.StarExpr:
	//	tid, ok := expr.X.(*ast.Ident)
	//	if !ok {
	//		panic("should not")
	//	}
	//	typeIdent = tid
	//	isStar = true
	//}

	typename = f.receiverTypeName
	isStar = f.attributes&StarReceiver != 0
	return
}

// AstFuncType returns the go/ast.FuncType for a Function.
func (f *Function) AstFuncType() *ast.FuncType {
	return f.AstDecl.Type
}

// AstPackage returns the same as Package().
func (f *Function) AstPackage() *Package {
	return f.Package()
}

// InterfaceMethod represents an interface function.
type InterfaceMethod struct {
	Examples []*Example

	InterfaceTypeName *TypeName
	Method            *Method // .AstFunc == nil, .AstInterface != nil

	// ToDo: an interface method might have several ast sources,
	//       so there should be multiple Methods ([]*Method).
}

// Name returns the name of a InterfaceMethod.
func (im *InterfaceMethod) Name() string {
	return im.Method.Name
}

// Name returns whether or not a InterfaceMethod is exported.
func (im *InterfaceMethod) Exported() bool {
	return token.IsExported(im.Name())
}

// Name returns the code position of a InterfaceMethod.
func (im *InterfaceMethod) Position() token.Position {
	return im.Method.Pkg.PPkg.Fset.PositionFor(im.Method.AstField.Pos(), false)
}

// Name returns the document of a InterfaceMethod.
func (im *InterfaceMethod) Documentation() string {
	return im.Method.AstField.Doc.Text()
}

// Name returns the comment of a InterfaceMethod.
func (im *InterfaceMethod) Comment() string {
	return im.Method.AstField.Comment.Text()
}

// Name returns the owner Package of a InterfaceMethod.
func (im *InterfaceMethod) Package() *Package {
	return im.InterfaceTypeName.Pkg
}

// Name returns the go/types.Type for a InterfaceMethod.
func (im *InterfaceMethod) TType() types.Type {
	return im.Method.Type.TT
}

// Name returns the type of a InterfaceMethod.
func (im *InterfaceMethod) TypeInfo(d *CodeAnalyzer) *TypeInfo {
	return im.Method.Type
}

// Name always returns true.
func (im *InterfaceMethod) IsMethod() bool {
	return true
}

// Name returns the string representation of a InterfaceMethod.
func (im *InterfaceMethod) String() string {
	// ToDo: show the inteface receiver in result.
	return im.Method.Type.TT.String()
}

//func (im *InterfaceMethod) IndexString() string {
//	var b strings.Builder
//	b.WriteString(f.Name())
//	b.WriteByte(' ')
//	WriteType(&b, f.AstDecl.Type, f.Pkg.PPkg.TypesInfo, true)
//	return b.String()
//}

// ReceiverTypeName returns the TypeName and whether or not the receiver is a pointer for a method function.
func (im *InterfaceMethod) ReceiverTypeName() (paramField *ast.Field, typename *TypeName, isStar bool) {
	//return nil, im.InterfaceTypeName.AstSpec.Name, false
	return nil, im.InterfaceTypeName, false
}

// AstFuncType returns the go/ast.FuncType for a InterfaceMethod.
func (im *InterfaceMethod) AstFuncType() *ast.FuncType {
	return im.Method.AstField.Type.(*ast.FuncType)
}

// AstPackage returns the Package where a InterfaceMethodis is specified.
// For embedding reason. The result might be different from the owner package.
func (im *InterfaceMethod) AstPackage() *Package {
	return im.Method.Pkg
}

// MethodSignature represents a hashable struct for a method.
type MethodSignature struct {
	Name string // must be an identifier other than "_"
	Pkg  string // the import path, for unepxorted method names only

	// ToDo: the above two can be replaced with two int32 IDs.

	//InOutTypes []int32 // global type indexes
	InOutTypes string

	NumInOutAndVariadic int
}

type EmbedMode uint8

const (
	EmbedMode_None     EmbedMode = iota
	EmbedMode_Direct             // TypeName (note: it might be a pointer alias)
	EmbedMode_Indirect           // *TypeName
)

// Field represents a struct field.
type Field struct {
	Examples []*Example

	astStruct *ast.StructType
	AstField  *ast.Field
	//AstInterface *ast.InterfaceType // for embedding interface in interface (the owner interface)

	Pkg  *Package // (nil for exported. ??? Seems not true.)
	Name string
	Type *TypeInfo

	Tag  string
	Mode EmbedMode
}

// Position returns the code position for a Field.
func (fld *Field) Position() token.Position {
	return fld.Pkg.PPkg.Fset.PositionFor(fld.AstField.Pos(), false)
}

// Documentation returns the documents of a Field.
func (fld *Field) Documentation() string {
	if doc := fld.AstField.Doc; doc != nil {
		return doc.Text()
	}
	return ""
}

// Comment returns the comment of a Field.
func (fld *Field) Comment() string {
	if comment := fld.AstField.Comment; comment != nil {
		return comment.Text()
	}
	return ""
}

// Method represent a method.
type Method struct {
	AstFunc *ast.FuncDecl // for concrete methods
	//AstInterface *ast.InterfaceType // for interface methods (the owner interface)
	AstField *ast.Field // for interface methods

	Pkg  *Package // (nil for exported. ??? Seems not true.)
	Name string
	Type *TypeInfo // ToDo: use custom struct including PointerRecv instead.

	PointerRecv         bool // duplicated info, for faster access
	ImplementsSomething bool // false if the method is unimportant for its reveiver to implement some interface type

	index uint32 // 0 means this method doesn;t contribute to any type implementations for sure.
}

// Position returns the code position of a Method.
func (mthd *Method) Position() token.Position {
	if mthd.AstFunc != nil { // method declaration
		return mthd.Pkg.PPkg.Fset.PositionFor(mthd.AstFunc.Pos(), false)
	} else { // if mthd.AstField != nil //initerface method specification
		return mthd.Pkg.PPkg.Fset.PositionFor(mthd.AstField.Pos(), false)
	}
}

// Documentation returns the document of a Method.
func (mthd *Method) Documentation() string {
	if mthd.AstFunc != nil { // method declaration
		if doc := mthd.AstFunc.Doc; doc != nil {
			return doc.Text()
		}
	} else { // if mthd.AstField != nil //initerface method specification
		if doc := mthd.AstField.Doc; doc != nil {
			return doc.Text()
		}
	}

	return ""
}

// Comment returns the comment of a Method.
func (mthd *Method) Comment() string {
	if mthd.AstField != nil { // if mthd.AstField != nil //initerface method specification
		if comment := mthd.AstField.Comment; comment != nil {
			return comment.Text()
		}
	}

	return ""
}

// EmbeddedField represengts am embedded field.
type EmbeddedField struct {
	*Field
	Prev *EmbeddedField
}

type SelectorCond uint8

const (
	SelectorCond_Normal SelectorCond = iota
	SelectorCond_Hidden
)

// Selector represents a selector, either a field or a method.
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

// Reset clears the data for a Selector.
func (s *Selector) Reset() {
	*s = Selector{}
}

// Object returns the go/types.Object represented by a Selector.
func (s *Selector) Object() types.Object {
	if s.Field != nil {
		for _, ident := range s.Field.AstField.Names {
			if ident.Name == s.Field.Name {
				return s.Field.Pkg.PPkg.TypesInfo.ObjectOf(ident)
			}
		}
		return nil // ToDo: handle the embedded field case
	}

	// Non-interface method
	if s.Method.AstFunc != nil {
		return s.Method.Pkg.PPkg.TypesInfo.ObjectOf(s.Method.AstFunc.Name)
	}

	// Interface method
	return s.Method.Pkg.PPkg.TypesInfo.ObjectOf(s.Method.AstField.Names[0])
}

// Position returns the code position of a Selector.
func (s *Selector) Position() token.Position {
	if s.Field != nil {
		return s.Field.Position()
	} else {
		return s.Method.Position()
	}
}

// Name returns the name of a Selector.
func (s *Selector) Name() string {
	if s.Field != nil {
		return s.Field.Name
	} else {
		return s.Method.Name
	}
}

// Package returns the owner package of a Selector.
func (s *Selector) Package() *Package {
	if s.Field != nil {
		return s.Field.Pkg
	} else {
		return s.Method.Pkg
	}
}

//func (s *Selector) Depth() int {
//	return len(s.EmbeddedFields)
//}

// PointerReceiverOnly returns whether or not a method selector is declared for a pointer type.
func (s *Selector) PointerReceiverOnly() bool {
	if s.Method == nil {
		panic("not a method selector")
	}

	return !s.Indirect && s.Method.PointerRecv
}

// String returns the string representation of a Selecctor.
func (s *Selector) String() string {
	return EmbededFieldsPath(s.EmbeddingChain, nil, s.Name(), s.Field != nil)
}

//func (s *Selector) Comment() string {
//	return "" // ToDo:
//}
//
//func (s *Selector) Documentation() string {
//	return "" // ToDo:
//}
//
//func (s *Selector) Exported() bool {
//	if s.Field != nil {
//		return token.IsExported(s.Field.Name)
//	} else {
//		return token.IsExported(s.Method.Name)
//	}
//}

// EmbededFieldsPath returns the string representation the middle embedding chain of a Selector.
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

// PrintSelectors prints a lists of Selectors.
func PrintSelectors(title string, selectors []*Selector) {
	log.Printf("%s (%d)\n", title, len(selectors))
	for _, sel := range selectors {
		log.Println("  ", sel)
	}
}

// Identifier represents an identifier occurrence in code.
type Identifier struct {
	//Pkg *Package // gettable from FileInfo

	FileInfo *SourceFileInfo
	AstIdent *ast.Ident
}

//type PackageLevelIdentifier struct {
//	FileInfo *SourceFileInfo
//	Examples []*Example
//}

type Example struct {
	example *doc.Example
	fset    *token.FileSet
}
