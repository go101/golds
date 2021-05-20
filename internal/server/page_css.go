package server

import (
	//"bytes"
	"net/http"
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

	pageKey := pageCacheKey{
		resType: ResTypeCSS,
		res:     themeName,
		options: options,
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		page := NewHtmlPage(goldsVersion, "", nil, ds.currentTranslation, createPagePathInfo(ResTypeCSS, themeName))

		theme := ds.themeByName(themeName)
		css := theme.CSS() + commonCSS
		t, err := template.New("css").Parse(css)
		if err != nil {
			panic("parse css template error: " + err.Error())
		}
		//var buf bytes.Buffer
		//if t.Execute(&buf, options) != nil {
		if t.Execute(page, options) != nil {
			panic("execute css template error: " + err.Error())
		}

		data = page.Done(w)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

var commonCSS = `

`
