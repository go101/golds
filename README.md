**Gold** is an experimental alternative Go local docs server.

### Features

* Show type implemention relations.
* Show promoted selectors, even on unexported embedded fields.
* Rich code view experience (good for study Go projects without opening IDEs).
* JavaScript-off friendly.
* Support generating static HTML pages (good for packages which don't satisfy go.dev license requirements).

This tool is still in its early phase. More features will be supported from time to time.

### Installation

Run `go get -u go101.org/gold` to install (and update) **Gold**.

### Usage

Start the docs server:
* Run `gold .` or `gold` to show docs of the package in the current directory (and all its dependency packages).
* Run `gold ./...` to show docs of all the package under the current directory (and all their dependency packages).
* Run `gold std` to show docs of standard pacakges.

The above commands will open browser automatically.
We can use the `-s` or `-silent` flags to turn off the behavior.

Generate static HTML docs pages:
* `gold -gen -dir=./generated`
* `gold -gen -dir=./generated ./...`
* `gold -gen -dir=./generated std`

We can run `gold -dir=.` or `gold -dir` from the generation directory to view the generated docs.

### Limitations

This project uses the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) package to parse code. The `golang.org/x/tools/go/package` is great, but it also has a shortcoming: there are no ways to get module/package downloading/preparing progress.

All packages must compile okay to get their docs shown.

Currently, testing packages are not analyzed.

### Analyzation Cases

The following results are got on a machine with an AMD-2200G CPU (4 cores 4 threads) and sufficient memory.

It is assumed that all involved modules/packages are fetched to local machine before running the `gold ./...` command.

Go SDK 1.14.3 is used in the analyzations.

| Project  | Package Count | Analyzation Time | Final Memory Usage | Notes |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| [go-sdl2](https://github.com/veandco/go-sdl2) v0.4.4 | 47 | 1.3s | 200M | _(need run `go mod init github.com/veandco/go-sdl2` before running **Gold**)_ |
| [bolt](https://github.com/boltdb/bolt) v1.3.1 | 51 | 1.6s | 140M |  |
| [nats-server](https://github.com/nats-io/nats-server) v2.1.7 | 136 | 2.3s | 400M | _(need run `go mod vendor` before running **Gold**)_ |
| [badger](https://github.com/dgraph-io/badger) v2.0.3 | 145 | 2.2s | 350M | |
| Gold v0.0.1 | 149 | 2.2s | 400M |  |
| [pion/webrtc](https://github.com/pion/webrtc) v2.2.9 | 189 | 2.1s | 400M | |
| [goleveldb](https://github.com/syndtr/goleveldb) v1.0.0 | 193 | 2.7s | 600M | |
| standard packages v1.15 | 199 | 2.6s | 400M |  |
| [tailscale](https://github.com/tailscale/tailscale) _v0.98.0_ | 275 | 2.5s | 539M |  |
| [etcd](https://github.com/etcd-io/etcd) _v3.4.7_ | 391 | 3.5s | 700M | _(need run `go mod vendor` before running **Gold**)_ |
| [go-ethereum](https://github.com/ethereum/go-ethereum) _v1.9.14_ | 459 | 5.5s | 1.3G | |
| [terraform](https://github.com/hashicorp/terraform) _v0.12.25_ | 777 | 5.7s | 1.5G | |
| [consul](https://github.com/hashicorp/consul) _v1.7.3_ | 803 | 7.2s | 1.9G | |
| [vitess](https://github.com/vitessio/vitess) _v6.0.20-20200525_ | 905 | 7.1s | 1.7G | |
| [istio](https://github.com/istio/istio) _1.6.0_ | 1860 | 10.7s | 2.8G | |
| [kubernetes](https://github.com/kubernetes/kubernetes) _v1.18.2_ | 2821 | 16.3s | 4G | |

