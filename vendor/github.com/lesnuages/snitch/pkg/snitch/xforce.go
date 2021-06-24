package snitch

import (
	"container/ring"
	"sync"
	"time"

	"github.com/demisto/goxforce"
)

// XForceScanner is an implentation of Scanner
// for the IBM X-Force threat intel platform
type XForceScanner struct {
	APIKey      string
	APIPassword string
	threshold   int
	Provider    string
	mutex       sync.Mutex
	stop        chan bool
	scanList    *ring.Ring
	ticker      *time.Ticker
}

const XForceMaxRequests = 6

// NewXForceScanner returns a new XForceScanner instance
func NewXForceScanner(apiKey string, password string, maxRequests int, name string) *XForceScanner {
	s := &XForceScanner{
		APIKey:      apiKey,
		APIPassword: password,
		Provider:    name,
		threshold:   maxRequests,
		stop:        make(chan bool),
		scanList:    ring.New(1),
	}
	s.ticker = time.NewTicker(s.Threshold())
	return s
}

func (s *XForceScanner) Name() string {
	return s.Provider
}

// Threshold returns the threshold value
// IBM X-Force API free tier limit is around 6 requests per hour (5000/month ~= 6.97/hour)
func (s *XForceScanner) Threshold() time.Duration {
	return 10 * time.Minute
}

// MaxRequests represents the maximum number of requests that we can make in one minute
func (s *XForceScanner) MaxRequests() int {
	return s.threshold
}

// Add adds a sample to the list
func (s *XForceScanner) Add(samp Sample) {
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

// Mutex --
func (s *XForceScanner) Mutex() *sync.Mutex {
	return &s.mutex
}

// Scan runs a scans on a hash
func (s *XForceScanner) Scan(samp Sample) (*ScanResult, error) {
	res := new(ScanResult)
	client, err := goxforce.New(goxforce.SetCredentials(s.APIKey, s.APIPassword))
	if err != nil {
		return nil, err
	}
	details, err := client.MalwareDetails(samp.hash)
	if err != nil {
		return nil, err
	}
	res.Sample = samp
	res.LastSeen = details.Malware.Created
	res.Provider = s.Provider
	return res, nil
}

func (s *XForceScanner) Start(results chan *ScanResult) {
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

func (s *XForceScanner) Stop() {
	s.ticker.Stop()
	s.stop <- true
}

// Remove deletes a sample from the scanning list
func (s *XForceScanner) Remove(sample Sample) {
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
