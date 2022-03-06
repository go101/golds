//go:build !go1.18
// +build !go1.18

package server

import (
	"go/ast"
	"go/types"

	"go101.org/golds/code"
)

type (
	astIndexExpr     = struct{ ast.IndexExpr }
	astIndexListExpr = struct {
		ast.IndexExpr
		Indices []ast.Expr
	}
	astUnaryExpr  = struct{ ast.UnaryExpr }
	astBinaryExpr = struct{ ast.BinaryExpr }

	typesTypeParam = struct{ types.Named }
)

func writeTypeParamsOfTypeName(page *htmlPage, res *code.TypeName) {
}

func writeTypeParamsOfFunciton(page *htmlPage, res *code.Function) {
}

func writeTypeParamsForMethodReceiver(page *htmlPage, methodFunc *ast.FuncDecl, forTypeName *code.TypeName) {
}

func (ds *docServer) writeTypeParameterListCallbackForTypeName(page *htmlPage, pkg *code.Package, tn *code.TypeName) func() {
	return nil
}

func (ds *docServer) writeTypeParameterListCallbackForFunction(page *htmlPage, pkg *code.Package, fv *code.Function) func() {
	return nil
}
