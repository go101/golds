package server

import (
	"net/http"
)

func (ds *docServer) svgFile(w http.ResponseWriter, r *http.Request, themeName string) {
	w.Header().Set("Content-Type", "image/svg+xml")

}

/*

<svg width="580" height="400" xmlns="http://www.w3.org/2000/svg">
 <g>
  <rect fill="#fff" id="canvas_background" height="402" width="582" y="-1" x="-1"/>
  <g display="none" overflow="visible" y="0" x="0" height="100%" width="100%" id="canvasGrid">
   <rect fill="url(#gridpattern)" stroke-width="0" y="0" x="0" height="100%" width="100%"/>
  </g>
 </g>
 <g>
  <rect id="svg_1" height="61" width="53" y="95.4375" x="122.5" stroke-width="1.5" stroke="#000" fill="#fff"/>
  <ellipse ry="24.5" rx="19.5" id="svg_2" cy="226.9375" cx="113" stroke-width="1.5" stroke="#000" fill="#fff"/>
  <ellipse ry="28" rx="64" id="svg_3" cy="253.4375" cx="218.5" fill-opacity="null" stroke-opacity="null" stroke-width="1.5" stroke="#000" fill="#fff"/>
  <path id="svg_4" d="m299.997513,115.459483l53.999946,-29.411307l54.000073,29.411307l-20.626099,47.588695l-66.747759,0l-20.62616,-47.588695z" fill-opacity="null" stroke-opacity="null" stroke-width="1.5" stroke="#000" fill="#fff"/>
  <text stroke="#000" transform="matrix(13.87485408782959,0,0,1,-6551.49020434916,0) " xml:space="preserve" text-anchor="start" font-family="Helvetica, Arial, sans-serif" font-size="24" id="svg_5" y="304.4375" x="495.5" fill-opacity="null" stroke-opacity="null" stroke-width="0" fill="#000000"/>
  <text xml:space="preserve" text-anchor="start" font-family="Helvetica, Arial, sans-serif" font-size="24" id="svg_6" y="312.4375" x="281.5" fill-opacity="null" stroke-opacity="null" stroke-width="0" stroke="#000" fill="#000000">hello 你好</text>
  <line stroke-linecap="null" stroke-linejoin="null" id="svg_7" y2="238.4375" x2="431.5" y1="160.4375" x1="240.5" fill-opacity="null" stroke-opacity="null" stroke="#000" fill="none"/>
  <line stroke-linecap="null" stroke-linejoin="null" id="svg_8" y2="268.4375" x2="426.5" y1="154.4375" x1="484.5" fill-opacity="null" stroke-opacity="null" stroke="#000" fill="none"/>
 </g>
</svg>

*/
