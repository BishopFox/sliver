//go:build windows || darwin || linux

package transports

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

type recordingWGBeaconDevice struct {
	closeCalls int
}

func (d *recordingWGBeaconDevice) Close() {
	d.closeCalls++
}

func TestCloseWGBeaconDeviceUsesPermanentClose(t *testing.T) {
	dev := &recordingWGBeaconDevice{}
	closeWGBeaconDevice(dev.Close)
	if dev.closeCalls != 1 {
		t.Fatalf("wireguard device Close called %d times, want 1", dev.closeCalls)
	}
}

func TestSendWGBeaconStreamWaitsForRemoteClose(t *testing.T) {
	client, server := newBeaconYamuxPair(t)
	marker := []byte("beacon-result")
	serverRead := make(chan struct{})
	releaseServer := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseServer) }) }
	t.Cleanup(release)
	serverErr := make(chan error, 1)

	go func() {
		stream, err := server.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = stream.Close() }()
		got := make([]byte, len(marker))
		if _, err := io.ReadFull(stream, got); err != nil {
			serverErr <- err
			return
		}
		if string(got) != string(marker) {
			serverErr <- errors.New("server received an unexpected payload")
			return
		}
		close(serverRead)
		<-releaseServer
		serverErr <- nil
	}()

	sendErr := make(chan error, 1)
	go func() {
		sendErr <- sendWGBeaconStream(client, time.Second, func(stream net.Conn) error {
			_, err := stream.Write(marker)
			return err
		})
	}()

	select {
	case err := <-serverErr:
		t.Fatalf("server failed before reading the request: %v", err)
	case err := <-sendErr:
		t.Fatalf("send returned before the server read the request: %v", err)
	case <-serverRead:
	case <-time.After(time.Second):
		t.Fatal("server did not receive the request")
	}

	select {
	case err := <-sendErr:
		t.Fatalf("send returned before the server closed its stream: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	select {
	case err := <-sendErr:
		if err != nil {
			t.Fatalf("send failed after the server closed its stream: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("send did not observe the server stream close")
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestSendWGBeaconStreamReceiptTimeout(t *testing.T) {
	client, server := newBeaconYamuxPair(t)
	marker := []byte("beacon-result")
	releaseServer := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseServer) }) }
	t.Cleanup(release)
	serverErr := make(chan error, 1)

	go func() {
		stream, err := server.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = stream.Close() }()
		got := make([]byte, len(marker))
		if _, err := io.ReadFull(stream, got); err != nil {
			serverErr <- err
			return
		}
		<-releaseServer
		serverErr <- nil
	}()

	err := sendWGBeaconStream(client, 50*time.Millisecond, func(stream net.Conn) error {
		_, err := stream.Write(marker)
		return err
	})
	if !errors.Is(err, yamux.ErrTimeout) {
		t.Fatalf("got receipt error %v, want %v", err, yamux.ErrTimeout)
	}
	release()
	if err := <-serverErr; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func newBeaconYamuxPair(t *testing.T) (*yamux.Session, *yamux.Session) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	config := yamux.DefaultConfig()
	config.EnableKeepAlive = false
	config.LogOutput = io.Discard
	server, err := yamux.Server(serverConn, config)
	if err != nil {
		t.Fatal(err)
	}
	client, err := yamux.Client(clientConn, config)
	if err != nil {
		_ = server.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return client, server
}
