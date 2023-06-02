package unixproxy

import (
	"fmt"
	"io"
	"log"

	"github.com/miekg/dns"
)

// NewDNSServer returns a DNS server which will listen on addr, and resolve all
// incoming A, AAAA, and HTTPS requests to localhost. Specifically, it resolves
// all A and HTTPS queries to the IPv4 address 127.0.0.1, and all AAAA queries
// to the IPv6 address ::1. It ignores all other request types.
//
// A nil logger parameter is valid and will result in no log output.
//
// This is intended for use on macOS systems, where many applications (including
// Safari and cURL) perform DNS lookups through a system resolver that ignores
// /etc/hosts. As a workaround, users can run this (limited) DNS resolver on a
// specific local port, and configure the system resolver to use it when
// resolving hosts matching the relevant host string.
//
// Assuming the default host of unixproxy.localhost, and assuming this resolver
// runs on 127.0.0.1:5354, create /etc/resolver/localhost with the following
// content.
//
//	nameserver 127.0.0.1
//	port 5354
//
// Then e.g. Safari will resolve any URL ending in .localhost by querying the
// resolver running on 127.0.0.1:5354. See `man 5 resolver` for more information
// on the /etc/resolver file format.
func NewDNSServer(addr string, logger *log.Logger) *dns.Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	mux := dns.NewServeMux()

	mux.HandleFunc(".", func(w dns.ResponseWriter, request *dns.Msg) {
		for i, q := range request.Question {
			logger.Printf("-> DNS %d/%d: %s", i+1, len(request.Question), q.String())
		}
		response := getResponse(request, logger)
		for i, a := range response.Answer {
			logger.Printf("<- DNS %d/%d: %s", i+1, len(response.Answer), a.String())
		}
		w.WriteMsg(response)
	})

	return &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: mux,
	}
}

func getResponse(request *dns.Msg, logger *log.Logger) *dns.Msg {
	var response dns.Msg
	response.SetReply(request)
	response.Compress = false

	if request.Opcode != dns.OpcodeQuery {
		return &response
	}

	var answer []dns.RR
	for _, q := range response.Question {
		var (
			rr  dns.RR
			err error
		)
		switch q.Qtype {
		case dns.TypeA:
			rr, err = dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", q.Name))
		case dns.TypeAAAA:
			rr, err = dns.NewRR(fmt.Sprintf("%s AAAA ::1", q.Name))
		case dns.TypeHTTPS:
			rr, err = dns.NewRR(fmt.Sprintf("%s HTTPS 1 127.0.0.1", q.Name))
		default:
			err = fmt.Errorf("unsupported question type %T", q.Qtype)
		}
		if err != nil {
			logger.Printf("%s %s: %v", dns.TypeToString[q.Qtype], q.Name, err)
			return &response
		}
		answer = append(answer, rr)
	}

	response.Answer = answer
	return &response
}
