

### Soon to do

* handle inks in comments: https://tip.golang.org/doc/comment, not support lists and headers
  * not handle [otherModulePkg.Name]
  * try to handle [sameModulePkg.Name] and [Name]
  * handle [XYZ] + [XYZ]: link
  * handle bare urls.

* show alias list for types, or identical tyoe list

* show id ref counts, and ref counts should affect populirities.

* add a debug flag, help users collect info

* details page: add a "+" before package, click it to show parent (and module root) packages
  * if a package doesn't belongs the module, italic it.

* use different default flag values for gocore, godoge and golds

* js: auto link "golang.org/x/..." etc.

* add a "typ" page kind, for unnamed types, because unnamed types don't belong to any packages.
  * now functions are not listed as values of
    type NetworkProtocolFactory func(*Stack) NetworkProtocol
    Maybe it is good to not list, but need a way to find these values.

  * info on this page (same as a defined type)
    * aliases of the type
    * underlied types
    * uses places
    * ...

* now, not collect uses for unnamed struct type fields.
  In the following code, only collect for x1 and y1
    var a struct {
      x int
      y struct {
        m bool
      }
    }
    use a fake type alias named with its hash for the struct type? 
    So that the nested field could be used in ref page urls.
    
    example: golang.zx2c4.com/wireguard/device.Device
    
    only do this when --source-code-reading=rich
    
* trace nested field: aPkg.Device.net.port for ref pages.
  Not a good solution! This problen should be the same as the last one.
  We need to use a fake type alias to "struct{port uint16}"
  and use "theAlias.port" to denote the ref id.
  
	type Device struct {
		net struct {
			port          uint16 // listening port
		}
	}

* the handling of "removeOriginalIdent" in output docs is not very reasonable

* golds gopath

* id introduced in version: 1.15-, 1.16, 1.17, ...
* check "Deprecated: " in comments

* generics
  * use https://pkg.go.dev/golang.org/x/exp/typeparams ?
  * click a type param, highlight all its refs in package details package
    * id hightlight in source code optimization: enclose each function in a div
      or each function is a hight scope unit: multiple highlighting
      * Note: a function literal might be enclosed in a package-level type spec.
  * for generic type/function, list its instances
  * for generic type, list values of each its instance
  * method implementation
    * method prototypes without or with TypeParam:
      whether or not list instanced methods
  * type implementations
    * types whose method prototypes without or with TypeParam:
      whether or not list instanced types
  * check d.forbidRegisterTypes voilations by generic code

* pure terminal mode
  https://old.reddit.com/r/golang/comments/tueopt/how_to_get_a_list_of_types_conforming_to_a/i33dn64/?context=3
  * interactive or not.
  * non-interactive needs to cache analysis result.

* use https://pkg.go.dev/golang.org/x/tools/go/buildutil to replace some go command runs.
  * how to pass -tags "tag1 tag2" options in "go build"

* options
    //	-package-docs-showing-initially=collapse|simple|expand (cancelled)
    //	-identifier-docs-showing-initially=collapse|oneline|expand (cancelled, but show one line defaultly)

* https://golang.org/pkg/go/doc/#Package
  bugs and notes, Examples

* <a class="deplucated">xxx</a><a>yyy</a> should change to <a><span class="deplucated">xxx</span>yy</a>

* text searching

* static analysis
  * mark unused variables (global, local, parameters, receiver)
    * some unuseds should be skipped (ex. a method implementing an interface)
  * ...

* some stat list is blank, but title is still shown

* show/warning Trojan Source https://news.ycombinator.com/item?id=29061987

* use padding instead of indent tabs

* support https://github.com/go101/golds/issues/25
  syupport "golds aModule@version"
  * create a temp dir to process

* show/run examples/tests/banchmarks 
  (Tests==true, cause reflect.EmbedWithUnexpMeth not found in analyzePackage_ConfirmTypeSources/registerDirectFields now)
  * use custom implementation? Ast load example_xxx_test.go file only, ...
    * collectionDeclarations ranges sourceFiles
    * load all example source codes in memory
    * render examples code in package details pages
  * https://blog.golang.org/examples

* module page. Containing Module: xxxx/xxxx
  * sort by requiredBys / line of codes
  * wait https://github.com/golang/go/issues/45649 to be fixed
  * show project links
  * show which packages are used in each required modules
  * package details page
    basic information
          import path: ...
          parent package: 
          belonging module:

* link to pkg.go.dev: with query params
  * GOOS, ....: https://github.com/golang/go/issues/44356

* dep page: list the importing source files.

* tests only work for Linux now.

* one-page doc for private packages: https://github.com/go101/golds/issues/19

* seperate comment and code in reading: https://github.com/go101/golds/issues/21

* pkg details page: show values by file/position order (only for javascript on)
* search ids on pkg details/overview pages

* hotley: HOME - to overview page.
  * in gen code, weite a hidden element which text is the overview page relative url.

* embed playground
  * to run examples

* The following function should be shown in the AsOutputOf lists of the Option and renderer.Option types.
   func WithXHTML() interface{Option; renderer.Option}
  * similarly for AsInputOf lists

* implicit
  * switch expr := srcNode.(type) { // this expr might need to be enclosed in mutilple labels
    case T1: _ = expr
    case T2: _ = expr
    }
* now, for "type T struct {m sync.Mutex}", "var t T", "t.m.Lock" will be registered to "sync.Mutex.Lock",
  instead of "T.Lock()"
* s = StructTypeFoo{} // unkeyed struct assignment should be viewed as full-keyed assignment: need recored in uses lists.


* type alias and same-underlyings list
  * https://github.com/golang/go/issues/44905

* show more values in type-of lists
  * type F func(), then list all "func xxx()" for type F
  * type S []T, then list all "[]T" values for type S (ex. image/color.Palette, list image/color.[]color.Color values)
  * list all values of implementors of an interface type I for I
  * ...

* stat: keyword use count: most implemented interface.

* also grey the same parts in asInputsOf/.... lists

* uses page filter: declartions | value destination | value source | in std | out of std
  * writes includes (v=x, field:x, and Struct{x}, ...)
  * as Type, as Field (for embedding field)

* move most readme content to go101.org.
  * keep case table, simple install, simple feature overview, simple usage (golds std, golds ./...)

* use "go.lds" config file for docs generation.
  * -use-config=true and -config=go.lds for -gen defaultly 
  * or use comment lines in go.mod (not recommended):
    // golds -nouses ./... # configX
    // golds -only-list-exporteds -source-code-reading=external ./... # configY

* some functions are called in .s files, ..., uses pages miss them

* code page: each function enclosed in a span so that local id hightlighting needs less time.
* reduce code page size
  * some buildIdentifier -> buildLink
  * use short class names: codeline => l
  * use short tag names
  * no need to <code></code> in each code line.
  * no need <span class="codeline", use pre > code > span in css instead.
  * replace \t\t\t with margin-left

* use css chart instead of svg? https://chartscss.org/

* css style
  * https://github.com/go101/golds/issues/13#issuecomment-769154192
    bigger font maybe. Matching the font at https://golang.org/pkg/ (Roboto 16px, 1.3em line height)
* enhance tests
  * test by ast comments
* add more comments, and clear some

* show statistics floating on the right of the overview page.

* use tree view for overview page to show modules.




* add "d➜" in overview page: hover on a package: show its brief intro.
* more hovers:
  * in code, show tooltip as the full selector path for shortened selectors.

* now, there is not a way to view the uses of embedded fields (control+click?)

* add a button on overview page to do static analysis

* -format=[html|json|txt|md]

* show which packages are CVS dirty in overview page.



* uses pages: show package reference list (for example, find all unsafe uses)
* id uses need consider whether or not the id is promoted.
  For promoted selectors, the receiver arg's type must be checked
  Need an option on page: check the owner type of selectors or not.
* uses pages: also count some implicit uses, such as unkeyed struct literals
* show identifier uses: use fake ids for some cases
  * unnamed types ([192]uint64, []*debug/dwarf.TypedefType, ...)
  * string literals
  * fields of package-level unnamed structs (current no ways to represent as TypeName.field, need a fake typename)
    * even for named types, its files obtained by embedding have not definitions, so now uses are not collected for them
  * methods of unnamed stricts (obtained by embedding, now uses are not collected for them)
  * filter: only show those in type specifications
  * // ToDo: the above code works for the "bar" and "baz" fields, but not for the "X" field.
				//
				// type Foo struct {
				// 	bar Type
				//	baz struct {
				//		X int
				//	}
				//}
				//
				// There are two ways to solve this problem:
				// 1. create a fake type name "unamed-12345" and use "unamed-12345.X" to denote the X field.
				// 2. modify ref-user page implementation to support "Foo.baz.X" (not recommended, may have loop problem).


* some buildPageHref can make page != nil
    and buildPageHref should be a method of DocServer
    add a Pkg field for pagePathInfo to optimize?
  
* use js to fold functions in code pages
  * use js to fold interface method implementations
* use js to check input in :target in onload, expend it if applicable
* add ol=nn,nn querystring to source page: srcpage?#line-nnn&cols=nn,nn+mm,mm
  so that JS can hightlight the identifier. Multiple id instances might exist in one line
* use cookie to remember options: show-unexporteds, sort-by

* field list: align them as which in code. But need to consider embedding chain...

* calculate value importance:
  * result/param type popularity matter
  * number of uses matter

* about https://github.com/go101/golds/issues/9 and to avoid depings affecting depeds' docs:
  * need to implement the module aware features firstly.
    https://github.com/rogpeppe/go-internal
  * std packages are in a std module
  * note: the dependencies of modules can be bidirectional
  * within a module: allow mutual references  
    for two packages not in the same module, only deping can reference deped.
  * assume v1.x.y doesn;t break v1.m.n (where x.y > m.n)
  * this is a hard problem without solutions. Close this issue?
    It is a problem which looks simple but actually hard intrinsicly.
    Golds deffers from godoc in that Golds generated docs of a packages depends on the packages depend on it.
    Mention docs size could be reduced much by using -source-code-reading=external. 

* modify the cache system to only cache most visited and recent ones

* method docs 
  * how to handle duplicated methods caused by interface embedding interfaces.
    Their docs might be different.

* rate limit http requests. 1000requests/3600seconds

* server state:
  * highlight id 0-n
  * searching uses for id goroutine 0-n
  (forget what these means)



* overview page: show std pkgages only
  * need maintain a seperated depHeight/depDepth for std module internally.

* search (non-semantic search, pure word searching)
  * ref: https://github.com/g-harel/gothrough

* gen mode: merge docs for several (GOOS, GOARCH) compositions. At least for std.


* Rewrite some implemenrations
  * global.pacakgeList, each pkg has a unique id (int32)
  * global.functionProtoypes, each has a unique id (int32)
  * global.identifierList, each has unique id
  * global.selectorNameIds {pkgId, identId int32}
  * global.methodPrototypes {selNameId, funcProtoId}, each correspods a unique id (int32(
  * global.method2typesTable map[methodProtoId][]*type. All the []*type share a common big []*type slice.
    The length of the big slice is sum(type[i].methodCount)

### 1.0 milestore must do

* remove Golds version from footer to avoid modifying all pages when using a new golds version.
* custom styles and support godoc style

### More to do

* graphics
  * show dep relations
    * filter: within a module or project

* module support
  * show mobule dependencies: "go mod graph" 

* some "embedding" in names should be "embedded"

* For std pacakges: show which version of Go introduced a particular function/type, etc.
  * or for any modules
  * note: https://github.com/golang/go/issues/44081
  * use godoc data for history data before Go 1.16.

* go-callvis like, call relations

* change theme and language

* FindPackageCommonPrefixPaths(pa, pb string) string
	ToDo: ToLower both?

* parse more source files
  * .s file syntax support
  * better cgo support, parse c code.
    * https://gitlab.com/cznic/ccgo
    * use original Go files and parse c files
    * https://godoc.org/github.com/cznic/cc#example-Statement
      https://pkg.go.dev/github.com/cznic/cc/v2?tab=doc
    * https://github.com/xlab/c-for-go
    * https://github.com/elliotchance/c2go
    * https://github.com/gotranspile/cxgo
    * port tinycc
    * https://github.com/DQNEO/8cc.go
    * is LSIF helpful?
	  https://lsif.dev/
	  lsif-c++ for cgo etc.

* list .md files and render markdown files

* use css fixate the top file path bar.

* special handling for the buitlin page, 
  // * make(Type ChannelKind|MapKind|SliceKind, sizes ...int) Type
  //   Type must denote a channel, map, or slice type.
  //   make(Type ChannelKind|MapKind|SliceKind, size integer) Type
  //   Type must denote a channel, map, or slice type.
  //   size must be a non-negative integer value (of any integer type) or a literal denoting a non-negative integer value.
  //   make(Type SliceKind, length integer, capacity interger) Type
  //   Type must denote a slice type.
  //   length and capacity must be both non-negative integer values (of any integer type) or literals denoting a non-negative integer values.
  //   The types of length and capacity may be different.
  // * new(Type AnyKind) *Type
  // * each with simple examples


* module info
* code search

* support multi GOOS pages, show all OS specified packages
  * show used build tags and other available ones
  * https://stackoverflow.com/questions/7044944/jquery-javascript-to-detect-os-without-a-plugin
    https://golang.org/pkg/syscall/ list all OS/arch pages
	//godo/doc/builder
	var windowsOnlyPackages = map[string]bool{
		"internal/syscall/windows":                     true,
		"internal/syscall/windows/registry":            true,
		"golang.org/x/exp/shiny/driver/internal/win32": true,
		"golang.org/x/exp/shiny/driver/windriver":      true,
		"golang.org/x/sys/windows":                     true,
		"golang.org/x/sys/windows/registry":            true,
	}

* also list unexported resources in code reading mode
  * collect unexported functions for types (asInputs/asOutputs/...)
  * ...

* packakge list
  * show by alpha order / by importedBys / by dependency height
  * if last token in import path and package name are different, mention it
  * list packages by one module, one background color
  * exclude dependency packages
* for all exported values,
  * filter: func | var | const | group by type | ...
  * find other values with the same type
  * function: hints: will an argument be modified in function body
* stat:
  * top N lists
  * top N used identifers
  * function stats also consider vars of function types.
  * all stats also consider unexported global and local resources
  * stat number of non-std packages, and non-std dependencies for each package, 
  * for an identifier, stat how many packages use it.
* package details
  * add parent and children packages
* imports
  * add links for import declarations
* docs for unepxorted types/vars
  * unnamed type: find all occurrences (use fake type ids)
  * (done) the promoted methods and fields of unexported fields
  * the exported methods and fields of exported variables of unexported types.
  * the exported methods and fields of results of unexported types of exported functions (or of fields of visible structs).
  visible structs mean the ones returned by exported functions or exported struct types.
  * the exported methods and fields of the exported alias of unexported types
* for a type
  * show the types with the same underlying type. (if is sturct, filer: ignore field tags or not)
  * as field types of, and embedded in n types
  * show filed tags in docs
  * show comparable/embeddable or not. Fill TypeInfo.attributes.
  * all alias list
  * values which can be converted to (some functions can be used as (implicitly converted to) http.HandleFunc values, alike)    
  * asParams/asResults lists exclude the methods of unexported types now.
  * asTypeOf items: sort by value | sort by code position | sort by name
  * method: show whether or not is promoted
  * as fields of types list (and embedded-in list, this is important, must do)
  * for interface: subset of list
  * for non-interface: embedded by list. (maybe it is better to add some filters to id-use pages: only show those in type specifications)
  * convertible/assignable types
  * show struct paddings/sizes
  * filter by kind
  * as-type / as-params / as-results lists detail:
    * merge method with the same signature
  * as-type: also combine values of []T, chan T, etc. (now only combine values of T and *T)
  * implementBy and implement lists should include exported aliases of unnmaed types.
    * show show a "==" label if the implementor and the implemented are equal.
  * it is important to find a way to list implemented unexported types, which is good for code reading.
  * list variables of function types in asParams and asResults lists.
  * for function types, also list functions of its underlying type as values
* an interface method might also has multiple docs, for interface embed overlapping interfaces

* custom type checker? refs:
  * go/*
  * https://go.googlesource.com/go/+/refs/heads/dev.typeparams/src/cmd/compile/internal/types2/

### Done

* (done) hotkey
  * overview page
    d - show one line docs
  * pacakge details page
    p - toggle package docs collapse/expand
    t - toggle types docs collapse/expand
    f - toggle functions docs collapse/expand
    v - toggle variables docs collapse/expand
    c - toggle constants docs collapse/expand
    a - toggle all docs collapse/expand
* (done) wdpkgs-listing=solo: from a dep pkg details page to overview, auto show the hidden one ...
* (giveup) support gopath psudo module name? https://groups.google.com/g/golang-nuts/c/-pmx4eksLpA
* (done) link code to external source hosting website:
* (done) Sort pkgs by LOC
* (done) stat: lines of code. ave. lines/file.
* (done) code: click an import path to highlight all its uses in the current file
* (done) top list: most parameters, most results
* (done) list contained resources under each source file (folding initally)
* (done) supports "golds a.go", need change "builtin" parse handling. remove: args = append(args, "builtin")
* (done) sort packages: ab-cd should after ab/xy
* (done) put unexported function in asParams/asResult lists
* (done) make overview and package detials pages always contain unexported info, Use JS to sort and show.
* (cancelled) add an index section
* (done) after some time: remove the old ".gold-update" class in css file.
* (cancelled) click "type" keyword to unhide the source type definition.
  And show underlying type in a further click.
* (done) all references of an identifier
* (done) for builtin function
  * link panic/recover/... to their implementation positions.
* (canceleed, not needed any more) for a value
  * if its type is unexported, but its type has exported methods, list the methods under the value.
* (done) rename gold to golds ? 
* (done) generation mode option:
  * -moregc: set GCPercent 67%.
  * -nouses: don't generate id uses pages
  * -simplecode: simple code pages
* (done) for a method
  * if it is an interface method, show all concrete implementations, 
  * if it is a concrete methods show all its implemented interface methods (to view docs)
* (done) show identifier uses/references (open in new window)
* (done) cache all source code (not much memory consumed, but will get some convenience)
* (done) gen mode: no need to cache pages
* (cancelled) html escape some doc texts. use htmp.Escape 
* (done) show non-exporteds for main packages, show main func entry "m->" before source file
* (done) click interface method to show multiple concrete methods.
  Use the method implementation page instead.
* (done) sort types by popularity
* (done) implement registerNamedInterfaceMethodsForInvolvedTypeNames
* (done) move stat title out of translation. (translations should not contain html)
* (done) stat:
  * n interfaces, m structs, ... (on overview and package detail pages)
  * exported variables/constants by type kinds
  * parameters/results by type kinds
* (done) gen zh-Cn std docs
  * show golang.google.cn/pkg/xxx for zh-CN translation 
* (done) use as early as possible SDK to generate testdata.json.tar.gz
* (done) debug ast file not found, why so many goroutines panics.
* (done) add color to code
* (done) package dependency page
* (done) show "(generated)" for cgo generated file path in source code page
* (done) SDL package: starting types positions are still not accurate
  * also need use lineStartOffsets table ...
* (done) D> file path; M> 123 main package
* (done) It is really a problem that gcc is needed to show std package docs.
  Need mention: https://github.com/golang/go/wiki/WindowsBuild
  Or add "gold -cgo=false std"
  (Temporarily os.Setenv("CGO_ENABLED", "0") for "gold std")
* (done) click package name to overview page and use the package as target.
* (done) embedded field in code should be clickable, 
* (done) field sorting not correct: http://localhost:56789/pkg:k8s.io/api/core/v1#name-ConfigMap
* (doen) cgo ast.File and Position not match problem: maintain a local modified go/packages package?
  * also good to get modules info
  (Use FileSet.PositionFor instead)
* (done) collect asParams and asResults in the current module,
  then collect them in nearby packages.
  (registered functions for types, but for builtin types, only increase a number.)
* (done) bug: builtin page: type byte  = byte
* (done) pre.line-numbers span.anchor:before {...} tab width problems
* (done) html generatioon
* (done) link to go.dev/pkg/xxx (shortcut)
* (done) func (x, y int): len(params []ast.Field) == 1, len(params[0].Names) == 2
  ast.Struct.Fields is alike. Check the uses!
* (done) when finding selector shadowing, need to consider unexported names needing package import pathes, ...
* (done) write links for
  * alias denoting
  * exported methods / fields
  * as outputs / as inputs 
* (done) package details page: click an exported type, don't go to source page
* (done) Alias to a type in another package, asOutputList, type is not bold displayed now.
* (done) print final memory usage.
