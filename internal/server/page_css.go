package server

import (
	"fmt"
	"net/http"
)

func (ds *docServer) cssFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "text/css")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	theme := ds.themeByName(themeName)

	fmt.Fprint(w, theme.CSS())
	w.Write(commonCSS)
}

var commonCSS = []byte(`



`)
