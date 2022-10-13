package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go101.org/golds/code"
)

type TestData_Package struct {
	Types      map[string]TestData_Type
	VarNames   []string
	ConstNames []string
	FuncNames  []string
}

type TestData_Type struct {
	FieldNames         []string
	MethodNames        []string
	ImplementedByCount int
	ImplementCount     int
	ValueCount         int
	AsInputCount       int
	AsOutputCount      int
	IsAlias            bool // ignore aliases in checking
}

func isInformalPackage(pkgPath string) bool {
	if strings.HasPrefix(pkgPath, "vendor/") {
		return true
	}
	if strings.HasPrefix(pkgPath, "internal/") {
		return true
	}
	if strings.Contains(pkgPath, "/internal/") {
		return true
	}
	if strings.HasSuffix(pkgPath, "/internal") {
		return true
	}
	if pkgPath == "internal" {
		return true
	}
	return false
}

func buildTestData_Package(details *PackageDetails) TestData_Package {
	ts := make(map[string]TestData_Type, len(details.TypeNames))
	//varNames := make([]string, 0, len(details.ValueResources))
	//constNames := make([]string, 0, len(details.ValueResources))
	//funcNames := make([]string, 0, len(details.ValueResources))
	varNames := make([]string, 0, len(details.Variables))
	constNames := make([]string, 0, len(details.Constants))
	funcNames := make([]string, 0, len(details.Functions))

	for _, twp := range details.TypeNames {
		t := twp.Type
		fieldNames := make([]string, 0, len(t.Fields))
		for _, f := range t.Fields {
			fieldNames = append(fieldNames, f.Name())
		}
		methodNames := make([]string, 0, len(t.Methods))
		for _, m := range t.Methods {
			methodNames = append(methodNames, m.Name())
		}

		// ...
		var implementedByCount int
		for _, impedBy := range t.ImplementedBys {
			if isInformalPackage(impedBy.Package().Path) {
				continue
			}
			implementedByCount++
		}

		var implementsCount int
		for _, impl := range t.Implements {
			if isInformalPackage(impl.Package().Path) {
				continue
			}
			implementsCount++
		}

		var valueCount int
		for _, v := range t.Values {
			if isInformalPackage(v.Package().Path) {
				continue
			}
			valueCount++
		}

		var asInputCount int
		for _, v := range t.AsInputsOf {
			if isInformalPackage(v.Package().Path) {
				continue
			}
			asInputCount++
		}

		var asOutputCount int
		for _, v := range t.AsOutputsOf {
			if isInformalPackage(v.Package().Path) {
				continue
			}
			asOutputCount++
		}

		// ...
		ts[t.TypeName.Name()] = TestData_Type{
			FieldNames:         fieldNames,
			MethodNames:        methodNames,
			ImplementedByCount: implementedByCount, //len(t.ImplementedBys),
			ImplementCount:     implementsCount,    // len(t.Implements),
			ValueCount:         valueCount,         // len(t.Values),
			AsInputCount:       asInputCount,       // len(t.AsInputsOf),
			AsOutputCount:      asOutputCount,      // len(t.AsOutputsOf),
			IsAlias:            t.TypeName.Alias != nil,
		}
	}

	//for _, v := range details.ValueResources {
	//	switch v := v.(type) {
	//	case *code.Variable:
	//		varNames = append(varNames, v.Name())
	//	case *code.Constant:
	//		constNames = append(constNames, v.Name())
	//	//case *code.Function:
	//	case code.FunctionResource:
	//		funcNames = append(funcNames, v.Name())
	//	}
	//}

	for _, v := range details.Variables {
		varNames = append(varNames, v.Value.Name())
	}

	for _, v := range details.Constants {
		constNames = append(constNames, v.Value.Name())
	}

	for _, v := range details.Functions {
		funcNames = append(funcNames, v.Value.Name())
	}

	return TestData_Package{
		Types:      ts,
		VarNames:   varNames,
		ConstNames: constNames,
		FuncNames:  funcNames,
	}
}

func buildTestData(args []string, silent bool, printUsage func(io.Writer)) map[string]TestData_Package {
	var analyzer code.CodeAnalyzer
	if err := analyzer.ParsePackages(nil, nil, code.ToolchainInfo{}, "std"); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	analyzer.AnalyzePackages(nil)

	numPkgs := analyzer.NumPackages()
	pkgTestDatas := make(map[string]TestData_Package, numPkgs)
	for i := 0; i < numPkgs; i++ {
		pkg := analyzer.PackageAt(i)
		if isInformalPackage(pkg.Path) {
			continue
		}

		details := buildPackageDetailsData(&analyzer, pkg.Path, false)
		pkgTestDatas[pkg.Path] = buildTestData_Package(details)

		if !silent {
			log.Printf("%s", pkg.Path)
		}
	}
	return pkgTestDatas
}

func GenTestData(args []string, outputDir string, silent bool, printUsage func(io.Writer)) {
	pkgTestDatas := buildTestData(args, silent, printUsage)

	if outputDir == "" {
		return
	}

	data, err := json.MarshalIndent(pkgTestDatas, "", "\t")
	//data, err := json.Marshal(pkgTestDatas)
	if err != nil {
		log.Fatalln("matshal error:", err)
	}

	dataFilePath := filepath.Join(outputDir, "generated-testdata-"+time.Now().Format("20060102150405"), "testdata.json")

	if err := os.MkdirAll(filepath.Dir(dataFilePath), 0700); err != nil {
		log.Fatalln("Mkdir error:", err)
	}

	if err := ioutil.WriteFile(dataFilePath, data, 0644); err != nil {
		log.Fatalln("Write file error:", err)
	}

	log.Printf("TestData generated at %s", dataFilePath)
}

// mainly for testing
func mapStringKeys(m map[string]TestData_Type) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Mainly for testing.
// Check whether or not a is a subset of b.
// Elements of a and b will be sorted.
// Assume no duplication in each of the two slices.
func assureSubsetStringSlice(a, b []string) error {
	if len(a) > len(b) {
		return fmt.Errorf("slice with more elements can't be a subset: %d vs. %d", len(a), len(b))
	}
	less := func(z []string) func(int, int) bool {
		return func(x, y int) bool {
			return z[x] < z[y]
		}
	}
	sort.Slice(a, less(a))
	sort.Slice(b, less(b))

	checkDuplication := func(z []string) {
		if len(z) == 0 {
			return
		}
		e := z[0]
		for i := 1; i < len(z); i++ {
			if z[i] == e {
				panic("duplicated elements: " + e)
			}
			e = z[i]
		}
	}
	checkDuplication(a)
	checkDuplication(b)

	i, k := 0, 0
	for i <= k && i < len(a) && k < len(b) {
		switch strings.Compare(a[i], b[k]) {
		case 1:
			k++
		case 0:
			i++
			k++
		case -1:
			return fmt.Errorf("%s is not found", a[i])
		}
	}
	if i == len(a) {
		return nil
	}
	return fmt.Errorf("%s is not found", a[i])
}
