package unixtransport_test

import (
	"net/http"

	"github.com/peterbourgon/unixtransport"
)

func ExampleRegister_default() {
	// Register the "http+unix" and "https+unix" protocols in the default client transport.
	unixtransport.Register(http.DefaultTransport.(*http.Transport))

	// This will issue a GET request to an HTTP server listening at /tmp/my.sock.
	// Note the three '/' characters between 'http+unix:' and 'tmp'.
	http.Get("http+unix:///tmp/my.sock")

	// This shows how to include a request path and query.
	http.Get("http+unix:///tmp/my.sock:/users/123?q=abc")
}

func ExampleRegister_custom() {
	t := &http.Transport{
		// ...
	}

	unixtransport.Register(t)

	c := &http.Client{
		Transport: t,
		// ...
	}

	c.Get("https+unix:///tmp/my.sock")
}
