package unixproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Handler is a reverse proxy to Unix sockets on the local filesystem.
//
// Requests are mapped to target Unix sockets based on their Host header. Each
// sub-domain element underneath the configured Host TLD is parsed as a filepath
// element relative to Root directory. If that resolved filepath is a valid Unix
// socket, the request is proxied to that socket, via an [http.Client] that
// utilizes [github.com/peterbourgon/unixtransport].
//
// For example, a Handler configured with Host "unixproxy.localhost" and Root
// "/tmp/abc" would map a request with Host "foo.bar.unixproxy.localhost" to a
// Unix socket at "/tmp/abc/foo/bar", if it exists.
type Handler struct {
	// Host is the top-level domain (TLD) of the Handler, which is expected to
	// end in ".localhost" as per RFC2606. The system should be configured to
	// resolve requests to this domain (and all subdomains) to localhost,
	// typically via an entry in "/etc/hosts".
	//
	// Optional. The default value is "unixproxy.localhost".
	Host string

	// Root is a valid directory on the local filesystem. The handler will look
	// in this directory tree, recursively, for valid Unix sockets, when
	// proxying a given request.
	//
	// Required.
	Root string

	// ErrorLogWriter is used as the destination writer for the ErrorLog of the
	// [http.ReverseProxy] used to proxy individual requests.
	//
	// Optional. By default, each [http.ReverseProxy] has a nil ErrorLog.
	ErrorLogWriter io.Writer
}

// Domains returns a slice of strings that represent valid Host headers for
// incoming requests. It computes this slice by walking the directory tree from
// the Root directory via [filepath.WalkDir], and converting the relative file
// path of any valid Unix socket to a FQDN underneath the Host TLD. Results are
// not cached.
func (h *Handler) Domains() ([]string, error) {
	var domains []string
	if err := filepath.WalkDir(h.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSocket == 0 {
			return nil
		}

		relpath, err := filepath.Rel(h.Root, path)
		if err != nil {
			return err
		}

		subdomain := strings.Replace(relpath, string(filepath.Separator), ".", -1)
		domain := strings.Trim(subdomain, ".") + "." + strings.Trim(h.Host, ".")
		domains = append(domains, domain)
		return nil
	}); err != nil {
		return nil, err
	}
	return domains, nil
}

// ServeHTTP implements http.Handler. If the request Host header is equal to the
// Host field (i.e. has no subdomains), ServeHTTP will serve a list of valid
// subdomains. Otherwise, the request will be proxied to a local Unix domain
// socket based on its subdomain.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/favicon.ico":
		http.NotFound(w, r)
	case r.Host == h.Host:
		h.handleIndex(w, r)
	default:
		h.handleProxy(w, r)
	}
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	domains, err := h.Domains()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accept := strings.ToLower(r.Header.Get("accept"))
	switch {
	case strings.Contains(accept, "text/html"):
		var buf bytes.Buffer
		if err := indexTemplate.Execute(&buf, struct{ Domains []string }{domains}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "text/html; charset=utf-8")
		buf.WriteTo(w)

	case strings.Contains(accept, "application/json"):
		w.Header().Set("content-type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		enc.Encode(domains)

	default:
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		for _, s := range domains {
			fmt.Fprintln(w, s)
		}
	}
}

func (h *Handler) handleProxy(w http.ResponseWriter, r *http.Request) {
	var (
		clean    = strings.TrimSuffix(r.Host, h.Host)
		elements = strings.Split(clean, ".")
		relative = filepath.Join(elements...)
		socket   = filepath.Join(h.Root, relative)
	)

	fi, err := os.Stat(socket) // TODO: sanitize, chroot, etc.
	if err != nil || fi.Mode()&os.ModeSocket == 0 {
		http.Error(w, fmt.Sprintf("target socket %s invalid", socket), http.StatusNotFound)
		return
	}

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = socket
		req.URL.Path = r.URL.Path
	}

	var proxyLog *log.Logger
	if h.ErrorLogWriter != nil {
		proxyLog = log.New(h.ErrorLogWriter, fmt.Sprintf("unixproxy: %s: ", relative), 0)
	}

	rp := &httputil.ReverseProxy{
		Transport: onlyUnixTransport,
		ErrorLog:  proxyLog,
		Director:  director,
	}

	rp.ServeHTTP(w, r)
}

var onlyUnixTransport = &http.Transport{
	DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err == nil {
			address = host
		}
		return (&net.Dialer{}).DialContext(ctx, "unix", address)
	},
}

var indexTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
<title>unixproxy</title>
</head>
<body>
<ul>
{{ range .Domains -}}
<li><a href="//{{.}}">{{.}}</a></li>
{{ else -}}
<li>No active sockets found</li>
{{ end -}}
</ul>
`))
