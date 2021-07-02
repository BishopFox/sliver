package snitch

import (
	"container/ring"
	"sync"
	"time"

	"github.com/VirusTotal/vt-go"
)

// VTScanner is an implentation of Scanner
// for the Virus Total threat intel platform
type VTScanner struct {
	APIKey    string
	threshold int
	Provider  string
	ticker    *time.Ticker
	mutex     sync.Mutex
	scanList  *ring.Ring
	stop      chan bool
}

const (
	VTMaxRequests      = 4
	VTDefaultThreshold = 12 * time.Minute
)

// NewVTScanner returns a new instance of VTScanner
func NewVTScanner(apiKey string, maxRequests int, name string) *VTScanner {
	s := &VTScanner{
		APIKey:    apiKey,
		threshold: maxRequests,
		Provider:  name,
		stop:      make(chan bool),
		scanList:  ring.New(1),
	}
	s.ticker = time.NewTicker(s.Threshold())
	return s
}

// Add adds a sample to the list
func (s *VTScanner) Add(samp Sample) {
	s.mutex.Lock()
	// We can't create a 0 lenght ring
	// so the first time a sample is added,
	// the s.scanList.Value will be a nil pointer
	if s.scanList.Value == nil {
		s.scanList.Value = samp
	} else {
		newRing := ring.New(1)
		newRing.Value = samp
		// No need to call Next() here, as Link() does it for us
		s.scanList.Link(newRing)
	}
	s.mutex.Unlock()
}

func (s *VTScanner) Name() string {
	return s.Provider
}

// Threshold returns the threshold value
// Virus Total free tier limit is 4 requests per minute, but 500 requests/day.
func (s *VTScanner) Threshold() time.Duration {
	return VTDefaultThreshold
}

// MaxRequests represents the maximum number of requests that we can make in one minute
func (s *VTScanner) MaxRequests() int {
	return s.threshold
}

// Scan checks a hash against the Virus Total platform records
func (s *VTScanner) Scan(samp Sample) (*ScanResult, error) {
	client := vt.NewClient(s.APIKey)
	object, err := client.GetObject(vt.URL("files/%s", samp.hash))
	if err != nil {
		return nil, err
	}
	last, err := object.GetTime("last_submission_date")
	if err != nil {
		return nil, err
	}
	return &ScanResult{
		Sample:   samp,
		LastSeen: last,
		Provider: s.Provider,
	}, nil
}

func (s *VTScanner) Start(results chan *ScanResult) {
	for {
		select {
		case <-s.stop:
			return
		case <-s.ticker.C:
			limit := s.MaxRequests()
			s.mutex.Lock()
			scanListLen := s.scanList.Len()
			s.mutex.Unlock()
			// We want to maximise the number of request we send
			// at each tick, but we don't want to waste requests either.
			// If we have less hashes to scan than the API rate limit,
			// we only scan what we have once per tick.
			if scanListLen < limit {
				limit = scanListLen
			}
			for i := 0; i < limit; i++ {
				s.mutex.Lock()
				val := s.scanList.Value
				s.scanList = s.scanList.Next()
				s.mutex.Unlock()
				if val == nil {
					continue
				}
				sample := val.(Sample)
				r, err := s.Scan(sample)
				if err != nil {
					continue
				}
				results <- r
				s.Remove(sample)
			}
		}
	}
}

func (s *VTScanner) Stop() {
	s.ticker.Stop()
	s.stop <- true
}

// Remove deletes a sample from the scanning list
func (s *VTScanner) Remove(sample Sample) {
	s.mutex.Lock()
	for i := 0; i < s.scanList.Len(); i++ {
		r := s.scanList.Value
		if r != nil && r.(Sample).hash == sample.hash {
			s.scanList = s.scanList.Prev()
			s.scanList.Unlink(1)
		}
		s.scanList = s.scanList.Next()
	}
	s.mutex.Unlock()
}
