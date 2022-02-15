//go:build go1.18
// +build go1.18

package code

import (
	"go/ast"
	"go/types"
)

type (
	astIndexExpr     = ast.IndexExpr
	astIndexListExpr = ast.IndexListExpr
	//astUnaryExpr = ast.UnaryExpr
	//astBinaryExpr = ast.BinaryExpr

	typesTypeParam = types.TypeParam
)

func originType(nt *types.Named) *types.Named {
	return nt.Origin()
}

func isParameterizedType(tt types.Type) bool {
	nt, ok := tt.(*types.Named)
	return ok && nt.TypeParams() != nil
}

func isTypeParam(tt types.Type) bool {
	_, ok := tt.(*types.TypeParam)
	return ok
}
