package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go101.org/golds/code"
)

// loading page
func (ds *docServer) loadingPage(w http.ResponseWriter, r *http.Request) {
	var pageUrl = r.URL.String()

	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
`)

	if r.FormValue("js") != "on" {
		fmt.Fprintf(w, `
<meta http-equiv="refresh" content="1.5; url=%s">
<script>
	function reload() {
		window.location.href="?js=on";
	}

	window.onload=function() {
		setTimeout(reload, 1111)
	}
</script>
`, pageUrl)

	} else {
		fmt.Fprintf(w, `
<script>
	window.onload = function() {
		function createXMLHttpRequest(url, type, params, callback) {
			var xhr = null;
			if (window.XMLHttpRequest) {
				xhr = new XMLHttpRequest();
			} else if (window.ActiveXObject) { // IE5 and IE6
				xhr = new ActiveXObject("Microsoft.XMLHTTP");
			}
			if (xhr != null) {
				if (type.toLowerCase() == 'get') {
					console.log('href',  url + '?' + params);
					xhr.open(type, url + '?' + params);
					xhr.send(null);
				}
				xhr.onreadystatechange = function () {
					if (xhr.readyState == 4) {
						if (xhr.status == 200) {
							callback(JSON.parse(xhr.response))
						} else {
							// todo
						}
					}
				};
			} else {
				console.log("XMLHTTP is not supported");
			}
		}

		var url = window.location.protocol+'//'+window.location.host+'/api:load';
		var page = 'from=';
		var from = 0;
		var code = '';
		var pre = document.getElementsByTagName('pre')[0];
		var timer = ''
		var needjump = false;

		function fromi() {
			for (;;from++){
				if (document.getElementById('loading-message-' + from)) {
					continue;
				}
				break;
			}
		}

		fromi();
		timer = setInterval(function () {
			if (needjump) {
				console.log("jump: ")
				clearInterval(timer);
				window.location.href = window.location.href.split('?')[0];
				return;
			}
			createXMLHttpRequest(url, 'get', page+from, function (data) {
				console.log('data', data);
				for (var i=0;i<data.length;i++){
					code=document.createElement('code');
					code.setAttribute('id','loading-message-'+data[i].ID);
					code.innerHTML=data[i].Message+ "<br/>";
					pre.appendChild(code);

					if (data[i].Message === "") {
						needjump=true;
					}

					console.log("data[i].Message=" + data[i].Message);
					console.log("needjump=" + needjump);
				}
				fromi();
			});
		}, 2000);
	}
</script>
`,
		)
	}

	fmt.Fprintf(w, `	
<title>%s</title>
</head>
<body>
<pre>
<code>%s</code>
`, ds.currentTranslation.Text_Analyzing(), ds.currentTranslation.Text_AnalyzingRefresh(pageUrl),
	)

	for _, lm := range ds.analyzingLogs {
		fmt.Fprintf(w, `
<code id="loading-message-%d">%s</code>`,
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
	fromIndex, _ := strconv.Atoi(r.FormValue("from"))

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if fromIndex > len(ds.analyzingLogs) {
		fromIndex = len(ds.analyzingLogs)
	}

	data, err := json.Marshal(ds.analyzingLogs[fromIndex:])
	if err != nil {
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())
		return
	}

	w.Write(data)
}

type LoadingLogMessage struct {
	ID      int
	Message string
}

func (ds *docServer) onAnalyzingSubTaskDone(task int, d time.Duration, args ...int32) {
	getMsg := func() string {
		var msg string
		switch task {
		case code.SubTask_PreparationDone:
			msg = ds.currentTranslation.Text_Analyzing_PreparationDone(d)
		case code.SubTask_NFilesParsed:
			msg = ds.currentTranslation.Text_Analyzing_NFilesParsed(int(args[0]), d)
		case code.SubTask_ParsePackagesDone:
			msg = ds.currentTranslation.Text_Analyzing_ParsePackagesDone(int(args[0]), d)
		case code.SubTask_CollectPackages:
			msg = ds.currentTranslation.Text_Analyzing_CollectPackages(int(args[0]), d)
		case code.SubTask_CollectModules:
			msg = ds.currentTranslation.Text_Analyzing_CollectModules(int(args[0]), d)
		case code.SubTask_CollectExamples:
			msg = ds.currentTranslation.Text_Analyzing_CollectExamples(d)
		case code.SubTask_SortPackagesByDependencies:
			msg = ds.currentTranslation.Text_Analyzing_SortPackagesByDependencies(d)
		case code.SubTask_CollectDeclarations:
			msg = ds.currentTranslation.Text_Analyzing_CollectDeclarations(d)
		case code.SubTask_CollectRuntimeFunctionPositions:
			msg = ds.currentTranslation.Text_Analyzing_CollectRuntimeFunctionPositions(d)
		case code.SubTask_ConfirmTypeSources:
			msg = ds.currentTranslation.Text_Analyzing_ConfirmTypeSources(d)
		case code.SubTask_CollectSelectors:
			msg = ds.currentTranslation.Text_Analyzing_CollectSelectors(d)
		case code.SubTask_FindImplementations:
			msg = ds.currentTranslation.Text_Analyzing_FindImplementations(d)
		case code.SubTask_RegisterInterfaceMethodsForTypes:
			msg = ds.currentTranslation.Text_Analyzing_RegisterInterfaceMethodsForTypes(d)
		case code.SubTask_MakeStatistics:
			msg = ds.currentTranslation.Text_Analyzing_MakeStatistics(d)
		case code.SubTask_CollectSourceFiles:
			msg = ds.currentTranslation.Text_Analyzing_CollectSourceFiles(d)
		case code.SubTask_CollectObjectReferences:
			msg = ds.currentTranslation.Text_Analyzing_CollectObjectReferences(d)
		case code.SubTask_CacheSourceFiles:
			msg = ds.currentTranslation.Text_Analyzing_CacheSourceFiles(d)
		}
		return msg
	}

	ds.registerAnalyzingLogMessage(getMsg)
}

func (ds *docServer) registerAnalyzingLogMessage(getMsg func() string) {
	var l *log.Logger
	var msg string
	defer func() {
		if l != nil && msg != "" {
			l.Println(msg)
		}
	}()

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	msg = getMsg()
	ds.analyzingLogs = append(ds.analyzingLogs, LoadingLogMessage{len(ds.analyzingLogs), msg})
	if msg != "" && ds.analyzingLogger != nil {
		ds.analyzingLogger.Println(msg)
	}
	return

}
