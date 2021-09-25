package tasks

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
	"google.golang.org/protobuf/proto"
)

// TasksFetchCmd - Manage beacon tasks
func TasksFetchCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}
	beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	tasks := beaconTasks.Tasks
	if len(tasks) == 0 {
		con.PrintErrorf("No tasks for beacon\n")
		return
	}

	idArg := ctx.Args.String("id")
	if idArg != "" {
		tasks = filterTasksByID(idArg, tasks)
		if len(tasks) == 0 {
			con.PrintErrorf("No beacon task found with id %s\n", idArg)
			return
		}
	}

	filter := ctx.Flags.String("filter")
	if filter != "" {
		tasks = filterTasksByTaskType(filter, tasks)
		if len(tasks) == 0 {
			con.PrintErrorf("No beacon tasks with filter type '%s'\n", filter)
			return
		}
	}

	var task *clientpb.BeaconTask
	if 1 < len(tasks) {
		task, err = SelectBeaconTask(tasks)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf(console.UpN+console.Clearln, 1)
	} else {
		task = tasks[0]
	}
	task, err = con.Rpc.GetBeaconTaskContent(context.Background(), &clientpb.BeaconTask{ID: task.ID})
	if err != nil {
		con.PrintErrorf("Failed to fetch task content: %s\n", err)
		return
	}
	PrintTask(task, con)
}

func filterTasksByID(taskID string, tasks []*clientpb.BeaconTask) []*clientpb.BeaconTask {
	filteredTasks := []*clientpb.BeaconTask{}
	for _, task := range tasks {
		if strings.HasPrefix(task.ID, strings.ToLower(taskID)) {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

func filterTasksByTaskType(taskType string, tasks []*clientpb.BeaconTask) []*clientpb.BeaconTask {
	filteredTasks := []*clientpb.BeaconTask{}
	for _, task := range tasks {
		if strings.HasPrefix(strings.ToLower(task.Description), strings.ToLower(taskType)) {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

// PrintTask - Print the details of a beacon task
func PrintTask(task *clientpb.BeaconTask, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.AppendRow(table.Row{console.Bold + "Beacon Task" + console.Normal, task.ID})
	tw.AppendSeparator()
	tw.AppendRow(table.Row{"State", emojiState(task.State) + " " + prettyState(strings.Title(task.State))})
	tw.AppendRow(table.Row{"Description", task.Description})
	tw.AppendRow(table.Row{"Created", time.Unix(task.CreatedAt, 0).Format(time.RFC1123)})
	if !time.Unix(task.SentAt, 0).IsZero() {
		tw.AppendRow(table.Row{"Sent", time.Unix(task.SentAt, 0).Format(time.RFC1123)})
	}
	if !time.Unix(task.CompletedAt, 0).IsZero() {
		tw.AppendRow(table.Row{"Completed", time.Unix(task.CompletedAt, 0).Format(time.RFC1123)})
	}

	tw.AppendRow(table.Row{"Request Size", util.ByteCountBinary(int64(len(task.Request)))})
	if !time.Unix(task.CompletedAt, 0).IsZero() {
		tw.AppendRow(table.Row{"Response Size", util.ByteCountBinary(int64(len(task.Response)))})
	}
	tw.AppendSeparator()
	con.Printf("%s\n", tw.Render())
	if !time.Unix(task.CompletedAt, 0).IsZero() {
		con.Println()
		if 0 < len(task.Response) {
			renderTaskResponse(task, con)
		} else {
			con.PrintInfof("No task response\n")
		}
	}
}

func emojiState(state string) string {
	switch strings.ToLower(state) {
	case "completed":
		return "✅"
	case "pending":
		return "⏳"
	case "failed":
		return "❌"
	default:
		return "❓"
	}
}

// Decode and render message specific content
func renderTaskResponse(task *clientpb.BeaconTask, con *console.SliverConsoleClient) {
	reqEnvelope := &sliverpb.Envelope{}
	proto.Unmarshal(task.Request, reqEnvelope)
	switch reqEnvelope.Type {

	// ---------------------
	// Exec commands
	// ---------------------
	case sliverpb.MsgExecuteAssemblyReq:
		execAssembly := &sliverpb.ExecuteAssembly{}
		err := proto.Unmarshal(task.Response, execAssembly)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, _ := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		hostname := "hostname"
		if beacon != nil {
			hostname = beacon.Hostname
		}
		assemblyPath := ""
		ctx := &grumble.Context{
			Command: &grumble.Command{Name: "execute-assembly"},
			Flags: grumble.FlagMap{
				"save": &grumble.FlagMapItem{Value: false, IsDefault: true},
				"loot": &grumble.FlagMapItem{Value: false, IsDefault: true},
			},
		}
		exec.PrintExecuteAssembly(execAssembly, hostname, assemblyPath, ctx, con)

	// execute-shellcode
	case sliverpb.MsgTaskReq:
		shellcodeExec := &sliverpb.Task{}
		err := proto.Unmarshal(task.Response, shellcodeExec)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		exec.PrintExecuteShellcode(shellcodeExec, con)

	case sliverpb.MsgExecuteReq:
		execReq := &sliverpb.ExecuteReq{}
		err := proto.Unmarshal(reqEnvelope.Data, execReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		execResult := &sliverpb.Execute{}
		err = proto.Unmarshal(task.Response, execResult)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		ctx := &grumble.Context{
			Flags: grumble.FlagMap{
				"ignore-stderr": &grumble.FlagMapItem{Value: false},
				"loot":          &grumble.FlagMapItem{Value: false},
				"stdout":        &grumble.FlagMapItem{Value: ""},
				"stderr":        &grumble.FlagMapItem{Value: ""},
				"output":        &grumble.FlagMapItem{Value: true},
			},
			Args: grumble.ArgMap{
				"command":   &grumble.ArgMapItem{Value: execReq.Path},
				"arguments": &grumble.ArgMapItem{Value: execReq.Args},
			},
		}
		exec.PrintExecute(execResult, ctx, con)

	case sliverpb.MsgSideloadReq:
		sideload := &sliverpb.Sideload{}
		err := proto.Unmarshal(reqEnvelope.Data, sideload)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, _ := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		hostname := "hostname"
		if beacon != nil {
			hostname = beacon.Hostname
		}
		ctx := &grumble.Context{
			Command: &grumble.Command{Name: "sideload"},
			Flags: grumble.FlagMap{
				"save": &grumble.FlagMapItem{Value: false},
			},
		}
		exec.PrintSideload(sideload, hostname, ctx, con)

	case sliverpb.MsgSpawnDllReq:
		spawnDll := &sliverpb.SpawnDll{}
		err := proto.Unmarshal(reqEnvelope.Data, spawnDll)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, _ := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		hostname := "hostname"
		if beacon != nil {
			hostname = beacon.Hostname
		}
		ctx := &grumble.Context{
			Command: &grumble.Command{Name: "spawndll"},
			Flags: grumble.FlagMap{
				"save": &grumble.FlagMapItem{Value: false},
			},
		}
		exec.PrintSpawnDll(spawnDll, hostname, ctx, con)

	// ---------------------
	// File system commands
	// ---------------------

	// Cat = download

	case sliverpb.MsgCdReq:
		pwd := &sliverpb.Pwd{}
		err := proto.Unmarshal(task.Response, pwd)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintPwd(pwd, con)

	case sliverpb.MsgDownload:
		download := &sliverpb.Download{}
		err := proto.Unmarshal(task.Response, download)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		taskResponseDownload(download, con)

	case sliverpb.MsgLsReq:
		ls := &sliverpb.Ls{}
		err := proto.Unmarshal(task.Response, ls)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		flags := grumble.FlagMap{
			"reverse":  &grumble.FlagMapItem{Value: false},
			"modified": &grumble.FlagMapItem{Value: false},
			"size":     &grumble.FlagMapItem{Value: false},
		}
		filesystem.PrintLs(ls, flags, "", con)

	case sliverpb.MsgMkdirReq:
		mkdir := &sliverpb.Mkdir{}
		err := proto.Unmarshal(task.Response, mkdir)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintMkdir(mkdir, con)

	case sliverpb.MsgPwdReq:
		pwd := &sliverpb.Pwd{}
		err := proto.Unmarshal(task.Response, pwd)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintPwd(pwd, con)

	case sliverpb.MsgRmReq:
		rm := &sliverpb.Rm{}
		err := proto.Unmarshal(task.Response, rm)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintRm(rm, con)

	case sliverpb.MsgUpload:
		upload := &sliverpb.Upload{}
		err := proto.Unmarshal(task.Response, upload)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintUpload(upload, con)

	// ---------------------
	// Default
	// ---------------------
	default:
		con.PrintErrorf("Cannot render task response for msg type %v\n", reqEnvelope.Type)
	}
}

func taskResponseDownload(download *sliverpb.Download, con *console.SliverConsoleClient) {
	const (
		dump   = "Dump Contents"
		saveTo = "Save to File ..."
	)
	action := saveTo
	prompt := &survey.Select{
		Message: "Choose an option:",
		Options: []string{dump, saveTo},
	}
	err := survey.AskOne(prompt, &action, survey.WithValidator(survey.Required))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	switch action {
	case dump:
		con.Printf("%s\n", string(download.Data))
	default:
		promptSaveToFile(download.Data, con)
	}
}

func promptSaveToFile(data []byte, con *console.SliverConsoleClient) {
	saveTo := ""
	saveToPrompt := &survey.Input{Message: "Save to: "}
	err := survey.AskOne(saveToPrompt, &saveTo)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if _, err := os.Stat(saveTo); !os.IsNotExist(err) {
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite existing file?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
	}
	err = ioutil.WriteFile(saveTo, data, 0600)
	if err != nil {
		con.PrintErrorf("Failed to save file: %s\n", err)
		return
	}
	con.PrintInfof("Wrote %d byte(s) to %s", len(data), saveTo)
}
