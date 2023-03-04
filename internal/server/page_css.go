package server

import (
	//"bytes"

	"io"
	"net/http"
	"os"
	"text/template"
)

//type cssFile struct {
//	content []byte
//	options cssFileOptions
//}

func (ds *docServer) cssFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "text/css")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if genDocsMode {
		themeName = deHashFilename(themeName)
	}

	options := struct {
		Colon string
		Fonts string
	}{
		Colon: ds.currentTranslation.Text_Colon(true),
		Fonts: ds.currentTranslation.Text_PreferredFontList(),
	}

	//if options != ds.theCSSFile.options {
	//	theme := ds.themeByName(themeName)
	//	css := theme.CSS() + commonCSS
	//	t, err := template.New("css").Parse(css)
	//	if err != nil {
	//		panic("parse css template error: " + err.Error())
	//	}
	//	var buf bytes.Buffer
	//	if t.Execute(&buf, options) != nil {
	//		panic("execute css template error: " + err.Error())
	//	}
	//	ds.theCSSFile = cssFile{
	//		content: buf.Bytes(),
	//		options: options,
	//	}
	//}
	//
	//w.Write(ds.theCSSFile.content)

	// load user CSS
	userCSS := ""
	bHasUserCSS := false
	userCSSPath := os.ExpandEnv("${HOME}/.config/golds/custom.css")
	if pF, err := os.Open(userCSSPath); err == nil {
		if tmp, err := io.ReadAll(pF); err == nil {
			userCSS = string(tmp)
			bHasUserCSS = true
		}
		pF.Close()
	}

	// load from cache if user CSS not provided
	pageKey := pageCacheKey{
		resType: ResTypeCSS,
		res:     themeName,
		options: options,
	}
	if !bHasUserCSS {
		if data, ok := ds.cachedPage(pageKey); ok && (len(data) > 0) {
			w.Write(data)
			return
		}
	}

	// rebuild page if not already in cache
	page := NewHtmlPage(
		goldsVersion, "", nil, ds.currentTranslation,
		createPagePathInfo(ResTypeCSS, themeName),
	)

	// apply template to CSS
	theme := ds.themeByName(themeName)
	css := commonCSS + theme.CSS() + userCSS
	t, err := template.New("css").Parse(css)
	if err != nil {
		panic("parse css template error: " + err.Error())
	}
	if t.Execute(page, options) != nil {
		panic("execute css template error: " + err.Error())
	}

	// render page
	data := page.Done(w)

	// save to cache when user CSS not provided
	if !bHasUserCSS {
		ds.cachePage(pageKey, data)
	}

	w.Write(data)
}

var commonCSS = `

`
