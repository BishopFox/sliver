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
	insecureRand "math/rand"
	"net/url"
	"strconv"
	"time"
)

var (
	ccServers = []string{
		// {{range $index, $value := .Config.C2}}
		"{{$value}}", // {{$index}}
		// {{end}}
	}

	maxErrors         = GetMaxConnectionErrors()
	reconnectInterval = time.Duration(0)
	pollTimeout       = time.Duration(0)

	ccCounter = new(int)

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

func nextCCServer() *url.URL {
	var next string
	switch "{{.Config.ConnectionStrategy}}" {
	case "r": // Random
		next = ccServers[insecureRand.Intn(len(ccServers))]
	case "rd": // Random Domain
		next = randomCCDomain(ccServers[*ccCounter%len(ccServers)])
	case "s": // Sequential
		next = ccServers[*ccCounter%len(ccServers)]
	default:
		next = ccServers[*ccCounter%len(ccServers)]
	}
	*ccCounter++
	uri, err := url.Parse(next)
	if err != nil {
		return nextCCServer()
	}
	return uri
}

// randomCCDomain - Random selection within a protocol
func randomCCDomain(next string) string {
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
		reconnect, err := strconv.Atoi(`{{.Config.ReconnectInterval}}`)
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
		poll, err := strconv.Atoi(`{{.Config.PollTimeout}}`)
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
