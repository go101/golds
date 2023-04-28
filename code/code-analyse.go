package code

import (
	"container/list"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/types/typeutil"

	"go101.org/golds/internal/util"
)

// AnalyzePackages analyzes the input packages.
func (d *CodeAnalyzer) AnalyzePackages(onSubTaskDone func(int, time.Duration, ...int32)) {
	//log.Println("[analyze packages ...]")

	var stopWatch = util.NewStopWatch()
	if onSubTaskDone == nil {
		onSubTaskDone = func(int, time.Duration, ...int32) {}
	}
	var logProgress = func(task int, args ...int32) {
		onSubTaskDone(task, stopWatch.Duration(true), args...)
	}

	stopWatch.Duration(true)

	d.sortPackagesByDepHeight()
	d.calculatePackagesDepDepths()
	logProgress(SubTask_SortPackagesByDependencies)

	for _, pkg := range d.packageList {
		d.analyzePackage_collectSourceFiles(pkg)
	}
	logProgress(SubTask_CollectSourceFiles)

	for _, pkg := range d.packageList {
		d.analyzePackage_CollectDeclarations(pkg)
	}
	logProgress(SubTask_CollectDeclarations)

	//log.Println("=== recorded type count:", len(d.allTypeInfos))

	//for _, pkg := range d.packageList {
	//	d.analyzePackage_ConfirmTypeSources(pkg) // need collect source files firstly
	//}
	//logProgress(SubTask_ConfirmTypeSources)

	for _, pkg := range d.packageList {
		d.analyzePackage_CollectDirectSelectors(pkg)
	}
	d.comfirmDirectSelectorsForInstantiatedTypes()
	d.collectSelectors()
	logProgress(SubTask_CollectSelectors)

	// ToDo: it might be best to not use the NewMethodSet function in std.
	//       Same for NewFieldSet

	d.forbidRegisterTypes = true
	//methodCache := d.findImplementations_Old()
	d.findImplementations()
	methodCache := &typeutil.MethodSetCache{}
	d.forbidRegisterTypes = false
	logProgress(SubTask_FindImplementations)

	d.registerNamedInterfaceMethodsForInvolvedTypeNames()
	logProgress(SubTask_RegisterInterfaceMethodsForTypes)

	d.collectObjectReferences()
	logProgress(SubTask_CollectObjectReferences)

	d.collectCodeExamples() // need the pkg.Directory confirmed in the last step
	logProgress(SubTask_CollectExamples)

	d.cacheSourceFiles()
	d.buildSourceFileTable()
	logProgress(SubTask_CacheSourceFiles)

	d.collectSomeRuntimeFunctionPositions()
	logProgress(SubTask_CollectRuntimeFunctionPositions)

	for _, pkg := range d.packageList {
		d.analyzePackage_CollectMoreStatistics(pkg)
	}
	d.collectMoreStatisticsFinal()
	logProgress(SubTask_MakeStatistics)

	// ...

	// The following is moved to TestAnalyzer.
	//
	//d.analyzePackages_CheckCollectSelectors(methodCache)
	_ = methodCache
	//logProgress("Check collect selectors", nil)

	// log.Println("[analyze packages done]")

	//log.Println(numNamedInterfaces, numNameds)
}

func (d *CodeAnalyzer) findImplementations() { // (resultMethodCache *typeutil.MethodSetCache) {
	// step 1: register all method signatures of underlying interface types.
	//         create a type list for each signature.
	// step 2: iteration all types, calculate their method signatures,
	//         (interface types can use their underlying cache calculated in step 1)
	//         ignore signatures which are note recorded in step 1.
	//         register the type into the type lists of method signatures.
	// step 3: iterate all underlying interfaces, iterate all method signatures,
	//         iterate the type list of a signature, TypeInfo.counter++
	//         ...

	type UnderlyingInterfaceInfo struct {
		t             *TypeInfo
		underlieds    []*TypeInfo // including the underlying itself
		methodIndexes []uint32

		//>> 1.18, ToDo
		// Ignore methods with type parameters now.

		typeIndexes []uint32 // specific types (sorted). The highest bit means tilde

		// t == t.Underlying

		// step 4: after reducing type set by checking method set, sort the types in the set:
		//         * for non-interface types, sorted by type indexes
		//         * for interface types, sorted by the first index in typeIndexes.
		//         Calculate the intersection of the result sorted indexes and UnderlyingInterfaceInfo.typeIndexes.
		//         Check tildes and "t == t.Underlying" in the intersection list, ...
		//<<
	}

	// ToDo: use map[InterfaceTypeIndex]*TypeInfo?
	var interfaceUnderlyings typeutil.Map

	//var interfaceUnderlyingTypes = make([]*TypeInfo, 0, 1024)
	//for _, t := range d.allTypeInfos {
	// New types might be registered in this loop,
	// so traditional for-loop is used here.
	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]

		// ToDo: auto register underlying type in RegisterType.
		//underlying := t.TT.Underlying()
		underlying := t.Underlying.TT
		// Reason if using the above line instead the line previous to the above line:
		// t.TT might be a TypeParam, in which case, the next line will produce a
		// new TypeInfo if the line previous to the above line is used.
		// Producing new TypeInfo at this time is prohibited.
		//
		// Surely, the analysis result is not correct for TypeParam, but it is not important.

		// ToDo: The next 3 lines should be able to be replaced with one line:
		//       underlyingTypeInfo = t.Underlying.
		// Here, they are for assert purpsoe.

		underlyingTypeInfo := d.RegisterType(underlying) // underlying must have been already registered

		//if m, n := underlyingTypeInfo.index, t.Underlying.index; m == 17838 && m != n {
		//	u := t.TT.Underlying()
		//	x := d.RegisterType(underlying)
		//	y := d.RegisterType(u)
		//	panic(fmt.Sprintf("%d != %d, %d, %d\naaa: %v\nbbb: %v\nccc: %v\n", m, n, x.index, y.index, underlyingTypeInfo.TT, t.Underlying.TT, t.TT))
		//}

		t.Underlying = underlyingTypeInfo
		underlyingTypeInfo.Underlying = underlyingTypeInfo

		if i, ok := underlying.(*types.Interface); ok && len(underlyingTypeInfo.AllMethods) > 0 { // i.NumMethods() > 0 {
			var uiInfo *UnderlyingInterfaceInfo
			if info := interfaceUnderlyings.At(i); info == nil {
				//interfaceUnderlyingTypes = append(interfaceUnderlyingTypes, underlyingTypeInfo)
				uiInfo = &UnderlyingInterfaceInfo{t: underlyingTypeInfo, underlieds: make([]*TypeInfo, 0, 3)}
				interfaceUnderlyings.Set(i, uiInfo)
				//log.Printf("!!! %T\n", uiInfo.t.TT)
			} else {
				uiInfo, _ = info.(*UnderlyingInterfaceInfo) // ToDo: remove _
			}
			uiInfo.underlieds = append(uiInfo.underlieds, t)
		}
	}

	//log.Println("number of underlying interfaces:", interfaceUnderlyings.Len())
	//interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
	//	uiInfo := info.(*UnderlyingInterfaceInfo)
	//	log.Println("     ", uiInfo.t.TT)
	//	for _, t := range uiInfo.underlieds {
	//		log.Println("           ", t.TT)
	//	}
	//})

	var lastMethodIndex uint32
	var allInterfaceMethods = make(map[MethodSignature]uint32, 8196)
	var method2TypeIndexes = make([][]uint32, 0, 8196)
	//var cache typeutil.MethodSetCache
	//resultMethodCache = &cache

	// 0 is an in valid method index
	allInterfaceMethods[MethodSignature{}] = 0
	method2TypeIndexes = append(method2TypeIndexes, nil)
	lastMethodIndex++

	// ...
	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
		uiInfo := info.(*UnderlyingInterfaceInfo)
		//log.Printf("### %d %T\n", uiInfo.t.index, uiInfo.t.TT)
		//methodSet := cache.MethodSet(uiInfo.t.TT)
		//uiInfo.methodIndexes = make([]uint32, methodSet.Len())
		selectors := uiInfo.t.AllMethods
		uiInfo.methodIndexes = make([]uint32, len(selectors))

		//for i := methodSet.Len() - 1; i >= 0; i-- {
		for i := len(selectors) - 1; i >= 0; i-- {
			x := d.lastTypeIndex

			//sel := methodSet.At(i)
			//funcObj, ok := sel.Obj().(*types.Func)
			//if !ok {
			//	panic("not a types.Func")
			//}
			//
			//sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure
			sel := selectors[i]
			//funcSig, ok := sel.Method.Type.TT.(*types.Signature)
			funcSig, ok := sel.Type().TT.(*types.Signature)
			if !ok {
				//panic(fmt.Sprintf("not a types.Signature: %T", sel.Method.Type.TT))
				panic(fmt.Sprintf("not a types.Signature: %T", sel.Type().TT))
			}
			pkgImportPath := ""
			if sel.Method.Pkg != nil {
				pkgImportPath = sel.Method.Pkg.Path
			}

			sig := d.BuildMethodSignatureFromFunctionSignature(funcSig, sel.Method.Name, pkgImportPath)

			// ToDo: if looks for go-ethereum project, here are some outputs. Check!?
			//if d.lastTypeIndex > x {
			//	log.Println("       > ", uiInfo.t.TT)
			//	log.Println("             >> ", sel)
			//}
			_ = x

			methodIndex, ok := allInterfaceMethods[sig]
			if ok {
				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], uiInfo.t.index)
			} else {
				methodIndex = lastMethodIndex
				sel.Method.index = methodIndex

				lastMethodIndex++
				allInterfaceMethods[sig] = methodIndex

				typeIndexes := make([]uint32, 0, 8)
				typeIndexes = append(typeIndexes, uiInfo.t.index)

				//log.Printf("   $$$ %d %T\n", uiInfo.t.index, d.allTypeInfos[uiInfo.t.index].TT)

				// method2TypeIndexes[methodIndex] = typeIndexes
				method2TypeIndexes = append(method2TypeIndexes, typeIndexes)
			}
			uiInfo.methodIndexes[i] = methodIndex

			//if len(selectors) == 1 {
			//	if sel.Name() == "Error" {
			//		log.Println("#################### uiInfo.t: ", uiInfo.t)
			//		log.Printf("=== methodIndex: %d %x %x",
			//			methodIndex,
			//			d.RegisterType(d.builtinPkg.PPkg.Types.Scope().Lookup("string").(*types.TypeName).Type()).index,
			//			d.RegisterType(types.Universe.Lookup("string").(*types.TypeName).Type()).index,
			//		)
			//		log.Printf("=== sig: %#v", sig)
			//	}
			//}
		}
	})

	//log.Println("number of method signatures:", lastMethodIndex, len(allInterfaceMethods), len(method2TypeIndexes))
	//for methodIndex, typeIndexes := range method2TypeIndexes {
	//	log.Println("     method#", methodIndex)
	//	for _, typeIndex := range typeIndexes {
	//		t := d.allTypeInfos[typeIndex]
	//		log.Printf("          %v : %T", t.TT, t.TT)
	//	}
	//}

	// log.Println("method2TypeIndexes = \n", method2TypeIndexes)

	for _, t := range d.allTypeInfos {
		//log.Println("111>>>", t.TT)
		if _, ok := t.TT.Underlying().(*types.Interface); ok {
			continue
		}

		//methodSet := cache.MethodSet(t.TT)
		selectors := t.AllMethods
		//log.Println("222>>>", t.TT, methodSet.Len())
		//for i := methodSet.Len() - 1; i >= 0; i-- {
		for i := len(selectors) - 1; i >= 0; i-- {
			//sel := methodSet.At(i)
			//funcObj, ok := sel.Obj().(*types.Func)
			//if !ok {
			//	panic("not a types.Func")
			//}
			//
			//sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure
			sel := selectors[i]
			//funcSig, ok := sel.Method.Type.TT.(*types.Signature)
			funcSig, ok := sel.Type().TT.(*types.Signature)
			if !ok {
				panic("not a types.Signature")
			}
			pkgImportPath := ""
			if sel.Method.Pkg != nil {
				pkgImportPath = sel.Method.Pkg.Path
			}

			sig := d.BuildMethodSignatureFromFunctionSignature(funcSig, sel.Method.Name, pkgImportPath)
			methodIndex, ok := allInterfaceMethods[sig]
			//log.Println("333>>>", methodIndex, ok)
			if ok {
				pt := d.RegisterType(types.NewPointer(t.TT))
				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], pt.index)
				if !sel.PointerReceiverOnly() {
					method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], t.index)
				}
			}

			//if len(selectors) == 1 {
			//	if sel.Name() == "Error" {
			//		log.Println("!!!!!!!!!!!!!!! t: ", t)
			//		log.Printf("=== methodIndex: %d %x %x",
			//			methodIndex,
			//			d.RegisterType(d.builtinPkg.PPkg.Types.Scope().Lookup("string").(*types.TypeName).Type()).index,
			//			d.RegisterType(types.Universe.Lookup("string").(*types.TypeName).Type()).index,
			//		)
			//		log.Printf("=== sig: %#v", sig)
			//	}
			//}
		}
	}

	//log.Println("number of interface method signatures:", lastMethodIndex, len(allInterfaceMethods), len(method2TypeIndexes))
	//for methodIndex, typeIndexes := range method2TypeIndexes {
	//	log.Println("     method#", methodIndex)
	//	for _, typeIndex := range typeIndexes {
	//		t := d.allTypeInfos[typeIndex]
	//		log.Println("          ", t.TT)
	//	}
	//}

	typeLookupTable := d.tempTypeLookupTable()
	defer d.resetTempTypeLookupTable()

	var searchRound uint32 = 0
	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
		uiInfo := info.(*UnderlyingInterfaceInfo)

		typeIndexes := method2TypeIndexes[uiInfo.methodIndexes[0]]
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			t.counter = searchRound + 1
		}
		searchRound++

		//if len(uiInfo.methodIndexes) == 1 {
		//	sel := uiInfo.t.AllMethods[0]
		//	if sel.Name() == "Error" {
		//		log.Println("======================================= uiInfo.t: ", uiInfo.t)
		//		log.Println("=== typeIndexes:", typeIndexes)
		//	}
		//}

		for _, methodIndex := range uiInfo.methodIndexes[1:] {
			typeIndexes = method2TypeIndexes[methodIndex]
			for _, typeIndex := range typeIndexes {
				t := d.allTypeInfos[typeIndex]
				if t.counter == searchRound {
					t.counter = searchRound + 1
				}
			}
			searchRound++
		}

		count := 0
		//typeIndexes = method2TypeIndexes[uiInfo.methodIndexes[len(uiInfo.methodIndexes)-1]]
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				////t.Implements = append(t.Implements, uiInfo.t)
				//t.Implements = append(t.Implements, uiInfo.underlieds...)
				for _, it := range uiInfo.underlieds {
					t.Implements = append(t.Implements, Implementation{Impler: t, Interface: it})
				}
				count++
			}
		}

		// ToDo: also apply this for findImplementations_Old
		registerTypeMethod := func(ti *TypeInfo) {
			if ti.TypeName == nil {
				return
			}

			//>> ToDo: now only show method implementations for origin types.
			if ti.TypeName.Denoting != ti {
				return
			}
			//<<

			pathPath := ti.TypeName.Package().Path
			for _, sel := range uiInfo.t.AllMethods {
				var selPkg string
				if !token.IsExported(sel.Name()) {
					selPkg = sel.Package().Path
				}
				d.registerTypeMethodContributingToTypeImplementations(pathPath, ti.TypeName.Name(), selPkg, sel.Name())
			}
		}

		// Register non-pointer ones firstly, then
		// register pointer ones whose bases have not been registered.
		d.resetTempTypeLookupTable()
		impBys := make([]*TypeInfo, 0, count)
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if _, ok := t.TT.(*types.Pointer); !ok {
					if itt, ok := t.TT.Underlying().(*types.Interface); ok {
						ittInfo := interfaceUnderlyings.At(itt).(*UnderlyingInterfaceInfo)
						for _, it := range ittInfo.underlieds {
							impBys = append(impBys, it)
							typeLookupTable[it.index] = struct{}{}
						}
					} else {
						impBys = append(impBys, t)
						typeLookupTable[typeIndex] = struct{}{}

						registerTypeMethod(t)
					}
				}
			}
		}
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if ptt, ok := t.TT.(*types.Pointer); ok {
					bt := d.RegisterType(ptt.Elem())
					if _, reged := typeLookupTable[bt.index]; !reged {
						impBys = append(impBys, t)

						//registerTypeMethod(t)
						registerTypeMethod(bt)
					}
				}
			}
		}
		uiInfo.t.ImplementedBys = impBys

		//log.Println("111 @@@", uiInfo.t.TT, ", uiInfo.methodIndexes:", uiInfo.methodIndexes)
		//for _, impBys := range impBys {
		//	log.Println("     ", impBys.TT)
		//}
	})

	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
		uiInfo := info.(*UnderlyingInterfaceInfo)
		for _, t := range uiInfo.underlieds {
			t.Implements = uiInfo.t.Implements
			t.ImplementedBys = uiInfo.t.ImplementedBys
		}
	})

	//for _, t := range d.allTypeInfos {
	//	if len(t.Implements) > 0 {
	//		log.Println(t.TT, "implements:")
	//		for _, it := range t.Implements {
	//			log.Println("     ", it.TT)
	//		}
	//	}
	//}

	for _, t := range d.allTypeInfos {
		if len(t.Implements) == 0 {
			continue
		}

		if ptt, ok := t.TT.(*types.Pointer); ok {
			bt := d.RegisterType(ptt.Elem()) // 333 b: here to check why new types are registered
			//bt.StarImplements = t.Implements

			// merge non-pointer and pointer implements.
			d.resetTempTypeLookupTable()
			for _, impl := range bt.Implements {
				typeLookupTable[impl.Interface.index] = struct{}{}
			}
			for _, impl := range t.Implements {
				if _, ok := typeLookupTable[impl.Interface.index]; ok {
					continue
				}
				//impl := impl // not needed, for the .Implements slice element is not pointer.
				bt.Implements = append(bt.Implements, impl)
			}
			t.Implements = nil

			// remove unnamed interfaces whose have named underlieds.
			// ToDo: avoid removing aliases to unnamed ones.
			// (The work is moved to package datail page generation.)
		}
	}

	return
}

// ToDo:
// The current implementation-finding algorithm uses TypeInfo.index as judge conidition.
// So the implementation is not ok for concurrency safe. To make it concurrentcy safe,
// we can sort each method2TypeIndexes slices, and copy the one for the first method,
// then get the overlapping for consequencing method slices.
// However, it looks the current implementation is fast enough.

//func (d *CodeAnalyzer) findImplementations_Old() (resultMethodCache *typeutil.MethodSetCache) {
//	// step 1: register all method signatures of underlying interface types.
//	//         create a type list for each signature.
//	// step 2: iteration all types, calculate their method signatures,
//	//         (interface types can use their underlying cache calculated in step 1)
//	//         ignore signatures which are note recorded in step 1.
//	//         register the type into the type lists of method signatures.
//	// step 3: iterate all underlying interfaces, iterate all method signatures,
//	//         iterate the type list of a signature, TypeInfo.counter++
//	//         ...
//
//	type UnderlyingInterfaceInfo struct {
//		t             *TypeInfo
//		methodIndexes []uint32
//		underlieds    []*TypeInfo // including the underlying itself
//	}
//
//	// ToDo: use map[InterfaceTypeIndex]*TypeInfo?
//	var interfaceUnderlyings typeutil.Map
//
//	//var interfaceUnderlyingTypes = make([]*TypeInfo, 0, 1024)
//	//for _, t := range d.allTypeInfos {
//	// New types might be registered in this loop,
//	// so traditional for-loop is used here.
//	for i := 0; i < len(d.allTypeInfos); i++ {
//		t := d.allTypeInfos[i]
//
//		// ToDo: auto register underlying type in RegisterType.
//		underlying := t.TT.Underlying()
//		underlyingTypeInfo := d.RegisterType(underlying) // underlying must have been already registered
//		t.Underlying = underlyingTypeInfo
//		underlyingTypeInfo.Underlying = underlyingTypeInfo
//
//		if i, ok := underlying.(*types.Interface); ok && i.NumMethods() > 0 {
//			var uiInfo *UnderlyingInterfaceInfo
//			info := interfaceUnderlyings.At(i)
//			if interfaceUnderlyings.At(i) == nil {
//				//interfaceUnderlyingTypes = append(interfaceUnderlyingTypes, underlyingTypeInfo)
//				uiInfo = &UnderlyingInterfaceInfo{t: underlyingTypeInfo, underlieds: make([]*TypeInfo, 0, 3)}
//				interfaceUnderlyings.Set(i, uiInfo)
//				//log.Printf("!!! %T\n", uiInfo.t.TT)
//			} else {
//				uiInfo, _ = info.(*UnderlyingInterfaceInfo)
//			}
//			uiInfo.underlieds = append(uiInfo.underlieds, t)
//		}
//	}
//
//	log.Println("number of underlying interfaces:", interfaceUnderlyings.Len())
//	//interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
//	//	uiInfo := info.(*UnderlyingInterfaceInfo)
//	//	log.Println("     ", uiInfo.t.TT)
//	//	for _, t := range uiInfo.underlieds {
//	//		log.Println("           ", t.TT)
//	//	}
//	//})
//
//	var lastMethodIndex uint32
//	var allInterfaceMethods = make(map[MethodSignature]uint32, 8196)
//	var method2TypeIndexes = make([][]uint32, 0, 8196)
//	var cache typeutil.MethodSetCache
//	resultMethodCache = &cache
//
//	// 0 is an in valid method index
//	allInterfaceMethods[MethodSignature{}] = 0
//	method2TypeIndexes = append(method2TypeIndexes, nil)
//	lastMethodIndex++
//
//	// ...
//	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
//		uiInfo := info.(*UnderlyingInterfaceInfo)
//		//log.Printf("### %d %T\n", uiInfo.t.index, uiInfo.t.TT)
//		methodSet := cache.MethodSet(uiInfo.t.TT)
//		uiInfo.methodIndexes = make([]uint32, methodSet.Len())
//
//		for i := methodSet.Len() - 1; i >= 0; i-- {
//			sel := methodSet.At(i)
//			funcObj, ok := sel.Obj().(*types.Func)
//			if !ok {
//				panic("not a types.Func")
//			}
//			x := d.lastTypeIndex
//			sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure
//			if d.lastTypeIndex > x {
//				log.Println("       > ", uiInfo.t.TT)
//				log.Println("             >> ", sel)
//			}
//
//			methodIndex, ok := allInterfaceMethods[sig]
//			if ok {
//				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], uiInfo.t.index)
//			} else {
//				methodIndex = lastMethodIndex
//				lastMethodIndex++
//				allInterfaceMethods[sig] = methodIndex
//
//				typeIndexes := make([]uint32, 0, 8)
//				typeIndexes = append(typeIndexes, uiInfo.t.index)
//
//				//log.Printf("   $$$ %d %T\n", uiInfo.t.index, d.allTypeInfos[uiInfo.t.index].TT)
//
//				// method2TypeIndexes[methodIndex] = typeIndexes
//				method2TypeIndexes = append(method2TypeIndexes, typeIndexes)
//			}
//			uiInfo.methodIndexes[i] = methodIndex
//		}
//	})
//
//	//log.Println("number of method signatures:", lastMethodIndex, len(allInterfaceMethods), len(method2TypeIndexes))
//	//for methodIndex, typeIndexes := range method2TypeIndexes {
//	//	log.Println("     method#", methodIndex)
//	//	for _, typeIndex := range typeIndexes {
//	//		t := d.allTypeInfos[typeIndex]
//	//		log.Printf("          %v : %T", t.TT, t.TT)
//	//	}
//	//}
//
//	// log.Println("method2TypeIndexes = \n", method2TypeIndexes)
//
//	for _, t := range d.allTypeInfos {
//		//log.Println("111>>>", t.TT)
//		if _, ok := t.TT.Underlying().(*types.Interface); ok {
//			continue
//		}
//
//		methodSet := cache.MethodSet(t.TT)
//		//log.Println("222>>>", t.TT, methodSet.Len())
//		for i := methodSet.Len() - 1; i >= 0; i-- {
//			sel := methodSet.At(i)
//			funcObj, ok := sel.Obj().(*types.Func)
//			if !ok {
//				panic("not a types.Func")
//			}
//
//			sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure
//
//			methodIndex, ok := allInterfaceMethods[sig]
//			//log.Println("333>>>", methodIndex, ok)
//			if ok {
//				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], t.index)
//			}
//		}
//	}
//
//	log.Println("number of interface method signatures:", lastMethodIndex, len(allInterfaceMethods), len(method2TypeIndexes))
//	//for methodIndex, typeIndexes := range method2TypeIndexes {
//	//	log.Println("     method#", methodIndex)
//	//	for _, typeIndex := range typeIndexes {
//	//		t := d.allTypeInfos[typeIndex]
//	//		log.Println("          ", t.TT)
//	//	}
//	//}
//
//	typeLookupTable := d.tempTypeLookupTable()
//	defer d.resetTempTypeLookupTable()
//
//	var searchRound uint32 = 0
//	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
//		uiInfo := info.(*UnderlyingInterfaceInfo)
//
//		typeIndexes := method2TypeIndexes[uiInfo.methodIndexes[0]]
//		for _, typeIndex := range typeIndexes {
//			t := d.allTypeInfos[typeIndex]
//			t.counter = searchRound + 1
//		}
//		searchRound++
//
//		if len(uiInfo.methodIndexes) == 1 {
//			sel := uiInfo.t.AllMethods[0]
//			if sel.Name() == "Error" {
//				log.Println("======================================= uiInfo.t: ", uiInfo.t)
//				log.Println("=== typeIndexes:", typeIndexes)
//			}
//		}
//
//		for _, methodIndex := range uiInfo.methodIndexes[1:] {
//			typeIndexes = method2TypeIndexes[methodIndex]
//			for _, typeIndex := range typeIndexes {
//				t := d.allTypeInfos[typeIndex]
//				if t.counter == searchRound {
//					t.counter = searchRound + 1
//				}
//			}
//			searchRound++
//		}
//
//		count := 0
//		//typeIndexes = method2TypeIndexes[uiInfo.methodIndexes[len(uiInfo.methodIndexes)-1]]
//		for _, typeIndex := range typeIndexes {
//			t := d.allTypeInfos[typeIndex]
//			if t.counter == searchRound {
//				////t.Implements = append(t.Implements, uiInfo.t)
//				//t.Implements = append(t.Implements, uiInfo.underlieds...)
//				for _, it := range uiInfo.underlieds {
//					t.Implements = append(t.Implements, Implementation{Impler: t, Interface: it})
//				}
//				count++
//			}
//		}
//
//		// Register non-pointer ones firstly, then
//		// register pointer ones whose bases have not been registered.
//		d.resetTempTypeLookupTable()
//		impBy := make([]*TypeInfo, 0, count)
//		for _, typeIndex := range typeIndexes {
//			t := d.allTypeInfos[typeIndex]
//			if t.counter == searchRound {
//				if _, ok := t.TT.(*types.Pointer); !ok {
//					if itt, ok := t.TT.Underlying().(*types.Interface); ok {
//						ittInfo := interfaceUnderlyings.At(itt).(*UnderlyingInterfaceInfo)
//						for _, it := range ittInfo.underlieds {
//							impBy = append(impBy, it)
//							typeLookupTable[it.index] = struct{}{}
//						}
//					} else {
//						impBy = append(impBy, t)
//						typeLookupTable[typeIndex] = struct{}{}
//					}
//				}
//			}
//		}
//		for _, typeIndex := range typeIndexes {
//			t := d.allTypeInfos[typeIndex]
//			if t.counter == searchRound {
//				if ptt, ok := t.TT.(*types.Pointer); ok {
//					bt := d.RegisterType(ptt.Elem())
//					if _, reged := typeLookupTable[bt.index]; !reged {
//						impBy = append(impBy, t)
//					}
//				}
//			}
//		}
//		uiInfo.t.ImplementedBys = impBy
//
//		//log.Println("111 @@@", uiInfo.t.TT, ", uiInfo.methodIndexes:", uiInfo.methodIndexes)
//		//for _, impBy := range impBy {
//		//	log.Println("     ", impBy.TT)
//		//}
//	})
//
//	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
//		uiInfo := info.(*UnderlyingInterfaceInfo)
//		for _, t := range uiInfo.underlieds {
//			t.Implements = uiInfo.t.Implements
//			t.ImplementedBys = uiInfo.t.ImplementedBys
//		}
//	})
//
//	//for _, t := range d.allTypeInfos {
//	//	if len(t.Implements) > 0 {
//	//		log.Println(t.TT, "implements:")
//	//		for _, it := range t.Implements {
//	//			log.Println("     ", it.TT)
//	//		}
//	//	}
//	//}
//
//	for _, t := range d.allTypeInfos {
//		if len(t.Implements) == 0 {
//			continue
//		}
//
//		if ptt, ok := t.TT.(*types.Pointer); ok {
//			bt := d.RegisterType(ptt.Elem()) // 333 b: here to check why new types are registered
//			//bt.StarImplements = t.Implements
//
//			// merge non-pointer and pointer implements.
//			d.resetTempTypeLookupTable()
//			for _, impl := range bt.Implements {
//				typeLookupTable[impl.Interface.index] = struct{}{}
//			}
//			for _, impl := range t.Implements {
//				if _, ok := typeLookupTable[impl.Interface.index]; ok {
//					continue
//				}
//				//impl := impl // not needed, for the .Implements slice element is not pointer.
//				bt.Implements = append(bt.Implements, impl)
//			}
//			t.Implements = nil
//
//			// remove unnamed interfaces whose have named underlieds.
//			// ToDo: avoid removing aliases to unnamed ones.
//			// (The work is moved to package datail page generation.)
//		}
//	}
//
//	return
//}

// This method should only be called when all selectors are confirmed.
func (d *CodeAnalyzer) registerNamedInterfaceMethodsForInvolvedTypeNames() {

	for _, pkg := range d.packageList {

		// ToDo:
		// sometime situations are much complicated.
		// An interface method might have several origins.
		// A global method table (key is method signature) is needed. Each method in the table
		// * might have several source postions/comments/documents.
		// (A global function table might be also needed. Method signature = func sig + method name)
		for _, tn := range pkg.AllTypeNames {
			// ToDo: in fact, some unexported interface names might also export methods.
			if !tn.Exported() {
				continue
			}

			t := tn.Denoting
			if _, ok := t.TT.Underlying().(*types.Interface); !ok {
				continue
			}

			for _, sel := range t.AllMethods {
				// need? registerFunctionForInvolvedTypeNames will also check it.
				if !token.IsExported(sel.Name()) {
					continue
				}

				//sig, ok := sel.Method.Type.TT.(*types.Signature)
				sig, ok := sel.Type().TT.(*types.Signature)
				if !ok {
					panic("impossible")
				}

				params, results := sig.Params(), sig.Results()
				m, n := 0, 0
				if params != nil {
					m = params.Len()
				}
				if results != nil {
					n = results.Len()
				}
				if m == 0 && n == 0 {
					continue
				}

				im := &InterfaceMethod{
					InterfaceTypeName: tn,
					//Method:            sel.Method,
					Selector: sel,
				}
				_, _, _ = d.registerFunctionForInvolvedTypeNames(im)
			}
		}
	}
}

func (d *CodeAnalyzer) collectSelectors() {
	//log.Println("=== analyze struct promoted fields/methods ...")

	// The method set returned by types.NewMethodSet loses much info.
	// So a custom implementation is needed.

	//var printSelectors = func(t *TypeInfo) {
	//	if t.DirectSelectors != nil {
	//		for i, sel := range t.DirectSelectors {
	//			log.Println(i, ">", sel.Id)
	//		}
	//	}
	//}

	var selectorMaps []map[string]*Selector

	var smm = &SeleterMapManager{
		apply: func() (r map[string]*Selector) {
			if selectorMaps == nil {
				selectorMaps = make([]map[string]*Selector, 8, 32)
				for i := range selectorMaps {
					selectorMaps[i] = make(map[string]*Selector, 128)
				}
			}
			if n := len(selectorMaps); n > 0 {
				r = selectorMaps[n-1]
				selectorMaps = selectorMaps[:n-1]
				return
			}
			log.Println("more than", len(selectorMaps), "being used now.")
			r = make(map[string]*Selector, 128)
			return
		},
		release: func(r map[string]*Selector) {
			for k := range r {
				delete(r, k)
			}

			if selectorMaps == nil {
				//return // should not
				panic("should not")
			}

			if len(selectorMaps) >= cap(selectorMaps) {
				log.Println("more than", cap(selectorMaps), "in free now.")
				return
			}

			selectorMaps = append(selectorMaps, r)
		},
	}

	var currentCounter uint32

	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]
		t.counter = 0
	}

	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]

		currentCounter++ // faster than map
		//log.Println("===================================", currentCounter)
		d.collectSelectorsForInterfaceType(t, 0, currentCounter, smm)
	}

	var checkedTypes = make(map[uint32]uint16) // type index: embedding depth
	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]

		//currentCounter++ // can't replace map

		d.collectSelectorsForNonInterfaceType(t, smm, checkedTypes)

		// print selectors
		//if len(t.AllMethods)+len(t.AllFields) > 0 {
		//	log.Println("============== t=", t)
		//}
		//if len(t.AllMethods) > 0 {
		//	PrintSelectors("methods", t.AllMethods)
		//}
		//if len(t.AllFields) > 0 {
		//	PrintSelectors("fields", t.AllFields)
		//}
	}

	// ToDo: verify the methodsets are the same as typeutl.MethodSet

	//var interfaceUnderlyingTypes = make([]*TypeInfo, 0, 1024)
	//for _, t := range d.allTypeInfos {
	// New types might be registered in this loop,
	// so traditional for-loop is used here.
	//for i := 0; i < len(d.allTypeInfos); i++ {
	//	t := d.allTypeInfos[i]
	//	underlying := t.TT.Underlying()
	//	switch
	//	if i, ok := underlying.(*types.Interface); ok && i.NumMethods() > 0 {
	//	}
	//}

	//log.Println(types.Unsafe.Scope().Lookup("Pointer").Type().Underlying())
	//log.Printf("%v", types.Unsafe.Scope().Lookup("Sizeof").Type())

	//log.Printf("%v", d.packageTable["builtin"])
	//log.Printf("%v", d.packageTable["builtin"].PPkg)
	//log.Printf("%v", d.packageTable["builtin"].PPkg.Types)
	//log.Printf("%v", d.packageTable["builtin"].PPkg.Types.Scope())
	//log.Printf("%v", d.packageTable["builtin"].PPkg.Types.Scope().Lookup("len"))
	//log.Printf("%v", d.packageTable["builtin"].PPkg.Types.Scope().Lookup("print").Type())

	// ToDo: iterate all types, and register some of them in respective pacakges.
	// * exported declared type aliases and named types
	// * non-exported declared type aliases and named types
	// * unnamed types which are types of exported variables/fields
	// For all these types,
	// * record which exported functions use them.
	// * unnamed pointer types will be ignored, their method set
	//   recoreded in their respective base types.

	// When generate docs:
	// * unnamed chan/array/map/slice/func/pointer types are not important.
	// * some unnamed interface and struct types are important.
	//

	// The ssame unnamed typed might appear in several different declarations.
	// The docs for declarations might be different.
}

// ...

//type selectListManager struct {
//	current *list.List
//	free    *list.List
//}

//func newSelectListManager() *selectListManager {
//	return &selectListManager{
//		current: list.New(),
//		free:    list.New(),
//	}
//}

type SeleterMapManager struct {
	apply   func() (r map[string]*Selector)
	release func(r map[string]*Selector)
}

var debug = false

func (d *CodeAnalyzer) collectSelectorsForInterfaceType(t *TypeInfo, depth int, currentCounter uint32, smm *SeleterMapManager) (r bool) {

	//if !debug {
	//	debug = t.TypeName != nil && t.TypeName.Name() == "Hasher111"
	//	if debug {
	//		defer func() {
	//			debug = false
	//		}()
	//	}
	//}

	//if debug {
	//	log.Println(">>> ", t, ", depth:", depth, ", counters:", t.counter, currentCounter, ", promoted:", (t.attributes&promotedSelectorsCollected) != 0)
	//}

	if t.counter == currentCounter {
		//panic(fmt.Sprintf("recursive interface embedding. %d, %s", t.counter, t.TT))
		r = true
		return
	}

	t.counter = currentCounter
	//log.Println(">>> ", t.counter, t.TT)

	if (t.attributes & promotedSelectorsCollected) != 0 {
		return
	}

	// ToDo: maintain an interface type list in the outer loop to avoid the assertion.
	itt, ok := t.TT.Underlying().(*types.Interface)
	if !ok {
		return
	}

	//>> 1.18
	if isTypeParam(t.TT) {
		return
	}
	//<<

	if t.Underlying == nil {
		// already set interface.Underlying in RegisterType now
		panic(fmt.Sprint("should never happen:", t.TT))

		// ToDo: move to RegisterType.
		//underlying := t.TT.Underlying()
		//UnderlyingTypeInfo := d.RegisterType(underlying)
		//t.Underlying = UnderlyingTypeInfo
		//UnderlyingTypeInfo.Underlying = UnderlyingTypeInfo
	}

	t.attributes |= promotedSelectorsCollected

	// ToDo: field and parameter/result interface types don't satisfy this.
	//if (t.Underlying.attributes & directSelectorsCollected) == 0 {
	//	panic("unnamed interface should have collected direct selectors now. " + fmt.Sprintf("%#v", t))
	//}

	if t != t.Underlying {
		// t is a named interface type.

		//log.Println("222", depth)
		if (t.Underlying.attributes & directSelectorsCollected) == 0 {

			//>> 1.18
			// ToDo: still needed?
			if isParameterizedType(t.TT) {
				return
			}
			//<<

			panic("unnamed interface should have collected direct selectors now. " +
				fmt.Sprintf("underlying index: %v. index: %v. name: %#v. %#v. %v",
					t.Underlying.index,
					t.index,
					t.TypeName.Name(),
					t.Underlying.TT,
					t.TT.Underlying(),
				),
			)
		}
		//if t.DirectSelectors != nil {
		//	panic("Selectors of named interface should be blank now")
		//}
		d.collectSelectorsForInterfaceType(t.Underlying, depth, currentCounter, smm)

		t.DirectSelectors = t.Underlying.DirectSelectors
		t.AllMethods = t.Underlying.AllMethods
	} else { // t == t.Underlying
		if t.Underlying.attributes&directSelectorsCollected == 0 {
			//if depth == 0 {
			//	return // ToDo: temp ignore field and parameter/result unnamed interface types
			//}
			//log.Printf("!!! %v:", t.TT)
			//log.Printf("!!! %v:", t.Underlying)
			itt := t.TT.(*types.Interface)
			var bd strings.Builder
			bd.Grow(2048)
			fmt.Fprintf(&bd, "  IsMethodSet(): %v\n", itt.IsMethodSet())
			fmt.Fprintf(&bd, "  IsImplicit(): %v\n", itt.IsImplicit())
			fmt.Fprintf(&bd, "  NumEmbeddeds(): %v\n", itt.NumEmbeddeds())
			for i := 0; i < itt.NumEmbeddeds(); i++ {
				fmt.Fprintf(&bd, "    EmbeddedType(%v): %v\n", i, itt.EmbeddedType(i))
			}
			fmt.Fprintf(&bd, "  NumMethods(): %v\n", itt.NumMethods())
			for i := 0; i < itt.NumMethods(); i++ {
				fmt.Fprintf(&bd, "    Method(%v): %v\n", i, itt.Method(i))
			}
			panic(fmt.Sprintf("unnamed interface should have collected direct selectors now. %#v.\nMore info:\n%s", t, bd.String()))
		}

		//hasEmbeddings := false
		//for _, s := range t.DirectSelectors {
		//	if s.Field != nil {
		//		hasEmbeddings = true
		//		break
		//	}
		//}

		// Interface type composite literals become very flexible now,
		// so that even if no embedding fields, duplicated methods may still exist.

		//if n := itt.NumEmbeddeds(); n == 0 {
		//if !hasEmbeddings { // the embedding ones might overlap with non-embedding ones
		//	t.AllMethods = t.DirectSelectors
		//} else {
		selectors := smm.apply()
		defer func() {
			smm.release(selectors)
		}()

		t.AllMethods = make([]*Selector, 0, len(t.DirectSelectors)+2*itt.NumEmbeddeds())

		for _, sel := range t.DirectSelectors {
			if sel.Method != nil {
				if old, ok := selectors[sel.Id]; ok {
					//if old.Method.Type != sel.Method.Type {
					if old.Type() != sel.Type() {
						panic("direct overlapped interface methods and signatures are different")
					} else {
						//log.Println("$$$ overlapping interface method:", sel.Id, ". (allowed since Go 1.14)")
						//log.Println("            ", t.TT)
						//log.Println("            ", t.DirectSelectors)

						// ToDo: go-ethethum has 3 such cases? why?
						//panic("direct overlapped interface methods are not allowed")
					}
				} else {
					selectors[sel.Id] = sel
					t.AllMethods = append(t.AllMethods, sel)
				}
			} else { // sel.Field != nil

				// It is some quirk here. An unnamed interface type
				// interface {I} might be the underlying type of
				// named interface type I.

				//log.Println("      ", sel.Field.Type)
				//d.collectSelectorsForInterfaceType(sel.Field.Type, depth+1, currentCounter, smm)
				//for _, sel := range sel.Field.Type.AllMethods {
				//	if old, ok := selectors[sel.Id]; ok {
				//		if old.Method.Type != sel.Method.Type {
				//			panic("overlapped interface methods but signatures are different")
				//		} else {
				//			log.Println("overlapping interface method:", sel.Id, ". (allowed since Go 1.14)")
				//		}
				//	} else {
				//		selectors[sel.Id] = sel
				//		t.AllMethods = append(t.AllMethods, sel)
				//	}
				//}

				// ToDo: verify the correctness of the following implementation.

				//d.collectSelectorsForInterfaceType(sel.Field.Type, depth+1, currentCounter, smm)
				//for _, sel := range sel.Field.Type.AllMethods {
				//ut := sel.Field.Type.Underlying
				ut := sel.Type().Underlying

				// ToDo: if t == ut, the following line should be a no-op.
				//       The logic is right?
				d.collectSelectorsForInterfaceType(ut, depth+1, currentCounter, smm)
				if t != ut { // it is possible t == ut
					for _, sel := range ut.AllMethods {
						if old, ok := selectors[sel.Id]; ok {
							//if old.Method.Type != sel.Method.Type {
							if old.Type() != sel.Type() {
								panic("overlapped interface methods but signatures are different")
							} else {
								// ToDo: The current implementation does not always find true overlappings.
								//log.Println("overlapping interface method:", sel.Id, ". (allowed since Go 1.14)")
							}
						} else {
							selectors[sel.Id] = sel
							t.AllMethods = append(t.AllMethods, sel)
						}
					}
				}
			}

		}

		//log.Println(depth, "===", len(t.DirectSelectors), len(t.AllMethods), t.TT)
		//}
	}

	return
}

func (d *CodeAnalyzer) collectSelectorsForNonInterfaceType(t *TypeInfo, smm *SeleterMapManager, checkedTypes map[uint32]uint16) {
	if (t.attributes & promotedSelectorsCollected) != 0 {
		return
	}

	defer func() {
		t.attributes |= promotedSelectorsCollected
	}()

	var namedType *TypeInfo  // which DirectSelectors are all explicit methods now.
	var structType *TypeInfo // which DirectSelectors are all direct fields now.

	switch t.TT.(type) {
	case *types.Named:
		namedType = t

		if namedType.TypeName == nil {

		} else if namedType.TypeName.Name() == "A" {
			debug = true
			defer func() {
				debug = false
			}()
		}

		switch t.Underlying.TT.(type) {
		case *types.Struct:
			structType = t.Underlying
			break
		case *types.Interface:
			// already done in collectSelectorsForInterfaceType.
			return
		case *types.Pointer:
			// named pointer types have no selectors.
			return
		default:
			t.AllMethods = t.DirectSelectors
			// no promoted selectors to collect.
			return
		}
	case *types.Struct:
		structType = t
		break
	case *types.Interface:
		// already done in collectSelectorsForInterfaceType.
		return
	case *types.Pointer:
		// selectors of *T will be recoreded in T.selectors, except T is an interface or pointer.
		return
	default:
		// Basics and other unnamed types have no selectors.
		return
	}

	if structType == nil {
		panic("should not")
	}

	if namedType == nil {

		//	if len(structType.DirectSelectors) == 1 {
		//		sel := structType.DirectSelectors[0]
		//		if sel.Field != nil &&
		//		sel.Field.Mode == EmbedMode_Direct &&
		//		sel.Field.Type.TypeName != nil &&
		//		sel.Field.Type.TypeName.Name() == "Type" &&
		//		//sel.Field.Type.TypeName.Package().Path == "go/types" &&
		//		true {
		//			log.Println("=====================", sel.Field.Type.TypeName.Name(), sel.Field.Type.TypeName.Package().Path, sel, "==========", sel.Field.Type.TypeName)
		//		}
		//	}

		numEmbeddeds := 0
		for _, sel := range structType.DirectSelectors {
			if sel.Field == nil {
				panic("should not")
			}
			if sel.Field.Mode != EmbedMode_None {
				numEmbeddeds++
				// ToDo: may break here?
			}
		}

		// The simple case.
		if numEmbeddeds == 0 {
			t.AllFields = t.DirectSelectors
			if namedType != nil { // ToDo: always false?
				t.AllMethods = t.DirectSelectors
			}

			// no promoted selectors, so return.
			return
		}

		// ...
		defer func() {
			for k := range checkedTypes {
				delete(checkedTypes, k)
			}
		}()

		// map[string]*Selector
		selectorMap := smm.apply()
		defer smm.release(selectorMap)

		selectorList := list.New()
		defer func() {
			numFields, numMethods := 0, 0
			for e := selectorList.Front(); e != nil; e = e.Next() {
				sel := e.Value.(*Selector)
				if sel.cond != SelectorCond_Hidden {
					//t.AllFields = append(t.AllFields, sel)
					if sel.Field != nil {
						numFields++
					} else { // sel.Method != nil
						numMethods++
					}
				}
			}

			t.AllFields = make([]*Selector, 0, numFields)
			t.AllMethods = make([]*Selector, 0, numMethods)

			for e := selectorList.Front(); e != nil; e = e.Next() {
				sel := e.Value.(*Selector)
				if sel.cond != SelectorCond_Hidden {
					if sel.Field != nil {
						t.AllFields = append(t.AllFields, sel)
					} else { // sel.Method != nil
						t.AllMethods = append(t.AllMethods, sel)
					}
				}
			}
		}()

		// Collect direct fields
		//structType.counter = currentCounter
		checkedTypes[structType.index] = 0
		for _, sel := range structType.DirectSelectors {
			if _, exist := selectorMap[sel.Id]; exist {
				panic("should not")
			} else {
				selectorMap[sel.Id] = sel
				selectorList.PushBack(sel)
			}
		}

		//log.Println("number of direct selectors:", selectorList.Len())

		// Returns how many new promoted embedded fields are inserted. (Not quite useful acctually.)
		var collectSelectorsFromEmbeddedField = func(embeddedField *Selector, insertAfter *list.Element) (numNewPromotedEmbeddedFields int) {
			depth := embeddedField.Depth + 1

			////if embeddedField.Field.Type.counter == currentCounter {
			////	return
			////}
			////embeddedField.Field.Type.counter = currentCounter
			//if d, checked := checkedTypes[embeddedField.Field.Type.index]; checked && d < depth {
			//	return
			//}
			//checkedTypes[embeddedField.Field.Type.index] = depth
			// Will do it below.

			embeddingChain := &EmbeddedField{
				Field: embeddedField.Field,
				Prev:  embeddedField.EmbeddingChain,
			}

			collect := func(t *TypeInfo, selectors []*Selector, indrect bool) {
				mustConflict := false
				if d, checked := checkedTypes[t.index]; checked {
					if d > depth {
						panic("impossible")
					}
					if d < depth {
						// no needs to continue
						return
					}
					mustConflict = true
					//log.Println("         old >>>", depth, d, t)
				} else {
					checkedTypes[t.index] = depth
					//log.Println("         new >>>", depth, t)
				}

				//log.Println("             >>>", len(selectors))

				for _, sel := range selectors {
					//log.Println("         ???", sel.Id)
					newCond := SelectorCond_Normal
					if old, exist := selectorMap[sel.Id]; exist {
						if old.Depth == depth {
							//log.Println("         !!! collide", sel.Id)
							old.cond = SelectorCond_Hidden
						} else if old.cond == SelectorCond_Normal { // old.Depth < depth
							//log.Println("         !!! shadow", sel.Id)
							//old.cond = SelectorCond_Shadowing
						}
						newCond = SelectorCond_Hidden
					} else {
						if mustConflict {
							panic("not conflict?! " + sel.Id)
						}
					}

					new := &Selector{
						Id:             sel.Id,
						Field:          sel.Field,
						Method:         sel.Method,
						Instantiated:   sel.Instantiated,
						RealType:       sel.RealType,
						EmbeddingChain: embeddingChain,
						Depth:          depth,
						Indirect:       embeddedField.Indirect || indrect,
						cond:           newCond,
					}
					selectorMap[sel.Id] = new
					insertAfter = selectorList.InsertAfter(new, insertAfter)
					//log.Println("         !!! add", new.Id)

					if new.Field != nil && new.Field.Mode != EmbedMode_None {
						numNewPromotedEmbeddedFields++
					}
				}
			}

			//log.Println("       000", embeddedField.Field.Type)
			//switch t := embeddedField.Field.Type; tt := t.TT.(type) {
			switch t := embeddedField.Type(); tt := t.TT.(type) {
			case *types.Named:
				// ToDo: since Go 1.18, this might be an instantiated type of a generic type.
				//       See: types.Info.Instances, should be handled in the collect phase.

				// ToDo: Note: the underlying is an interface if t is a generic type.

				//>> 1.18 (temp handling, need more handling. bug#33)
				//<<

				//log.Println(tt)
				//if s := tt.String(); strings.HasSuffix(s, "SSS[int]") {
				//	log.Printf("=== %[1]T, %[1]v", t.Underlying.TT)
				//	log.Printf("=== %[1]T, %[1]v", tt.Underlying())
				//}

				switch t.Underlying.TT.(type) {
				case *types.Struct:
					//log.Println("       111 aaa")
					// Collect direct methods
					collect(t, t.DirectSelectors, false)
					// Collect direct fields
					collect(t.Underlying, t.Underlying.DirectSelectors, false)
				case *types.Interface:
					//log.Println("       111 bbb")
					// Collect all methods
					collect(t, t.AllMethods, false) // <=> t.Underlying.AllMethods
				case *types.Pointer:
					// ToDo: merged with the default branch.
					//log.Println("       111 ccc")
					// named pointer types have no selectors.
				default:
					//log.Println("       111 ddd")
					// Collect direct methods
					collect(t, t.DirectSelectors, false)
				}
			case *types.Struct:
				//log.Println("       222")
				// Collect direct fields
				collect(t, t.DirectSelectors, false)
			case *types.Interface:
				//log.Println("       333")
				// Collect all methods
				collect(t, t.AllMethods, false)
			case *types.Pointer:
				//log.Println("       444")
				baseType := d.RegisterType(tt.Elem())
				switch baseTT := baseType.TT.(type) {
				case *types.Struct:
					//log.Println("       444 aaa")
					// Collect direct fields
					collect(baseType, baseType.DirectSelectors, true)
				case *types.Named:
					switch baseType.Underlying.TT.(type) {
					case *types.Struct:
						//log.Println("       444 bbb 111")
						// Collect direct methods
						collect(baseType, baseType.DirectSelectors, true)
						// Collect direct fields
						collect(baseType.Underlying, baseType.Underlying.DirectSelectors, true)
					case *types.Interface, *types.Pointer:
						//log.Println("       444 bbb 222")
						// None to collect. Not embeddable actually.
					default:
						//log.Println("       444 bbb 333")
						// Collect direct methods
						collect(baseType, baseType.DirectSelectors, true)
					}
				default:
					_ = baseTT
					//log.Println("       444 bbb 444", baseTT)
				}
			default:
				//log.Println("      555", tt)
			}

			return
		}

		for depth := uint16(0); ; depth++ {
			//if debug {
			//	log.Println("   ~~~ depth=", depth)
			//}
			needToCheckDeepers := false
			for e := selectorList.Front(); e != nil; e = e.Next() {
				sel := e.Value.(*Selector)
				//if debug {
				//	log.Println("     - sel=", sel.Id)
				//}
				if sel.Depth != depth || sel.Field == nil || sel.Field.Mode == EmbedMode_None {
					continue
				}

				collectSelectorsFromEmbeddedField(sel, e)
				needToCheckDeepers = true
			}
			if !needToCheckDeepers {
				break
			}
		}

		return
	}

	// The following is for named types.

	d.collectSelectorsForNonInterfaceType(structType, smm, checkedTypes)

	// This line is nonsense.
	//namedType.counter = currentCounter // <=> t.counter = currentCounter
	//checkedTypes[namedType.index] = 0

	// map[string]*Selector
	selectorMap := smm.apply()
	defer smm.release(selectorMap)

	// ...
	t.AllMethods = make([]*Selector, 0, len(namedType.DirectSelectors)+len(structType.AllMethods))
	t.AllFields = make([]*Selector, 0, len(structType.AllFields))

	// Direct declared methods.
	for _, sel := range namedType.DirectSelectors {
		if _, exist := selectorMap[sel.Id]; exist {
			panic("should not")
		} else {
			selectorMap[sel.Id] = sel
			t.AllMethods = append(t.AllMethods, sel)
		}
	}

	// Promoted methods.
	for _, sel := range structType.AllMethods {
		if _, exist := selectorMap[sel.Id]; exist {
			// log.Println(sel.Id, "is shadowed")
		} else {
			selectorMap[sel.Id] = sel
			t.AllMethods = append(t.AllMethods, sel)
		}
	}

	// Fields, including promoteds.
	for _, sel := range structType.AllFields {
		if _, exist := selectorMap[sel.Id]; exist {
			// log.Println(sel.Id, "is shadowed")
		} else {
			selectorMap[sel.Id] = sel
			t.AllFields = append(t.AllFields, sel)
		}
	}
}

// Important for registerFunctionForInvolvedTypeNames and registerValueForItsTypeName.
func (d *CodeAnalyzer) sortPackagesByDepHeight() {
	var seen = make(map[string]struct{}, len(d.packageList))
	var calculatePackageDepHeight func(pkg *Package)
	calculatePackageDepHeight = func(pkg *Package) {
		if _, ok := seen[pkg.Path]; ok {
			return
		}
		seen[pkg.Path] = struct{}{}

		var max = int32(0)
		for _, dep := range pkg.Deps {
			calculatePackageDepHeight(dep)
			if dep.DepHeight > max {
				max = dep.DepHeight
			} else if dep.DepHeight == 0 {
				log.Println("sortPackagesByDepHeight: the dep.DepHeight is not calculated yet!")
			}
		}
		pkg.DepHeight = max + 1
	}

	for _, pkg := range d.packageList {
		calculatePackageDepHeight(pkg)
	}

	if d.builtinPkg == nil {
		panic("should not")
	}
	var old = d.builtinPkg.DepHeight
	d.builtinPkg.DepHeight = 0

	// The analyse order:
	sort.Slice(d.packageList, func(i, j int) bool {
		return d.packageList[i].DepHeight < d.packageList[j].DepHeight
	})

	d.builtinPkg.DepHeight = old

	for i, pkg := range d.packageList {
		pkg.Index = i
	}

	//for _, pkg := range d.packageList {
	//	log.Println(">>>>>>>>>>>>>>>>", pkg.DepHeight, pkg.Path)
	//}
}

// main packages have depth==0, their direct dependencies have depth==1, ...
func (d *CodeAnalyzer) calculatePackagesDepDepths() {
	var seen = make(map[string]struct{}, len(d.packageList))
	for _, pkg := range d.packageList {
		if pkg.PPkg.Name == "main" {
			pkg.DepDepth = 1
			seen[pkg.Path] = struct{}{}
		}
	}
	var hasMain = len(seen) > 0

	const MaxDepth int32 = 0x7fffffff

	var calculatePackageDepDepth func(pkg *Package)
	calculatePackageDepDepth = func(pkg *Package) {
		if _, ok := seen[pkg.Path]; ok {
			return
		}
		seen[pkg.Path] = struct{}{}

		var min = MaxDepth
		for _, dep := range pkg.DepedBys {
			calculatePackageDepDepth(dep)
			if dep.DepDepth < min {
				min = dep.DepDepth
			} else if dep.DepDepth == 0 {
				log.Println("sortPackagesDepDepth: the dep.DepDepth is not calculated yet!")
			}
		}
		if min == MaxDepth {
			pkg.DepDepth = 1
		} else {
			pkg.DepDepth = min + 1
		}
	}

	var max = int32(0)
	for _, pkg := range d.packageList {
		calculatePackageDepDepth(pkg)
		if pkg.DepDepth > max {
			max = pkg.DepDepth
		}
	}

	var hasShollowestUserPackages = false
	if !hasMain {
		for _, pkg := range d.packageList {
			if pkg.DepDepth == 1 && !d.IsStandardPackage(pkg) {
				hasShollowestUserPackages = true
			}
		}
	}
	if hasMain || hasShollowestUserPackages {
		max++
		d.builtinPkg.DepDepth++ // == 2
	}

	for _, pkg := range d.packageList {
		if pkg.DepDepth == 1 && d.IsStandardPackage(pkg) {
			pkg.DepDepth = max
		} else if hasMain {
			if pkg.PPkg.Name == "main" {
				pkg.DepDepth--
			}
		} else if hasShollowestUserPackages {
			if !d.IsStandardPackage(pkg) {
				pkg.DepDepth--
			}
		} else {
			pkg.DepDepth--
		}
	}

	//pkgs := append(d.packageList[:0:0], d.packageList...)
	//sort.Slice(pkgs, func(i, j int) bool {
	//	if di, dj := pkgs[i].DepDepth, pkgs[j].DepDepth; di == dj {
	//		if x, y := d.IsStandardPackage(pkgs[i]), d.IsStandardPackage(pkgs[j]); x == y {
	//			return pkgs[i].Path < pkgs[j].Path
	//		} else {
	//			return y
	//		}
	//	} else {
	//		return di < dj
	//	}
	//})
	//for _, pkg := range pkgs {
	//	log.Println(">>>>>>>>>>>>>>>>", pkg.Index, pkg.DepDepth, pkg.Path)
	//}
}

func (d *CodeAnalyzer) analyzePackage_CollectDeclarations(pkg *Package) {
	if pkg.PackageAnalyzeResult != nil {
		panic(pkg.Path + " already analyzed")
	}

	//log.Println("[analyzing]", pkg.Path, pkg.PPkg.Name)

	pkg.PackageAnalyzeResult = NewPackageAnalyzeResult()
	defer func() {
		pkg.BuildResourceLookupTable()
	}()

	registerTypeName := func(tn *TypeName) {
		pkg.PackageAnalyzeResult.AllTypeNames = append(pkg.PackageAnalyzeResult.AllTypeNames, tn)
		d.RegisterTypeName(tn)
	}

	registerVariable := func(v *Variable) {
		pkg.PackageAnalyzeResult.AllVariables = append(pkg.PackageAnalyzeResult.AllVariables, v)
		d.RegisterType(v.TType())
	}

	registerConstant := func(c *Constant) {
		pkg.PackageAnalyzeResult.AllConstants = append(pkg.PackageAnalyzeResult.AllConstants, c)
	}

	registerImport := func(imp *Import) {
		pkg.PackageAnalyzeResult.AllImports = append(pkg.PackageAnalyzeResult.AllImports, imp)
	}

	registerFunction := func(f *Function) {
		pkg.PackageAnalyzeResult.AllFunctions = append(pkg.PackageAnalyzeResult.AllFunctions, f)
		//d.RegisterFunction(f)
		// function stats are moved to below
	}

	var isBuiltinPkg = pkg == d.builtinPkg // pkg.Path == "builtin"
	var isUnsafePkg = pkg.Path == "unsafe"
	//var isBuildinOrUnsafe = isBuiltinPkg || isUnsafePkg

	// ToDo: use info.TypeOf, info.ObjectOf

	var locOfPkg = 0
	defer func() {
		d.stat_OnPackageCodeLineCount(locOfPkg, pkg)
	}()

	//for i, file := range pkg.PPkg.Syntax {
	for i, fileInfo := range pkg.SourceFiles {
		file := fileInfo.AstFile
		if file == nil {
			continue
		}
		//d.stats.AstFiles++
		//d.stats.Imports += int32(len(file.Imports))
		//incSliceStat(d.stats.FilesByImportCount[:], len(file.Imports))
		//d.stats.CodeLinesWithBlankLines += int32(pkg.PPkg.Fset.PositionFor(pkg.PPkg.Syntax[i].End(), false).Line)
		loc := pkg.PPkg.Fset.PositionFor(file.End(), false).Line
		d.stat_OnNewAstFile(len(file.Imports), loc, filepath.Base(pkg.PPkg.CompiledGoFiles[i]), pkg)
		_ = filepath.Base
		locOfPkg += loc

		//if len(file.Imports) == 0 {
		//	log.Println("----", pkg.PPkg.Fset.PositionFor(file.Pos(), false))
		//}

		//ast.Inspect(file, func(n ast.Node) bool {
		//	log.Printf("%T\n", n)
		//	return true
		//})

		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				if fd.Name.Name == "_" {
					continue
				}

				//log.Printf("func %s", fd.Name.Name)
				//log.Printf("(%s) %s (%s) (%s)", fd.Recv, fd.Name.Name, fd.Type.Params, fd.Type.Results)

				// It looks the function declared in "builtin" are types.Func, instead of types.Builtin.
				// But the functions declared in "unsafe" are types.Builtin.

				var f *Function

				obj := pkg.PPkg.TypesInfo.Defs[fd.Name]
				switch funcObj := obj.(type) {
				default:
					panic(pkg.Path + "." + fd.Name.Name + " not a types.Func or types.Builtin, but " + fmt.Sprintf("%T", funcObj))
				case *types.Func:
					f = &Function{
						Func: funcObj,

						Pkg:     pkg,
						AstDecl: fd,
					}
					//log.Println("    ", funcObj.Type())
				case *types.Builtin:
					// unsafe ones.
					// ToDo: maybe it is good to manually create a types.Func for each the builtin object.

					f = &Function{
						Builtin: funcObj,

						Pkg:     pkg,
						AstDecl: fd,
					}
					//log.Println("    ", funcObj.Type())
				}
				registerFunction(f)
			} else if gd, ok := decl.(*ast.GenDecl); ok {
				switch gd.Tok {
				case token.TYPE:
					for _, spec := range gd.Specs {
						typeSpec := spec.(*ast.TypeSpec)
						if typeSpec.Name.Name == "_" {
							continue
						}

						obj := pkg.PPkg.TypesInfo.Defs[typeSpec.Name]
						typeObj, ok := obj.(*types.TypeName)
						if !ok {
							//log.Println(pkg.PPkg.Fset.PositionFor(typeSpec.Pos(), false))
							//log.Println(pkg.PPkg.TypesInfo.Defs)
							panic(fmt.Sprintf("not a types.TypeName: %[1]v, %[1]T. Spec: %v", obj, typeSpec.Name.Name))
						}

						tv := pkg.PPkg.TypesInfo.Types[typeSpec.Type]
						if !tv.IsType() {
							if pkg.Path != "unsafe" {
								panic(typeSpec.Name.Name + ": not type")
							}

							// Now, unsafe AST expressions are the only ast.Expr(s)
							// which are allowed to not associate with a TypeAndValue.
							// For unsafe, although tv.IsType() == false, tv.Type is valid.
							// See fillUnsafePackage for details.
							if tv.Type == nil {
								panic(typeSpec.Name.Name + ": tv.Type is nil")
							}
						}

						var srcTypeExpr = tv.Type
						srcTypeInfo := d.RegisterType(srcTypeExpr)

						var objName = typeObj.Name()

						// Exported names, such as Type and Type1 are fake types.
						if isBuiltinPkg {
							if token.IsExported(objName) {
								continue
							}

							//>> 1.18
							//if objName == "any" {
							//	d.blankInterface = srcTypeInfo
							//	// ToDo: maybe should be assigned with
							//	//       d.RegisterType(types.Universe.Lookup("any").(*types.TypeName).Type()).
							//	//       And move this if-block before the loop.
							//}
							//<<

							var ok bool
							// It looks the parsed one are not the internal one.
							//fmt.Println(typeObj == types.Universe.Lookup(objName)) // false
							// Replace it with the internal one.
							typeObj, ok = types.Universe.Lookup(objName).(*types.TypeName)
							if !ok {
								panic("builtin " + objName + " not found")
							}

							//log.Println(srcType, srcType.Underlying(), srcType == srcType.Underlying()) // true

							//srcType = typeObj.Type().Underlying() // why underlying here? error and its underlying is different.
							//log.Println(typeObj.Type(), srcType, typeObj.Type() == srcType) // true
							//srcType = typeObj.Type()

							//log.Println("######", typeObj.Type(), "######", srcType, "####", types.Identical(typeObj.Type(), srcType))

							// It looks the below twos are not equal, though
							// types.Idenfical(them) returns true. So, typeObj.Type()
							// and srcType are both internal ByteType, but not Uint8Type.
							// Sometimes, this might matter.
							//
							// ByteType:  types.Universe.Lookup("byte").(*types.TypeName).Type()
							// Uint8Type: types.Universe.Lookup("uint8").(*types.TypeName).Type()
							//
							// The type of a custom aliase == the type it denotes. So this is an inconsistency.
							// But it is not a bug. We should use "types.Identical" to compare types.Type values.
							//
							// // true true
							//log.Println("==================",
							//	d.RegisterType(types.Universe.Lookup("byte").(*types.TypeName).Type()) ==
							//		d.RegisterType(types.Universe.Lookup("uint8").(*types.TypeName).Type()),
							//	types.Identical(types.Universe.Lookup("byte").(*types.TypeName).Type(),
							//		types.Universe.Lookup("uint8").(*types.TypeName).Type(),
							//	),
							//)
						}

						//if "ArbitraryType" == objName {
						//	fmt.Println(">>>>>>>>>>>>>>>>>", objName, srcType, typeObj.Type())
						//}

						//if typeObj.Type() == nil {
						//	fmt.Println("===================", objName, srcType)
						//}

						newTypeInfo := d.RegisterType(typeObj.Type())

						if isBuiltinPkg {
							//defer func(t *TypeInfo) {
							//	// Not much meaningful, just simplify code.
							//	t.attributes = srcTypeInfo.attributes
							//	t.TypeName = srcTypeInfo.TypeName
							//}(srcTypeInfo)

							srcTypeInfo = newTypeInfo
						}

						//if isBuiltinPkg && !token.IsExported(objName) {
						//log.Println(typeSpec.Name.Name, srcTypeInfo == newTypeInfo)
						//}

						tn := &TypeName{
							TypeName: typeObj,

							Pkg:     pkg,
							AstDecl: gd,
							AstSpec: typeSpec,
						}
						if typeObj.IsAlias() {
							//if isBuiltinPkg {
							// byte != uint8
							// rune != int32
							//} else {
							if srcTypeInfo != newTypeInfo {
								panic(fmt.Sprintf("srcTypeInfo != newTypeInfo, %v, %v", srcTypeInfo, newTypeInfo))
							}
							//if !types.Identical(srcTypeInfo.TT, newTypeInfo.TT) {
							//	panic(fmt.Sprintf("srcTypeInfo != newTypeInfo, %v, %v", srcTypeInfo.TT, newTypeInfo.TT))
							//}
							//}

							//tn.Alias = &TypeAlias{
							//	Denoting: srcTypeInfo,
							//	//TypeName: tn,
							//}
							tn.Denoting = srcTypeInfo
							tn.attributes |= Alias
							srcTypeInfo.Aliases = append(srcTypeInfo.Aliases, tn)

							if isBuiltinPkg || isUnsafePkg {
								//tn.Alias.attributes |= Builtin
								tn.attributes |= Builtin
							}
						} else {
							//tn.Named = newTypeInfo
							tn.Denoting = newTypeInfo
							//newTypeInfo.Origin = newTypeInfo
							newTypeInfo.TypeName = tn
							if isBuiltinPkg || isUnsafePkg {
								//tn.Named.attributes |= Builtin
								newTypeInfo.attributes |= Builtin
							}
						}

						//>> 1.18
						tn.Source.Expr = typeSpec.Type
						tn.Source.Type = srcTypeInfo
						tn.TypeParams = d.astFieldListToTypeArgs(pkg, astTypeSpecTypeParams(typeSpec))
						//tn.typeArgs = append(tn.typeArgs, tn.TypeParams...)
						//<<

						registerTypeName(tn)
					}
				case token.VAR:
					for _, spec := range gd.Specs {
						valueSpec := spec.(*ast.ValueSpec)
						//log.Println("var", valueSpec.Names, valueSpec.Type, valueSpec.Values)

						for _, name := range valueSpec.Names {
							if name.Name == "_" {
								continue
							}

							obj := pkg.PPkg.TypesInfo.Defs[name]
							varObj, ok := obj.(*types.Var)
							if !ok {
								panic("not a types.Var")
							}

							v := &Variable{
								Var: varObj,

								Pkg:     pkg,
								AstDecl: gd,
								AstSpec: valueSpec,
							}

							registerVariable(v)
						}
					}
				case token.CONST:
					for _, spec := range gd.Specs {
						valueSpec := spec.(*ast.ValueSpec)
						//log.Println("const", valueSpec.Names, valueSpec.Type, valueSpec.Values)

						for _, name := range valueSpec.Names {
							if name.Name == "_" {
								continue
							}

							obj := pkg.PPkg.TypesInfo.Defs[name]
							constObj, ok := obj.(*types.Const)
							if !ok {
								panic("not a types.Const")
							}

							c := &Constant{
								Const: constObj,

								Pkg:     pkg,
								AstDecl: gd,
								AstSpec: valueSpec,
							}

							registerConstant(c)
						}
					}
				case token.IMPORT:
					// ToDo: importSpec.Name might be nil
					for _, spec := range gd.Specs {
						var obj types.Object
						importSpec := spec.(*ast.ImportSpec)
						if importSpec.Name != nil {
							//log.Println("import 1", importSpec.Name.Name, importSpec.Path.Value)
							obj = pkg.PPkg.TypesInfo.Defs[importSpec.Name]
						} else {
							//log.Println("import 2", importSpec.Path.Value)
							obj = pkg.PPkg.TypesInfo.Implicits[importSpec]
						}
						//log.Println(obj)

						pkgObj, ok := obj.(*types.PkgName)
						if !ok {
							//log.Println(pkg.PPkg.Fset.PositionFor(importSpec.Pos(), false))
							//log.Println(pkg.PPkg.TypesInfo.Implicits)
							panic(fmt.Sprintf("not a types.PkgName: %[1]v, %[1]T. Spec: %v, %v", obj, importSpec.Name, importSpec.Path.Value))
						}

						imp := &Import{
							PkgName: pkgObj,

							Pkg:     pkg,
							AstDecl: gd,
							AstSpec: importSpec,
						}
						registerImport(imp)
					}
				}
			}
		}
	}
}

func (d *CodeAnalyzer) analyzePackage_CollectMoreStatistics(pkg *Package) {
	if pkg.PackageAnalyzeResult == nil {
		panic(pkg.Path + " is not analyzed yet")
	}
	var isBuiltinPkg = pkg == d.builtinPkg // pkg.Path == "builtin"

	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		d.stats.roughTypeNameCount++

		if isBuiltinPkg != token.IsExported(tn.Name()) {
			//incSliceStat(d.stats.ExportedIdentifiersByLength[:], len(tn.Name()))
			//d.stats.ExportedIdentifersSumLength += int32(len(tn.Name()))
			//d.stats.ExportedIdentifers++
			d.stat_OnNewExportedIdentifer(len(tn.Name()), tn)

			denoting := tn.Denoting
			kind := denoting.Kind()
			d.stats.ExportedTypeNamesByKind[kind]++

			//if tn.Alias != nil {
			//	d.stats.ExportedTypeAliases++
			//	if t := tn.Alias.Denoting; t.TypeName != nil && t.TypeName.Exported() {
			//		continue // to avoid duplicated statistics
			//	}
			//}
			if tn.IsAlias() {
				d.stats.ExportedTypeAliases++
				if t := tn.Denoting; t.TypeName != nil && t.TypeName.Exported() {
					continue // to avoid duplicated statistics
				}
			}

			var numExportedMethods = 0
			for _, sel := range denoting.AllMethods {
				if token.IsExported(sel.Name()) {
					numExportedMethods++
				}
			}
			if kind == reflect.Interface {
				d.stats.roughExportedIdentifierCount += int32(numExportedMethods)
				//incSliceStat(d.stats.ExportedNamedInterfacesByMethodCount[:], len(denoting.AllMethods))
				//incSliceStat(d.stats.ExportedNamedInterfacesByExportedMethodCount[:], numExportedMethods)
				//d.stats.ExportedNamedInterfacesExportedMethods += int32(numExportedMethods)
				d.stat_OnNewExportedInterfaceTypeNames(len(denoting.AllMethods), numExportedMethods, tn)
				continue
			}
			//incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByMethodCount[:], len(denoting.AllMethods))
			//incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByExportedMethodCount[:], numExportedMethods)
			d.stat_OnNewExportedNonInterfaceTypeNames(len(denoting.AllMethods), numExportedMethods, tn)

			if numExportedMethods > 0 {
				d.stats.ExportedNamedNonInterfacesExportedMethods += int32(numExportedMethods)
				d.stats.roughExportedIdentifierCount += int32(numExportedMethods)
				d.stats.ExportedNamedNonInterfacesWithExportedMethods++
			}

			if kind == reflect.Struct {
				hasEmbeddeds, numExportedPromoteds := false, 0
				numExporteds, numExpliciteds, numExportedExpliciteds := 0, 0, 0
				for _, sel := range denoting.AllFields {
					if token.IsExported(sel.Name()) {
						numExporteds++
						if sel.Depth == 0 {
							numExportedExpliciteds++
						} else {
							numExportedPromoteds++
						}
					}
					if sel.Depth == 0 {
						//incSliceStat(d.stats.ExportedIdentifiersByLength[:], len(sel.Name()))
						//d.stats.ExportedIdentifersSumLength += int32(len(sel.Name()))
						//d.stats.ExportedIdentifers++
						d.stat_OnNewExportedIdentifer(len(sel.Name()), &struct {
							*TypeName
							*Selector
						}{tn, sel})

						numExpliciteds++
					} else {
						hasEmbeddeds = true
					}
				}
				ut := d.RegisterType(denoting.TT.Underlying())
				//if hasEmbeddeds {
				//	d.stats.ExportedNamedStructTypesWithPromotedFields++
				//}
				//if ut.EmbeddingFields > 0 {
				//	d.stats.ExportedNamedStructTypesWithEmbeddingFields++
				//}
				//incSliceStat(d.stats.ExportedNamedStructsByEmbeddingFieldCount[:], int(ut.EmbeddingFields))
				//
				//incSliceStat(d.stats.ExportedNamedStructsByExplicitFieldCount[:], numExpliciteds)
				//d.stats.ExportedNamedStructTypeExplicitFields += int32(numExpliciteds)
				//incSliceStat(d.stats.ExportedNamedStructsByExportedFieldCount[:], numExporteds)
				//d.stats.ExportedNamedStructTypeExportedFields += int32(numExporteds)
				//incSliceStat(d.stats.ExportedNamedStructsByExportedExplicitFieldCount[:], numExportedExpliciteds)
				//d.stats.ExportedNamedStructTypeExportedExplicitFields += int32(numExportedExpliciteds)
				//d.stats.roughExportedIdentifierCount += int32(numExportedExpliciteds)
				//
				//incSliceStat(d.stats.ExportedNamedStructsByExportedPromotedFieldCount[:], numExportedPromoteds)
				//
				//incSliceStat(d.stats.ExportedNamedStructsByFieldCount[:], len(denoting.AllFields))
				//d.stats.ExportedNamedStructTypeFields += int32(len(denoting.AllFields))
				d.stat_OnNewExportedStructTypeName(hasEmbeddeds, len(denoting.AllFields), int(ut.EmbeddingFields), numExpliciteds, numExporteds, numExportedExpliciteds, numExportedPromoteds, tn)
			}
		}
	}
}

func (d *CodeAnalyzer) collectMoreStatisticsFinal() {
	var sum = func(kinds ...reflect.Kind) (r int32) {
		for _, k := range kinds {
			r += d.stats.ExportedTypeNamesByKind[k]
		}
		return
	}
	d.stats.ExportedUnsignedTypeNames = sum(reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr)
	d.stats.ExportedIntergerTypeNames = d.stats.ExportedUnsignedTypeNames + sum(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)
	d.stats.ExportedNumericTypeNames = d.stats.ExportedIntergerTypeNames + sum(reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128)
	d.stats.ExportedBasicTypeNames = d.stats.ExportedNumericTypeNames + sum(reflect.Bool, reflect.String)
	d.stats.ExportedCompositeTypeNames = sum(reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct, reflect.UnsafePointer)
	d.stats.ExportedTypeNames = d.stats.ExportedCompositeTypeNames + d.stats.ExportedBasicTypeNames

	//d.stats.Packages = int32(len(d.packageList))
	//for _, pkg := range d.packageList {
	//	//if d.IsStandardPackage(pkg) {
	//	//	d.stats.StdPackages++
	//	//}
	//	//d.stats.FilesWithGenerateds += int32(len(pkg.SourceFiles))
	//	//d.stats.AllPackageDeps += int32(len(pkg.Deps))
	//	//incSliceStat(d.stats.PackagesByDeps[:], len(pkg.Deps))
	//	//d.stat_OnNewPackage(d.IsStandardPackage(pkg), len(pkg.SourceFiles), len(pkg.Deps), pkg.Path)
	//}

	d.stats.roughExportedIdentifierCount += d.stats.ExportedIdentifers
}

func (d *CodeAnalyzer) collectSomeRuntimeFunctionPositions() {
	// ...
	if runtimePkg := d.packageTable["runtime"]; runtimePkg != nil {
		fnames := []string{
			"selectgo",     // for select blocks (except one-case-plus-default ones)
			"selectnbsend", // one-case-plus-default select blocks
			"selectnbrecv", // select {case v = <-c:; default:}
			//"selectnbrecv2", // select {case v, ok = <-c:; default:} // removed since Go 1.17
			"chansend",  // c <- v
			"chanrecv1", // v = <- c
			"chanrecv2", // v, ok = <-c
			"makechan",
			"closechan",
			"gopanic",
			"gorecover",
			// gave up other built-in functions
		}
		d.runtimeFuncPositions = make(map[string]token.Position, 32)

		for _, f := range fnames {
			obj := runtimePkg.PPkg.Types.Scope().Lookup(f)
			if obj == nil {
				log.Printf("!!! runtime.%s is not found", f)
			}
			d.runtimeFuncPositions[f] = runtimePkg.PPkg.Fset.PositionFor(obj.Pos(), false)
		}
	}
}

//func (d *CodeAnalyzer) analyzePackage_ConfirmTypeSources(pkg *Package) {
//	var isBuiltin = pkg == d.builtinPkg // pkg.Path == "builtin"
//
//	//log.Println("[analyzing]", pkg.Path, pkg.PPkg.Name)
//	for _, file := range pkg.PPkg.Syntax {
//
//		//ast.Inspect(file, func(n ast.Node) bool {
//		//	log.Printf("%T\n", n)
//		//	return true
//		//})
//
//		for _, decl := range file.Decls {
//			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
//				for _, spec := range gd.Specs {
//					typeSpec := spec.(*ast.TypeSpec)
//
//					obj := pkg.PPkg.TypesInfo.Defs[typeSpec.Name]
//					typeObj := obj.(*types.TypeName)
//					if typeObj.Name() == "_" {
//						continue
//					}
//
//					if isBuiltin && token.IsExported(typeObj.Name()) {
//						continue
//					}
//
//					newTypeName := d.allTypeNameTable[d.Id2(typeObj.Pkg(), typeObj.Name())]
//					if newTypeName == nil {
//						panic("type name " + typeSpec.Name.Name + " not found: " + d.Id1(typeObj.Pkg(), typeObj.Name()))
//					}
//
//					tv := pkg.PPkg.TypesInfo.Types[typeSpec.Type]
//					srcTypeInfo := d.RegisterType(tv.Type)
//					_ = srcTypeInfo
//
//					/*
//					var findSource func(ast.Expr, bool)
//					findSource = func(srcNode ast.Expr, starSource bool) {
//						var source *TypeSource
//						if starSource {
//							if newTypeName.StarSource == nil {
//								newTypeName.StarSource = &TypeSource{}
//							}
//							source = newTypeName.StarSource
//						} else {
//							source = &newTypeName.Source
//						}
//
//						//var originType *TypeName
//						//var typeArgs []*TypeInfo
//
//						var sttNode *ast.StructType
//						var ittNode *ast.InterfaceType
//
//						switch expr := srcNode.(type) {
//						//>> 1.18, ToDo ToDo2 InstantiatedType
//						// handle Index and IndexList?
//						// It seems not needed.
//
//						// Although an instantiated type is a named type,
//						// the source type will be recored in source.UnnamedType field.
//						//case *astIndexExpr:
//						//	findSource(expr.X, starSource)
//						//	if source.TypeName == nil {
//						//		panic("should not"
//						//	}
//						//	// source.TypeName.Type() is the generic type, not the instantiated type
//						//
//						//	originType = source.TypeName
//						//	typeArgs =
//						//	// source.TypeName will be replaced below
//						//case *astIndexListExpr:
//						//	findSource(expr.X, starSource)
//						//	if source.TypeName == nil {
//						//		panic("should not"
//						//	}
//						//	// source.TypeName.Type() is the generic type, not the instantiated type
//						//
//						//	originType = source.TypeName
//						//	typeArgs =
//						//	// source.TypeName will be replaced below
//						//<<
//
//						case *ast.Ident:
//
//							//log.Println("???", d.Id(pkg.PPkg.Types, expr.Name))
//
//							//log.Println("   ", pkg.PPkg.TypesInfo.ObjectOf(expr))
//
//							srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr)
//							if srcObj == nil {
//								if pkg.Path != "unsafe" {
//									panic("srcObj is nil but package is not unsafe")
//								}
//								return
//							}
//							srcTypeObj := srcObj.(*types.TypeName)
//
//							//log.Println("   srcTypeObj.Pkg() =", srcTypeObj.Pkg())
//							// if srcType is a built type, srcTypeObj.Pkg() == nil
//
//							tn := d.allTypeNameTable[d.Id2(srcTypeObj.Pkg(), expr.Name)]
//							if tn == nil {
//								panic("type name " + expr.Name + " not found")
//							}
//							source.TypeName = tn
//
//							//log.Println(starSource, "ident,", pkg.Path+"."+typeSpec.Name.Name, "source is:", tn.Pkg.Path+"."+expr.Name)
//
//							return
//						case *ast.SelectorExpr:
//							//log.Println("selector,", pkg.Path+"."+typeSpec.Name.Name, "source is:")
//							srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr.X.(*ast.Ident))
//							srcPkg := srcObj.(*types.PkgName)
//
//							tn := d.allTypeNameTable[d.Id2(srcPkg.Imported(), expr.Sel.Name)]
//							if tn == nil {
//								panic("type name " + expr.Sel.Name + " not found")
//							}
//							source.TypeName = tn
//
//							//log.Println(starSource, "selector,", pkg.Path+"."+typeSpec.Name.Name, "source is:", tn.Pkg.Path+"."+expr.Sel.Name)
//							return
//						case *ast.ParenExpr:
//							//log.Println("paren,", pkg.Path+"."+typeSpec.Name.Name, "source is:")
//							findSource(expr.X, false)
//							return
//						case *ast.StarExpr:
//							if !starSource {
//								//log.Println("star,", pkg.Path+"."+typeSpec.Name.Name, "source is:")
//								findSource(expr.X, true)
//								return
//							}
//						case *ast.StructType:
//							sttNode = expr
//						case *ast.InterfaceType:
//							ittNode = expr
//						}
//
//						// ToDo: don't use the std go/types and go/pacakges packages.
//						//       Now, uint8 and byte are treat as two types by go/types.
//						//       Write a custom one tailored for docs and code analyzing.
//						//       Run "go mod tidy" before running golds using the custom packages
//						//       to ensure all modules are cached locally.
//
//						tv := pkg.PPkg.TypesInfo.Types[srcNode]
//						srcTypeInfo := d.RegisterType(tv.Type)
//						source.UnnamedType = srcTypeInfo
//						//log.Println(starSource, "default,", pkg.Path+"."+typeSpec.Name.Name, "source is:", tv.Type)
//
//						if sttNode != nil {
//							d.registerDirectFields(srcTypeInfo, sttNode, pkg)
//						} else if ittNode != nil {
//							if isBuiltin {
//								if typeSpec.Name.Name == "error" {
//									//	//errorTN, _ := types.Universe.Lookup("error").(*types.TypeName)
//									//	//errotUT := d.RegisterType(errorTN.Type().Underlying())
//									//	//d.registerExplicitlySpecifiedMethods(errotUT, ittNode, pkg)
//									//
//									//	//log.Println("=============== old:", srcTypeInfo.index)
//									//	// This one is the type shown in the builtin.go source code,
//									//	// not the one in type.Universal package. This one is only for docs purpose.
//									//	// ToDo: use custom builtin page, remove the special handling.
//									//	d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
//									//
//									//	// ToDo: load builtin.error != universal.error
//									//	srcTypeInfo = d.RegisterType(newTypeName.Named.TT.Underlying())
//									//
//									//	//log.Println("===============", errotUT.index, srcTypeInfo.index)
//
//									d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
//									srcTypeInfo = d.RegisterType(types.Universe.Lookup("error").(*types.TypeName).Type().Underlying())
//								} else if typeSpec.Name.Name == "comparable" {
//									d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
//									srcTypeInfo = d.RegisterType(types.Universe.Lookup("comparable").(*types.TypeName).Type().Underlying())
//								} else if typeSpec.Name.Name == "ordered" { // just a pure guess later versions will introduce this one
//									d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
//									srcTypeInfo = d.RegisterType(types.Universe.Lookup("ordered").(*types.TypeName).Type().Underlying())
//								}
//							}
//							d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
//						}
//					}
//					findSource(typeSpec.Type, false)
//					*/
//				}
//			}
//		}
//	}
//
//	return
//}

func (d *CodeAnalyzer) astFieldListToTypeArgs(pkg *Package, fieldList *ast.FieldList) []TypeExpr {
	if fieldList == nil {
		return nil
	}

	var n = fieldList.NumFields()
	if n == 0 {
		return nil
	}
	var args = make([]TypeExpr, 0, n)
	for _, field := range fieldList.List {
		for _, name := range field.Names {
			args = append(args, TypeExpr{
				Expr: name,
				Type: d.RegisterType(pkg.PPkg.TypesInfo.TypeOf(name)),
				Pkg:  pkg,
			})
		}
	}
	return args
}

func (d *CodeAnalyzer) astIndexExprToTypeArgs(pkg *Package, index *astIndexExpr) []TypeExpr {
	return []TypeExpr{
		{
			Expr: index.Index,
			Type: d.RegisterType(pkg.PPkg.TypesInfo.TypeOf(index.Index)),
			Pkg:  pkg,
		},
	}
}

func (d *CodeAnalyzer) astIndexListExprToTypeArgs(pkg *Package, list *astIndexListExpr) []TypeExpr {
	var args = make([]TypeExpr, 0, len(list.Indices))
	for _, index := range list.Indices {
		args = append(args, TypeExpr{
			Expr: index,
			Type: d.RegisterType(pkg.PPkg.TypesInfo.TypeOf(index)),
			Pkg:  pkg,
		})
	}
	return args
}

func (d *CodeAnalyzer) confirmInstantiatedInfoForTypeConstraints(typeParams *ast.FieldList, pkg *Package) {
	if typeParams == nil {
		return
	}
	if typeParams.NumFields() == 0 {
		return
	}
	for _, field := range typeParams.List {
		d.lookForAndRegisterUnnamedInterfaceAndStructTypes(field.Type, pkg)
	}
}

func (d *CodeAnalyzer) collectDirectSelectorsForSourceType(source TypeExpr, pkg *Package, isBuiltinPkg bool) {
	var collect func(ast.Expr, bool)
	collect = func(srcNode ast.Expr, starSource bool) {
		//var originType *TypeName
		//var typeArgs []*TypeInfo

		var typeArgs []TypeExpr
		var sttNode *ast.StructType
		var ittNode *ast.InterfaceType

		switch expr := srcNode.(type) {
		case *astIndexExpr:
			typeArgs = d.astIndexExprToTypeArgs(pkg, expr)
			// ToDo: it looks no needs to collect expr.Index
		case *astIndexListExpr:
			typeArgs = d.astIndexListExprToTypeArgs(pkg, expr)
			// ToDo: it looks no needs to collect expr.Slices
		case *ast.ParenExpr:
			//log.Println("paren,", pkg.Path+"."+typeSpec.Name.Name, "source is:")
			collect(expr.X, false)
			return
		case *ast.StarExpr:
			if !starSource {
				//log.Println("star,", pkg.Path+"."+typeSpec.Name.Name, "source is:")
				collect(expr.X, true)
				return
			}
		case *ast.StructType:
			sttNode = expr
		case *ast.InterfaceType:
			ittNode = expr
		}

		tt := pkg.PPkg.TypesInfo.TypeOf(srcNode)
		srcTypeInfo := d.RegisterType(tt)
		if ntt, ok := tt.(*types.Named); ok {
			if ott := originType(ntt); ott != ntt {
				origin := d.RegisterType(ott)
				if origin.TypeName == nil {
					panic("should not at this time point")
				}
				srcTypeInfo.TypeName = origin.TypeName
			}
		}

		//log.Println(starSource, "default,", pkg.Path+"."+typeSpec.Name.Name, "source is:", tv.Type)

		if typeArgs != nil {
			d.registerInstantiatedType(srcTypeInfo, typeArgs)
		} else if sttNode != nil {
			d.registerDirectFields(srcTypeInfo, sttNode, pkg)
		} else if ittNode != nil {
			// ToDo: why need two calls here? Forget the reason. But really need now,

			d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
			if isBuiltinPkg {
				// for "error", "comparable", ... etc.
				d.registerExplicitlySpecifiedMethods(source.Type.Underlying, ittNode, pkg)
			}
		}
	}
	collect(source.Expr, false)
}

func (d *CodeAnalyzer) analyzePackage_CollectDirectSelectors(pkg *Package) {

	var isBuiltinPkg = pkg == d.builtinPkg // pkg.Path == "builtin"

	// Only collect direct fields and methods for unnamed (struct and interface) types
	// in this step.
	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		d.confirmInstantiatedInfoForTypeConstraints(astTypeSpecTypeParams(tn.AstSpec), pkg)
		d.collectDirectSelectorsForSourceType(tn.Source, pkg, isBuiltinPkg)
	}

	//  We must do the collection work after all types are collected.
	for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
		d.confirmInstantiatedInfoForTypeConstraints(astFuncTypeTypeParams(f.AstDecl.Type), pkg)
		isMethod := d.tryToRegisterExplicitlyDeclaredMethod(f)
		if isMethod {
			if f.AstDecl.Recv == nil {
				panic("should not")
			}
			if len(f.AstDecl.Recv.List) != 1 {
				panic("should not")
			}

			// Unnecessary to do so.
			// d.confirmInstantiatedInfoForTypeConstraints(f.AstDecl.Recv, pkg)

			field := f.AstDecl.Recv.List[0]

			// NOTE: field.Type might denote a type alias denoting *T.

			tt := pkg.PPkg.TypesInfo.TypeOf(field.Type)
			if tt == nil {
				panic("should not")
			}

			var baseTT *types.Named
			switch tt := tt.(type) {
			case *types.Named:
				baseTT = tt
			case *types.Pointer:
				f.attributes |= StarReceiver
				var ok bool
				baseTT, ok = tt.Elem().(*types.Named)
				if !ok {
					panic("should not")
				}
			}

			ttn := baseTT.Obj()
			f.receiverTypeName = d.allTypeNameTable[d.Id2b(pkg, ttn.Name())]
			if f.receiverTypeName == nil {
				panic("should not")
			}
			if !ttn.Exported() {
				// ToDo: maybe, should not continue here.
				//       Mehtods of unexported types should be listed in the
				//       "unexported" sections of "AsInputOf" and "AsOutputOf",
				//       whether or not the methods are exported.
				continue
			}
		}

		//if f.Exported() {
		//	d.registerFunctionForInvolvedTypeNames(f)
		//}
		numParams, numResults, lastResultIsError := d.registerFunctionForInvolvedTypeNames(f)

		// ToDo: sometimes unexported ones are also needed to read code.
		if f.Exported() {
			if f.IsMethod() {
				d.stats.ExportedMethods++
				//incSliceStat(d.stats.MethodsByParameterCount[:], numParams)
				//incSliceStat(d.stats.FunctionsByResultCount[:], numResults)
			} else {
				d.stats.ExportedFunctions++
				//incSliceStat(d.stats.FunctionsByParameterCount[:], numParams)
				//incSliceStat(d.stats.MethodsByResultCount[:], numResults)
			}
			if lastResultIsError {
				d.stats.ExportedFunctionWithLastErrorResult++
			}
			//incSliceStat(d.stats.ExportedIdentifiersByLength[:], len(f.Name()))
			//d.stats.ExportedIdentifersSumLength += int32(len(f.Name()))
			//d.stats.ExportedIdentifers++
			d.stat_OnNewExportedIdentifer(len(f.Name()), f)

			//d.stats.ExportedFunctionParameters += int32(numParams)
			//d.stats.ExportedFunctionResults += int32(numResults)
			//incSliceStat(d.stats.ExportedFunctionsByParameterCount[:], numParams)
			//incSliceStat(d.stats.ExportedFunctionsByResultCount[:], numResults)
			d.stat_OnNewExportedFunction(numParams, numResults, f)
		}
	}
	//for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
	//	// moved to analyzePackage_CollectMoreStatistics
	//}
	if isBuiltinPkg {
		//var tnRune, tnInt32, tnByte, tnUint8 *TypeName
		//for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		//	switch tn.Name() {
		//	case "rune":
		//		tnRune = tn
		//	case "int32":
		//		tnInt32 = tn
		//	case "byte":
		//		tnByte = tn
		//	case "uint8":
		//		tnUint8 = tn
		//	}
		//}
		//if tnRune != nil && tnInt32 != nil { // always true
		//	log.Println("=============== tnInt32.Named:", tnInt32.Named)
		//	tnRune.Named = tnInt32.Named
		//}
		//if tnByte != nil && tnUint8 != nil { // always true
		//	log.Println("=============== tnUint8.Named:", tnUint8.Named)
		//	tnByte.Named = tnUint8.Named
		//}
		return
	}

	//if !d.IsStandardPackage(pkg) {
	//	d.debug = true
	//	defer func() { d.debug = false }()
	//}

	for _, v := range pkg.PackageAnalyzeResult.AllVariables {
		//>> ToDo: 1.18 consider instantiated TypeNames
		//<<
		d.registerValueForItsTypeName(v, v.AstSpec)
		//if d.debug {
		//	d.debug = false
		//	log.Println(v.Position())
		//}
		if v.Exported() {
			d.stats.ExportedVariables++

			//incSliceStat(d.stats.ExportedIdentifiersByLength[:], len(v.Name()))
			//d.stats.ExportedIdentifersSumLength += int32(len(v.Name()))
			//d.stats.ExportedIdentifers++
			d.stat_OnNewExportedIdentifer(len(v.Name()), v)

			kind := Kind(v.TType())
			d.stats.ExportedVariablesByTypeKind[kind]++
		}

		// ToDo: I forgot why to register types of variables. Maybe, not needed?
		//        Check it sometime.
		//
		// ToDo: 1.18 InstantiatedType
		//
		// ToDo: for a variable specification without the Type protion (then
		// must have an initial value portion), and if its type is a unamed
		// struct or interface type (or contains such types), then this type
		// has no chances to get its .DirectSelectors fulfilled.
		//
		// An example: var reserved = new(struct{ types.Type })
		//
		// Currently, this is not bad, for we don't need to calculate method set
		// for such types and the struct and interface types contained in it.
		if v.AstSpec.Type != nil {
			d.lookForAndRegisterUnnamedInterfaceAndStructTypes(v.AstSpec.Type, v.Pkg)
		}
	}

	//>> ToDo: 1.18
	// Types of constants may also be instantiated types.
	//<<
	for _, c := range pkg.PackageAnalyzeResult.AllConstants {
		d.registerValueForItsTypeName(c, c.AstSpec)
		if c.Exported() {
			d.stats.ExportedConstants++

			//incSliceStat(d.stats.ExportedIdentifiersByLength[:], len(c.Name()))
			//d.stats.ExportedIdentifersSumLength += int32(len(c.Name()))
			//d.stats.ExportedIdentifers++
			d.stat_OnNewExportedIdentifer(len(c.Name()), c)

			kind := Kind(c.TType())
			d.stats.ExportedConstantsByTypeKind[kind]++
		}
	}
}

func (d *CodeAnalyzer) comfirmDirectSelectorsForInstantiatedTypes() {
	var l = d.instantiatedTypes
	if l == nil {
		return
	}

	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]
		t.counter = 0
	}

	// An interface may has a method and an embedding field sharing the same name.
	var fieldMap = make(map[string]*TypeInfo, 256)
	var methodMap = make(map[string]*TypeInfo, 256)
	var currentCounter uint32
	var i = 0
	for e := l.Front(); e != nil; e = l.Front() {
		//log.Println("============================================ #", i)
		i++

		t := e.Value.(*TypeInfo)
		l.Remove(e)
		currentCounter++
		d.comfirmDirectSelectorsForInstantiatedType(t, currentCounter, fieldMap, methodMap)
	}

	//log.Println("======== numSeenInstantiatedTypes:", d.numSeenInstantiatedTypes)
}
