package theme

type Light struct{}

func (*Light) Name() string { return "light" }

func (*Light) CSS() string {
	return `
body {color: #333; font-family: {{ .Fonts }};}
.grey {color: #ccc;}
a {color: #079;}
a.path-duplicate {color: #9cd;}
.module-version {color: #555; font-style: italic; font-size: smaller; text-decoration: none;}
ol.package-list {line-height: 139%;}
h3 {background: #ddd;}

.b {font-weight: bold;}

.title:after {content: "{{ .Colon }}";}

.type-res, .value-res {padding-top: 2px; padding-bottom: 2px;}

/* content folding */
div:target {display: block;}
span.nodocs {padding-left: 1px; padding-right: 1px;}
span.nodocs:before {content: ". ";}
label {cursor: pointer; padding-left: 1px; padding-right: 1px;}
input.fold {display: none;}
/*input.fold + label +*/ .fold-items {display: none;}
/*input.fold + label +*/ .fold-docs {display: none;}
input.fold + label:before {content: "+ ";}
input.fold:checked + label + .fold-items {display: inline;}
input.fold:checked + label + .fold-docs {display: inline;}
input.fold:checked + label:before {content: "- ";}
input.fold:checked + label.fold-items:after {content: "{{ .Colon }}";}

.hidden {display: none;}
.show-inline {display: inline;}
.hide-inline {display: none;}
input.showhide {display: none;}
input.showhide:checked + i .show-inline {display: none;}
input.showhide:checked + i .hide-inline {display: inline;}
input.showhide:checked ~ span.hidden {display: inline;}
input.showhide:checked ~ div.hidden {display: block;}

/*
#pkg-doc {
	position: fixed;
	top: 0;
	height: 100%;
	width: 80%;
	background: #fcc;
}
#pkg-doc:not(:target) {
	right: -100%;
	transition: right .5s;
}
#pkg-doc:target {
	right: 0;
	transition: right 1s;
}
*/

/* code page */
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

hr {color: #888;}

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

#header {
	padding-bottom: 8px;
	border-bottom: 1px solid #888;
}

#footer {
	padding: 5px 8px;
	font-size: small;
	color: #555;
	border-top: 1px solid #888;
}

.golds-update {text-align: center; font-size: smaller; background: #eee; padding: 3px;}

`
}
