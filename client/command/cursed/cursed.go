package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"bytes"
	"fmt"
	insecureRand "math/rand"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// CursedChromeCmd - Execute a .NET assembly in-memory.
func CursedCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Collect existing curses from core
	cursedProcesses := [][]string{}
	core.CursedProcesses.Range(func(key, value interface{}) bool {
		curse := value.(*core.CursedProcess)
		cursedProcesses = append(cursedProcesses, []string{
			fmt.Sprintf("%d", curse.BindTCPPort),
			strings.Split(curse.SessionID, "-")[0],
			fmt.Sprintf("%d", curse.PID),
			curse.Platform,
			curse.ExePath,
			curse.DebugURL().String(),
		})
		return true
	})
	// Display table if we have 1 or more curses
	if 0 < len(cursedProcesses) {
		tw := table.NewWriter()
		tw.SetStyle(settings.GetTableStyle(con))
		tw.AppendHeader(table.Row{
			"Bind Port", "Session ID", "PID", "Platform", "Executable", "Debug URL",
		})
		for _, rowEntries := range cursedProcesses {
			row := table.Row{}
			for _, entry := range rowEntries {
				row = append(row, entry)
			}
			tw.AppendRow(table.Row(row))
		}
		con.Printf("%s\n", tw.Render())
	} else {
		con.PrintInfof("No cursed processes\n")
	}
}

// selectCursedProcess - Interactively select a cursed process from a list.
func selectCursedProcess(con *console.SliverClient) *core.CursedProcess {
	cursedProcesses := []*core.CursedProcess{}
	core.CursedProcesses.Range(func(key, value interface{}) bool {
		cursedProcesses = append(cursedProcesses, value.(*core.CursedProcess))
		return true
	})
	if len(cursedProcesses) < 1 {
		con.PrintErrorf("No cursed processes\n")
		return nil
	}

	port2process := map[int]*core.CursedProcess{}
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	for _, cursedProcess := range cursedProcesses {
		fmt.Fprintf(table, "%d\t%s\t%s\n",
			cursedProcess.BindTCPPort,
			fmt.Sprintf("[Session %s]", strings.Split(cursedProcess.SessionID, "-")[0]),
			cursedProcess.ExePath)
		port2process[cursedProcess.BindTCPPort] = cursedProcess
	}
	table.Flush()
	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.Select{
		Message: "Select a curse:",
		Options: options,
	}
	selected := ""
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	if selected == "" {
		return nil
	}
	selectedPortNumber, _ := strconv.Atoi(strings.Split(selected, " ")[0])
	return port2process[selectedPortNumber]
}

func getRemoteDebuggerPort(cmd *cobra.Command) int {
	port, _ := cmd.Flags().GetInt("remote-debugging-port")
	if port == 0 {
		port = insecureRand.Intn(30000) + 10000
	}
	return port
}
