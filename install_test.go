package unixtransport_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/peterbourgon/unixtransport"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	// Same structure as TestBasics, but using the Install API.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, 1, r.URL.Path) })
	server := httptest.NewUnstartedServer(handler)
	unixtransport.Install(t, server)
	defer server.Close()

	// http+unix should work.
	{
		var (
			rawurl = server.URL + "/foo"
			want   = "1 /foo"
			have   = get(t, server.Client(), rawurl)
		)
		if want != have {
			t.Errorf("%s: want %q, have %q", rawurl, want, have)
		}
	}
}
