package server

import (
	"testing"
)

func init() {
	enabledHtmlGenerationMod() // to test buildPageHref
}

func TestFindPackageCommonPrefixPaths(t *testing.T) {
	var testCases = [][3]string {
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
	var testCases = [][3]string {
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

func TestBuildPageHref(t *testing.T) {
	type testCase struct {
		from, to pagePathInfo
		expected string
	}
	type info = pagePathInfo
	var testCases = []testCase {
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
		content string
		expected int
	}
	var testCases = []testCase {
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

func TestDocsForStandardPackages(t *testing.T) {
	
}
