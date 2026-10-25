package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
	"unsafe"
)

const networkTimeout = 5 * time.Second

var allocations = map[uint32][]byte{}

func main() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 {
	return allocate(make([]byte, max(size, 1)))
}

//go:wasmexport free
func free(pointer, _ uint32) {
	delete(allocations, pointer)
}

//go:wasmexport encode
func encode(pointer, size uint32) uint64 {
	return transform(pointer, size)
}

//go:wasmexport decode
func decode(pointer, size uint32) uint64 {
	return transform(pointer, size)
}

func transform(pointer, size uint32) uint64 {
	command := string(memory(pointer, size))
	parts := strings.SplitN(command, "\x00", 3)
	var output []byte
	var err error
	switch parts[0] {
	case "tcp", "udp":
		if len(parts) != 3 {
			err = fmt.Errorf("%s command requires address and payload", parts[0])
			break
		}
		output, err = roundTrip(parts[0]+"4", parts[1], []byte(parts[2]))
	case "block":
		if len(parts) < 2 {
			err = fmt.Errorf("block command requires an address")
			break
		}
		output, err = blockingRead(parts[1])
	case "https":
		if len(parts) != 3 {
			err = fmt.Errorf("https command requires a URL and PEM roots")
			break
		}
		output, err = fetchHTTPS(parts[1], []byte(parts[2]))
	case "lookup":
		if len(parts) < 2 {
			err = fmt.Errorf("lookup command requires a hostname")
			break
		}
		var addresses []string
		addresses, err = net.LookupHost(parts[1])
		sort.Strings(addresses)
		output = []byte(strings.Join(addresses, ","))
	default:
		err = fmt.Errorf("unknown network command %q", parts[0])
	}
	if err != nil {
		output = []byte("error: " + err.Error())
	}
	resultPointer := allocate(output)
	return uint64(resultPointer)<<32 | uint64(uint32(len(output)))
}

func fetchHTTPS(target string, rootsPEM []byte) ([]byte, error) {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootsPEM) {
		return nil, fmt.Errorf("could not parse TLS roots")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Timeout:   networkTimeout,
		Transport: transport,
	}
	response, err := client.Get(target)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	return io.ReadAll(io.LimitReader(response.Body, 1<<20))
}

func blockingRead(address string) ([]byte, error) {
	connection, err := net.Dial("tcp4", address)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	response := make([]byte, 1)
	if _, err := io.ReadFull(connection, response); err != nil {
		return nil, err
	}
	return response, nil
}

func roundTrip(network, address string, payload []byte) ([]byte, error) {
	connection, err := net.DialTimeout(network, address, networkTimeout)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(networkTimeout)); err != nil {
		return nil, err
	}
	if _, err := connection.Write(payload); err != nil {
		return nil, err
	}
	response := make([]byte, len(payload))
	if _, err := io.ReadFull(connection, response); err != nil {
		return nil, err
	}
	return response, nil
}

func allocate(buffer []byte) uint32 {
	if len(buffer) == 0 {
		buffer = make([]byte, 1)
	}
	pointer := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buffer))))
	allocations[pointer] = buffer
	return pointer
}

func memory(pointer, size uint32) []byte {
	if size == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(pointer))), size)
}
