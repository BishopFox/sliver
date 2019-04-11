package command

import (
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
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

	// ActiveSliver - The current sliver we're interacting with
	ActiveSliver = &activeSliver{
		observers: []observer{},
	}

	defaultTimeout   = 30 * time.Second
	stdinReadTimeout = 10 * time.Millisecond
)

// RPCServer - Function used to send/recv envelopes
type RPCServer func(*sliverpb.Envelope, time.Duration) chan *sliverpb.Envelope

type observer func()

type activeSliver struct {
	Sliver    *clientpb.Sliver
	observers []observer
}

func (s *activeSliver) AddObserver(fn observer) {
	s.observers = append(s.observers, fn)
}

func (s *activeSliver) SetActiveSliver(sliver *clientpb.Sliver) {
	s.Sliver = sliver
	for _, fn := range s.observers {
		fn()
	}
}

func (s *activeSliver) DisableActiveSliver() {
	s.Sliver = nil
	for _, fn := range s.observers {
		fn()
	}
}

// Get Sliver by session ID or name
func getSliver(arg string, rpc RPCServer) *clientpb.Sliver {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgSessions,
		Data: []byte{},
	}, defaultTimeout)
	sessions := &clientpb.Sessions{}
	proto.Unmarshal((resp).Data, sessions)

	for _, sliver := range sessions.Slivers {
		if strconv.Itoa(int(sliver.ID)) == arg || sliver.Name == arg {
			return sliver
		}
	}
	return nil
}
