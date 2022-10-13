package theme

type Light struct{}

func (*Light) Name() string { return "light" }

func (*Light) CSS() string {
	return `
body {background: #fff; color: #333; font-family: {{ .Fonts }};}
.grey {color: #ccc;}
a {color: #079;}
.module-version {color: #555; font-style: italic; font-size: smaller; text-decoration: none;}
ol.package-list {line-height: 139%;}
h3 {background: #ddd;}

.md-text a {color: #333;}
.md-text a:hover {color: black;}
.md-text a:visited {color: #666;}
.md-text a:visited:hover {color: black;}

.b {font-weight: bold;}

.title {font-size: 110%; font-wieght: bold;}
.title:after {content: "{{ .Colon }}";}
.title-stat {font-size: medium; font-wieght: normal;}

.type-res, .value-res {padding-top: 2px; padding-bottom: 2px;}

.js-on {display: none;}

.button {border-radius: 3px; padding: 1px 3px;}
.chosen {background: #226; color: #ff8; cursor: default;}
.unchosen {}

#footer {
	padding: 5px 8px;
	font-size: small;
	color: #555;
	border-top: 1px solid #888;
}



/* overview page */

div.pkg {margin-top: 1px; padding-top: 1px; padding-bottom: 1px;}

a.path-duplicate {color: #9cd;}

.golds-update {text-align: center; font-size: smaller; background: #eee; padding: 3px;}

.pkg-summary {display: none;}
input#toggle-summary {display: none;}
input#toggle-summary:checked ~ div .pkg-summary {display: inline;}

div.alphabet .importedbys {display: none;}
div.alphabet .codelines {display: none;}
div.alphabet .depdepth {display: none;}
div.alphabet .depheight {display: none;}
div.importedbys .importedbys {display: inline;}
div.importedbys .codelines {display: none;}
div.importedbys .depdepth {display: none;}
div.importedbys .depheight {display: none;}
div.depdepth .depdepth {display: inline;}
div.depdepth .codelines {display: none;}
div.depdepth .importedbys {display: none;}
div.depdepth .depheight {display: none;}
div.depheight .depheight {display: inline;}
div.depheight .codelines {display: none;}
div.depheight .importedbys {display: none;}
div.depheight .depdepth {display: none;}
div.codelines .codelines {display: inline;}
div.codelines .depheight {display: none;}
div.codelines .importedbys {display: none;}
div.codelines .depdepth {display: none;}

div.codelines a.path-duplicate {color: #079;}
div.importedbys a.path-duplicate {color: #079;}
div.depdepth a.path-duplicate {color: #079;}
div.depheight a.path-duplicate {color: #079;}

i.codelines, i.importedbys, i.depdepth, i.depheight {font-size: smaller;}

/* package details page */

div:target {display: block;}
span.nodocs {padding-left: 1px; padding-right: 1px;}
span.nodocs:before {content: ". ";}
label {cursor: pointer; padding-left: 1px; padding-right: 1px;}
input.fold {display: none;}
/*input.fold + label +*/ .fold-items {display: none;}
/*input.fold + label +*/ .fold-docs {display: none;}
input.fold + label:before {content: "▶ ";}
input.fold:checked + label:before {content: "▼ ";}
input.fold:checked + label.fold-items:after {content: "{{ .Colon }}";}
input.fold:checked + label + .fold-items {display: inline;}
input.fold:checked + label + .fold-docs {display: inline;}
input.fold + label.stats:before {content: "";}
input.fold:checked + label.stats:before {content: "";}

.hidden {display: none;}
.show-inline {display: inline;}
.hide-inline {display: none;}
input.showhide {display: none;}
input.showhide:checked + i .show-inline {display: none;}
input.showhide:checked + i .hide-inline {display: inline;}
input.showhide:checked ~ span.hidden {display: inline;}
input.showhide:checked ~ div.hidden {display: block;}
input.showhide2:checked ~ span.hidden {display: inline;}

/* code page */

#header {
	padding-bottom: 8px;
	border-bottom: 1px solid #888;
}

hr {color: #888;}

pre.line-numbers {
	counter-reset: line;
}
pre.line-numbers span.codeline {
	counter-increment: line;
	margin-left: 44pt;
	tab-size: 7;
	-webkit-tab-size: 7;
	-moz-tab-size: 7;
	-ms-tab-size: 7;
}
pre.line-numbers span.codeline:before {
	display: inline-block;
	text-align:right;
	position: absolute;
	width: 40pt;
	left: 8pt;
	padding: 0 3pt 0 0;
	border-right: 0;
	content: counter(line)"|";
	user-select: none;
	-webkit-user-select: none;
	-moz-user-select: none;
	-ms-user-select: none;
}

.anchor {}
.codeline {}

.codeline:target, .anchor:target {border-top: 1px solid #d5ddbb; border-bottom: 1px solid #d5ddbb; background-color: #e5eecc;}

code .ident {color: blue;}
code .id-type {color: blue;}
code .id-value {color: blue;}
code .id-function {color: blue;}
code .lit-number {color: #e66;}
code .lit-string {color: #a66;}
code .keyword {color: brown;}
code .comment {color: green; font-style: italic;}

`
}
