## go-netstat / Sliver

A modified version of https://github.com/cakturk/go-netstat, modifications have been made for interoperability with the rest of Sliver:
* Added Darwin support
* Data structures have been ported to protobuf


### Using as a library
#### [Godoc](https://godoc.org/github.com/cakturk/go-netstat/netstat)
#### Getting the package
```
$ go get github.com/cakturk/go-netstat/netstat
```

```go
import (
	"fmt"

	"github.com/cakturk/go-netstat/netstat"
)

func displaySocks() error {
	// UDP sockets
	socks, err := netstat.UDPSocks(netstat.NoopFilter)
	if err != nil {
		return err
	}
	for _, e := range socks {
		fmt.Printf("%v\n", e)
	}

	// TCP sockets
	socks, err = netstat.TCPSocks(netstat.NoopFilter)
	if err != nil {
		return err
	}
	for _, e := range socks {
		fmt.Printf("%v\n", e)
	}

	// get only listening TCP sockets
	tabs, err = netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen
	})
	if err != nil {
		return err
	}
	for _, e := range socks {
		fmt.Printf("%v\n", e)
	}

	// list all the TCP sockets in state FIN_WAIT_1 for your HTTP server
	tabs, err = netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.FinWait1 && s.LocalAddr.Port == 80
	})
	// error handling, etc.

	return nil
}
```
