package server

import (
	"fmt"
	"go/token"
	"log"
	"net/http"
	"sort"
	"strings"

	"go101.org/gold/code"
)

func (ds *docServer) overviewPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Parsed {
		writeAutoRefreshHTML(w, r)
		return
	}

	if ds.packageListPage == nil {
		overview := ds.buildOverviewData()
		ds.packageListPage = ds.buildOverviewPage(overview)
	}
	w.Write(ds.packageListPage)
}

func (ds *docServer) buildOverviewPage(overview *Overview) []byte {
	page := NewHtmlPage(ds.currentTranslation.Text_Overview(), ds.currentTheme.Name(), true, "", ResTypeNone)
	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">%s</span></code>

<code><span class="title">%s</span></code>`,
		ds.currentTranslation.Text_Overview(),
		ds.currentTranslation.Text_PackageList(len(overview.Packages)),
	)

	ds.writePackagesForListing(page, overview.Packages, true, true)

	page.WriteString("</pre>")

	return page.Done()
}

func (ds *docServer) writePackagesForListing(page *htmlPage, packages []*PackageForListing, writeAnchorTarget, inGenModeRootPages bool) {
	const MainPkgArrow = "m-&gt;"
	const MainPkgArrowCharCount = 3
	const MinPrefixSpacesCount = 3
	var maxDigitCount = 2 // 2 for ". " suffix
	for n := len(packages); n > 0; n /= 10 {
		maxDigitCount++
	}
	var SPACES = strings.Repeat(" ", maxDigitCount+MainPkgArrowCharCount+MinPrefixSpacesCount+1) // +1 for space after MainPkgArrow

	for i, pkg := range packages {
		if writeAnchorTarget {
			fmt.Fprintf(page, `<div class="anchor" id="pkg-%d">`, pkg.Index)
		} else {
			page.WriteByte('\n')
		}
		page.WriteString(`<code>`)
		page.WriteString(SPACES[:MinPrefixSpacesCount])
		var index = fmt.Sprintf("%d. ", i+1)
		if pkg.Name == "main" {
			mainObj := pkg.Package.PPkg.Types.Scope().Lookup("main")
			var mainPos token.Position
			if mainObj == nil {
				log.Println("main function is not found in package", pkg.Path)
			} else {
				mainPos = pkg.Package.PPkg.Fset.PositionFor(mainObj.Pos(), false)
			}
			if mainPos.IsValid() {
				//mainPos.Line += ds.analyzer.SourceFileLineOffset(mainPos.Filename)
				ds.writeSrouceCodeLineLink(page, pkg.Package, mainPos, MainPkgArrow, "", true)
			} else {
				page.WriteString(MainPkgArrow)
			}
			page.WriteByte(' ')
			page.WriteString(SPACES[:maxDigitCount-len(index)])
			page.WriteString(index)
		} else {
			page.WriteString(SPACES[:MainPkgArrowCharCount+1])
			page.WriteString(SPACES[:maxDigitCount-len(index)])
			page.WriteString(index)
		}

		if pkg.Prefix != "" {
			//fmt.Fprintf(page,
			//	`<a href="/pkg:%s" class="path-duplicate">%s</a>`,
			//	pkg.Path, pkg.Prefix)
			buildPageHref(ResTypePackage, pkg.Path, inGenModeRootPages, pkg.Prefix, page)
		}
		if pkg.Remaining != "" {
			//fmt.Fprintf(page,
			//	`<a href="/pkg:%s">%s</a>`,
			//	pkg.Path, pkg.Remaining)
			buildPageHref(ResTypePackage, pkg.Path, inGenModeRootPages, pkg.Remaining, page)
		}
		page.WriteString(`</code>`)
		if writeAnchorTarget {
			page.WriteString(`</div>`)
		}
	}
}

type Overview struct {
	Packages []*PackageForListing
}

type PackageForListing struct {
	Package *code.Package

	Index     int
	Mod       *code.Module
	Name      string
	Path      string // blank for not analyzed yet
	Prefix    string // the part shared with the last one in list
	Remaining string // the part different from the last one in list
}

func (ds *docServer) buildOverviewData() *Overview {
	numPkgs := ds.analyzer.NumPackages()
	var pkgs = make([]PackageForListing, numPkgs)
	var result = make([]*PackageForListing, numPkgs)
	for i := range result {
		pkg := &pkgs[i]
		result[i] = pkg

		p := ds.analyzer.PackageAt(i)
		pkg.Package = p
		if p.Mod != nil {
			pkg.Mod = p.Mod
		}
		pkg.Path = p.Path()
		pkg.Remaining = p.Path()
		pkg.Name = p.PPkg.Name
		pkg.Index = p.Index
	}

	// ToDo: might be problematic sometimes. Should sort token by token.
	sort.Slice(result, func(a, b int) bool {
		return result[a].Path < result[b].Path
	})

	ImprovePackagesForListing(result)

	return &Overview{
		Packages: result,
	}
}

func ImprovePackagesForListing(pkgs []*PackageForListing) {
	if len(pkgs) <= 1 {
		return
	}

	var last = pkgs[0]
	for i := 1; i < len(pkgs); i++ {
		current := pkgs[i]
		current.Prefix = FindPackageCommonPrefixPaths(last.Remaining, current.Remaining)
		last = current
	}
	for i := 1; i < len(pkgs); i++ {
		current := pkgs[i]
		current.Remaining = current.Remaining[len(current.Prefix):]
	}
}

// ToDo: ToLower both?
func FindPackageCommonPrefixPaths(pa, pb string) string {
	var n = len(pa)
	if n > len(pb) {
		n = len(pb)
		pa, pb = pb, pa
	}
	var i = 0
	for ; i < n; i++ {
		if pa[i] == pb[i] {
			continue
		}
		break
	}
	if i == n {
		if len(pb) == n {
			return pa // or pb
		}
		if len(pb) > n && pb[n] == '/' {
			return pb[:n+1]
		}
	}
	for i--; i >= 0; i-- {
		if pb[i] == '/' {
			return pb[:i+1]
		}
	}
	return ""
}
