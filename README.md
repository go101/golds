**Gold** is an experimental alternative Go local docs server.

### Features

* Show type implemention relations.
* Show promoted selectors, even on unexported embedded fields.
* Rich code view experience (good for study Go projects without opening IDEs).
* JavaScript-off friendly, though richer experiences when JS is enabled.
* Support generating static HTML pages (good for packages which don't satisfy go.dev license requirements).

This tool is still in its early phase. More features might be supported later.

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
* `gold -gen=./generated`
* `gold -gen=./generated ./...`
* `gold -gen=./generated std`

We can run `gold -dir .` or `gold -dir` from the generation directory to view the generated docs.

### Limitations

This project uses the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) package to parse code. The `golang.org/x/tools/go/package` is great, but it also has a shortcoming: there are no ways to get module/package downloading/preparing progress.

All packages must compile okay to get their docs shown.

Currently, testing packages are not analyzed.

### Analysis Cases

The following results are got on a machine with an AMD-2200G CPU and sufficient memory.
It is assumed that all involved modules/packages are fetched to local machine before running **Gold**.

| Project  | Package Count | Analysis Time | Final Memory Usage | Notes |
| ------------- | ------------- | ------------- | ------------- | ------------- |
| [kubernetes](https://github.com/kubernetes/kubernetes) _v1.18.2_ | 2821 | 16.3s | 4G | |
| [etcd](https://github.com/etcd-io/etcd) _v3.4.7_ | 391 | 3s | 700M | _(need run `go mod vendor` before running **Gold**)_ |

