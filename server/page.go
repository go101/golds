package server

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
)

type pageResType string

const (
	ResTypeNone       pageResType = ""
	ResTypePackage    pageResType = "pkg"
	ResTypeDependency pageResType = "dep"
	ResTypeSource     pageResType = "src"
	ResTypeCSS        pageResType = "css"
	ResTypeJS         pageResType = "jvs"
)

func OpenBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	return exec.Command(cmd, append(args, url)...).Start()
}

func writeAutoRefreshHTML(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<meta http-equiv="refresh" content="1.5; url=%[1]v">
	
		<title>Analyzing ...</title>
	</head>

	<body>
	Analyzing ... (<a href="%[1]v">refresh</a>)
	</body>
</html>`, r.URL.String())
}

type htmlPage struct {
	bytes.Buffer
	theme *Theme
	trans Translation

	PathInfo pagePathInfo
}

type pagePathInfo struct {
	resType pageResType
	resPath string
}

func NewHtmlPage(title, themeName string, currentPageInfo pagePathInfo) *htmlPage {
	page := htmlPage{PathInfo: currentPageInfo}
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
<body onload="onPageLoaded()"><div>
`,
		title,
		buildPageHref(currentPageInfo, pagePathInfo{ResTypeCSS, themeName}, nil, ""),
		buildPageHref(currentPageInfo, pagePathInfo{ResTypeJS, "gold"}, nil, ""),
	)

	return &page
}

func (page *htmlPage) Done() []byte {
	writePageGenerationInfo(page)

	page.WriteString(`</div></body></html>`)
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
