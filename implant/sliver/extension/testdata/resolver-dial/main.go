package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const hostname = "sliver-resolver-dial.invalid."

func main() {
	for _, preferGo := range []bool{false, true} {
		if err := checkResolver(preferGo); err != nil {
			fmt.Fprintf(os.Stderr, "PreferGo=%t: %v\n", preferGo, err)
			os.Exit(1)
		}
	}
	fmt.Println("resolver-dial-ok")
}

func checkResolver(preferGo bool) error {
	var calls atomic.Int32
	resolver := &net.Resolver{
		PreferGo: preferGo,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			calls.Add(1)
			client, server := net.Pipe()
			deadline, _ := ctx.Deadline()
			_ = server.SetDeadline(deadline)
			go func() {
				defer server.Close()
				_ = respond(server)
			}()
			return client, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addresses, err := resolver.LookupHost(ctx, hostname)
	if err != nil {
		return fmt.Errorf("LookupHost: %w", err)
	}
	if !slices.Contains(addresses, "192.0.2.10") || !slices.Contains(addresses, "2001:db8::10") {
		return fmt.Errorf("LookupHost addresses = %v", addresses)
	}
	for _, query := range []struct{ network, want string }{
		{"ip4", "192.0.2.10"},
		{"ip6", "2001:db8::10"},
	} {
		addresses, err := resolver.LookupIP(ctx, query.network, hostname)
		if err != nil {
			return fmt.Errorf("LookupIP %s: %w", query.network, err)
		}
		if len(addresses) != 1 || addresses[0].String() != query.want {
			return fmt.Errorf("LookupIP %s addresses = %v, want %s", query.network, addresses, query.want)
		}
	}
	if calls.Load() < 4 {
		return fmt.Errorf("custom Dial called %d times, want at least 4", calls.Load())
	}
	return nil
}

// net.Pipe is a stream connection, so the resolver uses DNS-over-TCP framing.
func respond(conn net.Conn) error {
	var size [2]byte
	if _, err := io.ReadFull(conn, size[:]); err != nil {
		return err
	}
	query := make([]byte, binary.BigEndian.Uint16(size[:]))
	if _, err := io.ReadFull(conn, query); err != nil {
		return err
	}
	var message dnsmessage.Message
	if err := message.Unpack(query); err != nil {
		return err
	}
	message.Header.Response = true
	message.Header.RecursionAvailable = true
	for _, question := range message.Questions {
		answer := dnsmessage.Resource{Header: dnsmessage.ResourceHeader{
			Name: question.Name, Type: question.Type, Class: dnsmessage.ClassINET, TTL: 60,
		}}
		switch question.Type {
		case dnsmessage.TypeA:
			answer.Body = &dnsmessage.AResource{A: [4]byte{192, 0, 2, 10}}
		case dnsmessage.TypeAAAA:
			answer.Body = &dnsmessage.AAAAResource{AAAA: [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10}}
		default:
			continue
		}
		message.Answers = append(message.Answers, answer)
	}
	response, err := message.Pack()
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint16(size[:], uint16(len(response)))
	_, err = conn.Write(append(size[:], response...))
	return err
}
