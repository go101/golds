package themes

type Dark struct{}

func (Dark) Name() string { return "dark" }

func (Dark) CSS() string { return dark_css }

const dark_css = `
body {background: #0d1117; color: #c9d1d9; font-family: {{ .Fonts }};}

a {color: #8899ff;}

.md-text a {color: #c9d1d9;}
.md-text a:hover {color: white;}
.md-text a:visited {color: #abb;}
.md-text a:visited:hover {color: white;}

.b {font-weight: bold;}

.title {font-size: 110%; font-weight: bold;}
.title:after {content: "{{ .Colon }}";}
.title-stat {font-size: medium; font-weight: normal;}

.type-res, .value-res {padding-top: 2px; padding-bottom: 2px;}

.button {border-radius: 3px; padding: 1px 3px;}
.chosen {background: #cca; color: #223; cursor: default;}
.unchosen {}

#footer {
	padding: 5px 8px;
	font-size: small;
	color: #aaa;
	border-top: 1px solid #888;
}

/* overview page */

div.pkg {margin-top: 1px; padding-top: 1px; padding-bottom: 1px;}

a.path-duplicate {color: #185268}

.golds-update {text-align: center; font-size: smaller; background: #eee; padding: 3px;}

div.codelines a.path-duplicate {color: #8899ff;}
div.importedbys a.path-duplicate {color: #8899ff;}
div.depdepth a.path-duplicate {color: #8899ff;}
div.depheight a.path-duplicate {color: #8899ff;}

i.codelines, i.importedbys, i.depdepth, i.depheight {font-size: smaller;}

/* package details page */

span.nodocs {padding-left: 1px; padding-right: 1px;}
span.nodocs:before {content: "--- ";}
label {cursor: pointer; padding-left: 1px; padding-right: 1px;}

input.fold + label:before {content: "[+] ";}
input.fold:checked + label:before {content: "[-] ";}
input.fold:checked + label.fold-items:after {content: "{{ .Colon }}";}

/* code page */

#header {
	padding-bottom: 8px;
	border-bottom: 1px solid #888;
}

hr {color: #888;}

pre.line-numbers span.codeline {
	margin-left: 44pt;
	tab-size: 7;
	-webkit-tab-size: 7;
	-moz-tab-size: 7;
	-ms-tab-size: 7;
}
pre.line-numbers span.codeline:before {
	width: 40pt;
	left: 8pt;
	padding: 0 3pt 0 0;
	border-right: 0;
}

.anchor {}
.codeline {}

.codeline:target, .anchor:target {
	border-top: 1px solid #3d4b55;
	border-bottom: 1px solid #3d4b55;
	background-color: #2d3a44;
}

code .ident {color: #d1d8aa;}
code .id-type {color: #d1d8aa;}
code .id-value {color: #d1d8aa;}
code .id-function {color: #d1d8aa;}
code .lit-number {color: #a9d1a4;}
code .lit-string {color: #a9d1a4;}
code .keyword {color: #ff7b72;}
code .comment {color: #aaa; font-style: italic;}

// These lines are parsed and used in code.
// Please keep each of them in a seperated line and
// keep the "xxxxx {" prefixes unchanged.
code.chosen-ident {background: #eea; color: #115;}
code.chosen-id-import {background: brown; color: #eed;}

`
