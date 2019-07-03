### Usage:

```
Usage of ./go-netstat:
  -4    display only IPv4 sockets
  -6    display only IPv6 sockets
  -all
    	display both listening and non-listening sockets
  -help
    	display this help screen
  -lis
    	display only listening sockets
  -res
        lookup symbolic names for host addresses
  -tcp
    	display TCP sockets
  -udp
    	display UDP sockets
```
### Installation:

```
$ go get github.com/cakturk/go-netstat
```

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
