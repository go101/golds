package server

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"go101.org/golds/code"
	"go101.org/golds/internal/util"
)

var _ = log.Print

func (ds *docServer) packageDetailsPage(w http.ResponseWriter, r *http.Request, pkgPath string) {
	w.Header().Set("Content-Type", "text/html")

	// ToDo: create a custom "builtin" package page.

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.phase < Phase_Analyzed {
		w.WriteHeader(http.StatusTooEarly)
		ds.loadingPage(w, r)
		return
	}

	if genDocsMode {
		pkgPath = deHashScope(pkgPath)
	}

	pageKey := pageCacheKey{
		resType: ResTypePackage,
		res:     pkgPath,
	}

	data, ok := ds.cachedPage(pageKey)
	if !ok {
		//details := ds.buildPackageDetailsData(pkgPath)
		details := buildPackageDetailsData(ds.analyzer, pkgPath, collectUnexporteds)
		if details == nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Package (%s) not found", pkgPath)
			return
		}

		data = ds.buildPackageDetailsPage(w, details)
		ds.cachePage(pageKey, data)
	}
	w.Write(data)
}

func (ds *docServer) buildPackageDetailsPage(w http.ResponseWriter, pkg *PackageDetails) []byte {
	page := NewHtmlPage(goldsVersion, ds.currentTranslation.Text_Package(pkg.ImportPath), ds.currentTheme, ds.currentTranslation, createPagePathInfo1(ResTypePackage, pkg.ImportPath))

	fmt.Fprintf(page, `
<pre id="package-details"><code><span style="font-size:xx-large;">package <b>%s</b></span>
`,
		pkg.Name,
	)

	godevLink := pkg.ImportPath
	//if pkg.IsStandard {
	godevLink = strings.TrimPrefix(pkg.ImportPath, "vendor/")
	//}
	fmt.Fprintf(page, `
<span class="title">%s</span>
	<a href="%s#pkg-%s">%s</a>%s`,
		page.Translation().Text_ImportPath(),
		buildPageHref(page.PathInfo, createPagePathInfo(ResTypeNone, ""), nil, ""),
		pkg.ImportPath,
		pkg.ImportPath,
		page.Translation().Text_PackageDocsLinksOnOtherWebsites(godevLink, pkg.IsStandard),
	)

	isBuiltin := pkg.ImportPath == "builtin"
	if !isBuiltin {
		fmt.Fprintf(page, `

<span class="title">%s</span>
	%s`,
			page.Translation().Text_DependencyRelations(""),
			//page.Translation().Text_ImportStat(int(pkg.NumDeps), int(pkg.NumDepedBys), "/dep:"+pkg.ImportPath),
			page.Translation().Text_ImportStat(int(pkg.NumDeps), int(pkg.NumDepedBys), buildPageHref(page.PathInfo, createPagePathInfo1(ResTypeDependency, pkg.ImportPath), nil, "")),
		)
	}
	page.WriteString("\n")

	var isMainPackage = pkg.Package.PPkg.Name == "main"

	const classHiddenItem = "hidden"

	if len(pkg.Files) > 0 {

		writeFileTitle := func(info FileInfo) {
			if info.MainPosition != nil && info.DocText != "" {
				writeMainFunctionArrow(page, pkg.Package, *info.MainPosition)
				writeSourceCodeDocLink(page, pkg.Package, info.Filename, info.DocStartLine, info.DocEndLine)
			} else if info.MainPosition != nil {
				writeMainFunctionArrow(page, pkg.Package, *info.MainPosition)
				page.WriteString("   ")
			} else if info.DocText != "" {
				page.WriteString("   ")
				writeSourceCodeDocLink(page, pkg.Package, info.Filename, info.DocStartLine, info.DocEndLine)
			} else {
				page.WriteString("   ")
				page.WriteString("   ")
			}
			writeSrouceCodeFileLink(page, pkg.Package, info.Filename)
		}

		func() {
			page.WriteString("\n")
			page.WriteString(`<div id="files">`)
			defer page.WriteString("</div>")
			fmt.Fprint(page, `<span class="title">`, page.Translation().Text_InvolvedFiles(len(pkg.Files)), `</span>`)

			page.WriteString("\n")

			//writeLeadingSpaces := func() {
			//	page.WriteString("\n\t")
			//	page.WriteString("  ")
			//	page.WriteString("   ")
			//	page.WriteString("   ")
			//	page.WriteString("\t")
			//}
			//checked := ""
			//if isMainPackage {
			//	checked = " checked"
			//}
			for i, info := range pkg.Files {
				page.WriteString("\n\t")
				//if len(info.Resources) == 0 {
				if info.DocText == "" {
					page.WriteString(`<span class="nodocs">`)
					writeFileTitle(info)
					page.WriteString(`</span>`)
					continue
				}

				fid := fmt.Sprintf("file-%d", i)
				writeFoldingBlock(page, fid, "content", "items", true,
					func() {
						writeFileTitle(info)
					},
					func() {
						page.WriteString("\n")
						ds.renderDocComment(page, pkg.Package, "\t\t", info.DocText)

						if i < len(pkg.Files)-1 {
							page.WriteString("\n")
						}
					},
					//func() {
					//	if info.HasHiddenRes {
					//		writeLeadingSpaces()
					//		fmt.Fprintf(page, `<input%[1]s type='checkbox' class="showhide2" id='%[2]s'><i><label for='%[2]s'>%[3]s</label></i>`,
					//			checked, fid, page.Translation().Text_ListUnexportes())
					//	}
					//	for _, res := range info.Resources {
					//		func() {
					//			hidden := true
					//			if res.Type != nil {
					//				if res.Type.TypeName.Exported() {
					//					hidden = false
					//				}
					//			} else if res.Value.Exported() {
					//				hidden = false
					//			}
					//			hiddenClass := ""
					//			if hidden {
					//				hiddenClass = ` class="` + classHiddenItem + `"`
					//			}
					//			fmt.Fprintf(page, `<span%s>`, hiddenClass)
					//			defer page.WriteString(`</span>`)
					//			if hidden {
					//				page.WriteString(`<i>`)
					//				defer page.WriteString(`</i>`)
					//			}
					//			writeLeadingSpaces()
					//			if res.Type != nil {
					//				page.WriteString(" type ")
					//				fmt.Fprintf(page, `<a href="#name-%s">%s</a>`, res.Type.TypeName.Name(), res.Type.TypeName.Name())
					//				return
					//			}
					//
					//			switch res.Value.(type) {
					//			default:
					//				log.Println("impossible")
					//				return
					//			case *code.Variable:
					//				page.WriteString("  var ")
					//			case *code.Constant:
					//				page.WriteString("const ")
					//			case *code.Function:
					//				page.WriteString(" func ")
					//			}
					//
					//			fmt.Fprintf(page, `<a href="#name-%s">%s</a>`, res.Value.Name(), res.Value.Name())
					//		}()
					//	}
					//},
				)
			}
		}()
	}

	if len(pkg.Examples) > 0 {
		func() {
			page.WriteString("\n")
			page.WriteString(`<div id="examples">`)
			defer page.WriteString("</div>")
			fmt.Fprint(page, `<span class="title">`, page.Translation().Text_Examples(len(pkg.Examples)), `</span>`)

			page.WriteString("\n")

			for i, ex := range pkg.Examples {
				page.WriteString("\n\t")

				fid := fmt.Sprintf("example-%d", i)
				writeFoldingBlock(page, fid, "content", "items", false,
					func() {
						page.AsHTMLEscapeWriter().WriteString(ex.Name)
					},
					func() {
						page.WriteString("\n")

						// ToDo: need syntax hightlight writer.
						//       It is best to merge the example code with main code
						//       so that the exapmle code can be rendered as normal source code.
						if ex.Play != nil {
							format.Node(util.NewIndentWriter(
								page.AsHTMLEscapeWriter(),
								[]byte{'\t', '\t'}), pkg.ExampleFileSet, ex.Play)
						} else {
							format.Node(util.NewIndentWriter(
								page.AsHTMLEscapeWriter(),
								[]byte{'\t', ' ', ' '}), pkg.ExampleFileSet, ex.Code)
						}
						//if i < len(pkg.Examples)-1 {
						//	page.WriteString("\n")
						//}
					},
				)
			}
			page.WriteString("\n")
		}()
	}

	//var writePackageLevelValues = func(title, name string, values []code.ValueResource, numExporteds int) {
	var writePackageLevelValues = func(title, name string, values []ResourceWithPosition, numExporteds int) {

		page.WriteString("\n")

		func() {
			fmt.Fprintf(page, `<div id="exported-%s">`, name)
			defer page.WriteString("</div>")

			func() {
				page.WriteString(`<span class="title">`)
				defer page.WriteString(`</span>`)
				page.WriteString(title)
				page.WriteString(`<span class="title-stat"><i>`)
				defer page.WriteString(`</i></span>`)
				page.WriteString(page.Translation().Text_Parenthesis(false))
				defer page.WriteString(page.Translation().Text_Parenthesis(true))
				page.WriteString(page.Translation().Text_PackageLevelResourceSimpleStat(true, len(values), numExporteds, collectUnexporteds))
			}()

			page.WriteString("\n\n")

			for i, vwp := range values {
				v := vwp.Value
				if i == numExporteds {
					page.WriteString("\t")
					writeUnexportedResourcesHeader(page,
						name, !isMainPackage, len(values)-numExporteds)
				}

				unexported := i >= numExporteds

				extraClass := ""
				if unexported { // !v.Exported() {
					extraClass = " " + classHiddenItem
				}

				fmt.Fprintf(page, `<div class="anchor value-res%s" id="name-%s">`, extraClass, v.Name())
				if unexported {
					page.WriteString("<i>")
				}
				page.WriteString("\t")

				var writeFuncTypeParameters func()
				//>> 1.18
				if fv, ok := v.(*code.Function); ok {
					writeFuncTypeParameters = ds.writeTypeParameterListCallbackForFunction(page, pkg.Package, fv)
				}
				//<<

				if doc := v.Documentation(); doc == "" && writeFuncTypeParameters == nil {
					page.WriteString(`<span class="nodocs">`)
					ds.writeResourceIndexHTML(page, pkg.Package, v, true, true, true)
					page.WriteString(`</span>`)
				} else {
					writeFoldingBlock(page, v.Name(), "content", "docs", false,
						func() {
							ds.writeResourceIndexHTML(page, pkg.Package, v, true, true, true)
						},
						func() {
							if writeFuncTypeParameters != nil {
								writeFuncTypeParameters()
								page.WriteString("\n")
							}

							if doc != "" {
								page.WriteString("\n")
								ds.renderDocComment(page, pkg.Package, "\t\t", doc)
								page.WriteString("\n")
							}

							page.WriteString("\n")
						},
					)
				}

				if unexported {
					page.WriteString("</i>")
				}
				page.WriteString("</div>")
			}

			//if pkg.NumExportedValues == 0 {
			//	page.WriteString(`<div id="novalues">`)
			//	page.WriteString("\t")
			//	page.WriteString(page.Translation().Text_NoExportedValues())
			//	page.WriteString(`</div>`)
			//}
		}()
	}

	var writeItemWrapper = func(exported bool) (f func()) {
		if exported {
			page.WriteString(`<span>`)
			f = func() {
				page.WriteString(`</span>`)
			}
		} else {
			fmt.Fprintf(page, `<span class="%s"><i>`, classHiddenItem)
			f = func() {
				page.WriteString(`</i></span>`)
			}
		}
		page.WriteString("\n\t\t\t")
		return
	}

	var writeItemHeader = func(title, stat string) {
		page.WriteString(title)

		page.WriteString(page.Translation().Text_Parenthesis(false))
		defer page.WriteString(page.Translation().Text_Parenthesis(true))
		page.WriteString("<i>")
		defer page.WriteString("</i>")
		page.WriteString(stat)
	}

	if len(pkg.TypeNames) == 0 {
		goto WriteFunctions
	}

	page.WriteString("\n")

	page.WriteString(`<div id="exported-types">`)

	func() {
		page.WriteString(`<span class="title">`)
		defer page.WriteString(`</span>`)
		page.WriteString(page.Translation().Text_PackageLevelTypeNames())
		page.WriteString(`<span class="title-stat"><i>`)
		defer page.WriteString(`</i></span>`)
		page.WriteString(page.Translation().Text_Parenthesis(false))
		defer page.WriteString(page.Translation().Text_Parenthesis(true))
		page.WriteString(page.Translation().Text_PackageLevelResourceSimpleStat(true, len(pkg.TypeNames), int(pkg.NumExportedTypeNames), collectUnexporteds))
	}()

	page.WriteString("\n\n")

	page.WriteString(`<div id="exported-types-buttons" class="js-on">`)
	page.WriteString("\t/* ")
	if collectUnexporteds {
		page.WriteString(page.Translation().Text_SortBy("exporteds-types"))
	} else {
		page.WriteString(page.Translation().Text_SortBy(""))
	}
	page.WriteString(page.Translation().Text_Colon(false))
	page.WriteString(`<label id="sort-types-by-alphabet" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("alphabet"))
	page.WriteString(`</label>`)
	page.WriteString(" | ")
	page.WriteString(`<label id="sort-types-by-popularity" class="button">`)
	page.WriteString(page.Translation().Text_SortByItem("popularity"))
	page.WriteString(`</label>`)
	page.WriteString(" */</div>")

	for i, tdwp := range pkg.TypeNames {
		td := tdwp.Type
		if i == int(pkg.NumExportedTypeNames) {
			page.WriteString("</div><div>")
			page.WriteString("\t")
			writeUnexportedResourcesHeader(page,
				"typenames", !isMainPackage, len(pkg.TypeNames)-int(pkg.NumExportedTypeNames))
		}

		extraClass, typeIsExported := "", td.TypeName.Exported()
		if !typeIsExported {
			extraClass = " " + classHiddenItem
		}
		fmt.Fprintf(page, `<div class="anchor type-res%s" id="name-%s" data-popularity="%d">`, extraClass, td.TypeName.Name(), td.Popularity)
		page.WriteString("\t")

		//>> 1.18
		var writeTypeTypeParameters = ds.writeTypeParameterListCallbackForTypeName(page, pkg.Package, td.TypeName)
		//<<

		if doc := td.TypeName.Documentation(); doc == "" && writeTypeTypeParameters == nil && td.AllListsAreBlank {
			page.WriteString(`<span class="nodocs">`)
			ds.writeResourceIndexHTML(page, pkg.Package, td.TypeName, true, true, false)
			page.WriteString(`</span>`)
		} else {
			writeFoldingBlock(page, td.TypeName.Name(), "content", "docs", false,
				func() {
					ds.writeResourceIndexHTML(page, pkg.Package, td.TypeName, true, true, false)
				},
				func() {
					if writeTypeTypeParameters != nil {
						writeTypeTypeParameters()
						if doc != "" {
							page.WriteString("\n")
						}
					}

					if doc != "" {
						page.WriteString("\n")
						ds.renderDocComment(page, pkg.Package, "\t\t", doc)
					}

					// ToDo: for alias, if its denoting type is an exported named type, then stop here.
					//       (might be not a good idea. 1. such cases are rare. 2. if they happen, it does need to list ...)

					page.WriteByte('\n')
					hasLists := false
					if count, numExporteds := len(td.Fields), int(td.NumExportedFields); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "fields", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_Fields(),
									page.Translation().Text_PackageLevelResourceSimpleStat(true, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListFields:
								for _, fld := range td.Fields {
									if token.IsExported(fld.Name()) != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										if fldDoc, fldComment := fld.Field.Documentation(), fld.Field.Comment(); fldDoc == "" && fldComment == "" {
											page.WriteString(`<span class="nodocs">`)
											ds.writeFieldForListing(page, pkg.Package, fld, td.TypeName)
											page.WriteString(`</span>`)
										} else {
											writeFoldingBlock(page, td.TypeName.Name(), "field-"+fld.Name(), "docs", false,
												func() {
													ds.writeFieldForListing(page, pkg.Package, fld, td.TypeName)
												},
												func() {
													if fldDoc != "" {
														page.WriteString("\n")
														ds.renderDocComment(page, pkg.Package, "\t\t\t\t", fldDoc)
													}
													if fldComment != "" {
														page.WriteString("\n")
														ds.renderDocComment(page, pkg.Package, "\t\t\t\t// ", fldComment)
													}
													page.WriteString("\n")
												})
										}
									}()
								}

								if exported {
									if numUnexporteds := count - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "fields", typeIsExported, numUnexporteds, true)
										exported = false
										goto ListFields
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.Methods), int(td.NumExportedMethods); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "methods", "items", isBuiltin,
							func() {
								writeItemHeader(
									page.Translation().Text_Methods(),
									page.Translation().Text_PackageLevelResourceSimpleStat(true, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListMethods:
								for _, mthd := range td.Methods {
									if token.IsExported(mthd.Name()) != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										if mthdDoc, mthdComment := mthd.Method.Documentation(), mthd.Method.Comment(); mthdDoc == "" && mthdComment == "" {
											page.WriteString(`<span class="nodocs">`)
											ds.writeMethodForListing(page, pkg.Package, mthd, td.TypeName, true, false)
											page.WriteString(`</span>`)
										} else {
											writeFoldingBlock(page, td.TypeName.Name(), "method-"+mthd.Name(), "docs", false,
												func() {
													ds.writeMethodForListing(page, pkg.Package, mthd, td.TypeName, true, false)
												},
												func() {
													if mthdDoc != "" {
														page.WriteString("\n")
														ds.renderDocComment(page, pkg.Package, "\t\t\t\t", mthdDoc)
													}
													if mthdComment != "" {
														page.WriteString("\n")
														ds.renderDocComment(page, pkg.Package, "\t\t\t\t// ", mthdComment)
													}
													page.WriteString("\n")
												},
											)
										}
									}()
								}

								if exported {
									if numUnexporteds := len(td.Methods) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "methods", typeIsExported, numUnexporteds, true)
										exported = false
										goto ListMethods
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.ImplementedBys), int(td.NumExportedImpedBys); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "impledby", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_ImplementedBy(),
									page.Translation().Text_PackageLevelResourceSimpleStat(false, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListImpedBys:
								for _, by := range td.ImplementedBys {
									if by.TypeName.Exported() != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										ds.writeTypeForListing(page, by, pkg.Package, "", DotMStyle_NotShow)
										if _, ok := by.TypeName.Denoting().TT.Underlying().(*types.Interface); ok {
											page.WriteString(" <i>(interface)</i>")
										}
									}()
								}

								if exported {
									if numUnexporteds := len(td.ImplementedBys) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "impedBys", typeIsExported, numUnexporteds, false)
										exported = false
										goto ListImpedBys
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.Implements), int(td.NumExportedImpls); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "impls", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_Implements(),
									page.Translation().Text_PackageLevelResourceSimpleStat(false, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListImpls:
								for _, impl := range td.Implements {
									if impl.TypeName.Exported() != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										ds.writeTypeForListing(page, impl, pkg.Package, td.TypeName.Name(), DotMStyle_NotShow)
									}()
								}

								if exported {
									if numUnexporteds := len(td.Implements) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "impls", typeIsExported, numUnexporteds, false)
										exported = false
										goto ListImpls
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.AsOutputsOf), int(td.NumExportedAsOutputsOfs); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "results", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_AsOutputsOf(),
									page.Translation().Text_PackageLevelResourceSimpleStat(false, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListAsOutputsOf:
								for _, v := range td.AsOutputsOf {
									if v.Exported() != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										ds.writeValueForListing(page, v, pkg.Package, td.TypeName)
									}()
								}

								if exported {
									if numUnexporteds := len(td.AsOutputsOf) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "inputofs", typeIsExported, numUnexporteds, false)
										exported = false
										goto ListAsOutputsOf
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.AsInputsOf), int(td.NumExportedAsInputsOfs); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "params", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_AsInputsOf(),
									page.Translation().Text_PackageLevelResourceSimpleStat(false, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListAsInputsOf:
								for _, v := range td.AsInputsOf {
									if v.Exported() != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										ds.writeValueForListing(page, v, pkg.Package, td.TypeName)
									}()
								}

								if exported {
									if numUnexporteds := len(td.AsInputsOf) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "outputofs", typeIsExported, numUnexporteds, false)
										exported = false
										goto ListAsInputsOf
									}
								}
							},
						)
					}
					if count, numExporteds := len(td.Values), int(td.NumExportedValues); count > 0 {
						hasLists = true
						page.WriteString("\n\t\t")
						writeFoldingBlock(page, td.TypeName.Name(), "values", "items", false,
							func() {
								writeItemHeader(
									page.Translation().Text_AsTypesOf(),
									page.Translation().Text_PackageLevelResourceSimpleStat(true, count, numExporteds, collectUnexporteds),
								)
							},
							func() {
								exported := true
							ListAsTypesOf:
								for _, v := range td.Values {
									if v.Exported() != exported {
										continue
									}
									func() {
										defer writeItemWrapper(exported)()

										ds.writeValueForListing(page, v, pkg.Package, td.TypeName)
									}()
								}

								if exported {
									if numUnexporteds := len(td.Values) - numExporteds; numUnexporteds > 0 {
										page.WriteString("\n\t\t\t")
										writeHiddenItemsHeader(page, td.TypeName.Name(), "values", typeIsExported, numUnexporteds, true)
										exported = false
										goto ListAsTypesOf
									}
								}
							},
						)
					}
					page.WriteByte('\n')
					if hasLists {
						page.WriteByte('\n')
					}
				})
		}

		page.WriteString("</div>")
	}

	page.WriteString("</div>")

	//if pkg.NumExportedTypes == 0 {
	//	page.WriteString(`<div id="notypesnames">`)
	//	page.WriteString("\t")
	//	page.WriteString(page.Translation().Text_NoExportedTypeNames())
	//	page.WriteString(`</div>`)
	//}

WriteFunctions:

	if len(pkg.Functions) == 0 {
		goto WriteVariables
	}

	writePackageLevelValues(
		page.Translation().Text_PackageLevelFunctions(),
		"functions",
		pkg.Functions,
		int(pkg.NumExportedFunctions),
	)

WriteVariables:

	if len(pkg.Variables) == 0 {
		goto WriteConstants
	}

	writePackageLevelValues(
		page.Translation().Text_PackageLevelVariables(),
		"variables",
		pkg.Variables,
		int(pkg.NumExportedVariables),
	)

WriteConstants:

	if len(pkg.Constants) == 0 {
		goto Done
	}

	writePackageLevelValues(
		page.Translation().Text_PackageLevelConstants(),
		"constants",
		pkg.Constants,
		int(pkg.NumExportedConstants),
	)

Done:
	page.WriteString("</code></pre>")
	return page.Done(w)
}

type ResourceWithPosition struct {
	Position  token.Position
	FileIndex int32 // -1 means owner file not found
	Offset    int32

	//Res code.Resource
	// Use the following two instead of the above one to avoid
	// 1. change much code
	// 2. too many type assertions
	Type  *TypeDetails       // for PackageDetails.TypeNames only. Alway nil for FileInfo.
	Value code.ValueResource // also for TypeNames in FileInfo
}

type FileInfo struct {
	Filename     string
	MainPosition *token.Position // for main packages only
	Resources    []ResourceWithPosition
	//HasDocs      bool
	DocText                  string
	DocStartLine, DocEndLine int32
	//HasHiddenRes bool
}

type PackageDetails struct {
	//Mod *Module // ToDo:

	Package *code.Package

	IsStandard bool
	Index      int
	Name       string
	ImportPath string

	NumDeps     uint32
	NumDepedBys uint32

	Files []FileInfo
	//TypeNames []*TypeDetails
	TypeNames []ResourceWithPosition
	////ValueResources []code.ValueResource
	//Functions        []code.ValueResource
	//Variables        []code.ValueResource
	//Constants        []code.ValueResource
	Functions []ResourceWithPosition
	Variables []ResourceWithPosition
	Constants []ResourceWithPosition

	NumExportedTypeNames uint32
	NumExportedFunctions uint32
	NumExportedVariables uint32
	NumExportedConstants uint32

	// ToDo: use go/doc
	//IntroductionCode template.HTML
	Examples       []*doc.Example
	ExampleFileSet *token.FileSet
}

type TypeDetails struct {
	TypeName         *code.TypeName
	AllListsAreBlank bool
	Popularity       int

	Aliases []*TypeForListing // excluding self if self is an alias.

	Fields             []*SelectorForListing // []*code.Selector
	Methods            []*code.Selector
	NumExportedFields  int32
	NumExportedMethods int32

	// ToDo: Now both implements and implementebys miss aliases to unnamed types.
	//       (And miss many unnamed types. Maybe it is good to automatically
	//       create some aliases for the unnamed types without explicit aliases)

	ImplementedBys      []*TypeForListing
	Implements          []*TypeForListing
	NumExportedImpedBys int32
	NumExportedImpls    int32

	// ToDo: Including functions/methods, but not variables now?

	AsInputsOf              []*ValueForListing
	AsOutputsOf             []*ValueForListing
	NumExportedAsInputsOfs  int32
	NumExportedAsOutputsOfs int32

	// ToDo: also list functions for function types.
	//       But only for function types with at least
	//       one type declared in the current package,
	//       to avoid listing too many.

	Values            []*ValueForListing
	NumExportedValues int32
}

type ValueForListing struct {
	code.ValueResource
	InCurrentPkg bool
	CommonPath   string
}

type TypeForListing struct {
	*code.TypeName
	IsPointer    bool
	InCurrentPkg bool
	CommonPath   string // relative to the current package
}

type SelectorForListing struct {
	*code.Selector
	Middles []*code.Field

	numDuplicatedMiddlesWithLast int
}

// ToDo: adjust the coefficients
func (td *TypeDetails) calculatePopularity() {
	numValues := len(td.Values)
	if numValues > 3 {
		numValues = 3
	}
	td.Popularity = numValues*5 +
		len(td.Methods)*50 +
		len(td.Implements)*50 +
		len(td.ImplementedBys)*150 +
		len(td.AsInputsOf)*35 +
		len(td.AsOutputsOf)*75
}

// ds should be locked before calling this method.
// func (ds *docServer) buildPackageDetailsData(pkgPath string) *PackageDetails {
func buildPackageDetailsData(analyzer *code.CodeAnalyzer, pkgPath string, alsoCollectNonExporteds bool) *PackageDetails {
	pkg := analyzer.PackageByPath(pkgPath)
	if pkg == nil {
		return nil
	}

	pkgDetails := &PackageDetails{
		//PPkg: pkg.PPkg,
		//Mod:  pkg.Mod,
		//Info: pkg.PackageAnalyzeResult,

		Package: pkg,

		IsStandard: analyzer.IsStandardPackage(pkg),
		Index:      pkg.Index,
		Name:       pkg.PPkg.Name,
		ImportPath: pkg.PPkg.PkgPath,

		NumDeps:     uint32(len(pkg.Deps)),
		NumDepedBys: uint32(len(pkg.DepedBys)),
	}

	//analyzer.loadSourceFiles(pkg)

	isBuiltin := pkgPath == "builtin"

	// ...
	//files := make([]FileInfo, 0, len(pkg.PPkg.GoFiles)+len(pkg.PPkg.OtherFiles))
	files := make([]FileInfo, 0, len(pkg.SourceFiles))
	//lineStartOffsets := make(map[string][]int, len(pkg.PPkg.GoFiles))
	for i := range pkg.SourceFiles {
		f := &pkg.SourceFiles[i]
		if f.OriginalFile != "" {
			var start, end token.Position
			docText := ""
			if f.AstFile != nil && f.AstFile.Doc != nil {
				docText = f.AstFile.Doc.Text()
				start = pkg.PPkg.Fset.PositionFor(f.AstFile.Doc.Pos(), false)
				end = pkg.PPkg.Fset.PositionFor(f.AstFile.Doc.End(), false)
			}

			files = append(files, FileInfo{
				Filename: f.BareFilename,
				//HasDocs:  f.AstFile != nil && f.AstFile.Doc != nil,
				DocText:      docText,
				DocStartLine: int32(start.Line),
				DocEndLine:   int32(end.Line),
			})
		}
	}
	numAllResources := len(pkg.PackageAnalyzeResult.AllConstants) +
		len(pkg.PackageAnalyzeResult.AllVariables) +
		len(pkg.PackageAnalyzeResult.AllFunctions)
	numResesPerFile := numAllResources
	// ToDo: would better to cache several [1024]ResourceWithPosition in Server?
	if len(files) > 5 {
		numResesPerFile /= (len(files) - 1)
		numResesPerFile++
	}

	filename2index := make(map[string]int, len(files))
	for i := range files {
		filename2index[files[i].Filename] = i
		files[i].Resources = make([]ResourceWithPosition, 0, numAllResources)
	}
	regResForFile := func(res code.Resource) ResourceWithPosition {
		pos := res.Position()
		off, findex := int32(pos.Offset), int32(-1)
		if i, ok := filename2index[filepath.Base(pos.Filename)]; ok {
			findex = int32(i)
		}
		rwp := ResourceWithPosition{Position: pos, FileIndex: findex, Offset: off}
		if tn, ok := res.(*code.TypeName); ok {
			rwp.Type = &TypeDetails{TypeName: tn}
		} else {
			rwp.Value = res.(code.ValueResource)
		}
		if findex >= 0 {
			files[findex].Resources = append(files[findex].Resources, rwp)
		}
		return rwp
	}

	// Now, these file are also put into pkg.SourceFiles.
	//for _, path := range pkg.PPkg.OtherFiles {
	//	files = append(files, FileInfo{FilePath: path})
	//}

	if pkg.PPkg.Name == "main" {
		for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
			if f.Name() == "main" {
				mainPos := f.Position()
				filename := filepath.Base(mainPos.Filename)
				for i := range files {
					if files[i].Filename == filename {
						files[i].MainPosition = &mainPos
					}
				}
			}
		}
	}

	// ...

	//var valueResources = make([]code.ValueResource, 0,
	//	len(pkg.PackageAnalyzeResult.AllConstants)+
	//		len(pkg.PackageAnalyzeResult.AllVariables)+
	//		len(pkg.PackageAnalyzeResult.AllFunctions))

	//var functions = make([]code.ValueResource, 0, len(pkg.PackageAnalyzeResult.AllFunctions))
	//var variables = make([]code.ValueResource, 0, len(pkg.PackageAnalyzeResult.AllVariables))
	//var constants = make([]code.ValueResource, 0, len(pkg.PackageAnalyzeResult.AllConstants))
	var functions = make([]ResourceWithPosition, 0, len(pkg.PackageAnalyzeResult.AllFunctions))
	var variables = make([]ResourceWithPosition, 0, len(pkg.PackageAnalyzeResult.AllVariables))
	var constants = make([]ResourceWithPosition, 0, len(pkg.PackageAnalyzeResult.AllConstants))

	for _, f := range pkg.PackageAnalyzeResult.AllFunctions {
		if e := f.Exported(); (alsoCollectNonExporteds || e) && !f.IsMethod() {
			//functions = append(functions, f)
			rwp := regResForFile(f)
			functions = append(functions, rwp)
			if e {
				pkgDetails.NumExportedFunctions++
			}
		}
	}
	for _, v := range pkg.PackageAnalyzeResult.AllVariables {
		if e := v.Exported(); alsoCollectNonExporteds || e {
			//variables = append(variables, v)
			rwp := regResForFile(v)
			variables = append(variables, rwp)
			if e {
				pkgDetails.NumExportedVariables++
			}
		}
	}
	for _, c := range pkg.PackageAnalyzeResult.AllConstants {
		if e := c.Exported(); alsoCollectNonExporteds || e {
			//constants = append(constants, c)
			rwp := regResForFile(c)
			constants = append(constants, rwp)
			if e {
				pkgDetails.NumExportedConstants++
			}
		}
	}

	////sort.Slice(valueResources, func(i, j int) bool {
	////	// ToDo: cache lower names?
	////	return strings.ToLower(valueResources[i].Name()) < strings.ToLower(valueResources[j].Name())
	////})
	//sortValues := func(values []code.ValueResource) {
	//	sort.Slice(values, func(a, b int) bool {
	//		if ea, eb := values[a].Exported(), values[b].Exported(); ea != eb {
	//			return ea
	//		}
	//		// ToDo: cache lower names?
	//		return strings.ToLower(values[a].Name()) < strings.ToLower(values[b].Name())
	//	})
	//}
	sortValues := func(values []ResourceWithPosition) {
		sort.Slice(values, func(a, b int) bool {
			va, vb := values[a].Value, values[b].Value
			if ea, eb := va.Exported(), vb.Exported(); ea != eb {
				return ea
			}
			// ToDo: cache lower names?
			return strings.ToLower(va.Name()) < strings.ToLower(vb.Name())
		})
	}
	sortValues(functions)
	sortValues(variables)
	sortValues(constants)

	//var typeResources = make([]*TypeDetails, 0, len(pkg.PackageAnalyzeResult.AllTypeNames))
	var typeResources = make([]ResourceWithPosition, 0, len(pkg.PackageAnalyzeResult.AllTypeNames))

	//var unexportedTypesResources = make([]*code.TypeName, 0, len(pkg.PackageAnalyzeResult.AllTypeNames))
	for _, tn := range pkg.PackageAnalyzeResult.AllTypeNames {
		if e := tn.Exported(); e {
			pkgDetails.NumExportedTypeNames++
		} else if !alsoCollectNonExporteds {
			continue
		}

		denoting := tn.Denoting()
		//td := &TypeDetails{TypeName: tn}
		//typeResources = append(typeResources, td)
		rwp := regResForFile(tn)
		td := rwp.Type
		typeResources = append(typeResources, rwp)

		// Generally, we don't collect info for a type alias, execpt it denotes an unnamed or unexported type.
		// The info has been (or will be) collected for that denoting type.
		if tn.Alias != nil && tn.Alias.Denoting.TypeName != nil && tn.Alias.Denoting.TypeName.Exported() {
			continue
		}

		td.Fields, td.NumExportedFields = buildTypeFieldList(denoting, alsoCollectNonExporteds)
		td.Methods, td.NumExportedMethods = buildTypeMethodsList(denoting, alsoCollectNonExporteds)
		//td.ImplementedBys = make([]*code.TypeInfo, 0, len(denoting.ImplementedBys))
		td.ImplementedBys, td.NumExportedImpedBys = buildTypeImplementedByList(analyzer, pkg, denoting, alsoCollectNonExporteds, tn)
		//td.Implements = make([]code.Implementation, 0, len(denoting.Implements))
		td.Implements, td.NumExportedImpls = buildTypeImplementsList(analyzer, pkg, denoting, alsoCollectNonExporteds)

		if isBuiltin {
			continue
		}

		//td.Values = buildValueList(denoting.AsTypesOf, alsoCollectNonExporteds)
		td.AsInputsOf, td.NumExportedAsInputsOfs = buildValueList(denoting.AsInputsOf, pkg, alsoCollectNonExporteds)
		td.AsOutputsOf, td.NumExportedAsOutputsOfs = buildValueList(denoting.AsOutputsOf, pkg, alsoCollectNonExporteds)

		var values []code.ValueResource
		values = append(values, denoting.AsTypesOf...)
		// ToDo: also combine values of []T, chan T, ...
		//if t := analyzer.TryRegisteringType(types.NewPointer(denoting.TT)); t != nil {
		if t := analyzer.LookForType(types.NewPointer(denoting.TT)); t != nil {
			values = append(values, t.AsTypesOf...)
		}
		td.Values, td.NumExportedValues = buildValueList(values, pkg, alsoCollectNonExporteds)
	}

	for _, tdwp := range typeResources {
		td := tdwp.Type
		td.calculatePopularity()

		td.AllListsAreBlank =
			len(td.Fields) == 0 &&
				len(td.Methods) == 0 &&
				len(td.ImplementedBys) == 0 &&
				len(td.Implements) == 0 &&
				len(td.Values) == 0 &&
				len(td.AsInputsOf) == 0 &&
				len(td.AsOutputsOf) == 0
	}

	// default sort-by
	sort.Slice(typeResources, func(a, b int) bool {
		tna, tnb := typeResources[a].Type.TypeName, typeResources[b].Type.TypeName
		if ea, eb := tna.Exported(), tnb.Exported(); ea != eb {
			return ea
		}
		// ToDo: cache lower names?
		return strings.ToLower(tna.Name()) < strings.ToLower(tnb.Name())
	})

	//
	for i := range files {
		resources := files[i].Resources
		sort.Slice(resources, func(a, b int) bool {
			return resources[a].Offset < resources[b].Offset
		})
		//for k := range resources {
		//	if resources[k].Type != nil {
		//		if !resources[k].Type.TypeName.Exported() {
		//			files[i].HasHiddenRes = true
		//			break
		//		}
		//	} else if !resources[k].Value.Exported() {
		//		files[i].HasHiddenRes = true
		//		break
		//	}
		//}
	}

	// ...
	pkgDetails.Files = files
	//pkgDetails.ValueResources = valueResources
	pkgDetails.Functions = functions
	pkgDetails.Variables = variables
	pkgDetails.Constants = constants
	pkgDetails.TypeNames = typeResources

	pkgDetails.Examples = pkg.Examples
	pkgDetails.ExampleFileSet = analyzer.ExampleFileSet()

	return pkgDetails
}

func buildTypeFieldList(denoting *code.TypeInfo, alsoCollectNonExporteds bool) ([]*SelectorForListing, int32) {
	numExporteds, fields := int32(0), make([]*code.Selector, 0, len(denoting.AllFields))
	for _, fld := range denoting.AllFields {
		if e := token.IsExported(fld.Name()); alsoCollectNonExporteds || e {
			fields = append(fields, fld)
			if e {
				numExporteds++
			}
		}
	}
	return sortFieldList(fields), numExporteds
}

func createSelectorForListing(l *SelectorForListing, s *code.Selector) {
	l.Selector = s
	if s.Depth > 0 {
		l.Middles = make([]*code.Field, s.Depth)
		chain := s.EmbeddingChain
		for k := int(s.Depth) - 1; k >= 0; k-- {
			//log.Println(s.Depth, k, chain)
			l.Middles[k] = chain.Field
			chain = chain.Prev
		}
	}
}

func sortFieldList(selectors []*code.Selector) []*SelectorForListing {
	selList := make([]SelectorForListing, len(selectors))
	result := make([]*SelectorForListing, len(selectors))
	for i, sel := range selectors {
		selForListing := &selList[i]
		result[i] = selForListing
		//selForListing.Selector = sel
		//if sel.Depth > 0 {
		//	selForListing.Middles = make([]*code.Field, sel.Depth)
		//	chain := sel.EmbeddingChain
		//	for k := int(sel.Depth) - 1; k >= 0; k-- {
		//		//log.Println(sel.Depth, k, chain)
		//		selForListing.Middles[k] = chain.Field
		//		chain = chain.Prev
		//	}
		//}
		createSelectorForListing(selForListing, sel)
	}

	sort.Slice(result, func(a, b int) bool {
		sa, sb := result[a], result[b]
		if ea, eb := token.IsExported(sa.Name()), token.IsExported(sb.Name()); ea != eb {
			return ea
		}

		k := len(sa.Middles)
		if k > len(sb.Middles) {
			k = len(sb.Middles)
		}
		for i := 0; i < k; i++ {
			switch strings.Compare(strings.ToLower(sa.Middles[i].Name), strings.ToLower(sb.Middles[i].Name)) {
			case -1:
				return true
			case 1:
				return false
			}
		}
		if len(sa.Middles) < len(sb.Middles) {
			switch strings.Compare(strings.ToLower(sa.Name()), strings.ToLower(sb.Middles[k].Name)) {
			case 0, -1:
				return true
			case 1:
				return false
			}
		}
		if len(sa.Middles) > len(sb.Middles) {
			switch strings.Compare(strings.ToLower(sa.Middles[k].Name), strings.ToLower(sb.Name())) {
			case 0, 1:
				return false
			case -1:
				return true
			}
		}
		return sa.Name() < sb.Name()
	})

	for i := 1; i < len(result); i++ {
		last := result[i-1]
		sel := result[i]
		i, k := 0, len(last.Middles)
		if k > len(sel.Middles) {
			k = len(sel.Middles)
		}
		for ; i < k; i++ {
			if last.Middles[i].Name != sel.Middles[i].Name {
				break
			}
		}
		if len(last.Middles) < len(sel.Middles) {
			if last.Name() == sel.Middles[i].Name {
				i++
			}
		}
		sel.numDuplicatedMiddlesWithLast = i
	}

	return result
}

func buildTypeMethodsList(denoting *code.TypeInfo, alsoCollectNonExporteds bool) ([]*code.Selector, int32) {
	numExporteds, methods := int32(0), make([]*code.Selector, 0, len(denoting.AllMethods))
	for _, mthd := range denoting.AllMethods {
		if e := token.IsExported(mthd.Name()); alsoCollectNonExporteds || e {
			methods = append(methods, mthd)
			if e {
				numExporteds++
			}
		}
	}
	return sortMethodList(methods), numExporteds
}

func sortMethodList(selectors []*code.Selector) []*code.Selector {
	sort.Slice(selectors, func(a, b int) bool {
		return selectors[a].Name() < selectors[b].Name()
	})
	return selectors
}

func buildTypeImplementedByList(analyzer *code.CodeAnalyzer, pkg *code.Package, denoting *code.TypeInfo, alsoCollectNonExporteds bool, exceptTypeName *code.TypeName) ([]*TypeForListing, int32) {
	numExporteds, implementedBys := int32(0), make([]TypeForListing, 0, len(denoting.ImplementedBys))
	for _, impledBy := range denoting.ImplementedBys {
		bytn, isPointer := analyzer.RetrieveTypeName(impledBy)
		if bytn == nil || bytn == exceptTypeName {
			continue
		}
		if e := bytn.Exported(); alsoCollectNonExporteds || e {
			implementedBys = append(implementedBys, TypeForListing{
				TypeName:  bytn,
				IsPointer: isPointer,
			})
			if e {
				numExporteds++
			}
		}
	}
	return sortTypeList(implementedBys, pkg), numExporteds
}

func buildTypeImplementsList(analyzer *code.CodeAnalyzer, pkg *code.Package, denoting *code.TypeInfo, alsoCollectNonExporteds bool) ([]*TypeForListing, int32) {
	//implements = make([]code.Implementation, 0, len(denoting.Implements))
	numExporteds, implements := int32(0), make([]TypeForListing, 0, len(denoting.Implements))
	for _, impl := range analyzer.CleanImplements(denoting) {
		//if impl.Interface.TypeName == nil || token.IsExported(impl.Interface.TypeName.Name()) {
		//	td.Implements = append(td.Implements, impl)
		//}
		// Might miss: interface {Unwrap() error}
		itn := impl.Interface.TypeName
		if itn == nil {
			continue
		}
		if e := itn.Exported(); alsoCollectNonExporteds || e {
			_, isPointer := impl.Impler.TT.(*types.Pointer)
			implements = append(implements, TypeForListing{
				TypeName:  itn,
				IsPointer: isPointer,
			})
			if e {
				numExporteds++
			}
		}
	}
	return sortTypeList(implements, pkg), numExporteds
}

// Assume all types are named or pointer to named.
func sortTypeList(typeList []TypeForListing, pkg *code.Package) []*TypeForListing {
	result := make([]*TypeForListing, len(typeList))

	pkgPath := pkg.Path
	for i := range typeList {
		t := &typeList[i]
		result[i] = t
		t.InCurrentPkg = t.Package() == pkg
		if t.InCurrentPkg {
			t.CommonPath = pkgPath
		} else {
			t.CommonPath = FindPackageCommonPrefixPaths(t.Package().Path, pkgPath)
		}
	}

	sort.Slice(result, func(a, b int) bool {
		if ea, eb := result[a].Exported(), result[b].Exported(); ea != eb {
			return ea
		}

		if x, y := result[a].InCurrentPkg, result[b].InCurrentPkg; x || y {
			if x && y {
				return strings.ToLower(result[a].Name()) < strings.ToLower(result[b].Name())
			}
			return x
		}
		commonA, commonB := result[a].CommonPath, result[b].CommonPath
		if len(commonA) != len(commonB) {
			if len(commonA) == len(pkgPath) {
				return true
			}
			if len(commonB) == len(pkgPath) {
				return false
			}
			if len(commonA) > 0 || len(commonB) > 0 {
				return len(commonA) > len(commonB)
			}
		}
		pathA, pathB := strings.ToLower(result[a].Pkg.Path), strings.ToLower(result[b].Pkg.Path)
		r := strings.Compare(pathA, pathB)
		if r == 0 {
			return strings.ToLower(result[a].Name()) < strings.ToLower(result[b].Name())
		}
		if pathA == "builtin" {
			return true
		}
		if pathB == "builtin" {
			return false
		}
		return r < 0
	})

	return result
}

func buildValueList(values []code.ValueResource, pkg *code.Package, showUnexported bool) ([]*ValueForListing, int32) {
	numExporteds, n, listedValues := int32(0), 0, make([]ValueForListing, len(values))
	for i := range listedValues {
		if e := values[i].Exported(); showUnexported || e {
			lv := &listedValues[n]
			lv.ValueResource = values[i]
			n++
			if e {
				numExporteds++
			}
		}
	}
	return sortValueList(listedValues[:n], pkg), numExporteds
}

// The implementations sortValueList and sortTypeList are some reapetitive.
// Need generic.? (Or let ValueForListing and TypeForListing implement the same interface)
func sortValueList(valueList []ValueForListing, pkg *code.Package) []*ValueForListing {
	result := make([]*ValueForListing, len(valueList))

	pkgPath := pkg.Path
	for i := range valueList {
		v := &valueList[i]
		result[i] = v
		v.InCurrentPkg = v.Package() == pkg
		if !v.InCurrentPkg {
			v.CommonPath = FindPackageCommonPrefixPaths(v.Package().Path, pkgPath)
		}
	}

	compareWithoutPackges := func(a, b *ValueForListing) bool {
		fa, oka := a.ValueResource.(code.FunctionResource)
		fb, okb := b.ValueResource.(code.FunctionResource)

		if oka && okb {
			if p, q := fa.IsMethod(), fb.IsMethod(); p && q {
				_, tna, _ := fa.ReceiverTypeName()
				_, tnb, _ := fb.ReceiverTypeName()
				if r := strings.Compare(strings.ToLower(tna.Name()), strings.ToLower(tnb.Name())); r != 0 {
					return r < 0
				}
			} else if p != q {
				return q
			}
		}
		return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
	}

	sort.Slice(result, func(a, b int) bool {
		if ea, eb := result[a].Exported(), result[b].Exported(); ea != eb {
			return ea
		}

		if x, y := result[a].InCurrentPkg, result[b].InCurrentPkg; x || y {
			if x && y {
				return compareWithoutPackges(result[a], result[b])
			}
			return x
		}
		commonA, commonB := result[a].CommonPath, result[b].CommonPath
		if len(commonA) != len(commonB) {
			if len(commonA) == len(pkgPath) {
				return true
			}
			if len(commonB) == len(pkgPath) {
				return false
			}
			if len(commonA) > 0 || len(commonB) > 0 {
				return len(commonA) > len(commonB)
			}
		}
		r := strings.Compare(strings.ToLower(result[a].Package().Path), strings.ToLower(result[b].Package().Path))
		if r == 0 {
			return compareWithoutPackges(result[a], result[b])
		}
		if result[a].Package().Path == "builtin" {
			return true
		}
		if result[b].Package().Path == "builtin" {
			return false
		}
		return r < 0
	})

	return result
}

// The function is some repeatitive with writeResourceIndexHTML.
// func (ds *docServer) writeValueForListing(page *htmlPage, v *ValueForListing, pkg *code.Package, fileLineOffsets map[string][]int, forTypeName *code.TypeName) {
func (ds *docServer) writeValueForListing(page *htmlPage, v *ValueForListing, pkg *code.Package, forTypeName *code.TypeName) {
	pos := v.Position()
	//if lineOffsets, ok := fileLineOffsets[pos.Filename]; ok {
	//	correctPosition(lineOffsets, &pos)
	//} else {
	//	pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	//}

	//log.Println("   :", pos)

	switch res := v.ValueResource.(type) {
	default:
		panic("should not")
	case *code.Constant, *code.Variable:
		if _, ok := v.ValueResource.(*code.Constant); ok {
			fmt.Fprint(page, `const `)
		} else {
			fmt.Fprint(page, `  var `)
		}

		if v.Package() != pkg {
			//if v.Package().Path != "builtin" {
			page.WriteString(v.Package().Path)
			page.WriteByte('.')
			//}
			fmt.Fprintf(page, `<a href="`)
			//page.WriteString("/pkg:")
			//page.WriteString(v.Package().Path)
			buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, v.Package().Path), page, "")
		} else {
			fmt.Fprintf(page, `<a href="`)
		}
		page.WriteString("#name-")
		page.WriteString(v.Name())
		fmt.Fprintf(page, `">`)
		page.WriteString(v.Name())
		page.WriteString("</a>")

		if t := res.TypeInfo(ds.analyzer); t != forTypeName.Denoting() {
			page.WriteByte(' ')
			//page.WriteString(res.TType().String())
			specOwner := res.(code.AstValueSpecOwner)
			if astType := specOwner.AstValueSpec().Type; astType != nil {
				ds.WriteAstType(page, astType, specOwner.Package(), specOwner.Package(), false, nil, forTypeName)
			} else {
				// ToDo: track to get the AstType and use WriteAstType instead.
				ds.writeValueTType(page, res.TType(), specOwner.Package(), true, forTypeName)
			}
		}
	//case *code.Function, *code.InterfaceMethod:
	case code.FunctionResource:

		page.WriteString("func ")
		if vpkg := v.Package(); vpkg != pkg {
			if vpkg != nil {
				page.WriteString(v.Package().Path)
				page.WriteString(".")
			}
		}

		if res.IsMethod() {
			// note: recvParam might be nil for interface method.
			recvParam, tn, isStar := res.ReceiverTypeName()
			if isStar {
				if v.Package() != pkg {
					//fmt.Fprintf(page, `(*<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>).`, v.Package().Path, tn.Name())
					page.WriteString("(*")
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, v.Package().Path), page, tn.Name(), "name-", tn.Name())
					page.WriteString(")")
				} else {
					// ToDo: faster way: ds.analyzer.TryRegisteringType(tn.Type()) == forTypeName.Denoting()?
					if forTypeName != nil && types.Identical(tn.Type(), forTypeName.Denoting().TT) {
						fmt.Fprintf(page, `(*%[1]s)`, tn.Name())
					} else {
						fmt.Fprintf(page, `(*<a href="#name-%[1]s">%[1]s</a>)`, tn.Name())
					}
				}
				//fmt.Fprintf(page, "(*%s) ", tn.Name())
			} else {
				if v.Package() != pkg {
					//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>.`, v.Package().Path, tn.Name())
					buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, v.Package().Path), page, tn.Name(), "name-", tn.Name())
				} else {
					// ToDo: faster way: ds.analyzer.TryRegisteringType(tn.Type()) == forTypeName.Denoting()?
					if forTypeName != nil && types.Identical(tn.Type(), forTypeName.Denoting().TT) {
						fmt.Fprintf(page, `%[1]s`, tn.Name())
					} else {
						fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>`, tn.Name())
					}
				}
				//fmt.Fprintf(page, "(%s) ", tn.Name())
			}

			//>> 1.18
			var astFunc *ast.FuncDecl
			if f, ok := res.(*code.Function); ok {
				astFunc = f.AstDecl
			}
			writeTypeParamsForMethodReceiver(page, astFunc, forTypeName)
			//<<
			page.WriteString(".")

			//writeSrouceCodeLineLink(page, v.Package(), pos, v.Name(), "")
			writeSrouceCodeLineLink(page, res.AstPackage(), pos, v.Name(), "")

			//ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, pkg, false, recvParam, forTypeName)
			ds.WriteAstType(page, res.AstFuncType(), res.AstPackage(), pkg, false, nil, forTypeName)
			_ = recvParam // might be nil for interface method.
		} else {
			if v.Package() != pkg {
				//fmt.Fprintf(page, `<a href="/pkg:%[1]s#name-%[2]s">%[2]s</a>`, v.Package().Path, v.Name())
				buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, v.Package().Path), page, v.Name(), "name-", v.Name())
			} else {
				fmt.Fprintf(page, `<a href="#name-%[1]s">%[1]s</a>`, v.Name())
			}

			//>> 1.18
			if fv, ok := res.(*code.Function); ok { // always ok
				writeTypeParamsOfFunciton(page, fv)
			}
			//<<

			ds.WriteAstType(page, res.AstFuncType(), res.AstPackage(), pkg, false, nil, forTypeName)
		}
	}
}

const (
	DotMStyle_Unexported = -1
	DotMStyle_NotShow    = 0
	DotMStyle_Exported   = 1
)

// writeReceiverLink=false means for method implementation page.
// exportMethod is for method implementation page only.
func (ds *docServer) writeTypeForListing(page *htmlPage, t *TypeForListing, pkg *code.Package, implerName string, dotMStyle int) {
	if implerName == "" {
	} else if dotMStyle == DotMStyle_NotShow {
		if t.IsPointer {
			//page.WriteString("*T : ")
			fmt.Fprintf(page, "*%s : ", implerName)
		} else {
			//page.WriteString(" T : ")
			fmt.Fprintf(page, " %s : ", implerName)
		}
	} else if dotMStyle > 0 { // DotMStyle_Exported
		if t.IsPointer {
			//page.WriteString("(*T).M : ")
			fmt.Fprintf(page, "*%s.M : ", implerName)
		} else {
			//page.WriteString("     M : ")
			fmt.Fprintf(page, " %s.M : ", implerName)
		}
	} else { // DotMStyle_Unexported
		if t.IsPointer {
			//page.WriteString("(*T).m : ")
			fmt.Fprintf(page, "*%s.m : ", implerName)
		} else {
			//page.WriteString("     m : ")
			fmt.Fprintf(page, " %s.m : ", implerName)
		}
	}

	if t.Package() != pkg {
		if implerName == "" {
			if t.IsPointer {
				page.WriteString("*")
			} else {
				page.WriteString(" ")
			}
		}

		if t.Pkg.Path != "builtin" {
			page.WriteString(t.Pkg.Path)
			page.WriteByte('.')
		}

		//if implerName == "" && t.IsPointer {
		//	page.WriteString("(*")
		//	defer page.WriteByte(')')
		//}
	} else {
		if implerName == "" {
			if t.IsPointer {
				if dotMStyle == DotMStyle_NotShow {
					page.WriteString("*")
				} else { // for method implementation listing
					page.WriteString("(*")
					defer page.WriteByte(')')
				}
			} else {
				if dotMStyle == DotMStyle_NotShow {
					page.WriteString(" ")
				}
			}
		}
	}

	// Now, unexported types are always listed (but hidden initially).
	//if t.Exported() {
	// dotMStyle != DotMStyle_NotShow means in method implementation list page.
	if t.Package() != pkg || dotMStyle != DotMStyle_NotShow {
		fmt.Fprintf(page, `<a href="`)
		//page.WriteString("/pkg:")
		//page.WriteString(t.Pkg.Path)
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, t.Pkg.Path), page, "")
	} else {
		fmt.Fprintf(page, `<a href="`)
	}
	page.WriteString("#name-")
	page.WriteString(t.Name())
	fmt.Fprintf(page, `">`)
	page.WriteString(t.Name())
	page.WriteString("</a>")
	//} else {
	//	//page.WriteString("?show=all")
	//	writeSrouceCodeLineLink(page, t.Pkg, t.Position(), t.Name(), "")
	//}
}

func (ds *docServer) WriteEmbeddingChain(page *htmlPage, embedding *code.EmbeddedField) {
	if embedding == nil {
		return
	}

	if embedding.Prev != nil {
		ds.WriteEmbeddingChain(page, embedding.Prev)
	}

	pos := embedding.Field.Position()
	page.WriteString("<i>")
	writeSrouceCodeLineLink(page, embedding.Field.Pkg, pos, embedding.Field.Name, "")
	page.WriteString("</i>")
	page.WriteByte('.')
}

func (ds *docServer) writeFieldForListing(page *htmlPage, pkg *code.Package, sel *SelectorForListing, forTypeName *code.TypeName) {
	for i, fld := range sel.Middles {
		pos := fld.Position()
		//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
		class := ""
		if i < sel.numDuplicatedMiddlesWithLast {
			class = "path-duplicate"
		}

		// ToDo: the if-else blocks are identical now, so ...?
		if token.IsExported(fld.Name) {
			writeSrouceCodeLineLink(page, fld.Pkg, pos, fld.Name, class)
		} else {
			//writeSrouceCodeLineLink(page, fld.Pkg, pos, "<strike>"+fld.Name+"</strike>", class)
			//page.WriteString("<strike>.</strike>")
			//page.WriteString("<i>")
			writeSrouceCodeLineLink(page, fld.Pkg, pos, fld.Name, class)
			//page.WriteString(".</i>")
		}
		page.WriteString(".")
	}
	ds.writeFieldCodeLink(page, sel.Selector)
	page.WriteString(" <i>")
	ds.WriteAstType(page, sel.Field.AstField.Type, sel.Field.Pkg, pkg, true, nil, forTypeName)
	page.WriteString("</i>")
}

func (ds *docServer) writeFieldCodeLink(page *htmlPage, sel *code.Selector) {
	selField := sel.Field
	if selField == nil {
		panic("should not")
	}
	pos := sel.Position()
	//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
	writeSrouceCodeLineLink(page, sel.Package(), pos, selField.Name, "")
}

func (ds *docServer) writeMethodForListing(page *htmlPage, docPkg *code.Package, sel *code.Selector, forTypeName *code.TypeName, writeReceiver, onlyWriteMethodName bool) {
	method := sel.Method
	if method == nil {
		panic("should not")
	}

	if writeReceiver {
		if sel.PointerReceiverOnly() {
			//page.WriteString("(*T) ")
			page.WriteString("(*")
		} else {
			//page.WriteString("( T) ")
			page.WriteString("( ")
		}
		page.WriteString(forTypeName.Name())
		//>> 1.18
		writeTypeParamsForMethodReceiver(page, method.AstFunc, forTypeName)
		//<<
		page.WriteString(") ")
	}

	if method.Pkg.Path == "builtin" {
		if docPkg == method.Pkg {
			page.WriteString(method.Name)
		} else {
			// error.Error
			// ToDo: If later, there is other builtin methods, the function prototype needs to be changed.
			buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, "builtin"), page, method.Name, "name-error")
		}
	} else {
		pos := sel.Position()
		//pos.Line += ds.analyzer.SourceFileLineOffset(pos.Filename)
		writeSrouceCodeLineLink(page, sel.Package(), pos, method.Name, "")
	}

	if !onlyWriteMethodName {
		ds.writeMethodType(page, docPkg, method, forTypeName)
	}
}

func (ds *docServer) writeMethodType(page *htmlPage, docPkg *code.Package, method *code.Method, forTypeName *code.TypeName) {
	if method.AstFunc != nil {
		ds.WriteAstType(page, method.AstFunc.Type, method.Pkg, docPkg, false, nil, forTypeName)
	} else {
		ds.WriteAstType(page, method.AstField.Type, method.Pkg, docPkg, false, nil, forTypeName)
	}
}

func writeKindText(page *htmlPage, tt types.Type) {
	var kind string
	var bold = false

	switch tt.Underlying().(type) {
	default:
		return
	case *types.Basic:
		kind = page.Translation().Text_BasicType()
	case *types.Pointer:
		kind = "*"
	case *types.Struct:
		kind = reflect.Struct.String()
	case *types.Array:
		kind = "[...]"
	case *types.Slice:
		kind = "[]"
	case *types.Map:
		kind = reflect.Map.String()
	case *types.Chan:
		kind = reflect.Chan.String()
	case *types.Signature:
		kind = reflect.Func.String()
	case *types.Interface:
		kind = reflect.Interface.String()
		bold = true
	}

	if bold {
		fmt.Fprintf(page, ` <b><i>(%s)</i></b>`, kind)
	} else {
		fmt.Fprintf(page, ` <i>(%s)</i>`, kind)
	}
}

// func (ds *docServer) writeResourceIndexHTML(page *htmlPage, res code.Resource, fileLineOffsets map[string][]int, writeType, writeReceiver bool) {
func (ds *docServer) writeResourceIndexHTML(page *htmlPage, currentPkg *code.Package, res code.Resource, writeKeyword, writeType, writeComment bool) {
	//log.Println("   :", pos)

	isBuiltin := res.Package().Path == "builtin"

	writeResName := func() {
		var fPkg = res.Package()
		var fPosition token.Position
		if isBuiltin {
			if currentPkg != res.Package() || page.PathInfo.resType != ResTypePackage {
				buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, "builtin"), page, res.Name(), "name-", res.Name())
				return
			}

			fPkg = ds.analyzer.RuntimePackage()
			switch res.Name() {
			default:
			case "close":
				fPosition = ds.analyzer.RuntimeFunctionCodePosition("closechan")
			case "panic":
				fPosition = ds.analyzer.RuntimeFunctionCodePosition("gopanic")
			case "recover":
				fPosition = ds.analyzer.RuntimeFunctionCodePosition("gorecover")
			}
		} else {
			fPosition = res.Position()
		}

		if !fPosition.IsValid() {
			page.WriteString(res.Name())
			return
		}
		writeSrouceCodeLineLink(page, fPkg, fPosition, res.Name(), "")
	}

	switch res := res.(type) {
	default:
		panic("should not")
	case *code.TypeName:
		if writeKeyword {
			if buildIdUsesPages && !isBuiltin {
				page.WriteByte(' ')
				buildPageHref(page.PathInfo, createPagePathInfo2(ResTypeReference, res.Package().Path, "..", res.Name()), page, "type")
				page.WriteByte(' ')
			} else {
				page.WriteString(" type ")
			}
		}

		writeResName()

		if writeType {
			//>> 1.18
			writeTypeParamsOfTypeName(page, res)
			//<<

			showSource := false
			if isBuiltin {
				// builtin package source code are fake.
				showSource = res.Alias != nil
			} else {
				allowStar := res.Alias != nil
				for t, done := res.AstSpec.Type, false; !done; {
					switch e := t.(type) {
					//>> ToDo 1.18
					// astIndexExpr, astIndexListExpr ?
					//<<
					case *ast.Ident, *ast.SelectorExpr:
						showSource = true
						done = true
					case *ast.ParenExpr:
						t = e.X
					case *ast.StarExpr:
						// type A = *T
						if allowStar {
							t = e.X
							allowStar = false
						} else {
							done = true
						}
					default:
						done = true
					}
				}
			}

			if res.Alias != nil {
				page.WriteByte(' ')
				page.WriteByte('=')
			}

			//page.WriteString(types.TypeString(res.Denoting().TT, types.RelativeTo(res.Package().PPkg.Types)))
			if res.AstSpec.Type == nil {
				panic("res.Alias != nil, but res.AstSpec.Type == nil, ???")
			}

			if showSource {
				page.WriteByte(' ')
				ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, true, nil, nil)
				//ds.writeValueTType(page, res.Denoting().TT, res.Pkg, true, nil)
			}
			writeKindText(page, res.Denoting().TT)
		}
	case *code.Constant:
		if writeKeyword {
			if buildIdUsesPages && !isBuiltin {
				buildPageHref(page.PathInfo, createPagePathInfo2(ResTypeReference, res.Package().Path, "..", res.Name()), page, "const")
				page.WriteByte(' ')
			} else {
				page.WriteString("const ")
			}
		}

		writeResName()

		if writeType {
			btt, ok := res.TType().Underlying().(*types.Basic)
			if !ok {
				panic("constants should be always of basic types, but " + res.String() + " : " + res.TType().String())
			}
			if btt.Info()&types.IsUntyped == 0 {
				page.WriteByte(' ')
				//page.WriteString(types.TypeString(res.TType(), types.RelativeTo(res.Package().PPkg.Types)))
				if res.AstSpec.Type != nil {
					ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, false, nil, nil)
				} else {
					ds.writeValueTType(page, res.TType(), res.Pkg, true, nil)
				}
			}
			if !isBuiltin {
				page.WriteString(" = ")
				page.WriteString(res.Val().String())
			}
		}
	case *code.Variable:
		if writeKeyword {
			if buildIdUsesPages && !isBuiltin {
				page.WriteByte(' ')
				page.WriteByte(' ')
				buildPageHref(page.PathInfo, createPagePathInfo2(ResTypeReference, res.Package().Path, "..", res.Name()), page, "var")
				page.WriteByte(' ')
			} else {
				page.WriteString("  var ")
			}
		}

		writeResName()

		if writeType {
			page.WriteByte(' ')
			//page.WriteString(res.TType().String())
			if res.AstSpec.Type != nil {
				ds.WriteAstType(page, res.AstSpec.Type, res.Pkg, res.Pkg, false, nil, nil)
			} else {
				// ToDo: track to get the AstType and use WriteAstType instead.
				ds.writeValueTType(page, res.TType(), res.Pkg, true, nil)
			}
		}
	case *code.Function:
		if writeKeyword {
			if buildIdUsesPages && !isBuiltin {
				page.WriteByte(' ')
				buildPageHref(page.PathInfo, createPagePathInfo2(ResTypeReference, res.Package().Path, "..", res.Name()), page, "func")
				page.WriteByte(' ')
			} else {
				page.WriteString(" func ")
			}

			//var recv *types.Var
			//if res.Func != nil {
			//	sig := res.Func.Type().(*types.Signature)
			//	recv = sig.Recv()
			//}
			//// This if-block will be never entered now.
			//if recv != nil {
			//	switch tt := recv.Type().(type) {
			//	case *types.Named:
			//		fmt.Fprintf(page, `(%s) `, tt.Obj().Name())
			//	case *types.Pointer:
			//		if named, ok := tt.Elem().(*types.Named); ok {
			//			fmt.Fprintf(page, `(*%s) `, named.Obj().Name())
			//		} else {
			//			panic("should not")
			//		}
			//	default:
			//		panic("should not")
			//	}
			//}
		}

		//>> for statistis toppest list listing
		if res.IsMethod() {
			recvParam, tn, isStar := res.ReceiverTypeName()
			_ = recvParam
			if isStar {
				page.WriteString("(*")
				page.WriteString(tn.Name())
				page.WriteString(")")
			} else {
				page.WriteString(tn.Name())
			}
			page.WriteByte('.')
		}
		//<<

		writeResName()

		if writeType {
			//>> 1.18
			writeTypeParamsOfFunciton(page, res)
			//<<

			ds.WriteAstType(page, res.AstDecl.Type, res.Pkg, res.Pkg, false, nil, nil)
			//ds.writeValueTType(page, res.TType(), res.Pkg, false)
		}
	}

	if writeComment {
		if comment := res.Comment(); comment != "" {
			page.WriteString(" // ")
			ds.renderDocComment(page, currentPkg, "", comment)
		}
	}

	//fmt.Fprint(page, ` <a href="#">{/}</a>`)
}

func (ds *docServer) writeTypeName(page *htmlPage, tt *types.Named, docPkg *code.Package, alternativeTypeName string) {
	objpkg := tt.Obj().Pkg()
	isBuiltin := objpkg == nil
	if isBuiltin {
		objpkg = ds.analyzer.BuiltinPackge().PPkg.Types
	} else if objpkg != docPkg.PPkg.Types {
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, objpkg.Path()), page, objpkg.Name())
		page.Write(period)
	}
	ttName := alternativeTypeName
	if ttName == "" {
		ttName = tt.Obj().Name()
	}
	//page.WriteString(tt.Obj().Name())
	if isBuiltin || collectUnexporteds || tt.Obj().Exported() {
		buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, objpkg.Path()), page, ttName, "name-", tt.Obj().Name())
	} else {
		p := ds.analyzer.PackageByPath(objpkg.Path())
		if p == nil {
			panic("should not")
		}
		ttPos := p.PPkg.Fset.PositionFor(tt.Obj().Pos(), false)
		//log.Printf("============ %v, %v, %v", tt, pkg.Path, ttPos)
		writeSrouceCodeLineLink(page, p, ttPos, ttName, "")
	}

}

func (ds *docServer) writeValueTType(page *htmlPage, tt types.Type, docPkg *code.Package, writeFuncKeyword bool, forTypeName *code.TypeName) {
	switch tt := tt.(type) {
	default:
		panic("should not")
	case *types.Named:
		if forTypeName != nil && tt == forTypeName.Denoting().TT {
			page.WriteString(tt.Obj().Name())
		} else {
			ds.writeTypeName(page, tt, docPkg, "")
		}
	case *types.Basic:
		if forTypeName != nil && tt == forTypeName.Denoting().TT {
			page.WriteString(tt.Name())
		} else {
			buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, "builtin"), page, tt.Name(), "name-", tt.Name())
		}
	case *types.Pointer:
		page.Write(star)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Array:
		page.Write(leftSquare)
		fmt.Fprintf(page, "%d", tt.Len())
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Slice:
		page.Write(leftSquare)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Map:
		page.Write(mapKeyword)
		page.Write(leftSquare)
		ds.writeValueTType(page, tt.Key(), docPkg, true, forTypeName)
		page.Write(rightSquare)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Chan:
		if tt.Dir() == types.RecvOnly {
			page.Write(chanDir)
			page.Write(chanKeyword)
		} else if tt.Dir() == types.SendOnly {
			page.Write(chanKeyword)
			page.Write(chanDir)
		} else {
			page.Write(chanKeyword)
		}
		page.Write(space)
		ds.writeValueTType(page, tt.Elem(), docPkg, true, forTypeName)
	case *types.Signature:
		if writeFuncKeyword {
			page.Write(funcKeyword)
			//page.Write(space)
		}
		page.Write(leftParen)
		ds.writeTuple(page, tt.Params(), docPkg, tt.Variadic(), forTypeName)
		page.Write(rightParen)
		if rs := tt.Results(); rs != nil && rs.Len() > 0 {
			page.Write(space)
			if rs.Len() == 1 && rs.At(0).Name() == "" {
				ds.writeTuple(page, rs, docPkg, false, forTypeName)
			} else {
				page.Write(leftParen)
				ds.writeTuple(page, rs, docPkg, false, forTypeName)
				page.Write(rightParen)
			}
		}
	case *types.Struct:
		page.Write(structKeyword)
		//page.Write(space)
		page.Write(leftBrace)
		ds.writeStructFields(page, tt, docPkg, forTypeName)
		page.Write(rightBrace)
	case *types.Interface:
		page.Write(interfaceKeyword)
		//page.Write(space)
		page.Write(leftBrace)
		ds.writeInterfaceMethods(page, tt, docPkg, forTypeName)
		page.Write(rightBrace)
	}
}

func (ds *docServer) writeTuple(page *htmlPage, tuple *types.Tuple, docPkg *code.Package, variadic bool, forTypeName *code.TypeName) {
	n := tuple.Len()
	for i := 0; i < n; i++ {
		v := tuple.At(i)
		if v.Name() != "" {
			page.WriteString(v.Name())
			page.WriteByte(' ')
		}
		if i == n-1 {
			if variadic {
				st, ok := v.Type().(*types.Slice)
				if !ok {
					panic("should not")
				}
				page.WriteString("...")
				ds.writeValueTType(page, st.Elem(), docPkg, true, forTypeName)
			} else {
				ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
			}
		} else {
			ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
			page.WriteString(", ")
		}
	}
}

func (ds *docServer) writeStructFields(page *htmlPage, st *types.Struct, docPkg *code.Package, forTypeName *code.TypeName) {
	n := st.NumFields()
	for i := 0; i < n; i++ {
		v := st.Field(i)
		if v.Embedded() {
			// ToDo: try to find ast representation of the types of all variables.
			//       Otherwise, the embedded interface type aliases info are lost.
			// This is a suboptimal implementaiuon.
			if tn, ok := v.Type().(*types.Named); ok {
				ds.writeTypeName(page, tn, docPkg, v.Name())
			} else {
				page.WriteString(v.Name())
			}
		} else {
			page.WriteString(v.Name())
			page.WriteByte(' ')
			ds.writeValueTType(page, v.Type(), docPkg, true, forTypeName)
		}
		if i < n-1 {
			page.WriteString("; ")
		}
	}
}

func (ds *docServer) writeInterfaceMethods(page *htmlPage, it *types.Interface, docPkg *code.Package, forTypeName *code.TypeName) {
	//n, m := it.NumEmbeddeds(), it.NumExplicitMethods()
	//
	//for i := 0; i < m; i++ {
	//	f := it.ExplicitMethod(i)
	//	page.WriteString(f.Name())
	//	//page.WriteByte(' ')
	//	ds.writeValueTType(page, f.Type(), docPkg, false)
	//	if i < m-1 {
	//		page.WriteString("; ")
	//	}
	//}
	//if n > 0 && m > 0 {
	//	page.WriteString("; ")
	//}
	//
	//for i := 0; i < n; i++ {
	//	named := it.Embedded(i)
	//	ds.writeValueTType(page, named.Obj().Type(), docPkg, false)
	//	if i < n-1 {
	//		page.WriteString("; ")
	//	}
	//}

	// ToDo: try to find ast representation of the types of all variables.
	//       Otherwise, the embedded interface type aliases info are lost.
	// This is a suboptimal implementaiuon.
	var k = it.NumMethods()
	for i := 0; i < k; i++ {
		f := it.Method(i)
		page.WriteString(f.Name())
		//page.WriteByte(' ')
		ds.writeValueTType(page, f.Type(), docPkg, false, forTypeName)
		if i < k-1 {
			page.WriteString("; ")
		}
	}
}

var (
	blankID          = []byte("_")
	space            = []byte(" ")
	leftParen        = []byte("(")
	rightParen       = []byte(")")
	period           = []byte(".")
	comma            = []byte(", ")
	semicoloon       = []byte("; ")
	ellipsis         = []byte("...")
	star             = []byte("*")
	leftSquare       = []byte("[")
	rightSquare      = []byte("]")
	leftBrace        = []byte("{")
	rightBrace       = []byte("}")
	mapKeyword       = []byte("map")
	chanKeyword      = []byte("chan")
	chanDir          = []byte("&lt;-")
	funcKeyword      = []byte("func")
	structKeyword    = []byte("struct")
	interfaceKeyword = []byte("interface")

	BoldTagStart = []byte("<b>")
	BoldTagEnd   = []byte("</b>")
)

// This is a rewritten of WriteTypeEx.
// Please make sure w.Write never makes errors.
// "forTypeName", if it is not blank, should be declared in docPkg.
// ToDo: "too many fields/methods/params/results" is replaced with ".....".
func (ds *docServer) WriteAstType(w *htmlPage, typeLit ast.Expr, codePkg, docPkg *code.Package, funcKeywordNeeded bool, recvParam *ast.Field, forTypeName *code.TypeName) {
	switch node := typeLit.(type) {
	default:
		panic(fmt.Sprintf("WriteType, unknown node: %[1]T, %[1]v", node))
	//>> 1.18
	case *astUnaryExpr:
		w.WriteString(node.Op.String())
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
	case *astBinaryExpr:
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
		w.Write(space)
		w.WriteString(node.Op.String())
		w.Write(space)
		ds.WriteAstType(w, node.Y, codePkg, docPkg, true, nil, forTypeName)
	case *astIndexExpr:
		// ast.Ident or ast.SelectorExpr
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
		w.Write(leftSquare)
		ds.WriteAstType(w, node.Index, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightSquare)
	case *astIndexListExpr:
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
		w.Write(leftSquare)
		for i := range node.Indices {
			if i > 0 {
				w.Write(comma)
			}
			ds.WriteAstType(w, node.Indices[i], codePkg, docPkg, true, nil, forTypeName)
		}
		w.Write(rightSquare)
	//<<
	case *ast.Ident:
		// obj := codePkg.PPkg.TypesInfo.ObjectOf(node)
		// The above one might return a *types.Var object for embedding field.
		// So us the following one instead, to make sure it is a *types.TypeName.

		//>> ToDo: Go 1.18, obj might be a type parameter now!
		obj := codePkg.PPkg.TypesInfo.ObjectOf(node)
		if obj != nil { // !!! The identifers in builtin package have not objects!
			if _, ok := obj.Type().(*typesTypeParam); ok {
				w.WriteString(node.Name)
				return
			}
		}
		//<<

		obj = codePkg.PPkg.Types.Scope().Lookup(node.Name)
		if obj == nil {
			obj = types.Universe.Lookup(node.Name)
		}
		if obj == nil {
			//log.Printf("%s, %s: %s", docPkg.Path(), codePkg.Path(), node.Name)
			//panic("should not")

			// It really should panic here, but to make it tolerable,

			w.WriteString(node.Name)
			return
		}

		tn, ok := obj.(*types.TypeName)
		if !ok {
			panic(fmt.Sprintf("object should be a TypeName, but %T, %v.\nObject: %T", obj, obj, codePkg.PPkg.TypesInfo.ObjectOf(node)))
		}
		objType := tn.Type()

		objpkg := obj.Pkg()
		isBuiltin := objpkg == nil
		if isBuiltin {
			objpkg = ds.analyzer.BuiltinPackge().PPkg.Types
		} else if objpkg != docPkg.PPkg.Types {
			buildPageHref(w.PathInfo, createPagePathInfo1(ResTypePackage, objpkg.Path()), w, objpkg.Name())
			w.Write(period)
		}

		// ToDo: faster way: ds.analyzer.TryRegisteringType(objType) == forTypeName.Denoting()?
		if forTypeName != nil && types.Identical(objType, forTypeName.Denoting().TT) {
			w.Write(BoldTagStart)
			defer w.Write(BoldTagEnd)
		}

		if objpkg == docPkg.PPkg.Types && forTypeName != nil && node.Name == forTypeName.Name() {
			w.WriteString(node.Name)
		} else if docPkg.Path == "builtin" {
			if obj.Exported() { // like Type
				w.WriteString(node.Name)
			} else { // like int
				buildPageHref(w.PathInfo, createPagePathInfo1(ResTypePackage, objpkg.Path()), w, node.Name, "name-", node.Name)
			}
		} else if isBuiltin || collectUnexporteds || obj.Exported() {
			buildPageHref(w.PathInfo, createPagePathInfo1(ResTypePackage, objpkg.Path()), w, node.Name, "name-", node.Name)
		} else {
			p := ds.analyzer.PackageByPath(objpkg.Path())
			if p == nil {
				panic("should not")
			}
			ttPos := p.PPkg.Fset.PositionFor(obj.Pos(), false)
			//log.Printf("============ %v, %v, %v", tt, pkg.Path, ttPos)
			writeSrouceCodeLineLink(w, p, ttPos, node.Name, "")
		}
	case *ast.SelectorExpr:
		pkgId, ok := node.X.(*ast.Ident)
		if !ok {
			panic("should not")
		}
		importobj := codePkg.PPkg.TypesInfo.ObjectOf(pkgId)
		if importobj == nil {
			panic("should not")
		}
		pkgobj := importobj.(*types.PkgName)
		if pkgobj == nil {
			panic("should not")
		}
		pkgpkg := pkgobj.Imported()
		if pkgpkg == nil {
			panic("should not")
		}
		if pkgpkg != docPkg.PPkg.Types {
			//w.WriteString(pkgpkg.Name())
			buildPageHref(w.PathInfo, createPagePathInfo1(ResTypePackage, pkgpkg.Path()), w, pkgId.Name)
			w.Write(period)
		}

		//log.Println(pkgId.Name, node.Sel.Name, pkgpkg.Path(), codePkg.Path())
		obj := pkgpkg.Scope().Lookup(node.Sel.Name)
		if obj.Pkg() != pkgpkg {
			//panic("should not")

			// It really should panic here, but to make it tolerable,

			w.WriteString(node.Sel.Name)

			return
		}
		tn, ok := obj.(*types.TypeName)
		if !ok {
			panic(fmt.Sprintf("%v is a %T, not a type name", obj, obj))
		}

		if forTypeName != nil && types.Identical(tn.Type(), forTypeName.Denoting().TT) {
			w.Write(BoldTagStart)
			defer w.Write(BoldTagEnd)
		}

		if pkgpkg == docPkg.PPkg.Types && forTypeName != nil && node.Sel.Name == forTypeName.Name() {
			w.WriteString(node.Sel.Name)
		} else if collectUnexporteds || obj.Exported() { // || isBuiltin { // must not be builtin
			buildPageHref(w.PathInfo, createPagePathInfo1(ResTypePackage, pkgpkg.Path()), w, node.Sel.Name, "name-", node.Sel.Name)
		} else {
			//w.WriteString(node.Sel.Name)
			p := ds.analyzer.PackageByPath(pkgpkg.Path())
			if p == nil {
				panic("should not")
			}
			ttPos := p.PPkg.Fset.PositionFor(obj.Pos(), false)
			//log.Printf("============ %v, %v, %v", tt, pkg.Path, ttPos)
			writeSrouceCodeLineLink(w, p, ttPos, node.Sel.Name, "")
		}
	case *ast.ParenExpr:
		w.Write(leftParen)
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightParen)
	case *ast.StarExpr:
		w.Write(star)
		ds.WriteAstType(w, node.X, codePkg, docPkg, true, nil, forTypeName)
	case *ast.Ellipsis: // possible? (yes, variadic parameters)
		//panic("[...] should be impossible") // ToDo: go/types package has a case.
		//w.Write(leftSquare)
		w.Write(ellipsis)
		//w.Write(rightSquare)
		ds.WriteAstType(w, node.Elt, codePkg, docPkg, true, nil, forTypeName)
	case *ast.ArrayType:
		w.Write(leftSquare)
		if node.Len != nil {
			tv, ok := codePkg.PPkg.TypesInfo.Types[node.Len]
			if !ok {
				panic(fmt.Sprint("no values found for ", node.Len))
			}
			w.WriteString(tv.Value.String())
		}
		w.Write(rightSquare)
		ds.WriteAstType(w, node.Elt, codePkg, docPkg, true, nil, forTypeName)
	case *ast.MapType:
		w.Write(mapKeyword)
		w.Write(leftSquare)
		ds.WriteAstType(w, node.Key, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightSquare)
		ds.WriteAstType(w, node.Value, codePkg, docPkg, true, nil, forTypeName)
	case *ast.ChanType:
		if node.Dir == ast.RECV {
			w.Write(chanDir)
			w.Write(chanKeyword)
		} else if node.Dir == ast.SEND {
			w.Write(chanKeyword)
			w.Write(chanDir)
		} else {
			w.Write(chanKeyword)
		}
		w.Write(space)
		ds.WriteAstType(w, node.Value, codePkg, docPkg, true, nil, forTypeName)
	case *ast.FuncType:
		if funcKeywordNeeded {
			w.Write(funcKeyword)
			//w.Write(space)
		}
		w.Write(leftParen)
		ds.WriteAstFieldList(w, node.Params, true, comma, codePkg, docPkg, true, recvParam, forTypeName)
		w.Write(rightParen)
		if node.Results != nil && len(node.Results.List) > 0 {
			w.Write(space)
			if len(node.Results.List) == 1 && len(node.Results.List[0].Names) == 0 {
				ds.WriteAstFieldList(w, node.Results, true, comma, codePkg, docPkg, true, nil, forTypeName)
			} else {
				w.Write(leftParen)
				ds.WriteAstFieldList(w, node.Results, true, comma, codePkg, docPkg, true, nil, forTypeName)
				w.Write(rightParen)
			}
		}
	case *ast.StructType:
		w.Write(structKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		ds.WriteAstFieldList(w, node.Fields, false, semicoloon, codePkg, docPkg, true, nil, forTypeName)
		w.Write(rightBrace)
	case *ast.InterfaceType:
		w.Write(interfaceKeyword)
		//w.Write(space)
		w.Write(leftBrace)
		ds.WriteAstFieldList(w, node.Methods, false, semicoloon, codePkg, docPkg, false, nil, forTypeName)
		w.Write(rightBrace)
	}
}

func (ds *docServer) WriteAstFieldList(w *htmlPage, fieldList *ast.FieldList, isParamOrResultList bool, sep []byte, codePkg, docPkg *code.Package, funcKeywordNeeded bool, recvParam *ast.Field, forTypeName *code.TypeName) {
	if fieldList == nil {
		return
	}
	showRecvName := recvParam != nil && len(recvParam.Names) > 0
	showParamsNames := isParamOrResultList && len(fieldList.List) > 0 && len(fieldList.List[0].Names) > 0
	showParamsNames = showParamsNames || showRecvName

	fields := fieldList.List
	if recvParam != nil {
		fields = append([]*ast.Field{recvParam}, fields...)
	}

	for i, fld := range fields {
		if len(fld.Names) > 0 {
			for k, n := range fld.Names {
				w.Write([]byte(n.Name))
				if k+1 < len(fld.Names) {
					w.Write(comma)
				}
			}
			if funcKeywordNeeded {
				w.Write(space)
			} // else for interface methods
		} else if showParamsNames {
			w.Write(blankID)
			if funcKeywordNeeded {
				w.Write(space)
			} // else for interface methods
		}
		ds.WriteAstType(w, fld.Type, codePkg, docPkg, funcKeywordNeeded, nil, forTypeName)
		if i+1 < len(fields) {
			w.Write(sep)
		}
	}
}

//var basicKind2ReflectKind = [...]reflect.Kind{
//	types.Bool:          reflect.Bool,
//	types.Int:           reflect.Int,
//	types.Int8:          reflect.Int8,
//	types.Int16:         reflect.Int16,
//	types.Int32:         reflect.Int32,
//	types.Int64:         reflect.Int64,
//	types.Uint:          reflect.Uint,
//	types.Uint8:         reflect.Uint8,
//	types.Uint16:        reflect.Uint16,
//	types.Uint32:        reflect.Uint32,
//	types.Uint64:        reflect.Uint64,
//	types.Uintptr:       reflect.Uintptr,
//	types.Float32:       reflect.Float32,
//	types.Float64:       reflect.Float64,
//	types.Complex64:     reflect.Complex64,
//	types.Complex128:    reflect.Complex128,
//	types.String:        reflect.String,
//	types.UnsafePointer: reflect.UnsafePointer,
//}

// Only write interface
//func writeTypeKind(page *htmlPage, tt types.Type) {
//	switch tt := tt.Underlying().(type) {
//	default:
//		panic(fmt.Sprintf("should not: %T", tt))
//	case *types.Named:
//		panic("should not")
//	case *types.Basic:
//		page.WriteString(basicKind2ReflectKind[tt.Kind()].String())
//	case *types.Pointer:
//		page.WriteString(reflect.Ptr.String())
//	case *types.Struct:
//		page.WriteString(reflect.Struct.String())
//	case *types.Array:
//		page.WriteString(reflect.Array.String())
//	case *types.Slice:
//		page.WriteString(reflect.Slice.String())
//	case *types.Map:
//		page.WriteString(reflect.Map.String())
//	case *types.Chan:
//		page.WriteString(reflect.Chan.String())
//	case *types.Signature:
//		page.WriteString(reflect.Func.String())
//	case *types.Interface:
//		page.WriteString(reflect.Interface.String())
//	}
//}

//func (c *Constant) IndexString() string {
//	btt, ok := c.Type().(*types.Basic)
//	if !ok {
//		panic("constant should be always of basic type")
//	}
//	isTyped := btt.Info()&types.IsUntyped == 0
//
//	var b strings.Builder
//
//	b.WriteString(c.Name())
//	if isTyped {
//		b.WriteByte(' ')
//		b.WriteString(c.Type().String())
//	}
//	b.WriteString(" = ")
//	b.WriteString(c.Val().String())
//
//	return b.String()
//}

//func writeTypeNameIndexHTML(page *htmlPage, tn *code.TypeName)  {
//	fmt.Fprintf(page, ` type <a href="#name-%[1]s">%[1]s</a>`, tn.Name())
//}

func writeUnexportedResourcesHeader(page *htmlPage, resName string, hideInitially bool, numUnexporteds int) {
	checked := " checked"
	if hideInitially {
		checked = ""
	}

	showLabel := page.Translation().Text_UnexportedResourcesHeader(true, numUnexporteds, true)
	hideLabel := page.Translation().Text_UnexportedResourcesHeader(false, numUnexporteds, true)

	fmt.Fprintf(page, `<input type='checkbox'%[2]s class="showhide" id="unexported-%[1]s-showhide"><i><label for="unexported-%[1]s-showhide" class="show-inline">%[3]s</label><label for="unexported-%[1]s-showhide" class="hide-inline">%[4]s</label></i>`,
		resName, checked, showLabel, hideLabel)
}

func writeHiddenItemsHeader(page *htmlPage, resName, itemsCategory string, hideInitially bool, numUnexporteds int, exact bool) {
	checked := " checked"
	if hideInitially {
		checked = ""
	}

	showLabel := page.Translation().Text_UnexportedResourcesHeader(true, numUnexporteds, exact)
	hideLabel := page.Translation().Text_UnexportedResourcesHeader(false, numUnexporteds, exact)

	fmt.Fprintf(page, `<input type='checkbox'%[3]s class="showhide" id="%[1]s-showhide-%[2]s"><i><label for="%[1]s-showhide-%[2]s" class="show-inline">%[4]s</label><label for="%[1]s-showhide-%[2]s" class="hide-inline">%[5]s</label></i>`,
		resName, itemsCategory, checked, showLabel, hideLabel)
}

func writeFoldingBlock(page *htmlPage, resName, statName, contentKind string, expandInitially bool, writeTitleContent, listStatContent func()) {
	checked := ""
	if expandInitially || unfoldAllInitially {
		checked = " checked"
	}
	labelClass := ""
	if statName == "stats" {
		labelClass = ` class="stats"`
	}

	fmt.Fprintf(page, `<input type='checkbox'%[4]s class="fold" id="%[1]s-fold-%[2]s"><label%[5]s for="%[1]s-fold-%[2]s">`,
		resName, statName, contentKind, checked, labelClass)
	writeTitleContent()
	fmt.Fprintf(page, `</label><span id='%[1]s-fold-%[2]s-%[3]s' class="fold-%[3]s">`,
		resName, statName, contentKind, checked)

	listStatContent()
	page.WriteString("</span>")
}

//func writePageText(page *htmlPage, indent, text string, htmlEscape bool) {
//	writePageTextEx(page, indent, text, htmlEscape)
//}

//// func writePageTextEx(page *htmlPage, indent, text string, htmlEscape, removeOriginalIdent bool) {
//func writePageTextEx(page *htmlPage, indent, text string, htmlEscape bool) {
//	buffer := bytes.NewBufferString(text)
//	reader := bufio.NewReader(buffer)
//	notFirstLine, needAddMissingNewLine := false, false
//	for {
//		if needAddMissingNewLine {
//			page.WriteByte('\n')
//		}
//		line, isPrefix, err := reader.ReadLine()
//		if len(line) > 0 {
//			if notFirstLine {
//				page.WriteByte('\n')
//			}
//			page.WriteString(indent)
//
//			// ToDo: bug, Need to find the common prefix ident, then remove it for each line.
//			//if removeOriginalIdent {
//			//	if len(line) > 0 && line[0] == '\t' {
//			//		line = line[1:]
//			//	}
//			//}
//			needAddMissingNewLine = false
//		} else {
//			needAddMissingNewLine = true
//		}
//		if htmlEscape {
//			page.AsHTMLEscapeWriter().Write(line)
//		} else {
//			page.Write(line)
//		}
//		if errors.Is(err, io.EOF) {
//			break
//		}
//		if !isPrefix {
//			notFirstLine = true
//		}
//	}
//}

//func (ds *docServer) writePageText(page *htmlPage, ident, text string) {
//	ds.docRenderer.Render(page, text, ident, true, nil)
//}

var asciiCharTypes [128]byte

func init() {
	setRange := func(from, to rune, t byte) {
		for i := from; i <= to; i++ {
			asciiCharTypes[i] = t
		}
	}
	setRange('0', '9', 1)
	setRange('a', 'z', 2)
	setRange('A', 'Z', 2)
	setRange('_', '_', 2)
}

func (ds *docServer) renderDocComment(page *htmlPage, currentPkg *code.Package, ident, mdDoc string) {
	var makeURL func(string) string
	if renderDocLinks {
		makeURL = func(bracketedText string) (r string) {
			//defer func() {
			//	if r != "" {
			//		println("===", currentPkg.Path, bracketedText, r)
			//	}
			//}()

			checkResNameRoughly := func(resName string) bool {
				if len(resName) == 0 {
					return false
				}
				var k = 0
				if resName[0] <= 128 {
					if asciiCharTypes[resName[0]] != 2 {
						return false
					} else {
						k = 1
					}
				}
				for i, r := range resName[k:] {
					if r >= 128 {
						continue
					}
					if asciiCharTypes[r] == 0 {
						return false
					}
					if i >= 16 {
						break
					}
				}
				return true
			}

			bracketedText = strings.TrimLeft(bracketedText, "*")

			tokens := strings.Split(bracketedText, ".")
			if len(tokens) == 0 {
				return ""
			}

			buildPkgResLink := func(pkg *code.Package, res code.Resource) string {
				if res.Exported() || collectUnexporteds {
					return buildPageHref(page.PathInfo, createPagePathInfo1(ResTypePackage, pkg.Path), nil, "", "name-", res.Name())
				}

				return buildSrouceCodeLineLink(page.PathInfo, ds.analyzer, pkg, res.Position())
			}

			if len(tokens) == 1 {
				if !checkResNameRoughly(tokens[0]) {
					return ""
				}

				res := currentPkg.SearchResourceByName(tokens[0])
				if res == nil {
					return ""
				}

				return buildPkgResLink(currentPkg, res)
			}

			if !checkResNameRoughly(tokens[1]) {
				return ""
			}

			var trySearchRes = func(pkgPath string) (*code.Package, code.Resource) {
				pkg := currentPkg.Module().PackageByPath(pkgPath)
				if pkg == nil {
					return nil, nil
				}
				res := pkg.SearchResourceByName(tokens[1])
				if res == nil {
					return nil, nil
				}
				return pkg, res
			}

			var res code.Resource
			var pkg = ds.analyzer.StandardPackage(tokens[0])
			if pkg != nil {
				res = pkg.SearchResourceByName(tokens[1])
				if res == nil {
					pkg = nil
				}
			}
			if res == nil {
				pkg, res = trySearchRes(currentPkg.Path + "/" + tokens[0])
			}
			if res == nil {
				i := strings.LastIndexByte(currentPkg.Path, '/')
				if i > 0 {
					pkg, res = trySearchRes(currentPkg.Path[:i+1] + tokens[0])
				}
			}
			if res == nil {
				pkg, res = trySearchRes(currentPkg.ModulePath() + "/" + tokens[0])
			}

			if len(tokens) == 2 {
				if res != nil {
					return buildPkgResLink(pkg, res)
				}

				tn := currentPkg.TypeNameByName(tokens[0])
				if tn == nil {
					return ""
				}

				sel := tn.Denoting().SelectorByName(tokens[1])
				if sel == nil {
					return ""
				}

				pkg = sel.Package()
				if pkg == nil {
					pkg = currentPkg
				}

				return buildSrouceCodeLineLink(page.PathInfo, ds.analyzer, pkg, sel.Position())
			}

			// ToDo: more complex patter: StructType.Field.Field, pkg.StructType.Field.Field, ...
			if len(tokens) > 3 {
				return ""
			}

			// Now, only consider one case: pkg.Type.Selector

			if res == nil {
				return ""
			}

			tn, ok := res.(*code.TypeName)
			if !ok {
				return ""
			}

			sel := tn.Denoting().SelectorByName(tokens[2])
			if sel == nil {
				return ""
			}

			pkg = sel.Package()
			if pkg == nil {
				pkg = currentPkg
			}

			return buildSrouceCodeLineLink(page.PathInfo, ds.analyzer, pkg, sel.Position())
		}
	}
	page.WriteString(`<span class="md-text">`)
	ds.docRenderer.Render(page, mdDoc, ident, true, makeURL)
	page.WriteString(`</span>`)
}
