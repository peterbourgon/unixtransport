// package unixproxy provides an EXPERIMENTAL reverse proxy to Unix sockets.
//
// The intent of this package is to facilitate local development of distributed
// systems, by allowing normal HTTP clients that assume TCP (cURL, browsers,
// etc.) to address localhost servers via semantically-meaningful subdomains
// rather than opaque port numbers. The intermediating reverse-proxy works
// dynamically, without any explicit configuration.
//
// [Handler] provides the reverse-proxy logic. See documentation on that type
// for usage information.
//
// Application servers need to be able to listen on Unix sockets. The [ParseURI]
// and [ListenURI] helpers exist for this purpose. They accept both typical
// listen addresses e.g. "localhost:8081" or ":8081" as well as e.g.
// "unix:///tmp/my.sock" as input. Applications can use these helpers to create
// listeners, and bind HTTP servers to those listeners, rather than using
// default helpers that assume TCP, like [http.ListenAndServe].
//
// cmd/unixproxy is an example program utilizing package unixproxy.
package unixproxy
