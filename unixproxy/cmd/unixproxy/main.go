package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

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
		proxyAddr   = fs.String("proxy-addr", ":80", "HTTP listen endpoint for reverse proxy server")
		hostHeader  = fs.String("host-header", "unixproxy.localhost", "Host header where this service is reachable")
		socketsPath = fs.String("sockets-path", ".", "root path to look for Unix sockets")
	)
	if err := ff.Parse(fs, args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	logger := log.New(stderr, "", 0)

	proxyListener, err := unixproxy.ListenURI(ctx, *proxyAddr)
	if err != nil {
		return fmt.Errorf("listen on proxy addr: %w", err)
	}

	proxyHandler := &unixproxy.Handler{
		Host:           *hostHeader,
		Root:           *socketsPath,
		ErrorLogWriter: logger.Writer(),
	}

	logger.Printf("listening on %s", proxyListener.Addr())
	logger.Printf("serving host http://%s", *hostHeader)
	logger.Printf("proxying to sockets in %s", *socketsPath)

	var g run.Group

	{
		server := &http.Server{Handler: proxyHandler}
		g.Add(func() error {
			return server.Serve(proxyListener)
		}, func(error) {
			server.Close()
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
