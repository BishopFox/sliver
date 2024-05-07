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
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

// TasksFetchCmd - Manage beacon tasks.
func TasksFetchCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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

	var idArg string
	if len(args) > 0 {
		idArg = args[0]
	}
	if idArg != "" {
		tasks = filterTasksByID(idArg, tasks)
		if len(tasks) == 0 {
			con.PrintErrorf("No beacon task found with id %s\n", idArg)
			return
		}
	}

	filter, _ := cmd.Flags().GetString("filter")
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

// PrintTask - Print the details of a beacon task.
func PrintTask(task *clientpb.BeaconTask, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableWithBordersStyle(con))
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
	case "canceled":
		return "🚫"
	default:
		return "❓"
	}
}

// Decode and render message specific content.
func renderTaskResponse(task *clientpb.BeaconTask, con *console.SliverClient) {
	reqEnvelope := &sliverpb.Envelope{}
	proto.Unmarshal(task.Request, reqEnvelope)
	switch reqEnvelope.Type {

	// ---------------------
	// Environment commands
	// ---------------------
	case sliverpb.MsgEnvReq:
		envInfo := &sliverpb.EnvInfo{}
		err := proto.Unmarshal(task.Response, envInfo)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		environment.PrintGetEnvInfo(envInfo, con)

	case sliverpb.MsgSetEnvReq:
		setEnvReq := &sliverpb.SetEnvReq{}
		err := proto.Unmarshal(task.Request, setEnvReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		setEnv := &sliverpb.SetEnv{}
		err = proto.Unmarshal(task.Response, setEnv)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		environment.PrintSetEnvInfo(setEnvReq.Variable.Key, setEnvReq.Variable.Value, setEnv, con)

	case sliverpb.MsgUnsetEnvReq:
		unsetEnvReq := &sliverpb.UnsetEnvReq{}
		err := proto.Unmarshal(task.Request, unsetEnvReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		unsetEnv := &sliverpb.UnsetEnv{}
		err = proto.Unmarshal(task.Response, unsetEnv)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		environment.PrintUnsetEnvInfo(unsetEnvReq.Name, unsetEnv, con)

	// ---------------------
	// Call extension commands
	// ---------------------
	case sliverpb.MsgCallExtensionReq:
		callExtension := &sliverpb.CallExtension{}
		err := proto.Unmarshal(task.Response, callExtension)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		extensions.PrintExtOutput("", "", nil, callExtension, con)

	// ---------------------
	// Exec commands
	// ---------------------
	case sliverpb.MsgInvokeExecuteAssemblyReq:
		fallthrough
	case sliverpb.MsgInvokeInProcExecuteAssemblyReq:
		fallthrough
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

		f := pflag.NewFlagSet(constants.ExecuteAssemblyStr, pflag.ContinueOnError)
		f.BoolP("save", "s", false, "save output to file")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")

		assemblyCmd := &cobra.Command{Use: constants.ExecuteAssemblyStr}
		assemblyCmd.Flags().AddFlagSet(f)

		exec.HandleExecuteAssemblyResponse(execAssembly, assemblyPath, hostname, assemblyCmd, con)

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

		f := pflag.NewFlagSet(constants.ExecuteStr, pflag.ContinueOnError)
		f.BoolP("output", "o", true, "capture command output")
		f.BoolP("loot", "X", false, "save output as loot")
		f.BoolP("ignore-stderr", "S", false, "don't print STDERR output")
		f.StringP("stdout", "O", "", "remote path to redirect STDOUT to")
		f.StringP("stderr", "E", "", "remote path to redirect STDERR to")

		execCmd := &cobra.Command{Use: constants.ExecuteStr}
		execCmd.Flags().AddFlagSet(f)
		execCmd.SetArgs(append([]string{execReq.Path}, execReq.Args...))

		exec.PrintExecute(execResult, execCmd, con)

	case sliverpb.MsgSideloadReq:
		sideload := &sliverpb.Sideload{}
		err := proto.Unmarshal(task.Response, sideload)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, _ := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		hostname := "hostname"
		if beacon != nil {
			hostname = beacon.Hostname
		}

		f := pflag.NewFlagSet(constants.SideloadStr, pflag.ContinueOnError)
		f.BoolP("save", "s", false, "save output to file")
		f.BoolP("loot", "X", false, "save output as loot")

		sideloadCmd := &cobra.Command{Use: constants.SideloadStr}
		sideloadCmd.Flags().AddFlagSet(f)

		exec.HandleSideloadResponse(sideload, "", hostname, sideloadCmd, con)

	case sliverpb.MsgSpawnDllReq:
		spawnDll := &sliverpb.SpawnDll{}
		err := proto.Unmarshal(task.Response, spawnDll)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, _ := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		hostname := "hostname"
		if beacon != nil {
			hostname = beacon.Hostname
		}

		f := pflag.NewFlagSet(constants.SpawnDllStr, pflag.ContinueOnError)
		f.BoolP("save", "s", false, "save output to file")
		f.BoolP("loot", "X", false, "save output as loot")

		spawnDllCmd := &cobra.Command{Use: constants.SpawnDllStr}
		spawnDllCmd.Flags().AddFlagSet(f)

		exec.HandleSpawnDLLResponse(spawnDll, "", hostname, spawnDllCmd, con)

	case sliverpb.MsgSSHCommandReq:
		sshCommand := &sliverpb.SSHCommand{}
		err := proto.Unmarshal(task.Response, sshCommand)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		exec.PrintSSHCmd(sshCommand, con)

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

	case sliverpb.MsgDownloadReq:
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

		f := pflag.NewFlagSet("ls", pflag.ContinueOnError)
		f.BoolP("reverse", "r", false, "reverse sort order")
		f.BoolP("modified", "m", false, "sort by modified time")
		f.BoolP("size", "s", false, "sort by size")

		filesystem.PrintLs(ls, f, con)

	case sliverpb.MsgMvReq:
		mv := &sliverpb.Mv{}
		err := proto.Unmarshal(task.Response, mv)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}

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

	case sliverpb.MsgUploadReq:
		upload := &sliverpb.Upload{}
		err := proto.Unmarshal(task.Response, upload)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintUpload(upload, con)

	case sliverpb.MsgChmodReq:
		chmod := &sliverpb.Chmod{}
		err := proto.Unmarshal(task.Response, chmod)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintChmod(chmod, con)

	case sliverpb.MsgChownReq:
		chown := &sliverpb.Chown{}
		err := proto.Unmarshal(task.Response, chown)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintChown(chown, con)

	case sliverpb.MsgChtimesReq:
		chtimes := &sliverpb.Chtimes{}
		err := proto.Unmarshal(task.Response, chtimes)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintChtimes(chtimes, con)

	case sliverpb.MsgMemfilesListReq:
		memfilesList := &sliverpb.Ls{}
		err := proto.Unmarshal(task.Response, memfilesList)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintMemfiles(memfilesList, con)

	case sliverpb.MsgMemfilesAddReq:
		memfilesAdd := &sliverpb.MemfilesAdd{}
		err := proto.Unmarshal(task.Response, memfilesAdd)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintAddMemfile(memfilesAdd, con)

	case sliverpb.MsgMemfilesRmReq:
		memfilesRm := &sliverpb.MemfilesRm{}
		err := proto.Unmarshal(task.Response, memfilesRm)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		filesystem.PrintRmMemfile(memfilesRm, con)

	// ---------------------
	// Network commands
	// ---------------------
	case sliverpb.MsgIfconfigReq:
		ifconfig := &sliverpb.Ifconfig{}
		err := proto.Unmarshal(task.Response, ifconfig)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		network.PrintIfconfig(ifconfig, true, con)

	case sliverpb.MsgNetstatReq:
		netstat := &sliverpb.Netstat{}
		err := proto.Unmarshal(task.Response, netstat)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, err := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		if err != nil {
			con.PrintErrorf("Failed to fetch beacon: %s\n", err)
			return
		}
		network.PrintNetstat(netstat, beacon.PID, beacon.ActiveC2, false, con)

	// ---------------------
	// Privilege commands
	// ---------------------
	case sliverpb.MsgGetPrivsReq:
		privs := &sliverpb.GetPrivs{}
		err := proto.Unmarshal(task.Response, privs)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, err := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		if err != nil {
			con.PrintErrorf("Failed to fetch beacon: %s\n", err)
			return
		}
		privilege.PrintGetPrivs(privs, beacon.PID, con)

	case sliverpb.MsgInvokeGetSystemReq:
		getSystem := &sliverpb.GetSystem{}
		err := proto.Unmarshal(task.Response, getSystem)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		privilege.PrintGetSystem(getSystem, con)

	case sliverpb.MsgCurrentTokenOwnerReq:
		cto := &sliverpb.CurrentTokenOwner{}
		err := proto.Unmarshal(task.Response, cto)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}

	case sliverpb.MsgImpersonateReq:
		impersonateReq := &sliverpb.ImpersonateReq{}
		err := proto.Unmarshal(task.Response, impersonateReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		impersonate := &sliverpb.Impersonate{}
		err = proto.Unmarshal(task.Response, impersonate)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		privilege.PrintImpersonate(impersonate, impersonateReq.Username, con)

	case sliverpb.MsgMakeTokenReq:
		makeTokenReq := &sliverpb.MakeTokenReq{}
		err := proto.Unmarshal(task.Response, makeTokenReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		makeToken := &sliverpb.MakeToken{}
		err = proto.Unmarshal(task.Response, makeToken)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		privilege.PrintMakeToken(makeToken, makeTokenReq.Domain, makeTokenReq.Username, con)

	case sliverpb.MsgRunAsReq:
		runAsReq := &sliverpb.RunAsReq{}
		err := proto.Unmarshal(task.Response, runAsReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		runAs := &sliverpb.RunAs{}
		err = proto.Unmarshal(task.Response, runAs)
		if err != nil {
			con.PrintErrorf("Failed to decode task request: %s\n", err)
			return
		}
		beacon, err := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		if err != nil {
			con.PrintErrorf("Failed to fetch beacon: %s\n", err)
			return
		}
		privilege.PrintRunAs(runAs, runAsReq.ProcessName, runAsReq.Args, beacon.Name, con)

	// ---------------------
	// Processes commands
	// ---------------------
	case sliverpb.MsgProcessDumpReq:
		dump := &sliverpb.ProcessDump{}
		err := proto.Unmarshal(task.Response, dump)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		promptSaveToFile(dump.Data, con)

	case sliverpb.MsgPsReq:
		ps := &sliverpb.Ps{}
		err := proto.Unmarshal(task.Response, ps)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		beacon, err := con.Rpc.GetBeacon(context.Background(), &clientpb.Beacon{ID: task.BeaconID})
		if err != nil {
			con.PrintErrorf("Failed to get beacon: %s\n", err)
			return
		}

		f := pflag.NewFlagSet("ps", pflag.ContinueOnError) // Create the flag set.
		f.IntP("pid", "p", -1, "filter based on pid")
		f.StringP("exe", "e", "", "filter based on executable name")
		f.StringP("owner", "o", "", "filter based on owner")
		f.BoolP("print-cmdline", "c", true, "print command line arguments")
		f.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
		f.IntP("skip-pages", "S", 0, "skip the first n page(s)")
		f.BoolP("tree", "T", false, "print process tree")

		processes.PrintPS(beacon.OS, ps, true, f, con)

	case sliverpb.MsgTerminateReq:
		terminate := &sliverpb.Terminate{}
		err := proto.Unmarshal(task.Response, terminate)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		processes.PrintTerminate(terminate, con)

	// ---------------------
	// Registry commands
	// ---------------------
	case sliverpb.MsgRegistryCreateKeyReq:
		createKeyReq := &sliverpb.RegistryCreateKeyReq{}
		err := proto.Unmarshal(task.Request, createKeyReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		createKey := &sliverpb.RegistryCreateKey{}
		err = proto.Unmarshal(task.Response, createKey)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintCreateKey(createKey, createKeyReq.Path, createKeyReq.Key, con)

	case sliverpb.MsgRegistryDeleteKeyReq:
		deleteKeyReq := &sliverpb.RegistryDeleteKeyReq{}
		err := proto.Unmarshal(task.Request, deleteKeyReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		deleteKey := &sliverpb.RegistryDeleteKey{}
		err = proto.Unmarshal(task.Response, deleteKey)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintDeleteKey(deleteKey, deleteKeyReq.Path, deleteKeyReq.Key, con)

	case sliverpb.MsgRegistryListValuesReq:
		listValuesReq := &sliverpb.RegistryListValuesReq{}
		err := proto.Unmarshal(task.Request, listValuesReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		regList := &sliverpb.RegistryValuesList{}
		err = proto.Unmarshal(task.Response, regList)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintListValues(regList, listValuesReq.Hive, listValuesReq.Path, con)

	case sliverpb.MsgRegistrySubKeysListReq:
		listValuesReq := &sliverpb.RegistrySubKeyListReq{}
		err := proto.Unmarshal(task.Request, listValuesReq)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		regList := &sliverpb.RegistrySubKeyList{}
		err = proto.Unmarshal(task.Response, regList)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintListSubKeys(regList, listValuesReq.Hive, listValuesReq.Path, con)

	case sliverpb.MsgRegistryReadReq:
		regRead := &sliverpb.RegistryRead{}
		err := proto.Unmarshal(task.Response, regRead)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintRegRead(regRead, con)

	case sliverpb.MsgRegistryWriteReq:
		regWrite := &sliverpb.RegistryWrite{}
		err := proto.Unmarshal(task.Response, regWrite)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		registry.PrintRegWrite(regWrite, con)

	// ---------------------
	// Screenshot
	// ---------------------
	case sliverpb.MsgScreenshotReq:
		screenshot := &sliverpb.Screenshot{}
		err := proto.Unmarshal(task.Response, screenshot)
		if err != nil {
			con.PrintErrorf("Failed to decode task response: %s\n", err)
			return
		}
		promptSaveToFile(screenshot.Data, con)

	// ---------------------
	// Default
	// ---------------------
	default:
		con.PrintErrorf("Cannot render task response for msg type %v\n", reqEnvelope.Type)
	}
}

func taskResponseDownload(download *sliverpb.Download, con *console.SliverClient) {
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

func promptSaveToFile(data []byte, con *console.SliverClient) {
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
	err = os.WriteFile(saveTo, data, 0o600)
	if err != nil {
		con.PrintErrorf("Failed to save file: %s\n", err)
		return
	}
	con.PrintInfof("Wrote %d byte(s) to %s", len(data), saveTo)
}
