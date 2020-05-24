Gold is an experimental alternative Go local docs server.

### Features

* Show type implemention relations.
* Show promoted selectors, even on unexported embedded fields.
* Rich code view experience (good for study Go projects without opening IDEs).
* JavaScript-off friendly, though richer experiences when JS is enabled..
* Support generating static HTML pages (good for packages which don't satisfy go.dev license requirements).

### Usage

* Run `gold .` or `gold` to show docs of the package in the current directory (and all its dependency packages).
* Run `gold ./...` to show docs of all the package under the current directory (and all their dependency packages).
* Run `gold std` to show docs of standard pacakges.
