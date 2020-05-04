gold - Go local docs server.

Sell points:
* show implementions
* show promoted selectors, even on unexported embedded fields.
* rich code view experience
* JavaScript-off friendly

The HTML generation feature is for distributed docs.

It is really a problem that gcc is needed to show std package docs.
  Need mention: https://github.com/golang/go/wiki/WindowsBuild

NoJS friendly, though richer experiences when JS is enabled.

可以做成一个类似于SourceGraph的产品
+ plus code review tool.
+ other dev tools.

* package page
  go101.org/foo.com/bar
  go101.org/std/pkg
* module page
  go101.org/foo.com/bar/
* go101.org website pages
  go101.org/articles
  go101.org/static

https://github.com/golang/go/issues/34527

Package details:
* overall
  * click to show/hide docs directly udnerlying each resource
  * sort by name / position / similarity
  * 当一个package发生变化时，依赖于它的packages将的页面将出现一个reanalyze链接。
  * cache to本地文件中。每个package一个页面，(比较复杂，暂时不做)
  * 显示当前的build tags，展示其它常用tag 组合链接
  * one module, one background color
  * 当一个方法实现了某些接口时，link to those interface method docs.
  * statistics info: total N underlying interfaces, everage 2.31 methods each. Total N type delcaratiuons, M aliases.
  * if full source scan (for reference finding) is time consuming, show a progress bar on any page.
* type Alias = T
  (More aliases of T: xxx, yyy, zzz.)
  If T is a defined type, click it to jump to T's definition.
  If T is a non-defined struct/interface type, click it unhide the mehtod/field set of T
* type T X
  * comparable or not
  * Underlying type: Y, which is also the underlying type of these types: M (mAlias1, mAlias2), N, ...
  * Alias: ...
  * N exported Methods (M are promoted), K unepxorted methods. 
    func (T) M1()
    (promoted) func (T.Foo) M2()
    func (*T.Foo.Bar) M3()
    func (T.<?>.Foo1) M4()
    func (*T.<?>.Foo1.Bar1) M5()
  * used as parameter types in ... (inlcuding methods)
  * used as result types in ...  (inlcuding methods)
  * used as fields in ... (embedded or not)
  * implements ... (highlight std interfaces)
    * each implements followed a 小灯泡。点击之后实现的对应方法高亮并前缀小灯泡
  * interface
    expand embedded interfaces
    implemented by: ... in the same package. ... in sub/sibling/parent packages and N in other packages.
  * struct
    N fields
    Embedded.PromotedField (通过展示选择器全路径来显示promoted字段和方法。)
    (and N non-exported fields, not bad to list them)
    {
    	Embedded // (if this is an alias, show: == theDenotingType)
    	Foo int // == Embedded.Foo)
    	Bar bool // == Embedded.Foo.Bar)
    	
    	// Promoted from unexporteds
    	Foo1 int  (== <?>.Foo1)
    	Bar1 bool (== <?>.Foo1.Bar1)
    }
    filter: hide promoteds
* func F (...) (...)
  also list Type.Methods as functions
  * by files (and positions in files)
* var x T (initial value is ...)
  var x = ... (of type T)
* const X T = ...
  const X = ... (default type is T)
* Non-exported (very needed, for we need know methods of some exported variables of non-exported types)
  * types
    Sometimes, we need 
    	the exported methods/fields of an exported variable (or return results) of non-exported type.
    	the methods and fields of the exported alias of unexported types.
    	the promoted methods and fields of unexported embedded types/aliases.
  * funcs
  * vars
  * consts


Package list:
* if two listed adjicent package paths has prefix relation, then grey the prefx in the second part


* show all OS specified packages
   https://stackoverflow.com/questions/7044944/jquery-javascript-to-detect-os-without-a-plugin
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
* if last token in import path and package name are different, mention it


There are more cases of exported methods/fields are not listed by doc cmd and godoc webpages.
* the promoted methods and fields of unexported fields
* the exported methods and fields of exported variables of unexported types.
* the exported methods and fields of results of unexported types of exported functions (or of fields of visible structs).
  visible structs mean the ones returned by exported functions or exported struct types.
* the exported methods and fields of the exported alias of unexported types


FAQ:
* why gold is recommended to run locally? Is gold is really capable for running on internet?
  Not exactly, you can provide a public website for serving the docs, but it requires huge storage to cache the analyse result.
* what are the ... to run gold?
  * if the code need cgo, some c/c++ compilers might be needed.
  * some code base might need large memory capacity.
* Future p2p sharing analysis result.
  11 authorities, if at least 6 of them agree a result is ok, then local thinks it is ok.
  Analyst result are validated by attach some signs signed by auth certificates.

Ideas:
  * doc:
    * show struct paddings/sizes
    * hints: will an argument be modified in function body
    * tabs: funcs, types, consts, vars, imports
    * link each item to the official go doc page
    * show all OS specified packages
    * show by alpha order / by file order / by relation order
    * if last token in import path and package name are different, mention it.
  * translate wasm to go code

Details:
* cache html pages. 
  Program current format version n is compatible [n - 10, n) versions.
* when CPU usage < 25%, run one goroutine to do uncached work.

ToDo: when using custom methodset computing implemenation,
      need to consider (a, b Type) cases in calcualate method signature.

module@version
m::module/path@version/f::source/file/path
m::module/path@version/p::package/import/path

press ~ or home to home page

package details page hidden elements:
* click "package" to go home
* click package name to switch theme

Code view:
* click a declared identifier (with 衬底虚线), show a new popup (or open new window)
  * click on declared functions, show references
  * click on declared methods, show references and matched interface method specifications
  * 

Write a static checker
* ensure a finalizer only use the values references by its argument.
* ensure exported package-level variables are not modified by outers.
  Check some famous projects, if there is not any such cases, then
  create a proposal to let compilers view exported values as finals.

Run code page
* websocket: monitor page leave and shutdown unfinished Go processes.

github.com/liulaomo => github.com/tlllm/gold

可以考虑将std库上网。（和初始gold一块announce）
* 或者某些特定版本的k8s
* or other famous libraries
* links to godoc.org/go.dev
* go101.org/pkg:xxx
  go101.org/src:xxx
  go101.org/xxx:xxx
* go101.org/mod:std
  - url: /pkg:.*
    static_dir: web/generated/pkg
    secure: always
    redirect_http_response_code: 301
    expiration: "7d"
* go101.org home page
  * std packages link
  * polls: how large is your GOPATH/pkg folder
  * common Go websites
* std-doc.zip alike for libarary maintainer to serve docs and for users to download
  gold -action=generate -output=./files ./...
  * inline css and js files in every page.


gold
* local go documentation
* local go directory (golf for local go forum)
* local go develop
* gold .       the package at the current directory and its dependencies
  gold ./...   all the packages under the current directory and their dependencies
  gold all
  gold std     all standard packages

golden
* the development/experiment version of gold

rss: river stream server

type
* as parameters of
* as results of
* exported values

todo:
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


* value: find other values with the same type
* type: find convertible/assignable types
* type: as fields of
  for interface: subset of
  for non-interface: embedded by
* unnamed type: find all occurences, as inputs/outputs/fields of ...
* interface embed overlapping interfaces, a mehtod might have several doc sources
* method: sjhow whether or not is promoted
* when click function/variable/constant declaration, show reference list.
* for all exported values, filter: func | var | const | group by type | ...
* cgo on mac: how to?
* crash: /src:/home/d630/projects@asiainfo/aif/dfplatform/servicebroker/server/apis.go
  file sizes not match. 1761 : 2163
  (not allways producable)
* support modules
  * package.Load(needLess) to get all packages of std.
    treat std as a module
* all references of an identifier/unnamed-type: frame pages
* click anInterfaceValue.method, show select list to a concreate method
* ./tests/g last comment is not greened in code.
* support multi GOOS pages
* also should embedding chain for method list
* exported values: jump from code to details page, not highlight target.
* shortcuts
  ~ and backspace to back
  -: expand to next level
  +: collapse to next level
* asParams/asResults lists exclude the methods of unexported types.
* align asTypeOf items: sort by value | sort by position | sort by name
* better cgo support, parse c code.
  * use original Go files and parse c files
  * https://godoc.org/github.com/cznic/cc#example-Statement
    https://pkg.go.dev/github.com/cznic/cc/v2?tab=doc
  * https://github.com/xlab/c-for-go
* .s file syntax support
* handle panics in http handlers
* list non-exported elements in main packages
* some functions can be used as (implicitly converted to) http.HandleFunc values, alike
* special handling for the buitlin page, 
  by replacing make(t T, sizes ...int) T to make(T slice|map|channel, sizes ...int) T
* example code run, run testing/banchmark
* code search
* handle the case of when a method sourced from several different original ones by embedding.
  Their docs might be a little different.
* Alias to a type in another package, asOutputList, type is not bold displayed.
* more type stats:
  embedding in n types
* "go/doc": doc.Examples(...)


* when finding selector shadowing, need to consider unexported names needing package import pathes, ...
* func (x, y int): len(params []ast.Field) == 1, len(params[0].Names) == 2
  ast.Struct.Fields is alike. Check the uses!
* shortcuts:
  * ~, Backspace: back
  * HOME: to overview page
  * P: from code page to package detail page
  * 
* stat: n interfaces, m structs, ... (on overview and package detail pages)
  * as parameters/results of N functions
* add a RoughLastestSubmitTime, if the time is earlier than time.Now for one months, notify ...
  "two weeks not update. Update now?: to prepare for future golf promotion etc.
* adjust pkg details page, even simpler, ...
* rearrange code ...
* tests
* js loading in pages
* css style

