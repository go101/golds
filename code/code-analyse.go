package code

import (
	"container/list"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"sort"
	"time"

	"golang.org/x/tools/go/types/typeutil"

	"go101.org/gold/util"
)

func (d *CodeAnalyzer) AnalyzePackages(onSubTaskDone func(int, time.Duration, ...int32)) {
	//log.Println("[analyze packages ...]")

	var stopWatch = util.NewStopWatch()

	var logProgress = func(task int, args ...int32) {
		onSubTaskDone(task, stopWatch.Duration(), args...)
	}

	d.confirmPackageModules()

	stopWatch.Duration()

	d.sortPackagesByDependencies()

	logProgress(SubTask_SortPackagesByDependencies)

	for _, pkg := range d.packageList {
		d.analyzePackage_CollectDeclarations(pkg)
	}

	logProgress(SubTask_CollectDeclarations)

	d.analyzePackage_CollectSomeRuntimeFunctionPositions()

	logProgress(SubTask_CollectRuntimeFunctionPositions)

	//log.Println("=== recorded type count:", len(d.allTypeInfos))

	//log.Println("[analyze packages 2...]")

	for _, pkg := range d.packageList {
		d.analyzePackage_FindTypeSources(pkg)
	}

	logProgress(SubTask_FindTypeSources)

	//log.Println("[analyze packages 4...]")

	d.analyzePackages_CollectSelectors()

	logProgress(SubTask_CollectSelectors)

	// ToDo: it might be best to not use the NewMethodSet fucntion in std.
	//       Same for NewFieldSet

	//log.Println("[analyze packages 4...]")

	d.forbidRegisterTypes = true

	//methodCache := d.analyzePackages_FindImplementations_Old()
	d.analyzePackages_FindImplementations()
	methodCache := &typeutil.MethodSetCache{}

	d.forbidRegisterTypes = false

	logProgress(SubTask_FindImplementations)

	d.CollectSourceFiles()

	logProgress(SubTask_CollectSourceFiles)

	for _, pkg := range d.packageList {
		d.analyzePackage_CollectMoreStatistics(pkg)
	}
	d.analyzePackage_CollectMoreStatisticsFinal()

	logProgress(SubTask_MakeStatistics)

	// ...

	// This is a bug in std types.MethodSet implementation (Go SDK 1.14-)
	// https://github.com/golang/go/issues/37081
	//d.analyzePackages_CheckCollectSelectors(methodCache)
	_ = methodCache
	//logProgress("Check collect selectors", nil)

	// log.Println("[analyze packages done]")
}

func sortPackagesByDepLevels(pkgs []*Package) {
	var seen = make(map[string]struct{}, len(pkgs))
	var calculatePackageDepLevel func(pkg *Package)
	calculatePackageDepLevel = func(pkg *Package) {
		if _, ok := seen[pkg.Path()]; ok {
			return
		}
		seen[pkg.Path()] = struct{}{}

		var max = 0
		for _, dep := range pkg.Deps {
			calculatePackageDepLevel(dep)
			if dep.DepLevel > max {
				max = dep.DepLevel
			} else if dep.DepLevel == 0 {
				log.Println("sortPackagesByDepLevels, calculatePackageDepLevel, the dep.DepLevel is not calculated yet!")
			}
		}
		pkg.DepLevel = max + 1
	}

	for _, pkg := range pkgs {
		calculatePackageDepLevel(pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].DepLevel < pkgs[j].DepLevel
	})
}

func (d *CodeAnalyzer) analyzePackages_FindImplementations() { // (resultMethodCache *typeutil.MethodSetCache) {

	//

	//2. search implementations
	//a. 1st pass: collect all interface method signatures. Each interface method signature maintains a type list.
	//   map[sigID][]*type
	//b. 2nd pass: for each method of every type, if it is an interface method, register the type to the interface method
	//c. 3rd pass: for each interface type, iterate its method, increase Type.counter (must be handled in a single thread)
	//d. sort the implementations by package distances to the interface type. The shorter common prefixm the longer two packages distance.

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
		methodIndexes []uint32
		underlieds    []*TypeInfo // including the underlying itself
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
		underlying := t.TT.Underlying()
		underlyingTypeInfo := d.RegisterType(underlying) // underlying must have been already registered
		t.Underlying = underlyingTypeInfo
		underlyingTypeInfo.Underlying = underlyingTypeInfo

		if i, ok := underlying.(*types.Interface); ok && i.NumMethods() > 0 {
			var uiInfo *UnderlyingInterfaceInfo
			info := interfaceUnderlyings.At(i)
			if interfaceUnderlyings.At(i) == nil {
				//interfaceUnderlyingTypes = append(interfaceUnderlyingTypes, underlyingTypeInfo)
				uiInfo = &UnderlyingInterfaceInfo{t: underlyingTypeInfo, underlieds: make([]*TypeInfo, 0, 3)}
				interfaceUnderlyings.Set(i, uiInfo)
				//log.Printf("!!! %T\n", uiInfo.t.TT)
			} else {
				uiInfo, _ = info.(*UnderlyingInterfaceInfo)
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
			funcSig, ok := sel.Method.Type.TT.(*types.Signature)
			if !ok {
				panic(fmt.Sprintf("not a types.Signature: %T", sel.Method.Type.TT))
			}
			pkgImportPath := ""
			if sel.Method.Pkg != nil {
				pkgImportPath = sel.Method.Pkg.Path()
			}

			sig := d.BuildMethodSignatureFromFunctionSignature(funcSig, sel.Method.Name, pkgImportPath)

			if d.lastTypeIndex > x {
				log.Println("       > ", uiInfo.t.TT)
				log.Println("             >> ", sel)
			}

			methodIndex, ok := allInterfaceMethods[sig]
			if ok {
				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], uiInfo.t.index)
			} else {
				methodIndex = lastMethodIndex
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
			funcSig, ok := sel.Method.Type.TT.(*types.Signature)
			if !ok {
				panic("not a types.Signature")
			}
			pkgImportPath := ""
			if sel.Method.Pkg != nil {
				pkgImportPath = sel.Method.Pkg.Path()
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

		// Register non-pointer ones firstly, then
		// register pointer ones whose bases have not been registered.
		d.resetTempTypeLookupTable()
		impBy := make([]*TypeInfo, 0, count)
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if _, ok := t.TT.(*types.Pointer); !ok {
					if itt, ok := t.TT.Underlying().(*types.Interface); ok {
						ittInfo := interfaceUnderlyings.At(itt).(*UnderlyingInterfaceInfo)
						for _, it := range ittInfo.underlieds {
							impBy = append(impBy, it)
							typeLookupTable[it.index] = struct{}{}
						}
					} else {
						impBy = append(impBy, t)
						typeLookupTable[typeIndex] = struct{}{}
					}
				}
			}
		}
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if ptt, ok := t.TT.(*types.Pointer); ok {
					bt := d.RegisterType(ptt.Elem()) // 333 a: here to check why new types are registered
					if _, reged := typeLookupTable[bt.index]; !reged {
						impBy = append(impBy, t)
					}
				}
			}
		}
		uiInfo.t.ImplementedBys = impBy

		//log.Println("111 @@@", uiInfo.t.TT, ", uiInfo.methodIndexes:", uiInfo.methodIndexes)
		//for _, impBy := range impBy {
		//	log.Println("     ", impBy.TT)
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
// The current implementaiton-finding algorithm uses TypeInfo.index as judge conidition.
// So the implementation is not ok for concurrency safe. To make it concurrentcy safe,
// we can sort each method2TypeIndexes slices, and copy the one for the first method,
// then get the overlapping for consequencing method slices.
// However, it looks the current implementation is fast enough.

func (d *CodeAnalyzer) analyzePackages_FindImplementations_Old() (resultMethodCache *typeutil.MethodSetCache) {
	//

	//2. search implementations
	//a. 1st pass: collect all interface method signatures. Each interface method signature maintains a type list.
	//   map[sigID][]*type
	//b. 2nd pass: for each method of every type, if it is an interface method, register the type to the interface method
	//c. 3rd pass: for each interface type, iterate its method, increase Type.counter (must be handled in a single thread)
	//d. sort the implementations by package distances to the interface type. The shorter common prefixm the longer two packages distance.

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
		methodIndexes []uint32
		underlieds    []*TypeInfo // including the underlying itself
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
		underlying := t.TT.Underlying()
		underlyingTypeInfo := d.RegisterType(underlying) // underlying must have been already registered
		t.Underlying = underlyingTypeInfo
		underlyingTypeInfo.Underlying = underlyingTypeInfo

		if i, ok := underlying.(*types.Interface); ok && i.NumMethods() > 0 {
			var uiInfo *UnderlyingInterfaceInfo
			info := interfaceUnderlyings.At(i)
			if interfaceUnderlyings.At(i) == nil {
				//interfaceUnderlyingTypes = append(interfaceUnderlyingTypes, underlyingTypeInfo)
				uiInfo = &UnderlyingInterfaceInfo{t: underlyingTypeInfo, underlieds: make([]*TypeInfo, 0, 3)}
				interfaceUnderlyings.Set(i, uiInfo)
				//log.Printf("!!! %T\n", uiInfo.t.TT)
			} else {
				uiInfo, _ = info.(*UnderlyingInterfaceInfo)
			}
			uiInfo.underlieds = append(uiInfo.underlieds, t)
		}
	}

	log.Println("number of underlying interfaces:", interfaceUnderlyings.Len())
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
	var cache typeutil.MethodSetCache
	resultMethodCache = &cache

	interfaceUnderlyings.Iterate(func(_ types.Type, info interface{}) {
		uiInfo := info.(*UnderlyingInterfaceInfo)
		//log.Printf("### %d %T\n", uiInfo.t.index, uiInfo.t.TT)
		methodSet := cache.MethodSet(uiInfo.t.TT)
		uiInfo.methodIndexes = make([]uint32, methodSet.Len())

		for i := methodSet.Len() - 1; i >= 0; i-- {
			sel := methodSet.At(i)
			funcObj, ok := sel.Obj().(*types.Func)
			if !ok {
				panic("not a types.Func")
			}
			x := d.lastTypeIndex
			sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure
			if d.lastTypeIndex > x {
				log.Println("       > ", uiInfo.t.TT)
				log.Println("             >> ", sel)
			}

			methodIndex, ok := allInterfaceMethods[sig]
			if ok {
				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], uiInfo.t.index)
			} else {
				methodIndex = lastMethodIndex
				lastMethodIndex++
				allInterfaceMethods[sig] = methodIndex

				typeIndexes := make([]uint32, 0, 8)
				typeIndexes = append(typeIndexes, uiInfo.t.index)

				//log.Printf("   $$$ %d %T\n", uiInfo.t.index, d.allTypeInfos[uiInfo.t.index].TT)

				// method2TypeIndexes[methodIndex] = typeIndexes
				method2TypeIndexes = append(method2TypeIndexes, typeIndexes)
			}
			uiInfo.methodIndexes[i] = methodIndex
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

		methodSet := cache.MethodSet(t.TT)
		//log.Println("222>>>", t.TT, methodSet.Len())
		for i := methodSet.Len() - 1; i >= 0; i-- {
			sel := methodSet.At(i)
			funcObj, ok := sel.Obj().(*types.Func)
			if !ok {
				panic("not a types.Func")
			}

			sig := d.BuildMethodSignatureFromFuncObject(funcObj) // will not produce new type registrations for sure

			methodIndex, ok := allInterfaceMethods[sig]
			//log.Println("333>>>", methodIndex, ok)
			if ok {
				method2TypeIndexes[methodIndex] = append(method2TypeIndexes[methodIndex], t.index)
			}
		}
	}

	log.Println("number of interface method signatures:", lastMethodIndex, len(allInterfaceMethods), len(method2TypeIndexes))
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

		if len(uiInfo.methodIndexes) == 1 {
			sel := uiInfo.t.AllMethods[0]
			if sel.Name() == "Error" {
				log.Println("======================================= uiInfo.t: ", uiInfo.t)
				log.Println("=== typeIndexes:", typeIndexes)
			}
		}

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

		// Register non-pointer ones firstly, then
		// register pointer ones whose bases have not been registered.
		d.resetTempTypeLookupTable()
		impBy := make([]*TypeInfo, 0, count)
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if _, ok := t.TT.(*types.Pointer); !ok {
					if itt, ok := t.TT.Underlying().(*types.Interface); ok {
						ittInfo := interfaceUnderlyings.At(itt).(*UnderlyingInterfaceInfo)
						for _, it := range ittInfo.underlieds {
							impBy = append(impBy, it)
							typeLookupTable[it.index] = struct{}{}
						}
					} else {
						impBy = append(impBy, t)
						typeLookupTable[typeIndex] = struct{}{}
					}
				}
			}
		}
		for _, typeIndex := range typeIndexes {
			t := d.allTypeInfos[typeIndex]
			if t.counter == searchRound {
				if ptt, ok := t.TT.(*types.Pointer); ok {
					bt := d.RegisterType(ptt.Elem()) // 333 a: here to check why new types are registered
					if _, reged := typeLookupTable[bt.index]; !reged {
						impBy = append(impBy, t)
					}
				}
			}
		}
		uiInfo.t.ImplementedBys = impBy

		//log.Println("111 @@@", uiInfo.t.TT, ", uiInfo.methodIndexes:", uiInfo.methodIndexes)
		//for _, impBy := range impBy {
		//	log.Println("     ", impBy.TT)
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

func (d *CodeAnalyzer) analyzePackages_CollectSelectors() {
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

		d.collectSelectorsFroNonInterfaceType(t, smm, checkedTypes)

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

// ToDo: also check method signatures.
func (d *CodeAnalyzer) analyzePackages_CheckCollectSelectors(cache *typeutil.MethodSetCache) {
	for i := 0; i < len(d.allTypeInfos); i++ {
		t := d.allTypeInfos[i]
		switch tt := t.TT.(type) {
		case *types.Interface:
			if cache.MethodSet(tt).Len() != len(t.AllMethods) {
				panic(fmt.Sprintf("%v: interface (%d) method numbers not match. %d : %d.\n %v", t, t.index, cache.MethodSet(t.TT).Len(), len(t.AllMethods), t.AllMethods))
			}
		case *types.Pointer:
			switch btt := tt.Elem(); btt.Underlying().(type) {
			case *types.Interface, *types.Pointer:
				if num := cache.MethodSet(tt).Len(); num != 0 || len(t.AllMethods) != 0 {
					panic(fmt.Sprintf("%v: should not have methods. %d : %d", t, num, len(t.AllMethods)))
				}
			default:
				ttset := cache.MethodSet(tt)
				bttset := cache.MethodSet(btt)

				typesCount := len(d.allTypeInfos)
				bt := d.RegisterType(btt)
				num1, num2 := 0, 0
				for _, sel := range bt.AllMethods {
					num2++
					if !sel.PointerReceiverOnly() {
						num1++
					}
				}

				if len(d.allTypeInfos) > typesCount {
					//log.Println("> new types are added:", btt)
					//} else if ttset.Len() < num2 || bttset.Len() < num1 {
				} else if ttset.Len() != num2 || bttset.Len() != num1 {
					// This is a bug in std types.MethodSet implementation (Go SDK 1.14-)
					// https://github.com/golang/go/issues/37081

					log.Println("      promoted selectors collected?", t.attributes|promotedSelectorsCollected != 0, bt.attributes|promotedSelectorsCollected != 0)
					log.Println("      >>", bttset)
					panic(fmt.Sprintf("%v: method numbers not match: %d : %d and %d : %d. (%d) %v : %v", t, ttset.Len(), num2, bttset.Len(), num1, len(bt.DirectSelectors), bt.DirectSelectors, bt.AllMethods))
				}
			}
		}
	}
}

// ...

type SelectListManager struct {
	current *list.List
	free    *list.List
}

func NewSelectListManager() *SelectListManager {
	return &SelectListManager{
		current: list.New(),
		free:    list.New(),
	}
}

type SeleterMapManager struct {
	apply   func() (r map[string]*Selector)
	release func(r map[string]*Selector)
}

//var debug = false

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

	if t.Underlying == nil {
		// already set interface.Underlying in RegisterType now
		panic(fmt.Sprint("should never happen:", t.TT))

		// ToDo: move to RegisterType.
		underlying := t.TT.Underlying()
		UnderlyingTypeInfo := d.RegisterType(underlying)
		t.Underlying = UnderlyingTypeInfo
		UnderlyingTypeInfo.Underlying = UnderlyingTypeInfo
	}

	t.attributes |= promotedSelectorsCollected

	// ToDo: field and parameter/result interface types don't satisfy this.
	//if (t.Underlying.attributes & directSelectorsCollected) == 0 {
	//	panic("unnamed interface should have collected direct selectors now. " + fmt.Sprintf("%#v", t))
	//}

	//if debug {
	//	log.Println("==== 111", depth, t)
	//}
	if t != t.Underlying {
		//if t.TypeName.Name() == "TokenReviewInterface" || t.TypeName.Name() == "TokenReviewExpansion" {
		//	debug = true
		//	defer func() {
		//		debug = false
		//	}()
		//}
		// t is a named interface type.

		//if debug {
		//	log.Println("xxx ===", t.TT, len(t.DirectSelectors), "\n",
		//		t.Underlying.TT, len(t.Underlying.DirectSelectors), "\n",
		//		t.Underlying.DirectSelectors)
		//}

		//log.Println("222", depth)
		if (t.Underlying.attributes & directSelectorsCollected) == 0 {
			panic("unnamed interface should have collected direct selectors now. " +
				fmt.Sprintf("underlying index: %v. index: %v. name: %#v. %#v. %v",
					t.Underlying.index, t.index, t.TypeName.Name(), t.Underlying.TT, t.TT.Underlying()))
		}
		//if t.DirectSelectors != nil {
		//	panic("Selectors of named interface should be blank now")
		//}
		d.collectSelectorsForInterfaceType(t.Underlying, depth, currentCounter, smm)

		t.DirectSelectors = t.Underlying.DirectSelectors
		t.AllMethods = t.Underlying.AllMethods

		//if debug {
		//	log.Println("yyy ===", t.TT, len(t.AllMethods), "\n",
		//		t.Underlying.TT, len(t.Underlying.AllMethods), "\n",
		//		t.Underlying.AllMethods)
		//}
	} else { // t == t.Underlying
		//if debug {
		//	log.Println("333", depth)
		//}
		if (t.Underlying.attributes & directSelectorsCollected) == 0 {
			//if depth == 0 {
			//	return // ToDo: temp ignore field and paramter/result unnamed interface types
			//}
			log.Println("!!! t.index:", t.index, t.TT)
			panic("unnamed interface should have collected direct selectors now. " + fmt.Sprintf("%#v", t))
		}

		hasEmbeddings := false
		for _, s := range t.DirectSelectors {
			if s.Field != nil {
				hasEmbeddings = true
				break
			}
		}

		//if n := itt.NumEmbeddeds(); n == 0 {
		if !hasEmbeddings { // the embedding ones might overlap with non-embedding ones
			//if debug {
			//	log.Println("444", depth)
			//}
			t.AllMethods = t.DirectSelectors
		} else {
			//if debug {
			//	log.Println("555", depth)
			//}
			selectors := smm.apply()
			defer func() {
				smm.release(selectors)
			}()

			t.AllMethods = make([]*Selector, 0, len(t.DirectSelectors)+2*itt.NumEmbeddeds())

			for _, sel := range t.DirectSelectors {
				if sel.Method != nil {
					if old, ok := selectors[sel.Id]; ok {
						if old.Method.Type != sel.Method.Type {
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
					ut := sel.Field.Type.Underlying

					// The true is needed.
					//if true || !d.collectSelectorsForInterfaceType(ut, depth+1, currentCounter, smm) {
					d.collectSelectorsForInterfaceType(ut, depth+1, currentCounter, smm)
					if true {
						for _, sel := range ut.AllMethods {
							if old, ok := selectors[sel.Id]; ok {
								if old.Method.Type != sel.Method.Type {
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
		}
	}

	return
}

func (d *CodeAnalyzer) collectSelectorsFroNonInterfaceType(t *TypeInfo, smm *SeleterMapManager, checkedTypes map[uint32]uint16) {

	if (t.attributes & promotedSelectorsCollected) != 0 {
		return
	}

	defer func() {
		t.attributes |= promotedSelectorsCollected
	}()

	var namedType *TypeInfo
	var structType *TypeInfo

	switch t.TT.(type) {
	case *types.Named:
		namedType = t

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
		//debug := false
		//if len(structType.DirectSelectors) == 1 && structType.DirectSelectors[0].Name() == "C" {
		//	debug = true
		//}
		//if debug {
		//	log.Println("================================== structType=", structType)
		//}

		numEmbeddeds := 0
		for _, sel := range structType.DirectSelectors {
			if sel.Field == nil {
				panic("should not")
			}
			if sel.Field.Mode != EmbedMode_None {
				numEmbeddeds++
			}
		}

		// The simple case.
		if numEmbeddeds == 0 {
			t.AllFields = t.DirectSelectors
			if namedType != nil {
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
					} else {
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
					} else { // if sel.Method != nil
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

			//log.Println("       000")
			switch t := embeddedField.Field.Type; tt := t.TT.(type) {
			case *types.Named:
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

	d.collectSelectorsFroNonInterfaceType(structType, smm, checkedTypes)

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

// ToDo
func (d *CodeAnalyzer) confirmPackageModules() {
	// Two cases:
	// 1. check the .../vendor/modules.txt files
	// 2. check GOPATH/pkg/mod/...

	// # list all module dependency relations
	// go mod graph
	//	k8s.io/kubernetes sigs.k8s.io/yaml@v1.1.0
	//	...
	//	k8s.io/apiserver@v0.0.0 go.uber.org/zap@v1.10.0
	//	...

	// # from Michael Matloob
	// go list -f '{{.Module.Path}} {{.Module.Dir}}' pkg-import-path

	// # list all involved modules
	// go list -m all
	//	volcano.sh/volcano
	//	modernc.org/xc v1.0.0 => modernc.org/xc v1.0.0
	//	...
	// go list -f '{{.ImportPath}} {{.Module}}' all
	//	go101.org/gold/tests/n go101.org/gold
	//	golang.org/x/mod/internal/lazyregexp golang.org/x/mod v0.1.1-0.20191105210325-c90efee705ee
	//	unsafe <nil>
	//	vendor/golang.org/x/crypto/chacha20 <nil>
	//	...
	// go list -json all
	//	.Module
	//
	// Maybe, it is still better to analyze it manauuly.
	// Temp not to show module pages, module info is only used to find asParamsOf/asResultsOf

	// # get the module at CWD
	// go list -m
	//	volcano.sh/volcano
	//   or
	//	go list -m: not using modules

	// # find GOROOT to find std module info
	// go env

	//findPkgModule := func(pkg *Package) {
	//	// d.stdPackages
	//}
	//_ = findPkgModule

	//for _, pkg := range d.packageList {
	//	if len(pkg.PPkg.GoFiles) == 0 {
	//		continue
	//	}
	//	dir := filepath.Dir(pkg.PPkg.GoFiles[0])
	//	filename := filepath.Join(dir, "go.mod")
	//	filedata, err := ioutil.ReadFile(filename)
	//	if err != nil {
	//		if !errors.Is(err, os.ErrNotExist) {
	//			log.Printf("ioutil.ReadFile %s error: %s", filename, err)
	//		}
	//		continue
	//	}
	//
	//	modFile, err := modfile.ParseLax(filename, filedata, nil)
	//	if err != nil {
	//	}
	//
	//	mod := Module{
	//		Dir:     dir,
	//		Root:    modFile.Module.Mod.Path,
	//		Version: modFile.Module.Mod.Version,
	//	}
	//
	//	_ = mod
	//}

	// I decided to delay the impplementation of this funciton now.
	// One intention to confirm module information is
	// to support module pages, but this is not very essential.
	// Another intention to confirm module information
	// is to calculate the distances of pacakges.
	// However, it might be not perfect to determine
	// the distance of two packages by checking if
	// they are in the same module.
	//
	// The module info confirmed in this funciton will be only for showing.
}

// Important for registerFunctionForInvolvedTypeNames and registerValueForItsTypeName.
func (d *CodeAnalyzer) sortPackagesByDependencies() {
	var seen = make(map[string]struct{}, len(d.packageList))
	var calculatePackageDepLevel func(pkg *Package)
	calculatePackageDepLevel = func(pkg *Package) {
		if _, ok := seen[pkg.Path()]; ok {
			return
		}
		seen[pkg.Path()] = struct{}{}

		var max = 0
		for _, dep := range pkg.Deps {
			calculatePackageDepLevel(dep)
			if dep.DepLevel > max {
				max = dep.DepLevel
			} else if dep.DepLevel == 0 {
				log.Println("sortPackagesByDependencies: the dep.DepLevel is not calculated yet!")
			}
		}
		pkg.DepLevel = max + 1
	}

	for _, pkg := range d.packageList {
		calculatePackageDepLevel(pkg)
	}

	sort.Slice(d.packageList, func(i, j int) bool {
		return d.packageList[i].DepLevel < d.packageList[j].DepLevel
	})

	for i, pkg := range d.packageList {
		pkg.Index = i
	}

	//for _, pkg := range d.packageList {
	//	log.Println(">>>>>>>>>>>>>>>>", pkg.DepLevel, pkg.Path())
	//}
}

func (d *CodeAnalyzer) analyzePackage_CollectDeclarations(pkg *Package) {
	if pkg.PackageAnalyzeResult != nil {
		panic(pkg.Path() + " already analyzed")
	}

	//log.Println("[analyzing]", pkg.Path(), pkg.PPkg.Name)

	pkg.PackageAnalyzeResult = NewPackageAnalyzeResult()

	registerTypeName := func(tn *TypeName) {
		pkg.PackageAnalyzeResult.AllTypeNames = append(pkg.PackageAnalyzeResult.AllTypeNames, tn)
		d.RegisterTypeName(tn)
	}

	registerVariable := func(v *Variable) {
		pkg.PackageAnalyzeResult.AllVariables = append(pkg.PackageAnalyzeResult.AllVariables, v)
	}

	registerConstant := func(c *Constant) {
		pkg.PackageAnalyzeResult.AllConstants = append(pkg.PackageAnalyzeResult.AllConstants, c)
	}

	registerImport := func(imp *Import) {
		pkg.PackageAnalyzeResult.AllImports = append(pkg.PackageAnalyzeResult.AllImports, imp)
	}

	registerFunction := func(f *Function) {
		pkg.PackageAnalyzeResult.AllFunctions = append(pkg.PackageAnalyzeResult.AllFunctions, f)
		d.RegisterFunction(f)
		// function stats are moved to below
	}

	var isBuiltinPkg = pkg.Path() == "builtin"
	var isUnsafePkg = pkg.Path() == "unsafe"

	// ToDo: use info.TypeOf, info.ObjectOf

	for _, file := range pkg.PPkg.Syntax {
		d.stats.AstFiles++
		d.stats.Imports += int32(len(file.Imports))
		incSliceStat(d.stats.FilesByImportCount[:], len(file.Imports))

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

				// It looks the funciton delcared in "builtin" are types.Func, instead of types.Builtin.
				// But the funcitons declared in "unsafe" are types.Builtin.

				var f *Function

				obj := pkg.PPkg.TypesInfo.Defs[fd.Name]
				switch funcObj := obj.(type) {
				default:
					panic("not a types.Func")
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
							if pkg.Path() != "unsafe" {
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

						var srcType = tv.Type
						var objName = typeObj.Name()
						// Exported names, such as Type and Type1 are fake types.
						if isBuiltinPkg && !token.IsExported(objName) {
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
							srcType = typeObj.Type()

							// It looks the below twos are not equal, though
							// types.Idenfical(them) returns true. So, typeObj.Type()
							// and srcType are both internal ByteType, but not Uint8Type.
							// Sometimes, this might matter.
							//
							// ByteType:  types.Universe.Lookup("byte").(*types.TypeName).Type()
							// Uint8Type: types.Universe.Lookup("uint8").(*types.TypeName).Type()
							//
							// The type of a custom aliase is the type it denotes.
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

						srcTypeInfo := d.RegisterType(srcType)
						newTypeInfo := d.RegisterType(typeObj.Type())

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
							if srcTypeInfo != newTypeInfo {
								panic(fmt.Sprintf("srcTypeInfo != newTypeInfo, %v, %v", srcTypeInfo, newTypeInfo))
							}

							tn.Alias = &TypeAlias{
								Denoting: srcTypeInfo,
								TypeName: tn,
							}
							srcTypeInfo.Aliases = append(srcTypeInfo.Aliases, tn.Alias)

							if isBuiltinPkg || isUnsafePkg {
								tn.Alias.attributes |= Builtin
							}

							// ToDo: check embeddable

						} else {
							tn.Named = newTypeInfo
							newTypeInfo.TypeName = tn
							if isBuiltinPkg || isUnsafePkg {
								tn.Named.attributes |= Builtin
							}
						}

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

	//  We must do the collection work after all types are collected.
	for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
		if f.Func != nil {
			d.registerExplicitlyDeclaredMethod(f)
		}

		if f.IsMethod() && f.AstDecl.Recv != nil {
			if len(f.AstDecl.Recv.List) != 1 {
				panic("should not")
			}
			field := f.AstDecl.Recv.List[0]
			var id *ast.Ident
			switch expr := field.Type.(type) {
			default:
				panic("should not")
			case *ast.Ident:
				id = expr
			case *ast.StarExpr:
				tid, ok := expr.X.(*ast.Ident)
				if !ok {
					panic("should not")
				}
				id = tid
			}
			if !token.IsExported(id.Name) {
				// ToDo: If it is proved that some values of this type are
				//       exposed to other packages, then should not continue here.
				continue
			}
		}

		//if f.Exported() {
		//	d.registerFunctionForInvolvedTypeNames(f)
		//}
		numParams, numResults := d.registerFunctionForInvolvedTypeNames(f)

		if f.IsMethod() {
			d.stats.Methods++
			incSliceStat(d.stats.MethodsByParameterCount[:], numParams)
			incSliceStat(d.stats.FunctionsByResultCount[:], numResults)
		} else {
			d.stats.Functions++
			incSliceStat(d.stats.FunctionsByParameterCount[:], numParams)
			incSliceStat(d.stats.MethodsByResultCount[:], numResults)
		}
		if f.Builtin != nil || token.IsExported(f.Name()) {
			if f.IsMethod() {
				d.stats.ExportedMethods++
			} else {
				d.stats.ExportedFunctions++
			}
		}
	}
	//for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
	//	// moved to analyzePackage_CollectMoreStatistics
	//}
	if isBuiltinPkg {
		return
	}
	for _, v := range pkg.PackageAnalyzeResult.AllVariables {
		if v.Exported() {
			d.registerValueForItsTypeName(v)
			d.stats.ExportedVariables++
		}
	}
	for _, c := range pkg.PackageAnalyzeResult.AllConstants {
		if c.Exported() {
			d.registerValueForItsTypeName(c)
			d.stats.ExportedConstants++
		}
	}
}

func (d *CodeAnalyzer) analyzePackage_CollectMoreStatistics(pkg *Package) {
	if pkg.PackageAnalyzeResult == nil {
		panic(pkg.Path() + " is not analyzed yet")
	}
	var isBuiltinPkg = pkg.Path() == "builtin"

	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		if isBuiltinPkg != token.IsExported(tn.Name()) {
			if tn.Alias != nil {
				d.stats.ExportedTypeAliases++
			} else {
				kind := tn.Named.Kind()
				d.stats.ExportedNamedTypesByKind[kind]++
				d.stats.ExportedNamedTypes++

				var numExportedMethods = 0
				for _, sel := range tn.Named.AllMethods {
					if token.IsExported(sel.Name()) {
						numExportedMethods++
					}
				}
				if kind == reflect.Interface {
					incSliceStat(d.stats.ExportedNamedInterfacesByMethodCount[:], len(tn.Named.AllMethods))
					incSliceStat(d.stats.ExportedNamedInterfacesByExportedMethodCount[:], numExportedMethods)
					continue
				}
				incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByMethodCount[:], len(tn.Named.AllMethods))
				incSliceStat(d.stats.ExportedNamedNonInterfaceTypesByExportedMethodCount[:], numExportedMethods)

				if kind == reflect.Struct {
					incSliceStat(d.stats.NamedStructsByFieldCount[:], len(tn.Named.AllFields))

					var numExporteds, numExpliciteds, numExportedExpliciteds = 0, 0, 0
					for _, sel := range tn.Named.AllFields {
						if token.IsExported(sel.Name()) {
							numExporteds++
							if sel.Depth == 0 {
								numExportedExpliciteds++
							}
						}
						if sel.Depth == 0 {
							numExpliciteds++
						}
					}
					incSliceStat(d.stats.NamedStructsByExportedFieldCount[:], numExporteds)
					incSliceStat(d.stats.NamedStructsByExplicitFieldCount[:], numExpliciteds)
					incSliceStat(d.stats.NamedStructsByExportedExplicitFieldCount[:], numExportedExpliciteds)
				}
			}
		}
	}
}

func (d *CodeAnalyzer) analyzePackage_CollectMoreStatisticsFinal() {
	var sum = func(kinds ...reflect.Kind) (r int32) {
		for _, k := range kinds {
			r += d.stats.ExportedNamedTypesByKind[k]
		}
		return
	}
	d.stats.ExportedNamedUnsignedIntergerTypes = sum(reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr)
	d.stats.ExportedNamedIntergerTypes = d.stats.ExportedNamedUnsignedIntergerTypes + sum(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)
	d.stats.ExportedNamedNumericTypes = d.stats.ExportedNamedIntergerTypes + sum(reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128)

	d.stats.Packages = int32(len(d.packageList))
	for _, pkg := range d.packageList {
		d.stats.FilesWithGenerateds += int32(len(pkg.SourceFiles))
		d.stats.AllPackageDeps += int32(len(pkg.Deps))
		incSliceStat(d.stats.PackagesByDeps[:], len(pkg.Deps))
	}
}

func (d *CodeAnalyzer) analyzePackage_CollectSomeRuntimeFunctionPositions() {
	// ...
	if runtimePkg := d.packageTable["runtime"]; runtimePkg != nil {
		fnames := []string{
			"selectgo",      // for select blocks (except one-case-plus-default ones)
			"selectnbsend",  // one-case-plus-default select blocks
			"selectnbrecv",  // select {case v = <-c:; default:}
			"selectnbrecv2", // select {case v, ok = <-c:; default:}
			"chansend",      // c <- v
			"chanrecv1",     // v = <- c
			"chanrecv2",     // v, ok = <-c
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

func (d *CodeAnalyzer) analyzePackage_FindTypeSources(pkg *Package) {
	var isBuiltin = pkg.Path() == "builtin"

	//log.Println("[analyzing]", pkg.Path(), pkg.PPkg.Name)
	for _, file := range pkg.PPkg.Syntax {

		//ast.Inspect(file, func(n ast.Node) bool {
		//	log.Printf("%T\n", n)
		//	return true
		//})

		for _, decl := range file.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					typeSpec := spec.(*ast.TypeSpec)

					obj := pkg.PPkg.TypesInfo.Defs[typeSpec.Name]
					typeObj := obj.(*types.TypeName)
					if typeObj.Name() == "_" {
						continue
					}

					newTypeName := d.allTypeNameTable[d.Id2(typeObj.Pkg(), typeObj.Name())]
					if newTypeName == nil {
						panic("type name " + typeSpec.Name.Name + " not found: " + d.Id1(typeObj.Pkg(), typeObj.Name()))
					}

					var findSource func(ast.Expr, bool)
					findSource = func(srcNode ast.Expr, startSource bool) {
						var source *TypeSource
						if startSource {
							if newTypeName.StarSource == nil {
								newTypeName.StarSource = &TypeSource{}
							}
							source = newTypeName.StarSource
						} else {
							source = &newTypeName.Source
						}

						var sttNode *ast.StructType
						var ittNode *ast.InterfaceType

						switch expr := srcNode.(type) {
						case *ast.Ident:

							//log.Println("???", d.Id(pkg.PPkg.Types, expr.Name))

							//log.Println("   ", pkg.PPkg.TypesInfo.ObjectOf(expr))

							srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr)
							if srcObj == nil {
								if pkg.Path() != "unsafe" {
									panic("srcObj is nil but package is not unsafe")
								}
								return
							}
							srcTypeObj := srcObj.(*types.TypeName)

							//log.Println("   srcTypeObj.Pkg() =", srcTypeObj.Pkg())
							// if srcType is a built type, srcTypeObj.Pkg() == nil

							tn := d.allTypeNameTable[d.Id2(srcTypeObj.Pkg(), expr.Name)]
							if tn == nil {
								panic("type name " + expr.Name + " not found")
							}
							source.TypeName = tn

							//log.Println(startSource, "ident,", pkg.Path()+"."+typeSpec.Name.Name, "source is:", tn.Pkg.Path()+"."+expr.Name)

							return
						case *ast.SelectorExpr:
							//log.Println("selector,", pkg.Path()+"."+typeSpec.Name.Name, "source is:")
							srcObj := pkg.PPkg.TypesInfo.ObjectOf(expr.X.(*ast.Ident))
							srcPkg := srcObj.(*types.PkgName)

							tn := d.allTypeNameTable[d.Id2(srcPkg.Imported(), expr.Sel.Name)]
							if tn == nil {
								panic("type name " + expr.Sel.Name + " not found")
							}
							source.TypeName = tn

							//log.Println(startSource, "selector,", pkg.Path()+"."+typeSpec.Name.Name, "source is:", tn.Pkg.Path()+"."+expr.Sel.Name)
							return
						case *ast.ParenExpr:
							//log.Println("paren,", pkg.Path()+"."+typeSpec.Name.Name, "source is:")
							findSource(expr.X, false)
							return
						case *ast.StarExpr:
							if !startSource {
								//log.Println("star,", pkg.Path()+"."+typeSpec.Name.Name, "source is:")
								findSource(expr.X, true)
								return
							}
						case *ast.StructType:
							sttNode = expr
						case *ast.InterfaceType:
							ittNode = expr
						}

						// ToDo: don't use the std go/types and go/pacakges packages.
						//       Now, uint8 and byte are treat as two types by go/types.
						//       Write a custom one tailored for docs and code analyzing.
						//       Run "go mod tidy" before running gold using the custom packages
						//       to ensure all modules are cached locally.

						tv := pkg.PPkg.TypesInfo.Types[srcNode]
						srcTypeInfo := d.RegisterType(tv.Type)
						source.UnnamedType = srcTypeInfo
						//log.Println(startSource, "default,", pkg.Path()+"."+typeSpec.Name.Name, "source is:", tv.Type)

						if sttNode != nil {
							d.registerDirectFields(srcTypeInfo, sttNode, pkg)
						} else if ittNode != nil {
							if isBuiltin && typeSpec.Name.Name == "error" {
								/*
									//errorTN, _ := types.Universe.Lookup("error").(*types.TypeName)
									//errotUT := d.RegisterType(errorTN.Type().Underlying())
									//d.registerExplicitlySpecifiedMethods(errotUT, ittNode, pkg)

									//log.Println("=============== old:", srcTypeInfo.index)
									// This one is the type shown in the builtin.go source code,
									// not the one in type.Universal package. This one is only for docs purpose.
									// ToDo: use custom builtin page, remove the special handling.
									d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)

									// ToDo: load builtin.error != universal.error
									srcTypeInfo = d.RegisterType(newTypeName.Named.TT.Underlying())

									//log.Println("===============", errotUT.index, srcTypeInfo.index)
								*/

								d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
								srcTypeInfo = d.RegisterType(types.Universe.Lookup("error").(*types.TypeName).Type().Underlying())
							}
							d.registerExplicitlySpecifiedMethods(srcTypeInfo, ittNode, pkg)
						}
					}
					findSource(typeSpec.Type, false)
				}
			}
		}
	}

	return
}
