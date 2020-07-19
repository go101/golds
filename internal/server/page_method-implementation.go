package server

import (
	"errors"
	"fmt"
	"net/http"
)

type implPageKey struct {
	pkg string
	typ string
}

func (ds *docServer) methodImplementationPage(w http.ResponseWriter, r *http.Request, pkgPath, typeName string) {
	w.Header().Set("Content-Type", "text/html")

	//log.Println(pkgPath, bareFilename)

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	pageKey := implPageKey{pkg: pkgPath, typ: typeName}
	if ds.implPages[pageKey] == nil {
		result, err := ds.buildImplementationData(pkgPath, typeName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Build implementation info for (", typeName, ") in ", pkgPath, " error: ", err)
			return
		}
		ds.implPages[pageKey] = ds.buildImplementationPage(result)
	}
	w.Write(ds.implPages[pageKey])
}

func (ds *docServer) buildImplementationPage(result *MethodImplementationResult) []byte {
	// some methods are born by embedding other types.
	// Use the same design for local id: click such methods to highlight all same-origin ones.

	return nil
}

type MethodImplementationResult struct {
	TypeName    string
	IsInterface bool
}

func (ds *docServer) buildImplementationData(pkgPath, typeName string) (*MethodImplementationResult, error) {
	return nil, errors.New("not implemented yet")
}
