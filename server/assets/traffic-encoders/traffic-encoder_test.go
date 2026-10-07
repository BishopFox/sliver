package traffic_encoders

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders/traffic"
	serverEncoders "github.com/bishopfox/sliver/util/encoders/traffic"
)

//go:embed hex.wasm
var hexWASM []byte

func TestTrafficEncoderCompatibility_hex(t *testing.T) {

	// Hex

	implantSideHex, err := implantEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = implantSideHex.Close()
	})
	serverSideHex, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = serverSideHex.Close()
	})

	data := make([]byte, 1024)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	encodedData, err := implantSideHex.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	decodedData, err := serverSideHex.Decode(encodedData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data does not match original")
	}

	data = make([]byte, 1024)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	encodedData, err = serverSideHex.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	decodedData, err = implantSideHex.Decode(encodedData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data does not match original")
	}
}

//nolint:gocyclo // This integration test intentionally exercises each network operation and shutdown path.
func TestTrafficEncoderCompatibility_Network(t *testing.T) {
	wasmPath := buildNetworkTrafficEncoder(t)
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatal(err)
	}

	tcpAddress := startTCPEcho(t)
	udpAddress := startUDPEcho(t)
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("https-traffic-encoder-ok"))
	}))
	t.Cleanup(tlsServer.Close)
	tlsRoots := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsServer.Certificate().Raw})

	logger := func(message string) {
		t.Log(message)
	}
	implantEncoder, err := implantEncoders.CreateTrafficEncoder("network-encoder", wasm, logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := implantEncoder.Close(); err != nil {
			t.Errorf("close implant encoder: %v", err)
		}
	})
	serverEncoder, err := serverEncoders.CreateTrafficEncoder("network-encoder", wasm, logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := serverEncoder.Close(); err != nil {
			t.Errorf("close server encoder: %v", err)
		}
	})

	for name, encoder := range map[string]interface {
		Encode([]byte) ([]byte, error)
		Decode([]byte) ([]byte, error)
		Close() error
	}{
		"implant": implantEncoder,
		"server":  serverEncoder,
	} {
		t.Run(name, func(t *testing.T) {
			tcpPayload := []byte("tcp-" + name)
			got, err := encoder.Encode(networkCommand("tcp", tcpAddress, tcpPayload))
			if err != nil {
				t.Fatal(err)
			}
			if want := bytes.ToUpper(tcpPayload); !bytes.Equal(got, want) {
				t.Fatalf("TCP result = %q, want %q", got, want)
			}

			udpPayload := []byte("udp-" + name)
			got, err = encoder.Decode(networkCommand("udp", udpAddress, udpPayload))
			if err != nil {
				t.Fatal(err)
			}
			if want := bytes.ToUpper(udpPayload); !bytes.Equal(got, want) {
				t.Fatalf("UDP result = %q, want %q", got, want)
			}

			got, err = encoder.Encode(networkCommand("lookup", "localhost", nil))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) == 0 || strings.HasPrefix(string(got), "error:") {
				t.Fatalf("DNS result = %q", got)
			}

			got, err = encoder.Encode(networkCommand("https", tlsServer.URL, tlsRoots))
			if err != nil {
				t.Fatal(err)
			}
			if want := []byte("https-traffic-encoder-ok"); !bytes.Equal(got, want) {
				t.Fatalf("HTTPS result = %q, want %q", got, want)
			}

			blockAddress, accepted := startTCPBlackhole(t)
			encodeDone := make(chan error, 1)
			go func() {
				_, encodeErr := encoder.Encode(networkCommand("block", blockAddress, nil))
				encodeDone <- encodeErr
			}()
			select {
			case <-accepted:
			case <-time.After(5 * time.Second):
				t.Fatal("network encoder did not connect to blocking TCP server")
			}

			closeDone := make(chan error, 1)
			go func() {
				closeDone <- encoder.Close()
			}()
			select {
			case closeErr := <-closeDone:
				if closeErr != nil {
					t.Fatalf("close blocked encoder: %v", closeErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("closing a network-blocked encoder timed out")
			}
			select {
			case <-encodeDone:
			case <-time.After(5 * time.Second):
				t.Fatal("blocked encoder call did not stop after Close")
			}
		})
	}
}

func TestTrafficEncoderRejectsInvalidFunctionSignature(t *testing.T) {
	wasmPath := buildTrafficEncoderFixture(t, "bad-signature-encoder")
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatal(err)
	}

	for name, create := range map[string]func() error{
		"implant": func() error {
			encoder, createErr := implantEncoders.CreateTrafficEncoder("bad-signature-encoder", wasm, func(message string) {
				t.Log(message)
			})
			if encoder != nil {
				_ = encoder.Close()
			}
			return createErr
		},
		"server": func() error {
			encoder, createErr := serverEncoders.CreateTrafficEncoder("bad-signature-encoder", wasm, func(message string) {
				t.Log(message)
			})
			if encoder != nil {
				_ = encoder.Close()
			}
			return createErr
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := create()
			if err == nil {
				t.Fatal("invalid free signature was accepted")
			}
			if !strings.Contains(err.Error(), "free") || !strings.Contains(err.Error(), "incompatible") {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestTrafficEncoderRejectsMissingMemory(t *testing.T) {
	wasm, err := hex.DecodeString(
		"0061736d01000000" +
			"01110360017f017f60027f7f0060027f7f017e" +
			"03050400010202" +
			"072304066d616c6c6f6300000466726565000106656e636f64650002066465636f64650003" +
			"0a1304040041000b02000b040042000b040042000b",
	)
	if err != nil {
		t.Fatal(err)
	}

	for name, create := range map[string]func() error{
		"implant": func() error {
			encoder, createErr := implantEncoders.CreateTrafficEncoder("missing-memory", wasm, func(message string) {
				t.Log(message)
			})
			if encoder != nil {
				_ = encoder.Close()
			}
			return createErr
		},
		"server": func() error {
			encoder, createErr := serverEncoders.CreateTrafficEncoder("missing-memory", wasm, func(message string) {
				t.Log(message)
			})
			if encoder != nil {
				_ = encoder.Close()
			}
			return createErr
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := create()
			if err == nil {
				t.Fatal("traffic encoder without memory was accepted")
			}
			if !strings.Contains(err.Error(), "memory") {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func networkCommand(operation, address string, payload []byte) []byte {
	command := operation + "\x00" + address
	if payload != nil {
		command += "\x00" + string(payload)
	}
	return []byte(command)
}

func startTCPEcho(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer func() {
					if closeErr := connection.Close(); closeErr != nil {
						t.Errorf("close TCP echo connection: %v", closeErr)
					}
				}()
				buffer := make([]byte, 1024)
				count, readErr := connection.Read(buffer)
				if readErr == nil {
					_, _ = connection.Write(bytes.ToUpper(buffer[:count]))
				}
			}()
		}
	}()
	return listener.Addr().String()
}

func startTCPBlackhole(t *testing.T) (string, <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan struct{})
	stop := make(chan struct{})
	t.Cleanup(func() {
		close(stop)
		_ = listener.Close()
	})
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() {
			if closeErr := connection.Close(); closeErr != nil {
				t.Errorf("close TCP blackhole connection: %v", closeErr)
			}
		}()
		close(accepted)
		<-stop
	}()
	return listener.Addr().String(), accepted
}

func startUDPEcho(t *testing.T) string {
	t.Helper()
	connection, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
	})
	go func() {
		buffer := make([]byte, 1024)
		for {
			count, address, readErr := connection.ReadFrom(buffer)
			if readErr != nil {
				return
			}
			_, _ = connection.WriteTo(bytes.ToUpper(buffer[:count]), address)
		}
	}()
	return connection.LocalAddr().String()
}

func buildNetworkTrafficEncoder(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("SLIVER_WASM_NETWORK_ENCODER_WASM"); path != "" {
		return path
	}
	return buildTrafficEncoderFixture(t, "network-encoder")
}

func buildTrafficEncoderFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	rootAppDir := os.Getenv("SLIVER_ROOT_DIR")
	if rootAppDir == "" {
		t.Skip("run through ./go-tests.sh with unpacked compiler assets or set SLIVER_WASM_NETWORK_ENCODER_WASM")
	}

	goName := "go"
	wrapperName := "sliver-wasm-go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
		wrapperName += ".exe"
	}
	bundledGo := filepath.Join(rootAppDir, "go", "bin", goName)
	if _, err := os.Stat(bundledGo); err != nil {
		t.Skipf("bundled Go toolchain is not available at %s: %v", bundledGo, err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve repository root")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	wrapperPath := filepath.Join(rootAppDir, "go", "bin", wrapperName)
	if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
		wrapperPattern := ".sliver-wasm-go-test-*"
		if runtime.GOOS == "windows" {
			wrapperPattern += ".exe"
		}
		wrapperFile, createErr := os.CreateTemp(filepath.Dir(wrapperPath), wrapperPattern)
		if createErr != nil {
			t.Fatalf("create temporary wrapper path: %v", createErr)
		}
		wrapperPath = wrapperFile.Name()
		if closeErr := wrapperFile.Close(); closeErr != nil {
			t.Fatalf("close temporary wrapper path: %v", closeErr)
		}
		t.Cleanup(func() {
			_ = os.Remove(wrapperPath)
		})
		command := exec.Command(
			bundledGo,
			"build",
			"-trimpath",
			"-buildvcs=false",
			"-mod=vendor",
			"-o",
			wrapperPath,
			"./util/cmd/sliver-wasm-go",
		)
		command.Dir = repositoryRoot
		command.Env = trafficEncoderTestEnvironment(
			"GOOS="+runtime.GOOS,
			"GOARCH="+runtime.GOARCH,
			"CGO_ENABLED=0",
			"GOTOOLCHAIN=local",
			"GOWORK=off",
			"GOFLAGS=",
		)
		output, buildErr := command.CombinedOutput()
		if buildErr != nil {
			t.Fatalf("build sliver-wasm-go: %v\n%s", buildErr, output)
		}
	} else if err != nil {
		t.Fatalf("inspect sliver-wasm-go: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), fixtureName+".wasm")
	command := exec.Command(
		wrapperPath,
		"build",
		"-buildmode=c-shared",
		"-trimpath",
		"-o",
		outputPath,
		"./server/assets/traffic-encoders/testdata/"+fixtureName,
	)
	command.Dir = repositoryRoot
	command.Env = trafficEncoderTestEnvironment("GOWORK=off", "GOFLAGS=-mod=vendor")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build network traffic encoder: %v\n%s", err, output)
	}
	return outputPath
}

func trafficEncoderTestEnvironment(overrides ...string) []string {
	overrideKeys := make([]string, 0, len(overrides))
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		overrideKeys = append(overrideKeys, key)
	}
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		replaced := false
		for _, overrideKey := range overrideKeys {
			if strings.EqualFold(key, overrideKey) {
				replaced = true
				break
			}
		}
		if !replaced {
			environment = append(environment, entry)
		}
	}
	return append(environment, overrides...)
}

// Encoder specific tests
func TestHexPerformance(t *testing.T) {

	sizes := []int{1024, 1024 * 1024, 2 * 1024 * 1024, 4 * 1024 * 1024}

	// Stock encoder
	for i := 0; i < len(sizes); i++ {
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		stock := time.Now()
		encodedValue := hex.EncodeToString(originalValue)
		decodedValue, err := hex.DecodeString(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Stock encoder took %v (%d bytes)", time.Since(stock), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}

	// Traffic encoder
	for i := 0; i < len(sizes); i++ {
		encoder, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
			t.Log(msg)
		})
		if err != nil {
			t.Fatal(err)
		}
		defer encoder.Close()
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		start := time.Now()
		encodedValue, err := encoder.Encode(originalValue)
		if err != nil {
			t.Fatal(err)
		}
		// t.Logf("Got encoded value (%d bytes)", len(encodedValue))
		decodedValue, err := encoder.Decode(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("WASM Hex encoder took %v (%d bytes)", time.Since(start), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}
}
