package server

import (
	"fmt"
	"net/http"
)

func (ds *docServer) modulePage(w http.ResponseWriter, r *http.Request, rootVersion string) {
	// w.WriteHeader(http.StatusTooEarly)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "Module page is not implemented yet")

	if genDocsMode {
		//pkgPath = deHashScope(pkgPath)
	}
}

/*

module page
* packages
* requires
* requiredBy
* project link

*/
