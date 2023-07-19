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
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/systemd"
	"github.com/reeflective/team/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func daemoncmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		lhost, err := cmd.Flags().GetString("host")
		if err != nil {
			return fmt.Errorf("Failed to get --host flag: %w", err)
		}

		lport, err := cmd.Flags().GetUint16("port")
		if err != nil {
			return fmt.Errorf("Failed to get --port (%d) flag: %w", lport, err)
		}

		// Also written to logs in the teamserver code.
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "stacktrace from panic: \n"+string(debug.Stack()))
			}
		}()

		// Blocking call, your program will only exit/resume on Ctrl-C/SIGTERM
		return serv.ServeDaemon(lhost, lport)
	}
}

func startListenerCmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		persistent, _ := cmd.Flags().GetBool("persistent")
		ltype, _ := cmd.Flags().GetString("listener")

		_, err := serv.ServeAddr(ltype, lhost, lport)
		if err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Teamserver listener started on %s:%d\n", lhost, lport)

			if persistent {
				serv.ListenerAdd(ltype, lhost, lport)
			}
		} else {
			return fmt.Errorf(command.Warn+"Failed to start job %w", err)
		}

		return nil
	}
}

func closeCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		listeners := serv.Listeners()
		cfg := serv.GetConfig()

		for _, arg := range args {
			if arg == "" {
				continue
			}

			for _, ln := range listeners {
				if strings.HasPrefix(ln.ID, arg) {
					err := serv.ListenerClose(arg)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), command.Warn, err)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Closed %s listener (%s) [%s]\n", ln.Name, formatSmallID(ln.ID), ln.Description)
					}
				}
			}
		}

		for _, arg := range args {
			if arg == "" {
				continue
			}

			for _, saved := range cfg.Listeners {
				if strings.HasPrefix(saved.ID, arg) {
					serv.ListenerRemove(saved.ID)
					id := formatSmallID(saved.ID)
					fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Deleted %s listener (%s) from saved jobs\n", saved.Name, id)

					continue
				}
			}
		}
	}
}

func systemdConfigCmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		config := systemd.NewDefaultConfig()

		userf, _ := cmd.Flags().GetString("user")
		if userf != "" {
			config.User = userf
		}

		binPath, _ := cmd.Flags().GetString("binpath")
		if binPath != "" {
			config.Binpath = binPath
		}

		host, hErr := cmd.Flags().GetString("host")
		if hErr != nil {
			return hErr
		}

		port, pErr := cmd.Flags().GetUint16("port")
		if pErr != nil {
			return pErr
		}

		// The last argument is the systemd command:
		// its parent is the teamserver one, to which
		// should be attached the daemon command.
		daemonCmd, _, err := cmd.Parent().Find([]string{"daemon"})
		if err != nil {
			return fmt.Errorf("Failed to find teamserver daemon command in tree: %w", err)
		}

		config.Args = append(callerArgs(cmd.Parent()), daemonCmd.Name())
		if len(config.Args) > 0 && binPath != "" {
			config.Args[0] = binPath
		}

		if host != "" {
			config.Args = append(config.Args, strings.Join([]string{"--host", host}, " "))
		}

		if port != 0 {
			config.Args = append(config.Args, strings.Join([]string{"--port", strconv.Itoa(int(port))}, " "))
		}

		systemdConfig := systemd.NewFrom(serv.Name(), config)
		fmt.Fprint(cmd.OutOrStdout(), systemdConfig)

		return nil
	}
}

func statusCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.WarnLevel))
			}
		}

		cfg := serv.GetConfig()

		dbCfg := serv.DatabaseConfig()
		database := fmt.Sprintf("%s - %s [%s:%d] ", dbCfg.Dialect, dbCfg.Database, dbCfg.Host, dbCfg.Port)

		// General options, in-memory, default port, config path, database, etc
		fmt.Fprintln(cmd.OutOrStdout(), formatSection("General"))
		fmt.Fprint(cmd.OutOrStdout(), displayGroup([]string{
			"Home", serv.HomeDir(),
			"Port", strconv.Itoa(cfg.DaemonMode.Port),
			"Database", database,
			"Config", serv.ConfigPath(),
		}))

		// Logging files/level/status
		fakeLog := serv.NamedLogger("", "")

		fmt.Fprintln(cmd.OutOrStdout(), formatSection("Logging"))
		fmt.Fprint(cmd.OutOrStdout(), displayGroup([]string{
			"Level", fakeLog.Level.String(),
			"Root", log.FileName(filepath.Join(serv.LogsDir(), serv.Name()), true),
			"Audit", filepath.Join(serv.LogsDir(), "audit.json"),
		}))

		// Certificate files.
		certsPath := serv.CertificatesDir()
		if dir, err := os.Stat(certsPath); err == nil && dir.IsDir() {
			files, err := fs.ReadDir(os.DirFS(certsPath), ".")
			if err == nil || len(files) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), formatSection("Certificate files"))

				for _, file := range files {
					fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(certsPath, file.Name()))
				}
			}
		}

		// Listeners
		listenersTable := listenersTable(serv, cfg)

		if listenersTable != "" {
			fmt.Fprintln(cmd.OutOrStdout(), formatSection("Listeners"))
			fmt.Fprintln(cmd.OutOrStdout(), listenersTable)
		}
	}
}

func listenersTable(serv *server.Server, cfg *server.Config) string {
	listeners := serv.Listeners()

	tbl := &table.Table{}
	tbl.SetStyle(command.TableStyle)

	tbl.AppendHeader(table.Row{
		"ID",
		"Name",
		"Description",
		"State",
		"Persistent",
	})

	for _, listener := range listeners {
		persist := false

		for _, saved := range cfg.Listeners {
			if saved.ID == listener.ID {
				persist = true
			}
		}

		tbl.AppendRow(table.Row{
			formatSmallID(listener.ID),
			listener.Name,
			listener.Description,
			command.Green + command.Bold + "Up" + command.Normal,
			persist,
		})
	}

next:
	for _, saved := range cfg.Listeners {

		for _, ln := range listeners {
			if saved.ID == ln.ID {
				continue next
			}
		}

		tbl.AppendRow(table.Row{
			formatSmallID(saved.ID),
			saved.Name,
			fmt.Sprintf("%s:%d", saved.Host, saved.Port),
			command.Red + command.Bold + "Down" + command.Normal,
			true,
		})
	}

	if len(listeners) > 0 {
		return tbl.Render()
	}

	return ""
}

func fieldName(name string) string {
	return command.Blue + command.Bold + name + command.Normal
}

func callerArgs(cmd *cobra.Command) []string {
	var args []string

	if cmd.HasParent() {
		args = callerArgs(cmd.Parent())
	}

	args = append(args, cmd.Name())

	return args
}

func formatSection(msg string, args ...any) string {
	return "\n" + command.Bold + command.Orange + fmt.Sprintf(msg, args...) + command.Normal
}

// formatSmallID returns a smallened ID for table/completion display.
func formatSmallID(id string) string {
	if len(id) <= 8 {
		return id
	}

	return id[:8]
}

func displayGroup(values []string) string {
	var maxLength int
	var group string

	// Get the padding for headers
	for i, head := range values {
		if i%2 != 0 {
			continue
		}

		if len(head) > maxLength {
			maxLength = len(head)
		}
	}

	for i := 0; i < len(values)-1; i += 2 {
		field := values[i]
		value := values[i+1]

		headName := fmt.Sprintf("%*s", maxLength, field)
		fieldName := command.Blue + command.Bold + headName + command.Normal + " "
		group += fmt.Sprintf("%s: %s\n", fieldName, value)
	}

	return group
}
