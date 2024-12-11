package themes

type Light struct{}

func (Light) Name() string { return "light" }

func (Light) CSS() string { return light_css }

const light_css = `
body {background: #fff; color: #333; font-family: {{ .Fonts }};}

a {color: #079;}

.md-text a {color: #333;}
.md-text a:hover {color: black;}
.md-text a:visited {color: #666;}
.md-text a:visited:hover {color: black;}

.b {font-weight: bold;}

.title {font-size: 110%; font-weight: bold;}
.title:after {content: "{{ .Colon }}";}
.title-stat {font-size: medium; font-weight: normal;}

.type-res, .value-res {padding-top: 2px; padding-bottom: 2px;}

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

a.path-duplicate {color: #bdf;}

.golds-update {text-align: center; font-size: smaller; background: #eee; padding: 3px;}

div.codelines a.path-duplicate {color: #079;}
div.importedbys a.path-duplicate {color: #079;}
div.depdepth a.path-duplicate {color: #079;}
div.depheight a.path-duplicate {color: #079;}

i.codelines, i.importedbys, i.depdepth, i.depheight {font-size: smaller;}

/* package details page */

span.nodocs {padding-left: 1px; padding-right: 1px;}
span.nodocs:before {content: " -  ";}
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
	border-top: 1px solid #d5ddbb;
	border-bottom: 1px solid #d5ddbb;
	background-color: #e5eecc;
}

code .ident {color: blue;}
code .id-type {color: blue;}
code .id-value {color: blue;}
code .id-function {color: blue;}
code .lit-number {color: #e66;}
code .lit-string {color: #a66;}
code .keyword {color: brown;}
code .comment {color: green; font-style: italic;}


// These lines are parsed and used in code.
// Please keep each of them in a seperated line and
// keep the "xxxxx {" prefixes unchanged.
code.chosen-ident {background: #226; color: #ff8;}
code.chosen-id-import {background: brown; color: #eed;}

`
