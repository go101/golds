//go:build go1.18
// +build go1.18

package server

import (
	"go/ast"
	"go/types"

	"go101.org/golds/code"
)

type (
	astIndexExpr     = ast.IndexExpr
	astIndexListExpr = ast.IndexListExpr
	astUnaryExpr     = ast.UnaryExpr
	astBinaryExpr    = ast.BinaryExpr

	typesTypeParam = types.TypeParam
)

func _writeTypeParams(page *htmlPage, fields []*ast.Field) {
	page.WriteByte('[')
	defer page.WriteByte(']')
	for i, fld := range fields {
		for j, n := range fld.Names {
			page.WriteString(n.Name)
			if i < len(fields)-1 || j < len(fld.Names)-1 {
				page.WriteString(", ")
			}
		}
	}
}

func writeTypeParamsOfTypeName(page *htmlPage, res *code.TypeName) {
	if tps := res.AstSpec.TypeParams; tps != nil {
		_writeTypeParams(page, tps.List)
	}
}

func writeTypeParamsOfFunciton(page *htmlPage, res *code.Function) {
	if tps := res.AstDecl.Type.TypeParams; tps != nil {
		_writeTypeParams(page, tps.List)
	}
}

func writeTypeParamsForMethodReceiver(page *htmlPage, method *code.Method, forTypeName *code.TypeName) {
	if method.AstFunc != nil {
		var writeTypeNames func()
		switch e := method.AstFunc.Recv.List[0].Type.(type) {
		case *ast.IndexExpr:
			writeTypeNames = func() {
				page.WriteString(e.Index.(*ast.Ident).Name)
			}
		case *ast.IndexListExpr:
			writeTypeNames = func() {
				for _, index := range e.Indices {
					page.WriteString(index.(*ast.Ident).Name)
				}
			}
		}
		if writeTypeNames != nil {
			page.Write(leftSquare)
			writeTypeNames()
			page.Write(rightSquare)
		}
	} else { // interface method
		if tps := forTypeName.AstSpec.TypeParams; tps != nil {
			_writeTypeParams(page, tps.List)
		}
	}
}

func (ds *docServer) _writeTypeParameterList(page *htmlPage, pkg *code.Package, typePatams *ast.FieldList) {
	page.WriteString("\n\n\t\t")
	page.WriteString(page.Translation().Text_TypeParameters())
	for _, fld := range typePatams.List {
		for _, n := range fld.Names {
			page.WriteString("\n\t\t\t")
			page.WriteString(n.Name)
			page.WriteString(page.Translation().Text_Colon(true))
			ds.WriteAstType(page, fld.Type, pkg, pkg, true, nil, nil)
		}
	}
}

func (ds *docServer) writeTypeParameterListCallbackForTypeName(page *htmlPage, pkg *code.Package, tn *code.TypeName) func() {
	if tps := tn.AstSpec.TypeParams; tps != nil {
		return func() {
			ds._writeTypeParameterList(page, pkg, tps)
		}
	}

	return nil
}

func (ds *docServer) writeTypeParameterListCallbackForFunction(page *htmlPage, pkg *code.Package, fv *code.Function) func() {
	if tps := fv.AstDecl.Type.TypeParams; tps != nil {
		return func() {
			ds._writeTypeParameterList(page, pkg, tps)
		}
	}

	return nil
}
