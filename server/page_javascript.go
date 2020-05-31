package server

import (
	"net/http"
)

func (ds *docServer) javascriptFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(jsFile)
}

var jsFile = []byte(`
// ToDo
`)

//function updateUpdateTip() {
//}

//function getStyle(e, styleProp) {
//	if (e.currentStyle)
//		var y = e.currentStyle[styleProp];
//	else if (window.getComputedStyle)
//		var y = document.defaultView.getComputedStyle(e,null).getPropertyValue(styleProp);
//	return y;
//}
//
// use pure css instead now.
//function toggleVisibility(id) {
//	var div = document.getElementById(id);
//	div.style.display = getStyle(div, "display") == "none" ? "inline" : "none";
//}
