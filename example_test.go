package unixtransport_test

import (
	"net/http"

	"github.com/peterbourgon/unixtransport"
)

func ExampleRegisterDefault() {
	// Register the "http+unix" and "https+unix" protocols in the
	// net/http.DefaultClient's transport.
	unixtransport.RegisterDefault()

	// This makes a GET request to the HTTP server listening at /tmp/my.sock.
	// Note the three '/' characters between 'http+unix:' and 'tmp' -- the
	// scheme is 'http+unix://' and the socket is '/tmp/my.sock'.
	http.Get("http+unix:///tmp/my.sock")

	// This shows how to include a request path and query.
	http.Get("http+unix:///tmp/my.sock:/users/123?q=abc")
}

func ExampleRegister() {
	// Create your http.Transport.
	t := &http.Transport{
		// ...
	}

	// Register "http+unix" and "https+unix" protocols in that transport.
	unixtransport.Register(t)

	// Create an http.Client using the transport.
	c := &http.Client{
		Transport: t,
		// ...
	}

	// Make a GET request to the HTTP server listening at /tmp/my.sock.
	c.Get("https+unix:///tmp/my.sock")
}
