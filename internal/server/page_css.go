package server

import (
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
		css := commonCSS + theme.CSS()

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

.js-on {display: none;}

/* overview page */

.pkg-summary {display: none;}
input#toggle-summary {display: none;}
input#toggle-summary:checked ~ div .pkg-summary {display: inline;}

div.alphabet .importedbys {display: none;}
div.alphabet .codelines {display: none;}
div.alphabet .depdepth {display: none;}
div.alphabet .depheight {display: none;}
div.importedbys .importedbys {display: inline;}
div.importedbys .codelines {display: none;}
div.importedbys .depdepth {display: none;}
div.importedbys .depheight {display: none;}
div.depdepth .depdepth {display: inline;}
div.depdepth .codelines {display: none;}
div.depdepth .importedbys {display: none;}
div.depdepth .depheight {display: none;}
div.depheight .depheight {display: inline;}
div.depheight .codelines {display: none;}
div.depheight .importedbys {display: none;}
div.depheight .depdepth {display: none;}
div.codelines .codelines {display: inline;}
div.codelines .depheight {display: none;}
div.codelines .importedbys {display: none;}
div.codelines .depdepth {display: none;}

/* package details page */

div:target {display: block;}
input.fold {display: none;}
/*input.fold + label +*/ .fold-items {display: none;}
/*input.fold + label +*/ .fold-docs {display: none;}
input.fold:checked + label + .fold-items {display: inline;}
input.fold:checked + label + .fold-docs {display: inline;}
input.fold + label.stats:before {content: "";}
input.fold:checked + label.stats:before {content: "";}

.hidden {display: none;}
.show-inline {display: inline;}
.hide-inline {display: none;}
input.showhide {display: none;}
input.showhide:checked + i .show-inline {display: none;}
input.showhide:checked + i .hide-inline {display: inline;}
input.showhide:checked ~ span.hidden {display: inline;}
input.showhide:checked ~ div.hidden {display: block;}
input.showhide2:checked ~ span.hidden {display: inline;}

/* code page */

pre.line-numbers {
	counter-reset: line;
}
pre.line-numbers span.codeline {
	counter-increment: line;
}
pre.line-numbers span.codeline:before {
	display: inline-block;
	content: counter(line)"|";
	user-select: none;
	-webkit-user-select: none;
	-moz-user-select: none;
	-ms-user-select: none;
	text-align: right;
	position: absolute;
}

`
