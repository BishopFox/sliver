package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const networkTimeout = 10 * time.Second

func main() {
	if err := tcpRoundTrip(); err != nil {
		fmt.Fprintf(os.Stderr, "TCP round trip: %v\n", err)
		os.Exit(1)
	}
	if err := udpRoundTrip(); err != nil {
		fmt.Fprintf(os.Stderr, "UDP round trip: %v\n", err)
		os.Exit(1)
	}
	if err := closeRace(); err != nil {
		fmt.Fprintf(os.Stderr, "close race: %v\n", err)
		os.Exit(1)
	}
	if addresses, err := net.LookupHost("localhost"); err != nil || len(addresses) == 0 {
		fmt.Fprintf(os.Stderr, "lookup localhost: addresses=%v error=%v\n", addresses, err)
		os.Exit(1)
	}
	if _, err := net.LookupHost("invalid host name"); err == nil {
		fmt.Fprintln(os.Stderr, "invalid hostname lookup unexpectedly succeeded")
		os.Exit(1)
	} else {
		var dnsError *net.DNSError
		if !errors.As(err, &dnsError) || !dnsError.IsNotFound {
			fmt.Fprintf(os.Stderr, "invalid hostname lookup error = %#v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("network-smoke-ok")
}

func tcpRoundTrip() error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	serverResult := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- acceptErr
			return
		}
		defer connection.Close()
		if deadlineErr := connection.SetDeadline(time.Now().Add(networkTimeout)); deadlineErr != nil {
			serverResult <- deadlineErr
			return
		}
		request := make([]byte, 4)
		if _, readErr := io.ReadFull(connection, request); readErr != nil {
			serverResult <- readErr
			return
		}
		if !bytes.Equal(request, []byte("ping")) {
			serverResult <- fmt.Errorf("request = %q", request)
			return
		}
		_, writeErr := connection.Write([]byte("pong"))
		serverResult <- writeErr
	}()

	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), networkTimeout)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(networkTimeout)); err != nil {
		return err
	}
	if _, err := connection.Write([]byte("ping")); err != nil {
		return err
	}
	response := make([]byte, 4)
	if _, err := io.ReadFull(connection, response); err != nil {
		return err
	}
	if !bytes.Equal(response, []byte("pong")) {
		return fmt.Errorf("response = %q", response)
	}
	return <-serverResult
}

func closeRace() error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	acceptStarted := make(chan struct{})
	acceptResult := make(chan error, 1)
	go func() {
		close(acceptStarted)
		_, acceptErr := listener.Accept()
		acceptResult <- acceptErr
	}()
	<-acceptStarted
	time.Sleep(10 * time.Millisecond)
	if err := listener.Close(); err != nil {
		return err
	}
	select {
	case err := <-acceptResult:
		if !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("Accept error = %v, want net.ErrClosed", err)
		}
		return nil
	case <-time.After(networkTimeout):
		return errors.New("Accept remained blocked after Close")
	}
}

func udpRoundTrip() error {
	server, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer server.Close()
	client, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer client.Close()

	deadline := time.Now().Add(networkTimeout)
	if err := server.SetDeadline(deadline); err != nil {
		return err
	}
	if err := client.SetDeadline(deadline); err != nil {
		return err
	}
	if _, err := client.WriteTo([]byte("ping"), server.LocalAddr()); err != nil {
		return err
	}
	request := make([]byte, 4)
	n, clientAddress, err := server.ReadFrom(request)
	if err != nil {
		return err
	}
	if !bytes.Equal(request[:n], []byte("ping")) {
		return fmt.Errorf("request = %q", request[:n])
	}
	if _, err := server.WriteTo([]byte("pong"), clientAddress); err != nil {
		return err
	}
	response := make([]byte, 4)
	n, _, err = client.ReadFrom(response)
	if err != nil {
		return err
	}
	if !bytes.Equal(response[:n], []byte("pong")) {
		return fmt.Errorf("response = %q", response[:n])
	}
	return nil
}
