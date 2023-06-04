package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	switch os.Args[1] {
	case "sock":
		if err := mainSock(); err != nil {
			panic(err)
		}
	}
}

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
