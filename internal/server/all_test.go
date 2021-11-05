package server

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"go101.org/golds/internal/util"
)

func init() {
	enabledHtmlGenerationMod() // to test buildPageHref
	testingMode = true
}

func TestFindPackageCommonPrefixPaths(t *testing.T) {
	var testCases = [][3]string{
		{"aaa/bbb/ccc", "aaa/bbb/ddd", "aaa/bbb/"},
		{"aaa/bbb/ccc", "aaa/bbb/", "aaa/bbb/"},
		{"aaa/bbb/ccc", "aaa/bbb", "aaa/bbb/"},
		{"aaa/bbb/ccc", "aaabbb", ""},
		{"aaa/bbb/ccc", "aaa", "aaa/"},
		{"aaa", "aaa", "aaa"},
	}
	for _, tc := range testCases {
		if path := FindPackageCommonPrefixPaths(tc[0], tc[1]); path != tc[2] {
			t.Errorf("common prefix not match (%s, %s): %s vs. %s", tc[0], tc[1], path, tc[2])
		}
		if path := FindPackageCommonPrefixPaths(tc[1], tc[0]); path != tc[2] {
			t.Errorf("common prefix (swap) not match (%s, %s): %s vs. %s", tc[0], tc[1], path, tc[2])
		}
	}
}

func TestRelativePath(t *testing.T) {
	var testCases = [][3]string{
		{"aaa/bbb/ccc", "aaa/bbb/ddd", "ddd"},
		{"aaa/bbb/", "aaa/bbb/ddd", "ddd"},
		{"aaa/bbb/", "aaa/bbb/", ""},
		{"aaa/bbb/", "aaa/ccc", "../ccc"},
		{"aaa/bbb/.html", "aaa/ccc.html", "../ccc.html"},
		{"aaa/bbb/ccc", "aaa/xxx/ddd", "../xxx/ddd"},
		{"aaa/bbb/ccc", "aaa/xxx/", "../xxx/"},
		{"aaa/bbb/ccc", "xxx/bbb/", "../../xxx/bbb/"},
		{"aaa", "xxx/bbb", "xxx/bbb"},
		{"aaa", "bbb", "bbb"},
		{"aaa/bbb/ccc", "aaa/", "../"},
		{"aaa/bbb/ccc", "aaa", "../"},
		{"aaa", "aaa/bbb/ccc", "bbb/ccc"},
		{"aaa/", "aaa/bbb/ccc", "bbb/ccc"},
	}
	for _, tc := range testCases {
		if rel := RelativePath(tc[0], tc[1]); rel != tc[2] {
			t.Errorf("relative path not match (%s, %s): %s vs. %s", tc[0], tc[1], rel, tc[2])
		}
	}
}

func TestComparePackagePaths(t *testing.T) {
	var vs = []struct {
		x, y   string
		b1, b2 bool
	}{
		{"abc/cde-xyz", "abc/cde/xyz", false, true},
		{"abc/cde/xyz", "abc/cde-xyz", true, false},
		{"abc/cde/xyz", "abc/cde/xyz", true, true},
		{"abc/cde-xyz", "abc/cde-xyz", true, true},
		{"abc/cde-xyz", "afc/cde-xyz", true, true},
		{"afc/cde-xyz", "abc/cde-xyz", false, false},
	}

	for _, v := range vs {
		if ComparePackagePaths(v.x, v.y, '/') != v.b1 {
			t.Errorf("ComparePackagePaths(%s, %s) != %v", v.x, v.y, v.b1)
		}
		if v.x <= v.y != v.b2 {
			t.Errorf("(%s <= %s) == %v", v.x, v.y, v.b2)
		}
	}
}

func TestBuildPageHref(t *testing.T) {
	type testCase struct {
		from, to pagePathInfo
		expected string
	}
	type info = pagePathInfo
	var testCases = []testCase{
		{info{ResTypePackage, "xxx/yyy"}, info{ResTypePackage, "xxx/zzz"}, "zzz.html"},
		{info{ResTypePackage, "xxx/yyy/zzz"}, info{ResTypePackage, "xxx/zzz"}, "../zzz.html"},
		{info{ResTypePackage, "xxx/yyy/"}, info{ResTypePackage, "xxx/zzz"}, "../zzz.html"},
		{info{ResTypePackage, "xxx/yyy/"}, info{ResTypeCSS, "xxx/zzz"}, "../../../" + string(ResTypeCSS) + "/xxx/zzz." + string(ResTypeCSS)},
	}
	for _, tc := range testCases {
		if href := buildPageHref(tc.from, tc.to, nil, ""); href != tc.expected {
			t.Errorf("page href not match (%v, %v): %s vs. %s", tc.from, tc.to, href, tc.expected)
		}
	}
}

func TestBuildLineOffsets(t *testing.T) {
	type testCase struct {
		content  string
		expected int
	}
	var testCases = []testCase{
		{"", 1},
		{"aaa", 1},
		{"\n", 2},
		{"\nbbbb", 2},
		{"aaa\nbbbb", 2},
		{"\r\n", 2},
		{"aaa\r\nbbb", 2},
		{"aaa\r\n\n\rbbb", 3},
	}
	for _, tc := range testCases {
		if n, _ := BuildLineOffsets([]byte(tc.content), true); n != tc.expected {
			t.Errorf("line count not match (%s): %d vs. %d", tc.content, n, tc.expected)
		}
	}
}

func TestAssureSubsetStringSlice(t *testing.T) {
	ss := func(s ...string) []string {
		return append([]string(nil), s...)
	}
	type testCase struct {
		isSubsetOfB, isSubsetOfA bool
		a, b                     []string
	}
	var testCases = []testCase{
		{true, true, ss(), ss()},
		{true, false, ss(), ss("aa")},
		{true, false, ss("cc", "aa"), ss("bb", "aa", "cc")},
		{false, false, ss("cc", "aa", "dd"), ss("bb", "aa", "cc")},
		{false, true, ss("cc", "aa", "dd", "ee"), ss("aa", "cc")},
	}
	for _, tc := range testCases {
		if err := assureSubsetStringSlice(tc.a, tc.b); (err == nil) != tc.isSubsetOfB {
			t.Errorf("assure string slice subset not match: {a: %v, b: %v}", tc.a, tc.b)
		}
		if err := assureSubsetStringSlice(tc.b, tc.a); (err == nil) != tc.isSubsetOfA {
			t.Errorf("assure string slice subset not match: {a: %v, b: %v}", tc.b, tc.a)
		}
	}
}

func TestPreviousVersion(t *testing.T) {
	type testCase struct {
		version, previous string
	}
	var testCases = []testCase{
		{"0.0.0", ""},
		{"0.0.1", "0.0.0"},
		{"0.0.1-pre", "0.0.0"},
		{"0.0.x", ""},
		{"0.1.1", "0.1.0"},
		{"0.1.0", "0.0.9"},
		{"1.1.0", "1.0.9"},
		{"1.0.0", "0.9.9"},
		{"v1.0.0", "v0.9.9"},
		{"1.0.0-pre", "0.9.9"},
		{"v1.0.0-pre", "v0.9.9"},
	}
	for _, tc := range testCases {
		if PreviousVersion(tc.version) != tc.previous {
			t.Errorf("wrong previous version (%s) for %s", tc.previous, tc.version)
		}
	}
}

func TestDocsForStandardPackages(t *testing.T) {
	// ...
	data, err := ioutil.ReadFile(filepath.Join("..", "testing", "data", "testdata.json.tar.gz"))
	if err != nil {
		t.Errorf("Read testdata.json.tar.gz error: %s", err)
	}

	data, err = util.UncompressTarGzipData(data)
	if err != nil {
		t.Errorf("Uncompress testdata.json.tar.gz error: %s", err)
	}

	var testdataOld map[string]TestData_Package
	err = json.Unmarshal(data, &testdataOld)
	if err != nil {
		t.Errorf("Unmarshal test data error: %s", err)
	}

	// ...
	testdataNew := buildTestData([]string{"std"}, true, nil)
	_, _ = testdataNew, testdataOld

	// ...
	for pkgPath, pkgTestDataOld := range testdataOld {
		if isInformalPackage(pkgPath) {
			continue
		}
		pkgTestDataNew, ok := testdataNew[pkgPath]
		if !ok {
			if pkgPath != "runtime/cgo" {
				t.Errorf("Package %s is missing", pkgPath)
			}
			continue
		}

		if err := assureSubsetStringSlice(mapStringKeys(pkgTestDataOld.Types), mapStringKeys(pkgTestDataNew.Types)); err != nil {
			t.Errorf("[%s] Types become less: %s", pkgPath, err)
		}
		for typeName, typeTestDataOld := range pkgTestDataOld.Types {
			typeTestDataNew := pkgTestDataNew.Types[typeName]

			if typeTestDataNew.IsAlias || typeTestDataOld.IsAlias {
				// Go 1.16 add a "io/fs" package,
				// the old os.FileMode/FileInfo etc become aliases
				// of the corresponding ones in "io/fs" package.
				// Now, the collections for alias types are all blank.
				// To make the test pass, so here skip it.
				continue
			}

			if err := assureSubsetStringSlice(typeTestDataOld.FieldNames, typeTestDataNew.FieldNames); err != nil {
				t.Errorf("[%s] %s fields become less: %s", pkgPath, typeName, err)
			}
			if err := assureSubsetStringSlice(typeTestDataOld.MethodNames, typeTestDataNew.MethodNames); err != nil {
				t.Errorf("[%s] %s methods become less: %s", pkgPath, typeName, err)
			}
			if n, m := typeTestDataOld.ImplementedByCount, typeTestDataNew.ImplementedByCount; n > m {
				t.Errorf("[%s] %s implementdBy count becomes less: %d > %d", pkgPath, typeName, n, m)
			}
			if n, m := typeTestDataOld.ImplementCount, typeTestDataNew.ImplementCount; n > m {
				t.Errorf("[%s] %s implement count becomes less: %d > %d", pkgPath, typeName, n, m)
			}
			if n, m := typeTestDataOld.ValueCount, typeTestDataNew.ValueCount; n > m {
				t.Errorf("[%s] %s value count becomes less: %d > %d", pkgPath, typeName, n, m)
			}
			if n, m := typeTestDataOld.AsInputCount, typeTestDataNew.AsInputCount; n > m {
				t.Errorf("[%s] %s asInput count becomes less: %d > %d", pkgPath, typeName, n, m)
			}
			if n, m := typeTestDataOld.AsOutputCount, typeTestDataNew.AsOutputCount; n > m {
				t.Errorf("[%s] %s asOutput count becomes less: %d > %d", pkgPath, typeName, n, m)
			}
		}

		if err := assureSubsetStringSlice(pkgTestDataOld.VarNames, pkgTestDataNew.VarNames); err != nil {
			t.Errorf("[%s] Vars become less: %s", pkgPath, err)
		}
		if err := assureSubsetStringSlice(pkgTestDataOld.ConstNames, pkgTestDataNew.ConstNames); err != nil {
			t.Errorf("[%s] Consts become less: %s", pkgPath, err)
		}
		if err := assureSubsetStringSlice(pkgTestDataOld.FuncNames, pkgTestDataNew.FuncNames); err != nil {
			t.Errorf("[%s] Funcs become less: %s", pkgPath, err)
		}
	}
}

func TestGenerateDocsOfStandardPackages(t *testing.T) {
	opts := PageOutputOptions{GoldsVersion: "v0.0.0", PreferredLang: "en-US", SourceReadingStyle: SourceReadingStyle_rich}
	GenDocs(opts, []string{"std"}, "", true, nil, false, nil)
	opts = PageOutputOptions{GoldsVersion: "v0.0.0", PreferredLang: "en-US", SourceReadingStyle: SourceReadingStyle_rich, NotCollectUnexporteds: true}
	GenDocs(opts, []string{"std"}, "", true, nil, false, nil)
	opts = PageOutputOptions{GoldsVersion: "v0.0.0", PreferredLang: "en-US", SourceReadingStyle: SourceReadingStyle_external}
	GenDocs(opts, []string{"std"}, "", true, nil, false, nil)
	GenTestData([]string{"std"}, "", true, nil)
}
