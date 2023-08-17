package client

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"os/user"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/version"
	"github.com/sirupsen/logrus"
)

// Client is the core driver of an application teamclient.
// It provides the core tools needed by any application/program
// to be the client of an local/remote/in-memory teamserver.
//
// This client object is by default not connected to any teamserver,
// and has therefore no way of fulfilling its core duties, on purpose.
// The client also DOES NOT include any teamserver-side code.
//
// This teamclient core job is to:
//   - Fetch, configure and use teamserver endpoint configurations.
//   - Drive the process of connecting to & disconnecting from a server.
//   - Query a teamserver for its version and users information.
//
// Additionally, this client offers:
//   - Pre-configured loggers for all client-side related events.
//   - Various options to configure its backends and behaviors.
//   - A builtin, abstracted and app-specific filesystem (in memory or on disk).
//
// Various combinations of teamclient/teamserver usage are possible.
// Please see the Go module example/ directory for a list of them.
type Client struct {
	name         string         // Name of the teamclient/teamserver application.
	homeDir      string         // APP_ROOT_DIR var, evaluated once when creating the server.
	opts         *opts          // All configurable things for the teamclient.
	fileLogger   *logrus.Logger // By default, hooked to also provide stdout logging.
	stdoutLogger *logrus.Logger // Fallback logger.
	fs           *assets.FS     // Embedded or on-disk application filesystem.
	mutex        *sync.RWMutex  // Sync access.
	initOpts     sync.Once      // Some options can only be set once when creating the server.

	dialer  Dialer     // Connection backend for the teamclient.
	connect *sync.Once // A client can only connect once per run.

	// Client is the implementation of the core teamclient functionality,
	// which is to query a server version and its current users.
	// This type is either implementated by a teamserver when the client
	// is in-memory, or by a user-defined type which is generally a RPC.
	client team.Client
}

// Dialer represents a type using a teamclient core (and its configured teamserver
// remote) to setup, initiate and use a connection to this remote teamserver.
//
// The type parameter `clientConn` of this interface is a purely syntactic sugar
// to indicate that any dialer should/may return a user-defined but specific object
// from its Dial() method. Library users can register hooks to the teamclient.Client
// using this dialer, and this clientConn will be provided to these hooks.
//
// Examples of what this clientConn can be:
//   - The clientConn is a specific, but non-idiomatic RPC client (ex: a *grpc.ClientConn).
//   - A simple net.Conn over which anything can be done further.
//   - Nothing: a dialer might not need to use or even create a client connection.
type Dialer interface {
	// Init is used by any dialer to query the teamclient driving it about:
	//   - The remote teamserver address and transport credentials
	//   - The user registered in this remote teamserver configuration.
	//   - To make use of client-side loggers, filesystem and other utilities.
	Init(c *Client) error

	// Dial should connect to the endpoint available in any
	// of the client remote teamserver configurations.
	Dial() error

	// Close should close the connection or any related component.
	Close() error
}

// New is the required constructor of a new application teamclient.
// Parameters:
//   - The name of the application using the teamclient.
//   - Variadic options (...Options) which are applied at creation time.
//   - A type implementing the team.Client interface.
//
// Depending on constraints and use cases, the client uses different
// backends and/or RPC implementations providing this functionality:
//   - When used in-memory and as a client of teamserver.
//   - When being provided a specific dialer/client/RPC backend.
//
// The teamclient will only perform a few init things before being returned:
//   - Setup its filesystem, either on-disk (default behavior) or in-memory.
//   - Initialize loggers and the files they use, if any.
//
// This may return an error if the teamclient is unable to work with the provided
// options (or lack thereof), which may happen if the teamclient cannot use and write
// to its directories and log files. No client is returned if the error is not nil.
func New(app string, client team.Client, options ...Options) (*Client, error) {
	teamclient := &Client{
		name:    app,
		opts:    defaultOpts(),
		client:  client,
		connect: &sync.Once{},
		mutex:   &sync.RWMutex{},
		fs:      &assets.FS{},
	}

	teamclient.apply(options...)

	// Filesystem (in-memory or on disk)
	user, _ := user.Current()
	root := filepath.Join(user.HomeDir, "."+teamclient.name)
	teamclient.fs = assets.NewFileSystem(root, teamclient.opts.inMemory)

	// Logging (if allowed)
	if err := teamclient.initLogging(); err != nil {
		return nil, err
	}

	return teamclient, nil
}

// Connect uses the default client configurations to connect to the teamserver.
//
// This call might be blocking and expect user input: if multiple server
// configurations are found in the application directory, the application
// will prompt the user to choose one of them.
// If the teamclient was created WithConfig() option, or if passed in this
// call, user input is guaranteed NOT to be needed.
//
// It only connects the teamclient if it has an available dialer.
// If none is available, this function returns no error, as it is
// possible that this client has a teamclient implementation ready.
func (tc *Client) Connect(options ...Options) (err error) {
	tc.apply(options...)

	// Don't connect if we don't have the connector.
	if tc.dialer == nil {
		return nil
	}

	tc.connect.Do(func() {
		// If we don't have a provided configuration,
		// load one from disk, otherwise do nothing.
		err = tc.initConfig()
		if err != nil {
			err = tc.errorf("%w: %w", ErrConfig, err)
			return
		}

		// Initialize the dialer with our client.
		err = tc.dialer.Init(tc)
		if err != nil {
			err = tc.errorf("%w: %w", ErrConfig, err)
			return
		}

		err = tc.dialer.Dial()
		if err != nil {
			err = tc.errorf("%w: %w", ErrClient, err)
			return
		}
	})

	return err
}

// Disconnect disconnects the client from the server, closing the connection
// and the client log file. Any errors are logged to this file and returned.
// If the teamclient has been passed the WithNoDisconnect() option, it won't
// disconnect.
func (tc *Client) Disconnect() error {
	if tc.opts.console {
		return nil
	}

	// The client can reconnect..
	defer func() {
		tc.connect = &sync.Once{}
	}()

	if tc.dialer == nil {
		return nil
	}

	err := tc.dialer.Close()
	if err != nil {
		tc.log().Error(err)
	}

	return err
}

// Users returns a list of all users registered to the application server.
// If the teamclient has no backend, it returns an ErrNoTeamclient error.
// If the backend returns an error, the latter is returned as is.
func (tc *Client) Users() (users []team.User, err error) {
	if tc.client == nil {
		return nil, ErrNoTeamclient
	}

	res, err := tc.client.Users()
	if err != nil && len(res) == 0 {
		return nil, err
	}

	return res, nil
}

// VersionClient returns the version information of the client, and thus
// does not require the teamclient to be connected to a teamserver.
// This function satisfies the VersionClient() function of the team.Client interface,
// which means that library users are free to reimplement it however they wish.
func (tc *Client) VersionClient() (ver team.Version, err error) {
	if tc.client != nil {
		return tc.client.VersionClient()
	}

	semVer := version.Semantic()
	compiled, _ := version.Compiled()

	var major, minor, patch int32

	if len(semVer) == 3 {
		major = int32(semVer[0])
		minor = int32(semVer[1])
		patch = int32(semVer[2])
	}

	return team.Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Commit:     version.GitCommit(),
		Dirty:      version.GitDirty(),
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// VersionServer returns the version information of the server to which
// the client is connected.
// If the teamclient has no backend, it returns an ErrNoTeamclient error.
// If the backend returns an error, the latter is returned as is.
func (tc *Client) VersionServer() (ver team.Version, err error) {
	if tc.client == nil {
		return ver, ErrNoTeamclient
	}

	version, err := tc.client.VersionServer()
	if err != nil {
		return
	}

	return version, nil
}

// Name returns the name of the client application.
func (tc *Client) Name() string {
	return tc.name
}

// Filesystem returns an abstract filesystem used by the teamclient.
// This filesystem can be either of two things:
//   - By default, the on-disk filesystem, without any specific bounds.
//   - If the teamclient was created with the InMemory() option, a full
//     in-memory filesystem (with root `.app/`).
//
// Use cases for this filesystem might include:
//   - The wish to have a fully abstracted filesystem to work for testing
//   - Ensuring that the filesystem code in your application remains the
//     same regardless of the underlying, actual filesystem.
//
// The type returned is currently an internal type because it wraps some
// os.Filesystem methods for working more transparently: this may change
// in the future if the Go stdlib offers write support to its new io/fs.FS.
func (tc *Client) Filesystem() *assets.FS {
	return tc.fs
}
