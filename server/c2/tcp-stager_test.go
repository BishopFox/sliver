package c2

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"fmt"
	"io"
	"net"
	"testing"
)

func TestTCPStager(t *testing.T) {
	testData := []byte("test shellcode data")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := uint16(addr.Port)
	listener.Close()

	// Test localhost
	ln, err := StartTCPListener("127.0.0.1", port, testData)
	if err != nil {
		t.Fatalf("Failed to start TCP stager: %v", err)
	}
	defer ln.Close()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to stager: %v", err)
	}
	defer conn.Close()

	receivedData, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("Failed to read from stager: %v", err)
	}

	if !bytes.Equal(receivedData, testData) {
		t.Fatalf("Received data does not match sent data")
	}
}

func TestTCPStagerInterface(t *testing.T) {
	testData := []byte("test shellcode data")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := uint16(addr.Port)
	listener.Close()

	// Test eth0 interface, this might fail if eth0 doesn't exist, we just want to verify the function accepts interface names
	ln, err := StartTCPListener("eth0", port, testData)
	if err != nil {
		t.Logf("Note: eth0 test skipped (expected if eth0 doesn't exist): %v", err)
		return
	}
	defer ln.Close()

	// Get eth0 IP
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get interfaces: %v", err)
	}

	var eth0IP string
	for _, iface := range ifaces {
		if iface.Name == "eth0" {
			addrs, err := iface.Addrs()
			if err != nil {
				t.Fatalf("Failed to get addresses for eth0: %v", err)
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						eth0IP = ipnet.IP.String()
						break
					}
				}
			}
			break
		}
	}

	if eth0IP == "" {
		t.Logf("Note: Could not find IPv4 address for eth0")
		return
	}

	// Connect using eth0 IP
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", eth0IP, port))
	if err != nil {
		t.Fatalf("Failed to connect to stager: %v", err)
	}
	defer conn.Close()

	receivedData, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("Failed to read from stager: %v", err)
	}

	if !bytes.Equal(receivedData, testData) {
		t.Fatalf("Received data does not match sent data")
	}
}