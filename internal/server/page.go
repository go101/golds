package server

import (
	"bytes"
	"fmt"
	"strings"
)

type pageResType string

const (
	ResTypeNone       pageResType = ""
	ResTypeAPI        pageResType = "api"
	ResTypePackage    pageResType = "pkg"
	ResTypeDependency pageResType = "dep"
	ResTypeSource     pageResType = "src"
	ResTypeCSS        pageResType = "css"
	ResTypeJS         pageResType = "jvs"
	ResTypeSVG        pageResType = "svg"
)

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

func (page *htmlPage) Done() []byte {
	//if genDocsMode {}

	fmt.Fprintf(page, `<pre id="footer">
Generated with <a href="https://go101.org/article/tool-gold.html"><b>Gold</b></a> <i>%s</i>.
<b>Gold</b> is a <a href="https://go101.org">Go 101</a> project started by <a href="https://tapirgames.com">TapirLiu</a>.
Please follow <a href="https://twitter.com/go100and1">@Go100and1</a> to get the latest news of <b>Gold</b>.
PR and bug reqports are welcomed and can be submitted <a href="https://github.com/go101/gold">here</a>.
</pre>`,
		page.goldVersion,
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
