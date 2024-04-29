package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	switch os.Args[1] {
	case "ls":
		var repeat bool
		if len(os.Args) == 4 {
			repeat = os.Args[3] == "repeat"
		}
		// Go doesn't open with O_DIRECTORY, so we don't end up with ENOTDIR,
		// rather EBADF trying to read the directory later.
		if err := mainLs(os.Args[2], repeat); errors.Is(err, syscall.EBADF) {
			fmt.Println("ENOTDIR")
		} else if err != nil {
			panic(err)
		}
	case "stat":
		if err := mainStat(); err != nil {
			panic(err)
		}
	case "sock":
		if err := mainSock(); err != nil {
			panic(err)
		}
	case "nonblock":
		if err := mainNonblock(os.Args[2], os.Args[3:]); err != nil {
			panic(err)
		}
	}

	// Handle go-specific additions
	switch os.Args[1] {
	case "http":
		if err := mainHTTP(); err != nil {
			panic(err)
		}
	case "stdin":
		if err := mainStdin(); err != nil {
			panic(err)
		}
	case "stdout":
		mainStdout()
	case "largestdout":
		mainLargeStdout()
	}
}

func mainLs(path string, repeat bool) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()

	if err = printFileNames(d); err != nil {
		return err
	} else if repeat {
		// rewind
		if _, err = d.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return printFileNames(d)
	}
	return nil
}

func printFileNames(d *os.File) error {
	if names, err := d.Readdirnames(-1); err != nil {
		return err
	} else {
		for _, n := range names {
			fmt.Println("./" + n)
		}
	}
	return nil
}

func mainStat() error {
	var isatty = func(name string, fd uintptr) error {
		f := os.NewFile(fd, "")
		if st, err := f.Stat(); err != nil {
			return err
		} else {
			ttyMode := fs.ModeDevice | fs.ModeCharDevice
			isatty := st.Mode()&ttyMode == ttyMode
			fmt.Println(name, "isatty:", isatty)
			return nil
		}
	}

	for fd, name := range []string{"stdin", "stdout", "stderr", "/"} {
		if err := isatty(name, uintptr(fd)); err != nil {
			return err
		}
	}
	return nil
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

// Reproducer for https://github.com/tetratelabs/wazero/issues/1538
func mainStdin() error {
	go func() {
		time.Sleep(1 * time.Second)
		os.Stdout.WriteString("waiting for stdin...\n")
	}()

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	os.Stdout.Write(b)
	return nil
}

func mainStdout() {
	os.Stdout.WriteString("test")
}

func mainLargeStdout() {
	const ntest = 1024

	var decls, calls bytes.Buffer

	for i := 1; i <= ntest; i++ {
		s := strconv.Itoa(i)
		decls.WriteString(strings.Replace(decl, "$", s, -1))
		calls.WriteString(strings.Replace("call(test$)\n\t", "$", s, -1))
	}

	program = strings.Replace(program, "$DECLS", decls.String(), 1)
	program = strings.Replace(program, "$CALLS", calls.String(), 1)
	fmt.Print(program)
}

var program = `package main

var count int

func call(f func() bool) {
	if f() {
		count++
	}
}

$DECLS

func main() {
	$CALLS
	if count != 0 {
		println("failed", count, "case(s)")
	}
}
`

const decl = `
type T$ [$]uint8
func test$() bool {
	v := T${1}
	return v == [$]uint8{2} || v != [$]uint8{1}
}`
