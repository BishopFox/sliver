# go-socks5 

[![GoDoc](https://godoc.org/github.com/things-go/go-socks5?status.svg)](https://godoc.org/github.com/things-go/go-socks5)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/things-go/go-socks5?tab=doc)
![Action Status](https://github.com/things-go/go-socks5/workflows/Go/badge.svg)
[![codecov](https://codecov.io/gh/things-go/go-socks5/branch/master/graph/badge.svg)](https://codecov.io/gh/things-go/go-socks5)
[![Go Report Card](https://goreportcard.com/badge/github.com/things-go/go-socks5)](https://goreportcard.com/report/github.com/things-go/go-socks5)
[![License](https://img.shields.io/github/license/things-go/go-socks5)](https://github.com/things-go/go-socks5/raw/master/LICENSE)
[![Tag](https://img.shields.io/github/v/tag/things-go/go-socks5)](https://github.com/things-go/go-socks5/tags)

Provides the `socks5` package that implements a [SOCKS5](http://en.wikipedia.org/wiki/SOCKS).
SOCKS (Secure Sockets) is used to route traffic between a client and server through
an intermediate proxy layer. This can be used to bypass firewalls or NATs.

### Feature


The package has the following features:
- Support socks5 server
- Support TCP/UDP and IPv4/IPv6
- Unit tests
- "No Auth" mode
- User/Password authentication optional user addr limit
- Support for the CONNECT command
- Support for the ASSOCIATE command
- Rules to do granular filtering of commands
- Custom DNS resolution
- Custom goroutine pool
- buffer pool design and optional custom buffer pool
- Custom logger

### TODO

The package still needs the following:
- Support for the BIND command

### Installation

Use go get.
```bash
    go get github.com/things-go/go-socks5
```

Then import the socks5 server package into your own code.

```bash
    import "github.com/things-go/go-socks5"
```

### Example

Below is a simple example of usage, more see [example](https://github.com/things-go/go-socks5/tree/master/_example)

[embedmd]:# (_example/main.go go)
```go
package main

import (
	"log"
	"os"

	"github.com/things-go/go-socks5"
)

func main() {
	// Create a SOCKS5 server
	server := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)

	// Create SOCKS5 proxy on localhost port 8000
	if err := server.ListenAndServe("tcp", ":8000"); err != nil {
		panic(err)
	}
}
```

### Reference
- [rfc1928](https://www.ietf.org/rfc/rfc1928.txt) 
- original armon's [go-sock5](https://github.com/armon/go-socks5) library

## License

This project is under MIT License. See the [LICENSE](LICENSE) file for the full license text.

