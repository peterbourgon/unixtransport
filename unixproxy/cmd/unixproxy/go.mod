module unixproxy

go 1.20

replace github.com/peterbourgon/unixtransport => ../../..

require (
	github.com/oklog/run v1.1.0
	github.com/peterbourgon/ff/v3 v3.3.0
	github.com/peterbourgon/unixtransport v0.0.0-00010101000000-000000000000
)
