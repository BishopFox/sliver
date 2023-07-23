package commands

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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Generate returns a command tree to embed in client applications connecting
// to a teamserver. It requires only the client to use its functions.
//
// All commands of the tree include an automatic call to client.Connect() to make
// sure they can reach the server for the stuff they need. Thus no pre-runners are
// bound to them for this purpose, and users of this command tree are free to add any.
//
// This tree is only safe to embed within closed-loop applications provided that the
// client *Client was created with WithNoDisconnect(), so that commands can reuse
// their connections more than once before closing.
func Generate(cli *client.Client) *cobra.Command {
	clientCmds := clientCommands(cli)
	return clientCmds
}

// PreRun returns a cobra command runner which connects the client to its teamserver.
// If the client is connected, nothing happens and its current connection reused.
//
// Feel free to use this function as a model for your own teamclient pre-runners.
func PreRun(teamclient *client.Client, opts ...client.Options) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		teamclient.SetLogWriter(cmd.OutOrStdout(), cmd.ErrOrStderr())

		// Ensure we are connected or do it.
		return teamclient.Connect(opts...)
	}
}

// PreRunNoDisconnect is a pre-runner that will connect the teamclient with the
// client.WithNoDisconnect() option, so that after each execution, the client
// connection is kept open. This pre-runner is thus useful for console apps.
//
// Feel free to use this function as a model for your own teamclient pre-runners.
func PreRunNoDisconnect(teamclient *client.Client, opts ...client.Options) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		teamclient.SetLogWriter(cmd.OutOrStdout(), cmd.ErrOrStderr())

		opts = append(opts, client.WithNoDisconnect())

		// The NoDisconnect will prevent teamclient.Disconnect() to close the conn.
		return teamclient.Connect(opts...)
	}
}

// PostRun is a cobra command runner that simply calls client.Disconnect() to close
// the client connection from its teamserver. If the client/commands was configured
// with WithNoDisconnect, this pre-runner will do nothing.
func PostRun(client *client.Client) command.CobraRunnerE {
	return func(cmd *cobra.Command, _ []string) error {
		return client.Disconnect()
	}
}

func clientCommands(cli *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:          "teamclient",
		Short:        "Client-only teamserver commands (import configs, show users, etc)",
		SilenceUsage: true,
	}

	teamFlags := pflag.NewFlagSet("teamserver", pflag.ContinueOnError)
	teamFlags.CountP("verbosity", "v", "Counter flag (-vvv) to increase log verbosity on stdout (1:panic -> 7:debug)")
	teamCmd.PersistentFlags().AddFlagSet(teamFlags)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print teamserver client version",
		RunE:  versionCmd(cli),
	}

	teamCmd.AddCommand(versionCmd)

	importCmd := &cobra.Command{
		Use:   "import",
		Short: fmt.Sprintf("Import a teamserver client configuration file for %s", cli.Name()),
		Run:   importCmd(cli),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveDefault
		},
	}

	iFlags := pflag.NewFlagSet("import", pflag.ContinueOnError)
	iFlags.BoolP("default", "d", false, "Set this config as the default one, if no default config exists already.")
	importCmd.Flags().AddFlagSet(iFlags)

	iComps := carapace.Gen(importCmd)
	iComps.PositionalCompletion(
		carapace.Batch(
			carapace.ActionCallback(ConfigsCompleter(cli, "teamclient/configs", ".teamclient.cfg", "other teamserver apps", true)),
			carapace.ActionFiles().Tag("server configuration").StyleF(getConfigStyle(".teamclient.cfg")),
		).ToA(),
	)

	teamCmd.AddCommand(importCmd)

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Display a table of teamserver users and their status",
		RunE:  usersCmd(cli),
	}

	teamCmd.AddCommand(usersCmd)

	return teamCmd
}

// ConfigsCompleter completes file paths to other teamserver application configs (clients/users CA, etc)
// The filepath is the directory  between .app/ and the target directory where config files of a certain
// type should be found, ext is the normal/default extension for those target files, and tag is used in comps.
func ConfigsCompleter(cli *client.Client, filePath, ext, tag string, noSelf bool) carapace.CompletionCallback {
	return func(ctx carapace.Context) carapace.Action {
		var compErrors []carapace.Action

		homeDir, err := os.UserHomeDir()
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to get user home dir: %s", err))
		}

		dirs, err := os.ReadDir(homeDir)
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to list user directories: %s", err))
		}

		var results []string

		for _, dir := range dirs {
			if !isConfigDir(cli, dir, noSelf) {
				continue
			}

			configPath := filepath.Join(homeDir, dir.Name(), filePath)

			if configs, err := os.Stat(configPath); err == nil {
				if !configs.IsDir() {
					continue
				}

				files, _ := os.ReadDir(configPath)
				for _, file := range files {
					if !strings.HasSuffix(file.Name(), ext) {
						continue
					}

					filePath := filepath.Join(configPath, file.Name())

					cfg, err := cli.ReadConfig(filePath)
					if err != nil || cfg == nil {
						continue
					}

					results = append(results, filePath)
					results = append(results, fmt.Sprintf("[%s] %s:%d", cfg.User, cfg.Host, cfg.Port))
				}
			}
		}

		configsAction := carapace.ActionValuesDescribed(results...).StyleF(getConfigStyle(ext))

		if len(compErrors) > 0 {
			return carapace.Batch(append(compErrors, configsAction)...).ToA()
		}

		return configsAction.Tag(tag)
	}
}

// ConfigsAppCompleter completes file paths to the current application configs.
func ConfigsAppCompleter(cli *client.Client, tag string) carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		var compErrors []carapace.Action

		configPath := cli.ConfigsDir()

		files, err := os.ReadDir(configPath)
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to list user directories: %s", err))
		}

		var results []string

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), command.ClientConfigExt) {
				continue
			}

			filePath := filepath.Join(configPath, file.Name())

			cfg, err := cli.ReadConfig(filePath)
			if err != nil || cfg == nil {
				continue
			}

			results = append(results, filePath)
			results = append(results, fmt.Sprintf("[%s] %s:%d", cfg.User, cfg.Host, cfg.Port))
		}

		configsAction := carapace.ActionValuesDescribed(results...).StyleF(getConfigStyle(command.ClientConfigExt))

		return carapace.Batch(append(
			compErrors,
			configsAction.Tag(tag),
			carapace.ActionFiles())...,
		).ToA()
	})
}

func isConfigDir(cli *client.Client, dir fs.DirEntry, noSelf bool) bool {
	if !strings.HasPrefix(dir.Name(), ".") {
		return false
	}

	if !dir.IsDir() {
		return false
	}

	if strings.TrimPrefix(dir.Name(), ".") != cli.Name() {
		return false
	}

	if noSelf {
		return false
	}

	return true
}

func getConfigStyle(ext string) func(s string, sc style.Context) string {
	return func(s string, sc style.Context) string {
		if strings.HasSuffix(s, ext) {
			return style.Red
		}

		return s
	}
}
