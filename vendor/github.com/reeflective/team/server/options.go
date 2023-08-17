package server

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
	"os"
	"strings"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/db"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const noTeamdir = "no team subdirectory"

// Options are server options.
// With these you can set/reset or modify the behavior of a teamserver
// at various stages of its lifetime, or when performing some specific
// actions.
// Note that some options can only be used once, while others can be
// used multiple times. Examples of the former are log files and database
// backends, while the latter includes listeners/hooks.
// Each option will specify this in its description.
type Options func(opts *opts)

type opts struct {
	homeDir         string
	teamDir         string
	logFile         string
	local           bool
	noLogs          bool
	inMemory        bool
	continueOnError bool

	config    *Config
	dbConfig  *db.Config
	db        *gorm.DB
	logger    *logrus.Logger
	listeners []Listener
}

// default in-memory configuration, ready to run.
func newDefaultOpts() *opts {
	options := &opts{
		config: getDefaultServerConfig(),
		local:  false,
	}

	return options
}

func (ts *Server) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(ts.opts)
	}

	// The server will apply options multiple times
	// in its lifetime, but some options can only be
	// set once when created.
	ts.initOpts.Do(func() {
		// Application home directory.
		homeDir := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(ts.name)))
		if homeDir != "" {
			ts.homeDir = homeDir
		} else {
			ts.homeDir = ts.opts.homeDir
		}

		// Team directory.
		if ts.opts.teamDir == noTeamdir {
			ts.opts.teamDir = ""
		} else if ts.opts.teamDir == "" {
			ts.opts.teamDir = assets.DirServer
		}

		// User-defined database.
		if ts.opts.db != nil {
			ts.db = ts.opts.db
		}
	})

	// Load any listener backends any number of times.
	for _, listener := range ts.opts.listeners {
		ts.handlers[listener.Name()] = listener
	}

	// Make the first one as the default if needed.
	if len(ts.opts.listeners) > 0 && ts.self == nil {
		ts.self = ts.opts.listeners[0]
	}

	// And clear the most recent listeners passed via options.
	ts.opts.listeners = make([]Listener, 0)
}

//
// *** General options ***
//

// WithInMemory deactivates all interactions of the client with the filesystem.
// This applies to logging, but will also to any forward feature using files.
//
// Implications on database backends:
// By default, all teamservers use sqlite3 as a backend, and thus will run a
// database in memory. All other databases are assumed to be unable to do so,
// and this option will thus trigger an error whenever the option is applied,
// whether it be at teamserver creation, or when it does start listeners.
//
// This option can only be used once, and must be passed to server.New().
func WithInMemory() Options {
	return func(opts *opts) {
		opts.noLogs = true
		opts.inMemory = true
	}
}

// WithDefaultPort sets the default port on which the teamserver should start listeners.
// This default is used in the default daemon configuration, and as command flags defaults.
// The default port set for teamserver applications is port 31416.
//
// This option can only be used once, and must be passed to server.New().
func WithDefaultPort(port uint16) Options {
	return func(opts *opts) {
		opts.config.DaemonMode.Port = int(port)
	}
}

// WithDatabase sets the server database to an existing database.
// Note that it will run an automigration of the teamserver types (certificates and users).
//
// This option can only be used once, and must be passed to server.New().
func WithDatabase(db *gorm.DB) Options {
	return func(opts *opts) {
		opts.db = db
	}
}

// WithDatabaseConfig sets the server to use a database backend with a given configuration.
//
// This option can only be used once, and must be passed to server.New().
func WithDatabaseConfig(config *db.Config) Options {
	return func(opts *opts) {
		opts.dbConfig = config
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

// WithTeamDirectory sets the name (not a path) of the teamserver-specific subdirectory.
// For example, passing "my_server_dir" will make the teamserver use ~/.app/my_server_dir/
// instead of ~/.app/teamserver/.
// If this function is called with an empty string, the teamserver will not use any
// subdirectory for its own outputs, thus using ~/.app as its teamserver directory.
//
// This option can only be used once, and must be passed to server.New().
func WithTeamDirectory(name string) Options {
	return func(opts *opts) {
		if name == "" {
			name = noTeamdir
		}

		opts.teamDir = name
	}
}

//
// *** Logging options ***
//

// WithNoLogs deactivates all logging normally done by the teamserver
// if noLogs is set to true, or keeps/reestablishes them if false.
//
// This option can only be used once, and must be passed to server.New().
func WithNoLogs(noLogs bool) Options {
	return func(opts *opts) {
		opts.noLogs = noLogs
	}
}

// WithLogFile sets the path to the file where teamserver logging should be done.
// The default path is ~/.app/teamserver/logs/app.teamserver.log.
//
// This option can only be used once, and must be passed to server.New().
func WithLogFile(filePath string) Options {
	return func(opts *opts) {
		opts.logFile = filePath
	}
}

// WithLogger sets the teamserver to use a specific logger for
// all logging, except the audit log which is indenpendent.
//
// This option can only be used once, and must be passed to server.New().
func WithLogger(logger *logrus.Logger) Options {
	return func(opts *opts) {
		opts.logger = logger
	}
}

//
// *** Server network/RPC options ***
//

// WithListener registers a listener/server stack with the teamserver.
// The teamserver can then serve this listener stack for any number of bind
// addresses, which users can trigger through the various server.Serve*() methods.
//
// It accepts an optional list of pre-serve hook functions:
// These should accept a generic object parameter which is none other than the
// serverConn returned by the listener.Serve(ln) method. These hooks will
// be very useful- if not necessary- for library users to manipulate their server.
// See the server.Listener type documentation for details.
//
// This option can be used multiple times, either when using
// team/server.New() or with the different server.Serve*() methods.
func WithListener(ln Listener) Options {
	return func(opts *opts) {
		if ln == nil {
			return
		}

		opts.listeners = append(opts.listeners, ln)
	}
}

// WithContinueOnError sets the server behavior when starting persistent listeners
// (either automatically when calling teamserver.ServeDaemon(), or when using
// teamserver.StartPersistentListeners()).
// If true, an error raised by a listener will not prevent others to try starting, and
// errors will be joined into a single one, separated with newlines and logged by default.
// The teamserver has this set to false by default.
//
// This option can be used multiple times.
func WithContinueOnError(continueOnError bool) Options {
	return func(opts *opts) {
		opts.continueOnError = continueOnError
	}
}
