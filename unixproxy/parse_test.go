package unixproxy_test

import (
	"testing"

	"github.com/peterbourgon/unixtransport/unixproxy"
)

func TestParseURI(t *testing.T) {
	for _, testcase := range []struct {
		uri              string
		network, address string
		err              bool
	}{
		// The package is designed to enable this kind of thing.
		{uri: "tcp://:8080", network: "tcp", address: ":8080"},
		{uri: "udp://:12345", network: "udp", address: ":12345"},
		{uri: "unix:///tmp/my.sock", network: "unix", address: "/tmp/my.sock"},

		// More normal stuff should work too, though.
		{uri: ":8080", network: "tcp", address: ":8080"},
		{uri: "localhost:8080", network: "tcp", address: "localhost:8080"},
		{uri: "localhost:0", network: "tcp", address: "localhost:0"},
		{uri: "localhost:", network: "tcp", address: "localhost:"},

		// Verify clean parsing of typical URLs.
		{uri: "example.com:443/path/part", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443?query=value", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/path?and=query", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/path?and=query#andfragment", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/#justfragment", network: "tcp", address: "example.com:443"},

		// Edge conditions.
		{uri: "anything://:8080", network: "anything", address: ":8080"},
		{uri: "file:///path/to/file.txt", network: "file", address: "/path/to/file.txt"},
		{uri: "file://rel/path/index.txt", network: "file", address: "rel"},
		{uri: "foo://bar", network: "foo", address: "bar"},
		{uri: "unix://tmp/my.sock", network: "unix", address: "tmp"},               // no unix relative paths
		{uri: "unix://tmp:8080/abc/my.sock", network: "unix", address: "tmp:8080"}, // likewise above
		{uri: ":", network: "tcp", address: ":"},
		{uri: "", err: true},
	} {
		t.Run(testcase.uri, func(t *testing.T) {
			network, address, err := unixproxy.ParseURI(testcase.uri)
			switch {
			case testcase.err:
				if err == nil {
					t.Fatalf("want error, have none (network %q, address %q)", network, address)
				}
			case !testcase.err:
				if network != testcase.network || address != testcase.address {
					t.Fatalf("want %q %q, have %q %q", testcase.network, testcase.address, network, address)
				}
			}
		})
	}
}
