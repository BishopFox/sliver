//go:build windows || darwin || linux

package wireguard

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
)

type ownershipTestDevice struct {
	upErr        error
	configureErr error
	upCalls      int
	closeCalls   int
	configured   string
}

func (d *ownershipTestDevice) Up() error {
	d.upCalls++
	return d.upErr
}

func (d *ownershipTestDevice) Close() {
	d.closeCalls++
}

func (d *ownershipTestDevice) IpcSetOperation(config io.Reader) error {
	data, err := io.ReadAll(config)
	if err != nil {
		return err
	}
	d.configured = string(data)
	return d.configureErr
}

type ownershipTestDialer struct {
	dial      func(network string, address string) (net.Conn, error)
	dialCalls int
	network   string
	address   string
}

func (d *ownershipTestDialer) Dial(network string, address string) (net.Conn, error) {
	d.dialCalls++
	d.network = network
	d.address = address
	return d.dial(network, address)
}

func TestGetSessKeysWithDeviceClosesTemporaryDevice(t *testing.T) {
	originalPrivate, originalPublic, originalAddress := wgSessPrivKey, wgSessPubKey, tunAddress
	t.Cleanup(func() {
		wgSessPrivKey, wgSessPubKey, tunAddress = originalPrivate, originalPublic, originalAddress
	})

	privateKey := strings.Repeat("a", 64)
	publicKey := strings.Repeat("b", 64)
	dialer := &ownershipTestDialer{dial: func(string, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			_, _ = server.Write([]byte(privateKey + "|" + publicKey + "|100.64.0.42"))
			_ = server.Close()
		}()
		return client, nil
	}}
	dev := &ownershipTestDevice{}

	if err := getSessKeysWithDevice(dev, dialer); err != nil {
		t.Fatal(err)
	}
	if dev.upCalls != 1 || dev.closeCalls != 1 {
		t.Fatalf("temporary device calls: Up=%d Close=%d, want 1 each", dev.upCalls, dev.closeCalls)
	}
	if wgSessPrivKey != privateKey || wgSessPubKey != publicKey || tunAddress != "100.64.0.42" {
		t.Fatalf("unexpected session keys: private=%q public=%q address=%q", wgSessPrivKey, wgSessPubKey, tunAddress)
	}
}

func TestGetSessKeysWithDeviceClosesTemporaryDeviceOnErrors(t *testing.T) {
	originalPrivate, originalPublic, originalAddress := wgSessPrivKey, wgSessPubKey, tunAddress
	t.Cleanup(func() {
		wgSessPrivKey, wgSessPubKey, tunAddress = originalPrivate, originalPublic, originalAddress
	})

	upErr := errors.New("up failed")
	dialErr := errors.New("dial failed")
	tests := []struct {
		name       string
		dev        *ownershipTestDevice
		dial       func(string, string) (net.Conn, error)
		wantErr    error
		wantDialed bool
	}{
		{
			name:    "device up",
			dev:     &ownershipTestDevice{upErr: upErr},
			dial:    func(string, string) (net.Conn, error) { return nil, errors.New("unexpected dial") },
			wantErr: upErr,
		},
		{
			name:       "key exchange dial",
			dev:        &ownershipTestDevice{},
			dial:       func(string, string) (net.Conn, error) { return nil, dialErr },
			wantErr:    dialErr,
			wantDialed: true,
		},
		{
			name: "key exchange response",
			dev:  &ownershipTestDevice{},
			dial: func(string, string) (net.Conn, error) {
				client, server := net.Pipe()
				go func() {
					_, _ = server.Write([]byte("malformed"))
					_ = server.Close()
				}()
				return client, nil
			},
			wantDialed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dialer := &ownershipTestDialer{dial: test.dial}
			err := getSessKeysWithDevice(test.dev, dialer)
			if err == nil {
				t.Fatal("expected key exchange to fail")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
			if test.dev.closeCalls != 1 {
				t.Fatalf("temporary device Close called %d times, want 1", test.dev.closeCalls)
			}
			if got := dialer.dialCalls > 0; got != test.wantDialed {
				t.Fatalf("dial called=%v, want %v", got, test.wantDialed)
			}
		})
	}
}

func TestDialDeviceTransfersOrReleasesOwnership(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, server := net.Pipe()
		defer server.Close()
		dev := &ownershipTestDevice{}
		dialer := &ownershipTestDialer{dial: func(string, string) (net.Conn, error) { return client, nil }}
		connection, err := dialDevice(dialer, dev, "tcp", "100.64.0.1:8888")
		if err != nil {
			t.Fatal(err)
		}
		defer connection.Close()
		if dev.closeCalls != 0 {
			t.Fatalf("successful dial closed returned device %d times", dev.closeCalls)
		}
	})

	t.Run("failure", func(t *testing.T) {
		wantErr := errors.New("dial failed")
		dev := &ownershipTestDevice{}
		dialer := &ownershipTestDialer{dial: func(string, string) (net.Conn, error) { return nil, wantErr }}
		connection, err := dialDevice(dialer, dev, "tcp", "100.64.0.1:8888")
		if connection != nil || !errors.Is(err, wantErr) {
			t.Fatalf("got connection=%v error=%v, want nil and %v", connection, err, wantErr)
		}
		if dev.closeCalls != 1 {
			t.Fatalf("failed dial closed device %d times, want 1", dev.closeCalls)
		}
	})
}

func TestConfigureDeviceTransfersOrReleasesOwnership(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dev := &ownershipTestDevice{}
		if err := configureDevice(dev, bytes.NewBufferString("private_key=test\n")); err != nil {
			t.Fatal(err)
		}
		if dev.closeCalls != 0 {
			t.Fatalf("successful configuration closed returned device %d times", dev.closeCalls)
		}
		if dev.configured != "private_key=test\n" {
			t.Fatalf("unexpected configuration %q", dev.configured)
		}
	})

	t.Run("failure", func(t *testing.T) {
		wantErr := errors.New("configuration failed")
		dev := &ownershipTestDevice{configureErr: wantErr}
		err := configureDevice(dev, bytes.NewBufferString("private_key=test\n"))
		if !errors.Is(err, wantErr) {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
		if dev.closeCalls != 1 {
			t.Fatalf("failed configuration closed device %d times, want 1", dev.closeCalls)
		}
	})
}
