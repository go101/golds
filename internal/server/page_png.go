package server

import (
	"encoding/base64"
	"net/http"
	"strings"

	"go101.org/golds/internal/server/images"
)

func (ds *docServer) pngFile(w http.ResponseWriter, r *http.Request, pngFilename string) {
	w.Header().Set("Content-Type", "image/png")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if genDocsMode {
		pngFilename = deHashFilename(pngFilename)
	}

	pageKey := pageCacheKey{
		resType: ResTypePNG,
		res:     pngFilename,
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		data = decodeBase64Data(pngFilename)
		ds.cachePage(pageKey, data)

		// For docs generation.
		page := NewHtmlPage(goldsVersion, "", nil, ds.currentTranslation, createPagePathInfo(ResTypePNG, pngFilename))
		page.Write(data)
		_ = page.Done(w)
	}

	w.Write(data)
}

var imageFiles = map[string]string{
	"go101-wechat":  images.Go101WeChat_png,
	"go101-twitter": images.Go101Twitter_png,
}

func decodeBase64Data(pngFilename string) []byte {
	base64Str, ok := imageFiles[pngFilename]
	if !ok {
		panic("not found image file: " + pngFilename)
	}
	encoded := []byte(strings.TrimSpace(strings.ReplaceAll(base64Str, "\n", "")))
	decoded := make([]byte, len(encoded)*3/4+3)
	n, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		panic(err)
	}
	return decoded[:n]
}
