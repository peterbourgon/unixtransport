module unixproxy

go 1.20

require (
	github.com/oklog/run v1.1.0
	github.com/peterbourgon/ff/v3 v3.3.0
	github.com/peterbourgon/unixtransport v0.0.0-00010101000000-000000000000
)

require (
	github.com/miekg/dns v1.1.54 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/tools v0.3.0 // indirect
)

replace github.com/peterbourgon/unixtransport => ../../..
