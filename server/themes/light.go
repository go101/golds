package theme

type Light struct{}

func (*Light) Name() string { return "light" }

func (*Light) CSS() string {
	return `
body {color: #333;}
.grey {color: #ccc;}
a {color: #079;}
a.path-duplicate {color: #9cd;}
.module-version {color: #555; font-style: italic; font-size: smaller; text-decoration: none;}
ol.package-list {line-height: 139%;}
h3 {background: #ddd;}


/* type stat list */
label {cursor: pointer; padding-left: 1px; padding-right: 1px;}
input.stat {display: none;}
input + label + .stat-content {display: none;}
input:checked + label + .stat-content {display: inline;}
input + label:before {content: "+ ";}
input:checked + label:before {content: "- ";}
input:checked + label:after {content: ":";}

.title:after {content: ":";}

/* code page */
pre.line-numbers {
	counter-reset: line;
}
pre.line-numbers span.anchor {
	counter-increment: line;
	margin-left: 44pt;
	tab-size: 7;
	-webkit-tab-size: 7;
	-moz-tab-size: 7;
	-ms-tab-size: 7;
}
pre.line-numbers span.anchor:before {
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

.anchor:target {border-top: 2px solid #D4D4D4; border-bottom: 2px solid #D4D4D4; background-color: #e5eecc;}

code .ident {color: blue;}
code .id-type {color: blue;}
code .id-value {color: blue;}
code .id-function {color: blue;}
code .lit-number {color: #696;}
code .lit-string {color: #a66;}
code .keyword {color: brown;}
code .comment {color: green; font-style: italic;}

#gen-footer {
	padding-top: 5px;
	text-align: center;
	font-size: small;
	color: #555;
	border-top: 1px solid #888;
}

.gold-update {text-align: center; font-size: smaller; background: #eee; padding: 3px;}
.hidden {display: none;}

`
}
