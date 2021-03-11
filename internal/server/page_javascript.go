package server

import (
	"net/http"
)

func (ds *docServer) javascriptFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "application/javascript")
	if !genDocsMode {
		w.Write(jsFile)
		return
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	page := NewHtmlPage(goldsVersion, "", nil, ds.currentTranslation, pagePathInfo{ResTypeJS, "golds"})
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
	}

	if (document.getElementById("package-details") != null) {
		initPackageDetailsPage();
	}
}

function initOverviewPage() {
	var buttons1 = document.getElementById("buttons1");
	if (buttons1 == null) {
		return;
	}

	var pkgContainer = document.getElementById("packages");
	var nodesPkg = pkgContainer.querySelectorAll(".pkg");
	var pkgsByAlphabet = new Array(nodesPkg.length);
	var pkgsByImportedby = new Array(nodesPkg.length);
	var pkgsByDepDepth = new Array(nodesPkg.length);
	//var pkgsByDepHeight = new Array(nodesPkg.length);
	for (var i = 0; i < nodesPkg.length; i++) {
		var n = nodesPkg[i];
		var t = {
			node: n,
			order: n.querySelector(".order"),
			importedbys: parseInt(n.dataset.importedbys),
			depdepth: parseInt(n.dataset.depdepth),
			depheight: parseInt(n.dataset.depheight),
		};
		pkgsByAlphabet[i] = t;
		pkgsByImportedby[i] = t;
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
	var showSortByDepDepthButton = true;
	for (var i = 0; i < nodesPkg.length; i++) {
		if (pkgsByAlphabet[i].node.id != pkgsByImportedby[i].node.id) {
			showSortByImportBysButton = true;
			break;
		}
	}
	for (var i = 0; i < nodesPkg.length; i++) {
		if (pkgsByAlphabet[i].node.id != pkgsByDepDepth[i].node.id) {
			showSortByDepDepthButton = true;
			break;
		}
	}
	if (!showSortByImportBysButton && !showSortByDepDepthButton) {
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
	autoExpandForPackageDetailsPage();

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

function autoExpandForPackageDetailsPage() {
	const hashChanged = function(newHash) {
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
