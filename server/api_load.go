package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go101.org/gold/code"
)

// loading page
func (ds *docServer) loadingPage(w http.ResponseWriter, r *http.Request) {
	var pageUrl = r.URL.String()
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="1.5; url=%s">
<title>%s</title>
</head>
<body onload="checkLoadProgress();">
<pre>
<code>%s</code>

`,
		pageUrl, ds.currentTranslation.Text_Analyzing(), ds.currentTranslation.Text_AnalyzingRefresh(pageUrl),
	)

	for _, lm := range ds.analyzingLogs {
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

	if fromIndex > len(ds.analyzingLogs) {
		fromIndex = len(ds.analyzingLogs)
	}

	data, err := json.Marshal(ds.analyzingLogs[fromIndex:])
	if err != nil {
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())
	}

	w.Write(data)
}

type LoadingLogMessage struct {
	ID      int
	Message string
}

func (ds *docServer) onAnalyzingSubTaskDone(task int, d time.Duration, args ...int32) {
	var msg string
	switch task {
	default:
		return
	case code.SubTask_PreparationDone:
		msg = ds.currentTranslation.Text_Analyzing_PreparationDone(d)
	case code.SubTask_NFilesParsed:
		msg = ds.currentTranslation.Text_Analyzing_NFilesParsed(int(args[0]), d)
	case code.SubTask_ParsePackagesDone:
		msg = ds.currentTranslation.Text_Analyzing_ParsePackagesDone(int(args[0]), int(args[1]), d)
	case code.SubTask_CollectPackages:
		msg = ds.currentTranslation.Text_Analyzing_CollectPackages(int(args[0]), d)
	case code.SubTask_SortPackagesByDependencies:
		msg = ds.currentTranslation.Text_Analyzing_SortPackagesByDependencies(d)
	case code.SubTask_CollectDeclarations:
		msg = ds.currentTranslation.Text_Analyzing_CollectDeclarations(d)
	case code.SubTask_CollectRuntimeFunctionPositions:
		msg = ds.currentTranslation.Text_Analyzing_CollectRuntimeFunctionPositions(d)
	case code.SubTask_FindTypeSources:
		msg = ds.currentTranslation.Text_Analyzing_FindTypeSources(d)
	case code.SubTask_CollectSelectors:
		msg = ds.currentTranslation.Text_Analyzing_CollectSelectors(d)
	case code.SubTask_CheckCollectedSelectors:
		msg = ds.currentTranslation.Text_Analyzing_CheckCollectedSelectors(d)
	case code.SubTask_FindImplementations:
		msg = ds.currentTranslation.Text_Analyzing_FindImplementations(d)
	case code.SubTask_MakeStatistics:
		msg = ds.currentTranslation.Text_Analyzing_MakeStatistics(d)
	case code.SubTask_CollectSourceFiles:
		msg = ds.currentTranslation.Text_Analyzing_CollectSourceFiles(d)
	}

	ds.registerAnalyzingLogMessage(msg)
}

func (ds *docServer) registerAnalyzingLogMessage(msg string) {
	var l *log.Logger
	defer func() {
		if l != nil {
			l.Println(msg)
		}
	}()

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.analyzingLogs = append(ds.analyzingLogs, LoadingLogMessage{len(ds.analyzingLogs), msg})
	l = ds.analyzingLogger
	return

}
