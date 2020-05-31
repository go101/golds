package server

import (
	"fmt"
	"net/http"
	"sort"
)

func (ds *docServer) packageDependenciesPage(w http.ResponseWriter, r *http.Request, pkgPath string) {
	w.Header().Set("Content-Type", "text/html")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	if ds.dependencyPages[pkgPath] == nil {
		// ToDo: not found

		depInfo := ds.buildPackageDependenciesData(pkgPath)
		if depInfo == nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Package (%s) not found", pkgPath)
			return
		}

		ds.dependencyPages[pkgPath] = ds.buildPackageDependenciesPage(depInfo)
	}
	w.Write(ds.dependencyPages[pkgPath])
}

type PackageDependencyInfo struct {
	Name       string
	ImportPath string
	Index      int

	Imports     []*PackageForListing
	ImportedBys []*PackageForListing
}

func (ds *docServer) buildPackageDependenciesData(pkgPath string) *PackageDependencyInfo {
	pkg := ds.analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil
	}

	result := &PackageDependencyInfo{
		Name:       pkg.PPkg.Name,
		ImportPath: pkgPath,
		Index:      pkg.Index,
	}

	imports := make([]PackageForListing, len(pkg.Deps))
	result.Imports = make([]*PackageForListing, len(pkg.Deps))
	for i, pkg := range pkg.Deps {
		result.Imports[i] = &imports[i]

		result.Imports[i].Package = pkg
		result.Imports[i].Path = pkg.Path()
		result.Imports[i].Remaining = pkg.Path()
		result.Imports[i].Name = pkg.PPkg.Name
		result.Imports[i].Index = pkg.Index
	}

	importedBys := make([]PackageForListing, len(pkg.DepedBys))
	result.ImportedBys = make([]*PackageForListing, len(pkg.DepedBys))
	for i, pkg := range pkg.DepedBys {
		result.ImportedBys[i] = &importedBys[i]

		result.ImportedBys[i].Package = pkg
		result.ImportedBys[i].Path = pkg.Path()
		result.ImportedBys[i].Remaining = pkg.Path()
		result.ImportedBys[i].Name = pkg.PPkg.Name
		result.ImportedBys[i].Index = pkg.Index
	}

	sortPackageList := func(pkgs []*PackageForListing) {
		sort.Slice(pkgs, func(a, b int) bool {
			return pkgs[a].Path < pkgs[b].Path
		})
	}
	sortPackageList(result.Imports)
	sortPackageList(result.ImportedBys)

	ImprovePackagesForListing(result.Imports)
	ImprovePackagesForListing(result.ImportedBys)

	return result
}

func (ds *docServer) buildPackageDependenciesPage(depInfo *PackageDependencyInfo) []byte {
	page := NewHtmlPage(ds.goldVersion, ds.currentTranslation.Text_DependencyRelations(depInfo.ImportPath), ds.currentTheme.Name(), pagePathInfo{ResTypeDependency, depInfo.ImportPath})

	fmt.Fprintf(page, `
<pre><code><span style="font-size:xx-large;">package <b>%s</b></span>
`,
		depInfo.Name,
	)

	fmt.Fprintf(page, `
<span class="title">%s</span>
	<a href="%s">%s</a>`,
		ds.currentTranslation.Text_ImportPath(),
		buildPageHref(page.PathInfo, pagePathInfo{ResTypePackage, depInfo.ImportPath}, nil, ""),
		depInfo.ImportPath,
	)

	if len(depInfo.Imports) > 0 {
		fmt.Fprint(page, "\n\n", `<span class="title">`, ds.currentTranslation.Text_Imports(), `</span>`)
		ds.writePackagesForListing(page, depInfo.Imports, false, false)
	}

	if len(depInfo.ImportedBys) > 0 {
		fmt.Fprint(page, "\n\n", `<span class="title" id="imported-by">`, ds.currentTranslation.Text_ImportedBy(), `</span>`)
		ds.writePackagesForListing(page, depInfo.ImportedBys, false, false)
	}

	return page.Done()
}
