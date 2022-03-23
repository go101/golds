module go101.org/golds

go 1.18

// ToDo: use retract when Go 1.19.
//       and in readme, mention min Go version is Go 1.17.
//retract (
//	v0.2.7 // todo: up to v0.3.0
//	v0.2.4
//	v0.2.3
//    v0.3.7
//)

require (
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/text v0.3.7
	golang.org/x/tools v0.1.10
)

require (
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

// replace golang.org/x/tools => ./replaces/golang.org/x/tools
