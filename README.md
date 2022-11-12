[![go-recipes](https://raw.githubusercontent.com/nikolaydubina/go-recipes/main/badge.svg?raw=true)](https://github.com/nikolaydubina/go-recipes)

**Golds** is a **Go** **l**ocal **d**ocs **s**erver, a Go docs generator, and a Go code reader.
It tries to extract as much information as possible from Go code to help gophers understand, study and use Go packages.

* [Demo of the generated docs for standard packages](https://docs.go101.org/index.html)
  (generated with `golds -gen -nouses -only-list-exporteds std`).
* Please follow [@Go100and1](https://twitter.com/go100and1) to get the latest news of **Golds**
  (and Go details/facts/tips/etc.).

### Installation & Usages

The module path of this project is `go101.org/golds` and the default Go program name is `golds`.
For more information about installation and usages,
please visit [this page on Go101.org](https://go101.org/apps-and-libs/golds.html).

### Main Features

* JavaScript-off friendly, though the experience is a little richer if JavaScript is on.
  No tracing, no auto external websites visiting.
* Lists types not only by alphabet, but also by popularity (only available when JavaScript is on),
  which is good to understanding some packages with many types declared.
* Supports listing unexported types and values, which is helpful to understand some packages.
* Treats `main` packages as same as library packages.
* Rich package-level type/value information collection:
  * Shows type implementation relations ([demo 1](https://docs.go101.org/std/pkg/go/ast.html#name-Node) and [demo 2](https://docs.go101.org/std/pkg/bytes.html#name-Buffer)).
  * Shows method implementation relations ([demo](https://docs.go101.org/std/imp/io.Reader.html#name-Read)).
  * Shows promoted selectors, even on unexported embedded fields ([demo](https://docs.go101.org/std/pkg/archive/zip.html#name-File)).
  * Shows as-parameters-of and as-results-of function/method lists (including interface methods).
  * Shows the package-level value list of a package-level type.
  * Shows uses of package-level declared types/constants/variables/functions (by clicking the `type`/`const`/`var`/`func` keywords).
  * Press `p` (package), `e` (examples), `t` (types), `f` (functions), `v` (variables), `c` (constants), and `a` (all) to toggle doc expand/collapse statuses (only available when JavaScript is on).
* Smooth code view experiences (good for studying Go projects without opening IDEs):
  * Click a local identifier to highlight all the occurrences of the identifier.
  * Click a use of a non-local identifier to jump to the declaration of the non-local identifier.
  * Click the name of a field or a method in its declaration to show its uses (only for package-level named struct types now).
    If the name represents a method, in the uses page, click the _(method)_ text to show which interface methods the method implements.
  * Click the name of a method specified in an interface type declaration to show the methods implementing it (only for package-level named interface types now).
    In the method-implementation page, click each the name of an interface method to show the uses of the interface method.
* Shows code statistics ([demo](https://docs.go101.org/std/statistics.html)).
  * Click some of the SVG pictures to show corresponding toppest list.
* Supports generating static HTML docs pages, to avoid rebuilding the docs later.
  This is good for package developers to host docs of their own packages.
  (The docs of standard packages are generated within about 10 seconds, and the docs of the kubernetes project packages are generated within about 100 seconds.)
* All functionalities are implemented locally, no external websites are needed (except explicitly required by you).
* Just fell free to open any number of pages in new browser windows as needed.

_(NOTE: This tool is still in its early experimental phase. More new features will be added from time to time in future versions.)_

### Limitations

Go Toolchain 1.13+ is needed to build and run **Golds**.

This project uses the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) package to parse code.
The `golang.org/x/tools/go/package` package is great, but it also has a shortcoming: there are no ways to get module/package downloading/preparing progress.

All packages must compile okay to get their docs shown.

Only a code snapshot is analyzed. When code changes, a new analyzation is needed from scratch.

Testing packages are excluded currently.

### Analyzation Cases

The following results are got on a machine with an AMD-2200G CPU (4 cores 4 threads) and sufficient memory.
Go Toolchain 1.14.3 is used in the analyzations.

Before running the `golds ./...` command, the `go build ./...` command is run to ensure that
all involved modules/packages are fetched to local machine and verify cgo tools (if needed) have been installed.

| Project  | Package Count | Analyzation Time | Final Used Memory | Notes |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| [imgui-go](https://github.com/inkyblackness/imgui-go) _v2.5.0_ | 35 | 1.2s | 125M | |
| [gotk3](https://github.com/gotk3/gotk3) _rev:030ba00_ | 40 | 3s | 305M | |
| [go-sdl2](https://github.com/veandco/go-sdl2) _v0.4.4_ | 47 | 1.3s | 200M | _(need run `go mod init github.com/veandco/go-sdl2` before running **Golds**)_ |
| [bolt](https://github.com/boltdb/bolt) _v1.3.1_ | 51 | 1.6s | 140M | |
| [nucular](https://github.com/aarzilli/nucular) _rev:b1fe9b2_ | 97 | 2s | 250M | |
| [tview](https://github.com/rivo/tview) _rev:823f280_ | 102 | 2s | 200M | _(run `golds .` instead of `golds ./...`)_ |
| [gorilla/websocket](https://github.com/gorilla/websocket) _v1.4.2_ | 118 | 1.8s | 337M | |
| [gio](https://git.sr.ht/~eliasnaur/gio) _rev:3314696_ | 119 | 3.1s | 1G | |
| [nats-server](https://github.com/nats-io/nats-server) _v2.1.7_ | 136 | 2.3s | 400M | _(need run `go mod vendor` before running **Golds**)_ |
| [badger](https://github.com/dgraph-io/badger) _v2.0.3_ | 145 | 2.2s | 350M | |
| Golds v0.0.1 | 151 | 2.5s | 400M | _(run `golds .` instead of `golds ./...`)_ |
| [pion/webrtc](https://github.com/pion/webrtc) _v2.2.9_ | 189 | 2.1s | 400M | |
| [goleveldb](https://github.com/syndtr/goleveldb) _v1.0.0_ | 193 | 2.7s | 600M | |
| standard packages v1.14 | 199 | 2.6s | 400M | |
| [Pebble](https://github.com/cockroachdb/pebble) _rev:284ba06_ | 200 | 2.2s | 500M | |
| [ebiten](https://github.com/hajimehoshi/ebiten) _v1.11.1_ | 214 | 2.1s | 472M | |
| [dominikh/go-tools](https://github.com/dominikh/go-tools) _rev:919f4c9_ | 235 | 4.5s | 465M | |
| [tailscale](https://github.com/tailscale/tailscale) _v0.98.0_ | 275 | 2.5s | 539M | |
| [etcd](https://github.com/etcd-io/etcd) _v3.4.7_ | 391 | 3.5s | 700M | _(need run `go mod vendor` before running **Golds**)_ |
| [go-ethereum](https://github.com/ethereum/go-ethereum) _v1.9.14_ | 459 | 5.5s | 1.3G | |
| [minio](https://github.com/minio/minio) _RELEASE.2020-05-16T01-33-21Z_ | 639 | 5.1s | 1.2G | |
| [terraform](https://github.com/hashicorp/terraform) _v0.12.25_ | 777 | 5.7s | 1.5G | |
| [consul](https://github.com/hashicorp/consul) _v1.7.3_ | 803 | 7.2s | 1.9G | |
| [vitess](https://github.com/vitessio/vitess) _v6.0.20-20200525_ | 905 | 7.1s | 1.7G | |
| [nomad](https://github.com/hashicorp/nomad) _v0.12.4_ | 897 | 7.5s | 2.1G | |
| [Traefik](https://github.com/traefik/traefik) _v2.3.0_ | 1199 | 8.9s | 2G | _(need [generate bindata](https://doc.traefik.io/traefik/contributing/building-testing/#build-traefik) before running **Golds**)_ |
| [istio](https://github.com/istio/istio) _1.6.0_ | 1860 | 10.7s | 2.8G | |
| [openshift/origin](https://github.com/openshift/origin) _rev:5022f83_ | 2640 | 16.1s | 4G | |
| [kubernetes](https://github.com/kubernetes/kubernetes) _v1.18.2_ | 2821 | 16.3s | 4G | |



