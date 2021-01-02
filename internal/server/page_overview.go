package server

import (
	"fmt"
	"go/token"
	"log"
	"net/http"
	"sort"
	"strings"

	"go101.org/golds/code"
)

type overviewPageOptions struct {
	sortBy string // "alphabet", "importedbys"
}

func (ds *docServer) overviewPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	//if ds.phase < Phase_Parsed {
	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	} else if r.FormValue("js") != "" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if !genDocsMode {
		if ds.confirmUpdateTip(); ds.updateTip != ds.cachedUpdateTip {
			ds.cachedUpdateTip = ds.updateTip
			//ds.theOverviewPage = nil

			// clear possible cached pages
			pageKey := pageCacheKey{
				resType: ResTypeNone,
				res:     "",
			}
			pageKey.options = overviewPageOptions{sortBy: "alphabet"}
			ds.cachePage(pageKey, nil)
			pageKey.options = overviewPageOptions{sortBy: "importedbys"}
			ds.cachePage(pageKey, nil)
		}
	}

	pageKey := pageCacheKey{
		resType: ResTypeNone,
		res:     "",
	}
	oldOptions, ok := ds.cachedPageOptions(pageKey).(overviewPageOptions)

	var sortBy = r.FormValue("sortby")
	switch sortBy {
	case "alphabet", "importedbys":
	default:
		if ok {
			sortBy = oldOptions.sortBy
		} else {
			sortBy = "alphabet"
		}
	}

	newOptions := overviewPageOptions{sortBy: sortBy}
	if newOptions != oldOptions {
		ds.cachePageOptions(pageKey, newOptions)
	}

	//if ds.theOverviewPage == nil || sortBy != ds.theOverviewPage.sortBy {
	//	overview := ds.buildOverviewData(sortBy)
	//	ds.theOverviewPage = &overviewPage{
	//		content: ds.buildOverviewPage(overview, sortBy),
	//		sortBy:  sortBy,
	//	}
	//}
	//w.Write(ds.theOverviewPage.content)

	pageKey.options = newOptions
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		overview := ds.buildOverviewData(newOptions.sortBy)
		data = ds.buildOverviewPage(w, overview, newOptions.sortBy)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildOverviewPage(w http.ResponseWriter, overview *Overview, sortBy string) []byte {
	page := NewHtmlPage(ds.goldVersion, ds.currentTranslation.Text_Overview(), ds.currentTheme, ds.currentTranslation, pagePathInfo{ResTypeNone, ""})
	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">%s</span></code></pre>
`,
		page.Translation().Text_Overview(),
	)

	if !genDocsMode {
		ds.writeUpdateGoldBlock(page)
	}

	ds.writeSimpleStatsBlock(page, &overview.Stats)

	page.WriteString("<pre>")

	if genDocsMode {
		fmt.Fprintf(page, `<code><span class="title">%s</span></code>`,
			page.Translation().Text_PackageList(),
		)
	} else {
		var textSortByAlphabet = page.Translation().Text_SortByItem("alphabet")
		var textSortByImportedBys = page.Translation().Text_SortByItem("importedbys")

		switch sortBy {
		case "alphabet":
			textSortByImportedBys = fmt.Sprintf(`<a href="%s">%s</a>`, "?sortby=importedbys", textSortByImportedBys)
		case "importedbys":
			textSortByAlphabet = fmt.Sprintf(`<a href="%s">%s</a>`, "?sortby=alphabet", textSortByAlphabet)
		}

		fmt.Fprintf(page, `<code><span class="title">%s (%s%s | %s)</span></code>`,
			page.Translation().Text_PackageList(),
			page.Translation().Text_SortBy(),
			textSortByAlphabet,
			textSortByImportedBys,
		)
	}

	ds.writePackagesForListing(page, overview.Packages, true, sortBy)

	page.WriteString("</pre>")

	return page.Done(w)
}

func (ds *docServer) writePackagesForListing(page *htmlPage, packages []*PackageForListing, writeAnchorTarget bool, sortBy string) {
	const MainPkgArrowCharCount = 3
	const MinPrefixSpacesCount = 3
	var maxDigitCount = 2 // 2 for ". " suffix
	for n := len(packages); n > 0; n /= 10 {
		maxDigitCount++
	}
	var SPACES = strings.Repeat(" ", maxDigitCount+MainPkgArrowCharCount+MinPrefixSpacesCount+1) // +1 for space after MainPkgArrow

	var maxDepLevel int32
	for _, pkg := range packages {
		if pkg.DepLevel > maxDepLevel {
			maxDepLevel = pkg.DepLevel
		}
	}

	listPackage := func(i int, pkg *PackageForListing) {
		if writeAnchorTarget {
			fmt.Fprintf(page, `<div class="anchor" id="pkg-%s" data-importedbys="%d" data-dependencylevel="%d">`, pkg.Path, pkg.NumImportedBys, pkg.DepLevel)
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
			writeMainFunctionArrow(page, pkg.Package, mainPos)
			page.WriteString(SPACES[:maxDigitCount-len(index)])
			page.WriteString(index)
		} else {
			page.WriteString(SPACES[:MainPkgArrowCharCount+1])
			page.WriteString(SPACES[:maxDigitCount-len(index)])
			page.WriteString(index)
		}

		if pkg.Prefix != "" {
			fmt.Fprintf(page,
				`<a href="%s" class="path-duplicate">%s</a>`,
				buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, pkg.Path}, nil, ""),
				pkg.Prefix,
			)

		}
		if pkg.Remaining != "" {
			fmt.Fprintf(page,
				`<a href="%s">%s</a>`,
				buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, pkg.Path}, nil, ""),
				pkg.Remaining,
			)

		}
		if sortBy == "importedbys" {
			fmt.Fprintf(page, ` <i>(%d)</i>`, pkg.NumImportedBys)
		}
		page.WriteString(`</code>`)
		if writeAnchorTarget {
			page.WriteString(`</div>`)
		}
	}

	promoteWDPkgs := emphasizeWDPackages || ds.emphasizeWDPkgs

	if promoteWDPkgs {
		lastInWorkingDirectory := false
		for i, pkg := range packages {
			if lastInWorkingDirectory != pkg.InWorkingDirectory {
				if lastInWorkingDirectory {
					page.WriteByte('\n')
				}
				lastInWorkingDirectory = pkg.InWorkingDirectory
			}
			listPackage(i, pkg)
		}
	} else {
		for i, pkg := range packages {
			listPackage(i, pkg)
		}
	}
}

var divVisibility = map[bool]string{false: " hidden", true: ""}

func (ds *docServer) writeUpdateGoldBlock(page *htmlPage) {
	fmt.Fprintf(page, `
<pre id="%s" class="golds-update%s">%s</pre>
<pre id="%s" class="golds-update hidden">%s</pre>
<pre id="%s" class="golds-update%s">%s</pre>
`,
		UpdateTip2DivID[UpdateTip_ToUpdate], divVisibility[ds.updateTip == UpdateTip_ToUpdate], fmt.Sprintf(page.Translation().Text_UpdateTip("ToUpdate"), GoldsUpdateGoSubCommand(ds.appPkgPath)),
		UpdateTip2DivID[UpdateTip_Updating], page.Translation().Text_UpdateTip("Updating"),
		UpdateTip2DivID[UpdateTip_Updated], divVisibility[ds.updateTip == UpdateTip_Updated], page.Translation().Text_UpdateTip("Updated"),
	)
}

func (ds *docServer) writeSimpleStatsBlock(page *htmlPage, stats *code.Stats) {
	text := page.Translation().Text_SimpleStats(stats)
	text = strings.Replace(text, "\n", "\n\t", -1)
	moreLink := buildPageHref(page.PathInfo, pagePathInfo{ResTypeNone, "statistics"}, nil, "")
	fmt.Fprintf(page, `
<pre><code><span class="title">%s</span></code>
	%s
</pre>`,
		page.Translation().Text_StatisticsWithMoreLink(moreLink),
		text,
	)
}

type Overview struct {
	Packages []*PackageForListing

	code.Stats
}

type PackageForListing struct {
	Package *code.Package

	Index     int
	Mod       *code.Module
	Name      string
	Path      string // blank for not analyzed yet
	Prefix    string // the part shared with the last one in list
	Remaining string // the part different from the last one in list

	DepLevel       int32
	NumImportedBys int32

	InWorkingDirectory bool
}

func (ds *docServer) buildOverviewData(sortBy string) *Overview {
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

		pkg.DepLevel = int32(p.DepLevel)
		pkg.NumImportedBys = int32(len(p.DepedBys))

		pkg.InWorkingDirectory = strings.HasPrefix(p.Directory, ds.workingDirectory)
	}

	switch sortBy {
	case "alphabet":
		// ToDo: might be problematic sometimes. Should sort token by token.
		sort.Slice(result, func(a, b int) bool {
			promoteWDPkgs := emphasizeWDPackages || ds.emphasizeWDPkgs
			if promoteWDPkgs {
				if result[a].InWorkingDirectory != result[b].InWorkingDirectory {
					return result[a].InWorkingDirectory
				}
			}
			return result[a].Path < result[b].Path
		})
		ImprovePackagesForListing(result)
	case "importedbys":
		var pkgs = result
		for i, pkg := range pkgs {
			if pkg.Path == "builtin" {
				pkg.NumImportedBys = int32(len(pkgs) - 1)
				pkgs[0], pkgs[i] = pkg, pkgs[0]
				pkgs = pkgs[1:]
				break
			}
		}
		sort.Slice(pkgs, func(a, b int) bool {
			switch n := pkgs[a].NumImportedBys - pkgs[b].NumImportedBys; {
			case n > 0:
				return true
			case n < 0:
				return false
			}
			return pkgs[a].Path < pkgs[b].Path
		})
	}

	return &Overview{
		Packages: result,
		Stats:    ds.analyzer.Statistics(),
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
		if len(current.Prefix) < len(current.Remaining) {
			current.Remaining = current.Remaining[len(current.Prefix):]
		}
	}
}

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
