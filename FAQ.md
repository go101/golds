
(ToDo: move to go101 website)

#### What does `Gold` mean?

go local docs server

go local directory server `gold -dir .`

#### Why Gold?

* show type implementaion relations
* distributed package docs (the Go culture). Make a central docs website is really hard in the Go modules age.

#### golden

The development/experiment version of gold

### why gold is recommended to run locally? Is gold is really capable for running on internet?

Not exactly, you can provide a public website for serving the docs, but it requires huge storage to cache the analyse result.

### what are the requirements to run gold?

* if the code need cgo, some c/c++ compilers might be needed.
* some code base might need large memory capacity.

### cgo

https://github.com/golang/go/wiki/WindowsBuild