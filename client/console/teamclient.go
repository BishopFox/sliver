package console

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
	"context"
	"errors"
	"runtime"
	"time"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// PreRunConnect is a spf13/cobra-compliant runner function to be included
// in/as any of the runners that such cobra.Commands offer to use.
//
// The function will connect the Sliver teamclient to a remote server,
// register its client RPC interfaces, and start handling events/log streams.
//
// Note that this function will always check if it used as part of a completion
// command execution call, in which case asciicast/logs streaming is disabled.
func (con *SliverClient) PreRunConnect(cmd *cobra.Command, args []string) error {
	con.FilterCommands(cmd)

	// If commands are imcompatible with the current requirements.
	err := con.App.ActiveMenu().CheckIsAvailable(cmd)
	if err != nil {
		return err
	}

	// Some commands don't need a remote teamserver connection.
	if con.isOffline(cmd) {
		return nil
	}

	// Run any additional pre-run hooks, generally those registered
	// by the sliver-server binary to ensure assets are setup, etc.
	if err := con.runPreConnectHooks(cmd, args); err != nil {
		return err
	}

	// Check if the user told us to connect to a specific server
	// instead of prompting him with all the configs we found.
	clientOpts, err := con.loadConfig(cmd)
	if err != nil {
		return err
	}

	// Let our teamclient connect the transport/RPC stack.
	// Note that this uses a sync.Once to ensure we don't
	// connect more than once.
	if err := con.Teamclient.Connect(clientOpts...); err != nil {
		return err
	}

	// Register our Sliver client services, and monitor events.
	// Also set ourselves up to save our client commands in history.
	con.connect(con.dialer.Conn)

	// Never enable asciicasts/logs streaming when this
	// client is used to perform completions. Both of these will tinker
	// with very low-level IO and very often don't work nice together.
	if compCommandCalled(cmd) {
		return nil
	}

	// Else, initialize our logging/asciicasts streams.
	return con.startClientLog()
}

// loadConfig uses the --config flag (if existing), to override the server remote
// configuration to use, therefore skipping user prompts when there are more than one.
func (con *SliverClient) loadConfig(cmd *cobra.Command) ([]client.Options, error) {
	// No overriding
	if !cmd.Flags().Changed("config") {
		return nil, nil
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	// Let the teamclient attempt to read the config.
	config, err := con.Teamclient.ReadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Should not happen, but just in case.
	if config == nil {
		return nil, errors.New("The teamclient returned no config, but no error")
	}

	return append([]client.Options{}, client.WithConfig(config)), nil
}

// PreRunComplete is a special connection mode which should be
// called in completer functions that need to use the client RPC.
// It is almost equivalent to client.ConnectRun(), but for completions.
//
// If the connection failed, an error is returned along with a completion
// action include the error as a status message, to be returned by completers.
//
// This function is safe to call regardless of the client being used
// as a closed-loop console mode or in an exec-once CLI mode.
func (con *SliverClient) PreRunComplete() (carapace.Action, error) {
	if con.Rpc != nil {
		return carapace.ActionValues(), nil
	}

	// This almost only ever runs teamserver-side pre-runs.
	// We don't need to pass a command to this call, since
	// it does call hooks that should handle nil commands.
	err := con.runPreConnectHooks(nil, nil)
	if err != nil {
		return carapace.ActionMessage("connection error: %s", err), err
	}

	err = con.Teamclient.Connect()
	if err != nil {
		return carapace.ActionMessage("connection error: %s", err), err
	}

	// Register our Sliver client services, and monitor events.
	// Also set ourselves up to save our client commands in history.
	con.connect(con.dialer.Conn)

	return carapace.ActionValues(), nil
}

// PostRunDisconnect disconnects the client from its Sliver server,
// closing all its event/log streams and files, then closing the core
// Sliver RPC client connection. This should be ran as a post-runner.
// After this call, the client can reconnect should it want to.
func (con *SliverClient) PostRunDisconnect(cmd *cobra.Command, args []string) error {
	con.closeClientStreams()

	// Close the RPC client connection.
	return con.Teamclient.Disconnect()
}

// Users returns a list of all users registered with the app teamserver.
// If the gRPC teamclient is not connected or does not have an RPC client,
// an ErrNoRPC is returned.
func (con *SliverClient) Users() (users []team.User, err error) {
	if con.Rpc == nil {
		return nil, errors.New("No Sliver client RPC")
	}

	res, err := con.Rpc.GetUsers(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, con.UnwrapServerErr(err)
	}

	for _, user := range res.GetUsers() {
		users = append(users, team.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: time.Unix(user.LastSeen, 0),
		})
	}

	return
}

// VersionClient implements team.Client.VersionClient() interface method, overriding
// the default teamclient version output to use our Makefile-prepared one.
func (con *SliverClient) VersionClient() (v team.Version, err error) {
	dirty := version.GitDirty != ""
	semVer := version.SemanticVersion()
	compiled, _ := version.Compiled()
	return team.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     version.GitCommit,
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// VersionServer returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (con *SliverClient) VersionServer() (version team.Version, err error) {
	if con.Rpc == nil {
		return version, errors.New("No Sliver client RPC")
	}

	ver, err := con.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		return version, errors.New(status.Convert(err).Message())
	}

	return team.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}

// connect requires a working gRPC connection to the sliver server.
// It starts monitoring events, implant tunnels and client logs streams.
func (con *SliverClient) connect(conn *grpc.ClientConn) {
	con.Rpc = rpcpb.NewSliverRPCClient(conn)

	// Events
	go con.startEventLoop()
	go core.TunnelLoop(con.Rpc)

	// History sources
	sliver := con.App.Menu(consts.ImplantMenu)

	histuser, err := con.newImplantHistory(true)
	if err == nil {
		sliver.AddHistorySource("implant history (user)", histuser)
	}

	histAll, err := con.newImplantHistory(false)
	if err == nil {
		sliver.AddHistorySource("implant history (all users)", histAll)
	}

	con.ActiveTarget.hist = histAll

	con.closeLogs = append(con.closeLogs, func() {
		histuser.Close()
		histAll.Close()
	})
}
