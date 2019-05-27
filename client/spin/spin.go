package spin

// Adapted from https://github.com/tj/go-spin

import (
	"fmt"
	"sync"
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
)

// Spinner types.
var (
	Box1    = `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
	Box2    = `⠋⠙⠚⠞⠖⠦⠴⠲⠳⠓`
	Box3    = `⠄⠆⠇⠋⠙⠸⠰⠠⠰⠸⠙⠋⠇⠆`
	Box4    = `⠋⠙⠚⠒⠂⠂⠒⠲⠴⠦⠖⠒⠐⠐⠒⠓⠋`
	Box5    = `⠁⠉⠙⠚⠒⠂⠂⠒⠲⠴⠤⠄⠄⠤⠴⠲⠒⠂⠂⠒⠚⠙⠉⠁`
	Box6    = `⠈⠉⠋⠓⠒⠐⠐⠒⠖⠦⠤⠠⠠⠤⠦⠖⠒⠐⠐⠒⠓⠋⠉⠈`
	Box7    = `⠁⠁⠉⠙⠚⠒⠂⠂⠒⠲⠴⠤⠄⠄⠤⠠⠠⠤⠦⠖⠒⠐⠐⠒⠓⠋⠉⠈⠈`
	Spin1   = `|/-\`
	Spin2   = `◴◷◶◵`
	Spin3   = `◰◳◲◱`
	Spin4   = `◐◓◑◒`
	Spin5   = `▉▊▋▌▍▎▏▎▍▌▋▊▉`
	Spin6   = `▌▄▐▀`
	Spin7   = `╫╪`
	Spin8   = `■□▪▫`
	Spin9   = `←↑→↓`
	Default = Box1
)

// Spinner is exactly what you think it is.
type Spinner struct {
	mu     sync.Mutex
	frames []rune
	length int
	pos    int
}

// New returns a spinner initialized with Default frames.
func New() *Spinner {
	s := &Spinner{}
	s.Set(Default)
	return s
}

// Set frames to the given string which must not use spaces.
func (s *Spinner) Set(frames string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = []rune(frames)
	s.length = len(s.frames)
}

// Current returns the current rune in the sequence.
func (s *Spinner) Current() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.frames[s.pos%s.length]
	return string(r)
}

// Next returns the next rune in the sequence.
func (s *Spinner) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.frames[s.pos%s.length]
	s.pos++
	return string(r)
}

// Reset the spinner to its initial frame.
func (s *Spinner) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pos = 0
}

// Until - Spint until ctrl channel signals
func Until(msg string, ctrl chan bool) {
	defer close(ctrl)
	s := New()
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			fmt.Printf(clearln+" %s  %s", s.Next(), msg)
		case <-ctrl:
			fmt.Printf(clearln)
			ctrl <- true
			return
		}
	}
}
