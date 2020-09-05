package server

import (
	"bytes"
	"fmt"
	"go/build"
	"strings"

	"go101.org/gold/internal/server/translations"
)

type pageResType string

const (
	ResTypeNone           pageResType = ""
	ResTypeAPI            pageResType = "api"
	ResTypeModule         pageResType = "mod"
	ResTypePackage        pageResType = "pkg"
	ResTypeDependency     pageResType = "dep"
	ResTypeImplementation pageResType = "imp"
	ResTypeSource         pageResType = "src"
	ResTypeReference      pageResType = "use"
	ResTypeCSS            pageResType = "css"
	ResTypeJS             pageResType = "jvs"
	ResTypeSVG            pageResType = "svg"
	ResTypePNG            pageResType = "png"
)

type pageCacheKey struct {
	resType pageResType
	res     interface{}
	options interface{}
}

type pageCacheValue struct {
	data    []byte
	options interface{}
}

func (ds *docServer) cachePage(key pageCacheKey, data []byte) {
	if !genDocsMode {
		ds.cachedPages[key] = data
	}
}

func (ds *docServer) cachedPage(key pageCacheKey) (data []byte, ok bool) {
	if genDocsMode {
	} else {
		data, ok = ds.cachedPages[key]
	}
	return
}

func (ds *docServer) cachePageOptions(key pageCacheKey, options interface{}) {
	if !genDocsMode {
		key.options = nil
		ds.cachedPagesOptions[key] = options
	}
}

func (ds *docServer) cachedPageOptions(key pageCacheKey) (options interface{}) {
	if genDocsMode {
	} else {
		key.options = nil
		options = ds.cachedPagesOptions[key]
	}
	return
}

type htmlPage struct {
	bytes.Buffer

	theme       *Theme
	trans       Translation
	goldVersion string

	PathInfo pagePathInfo
}

type pagePathInfo struct {
	resType pageResType
	resPath string
}

func NewHtmlPage(goldVersion, title, themeName string, currentPageInfo pagePathInfo) *htmlPage {
	page := htmlPage{PathInfo: currentPageInfo, goldVersion: goldVersion}
	page.Grow(4 * 1024 * 1024)

	fmt.Fprintf(&page, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<link href="%s" rel="stylesheet">
<script src="%s"></script>
<body><div>
`,
		title,
		buildPageHref(currentPageInfo, pagePathInfo{ResTypeCSS, addVersionToFilename(themeName, page.goldVersion)}, nil, ""),
		buildPageHref(currentPageInfo, pagePathInfo{ResTypeJS, addVersionToFilename("gold", page.goldVersion)}, nil, ""),
	)

	return &page
}

func (page *htmlPage) Done(translation Translation) []byte {
	//if genDocsMode {}

	var qrImgLink string
	switch translation.(type) {
	case *translations.Chinese:
		qrImgLink = buildPageHref(page.PathInfo, pagePathInfo{ResTypePNG, "go101-wechat"}, nil, "")
	case *translations.English:
		qrImgLink = buildPageHref(page.PathInfo, pagePathInfo{ResTypePNG, "go101-twitter"}, nil, "")
	}

	fmt.Fprintf(page, `<pre id="footer">
%s
</pre>`,
		translation.Text_GeneratedPageFooter(page.goldVersion, qrImgLink, build.Default.GOOS, build.Default.GOARCH),
	)

	page.WriteString(`
</div></body></html>`,
	)
	return append([]byte(nil), page.Bytes()...)
}

func (page *htmlPage) writePageLink(writeHref func(), linkText string, fragments ...string) {
	if linkText != "" {
		page.WriteString(`<a href="`)
	}
	writeHref()
	if len(fragments) > 0 {
		page.WriteByte('#')
		for _, fm := range fragments {
			page.WriteString(fm)
		}
	}
	if linkText != "" {
		page.WriteString(`">`)
		page.WriteString(linkText)
		page.WriteString(`</a>`)
	}
}

func addVersionToFilename(filename string, version string) string {
	return filename + "-" + version
}

func removeVersionFromFilename(filename string, version string) string {
	return strings.TrimSuffix(filename, "-"+version)
}
