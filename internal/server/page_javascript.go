package server

import (
	"net/http"
)

func (ds *docServer) javascriptFile(w http.ResponseWriter, r *http.Request, filename string) {
	w.Header().Set("Content-Type", "application/javascript")
	if !genDocsMode {
		w.Write(jsFile)
		return
	}

	if genDocsMode {
		filename = deHashFilename(filename) // not used now
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	page := NewHtmlPage(goldsVersion, "", nil, ds.currentTranslation, createPagePathInfo(ResTypeJS, "golds"))
	page.Write(jsFile)
	_ = page.Done(w)
}

var jsFile = []byte(`
function todo() {
	const urlParams = new URLSearchParams(window.location.search);
	var name = urlParams.get('name');
	// pase name to get method/field/... if provided.
	var line = urlParams.get('line');
	var id = urlParams.get('id');
	var segments = urlParams.get('segments');
}

function onPageLoad() {
	var lastClicked;
	var lastClickedBorder;
	document.addEventListener("click", function(e){
		if (e.target.tagName == "A") {
			if (lastClicked != e.target) {
				if (lastClicked != null) {
					lastClicked.style.border = lastClickedBorder;
				}
				lastClicked = e.target;
				lastClickedBorder = lastClicked.style.border;
				lastClicked.style.border = "medium dashed #079";
			}
		}
		e.stopPropagation();
	});

	if (document.getElementById("overview") != null) {
		initOverviewPage();
		return
	}

	document.addEventListener("keydown", function(e){
		if (e.ctrlKey || e.altKey || e.shiftKey) {
			return;
		}
		var key = e.key || e.which || e.keyCode;
		if (key == 35) { // HOME
			if (document.body.scrollTop == 0) {
				// ToDo: jump to overview page.
				//       The implementation is different between sever and doc generation modes.
			}
		}
	});

	if (document.getElementById("package-details") != null) {
		initPackageDetailsPage();
	}
}

function initOverviewPage() {
	document.addEventListener("keydown", function(e){
		if (e.ctrlKey || e.altKey || e.shiftKey) {
			return;
		}
		var key = e.key || e.which || e.keyCode;
		if (key == 'd') {
			var toggleSummary = document.querySelector("#toggle-summary");
			toggleSummary.checked = !toggleSummary.checked
		}
	});

	var buttons1 = document.getElementById("buttons1");
	if (buttons1 == null) {
		return;
	}

	var pkgContainer = document.getElementById("packages");
	var nodesPkg = pkgContainer.querySelectorAll(".pkg");
	var pkgsByAlphabet = new Array(nodesPkg.length);
	var pkgsByImportedby = new Array(nodesPkg.length);
	var pkgsByCodeLines = new Array(nodesPkg.length);
	var pkgsByDepDepth = new Array(nodesPkg.length);
	//var pkgsByDepHeight = new Array(nodesPkg.length);
	for (var i = 0; i < nodesPkg.length; i++) {
		var n = nodesPkg[i];
		var t = {
			node: n,
			order: n.querySelector(".order"),
			importedbys: parseInt(n.dataset.importedbys),
			codelines: parseInt(n.dataset.loc),
			depdepth: parseInt(n.dataset.depdepth),
			depheight: parseInt(n.dataset.depheight),
		};
		pkgsByAlphabet[i] = t;
		pkgsByImportedby[i] = t;
		pkgsByCodeLines[i] = t;
		pkgsByDepDepth[i] = t;
	}
	pkgsByImportedby.sort(function(a, b) {
		if (a.importedbys == b.importedbys) {
			if (a.node.id < b.node.id) {
				return -1;
			}
			return 1;
		}
		return b.importedbys - a.importedbys;
	});
	pkgsByCodeLines.sort(function(a, b) {
		if (a.codelines == b.codelines) {
			if (a.node.id < b.node.id) {
				return -1;
			}
			return 1;
		}
		return b.codelines - a.codelines;
	});
	pkgsByDepDepth.sort(function(a, b) {
		if (a.depdepth == b.depdepth) {
			if (a.node.id < b.node.id) {
				return -1;
			}
			return 1;
		}
		if (a.depdepth < b.depdepth) {
			return -1;
		}
		return 1;
	});

	var showSortByImportBysButton = true;
	var showSortByCodeLinesButton = true;
	var showSortByDepDepthButton = true;
	for (var i = 0; i < nodesPkg.length; i++) {
		if (pkgsByAlphabet[i].node.id != pkgsByImportedby[i].node.id) {
			showSortByImportBysButton = true;
			break;
		}
	}
	for (var i = 0; i < nodesPkg.length; i++) {
		if (pkgsByAlphabet[i].node.id != pkgsByCodeLines[i].node.id) {
			showSortByCodeLinesButton = true;
			break;
		}
	}
	for (var i = 0; i < nodesPkg.length; i++) {
		if (pkgsByAlphabet[i].node.id != pkgsByDepDepth[i].node.id) {
			showSortByDepDepthButton = true;
			break;
		}
	}
	if (!showSortByImportBysButton && !showSortByCodeLinesButton && !showSortByDepDepthButton) {
		return;
	}

	var SPACES = "          ";
	var maxDigitCount = parseInt(document.getElementById("max-digit-count").innerText);
	var pkgStartOrderId = 1;
	var content1 = buttons1.querySelector(".buttons-content");
	var content = content1;
	var wdPkgContainer = document.getElementById("wd-packages");
	if (wdPkgContainer == null) {
		buttons1.style.display = "inline";
	} else {
		pkgStartOrderId += wdPkgContainer.querySelectorAll(".pkg").length;

		var buttons2 = document.getElementById("buttons2");
		var content2 = buttons2.querySelector(".buttons-content");
		content2.innerHTML = content1.innerHTML;
		content1.innerHTML = "";
		content = content2;

		buttons2.style.display = "inline";
	}
	if (!showSortByImportBysButton) {
		content.querySelector("#importedbys").display = "none";
	}
	if (!showSortByDepDepthButton) {
		content.querySelector("#depdepth").display = "none";
	}

	var sortByAlphabet = content.querySelector("#btn-alphabet");
	var sortByImportedbys = content.querySelector("#btn-importedbys");
	var sortByCodeLines = content.querySelector("#btn-codelines");
	var sortByDepdepth = content.querySelector("#btn-depdepth");

	sortByAlphabet.classList.add("chosen");
	var currentSortBy = "alphabet";
	var currentButton = sortByAlphabet;

	sortByAlphabet.addEventListener('click', function(event) {
		if (currentSortBy == "alphabet") {
			return;
		}

		pkgsByAlphabet.forEach(function (x, i) {
			pkgContainer.appendChild(x.node);
			var o = (i+pkgStartOrderId).toString();
			x.order.innerText = SPACES.substr(0, maxDigitCount-o.length) + o;
		});

		pkgContainer.classList.remove(currentSortBy);
		currentSortBy = "alphabet";
		pkgContainer.classList.add(currentSortBy);

		currentButton.classList.remove("chosen");
		currentButton = sortByAlphabet;
		currentButton.classList.add("chosen");
	});
	sortByImportedbys.addEventListener('click', function(event) {
		if (currentSortBy == "importedbys") {
			return;
		}

		pkgContainer.innerHTML = "";
		pkgsByImportedby.forEach(function (x, i) {
			pkgContainer.appendChild(x.node);
			var o = (i+pkgStartOrderId).toString();
			x.order.innerText = SPACES.substr(0, maxDigitCount-o.length) + o;
		});

		pkgContainer.classList.remove(currentSortBy);
		currentSortBy = "importedbys";
		pkgContainer.classList.add(currentSortBy);

		currentButton.classList.remove("chosen");
		currentButton = sortByImportedbys;
		currentButton.classList.add("chosen");
	});
	sortByCodeLines.addEventListener('click', function(event) {
		if (currentSortBy == "codelines") {
			return;
		}

		pkgContainer.innerHTML = "";
		pkgsByCodeLines.forEach(function (x, i) {
			pkgContainer.appendChild(x.node);
			var o = (i+pkgStartOrderId).toString();
			x.order.innerText = SPACES.substr(0, maxDigitCount-o.length) + o;
		});

		pkgContainer.classList.remove(currentSortBy);
		currentSortBy = "codelines";
		pkgContainer.classList.add(currentSortBy);

		currentButton.classList.remove("chosen");
		currentButton = sortByCodeLines;
		currentButton.classList.add("chosen");
	});
	sortByDepdepth.addEventListener('click', function(event) {
		if (currentSortBy == "depdepth") {
			return;
		}

		pkgContainer.innerHTML = "";
		pkgsByDepDepth.forEach(function (x, i) {
			pkgContainer.appendChild(x.node);
			var o = (i+pkgStartOrderId).toString();
			x.order.innerText = SPACES.substr(0, maxDigitCount-o.length) + o;
		});

		pkgContainer.classList.remove(currentSortBy);
		currentSortBy = "depdepth";
		pkgContainer.classList.add(currentSortBy);

		currentButton.classList.remove("chosen");
		currentButton = sortByDepdepth;
		currentButton.classList.add("chosen");
	});
}

function initPackageDetailsPage() {
	autoExpandForPackageDetailsPageByPageAnchor();

	var toggleCheckboxes = function(cbs) {
		var numCheckeds = 0;
		for (var i = 0; i < cbs.length; i++) {
			if (cbs[i].checked) {
				numCheckeds++;
			}
		}
		for (var i = 0; i < cbs.length; i++) {
			cbs[i].checked = numCheckeds != cbs.length;
		}
	}
	document.addEventListener("keydown", function(e){
		if (e.ctrlKey || e.altKey || e.shiftKey) {
			return;
		}

		var cbsFile = [], cbsExample = [], cbsTypes = [], cbsFuncs = [], cbsVars = [], cbsConsts = [];

		var key = e.key || e.which || e.keyCode;
		if (key == 'p' || key == 'a') {
			var files = document.getElementById("files");
			if (files != null) {
				if (key == 'p') {files.scrollIntoView();}
				cbsFile = files.querySelectorAll("input[type='checkbox']");
			}
		}
		if (key == 'e' || key == 'a') {
			var examples = document.getElementById("examples");
			if (examples != null) {
				if (key == 'e') {examples.scrollIntoView();}
				cbsExample = examples.querySelectorAll("input[type='checkbox']");
			}
		}
		if (key == 't' || key == 'a') {
			var types = document.getElementById("exported-types");
			if (types != null) {
				if (key == 't') {types.scrollIntoView();}
				cbsTypes = types.querySelectorAll(".type-res > input[type='checkbox']");
			}
		}
		if (key == 'f' || key == 'a') {
			var funcs = document.getElementById("exported-functions");
			if (funcs != null) {
				if (key == 'f') {funcs.scrollIntoView();}
				cbsFuncs = funcs.querySelectorAll(".value-res > input[type='checkbox']");
			}
		}
		if (key == 'v' || key == 'a') {
			var vars = document.getElementById("exported-variables");
			if (vars != null) {
				if (key == 'v') {vars.scrollIntoView();}
				cbsVars = vars.querySelectorAll(".value-res > input[type='checkbox']");
			}
		}
		if (key == 'c' || key == 'a') {
			var consts = document.getElementById("exported-constants");
			if (consts != null) {
				if (key == 'c') {consts.scrollIntoView();}
				cbsConsts = consts.querySelectorAll(".value-res > input[type='checkbox']");
			}
		}
		var cbsAll = new Array(cbsFile.length + cbsExample.length + cbsTypes.length + cbsFuncs.length + cbsVars.length + cbsConsts.length).slice(0, 0);
		cbsAll.push(...cbsFile);
		cbsAll.push(...cbsExample);
		cbsAll.push(...cbsTypes);
		cbsAll.push(...cbsFuncs);
		cbsAll.push(...cbsVars);
		cbsAll.push(...cbsConsts);
		toggleCheckboxes(cbsAll);
	});

	var buttons = document.getElementById("exported-types-buttons");
	var container = document.getElementById("exported-types");
	var nodesTypeRes = container.querySelectorAll(".type-res");
	var typesByAlphabet = new Array(nodesTypeRes.length);
	var typesByPopularity = new Array(nodesTypeRes.length);
	for (var i = 0; i < nodesTypeRes.length; i++) {
		var n = nodesTypeRes[i];
		var t = {node: n, popularity: parseInt(n.dataset.popularity)};
		typesByAlphabet[i] = t;
		typesByPopularity[i] = t;
	}
	typesByPopularity.sort(function(a, b) {
		if (a.popularity == b.popularity) {
			if (a.node.id < b.node.id) {
				return -1;
			}
			return 1;
		}
		return b.popularity - a.popularity;
	});

	//var printArray = function(a, title) {
	//	console.log("==================== ", title);
	//	for (var i = 0; i < a.length; i++) {
	//		console.log(i, ": ", a[i].popularity, ", ", a[i].node.id);
	//	}
	//}
	//printArray(typesByAlphabet, "typesByAlphabet");
	//printArray(typesByPopularity, "typesByPopularity");

	var showSortingButtons = false;
	for (var i = 0; i < nodesTypeRes.length; i++) {
		if (typesByAlphabet[i].node.id != typesByPopularity[i].node.id) {
			showSortingButtons = true;
			break;
		}
	}
	if (!showSortingButtons) {
		return;
	}

	//showJavaScriptRelatedElements(buttons);
	buttons.style.display = "block";

	var currentSortBy = "alphabet";
	var sortByAlphabet = buttons.querySelector("#sort-types-by-alphabet");
	var sortByPopularity = buttons.querySelector("#sort-types-by-popularity");
	sortByAlphabet.classList.add("chosen");
	sortByAlphabet.addEventListener('click', function(event) {
		if (currentSortBy == "alphabet") {
			return;
		}

		typesByAlphabet.forEach(function (x) {
			container.appendChild(x.node);
		});

		currentSortBy = "alphabet";
		sortByAlphabet.classList.add("chosen");
		sortByPopularity.classList.remove("chosen");
	});
	sortByPopularity.addEventListener('click', function(event) {
		if (currentSortBy == "popularity") {
			return;
		}

		typesByPopularity.forEach(function (x) {
			container.appendChild(x.node);
		});

		currentSortBy = "popularity";
		sortByPopularity.classList.add("chosen");
		sortByAlphabet.classList.remove("chosen");
	});
}

function autoExpandForPackageDetailsPageByPageAnchor() {
	const hashChanged = function(newHash) {
		if (newHash.length < 1) {
			return
		}
		var div = document.getElementById(newHash.substr(1));
		if (div == null) {
			return;
		}
		const prefix = "#name-";
		if (newHash.indexOf(prefix) != 0) {
			return;
		}
		var checkbox = document.getElementById(newHash.substr(prefix.length)+"-fold-content");
		if (checkbox == null) {
			return;
		}
		checkbox.checked = true;
	};

	if ("onhashchange" in window) {
		window.onhashchange = function () {
			hashChanged(window.location.hash);
		}
	} else {
		var storedHash = window.location.hash;
		window.setInterval(function () {
			if (window.location.hash != storedHash) {
				storedHash = window.location.hash;
				hashChanged(storedHash);
			}
		}, 100);
	}
	hashChanged(window.location.hash);
}


`)
