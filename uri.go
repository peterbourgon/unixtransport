package unixtransport

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ParseURI parses the given URI to a network and address that can be passed to
// e.g. [net.Listen].
//
// The URI scheme is interpreted as the network, and the host (and port) are
// interpreted as the address. For example, "tcp://:80" is parsed to a network
// of "tcp" and an address of ":80", and "unix:///tmp/my.sock" is parsed to a
// network of "unix" and an address of "/tmp/my.sock".
//
// If the URI doesn't have a scheme, "tcp://" is assumed by default, in an
// attempt to keep basic compatibilty with common listen addresses like
// "localhost:8080" or ":9090".
func ParseURI(uri string) (network, address string, _ error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", "", fmt.Errorf("empty URI")
	}

	if !strings.Contains(uri, "://") {
		uri = "tcp://" + uri
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}

	if u.Scheme == "" {
		u.Scheme = "tcp"
	}

	if u.Host == "" && u.Path != "" {
		u.Host = u.Path
		u.Path = ""
	}

	if u.Host == "" {
		return "", "", fmt.Errorf("empty host in URI %q", uri)
	}

	return u.Scheme, u.Host, nil
}

// ListenURI is a helper function that calls [ListenURIConfig] with a default
// [net.ListenConfig].
//
// The provided context is only used when resolving the address, it has no
// effect on the returned listener.
//
// For more precise control over the listener, use [ParseURI] and construct the
// listener directly.
func ListenURI(ctx context.Context, uri string) (net.Listener, error) {
	return ListenURIConfig(ctx, uri, net.ListenConfig{})
}

// ListenURIConfig is a helper function that parses the provided URI via
// [ParseURI], and returns a [net.Listener] listening on the parsed network and
// address, and using the provided config.
//
// The provided context is only used when resolving the address, it has no
// effect on the returned listener.
//
// For more precise control over the listener, use [ParseURI] and construct the
// listener directly.
func ListenURIConfig(ctx context.Context, uri string, config net.ListenConfig) (net.Listener, error) {
	network, address, err := ParseURI(uri)
	if err != nil {
		return nil, err
	}

	listener, err := config.Listen(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	return listener, nil
}
