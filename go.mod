module go101.org/golds

go 1.20

// ToDo: use retract when Go 1.19.
//       and in readme, mention min Go version is Go 1.17.
//retract (
//	v0.2.7 // todo: up to v0.3.0
//	v0.2.4
//	v0.2.3
//    v0.3.7
//)

require (
	golang.org/x/net v0.7.0
	golang.org/x/text v0.7.0
	golang.org/x/tools v0.4.0
)

require (
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
)

// replace golang.org/x/tools => ./replaces/golang.org/x/tools
