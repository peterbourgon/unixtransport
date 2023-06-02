// package unixproxy provides an EXPERIMENTAL reverse proxy to Unix sockets.
//
// The intent of this package is to facilitate local development of distributed
// systems, by allowing normal HTTP clients that assume TCP (cURL, browsers,
// etc.) to address localhost servers via semantically-meaningful subdomains
// rather than opaque port numbers.
//
// For example, rather than addressing your application server as localhost:8081
// and your Prometheus instance as localhost:9090, you could use
//
//	http://myapp.unixproxy.localhost
//	http://prometheus.unixproxy.localhost
//
// Or, you could have 3 clusters of 3 instances each, addressed as
//
//	http://{nyc,lax,fra}.{1,2,3}.unixproxy.localhost
//
// The intermediating reverse-proxy, provided by [Handler], works dynamically,
// without any explicit configuration. See documentation on that type for usage
// information.
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
