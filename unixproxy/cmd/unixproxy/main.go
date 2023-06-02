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
		addrFlag = fs.String("addr", ":80", "listen endpoint for HTTP reverse proxy server")
		hostFlag = fs.String("host", "unixproxy.localhost", "Host header where this service is reachable")
		rootFlag = fs.String("root", ".", "root path to look for Unix sockets")
		dnsFlag  = fs.String("dns", "", "listen endpoint for localhost DNS resolver (optional)")
	)
	fs.Usage = usageFor(fs)
	if err := ff.Parse(fs, args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	logger := log.New(stderr, "", 0)

	proxyListener, err := unixproxy.ListenURI(ctx, *addrFlag)
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

		fmt.Fprintf(buf, "DOCUMENTATION\n")
		fmt.Fprintf(buf, "  https://github.com/peterbourgon/unixtransport/tree/main/unixproxy\n")
		fmt.Fprintf(buf, "\n")

		fmt.Fprint(os.Stdout, buf.String())
	}
}
