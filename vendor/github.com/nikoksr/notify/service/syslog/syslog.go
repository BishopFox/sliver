package syslog

import (
	"context"
	"fmt"
	"io"
	"log/syslog"
	"strings"
)

// mockSyslogWriter abstracts log/syslog for writing unit tests.
type syslogWriter interface {
	io.WriteCloser
}

// Service encapsulates a syslog daemon writer.
type Service struct {
	writer syslogWriter
}

// dial is a wrapper function around syslog.Dial. It normalizes the prefix tag in case that it's empty and returns
// a new Service with the writer field set to writer from the call to syslog.Dial.
func dial(network, raddr string, priority syslog.Priority, tag string) (*Service, error) {
	if strings.TrimSpace(tag) == "" {
		tag = "notify"
	}

	// Usually we could call syslog.New and syslog.Dial respectively for specific use-cases. But since syslog.New is
	// only a wrapper around a call to syslog.Dial without information about the network we're doing the same here to
	// keep the API a little more clean.
	writer, err := syslog.Dial(network, raddr, priority, tag)
	if err != nil {
		return nil, err
	}

	return &Service{writer: writer}, nil
}

// New returns a new instance of a Service notification service. Parameter 'tag' is used as a log prefix and may be left
// empty, it has a fallback value.
func New(priority syslog.Priority, tag string) (*Service, error) {
	return dial("", "", priority, tag)
}

// NewFromDial returns a new instance of a Service notification service. The underlying syslog writer establishes a
// connection to a log daemon by connecting to address raddr on the specified network. Parameter 'tag' is used as a log
// prefix and may be left empty, it has a fallback value.
// Calling NewFromDial with network and raddr being empty strings is equal in function to calling New directly.
func NewFromDial(network, raddr string, priority syslog.Priority, tag string) (*Service, error) {
	return dial(network, raddr, priority, tag)
}

// Close the underlying syslog writer.
func (s *Service) Close() error {
	return s.writer.Close()
}

// Send takes a message subject and a message body and sends them to all previously set channels.
// user used for sending the message has to be a member of the channel.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err := s.writer.Write([]byte(subject + ": " + message))
		if err != nil {
			return fmt.Errorf("write to syslog: %w", err)
		}
	}

	return nil
}
