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
	"net"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"syscall"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
)

// Listener represents a teamserver listener stack.
// Any type implementing this interface can be served and controlled
// by a team/server.Server core, and remote clients can connect to it
// with the appropriate/corresponding team/client.Dialer backend.
//
// Errors: all errors returned by the listener interface methods are considered critical,
// (except the Close() error one), and thus will stop the listener start/server process
// when raised. Thus, you should only return errors that are critical to the operation
// of your listener. You can use the teamserver loggers to log/print non-critical ones.
type Listener interface {
	// Name returns the name of the "listener/server/RPC" stack
	// of this listener, eg. "gRPC" for a gRPC listener, "myCustomHTTP"
	// for your quick-and-dirty custom stack, etc.
	// Note that this name is used as a key by the teamserver to store the
	// different listener stacks it may use, so this name should be unique
	// among all listener stacks registered to a given teamserver runtime.
	Name() string

	// Init is used by the listener to access the core teamserver, needed for:
	//   - Fetching server-side transport/session-level credentials.
	//   - Authenticating users connections/requests.
	//   - Using the builtin teamserver loggers, filesystem and other utilities.
	// Any non-nil error returned will abort the listener starting process.
	Init(s *Server) error

	// Listen is used to create and bind a network listener to some address
	// Implementations are free to handle incoming connections the way they
	// want, since they have had access to the server in Init() for anything
	// related they might need.
	// As an example, the gRPC default transport serves a gRPC server on this
	// listener, registers its RPC services, and returns the listener for the
	// teamserver to wrap it in job control.
	// This call MUST NOT block, just like the normal usage of net.Listeners.
	Listen(addr string) (ln net.Listener, err error)

	// Close should close the listener stack.
	// This can mean different things depending on use case, but some are not recommended.
	//   - It can simply close the "listener" layer without shutting down the "server/RPC" layer.
	//   - It can shutdown anything, thus in effect disconnecting all of its clients from server.
	Close() error
}

// Serve attempts the default listener of the teamserver (which is either
// the first one to have been registered, or the only one registered at all).
// It the responsibility of any teamclients produced by the teamserver.Self()
// method to call their Connect() method: the server will answer.
func (ts *Server) Serve(cli *client.Client, opts ...Options) error {
	if ts.self == nil {
		return ErrNoListener
	}

	// Some errors might come from user-provided hooks,
	// so we don't wrap errors again, our own errors
	// have been prepared accordingly in this call.
	err := ts.serve(ts.self, "", "", 0, opts...)
	if err != nil {
		return err
	}

	// Use a fake config with a non-empty name.
	cliOpts := []client.Options{
		client.WithConfig(&client.Config{User: "server"}),
	}

	return cli.Connect(cliOpts...)
}

// ServeDaemon is a blocking call which starts the teamserver as daemon process, using
// either the provided host:port arguments, or the ones found in the teamserver config.
// This function will also (and is the only one to) start all persistent team listeners.
//
// It blocks by waiting for a syscal.SIGTERM (eg. CtrlC on Linux) signal. Upon receival,
// the teamserver will close the main listener (the daemon one), but not persistent ones.
//
// Errors raised by closing the listener are wrapped in an ErrListener, logged and returned.
func (ts *Server) ServeDaemon(host string, port uint16, opts ...Options) (err error) {
	log := ts.NamedLogger("daemon", "main")

	// cli args take president over config
	if host == blankHost {
		host = ts.opts.config.DaemonMode.Host
		log.Debugf("No host specified, using config file default: %s", host)
	}

	if port == blankPort {
		port = uint16(ts.opts.config.DaemonMode.Port)
		log.Debugf("No port specified, using config file default: %d", port)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic:\n%s", debug.Stack())
		}
	}()

	// Start the listener.
	log.Infof("Starting %s teamserver daemon on %s:%d ...", ts.Name(), host, port)

	listenerID, err := ts.ServeAddr(ts.self.Name(), host, port, opts...)
	if err != nil {
		return err
	}

	// Now that the main teamserver listener is started,
	// we can start all our persistent teamserver listeners.
	// That way, if any of them collides with our current bind,
	// we just serve it for him
	hostPort := regexp.MustCompile(fmt.Sprintf("%s:%d", host, port))

	err = ts.ListenerStartPersistents()
	if err != nil && hostPort.MatchString(err.Error()) {
		log.Errorf("Error starting persistent listeners: %s\n", err)
	}

	done := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		<-signals
		log.Infof("Received SIGTERM, exiting ...")

		err = ts.ListenerClose(listenerID)
		if err != nil {
			log.Errorf("%s: %s", ErrListener, err)
		}
		done <- true
	}()
	<-done

	return err
}

// ServeAddr attempts to serve a listener stack identified by "name" (the listener should be registered
// with the teamserver with WithListener() option), on a given host:port address, with any provided option.
// If returns either a critical error raised by the listener, or the ID of the listener job, for control.
// The call is non-blocking, contrarily to the server.ServeDaemon() method.
func (ts *Server) ServeAddr(name string, host string, port uint16, opts ...Options) (id string, err error) {
	// If server was not initialized yet, do it.
	// This at least will update any listener/server-specific options.
	err = ts.init(opts...)
	if err != nil {
		return "", ts.errorf("%w: %w", ErrTeamServer, err)
	}

	// Ensure we have at least one available listener.
	handler := ts.handlers[name]

	if handler == nil {
		handler = ts.self
	}

	if handler == nil {
		return "", ErrNoListener
	}

	// Generate the listener ID now so we can return it.
	listenerID := getRandomID()

	err = ts.serve(handler, listenerID, host, port, opts...)

	return listenerID, err
}

// serve will attempt to serve a given listener/server stack to a given (host:port) address.
// If the ID parameter is empty, a job ID for this listener will be automatically generated.
// Any errors raised by the listener itself are considered critical and returned wrapped in a ListenerErr.
func (ts *Server) serve(ln Listener, ID, host string, port uint16, opts ...Options) error {
	log := ts.NamedLogger("teamserver", "handler")

	// If server was not initialized yet, do it.
	// This has no effect redundant with the ServeAddr() method.
	err := ts.init(opts...)
	if err != nil {
		return ts.errorf("%w: %w", ErrTeamServer, err)
	}

	// Let the handler initialize itself: load everything it needs from
	// the server, configuration, fetch certificates, log stuff, etc.
	err = ln.Init(ts)
	if err != nil {
		return ts.errorWith(log, "%w: %w", ErrListener, err)
	}

	// Now let the handler start listening on somewhere.
	laddr := fmt.Sprintf("%s:%d", host, port)

	// This call should not block, serve the listener immediately.
	listener, err := ln.Listen(laddr)
	if err != nil {
		return ts.errorWith(log, "%w: %w", ErrListener, err)
	}

	// The server is running, so add a job anyway.
	ts.addListenerJob(ID, ln.Name(), host, int(port), listener)

	return nil
}

// Handlers returns a copy of its teamserver listeners map.
// This can be useful if you want to start them with the server ServeListener() method.
// Or -but this is not recommended by this library- to use those listeners without the
// teamserver driving the init/start/serve/stop process.
func (ts *Server) Handlers() map[string]Listener {
	handlers := make(map[string]Listener, len(ts.handlers))

	for name, handler := range ts.handlers {
		handlers[name] = handler
	}

	return handlers
}

func (ts *Server) init(opts ...Options) error {
	var err error

	// Always reaply options, since it could be used by different listeners.
	ts.apply(opts...)

	ts.initServe.Do(func() {
		// Database configuration.
		if err = ts.initDatabase(); err != nil {
			return
		}

		// Load any relevant server configuration: on disk,
		// contained in options, or the default one.
		ts.opts.config = ts.GetConfig()

		// Certificate infrastructure, will make the code panic if unable to work properly.
		certsLog := ts.NamedLogger("certs", "certificates")
		ts.certs = certs.NewManager(ts.fs, ts.dbSession(), certsLog, ts.Name(), ts.TeamDir())
	})

	return err
}
