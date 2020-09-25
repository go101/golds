**Gold** is a Go local docs server, a Go docs generator, and a Go code reader.
It tries to extract as much information as possible from Go code to help gophers understand, study and use Go packages.

* [Demo of the generated docs for standard packages](https://docs.go101.org/index.html)
  (please note that the demo site lacks of several features in the local server version).
* [FAQ](https://go101.org/article/tool-gold.html#faq).
* Please follow [@Go100and1](https://twitter.com/go100and1) to get the latest news of **Gold**.

### Installation

Run `go get -u go101.org/gold` to install (and update) **Gold**. 
_(The `GO111MODULE` enviroment variable might need to be set as `on` temporarily to utilize the `GOPROXY` setting,
depending on your Go Toolchain version and the directory in which the installation command runs.)_

Note, if the tool program name `gold` conflicts with another tool with the same name you are using,
you can run any of the following commands to install **Gold** as a program with a different name:
* **Go** **do**cs **ge**nerator  
  `go get -u go101.org/gold/godoge`
* **Go** **co**de **re**ader  
  `go get -u go101.org/gold/gocore`
* **Go** **l**ocal **d**ocs **s**erver  
  `go get -u go101.org/gold/golds`

If for any reason the `go get` way doesn't work, you may also clone this project firstly, then run the `go install` command in the respective program folders to install **Gold** as `gold`, `godoge`, `gocore`, or `golds`.

### Features

* Supports listing exported types not only by alphabet, but also by popularity, which is good to
  understanding some packages exporting many types.
* Supports listing unexported types, which is good to read some packages.
* Rich package-level type/value information collection:
  * Shows type implemention relations ([demo 1](https://docs.go101.org/std/pkg/go/ast.html#name-Node) and [demo 2](https://docs.go101.org/std/pkg/bytes.html#name-Buffer)).
  * Shows method implementation relations ([demo](https://docs.go101.org/std/imp/io.Reader.html#name-Read)).
  * Shows promoted selectors, even on unexported embedded fields ([demo](https://docs.go101.org/std/pkg/archive/zip.html#name-File)).
  * Shows as-parameters-of and as-results-of function/method list (including interface methods).
  * Shows the package-level value lists of a package-level type.
  * Shows uses of package-level declared types/constants/variables/functions (by clicking the `type`/`const`/`var`/`func` keywords).
* Smooth code view experiences (good for studying Go projects without opening IDEs):
  * Click a local identifier to highlight all the occurences of the identifier.
  * Click a use of a non-local identifier to jump to the declaration of the non-local identifier.
  * Click the name of a field or a method in its declaration to show its uses (only for package-level named struct types now).
  * Click the name of a method specified in an interface type declaration to show the methods implementing it (only for package-level named interface types now)..
* Shows code statistics ([demo](https://docs.go101.org/std/statistics.html)).
* Supports generating static HTML docs pages, to avoid rebuilding the docs later.
  This is good for package developers to host docs of their own packages.
  (The docs of standard packages are generated within about 7 seconds, and the docs of the kubernetes project packages are generated within about one minute.)
* All functionalities are implemented locally, no external websites are needed.
* Just fell free to open any number of pages in new browser windows as needed.
* JavaScript-off friendly. No tracing, no auto external websites visiting.

_(NOTE: This tool is still in its early experimental phase. More new features will be added from time to time in future versions.)_

### Limitations

Go Toolchain 1.13+ is needed to run **Gold** (and 1.14+ is needed to build **Gold**).

This project uses the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) package to parse code.
The `golang.org/x/tools/go/package` package is great, but it also has a shortcoming: there are no ways to get module/package downloading/preparing progress.

All packages must compile okay to get their docs shown.

Only a code snapshot is analyzed. When code changes, a new analyzation is needed from scratch.

Testing packages are excluded currently.

Code examples in docs are not shown currently.

### Usage

Start the docs server:
* Run `gold .` to show docs of the package in the current directory (and all its dependency packages).
* Run `gold ./...` to show docs of all packages under the current directory (and all their dependency packages).
* Run `gold std` to show docs of standard packages.
* Run `gold aPackage` to show docs of the specified package (and all its dependency packages).

Each of the above commands will open a browser window automatically.
We can use the `-s` or `-silent` options to turn off the behavior.

Generate static HTML docs pages (the `-dir` option is optional in this mode, its default value is `.`):
* `gold -gen -dir=generated .`
* `gold -gen -dir=generated ./...`
* `gold -gen -dir=generated std`

The above commands will generated the full docs of specified packages.
The following options are available to generate compact docs:
* `-nouses`: don't generate identifier-uses pages (identifier-uses pages will occupy about 9/10 of the total page count and 2/3 of the full docs size).
* `-plainsrc`: generate simpler source code pages (no highlighting and no code navigations to reduce the total page size by 1/6 of the full docs size).

The size of the docs generated by `gold -gen -nouses -plainsrc ./...` is about 1/6 of `gold -gen ./...`.

The `-emphasize-wdpkgs` option is used to list the packages within the working directory before other packages in the first page
(for HTML docs generation mode only).

We can run `gold -dir=.` (or simply `gold`) from the HTML docs generation directory to view the generated docs in browser.
(**Gold** also means __Go local directory server__. The `-s` or `-silent` options also work in this mode.)

The `gold` command recognizes the `GOOS` and `GOARCH` environment variables.

### Analyzation Cases

The following results are got on a machine with an AMD-2200G CPU (4 cores 4 threads) and sufficient memory.
Go Toolchain 1.14.3 is used in the analyzations.

Before running the `gold ./...` command, the `go build ./...` command is run to ensure that
all involved modules/packages are fetched to local machine and verify cgo tools (if needed) have been installed.

| Project  | Package Count | Analyzation Time | Final Used Memory | Notes |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| [imgui-go](https://github.com/inkyblackness/imgui-go) _v2.5.0_ | 35 | 1.2s | 125M | |
| [gotk3](https://github.com/gotk3/gotk3) _rev:030ba00_ | 40 | 3s | 305M | |
| [go-sdl2](https://github.com/veandco/go-sdl2) _v0.4.4_ | 47 | 1.3s | 200M | _(need run `go mod init github.com/veandco/go-sdl2` before running **Gold**)_ |
| [bolt](https://github.com/boltdb/bolt) _v1.3.1_ | 51 | 1.6s | 140M | |
| [nucular](https://github.com/aarzilli/nucular) _rev:b1fe9b2_ | 97 | 2s | 250M | |
| [tview](https://github.com/rivo/tview) _rev:823f280_ | 102 | 2s | 200M | _(run `gold .` instead of `gold ./...`)_ |
| [gorilla/websocket](https://github.com/gorilla/websocket) _v1.4.2_ | 118 | 1.8s | 337M | |
| [gio](https://git.sr.ht/~eliasnaur/gio) _rev:3314696_ | 119 | 3.1s | 1G | |
| [nats-server](https://github.com/nats-io/nats-server) _v2.1.7_ | 136 | 2.3s | 400M | _(need run `go mod vendor` before running **Gold**)_ |
| [badger](https://github.com/dgraph-io/badger) _v2.0.3_ | 145 | 2.2s | 350M | |
| Gold v0.0.1 | 151 | 2.5s | 400M | _(run `gold .` instead of `gold ./...`)_ |
| [pion/webrtc](https://github.com/pion/webrtc) _v2.2.9_ | 189 | 2.1s | 400M | |
| [goleveldb](https://github.com/syndtr/goleveldb) _v1.0.0_ | 193 | 2.7s | 600M | |
| standard packages v1.14 | 199 | 2.6s | 400M | |
| [Pebble](https://github.com/cockroachdb/pebble) _rev:284ba06_ | 200 | 2.2s | 500M | |
| [ebiten](https://github.com/hajimehoshi/ebiten) _v1.11.1_ | 214 | 2.1s | 472M | |
| [tailscale](https://github.com/tailscale/tailscale) _v0.98.0_ | 275 | 2.5s | 539M | |
| [etcd](https://github.com/etcd-io/etcd) _v3.4.7_ | 391 | 3.5s | 700M | _(need run `go mod vendor` before running **Gold**)_ |
| [go-ethereum](https://github.com/ethereum/go-ethereum) _v1.9.14_ | 459 | 5.5s | 1.3G | |
| [minio](https://github.com/minio/minio) _RELEASE.2020-05-16T01-33-21Z_ | 639 | 5.1s | 1.2G | |
| [terraform](https://github.com/hashicorp/terraform) _v0.12.25_ | 777 | 5.7s | 1.5G | |
| [consul](https://github.com/hashicorp/consul) _v1.7.3_ | 803 | 7.2s | 1.9G | |
| [vitess](https://github.com/vitessio/vitess) _v6.0.20-20200525_ | 905 | 7.1s | 1.7G | |
| [nomad](https://github.com/hashicorp/nomad) _v0.12.4_ | 897 | 7.5s | 2.1G | |
| [Traefik](https://github.com/traefik/traefik) _v2.3.0_ | 1199 | 8.9s | 2G | _(need [generate bindata](https://doc.traefik.io/traefik/contributing/building-testing/#build-traefik) before running *Gold*)_ |
| [istio](https://github.com/istio/istio) _1.6.0_ | 1860 | 10.7s | 2.8G | |
| [openshift/origin](https://github.com/openshift/origin) _rev:5022f83_ | 2640 | 16.1s | 4G | |
| [kubernetes](https://github.com/kubernetes/kubernetes) _v1.18.2_ | 2821 | 16.3s | 4G | |



