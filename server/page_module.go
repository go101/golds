package server

import (
	"fmt"
	"net/http"
)

func (ds *docServer) modulePage(w http.ResponseWriter, r *http.Request, rootVersion string) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "Module page is not implemented yet")
}
