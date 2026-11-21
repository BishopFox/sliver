package main

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultURL     = "https://example.com/"
	maxResponseLen = 1 << 20
)

// publicRootsPEM is a pinned Mozilla public CA extract, embedded because Go's
// wasip1 port has no operating-system certificate store.
//
//go:embed public-roots.pem
var publicRootsPEM []byte

func main() {
	target := defaultURL
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(publicRootsPEM) {
		fmt.Fprintln(os.Stderr, "could not parse embedded public TLS roots")
		os.Exit(1)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
	}
	defer transport.CloseIdleConnections()

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	response, err := client.Get(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GET %s: %v\n", target, err)
		os.Exit(1)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseLen))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", target, err)
		os.Exit(1)
	}
	fmt.Printf("GET %s: %s\n%s", target, response.Status, body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		os.Exit(1)
	}
}
