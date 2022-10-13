package server

import (
	"fmt"
	"go/token"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"go101.org/golds/code"
)

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
			ds.cachePage(pageKey, nil)
		}
	}

	pageKey := pageCacheKey{
		resType: ResTypeNone,
		res:     "",
	}

	data, ok := ds.cachedPage(pageKey)
	if !ok {
		overview := ds.buildOverviewData()
		data = ds.buildOverviewPage(w, overview)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildOverviewPage(w http.ResponseWriter, overview *Overview) []byte {
	page := NewHtmlPage(goldsVersion, ds.currentTranslation.Text_Overview(), ds.currentTheme, ds.currentTranslation, createPagePathInfo(ResTypeNone, ""))
	fmt.Fprintf(page, `
<pre id="overview"><code><span style="font-size:xx-large;">%s</span></code></pre>
`,
		page.Translation().Text_Overview(),
	)

	if !genDocsMode {
		ds.writeUpdateGoldBlock(page)
	}

	if showStatistics {
		ds.writeSimpleStatsBlock(page, &overview.Stats)
	}

	page.WriteString("<pre><code>")

	page.WriteString(`<span class="title">`)
	page.WriteString(page.Translation().Text_PackageList())
	page.WriteString(`<span id="buttons1" class="js-on title-stat">`)
	page.WriteString(page.Translation().Text_Parenthesis(false))
	page.WriteString(`<span class="buttons-content">`)
	page.WriteString(page.Translation().Text_SortBy("packages"))
	page.WriteString(page.Translation().Text_Colon(false))
	page.WriteString(`<label id="btn-alphabet" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("alphabet"))
	page.WriteString(`</label>`)
	page.WriteString(`<span id="importedbys"> | `)
	page.WriteString(`<label id="btn-importedbys" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("importedbys"))
	page.WriteString(`</label></span>`)
	page.WriteString(`<span id="codelines"> | `)
	page.WriteString(`<label id="btn-codelines" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("codelines"))
	page.WriteString(`</label></span>`)
	page.WriteString(`<span id="depdepth"> | `)
	page.WriteString(`<label id="btn-depdepth" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("depdepth"))
	page.WriteString(`</label></span>`)
	page.WriteString(`</span>`)
	page.WriteString(page.Translation().Text_Parenthesis(true))
	page.WriteString("</span>")
	page.WriteString(`</span>`)

	ds.writePackagesForListing(page, overview.Packages, true)

	page.WriteString("</code></pre>")

	return page.Done(w)
}

func (ds *docServer) writePackagesForListing(page *htmlPage, packages []*PackageForListing, writeAnchorTarget bool) {
	if len(packages) == 0 {
		return
	}

	const MainPkgArrowCharCount = 2
	const MinPrefixSpacesCount = 3
	var maxDigitCount = 0 // 2 // 2 for ". " suffix
	for n := len(packages); n > 0; n /= 10 {
		maxDigitCount++
	}
	var SPACES = strings.Repeat(" ", maxDigitCount+MainPkgArrowCharCount+MinPrefixSpacesCount+1) // +1 for space after MainPkgArrow

	//var maxDepHeight int32
	//for _, pkg := range packages {
	//	if pkg.DepHeight > maxDepHeight {
	//		maxDepHeight = pkg.DepHeight
	//	}
	//}

	listPackage := func(i int, pkg *PackageForListing, hidden, writeDataAttrs bool) {
		if writeAnchorTarget {
			extraClass := ""
			if hidden {
				extraClass = " hidden"
			}
			main := ""
			if pkg.Name == "main" {
				main = ` data-main="1"`
			}
			// ToDo: could pkg path contains invalid id chars?
			//       if so, use pkg id instead.
			if writeDataAttrs {
				fmt.Fprintf(page, `<i id="max-digit-count" class="hidden">%d</i>`, maxDigitCount)
			}
			fmt.Fprintf(page, `<div class="anchor pkg alphabet%s" id="pkg-%s"`, extraClass, pkg.Path)
			if writeDataAttrs {
				fmt.Fprintf(page, ` data-module="%s" data-loc="%d" data-importedbys="%d" data-depheight="%d" data-depdepth="%d"%s`, pkg.Module, pkg.LOC, pkg.NumImportedBys, pkg.DepHeight, pkg.DepDepth, main)
			}
			page.WriteString(`>`)
			defer page.WriteString(`</div>`)
		} else {
			page.WriteString("\n")
		}

		page.WriteString(`<code>`)
		defer page.WriteString(`</code>`)
		page.WriteString(SPACES[:MinPrefixSpacesCount])
		if pkg.Name == "main" {
			mainObj := pkg.Package.PPkg.Types.Scope().Lookup("main")
			var mainPos token.Position
			if mainObj == nil {
				log.Println("main function is not found in package", pkg.Path)
			} else {
				mainPos = pkg.Package.PPkg.Fset.PositionFor(mainObj.Pos(), false)
			}
			writeMainFunctionArrow(page, pkg.Package, mainPos)
		} else {
			page.WriteString(SPACES[:MainPkgArrowCharCount+1])
		}
		var index = strconv.Itoa(i + 1)
		fmt.Fprintf(page, `<span class="order">%s%d</span>. `, SPACES[:maxDigitCount-len(index)], i+1)

		if hidden {
			// a hidden one as :target will be shown.
			fmt.Fprintf(page,
				`<a href="%s">%s</a>`,
				buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, pkg.Path), nil, ""),
				pkg.Path,
			)
		} else {
			if pkg.Prefix != "" {
				fmt.Fprintf(page,
					`<a href="%s" class="path-duplicate">%s</a>`,
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, pkg.Path), nil, ""),
					pkg.Prefix,
				)
			}
			if pkg.Remaining != "" {
				fmt.Fprintf(page,
					`<a href="%s">%s</a>`,
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, pkg.Path), nil, ""),
					pkg.Remaining,
				)
			}
		}

		if writeDataAttrs {
			if pkg.Path != "builtin" {
				func() {
					page.WriteString(`<i class="importedbys"> (`)
					defer page.WriteString(`)</i>`)
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypeDependency, pkg.Path), page, strconv.Itoa(int(pkg.NumImportedBys)), "imported-by")
				}()
			} else {
				fmt.Fprintf(page, `<i class="importedbys"> (%d)</i>`, pkg.NumImportedBys)
			}
			fmt.Fprintf(page, `<i class="codelines"> (%d)</i>`, pkg.LOC)
			fmt.Fprintf(page, `<i class="depheight"> (%d)</i>`, pkg.DepHeight)
			fmt.Fprintf(page, `<i class="depdepth"> (%d)</i>`, pkg.DepDepth)
		}

		const PackageSpace = "Package "
		d := pkg.OneLineDoc
		if strings.HasPrefix(d, PackageSpace) {
			d = d[len(PackageSpace):]
		}
		if strings.HasPrefix(d, pkg.Name) {
			d = d[len(pkg.Name):]
		}
		if strings.HasPrefix(d, " ") {
			d = d[1:]
		}
		if len(d) > 0 {
			page.WriteString(`<span class="pkg-summary"> - `)
			page.AsHTMLEscapeWriter().WriteString(d)
			defer page.WriteString(`</span>`)
		}
	}

	if !writeAnchorTarget {
		page.WriteString(`<div id="packages">`)
		for i, pkg := range packages {
			listPackage(i, pkg, false, false)
		}
		page.WriteString(`</div>`)
		return
	}

	page.WriteString(`<input type='checkbox' id="toggle-summary">`)

	switch wdPkgsListingManner {
	case WdPkgsListingManner_promoted, WdPkgsListingManner_solo:
		showOthers := wdPkgsListingManner != WdPkgsListingManner_solo

		page.WriteString(`<div id="wd-packages" class="alphabet">`)
		i := 0
		for _, pkg := range packages {
			if pkg.InWorkingDirectory {
				listPackage(i, pkg, false, showOthers)
				i++
			}
		}
		page.WriteString(`</div>`)

		if showOthers {
			page.WriteByte('\n')
			page.WriteString(`<span id="buttons2" class="js-on">`)
			page.WriteString("\n")
			page.WriteString(SPACES[:len(SPACES)-maxDigitCount])
			page.WriteString("/* ")
			page.WriteString(`<span class="buttons-content">`)
			page.WriteString(`</span>`)
			page.WriteString(` */</span>`)
		}

		page.WriteString(`<div id="packages" class="alphabet">`)
		for _, pkg := range packages {
			if !pkg.InWorkingDirectory {
				listPackage(i, pkg, !showOthers, showOthers)
				i++
			}
		}
		page.WriteString(`</div>`)
	case WdPkgsListingManner_general:
		page.WriteString(`<div id="packages">`)
		for i, pkg := range packages {
			listPackage(i, pkg, false, true)
		}
		page.WriteString(`</div>`)
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
	moreLink := buildPageHref(page.PathInfo, createPagePathInfo(ResTypeNone, "statistics"), nil, "")
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
	Name      string
	Path      string // blank for not analyzed yet
	Prefix    string // the part shared with the last one in list
	Remaining string // the part different from the last one in list
	//Module    *code.Module

	Module string // module path

	OneLineDoc string

	NumImportedBys int32
	DepHeight      int32
	DepDepth       int32 // The value mains how close to main pacakges.
	LOC            int32

	//IsStandard         bool
	InWorkingDirectory bool
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
		//if p.Module != nil {
		//	pkg.Module = p.Module
		//}
		pkg.Path = p.Path
		pkg.Remaining = p.Path
		pkg.Name = p.PPkg.Name
		pkg.Index = p.Index

		pkg.Module = p.ModulePath()
		pkg.OneLineDoc = p.OneLineDoc

		pkg.LOC = p.CodeLinesWithBlankLines
		pkg.DepHeight = p.DepHeight
		pkg.DepDepth = p.DepDepth
		pkg.NumImportedBys = int32(len(p.DepedBys))
		if pkg.Name == "builtin" {
			pkg.NumImportedBys = int32(numPkgs) - 1
		}

		pkg.InWorkingDirectory = strings.HasPrefix(p.Directory, ds.initialWorkingDirectory)
	}

	// ToDo: might be problematic sometimes. Should sort token by token.
	promoteWDPkgs := wdPkgsListingManner == WdPkgsListingManner_promoted || wdPkgsListingManner == WdPkgsListingManner_solo
	sort.Slice(result, func(a, b int) bool {
		if promoteWDPkgs {
			if result[a].InWorkingDirectory != result[b].InWorkingDirectory {
				return result[a].InWorkingDirectory
			}
		}
		// ...

		//return result[a].Path < result[b].Path
		return code.ComparePackagePaths(result[a].Path, result[b].Path, '/')
	})
	ImprovePackagesForListing(result)

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
	if n <= len(pb) { // BCE hint
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
	}
	return ""
}
