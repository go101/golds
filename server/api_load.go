package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// loading page

func (ds *docServer) loadingPage(w http.ResponseWriter, r *http.Request) {
	var pageUrl = r.URL.String()
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="1.5; url=%s">
<title>Analyzing ...</title>
</head>
<body onload="checkLoadProgress();">
<pre>
<code>%s</code>

`,
		pageUrl, ds.currentTranslation.Text_AnalyzingRefresh(pageUrl),
	)

	for _, lm := range ds.loadingLogs {
		fmt.Fprintf(w, `<code id="loading-message-%d">%s</code>
`,
			lm.ID, lm.Message,
		)
	}

	fmt.Fprintf(w, `
</pre>
</body>
</html>`,
	)
}

// api:load
func (ds *docServer) loadAPI(w http.ResponseWriter, r *http.Request) {
	fromIndex, err := strconv.Atoi(r.FormValue("from"))
	if err != nil {
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if fromIndex > len(ds.loadingLogs) {
		fromIndex = len(ds.loadingLogs)
	}

	data, err := json.Marshal(ds.loadingLogs[fromIndex:])
	if err != nil {
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())
	}

	w.Write(data)
}

type LoadingLogMessage struct {
	ID      int
	Message string
}

func (ds *docServer) registerLoadingLogMessage(msg string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.loadingLogs = append(ds.loadingLogs, LoadingLogMessage{len(ds.loadingLogs), msg})
}
