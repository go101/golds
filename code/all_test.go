package code

import (
	"go/types"
	"math/rand"
	"testing"
	"time"

	"golang.org/x/tools/go/types/typeutil"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestIncSliceStat(t *testing.T) {
	for range [16]struct{}{} {
		n := 2 + rand.Intn(32)
		a := make([]int32, n)
		var zeros, maxes int32 = 0, 0
		for range [4096]struct{}{} {
			k := rand.Intn(n + n)
			if k == 0 {
				zeros++
			} else if k >= n-1 {
				maxes++
			}
			incSliceStat(a, k)
		}
		if a[0] != zeros {
			t.Errorf("zeros not match: %d vs. %d", a[0], zeros)
		}
		if a[n-1] != maxes {
			t.Errorf("maxes not match: %d vs. %d", a[n-1], maxes)
		}
	}
}

func TestID(t *testing.T) {
	var analyzer CodeAnalyzer
	var check1 = func(pkg *Package, id string, expected string) {
		if theId := analyzer.Id1b(pkg, id); theId != expected {
			t.Errorf("Id1b: Id not match: %s vs. %s", theId, expected)
		}
	}
	var check2 = func(pkg *Package, id string, expected string) {
		if theId := analyzer.Id2b(pkg, id); theId != expected {
			t.Errorf("Id2b: Id not match: %s vs. %s", theId, expected)
		}
	}

	analyzer.ParsePackages(nil, nil, ToolchainInfo{}, "builtin", "math")
	stdPkg := analyzer.PackageByPath("builtin")
	mathPkg := analyzer.PackageByPath("math")

	check1(stdPkg, "int", "builtin.int")
	check1(stdPkg, "Int", "Int")
	check1(mathPkg, "int", "math.int")
	check1(mathPkg, "Int", "Int")

	check2(stdPkg, "int", "builtin.int")
	check2(stdPkg, "Int", "builtin.Int")
	check2(mathPkg, "int", "math.int")
	check2(mathPkg, "Int", "math.Int")
}

func TestRegisterType(t *testing.T) {
	var analyzer CodeAnalyzer
	var builtinType = func(name string) types.Type {
		return types.Universe.Lookup(name).Type()
	}
	if analyzer.RegisterType(builtinType("int32")) != analyzer.RegisterType(builtinType("rune")) {
		t.Errorf("int32 != rune")
	}
	if analyzer.RegisterType(builtinType("byte")) != analyzer.RegisterType(builtinType("uint8")) {
		t.Errorf("uint8 != byte")
	}
	if analyzer.RegisterType(builtinType("byte")) == analyzer.RegisterType(builtinType("int8")) {
		t.Errorf("byte == int8")
	}
}

// ToDo: also check method signatures.
// There is a bug in std types.MethodSet implementation (Go SDK 1.14-)
// https://github.com/golang/go/issues/37081
// Luckily, his test is okay to test with the results of standard packages.
func TestAnalyzeStandardPackage(t *testing.T) {
	var analyzer CodeAnalyzer
	analyzer.ParsePackages(nil, nil, ToolchainInfo{}, "std")
	analyzer.AnalyzePackages(nil)

	var cache = &typeutil.MethodSetCache{}

	for i := 0; i < len(analyzer.allTypeInfos); i++ {
		ti := analyzer.allTypeInfos[i]
		switch tt := ti.TT.(type) {
		case *types.Interface:
			if cache.MethodSet(tt).Len() != len(ti.AllMethods) {
				t.Errorf("%v: interface (%d) method numbers not match. %d : %d.\n %v", ti, ti.index, cache.MethodSet(ti.TT).Len(), len(ti.AllMethods), ti.AllMethods)
			}
		case *types.Pointer:
			switch btt := tt.Elem(); btt.Underlying().(type) {
			case *types.Interface, *types.Pointer:
				if num := cache.MethodSet(tt).Len(); num != 0 || len(ti.AllMethods) != 0 {
					t.Errorf("%v: should not have methods. %d : %d", ti, num, len(ti.AllMethods))
				}
			default:
				ttset := cache.MethodSet(tt)
				bttset := cache.MethodSet(btt)

				typesCount := len(analyzer.allTypeInfos)
				bt := analyzer.RegisterType(btt)
				num1, num2 := 0, 0
				for _, sel := range bt.AllMethods {
					num2++
					if !sel.PointerReceiverOnly() {
						num1++
					}
				}

				// ToDo: now, we ignore types like struct{T} etc.
				if bt.TypeName == nil {
					break
				}

				if len(analyzer.allTypeInfos) > typesCount {
					t.Log("============================")
					t.Log("> new types are added: ", btt)
					t.Errorf("  ti = %#v", ti)
				} else if ttset.Len() != num2 || bttset.Len() != num1 {
					// This is a bug in std types.MethodSet implementation (Go SDK 1.14-)
					// https://github.com/golang/go/issues/37081

					t.Log("============================")
					t.Logf("> method numbers not match: %d : %d and %d : %d", ttset.Len(), num2, bttset.Len(), num1)
					t.Log("      promoted selectors collected? ", ti.attributes|promotedSelectorsCollected != 0, bt.attributes|promotedSelectorsCollected != 0)
					t.Log("      >> btt = ", btt)
					t.Log("      -- ", bttset)
					t.Log("      >> tt = ", tt)
					t.Log("      -- ", ttset)
					t.Log("      >> bt = ", bt)
					t.Errorf("      -- %v: .\n      -- (%d)      -- %v      -- %v", ti, len(bt.DirectSelectors), bt.DirectSelectors, bt.AllMethods)
				}
			}
		}
	}
}
