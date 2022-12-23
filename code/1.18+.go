//go:build go1.18
// +build go1.18

package code

import (
	"log"
	//"fmt"
	"go/ast"
	"go/types"
)

var _ = log.Print

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

/// ToDo: for a type may be denoted by multiple different ast expressions,
//        the implementaion is not perfect.

// typeInfo must be an instantiated type.
func (d *CodeAnalyzer) comfirmDirectSelectorsForInstantiatedType(typeInfo *TypeInfo, currentCounter uint32, fieldMap, methodMap map[string]*TypeInfo) {
	//log.Println(0, typeInfo.TT)

	if (typeInfo.attributes & directSelectorsCollected) != 0 {
		return
	}
	typeInfo.attributes |= directSelectorsCollected

	defer func(t *TypeInfo) {
		t.counter = currentCounter // ToDo: maybe useless
	}(typeInfo)

	//log.Println(1)

	var clearFieldMap = func() {
		for k := range fieldMap {
			delete(fieldMap, k)
		}
	}
	var clearMethodMap = func() {
		for k := range methodMap {
			delete(methodMap, k)
		}
	}

	// For the named type itself.
	ntt := typeInfo.TT.(*types.Named)
	clearMethodMap()
	for i := ntt.NumMethods() - 1; i >= 0; i-- {
		m := ntt.Method(i)
		methodMap[m.Name()] = d.RegisterType(m.Type())
	}

	origin := typeInfo.TypeName.Denoting
	typeInfo.DirectSelectors = make([]*Selector, len(origin.DirectSelectors))
	for i, sel := range origin.DirectSelectors {
		if sel.Method == nil {
			panic("should not")
		}

		insSel := *sel
		typeInfo.DirectSelectors[i] = &insSel

		insSel.Instantiated = typeInfo.Instantiated
		insSel.RealType = methodMap[sel.Method.Name]
		if insSel.RealType == nil {
			panic("should not")
		}
	}

	// For the underlying type of the named type.
	underlying := typeInfo.Underlying
	defer func() {
		//if underlying.counter < currentCounter {
		underlying.counter = currentCounter
		//}
	}()

	//log.Println(2, len(underlying.DirectSelectors), typeInfo.Underlying)

	defer func() {
		underlying.attributes |= directSelectorsCollected

		//for _, s := range underlying.DirectSelectors {
		//	log.Println(">>> (defer)", s.String(), s.Type())
		//}
	}()

	//if underlying.DirectSelectors != nil {
	//	// For many reasons, the DirectSelectors of an underlying type might have been collected,
	//	// in which case, Instantiated is either nil or doesn't depend on *types.TypeParam,
	//	// so that DirectSelectors has been collected for another instantiated type.
	//	// The two Instantiated types share the same underlying type.
	//    //
	//    // Possible reasons: the underlying types of some different instantiated types
	//    // might be identical. They even may be identical with non-generic types.
	//	//
	//	// ToDo: so here is an imperfection in rendering.
	//	return
	//
	//	// Another case: the underlying type of "I1[T]" is the same as the generic type T3.
	//	// ToDo: to avoid this.
	//	//
	//	// type I1[T any] interface { m1() T }
	//	//
	//	// type I3[T any] interface {
	//	//	interface {
	//	//		I1[T]
	//	//	}
	//	// }
	//}

	typeArgs := typeInfo.Instantiated.TypeArgs
	source := typeInfo.TypeName.Source
	lastPkg := typeInfo.TypeName.Pkg
	for source.Type.TypeName != nil {
		lastPkg, source, typeArgs = transformTypeArgs(source, typeArgs)
	}

	if source.Type == underlying { // true if the type doesn't use any TypeParam.
		return
	}

	//switch tt := source.Type.TT.(type) {
	switch tt := underlying.TT.(type) {
	default:
		return
	case *types.Named:
		panic("should not")
	case *types.Struct:
		stt := tt
		clearFieldMap()
		for i := stt.NumFields() - 1; i >= 0; i-- {
			v := stt.Field(i)
			fieldMap[v.Name()] = d.RegisterType(v.Type())
		}

		instantiated := &InstantiatedInfo{TypeArgs: typeArgs}
		underlying.DirectSelectors = make([]*Selector, len(source.Type.DirectSelectors))
		for i, sel := range source.Type.DirectSelectors {
			if sel.Field == nil {
				panic("should not")
			}

			insSel := *sel
			underlying.DirectSelectors[i] = &insSel

			insSel.Instantiated = instantiated
			insSel.RealType = fieldMap[sel.Field.Name]
			if insSel.RealType == nil {
				panic("should not")
			}

			if sel.Field.Mode == EmbedMode_None {
				// To save computation.
				continue
			}

			// EmbedMode_Indirect or EmbedMode_Direct

			realType := insSel.RealType

			//if sel.Field.Mode == EmbedMode_Indirect {
			// ...
			//}

			// It is not a good idea to use the EmbedMode_Indirect enum
			// to make decisisons here, for an embedding field might be
			// an alias to a pointer type.

			if ptt, ok := realType.TT.(*types.Pointer); ok {
				realType = d.RegisterType(ptt.Elem())
			}

			d.registerInstantiatedType(realType, instantiated.TypeArgs)
		}

	case *types.Interface:
		// ToDo: For the fact that, in "interface { interface { ... } }",
		// the TypeInfos of the outer and inner interfaces are the same one,

		var collectTypes func(itt *types.Interface)
		collectTypes = func(itt *types.Interface) {
			//log.Println("4. =====", itt.NumExplicitMethods(), itt.NumEmbeddeds())

			for i := itt.NumMethods() - 1; i >= 0; i-- {
				m := itt.Method(i)
				methodMap[m.Name()] = d.RegisterType(m.Type())

				//log.Println("41. =====", m)
			}

			for i := itt.NumEmbeddeds() - 1; i >= 0; i-- {
				et := itt.EmbeddedType(i)

				//log.Println("42. =====", et)

				switch tt2 := et.(type) {
				case *types.Named:
					//log.Println("421. =====", tt2.Obj().Name())
					fieldMap[tt2.Obj().Name()] = d.RegisterType(et)
				case *types.Interface:
					//log.Println("422. =====", tt2)
					t2 := d.RegisterType(tt2)
					if t2.counter < currentCounter {
						t2.counter = currentCounter
						collectTypes(tt2)
					}
				default: // Union is ignored now.
					// ToDo: maybe need to consider later.

					//log.Println("423. =====", tt2)
				}
			}
		}
		_ = lastPkg

		//var collectTypesFromUnnamedInterfaces func(expr *ast.InterfaceType)
		//collectTypesFromUnnamedInterfaces = func(expr *ast.InterfaceType) {
		//	expr, ok := source.Expr.(*ast.InterfaceType)
		//	if !ok {
		//		return
		//	}
		//
		//	for _, method := range expr.Methods.List {
		//		// method is a *ast.Field.
		//		if len(method.Names) == 0 {
		//			switch expr := method.Type.(type) {
		//			case *ast.InterfaceType:
		//				fieldTT := lastPkg.PPkg.TypesInfo.TypeOf(expr)
		//				if fieldTT == nil {
		//					panic("should not")
		//				}
		//
		//				t := d.RegisterType(fieldTT)
		//				if t.counter < currentCounter {
		//					t.counter = currentCounter
		//					collectTypes(fieldTT.(*types.Interface))
		//					collectTypesFromUnnamedInterfaces(expr)
		//				}
		//			}
		//		}
		//	}
		//}

		// It is allowed to make method and field names duplicated.
		// That is why two maps are used.
		clearMethodMap()
		clearFieldMap()

		collectTypes(tt)
		//collectTypesFromUnnamedInterfaces(source.Expr.(*ast.InterfaceType))

		instantiated := &InstantiatedInfo{TypeArgs: typeArgs}
		if underlying.DirectSelectors == nil {
			underlying.DirectSelectors = make([]*Selector, 0, len(source.Type.DirectSelectors))
		}

		for _, sel := range source.Type.DirectSelectors {
			var realType *TypeInfo
			if sel.Method != nil {
				// for embedding reason, realType might be nil.
				realType = methodMap[sel.Method.Name]

				//log.Printf("4 a: %#v", sel.Method.Type.TT)

				if realType == nil {
					log.Println(sel.Method.Name)
					panic("should not")
				}
			} else if sel.Field == nil {
				panic("should not")
			} else {
				// interface field might be unnamed, so realType might be nil.
				realType = fieldMap[sel.Field.Name]

				//log.Printf("4 b: %#v", sel.Field.Type.TT)

				if realType == nil {
					log.Println(sel.Field.Name)//, fieldMap)
					panic("should not")
				}
			}

			insSel := *sel
			underlying.DirectSelectors = append(underlying.DirectSelectors, &insSel)

			insSel.Instantiated = instantiated
			insSel.RealType = realType

			if sel.Field != nil {
				d.registerInstantiatedType(realType, instantiated.TypeArgs)
			}
		}
	}
}

func transformTypeArgs(source TypeExpr, typeArgs []TypeExpr) (*Package, TypeExpr, []TypeExpr) {
	n := len(source.Type.TypeName.TypeParams)
	if len(source.Type.Instantiated.TypeArgs) != n {
		panic("should not")
	}
	nextTypeArgs := make([]TypeExpr, n)
	for i := range source.Type.Instantiated.TypeArgs {
		argType := source.Type.Instantiated.TypeArgs[i].Type
		if tp, ok := argType.TT.(*typesTypeParam); ok {
			nextTypeArgs[i] = typeArgs[tp.Index()]
		} else {
			nextTypeArgs[i] = source.Type.Instantiated.TypeArgs[i]
		}
	}

	return source.Type.TypeName.Pkg, source.Type.TypeName.Source, nextTypeArgs
}
