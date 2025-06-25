package unixtransport_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/peterbourgon/unixtransport"
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

		// Verify clean parsing of scheme-less URIs.
		{uri: "example.com:443/path/part", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443?query=value", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/path?and=query", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/path?and=query#andfragment", network: "tcp", address: "example.com:443"},
		{uri: "example.com:443/#justfragment", network: "tcp", address: "example.com:443"},

		// Normal URL schemes like http:// are parsed as literal networks and not transformed to e.g. tcp.
		{uri: "http://example.com:8080", network: "http", address: "example.com:8080"},
		{uri: "https://example.com?foo=bar", network: "https", address: "example.com"},
		{uri: "http+unix://tmp/unix/socket:/something/else", network: "http+unix", address: "tmp"},
		{uri: "http+unix:///tmp/unix/socket:/something/else", network: "http+unix", address: "/tmp/unix/socket:/something/else"},

		// Edge conditions.
		{uri: "anything://:8080", network: "anything", address: ":8080"},
		{uri: "file:///path/to/file.txt", network: "file", address: "/path/to/file.txt"},
		{uri: "file://rel/path/index.txt", network: "file", address: "rel"},
		{uri: "foo://bar", network: "foo", address: "bar"},
		{uri: "unix://tmp/my.sock", network: "unix", address: "tmp"},               // no unix relative paths
		{uri: "unix://tmp:8080/abc/my.sock", network: "unix", address: "tmp:8080"}, // likewise above
		{uri: ":", network: "tcp", address: ":"},
		{uri: "", err: true},
		{uri: "localhost:8080/a", network: "tcp", address: "localhost:8080"},
	} {
		t.Run(testcase.uri, func(t *testing.T) {
			network, address, err := unixtransport.ParseURI(testcase.uri)
			switch {
			case testcase.err:
				if err == nil {
					t.Fatalf("want error, have none (network %q, address %q)", network, address)
				}
			case !testcase.err:
				if network != testcase.network || address != testcase.address {
					t.Fatalf("want %q %q, have %q %q (err=%v)", testcase.network, testcase.address, network, address, err)
				}
			}
		})
	}
}

func TestListenURI(t *testing.T) {
	t.Parallel()

	ln, err := unixtransport.ListenURI(context.Background(), "tcp://127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenURI failed: %v", err)
	}

	addr := ln.Addr().String()
	ln.Close()
	if _, _, err := net.SplitHostPort(addr); err != nil {
		t.Errorf("returned Addr() %q is not host:port", addr)
	}

	if _, err := unixtransport.ListenURI(context.Background(), ""); err == nil {
		t.Errorf("ListenURI('') expected error, got nil")
	}
}

// TestListenURIConfig verifies ListenURIConfig over both TCP and Unix endpoints.
func TestListenURIConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// TCP case
	ln, err := unixtransport.ListenURIConfig(ctx, "tcp://127.0.0.1:0", net.ListenConfig{})
	if err != nil {
		t.Fatalf("TCP ListenURIConfig failed: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	if _, _, err := net.SplitHostPort(addr); err != nil {
		t.Errorf("returned Addr %q is not host:port: %v", addr, err)
	}

	// Unix socket case
	dir := t.TempDir()
	socket := filepath.Join(dir, "sock")
	uri := "unix://" + socket
	ln2, err := unixtransport.ListenURIConfig(ctx, uri, net.ListenConfig{})
	if err != nil {
		t.Fatalf("Unix ListenURIConfig failed: %v", err)
	}
	ln2.Close()
	fi, err := os.Stat(socket)
	if err != nil {
		t.Errorf("socket file %q not created: %v", socket, err)
	} else if fi.Mode()&os.ModeSocket == 0 {
		t.Errorf("%q is not a socket", socket)
	}
}

func TestListenURIConfig_Control(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	socket := filepath.Join(t.TempDir(), "sock")

	type networkAddress struct {
		network string
		address string
	}

	cases := []struct {
		uri  string
		want networkAddress
	}{
		{
			uri:  "[::1]:0",
			want: networkAddress{"tcp6", "[::1]:0"},
		},
		{
			uri:  "tcp://127.0.0.1:0",
			want: networkAddress{"tcp4", "127.0.0.1:0"},
		},
		{
			uri:  "unix://" + socket,
			want: networkAddress{"unix", socket},
		},
	}
	for _, tc := range cases {
		t.Run(tc.uri, func(t *testing.T) {
			var seen []networkAddress

			cfg := net.ListenConfig{
				Control: func(network, address string, c syscall.RawConn) error {
					seen = append(seen, networkAddress{network, address})
					return nil
				},
			}

			ln, err := unixtransport.ListenURIConfig(ctx, tc.uri, cfg)
			if err != nil {
				t.Fatalf("ListenURIConfig: %v", err)
			}
			t.Logf("ln: %s", ln.Addr())
			ln.Close()

			if want, have := 1, len(seen); want != have {
				t.Fatalf("seen: want %d, have %d", want, have)
			}

			if want, have := tc.want, seen[0]; want != have {
				t.Errorf("seen: want %q, have %q", want, have)
			}
		})
	}
}
