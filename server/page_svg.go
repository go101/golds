package server

import (
	"net/http"
)

func (ds *docServer) svgFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "image/svg+xml")

}
