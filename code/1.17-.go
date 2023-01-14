//go:build !go1.18
// +build !go1.18

package code

import (
	"go/ast"
	"go/types"
)

type (
	astIndexExpr     = struct{ ast.IndexExpr }
	astIndexListExpr = struct {
		ast.IndexExpr
		Indices []ast.Expr
	}
	//astUnaryExpr = struct { ast.UnaryExpr }
	//astBinaryExpr = struct { ast.BinaryExpr }

	typesTypeParam = struct {
		types.Named
		Index func() int
	}
)

func originType(nt *types.Named) *types.Named {
	return nt
}

func astTypeSpecTypeParams(ts *ast.TypeSpec) *ast.FieldList {
	return nil
}

func astFuncTypeTypeParams(ft *ast.FuncType) *ast.FieldList {
	return nil
}

func isParameterizedType(tt types.Type) bool {
	return false
}

func isTypeParam(tt types.Type) bool {
	return false
}

func (d *CodeAnalyzer) comfirmDirectSelectorsForInstantiatedType(typeInfo *TypeInfo, currentCounter uint32, fieldMap, methodMap map[string]*TypeInfo) {
}
