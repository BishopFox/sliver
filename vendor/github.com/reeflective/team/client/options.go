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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Options are client options.
// You can set or modify the behavior of a teamclient at various
// steps with these options, which are a variadic parameter of
// several client.Client methods.
// Note that some options can only be used once, while others can be
// used multiple times. Examples of the former are log files, while
// the latter includes dialers/hooks.
// Each option will specify this in its description.
type Options func(opts *opts)

type opts struct {
	homeDir  string
	noLogs   bool
	logFile  string
	inMemory bool
	console  bool
	local    bool
	stdout   io.Writer
	config   *Config
	logger   *logrus.Logger
	dialer   Dialer[any]
	hooks    []func(s any) error
}

func defaultOpts() *opts {
	return &opts{
		config: &Config{},
	}
}

func (tc *Client) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(tc.opts)
	}

	// The server will apply options multiple times
	// in its lifetime, but some options can only be
	// set once when created.
	tc.initOpts.Do(func() {
		// Application home directory.
		homeDir := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(tc.name)))
		if homeDir != "" {
			tc.homeDir = homeDir
		} else {
			tc.homeDir = tc.opts.homeDir
		}
	})

	if tc.opts.dialer != nil {
		tc.dialer = tc.opts.dialer
	}

	if tc.opts.stdout != nil {
		tc.stdoutLogger.Out = tc.opts.stdout
	}
}

//
// *** General options ***
//

// WithInMemory deactivates all interactions of the client with the on-disk filesystem.
// This will in effect use in-memory files for all file-based logging and database data.
// This might be useful for testing, or if you happen to embed a teamclient in a program
// without the intent of using it now, etc.
//
// This option can only be used once, and should be passed to server.New().
func WithInMemory() Options {
	return func(opts *opts) {
		opts.noLogs = true
		opts.inMemory = true
	}
}

// WithConfig sets the client to use a given remote teamserver configuration which
// to connect to, instead of using default on-disk user/application configurations.
func WithConfig(config *Config) Options {
	return func(opts *opts) {
		opts.config = config
	}
}

// WithHomeDirectory sets the default path (~/.app/) of the application directory.
// This path can still be overridden at the user-level with the env var APP_ROOT_DIR.
//
// This option can only be used once, and must be passed to server.New().
func WithHomeDirectory(path string) Options {
	return func(opts *opts) {
		opts.homeDir = path
	}
}

//
// *** Logging options ***
//

// WithNoLogs deactivates all logging normally done by the teamclient
// if noLogs is set to true, or keeps/reestablishes them if false.
//
// This option can only be used once, and should be passed to server.New().
func WithNoLogs(noLogs bool) Options {
	return func(opts *opts) {
		opts.noLogs = noLogs
	}
}

// WithLogFile sets the path to the file where teamclient logging should be done.
// If not specified, the client log file is ~/.app/teamclient/logs/app.teamclient.log.
//
// This option can only be used once, and should be passed to server.New().
func WithLogFile(filePath string) Options {
	return func(opts *opts) {
		opts.logFile = filePath
	}
}

// WithLogger sets the teamclient to use a specific logger for logging.
//
// This option can only be used once, and should be passed to server.New().
func WithLogger(logger *logrus.Logger) Options {
	return func(opts *opts) {
		opts.logger = logger
	}
}

//
// *** Client network/RPC options ***
//

// WithDialer sets a custom dialer to connect to the teamserver.
// See the Dialer type documentation for implementation/usage details.
//
// It accepts an optional list of hooks to run on the generic clientConn
// returned by the client.Dialer Dial() method (see Dialer doc for details).
// This client object can be pretty much any client-side conn/RPC object.
// You will have to typecast this conn in your hooks, casting it to the type
// that your teamclient Dialer.Dial() method returns.
//
// This option can be used multiple times, either when using
// team/client.New() or when using the teamclient.Connect() method.
func WithDialer(dialer Dialer[any], hooks ...func(clientConn any) error) Options {
	return func(opts *opts) {
		opts.dialer = dialer
		opts.hooks = append(opts.hooks, hooks...)
	}
}

// WithLocalDialer sets the teamclient to connect with an in-memory dialer
// (provided when creating the teamclient). This in effect only prevents
// the teamclient from looking and loading/prompting remote client configs.
//
// Because this is automatically called by the teamserver.Serve(client)
// function, you should probably not have any reason to use this option.
func WithLocalDialer() Options {
	return func(opts *opts) {
		opts.local = true
	}
}

// WithNoDisconnect is meant to be used when the teamclient commands are used
// in a closed-loop (readline-style) application, where the connection is used
// more than once in the lifetime of the Go program.
// If this is the case, this option will ensure that any cobra client command
// runners produced by this library will not disconnect after each execution.
//
// This option can only be used once, and should be passed to server.New().
func WithNoDisconnect() Options {
	return func(opts *opts) {
		opts.console = true
	}
}
