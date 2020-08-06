package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"go101.org/gold/internal/server/images"
)

type ImageFile struct {
	base64  string
	decoded []byte
}

var imageFiles = map[string]ImageFile{
	"go101-wechat":  ImageFile{base64: images.Go101WeChat_png},
	"go101-twitter": ImageFile{base64: images.Go101Twitter_png},
}

func (ds *docServer) pngFile(w http.ResponseWriter, r *http.Request, pngFilename string) {

	var pngData []byte

	defer func() {
		if pngData == nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "svg file ", pngFilename, " not found")
			//} else if len(pngData) == 0 {
			//	w.Header().Set("Content-Type", "text/html")
			//	w.WriteHeader(http.StatusTooEarly)
			//	fmt.Fprint(w, "svg file ", pngFilename, " is not ready")
		} else {
			w.Header().Set("Content-Type", "image/png")
			w.Write(pngData)
		}
	}()

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	//if ds.phase < Phase_Analyzed {
	//	pngData = []byte{}
	//	return
	//}

	imgFile, ok := imageFiles[pngFilename]
	if !ok {
		return
	}

	if imgFile.decoded == nil {
		encoded := []byte(strings.TrimSpace(strings.ReplaceAll(imgFile.base64, "\n", "")))
		decoded := make([]byte, len(encoded)*3/4+3)
		n, err := base64.StdEncoding.Decode(decoded, encoded)
		if err != nil {
			panic("decode " + pngFilename + " error: " + err.Error())
		}
		imgFile.decoded = decoded[:n]
	}

	pngData = imgFile.decoded
}
