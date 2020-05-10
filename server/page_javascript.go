package server

import (
	"net/http"
)

func (ds *docServer) javascriptFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(jsFile)
}

var jsFile = []byte(`

function updateUpdateTip() {
	// 1. Check whether or not a div with ID "updating" exists, if false, return.
	// 2. Hide the "to-update", "updating" and "updated" divs.
	// 3. call "GET /update" API. If the return body is
	//    * {"updateStatus": "to-update"}, show the "to-update" div.
	//    * {"updateStatus": "updating"}, show the "updating" div.
	//    * {"updateStatus": "updated"}, show the "updated" div.
	// 4. If the return body is not {"updateStatus": "updating"}, return.
	// 5. Call the API from time to time, until the return body is not {"updateStatus": "updating"}.
	//    Hide the "updating" div, and if the final return is {"updateStatus": "updated"}, show the "updated" div.
}

`)

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
