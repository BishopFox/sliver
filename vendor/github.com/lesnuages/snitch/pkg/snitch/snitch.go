package snitch

import (
	"time"
)

// Sample maps an implant name with its MD5 hash
type Sample struct {
	implantName string
	hash        string
}

func (n *Sample) Name() string {
	return n.implantName
}

// Snitch -- the Snitch struct
type Snitch struct {
	scanners      map[string]Scanner
	samples       chan Sample
	stop          chan bool
	scanResults   chan *ScanResult
	HandleFlagged func(*ScanResult)
}

// Scanner is the abstract representation of a malware scanner
type Scanner interface {
	// Add adds a sample to the scanning list
	Add(Sample)
	// Remove deletes a sample from the list
	Remove(Sample)
	// Threshold is the time we need to sleep before each batch of requests
	Threshold() time.Duration
	// MaxRequests represents the maximum number of requests we can make
	// before going to sleep
	MaxRequests() int
	// Scan performs the request to the API
	Scan(Sample) (*ScanResult, error)
	// Name of the scanner
	Name() string
	// Start starts the scanning loop
	Start(chan *ScanResult)
	// Stop stops a scan loop
	Stop()
}

// ScanResult stores a scan result
type ScanResult struct {
	Sample   Sample
	Provider string
	LastSeen time.Time
}

// NewSnitch returns a new Snitch instance
func NewSnitch() *Snitch {
	return &Snitch{
		scanners:    make(map[string]Scanner),
		samples:     make(chan Sample),
		stop:        make(chan bool),
		scanResults: make(chan *ScanResult),
	}
}

func WithHandleFlagged(handleFlagged func(*ScanResult)) *Snitch {
	s := NewSnitch()
	s.HandleFlagged = handleFlagged
	return s
}

func (s *Snitch) AddScanner(scanner Scanner) {
	s.scanners[scanner.Name()] = scanner
}

// Start kicks off the regular scans
func (s *Snitch) Start() {
	go s.start()
}

func (s *Snitch) start() {
	// Start all the registered scanners
	for _, sc := range s.scanners {
		go sc.Start(s.scanResults)
	}
	for {
		select {
		case sample := <-s.samples:
			// Add new samples to the scanners' list
			for _, sc := range s.scanners {
				sc.Add(sample)
			}
		case result := <-s.scanResults:
			// Pass the results to the caller
			go s.HandleFlagged(result)
		case <-s.stop:
			// Stop the scan loops
			for _, sc := range s.scanners {
				sc.Stop()
			}
			return
		}
	}
}

// Stop stops the scanning loop
func (s *Snitch) Stop() {
	s.stop <- true
}

// Add adds a hash to the monitored list
func (s *Snitch) Add(name string, hash string) {
	s.samples <- Sample{implantName: name, hash: hash}
}
