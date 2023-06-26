package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
)

func main() {
	switch os.Args[1] {
	case "sock":
		if err := mainSock(); err != nil {
			panic(err)
		}
	case "http":
		if err := mainHTTP(); err != nil {
			panic(err)
		}
	case "nonblock":
		if err := mainNonblock(os.Args[2], os.Args[3:]); err != nil {
			panic(err)
		}
	}
}

// mainSock is an explicit test of a blocking socket.
func mainSock() error {
	// Get a listener from the pre-opened file descriptor.
	// The listener is the first pre-open, with a file-descriptor of 3.
	f := os.NewFile(3, "")
	l, err := net.FileListener(f)
	defer f.Close()
	if err != nil {
		return err
	}
	defer l.Close()

	// Accept a connection
	conn, err := l.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Do a blocking read of up to 32 bytes.
	// Note: the test should write: "wazero", so that's all we should read.
	var buf [32]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		return err
	}
	fmt.Println(string(buf[:n]))
	return nil
}

// mainHTTP implicitly tests non-blocking sockets, as they are needed for
// middleware.
func mainHTTP() error {
	// Get the file representing a pre-opened TCP socket.
	// The socket (listener) is the first pre-open, with a file-descriptor of
	// 3 because the host didn't add any pre-opened files.
	listenerFD := 3
	f := os.NewFile(uintptr(listenerFD), "")

	// Wasm runs similarly to GOMAXPROCS=1, so multiple goroutines cannot work
	// in parallel. non-blocking allows the poller to park the go-routine
	// accepting connections while work is done on one.
	if err := syscall.SetNonblock(listenerFD, true); err != nil {
		return err
	}

	// Convert the file representing the pre-opened socket to a listener, so
	// that we can integrate it with HTTP middleware.
	ln, err := net.FileListener(f)
	defer f.Close()
	if err != nil {
		return err
	}
	defer ln.Close()

	// Serve middleware that echos the request body to the response once, then quits.
	h := &echoOnce{ch: make(chan struct{}, 1)}
	go http.Serve(ln, h)
	<-h.ch
	return nil
}

type echoOnce struct {
	ch chan struct{}
}

func (e echoOnce) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Copy up to 32 bytes from the request to the response, appending a newline.
	// Note: the test should write: "wazero", so that's all we should read.
	var buf [32]byte
	if n, err := r.Body.Read(buf[:]); err != nil && err != io.EOF {
		panic(err)
	} else if n, err = w.Write(append(buf[:n], '\n')); err != nil {
		panic(err)
	}
	// Once one request was served, close the channel.
	close(e.ch)
}

// Adapted from nonblock.go
// https://github.com/golang/go/blob/0fcc70ecd56e3b5c214ddaee4065ea1139ae16b5/src/runtime/internal/wasitest/testdata/nonblock.go
func mainNonblock(mode string, files []string) error {
	ready := make(chan struct{})

	var wg sync.WaitGroup
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		switch mode {
		case "open":
		case "create":
			fd := f.Fd()
			if err = syscall.SetNonblock(int(fd), true); err != nil {
				return err
			}
			f = os.NewFile(fd, path)
		default:
			return fmt.Errorf("invalid test mode")
		}

		spawnWait := make(chan struct{})

		wg.Add(1)
		go func(f *os.File) {
			defer f.Close()
			defer wg.Done()

			// Signal the routine has been spawned.
			close(spawnWait)

			// Wait until ready.
			<-ready

			var buf [256]byte

			if n, err := f.Read(buf[:]); err != nil {
				panic(err)
			} else {
				os.Stderr.Write(buf[:n])
			}
		}(f)

		// Spawn one goroutine at a time.
		<-spawnWait
	}

	println("waiting")
	close(ready)
	wg.Wait()
	return nil
}
