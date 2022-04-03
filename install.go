package unixtransport

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// Install takes an unstarted *httptest.Server and configures the server and
// associated client for HTTP(S) over UNIX socket transport. The server passed
// to Install is started immediately, with optional TLS support if s.TLS is not
// nil.
//
// If the server is already started, Install will panic.
func Install(tb testing.TB, s *httptest.Server) {
	tb.Helper()

	ln, err := net.Listen("unix", filepath.Join(tb.TempDir(), "unixtransport.sock"))
	if err != nil {
		tb.Errorf("unixtransport: httptest.Server not configured: %v", err)
		return
	}

	tb.Cleanup(func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			tb.Errorf("unixtransport: close listener: %v", err)
		}
	})

	// Plumb in the listener and start the server using that listener. This sets
	// srv.URL, which we must later override.
	s.Listener = ln

	var scheme string
	if s.TLS != nil {
		scheme = "https+unix"
		s.StartTLS()
	} else {
		scheme = "http+unix"
		s.Start()
	}

	Register(s.Client().Transport.(*http.Transport))

	// Manually construct a valid URL suitable for use by unixtransport-enabled
	// HTTP(S) clients.
	s.URL = fmt.Sprintf("%s://%s:", scheme, ln.Addr())
}
