**Gold** is a Go local docs server, Go docs generator, and a Go code reader.
It tries to extract as much information as possible from Go code to help gophers use and study Go packages.

* [Demo of the generated docs for standard packages](https://docs.go101.org/index.html)
  (please note that the demo site lacks of several features in the local server version).
* [FAQ](https://go101.org/article/tool-gold.html#faq).
* Please follow [@Go100and1](https://twitter.com/go100and1) to get the latest news of **Gold**.

### Installation

Run `go get -u go101.org/gold` to install (and update) **Gold**.

Notes:
* If the tool name `gold` conflicts with another tool with the same name you are using,
you can run `go get -u go101.org/gold/godoge` instead to install **Gold** as _**godoge**_.
* The `GO111MODULE` enviroment variable might need to be set as `on` to utilize the `GOPROXY` setting,
depending on your Go Toolchain version and the directory in which the installation command runs.

### Features

* Supports listing exported types not only by alphabet, but also by popularity, which is good to understanding some packages exporting many types.
* Rich type information collection:
  * Shows type implemention relations ([demo 1](https://docs.go101.org/std/pkg/go/ast.html#name-Node) and [demo 2](https://docs.go101.org/std/pkg/bytes.html#name-Buffer)).
  * Shows method implementation relations.
  * Shows promoted selectors, even on unexported embedded fields ([demo](https://docs.go101.org/std/pkg/archive/zip.html#name-File)).
  * Shows as-parameters-of and as-results-of function/method list (including interface methods).
* Smooth code view experiences (good for studying Go projects without opening IDEs):
  * Click a local identifier to highlight all the occurences of the identifier.
  * Click a reference to a non-local identifier to jump to the declaration of the non-local identifier, 
    and fell free to open pages in new browser windows as needed.
* Shows code statistics ([demo](https://docs.go101.org/std/statistics.html)).
* Supports generating static HTML docs pages, to avoid rebuilding the docs later.
  And this is good for package developers to host docs of their packages.
* All functionalities are implemented locally, no external websites are needed.
* JavaScript-off friendly. No tracing, no auto external websites visiting.

_(NOTE: This tool is still in its early experimental phase. More new features will be added from time to time in future versions.)_

### Limitations

Go Toolchain 1.13+ is needed to run and run **Gold** (and 1.14+ is needed to build).

This project uses the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) package to parse code. The `golang.org/x/tools/go/package` is great, but it also has a shortcoming: there are no ways to get module/package downloading/preparing progress.

All packages must compile okay to get their docs shown.

Only a code snapshot is analyzed. When code changes, a new analyzation is needed from scratch.

Testing packages are excluded currently.

Code examples in docs are not shown currently.

### Usage

Start the docs server:
* Run `gold .` or `gold` to show docs of the package in the current directory (and all its dependency packages).
* Run `gold ./...` to show docs of all packages under the current directory (and all their dependency packages).
* Run `gold std` to show docs of standard packages.

Each of the above commands will open a browser window automatically.
We can use the `-s` or `-silent` flags to turn off the behavior.

Generate static HTML docs pages (the `-dir` flag is optional in this mode, its default value is `.`):
* `gold -gen -dir=generated`
* `gold -gen -dir=generated ./...`
* `gold -gen -dir=generated std`

We can run `gold -dir=.` from the HTML docs generation directory to view the generated docs.

### Analyzation Cases

The following results are got on a machine with an AMD-2200G CPU (4 cores 4 threads) and sufficient memory.
Go Toolchain 1.14.3 is used in the analyzations.

Before running the `gold ./...` command, the `go build ./...` command is run to ensure that
all involved modules/packages are fetched to local machine and verify cgo tools (if needed) have been installed.

| Project  | Package Count | Analyzation Time | Final Used Memory | Notes |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| [go-sdl2](https://github.com/veandco/go-sdl2) v0.4.4 | 47 | 1.3s | 200M | _(need run `go mod init github.com/veandco/go-sdl2` before running **Gold**)_ |
| [bolt](https://github.com/boltdb/bolt) v1.3.1 | 51 | 1.6s | 140M | |
| [tview](https://github.com/rivo/tview) rev:823f280 | 102 | 2s | 200M | _(run `gold .` instead of `gold ./...`)_ |
| [gorilla/websocket](https://github.com/gorilla/websocket) v1.4.2 | 118 | 1.8s | 337M | |
| [gio](https://git.sr.ht/~eliasnaur/gio) rev:3314696 | 119 | 3.1s | 1G | |
| [nats-server](https://github.com/nats-io/nats-server) v2.1.7 | 136 | 2.3s | 400M | _(need run `go mod vendor` before running **Gold**)_ |
| [badger](https://github.com/dgraph-io/badger) v2.0.3 | 145 | 2.2s | 350M | |
| Gold v0.0.1 | 151 | 2.5s | 400M | _(run `gold .` instead of `gold ./...`)_ |
| [pion/webrtc](https://github.com/pion/webrtc) v2.2.9 | 189 | 2.1s | 400M | |
| [goleveldb](https://github.com/syndtr/goleveldb) v1.0.0 | 193 | 2.7s | 600M | |
| standard packages v1.14 | 199 | 2.6s | 400M | |
| [ebiten](https://github.com/hajimehoshi/ebiten) _v1.11.1_ | 214 | 2.1s | 472M | |
| [tailscale](https://github.com/tailscale/tailscale) _v0.98.0_ | 275 | 2.5s | 539M | |
| [etcd](https://github.com/etcd-io/etcd) _v3.4.7_ | 391 | 3.5s | 700M | _(need run `go mod vendor` before running **Gold**)_ |
| [go-ethereum](https://github.com/ethereum/go-ethereum) _v1.9.14_ | 459 | 5.5s | 1.3G | |
| [minio](https://github.com/minio/minio) _RELEASE.2020-05-16T01-33-21Z_ | 639 | 5.1s | 1.2G | |
| [terraform](https://github.com/hashicorp/terraform) _v0.12.25_ | 777 | 5.7s | 1.5G | |
| [consul](https://github.com/hashicorp/consul) _v1.7.3_ | 803 | 7.2s | 1.9G | |
| [vitess](https://github.com/vitessio/vitess) _v6.0.20-20200525_ | 905 | 7.1s | 1.7G | |
| [istio](https://github.com/istio/istio) _1.6.0_ | 1860 | 10.7s | 2.8G | |
| [kubernetes](https://github.com/kubernetes/kubernetes) _v1.18.2_ | 2821 | 16.3s | 4G | |

There are still some famous projects failing to build (with the `go build ./...` command, at May 27th, 2020), such as docker, gvisor and traefik, so **Gold** is unable to build docs for them.

There are also some projects not using go modules, such as hashicorp/nomad and openshift/origin, and GOPROXY doesn't take effect for the `go mod init` command (as of Go Toolchain 1.14), so I couldn't build docs for these projects on my machine (this is my network problem, it might work on your machine).
