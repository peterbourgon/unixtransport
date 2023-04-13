package unixproxy_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peterbourgon/unixtransport/unixproxy"
)

func TestHandlerBasic(t *testing.T) {
	proxy, close := testHandlerServer(t, context.Background())
	defer close()

	if want, have := strings.Join([]string{
		"bar.unixproxy.localhost", // lexicographical order from walk
		"baz.unixproxy.localhost", //
		"foo.unixproxy.localhost", //
	}, "\n"), testBasicRequest(t, proxy, "unixproxy.localhost"); want != have {
		t.Errorf("GET unixproxy.localhost: want %q, have %q", want, have)
	}

	if want, have := "hello from foo", testBasicRequest(t, proxy, "foo.unixproxy.localhost"); want != have {
		t.Errorf("GET foo.unixproxy.localhost: want %q, have %q", want, have)
	}

	if want, have := "hello from bar", testBasicRequest(t, proxy, "bar.unixproxy.localhost"); want != have {
		t.Errorf("GET bar.unixproxy.localhost: want %q, have %q", want, have)
	}

	if want, have := "hello from baz", testBasicRequest(t, proxy, "baz.unixproxy.localhost"); want != have {
		t.Errorf("GET baz.unixproxy.localhost: want %q, have %q", want, have)
	}
}

func testHandlerServer(t *testing.T, ctx context.Context) (*httptest.Server, func()) {
	t.Helper()

	root := t.TempDir()

	var closers []func()

	for _, x := range []string{"foo", "bar", "baz"} {
		response := fmt.Sprintf("hello from %s", x)
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, response)
		}))

		listener, err := unixproxy.ListenURI(ctx, "unix://"+root+"/"+x)
		if err != nil {
			t.Fatalf("%s: %v", x, err)
		}

		server.Listener = listener
		server.Start()
		closers = append(closers, func() {
			server.Close()
			listener.Close()
		})
	}

	proxy := httptest.NewServer(&unixproxy.Handler{
		Host: "unixproxy.localhost",
		Root: root,
	})

	close := func() {
		for _, f := range closers {
			f()
		}
	}

	return proxy, close
}

func testBasicRequest(t *testing.T, proxy *httptest.Server, host string) string {
	t.Helper()

	req, _ := http.NewRequest("GET", proxy.URL, nil)
	req.Host = host
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return strings.TrimSpace(string(body))
}
