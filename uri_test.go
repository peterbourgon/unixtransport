package unixtransport_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
		{uri: "://", err: true},
		{uri: "tcp://", err: true},
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

	ctx := context.Background()
	socket := filepath.Join(t.TempDir(), "sock")

	ln, err := unixtransport.ListenURI(ctx, "unix://"+socket)
	if err != nil {
		t.Fatalf("ListenURI failed: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello", r.URL.Path)
	})
	server := httptest.NewUnstartedServer(handler)
	server.Listener = ln
	server.Start()
	t.Cleanup(func() { server.Close() })

	transport := &http.Transport{}
	unixtransport.Register(transport)
	client := &http.Client{Transport: transport}

	rawurl := "http+unix://" + socket + ":/foo"
	want := "hello /foo"
	have := get(t, client, rawurl)
	if want != have {
		t.Errorf("%s: want %q, have %q", rawurl, want, have)
	}
}

func TestListenURIConfig(t *testing.T) {
	t.Parallel()

	type networkAddress struct {
		network string
		address string
	}

	var seen []networkAddress
	control := func(network string, address string, c syscall.RawConn) error {
		seen = append(seen, networkAddress{network, address})
		return nil
	}

	var (
		socket = filepath.Join(t.TempDir(), "sock")
		ctx    = context.Background()
		uri    = "unix://" + socket
		cfg    = net.ListenConfig{Control: control}
	)

	ln, err := unixtransport.ListenURIConfig(ctx, uri, cfg)
	if err != nil {
		t.Fatalf("Unix ListenURIConfig failed: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello", r.URL.Path)
	})
	server := httptest.NewUnstartedServer(handler)
	server.Listener = ln
	server.Start()
	t.Cleanup(func() { server.Close() })

	transport := &http.Transport{}
	unixtransport.Register(transport)
	client := &http.Client{Transport: transport}
	rawurl := "http+unix://" + socket + ":/foo"
	if want, have := "hello /foo", get(t, client, rawurl); want != have {
		t.Errorf("%s: want %q, have %q", rawurl, want, have)
	}

	if want, have := []networkAddress{{"unix", socket}}, seen; !reflect.DeepEqual(want, have) {
		t.Errorf("seen: want %+v, have %+v", want, have)
	}
}

func TestListenURIRemoveFailures(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("invalid URI", func(t *testing.T) {
		uri := "unix:///"
		if _, err := unixtransport.ListenURI(ctx, uri); err == nil {
			t.Fatalf("ListenURI(%s): expected error, got none", uri)
		}
	})

	t.Run("bad permission", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "dir")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("os.Mkdir: %v", err)
		}

		sock := filepath.Join(dir, "sock")
		if err := os.WriteFile(sock, []byte{}, 0o644); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}

		// The only way to trigger an error on the os.Remove of the socket file
		// in a unit test like this one, is to remove the write permission on
		// the parent directory.
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatalf("os.Chmod(%s, 0555): %v", dir, err)
		}
		defer func() { // allow test cleanup to remove the TempDir
			if err := os.Chmod(dir, 0o755); err != nil {
				t.Errorf("os.Chmod(%s, 0755): %v", dir, err)
			}
		}()

		uri := "unix://" + sock
		if _, err := unixtransport.ListenURI(ctx, uri); err == nil {
			t.Fatalf("ListenURI(%s): expected error, got none", uri)
		}
	})

	t.Run("socket is directory", func(t *testing.T) {
		d := filepath.Join(t.TempDir(), "dir-as-sock")
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("os.Mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(d, "xxx"), []byte(`abc`), 0o644); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
		uri := "unix://" + d
		if _, err := unixtransport.ListenURI(ctx, uri); err == nil {
			t.Errorf("ListenURI(%s): expected error, got none", uri)
		}
	})

	t.Run("listen error", func(t *testing.T) {
		uri := "doesnotexist://foo"
		if _, err := unixtransport.ListenURI(ctx, uri); err == nil {
			t.Fatalf("ListenURI(%s): expected error, got none", uri)
		}
	})
}
