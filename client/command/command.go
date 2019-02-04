package command

import (
	pb "sliver/protobuf/client"
	"time"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

var (
	defaultTimeout = 30 * time.Second
)

type observer func()

type activeSliver struct {
	Sliver    *pb.Sliver
	observers []observer
}

func (s *activeSliver) AddObserver(fn observer) {
	s.observers = append(s.observers, fn)
}

func (s *activeSliver) SetActiveSliver(sliver *pb.Sliver) {
	s.Sliver = sliver
	for _, fn := range s.observers {
		fn()
	}
}

var (
	// ActiveSliver - The current sliver we're interacting with
	ActiveSliver = &activeSliver{
		observers: []observer{},
	}
)

// RPCServer - Function used to send/recv envelopes
type RPCServer func(*pb.Envelope, time.Duration) *pb.Envelope
