package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"
	"text/tabwriter"

	"github.com/oklog/run"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/unixtransport"
	"github.com/peterbourgon/unixtransport/unixproxy"
)

func main() {
	err := exe(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Args[1:],
	)
	switch {
	case err == nil:
		os.Exit(0)
	case errors.Is(err, flag.ErrHelp):
		os.Exit(1)
	case isSignalError(err):
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func exe(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("unixproxy", flag.ContinueOnError)
	var (
		addrFlag = fs.String("addr", ":80", "listen address for HTTP reverse proxy server")
		hostFlag = fs.String("host", "unixproxy.localhost", "Host header where this service is reachable")
		rootFlag = fs.String("root", ".", "root path to look for Unix sockets")
		dnsFlag  = fs.String("dns", "", "listen address for optional local DNS resolver (e.g. ':5354')")
	)
	fs.Usage = usageFor(fs)
	if err := ff.Parse(fs, args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	logger := log.New(stderr, "", 0)

	proxyListener, err := unixtransport.ListenURI(ctx, *addrFlag)
	if err != nil {
		return fmt.Errorf("listen on proxy addr: %w", err)
	}

	proxyHandler := &unixproxy.Handler{
		Host:           *hostFlag,
		Root:           *rootFlag,
		ErrorLogWriter: logger.Writer(),
	}

	logger.Printf("serving host http://%s", *hostFlag)
	logger.Printf("sockets root %s", *rootFlag)

	var g run.Group

	{
		logger.Printf("proxy listening on %s", proxyListener.Addr())
		server := &http.Server{Handler: proxyHandler}
		g.Add(func() error {
			return server.Serve(proxyListener)
		}, func(error) {
			server.Close()
		})
	}

	if *dnsFlag != "" {
		logger.Printf("DNS resolver listening on %s", *dnsFlag)
		server := unixproxy.NewDNSServer(*dnsFlag, logger)
		g.Add(func() error {
			return server.ListenAndServe()
		}, func(error) {
			server.ShutdownContext(ctx)
		})
	}

	{
		g.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))
	}

	return g.Run()
}

func isSignalError(err error) bool {
	var sig run.SignalError
	return errors.As(err, &sig)
}

func usageFor(fs *flag.FlagSet) func() {
	return func() {
		buf := &bytes.Buffer{}
		fmt.Fprintf(buf, "USAGE\n")
		fmt.Fprintf(buf, "  %s [flags]\n", fs.Name())
		fmt.Fprintf(buf, "\n")

		fmt.Fprintf(buf, "FLAGS\n")
		tw := tabwriter.NewWriter(buf, 0, 4, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			def := f.DefValue
			if def == "" {
				def = "..."
			}
			fmt.Fprintf(tw, "  --%s=%s\t%s\n", f.Name, def, f.Usage)
		})
		tw.Flush()
		fmt.Fprintf(buf, "\n")

		fmt.Fprintf(buf, "EXAMPLES\n")
		fmt.Fprintf(buf, "  Make Unix sockets under /tmp/foo accessible at http://cool.pizza (macOS)\n")
		fmt.Fprintf(buf, "\n")
		fmt.Fprintf(buf, `    sudo printf "nameserver 127.0.0.1\nport 5354\n" > /etc/resolver/pizza`+"\n")
		fmt.Fprintf(buf, "    sudo unixproxy --root=/tmp/foo --host=cool.pizza --addr=:80 --dns=:5354\n")
		fmt.Fprintf(buf, "    open 'http://cool.pizza'\n")
		fmt.Fprintf(buf, "\n")

		fmt.Fprintf(buf, "DOCUMENTATION\n")
		fmt.Fprintf(buf, "  https://pkg.go.dev/github.com/peterbourgon/unixtransport/unixproxy\n")
		fmt.Fprintf(buf, "\n")

		fmt.Fprint(os.Stdout, buf.String())
	}
}
