package server

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

func (ds *docServer) svgFile(w http.ResponseWriter, r *http.Request, svgFile string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusTooEarly)
		fmt.Fprint(w, "svg file ", svgFile, " is not ready")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")

	pageKey := pageCacheKey{
		resType: ResTypeSVG,
		res:     svgFile,
	}
	data, ok := ds.cachedPage(pageKey)
	if !ok {
		data = ds.buildSVG(svgFile)
		ds.cachePage(pageKey, data)

		// For docs generation.
		page := NewHtmlPage(ds.goldVersion, "", ds.currentTheme.Name(), pagePathInfo{ResTypeSVG, svgFile}, false)
		page.Write(data)
		_ = page.Done(ds.currentTranslation, w)
	}
	w.Write(data)
}

func (ds *docServer) buildSVG(svgFile string) (svgData []byte) {
	xName := func(max int) func(int) string {
		return func(i int) string {
			if i == max {
				return fmt.Sprintf("(%d+)", i)
			} else {
				return strconv.Itoa(i)
			}
		}
	}

	xNameFromOne := func(max int) func(int) string {
		return func(i int) string {
			i++
			if i == max {
				return fmt.Sprintf("(%d+)", i)
			} else {
				return strconv.Itoa(i)
			}
		}
	}

	kindName := func(i int) string {
		k := reflect.Kind(i + 1)
		switch k {
		default:
			return reflect.Kind(k).String()
		case reflect.Array:
			return "[...]T"
		case reflect.Slice:
			return "[ ]T"
		case reflect.Ptr:
			return "*T"
		}
	}

	stats := ds.analyzer.Statistics()
	chartTitle := ds.currentTranslation.Text_ChartTitle(svgFile)
	switch svgFile {
	case "gosourcefiles-by-imports":
		svgData = createSourcefileImportsSVG(chartTitle, stats.FilesByImportCount[:], xName(len(stats.FilesByImportCount)-1))
	case "packages-by-dependencies":
		svgData = createSourcefileImportsSVG(chartTitle, stats.PackagesByDeps[:], xName(len(stats.PackagesByDeps)-1))
	case "exportedtypenames-by-kinds":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedTypeNamesByKind[1:], kindName)
	case "exportedstructtypes-by-embeddingfields":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByEmbeddingFieldCount[:], xName(len(stats.ExportedNamedStructsByEmbeddingFieldCount)-1))
	//case "exportedstructtypes-by-allfields":
	//	svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByFieldCount[:], xName(len(stats.ExportedNamedStructsByFieldCount)-1))
	case "exportedstructtypes-by-explicitfields":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByExplicitFieldCount[:], xName(len(stats.ExportedNamedStructsByExplicitFieldCount)-1))
	//case "exportedstructtypes-by-exportedfields":
	//	svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByExportedFieldCount[:], xName(len(stats.ExportedNamedStructsByExportedFieldCount)-1))
	case "exportedstructtypes-by-exportedexplicitfields":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByExportedExplicitFieldCount[:], xName(len(stats.ExportedNamedStructsByExportedExplicitFieldCount)-1))
	case "exportedstructtypes-by-exportedpromotedfields":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedStructsByExportedPromotedFieldCount[:], xName(len(stats.ExportedNamedStructsByExportedPromotedFieldCount)-1))
	case "exportedfunctions-by-parameters":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedFunctionsByParameterCount[:], xName(len(stats.ExportedFunctionsByParameterCount)-1))
	case "exportedfunctions-by-results":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedFunctionsByResultCount[:], xName(len(stats.ExportedFunctionsByResultCount)-1))
	case "exportedidentifiers-by-lengths":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedIdentifiersByLength[1:], xNameFromOne(len(stats.ExportedIdentifiersByLength)-1))
	case "exportednoninterfacetypes-by-exportedmethods":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedNonInterfaceTypesByExportedMethodCount[:], xName(len(stats.ExportedNamedNonInterfaceTypesByExportedMethodCount)-1))
	case "exportedvariables-by-typekinds":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedVariablesByTypeKind[1:], kindName)
	case "exportedconstants-by-typekinds":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedConstantsByTypeKind[1:], kindName)
	case "exportedinterfacetypes-by-exportedmethods":
		svgData = createSourcefileImportsSVG(chartTitle, stats.ExportedNamedInterfacesByExportedMethodCount[:], xName(len(stats.ExportedNamedInterfacesByExportedMethodCount)-1))
	default:
	}

	return
}

// ToDo: add a bgColor parameter.
func createSourcefileImportsSVG(title string, stat []int32, xName func(i int) string) []byte {
	if xName == nil {
		xName = func(i int) string {
			return strconv.Itoa(i)
		}
	}

	maxV := int32(0)
	n := 0
	for i, v := range stat {
		if v > maxV {
			maxV = v
		}
		if v != 0 {
			n = i + 1
		}
	}
	stat = stat[:n]

	barCount := 0
	for i := 0; i < len(stat); i++ {
		barCount++
		v := stat[i]
		if v == 0 {
			k := i + 1
			for ; k < len(stat); k++ {
				if stat[k] != 0 {
					break
				}
			}
			if k-i > 1 {
				i = k - 1
				continue
			}

		}
	}

	const dotRadius = 1
	const dotMargin = 4 * dotRadius
	const titleHeight, marginH, marginV, marginTop = 16, 8, 8, 9
	const nameTextW, valueTextW = 104, 82
	const barMaxW, barH = 320, 12
	const barMarginV, barMarginLeft, barMarginRight = 5, 3, 3
	const svgW = marginH + nameTextW + barMarginLeft + barMaxW + barMarginRight + valueTextW + marginH
	var svgH = marginV + titleHeight + marginTop
	if barCount > 0 {
		svgH += barCount*barH + (barCount-1)*barMarginV + marginV
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024*16))
	fmt.Fprintf(buf, `<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
<rect fill="#fff" id="canvas_background" width="%d" height="%d" y="-1" x="-1"/>
`,
		svgW, svgH, svgW+2, svgH+2,
	)

	barY := titleHeight + marginV

	fmt.Fprintf(buf, `<text xml:space="preserve" text-anchor="middle" font-weight="bold" font-family='"Courier New", Courier, monospace' font-size="12" x="%d" y="%d" fill="#000">%s</text>
`,
		svgW/2,
		barY-5,
		title,
	)

	barY += marginTop
	barX := marginH + nameTextW
	for i := 0; i < len(stat); i++ {
		drawDots := false

		v := stat[i]
		if v == 0 {
			k := i + 1
			for ; k < len(stat); k++ {
				if stat[k] != 0 {
					break
				}
			}
			if k-i > 1 {
				i = k - 1
				drawDots = true
			}

		}

		if drawDots {
			dotX := barX + dotRadius
			dotY := barY + barH/2 - dotMargin
			for range [3]struct{}{} {
				fmt.Fprintf(buf, `<circle cx="%d" cy="%d" r="%d" fill="#000"/>
`,
					dotX, dotY, dotRadius,
				)
				dotY += dotMargin
			}
		} else {
			valueTextX := float64(barX + barMarginRight)
			if maxV > 0 {
				barWidth := float64(barMaxW) * float64(v) / float64(maxV)
				valueTextX += barWidth
				fmt.Fprintf(buf, `<rect x="%d" y="%d" width="%.2f" height="%d" fill="#000" />
`,
					barX, barY, barWidth, barH,
				)
			}

			textY := barY + barH - 3
			nameTextX := barX - barMarginLeft
			nameText := xName(i)

			fmt.Fprintf(buf, `<text xml:space="preserve" text-anchor="end" font-family='"Courier New", Courier, monospace' font-size="12" x="%d" y="%d" fill="#000">%s</text>
`,
				nameTextX,
				textY,
				nameText,
			)

			if v != 0 {
				fmt.Fprintf(buf, `<text xml:space="preserve" text-anchor="start" font-style="italic" font-family='"Courier New", Courier, monospace' font-size="12" x="%.2f" y="%d" fill="#000">(%s)</text>
`,
					valueTextX,
					textY,
					strconv.Itoa(int(v)),
				)
			}
		}

		barY += barH + barMarginV
	}

	buf.WriteString(`</svg>`)
	return buf.Bytes()
}

// ToDo: add a bgColor parameter.
func createSourcefileImportsSVG_old(stat []int32, xName func(i int) string) []byte {
	if xName == nil {
		xName = func(i int) string {
			return strconv.Itoa(i)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024*16))
	buf.WriteString(`<svg width="528" height="168" xmlns="http://www.w3.org/2000/svg">
<rect fill="#ddf" id="canvas_background" height="402" width="582" y="-1" x="-1"/>
`,
	)

	maxV := int32(0)
	for _, v := range stat {
		if v > maxV {
			maxV = v
		}
	}

	n := len(stat)
	dn := (n + 15) / 16

	const barWidth = 6
	const maxBarHeight = 132

	for i, v := range stat {
		x := 8 + i*8
		if maxV > 0 {
			barHeight := float64(maxBarHeight) * float64(v) / float64(maxV)
			y := 19.0 + float64(maxBarHeight) - barHeight

			fmt.Fprintf(buf, `<rect x="%d" y="%.2f" width="%d" height="%.2f" fill="#000" />
`,
				x, y, barWidth, barHeight,
			)
		}

		if i%dn == 0 {
			textX := x + barWidth/2
			topY, bottomY := 13, 162

			fmt.Fprintf(buf, `<text xml:space="preserve" text-anchor="middle" font-family="Helvetica, Arial, sans-serif" font-size="12" x="%d" y="%d" fill="#000">%s</text>
`,
				textX,
				topY,
				strconv.Itoa(int(v)),
			)

			fmt.Fprintf(buf, `<text xml:space="preserve" text-anchor="middle" font-family="Helvetica, Arial, sans-serif" font-size="12" x="%d" y="%d" fill="#000">%s</text>
`,
				textX,
				bottomY,
				xName(i),
			)
		}
	}

	buf.WriteString(`</svg>`)
	return buf.Bytes()
}
