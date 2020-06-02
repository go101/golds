

### Soon to do

* implement registerNamedInterfaceMethodsForInvolvedTypeNames
* enhance tests
  * test by ast comments
* css style
* add comments
* js:
  * shortcuts:
    * ~, Backspace: back
    * HOME: to overview page
    * P: from code page to package detail page
    * -: collapse value/type docs
    * +: expand value/type docs
  * filter values (var | const | func)
    filter types (interfaces)
    fitler packages (main | std)
    sort packages by importedBys (by most imports is non-sense)
    sort types (by method count or popularity, used as parameters/results)
  * search on pkg details pages, and filter packages on overview page
  * click "package"/overview to switch theme/language

### More to do

* For std pacakges: show which version of Go introduced a particular function/type, etc.
  * or for any modules

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
* show/run examples/tests/banchmarks
  * "go/doc": doc.Examples(...)
  * use go/doc to retrieve package docs
  * websocket: monitor page leave and shutdown unfinished Go processes.
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

* packakge list
  * show by alpha order / by importedBys / by dependency level
  * if last token in import path and package name are different, mention it
  * list packages by one module, one background color
* for all exported values,
  * filter: func | var | const | group by type | ...
  * find other values with the same type
  * function: hints: will an argument be modified in function body
* stat:
  * n interfaces, m structs, ... (on overview and package detail pages)
  * top N lists
  * embedding in n types
  * top N used identifers
* package details
  * add parent and children packages
* imports
  * add links for import declarations
* docs for unepxorted types/vars
  * unnamed type: find all occurences
  * (done) the promoted methods and fields of unexported fields
  * the exported methods and fields of exported variables of unexported types.
  * the exported methods and fields of results of unexported types of exported functions (or of fields of visible structs).
  visible structs mean the ones returned by exported functions or exported struct types.
  * the exported methods and fields of the exported alias of unexported types
* for a type
  * click "type" keyword to unhide the source type definition.
    And show underlying type in a further click.
  * show the types with the same underlying type.
  * show comparable/embeddable or not. Fill TypeInfo.attributes.
  * all alias list
  * values which can be converted to (some functions can be used as (implicitly converted to) http.HandleFunc values, alike)    
  * asParams/asResults lists exclude the methods of unexported types now.
  * asTypeOf items: sort by value | sort by position | sort by name
  * method: show whether or not is promoted
  * as fields of types list (and embedded-in list)
  * for interface: subset of list
  * for non-interface: embedded by list
  * convertible/assignable types
  * show struct paddings/sizes
  * sort types by popularity
  * filter by kind
  * as-type / as-params / as-results lists detail:
    * merge method with the same signature
  * as-type: also combine values of []T, chan T, etc. (now only combine values of T and *T)
  * implementBy and implement lists should include exported aliases of unnmaed types.
  * it is important to find a way to list implemented unexported types, which is good for code reading.
  * list variables of function types in asParams and asResults lists.
* for a value
  * if its type is unexported, but its type has exported methods, list the methods under the value.
* for a method
  * if it is an interface method, show all concrete implementations, need JS
  * if it is a concrete methods, show all its implemented interface methods (to view docs), need JS
  * an interface method might also has multiple docs, for interface embed overlapping interfaces
* for builtin function
  * link panic/recover/... to their implementation positions.
* all references of an identifier: in frame pages
  * for types, show unexported methods/fields/...


### Done

* (doen) gen zh-Cn std docs
  * show golang.google.cn/pkg/xxx for zh-CN translation 
* (done) use as early as possible SDK to generate testdata.json.tar.gz
* (done) debug ast file not found, why so many goroutines panics.
* (done) add color to code
* (done) package dependency page
* (done) put "(D)" in front of each source file which containing package docs.
* (done) show "(generated)" for cgo generated file path in source code page
* (done) SDL package: starting types positions are still not accurate
  * also need use lineStartOffsets table ...
* (done) D> file path; M> 123 main pacakge
* (done) It is really a problem that gcc is needed to show std package docs.
  Need mention: https://github.com/golang/go/wiki/WindowsBuild
  Or add "gold -cgo=false std"
  (Temporarily os.Setenv("CGO_ENABLED", "0") for "gold std")
* (done) clickable for M-> and D->
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
* (done) // 调换次序
	log.Println("[analyze packages 4...]")
	methedCache := d.analyzePackages_FindImplementations()
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
