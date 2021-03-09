

### Soon to do


* also grey the same parts in asInputsOf/.... lists

* move most readme content to go101.org.
  * keep case table, simple install, simple feature overview, simple usage (golds std, golds ./...)

* code page: each funciton enclosed in a span so that local id hightlighting needs less time.
* reduce code page size
  * some buildIdentifier -> buildLink
  * ...

* css style
  * https://github.com/go101/golds/issues/13
* enhance tests
  * test by ast comments
* add more comments, and clear some

* show statistics floating on the right of the overview page.

* show/run examples/tests/banchmarks 
  (Tests==true, cause reflect.EmbedWithUnexpMeth not found in analyzePackage_ConfirmTypeSources/registerDirectFields now)
  * run source code, run main package
  * Open a new page to avoid using JavaScript?
  * "go/doc": doc.Examples(...)importance
  * websocket: monitor page leave and shutdown unfinished Go processes.


* add "d➜" in overview page: hover on a package: show its brief intro.
* more hovers:
  * in code, show tooltip as the full selector path for shortened selectors.

* now, there is not a way to view the uses of embedded fields (control+click?)

* implicit
  * switch expr := srcNode.(type) { // this expr might need to be enclosed in mutilple labels
    case T1: _ = expr
    case T2: _ = expr
    }

* -format=[html|json|txt|md]

* type alias and same-underlyings list

* hotkey
  * t/f/v/c to expend/collapse resources
  * HOME: to overview page.
    * add a settings page on overview page: switch theme/language etc.
  * P: from code page to package detail page
  * e: toggle show (fields/methods) promoteds | show full selector path | show explicits only

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

* modify the cache system to only cache most visited and recent ones

* method docs 
  * how to handle duplicated methods caused by interface embedding interfaces.
    Their docs might be different.

* rate limit http requests. 1000requests/3600seconds

* server state:
  * highlight id 0-n
  * searching uses for id goroutine 0-n
  (forget what these means)

* module page. Containing Module: xxxx/xxxx
  * sort by requiredBys

* pkg details page: show values by file/position order (only for javascript on)

* overview page: show std pkgages only
  * need maintain a seperated depHeight/depDepth for std module internally.

* search (non-semantic search, pure word searching)
  * ref: https://github.com/g-harel/gothrough

* gen mode: merge docs for several (GOOS, GOARCH) compositions. At least for std.

* search ids on pkg details/overview pages

* Rewrite some implemenrations
  * global.pacakgeList, each pkg has a unique id (int32)
  * global.functionProtoypes, each has a unique id (int32)
  * global.identifierList, each has unique id
  * global.selectorNameIds {pkgId, identId int32}
  * global.methodPrototypes {selNameId, funcProtoId}, each correspods a unique id (int32(
  * global.method2typesTable map[methodProtoId][]*type. All the []*type share a common big []*type slice.
    The length of the big slice is sum(type[i].methodCount)



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
    * use original Go files and parse c files
    * https://godoc.org/github.com/cznic/cc#example-Statement
      https://pkg.go.dev/github.com/cznic/cc/v2?tab=doc
    * https://github.com/xlab/c-for-go
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
  * unnamed type: find all occurences (use fake type ids)
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
* (done) D> file path; M> 123 main pacakge
* (done) It is really a problem that gcc is needed to show std package docs.
  Need mention: https://github.com/golang/go/wiki/WindowsBuild
  Or add "gold -cgo=false std"
  (Temporarily os.Setenv("CGO_ENABLED", "0") for "gold std")
* (done) click package name to overview page and use the pacakge as target.
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
