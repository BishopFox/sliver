package processes

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Known security tools shown by `ps`/`pstree`.
// __PH0__/__PH1__. 显示的 Known 安全工具
// Process executable -> product name.
// Process 可执行文件 -> 产品 name.
var knownSecurityTools = map[string]string{
	"ccSvcHst.exe":                    "Symantec Endpoint Protection", // Symantec Endpoint Protection (SEP)
	"cb.exe":                          "Carbon Black",                 // Carbon Black
	"RepMgr.exe":                      "Carbon Black Cloud Sensor",    // Carbon Black Cloud Sensor
	"RepUtils.exe":                    "Carbon Black Cloud Sensor",    // Carbon Black Cloud Sensor
	"RepUx.exe":                       "Carbon Black Cloud Sensor",    // Carbon Black Cloud Sensor
	"RepWSC.exe":                      "Carbon Black Cloud Sensor",    // Carbon Black Cloud Sensor
	"scanhost.exe":                    "Carbon Black Cloud Sensor",    // Carbon Black Cloud Sensor
	"elastic-agent.exe":               "Elastic Agent",                // Elastic Agent
	"elastic-endpoint.exe":            "Elastic Agent",                // Elastic Agent
	"filebeat.exe":                    "Elastic Agent",                // Elastic Agent - log shipper
	"filebeat.exe":                    "Elastic Agent",                // Elastic Agent - 原木托运人
	"metricbeat.exe":                  "Elastic Agent",                // Elastic Agent - metric shipper
	"metricbeat.exe":                  "Elastic Agent",                // Elastic Agent - 公制托运人
	"smartscreen.exe":                 "Windows Smart Screen",         // Windows Defender Smart Screen
	"MpCmdRun.exe":                    "Windows Defender",             // Windows Defender Command-line
	"MpCmdRun.exe":                    "Windows Defender",             // Windows Defender Command__PH0__
	"MonitoringHost.exe":              "Windows Defender",             // Microsoft Monitoring Agent
	"HealthService.exe":               "Windows Defender",             // Microsoft Monitoring Agent
	"MsMpEng.exe":                     "Windows Defender",             // Windows Defender (Service Executable)
	"NisSrv.exe":                      "Windows Defender",             // Windows Defender (Network Realtime Inspection)
	"SenseIR.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Live Response Session)
	"SenseNdr.exe":                    "Windows Defender MDE",         // Windows Defender Endpoint (Network Detection and Response)
	"SenseNdr.exe":                    "Windows Defender MDE",         // Windows Defender Endpoint （Network Detection 和 Response）
	"SenseSC.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Screenshot Capture Module)
	"SenseCE.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Classification Engine Module)
	"SenseCM.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Configuration Management Module)
	"SenseSampleUploader.exe":         "Windows Defender MDE",         // Windows Defender Endpoint (Sample Uploader Module)
	"SenseCncProxy.exe":               "Windows Defender MDE",         // Windows Defender Endpoint (Communication Module)
	"MsSense.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Service Executable)
	"CSFalconService.exe":             "CrowdStrike",                  // Crowdstrike Falcon Service
	"CSFalconContainer.exe":           "CrowdStrike",                  // CrowdStrike Falcon Container Security
	"bdservicehost.exe":               "Bitdefender",                  // Bitdefender (Total Security)
	"bdagent.exe":                     "Bitdefender",                  // Bitdefender (Total Security)
	"bdredline.exe":                   "Bitdefender",                  // Bitdefender Redline Update Service (Source https://community.bitdefender.com/en/discussion/82135/bdredline-exe-bitdefender-total-security-2020)
	"bdredline.exe":                   "Bitdefender",                  // Bitdefender Redline Update Service (Source __PH0__
	"Deep Security Manager.exe":       "Trend Micro",                  // TM Deep Security Manager
	"coreServiceShell.exe":            "Trend Micro",                  // TM Anti-malware scan process
	"coreServiceShell.exe":            "Trend Micro",                  // TM Anti__PH0__ 扫描过程
	"ds_monitor.exe":                  "Trend Micro",                  // TM Deep Security Monitor
	"Notifier.exe":                    "Trend Micro",                  // TM Deep Security Notifier's process
	"Notifier.exe":                    "Trend Micro",                  // TM Deep Security Notifier 的流程
	"dsa.exe":                         "Trend Micro",                  // TM Agent's main process
	"dsa.exe":                         "Trend Micro",                  // TM Agent的主要流程
	"ds_nuagent.exe":                  "Trend Micro",                  // TM Advanced TLS traffic inspection
	"ds_nuagent.exe":                  "Trend Micro",                  // TM Advanced TLS 交通检查
	"coreFrameworkHost.exe":           "Trend Micro",                  // TM Anti-malware scan process
	"coreFrameworkHost.exe":           "Trend Micro",                  // TM Anti__PH0__ 扫描过程
	"SentinelServiceHost.exe":         "SentinelOne",                  // Sentinel One
	"SentinelStaticEngine.exe":        "SentinelOne",                  // Sentinel One
	"SentinelStaticEngineScanner.exe": "SentinelOne",                  // Sentinel One
	"SentinelAgent.exe":               "SentinelOne",                  // Sentinel One
	"SentinelAgentWorker.exe":         "SentinelOne",                  // Sentinel One
	"SentinelHelperService.exe":       "SentinelOne",                  // Sentinel One
	"SentinelBrowserNativeHost.exe":   "SentinelOne",                  // Sentinel One
	"SentinelUI.exe":                  "SentinelOne",                  // Sentinel One
	"Sysmon.exe":                      "Sysmon",                       // Sysmon
	"Sysmon64.exe":                    "Sysmon64",                     // Sysmon64
	"CylanceSvc.exe":                  "Cylance",                      // Cylance
	"CylanceUI.exe":                   "Cylance",                      // Cylance
	"TaniumClient.exe":                "Tanium",                       // Tanium
	"TaniumCX.exe":                    "Tanium",                       // Tanium
	"TaniumDetectEngine.exe":          "Tanium",                       // Tanium
	"collector.exe":                   "Rapid 7 Collector",            // Rapid 7 Insight Platform Collector
	"ir_agent.exe":                    "Rapid 7 Insight Agent",        // Rapid 7 Insight Agent
	"eguiproxy.exe":                   "ESET Security",                // ESET Internet Security
	"ekrn.exe":                        "ESET Security",                // ESET Internet Security
	"efwd.exe":                        "ESET Security",                // ESET Internet Security
	"AmSvc.exe":                       "Cybereason ActiveProbe",       // Cybereason ActiveProbe
	"CrAmTray.exe":                    "Cybereason ActiveProbe",       // Cybereason ActiveProbe
	"CrsSvc.exe":                      "Cybereason ActiveProbe",       // Cybereason ActiveProbe
	"CybereasonAV.exe":                "Cybereason ActiveProbe",       // Cybereason ActiveProbe
	"cortex-xdr-payload.exe":          "Palo Alto Cortex",             // Cortex XDR - offline triage
	"cortex-xdr-payload.exe":          "Palo Alto Cortex",             // Cortex XDR - 离线分类
	"cysandbox.exe":                   "Palo Alto Cortex",             // Cortex XDR - sandbox
	"cysandbox.exe":                   "Palo Alto Cortex",             // Cortex XDR - 沙箱
	"cyuserservice.exe":               "Palo Alto Cortex",             // Cortex XDR - user service
	"cyuserservice.exe":               "Palo Alto Cortex",             // Cortex XDR - 用户服务
	"cywscsvc.exe":                    "Palo Alto Cortex",             // Cortex XDR - security center service
	"cywscsvc.exe":                    "Palo Alto Cortex",             // Cortex XDR - 安全中心服务
	"tlaworker.exe":                   "Palo Alto Cortex",             // Cortex XDR - local analysis worker
	"tlaworker.exe":                   "Palo Alto Cortex",             // Cortex XDR - 本地分析人员
	"AEEngine.exe":                    "Faronics Anti-Executable",     // Faronics Anti-Executable - security service
	"AEEngine.exe":                    "Faronics Anti-Executable",     // Faronics Anti__PH0__ - 安全服务
	"Antiexecutable.exe":              "Faronics Anti-Executable",     // Faronics Anti-Executable - gui and tray icon
	"Antiexecutable.exe":              "Faronics Anti-Executable",     // Faronics Anti__PH0__ - GUI 和托盘图标
}

// PsCmd - List processes on the remote system
// 远程系统上的 PsCmd - List 进程
func PsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	ownerFilter, _ := cmd.Flags().GetString("owner")
	cmdLine, _ := cmd.Flags().GetBool("print-cmdline")
	tree, _ := cmd.Flags().GetBool("tree")
	fullInfo, _ := cmd.Flags().GetBool("full")

	if tree && fullInfo {
		con.PrintWarnf("Process tree and full process metadata were requested. " +
			"Full process metadata is not necessary for the process tree, so the request for full process metadata will be ignored.\n\n")
		fullInfo = false
	}

	if ownerFilter != "" && !fullInfo {
		con.PrintErrorf("Filtering on process owner was requested, but full process metadata was not requested. Re-run the command, and specify the -f flag.\n")
		return
	}

	if cmdLine && !fullInfo {
		con.PrintErrorf("Process command line arguments were requested, but full process metadata was not requested. Re-run the command, and specify the -f flag.\n")
		return
	}

	// Get OS information
	// Get OS 信息
	os := getOS(session, beacon)

	/*
		Because a full list can trigger EDR on some platforms,
		Because 完整列表可以在某些平台上触发 EDR，
		namely Windows, filtering of the process list must be
		即Windows，进程列表的过滤必须是
		done on the implant side.
		在 implant side. 上完成
	*/
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		FullInfo: fullInfo,
		Request:  con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if ps.Response != nil && ps.Response.Async {
		con.AddBeaconCallback(ps.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, ps)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintPS(os, ps, false, fullInfo, cmd.Flags(), con)
			products := findKnownSecurityProducts(ps)
			if 0 < len(products) {
				con.Println()
				con.PrintWarnf("Security Product(s): %s\n", strings.Join(products, ", "))
			}
		})
		con.PrintAsyncResponse(ps.Response)
	} else {
		PrintPS(os, ps, true, fullInfo, cmd.Flags(), con)
		products := findKnownSecurityProducts(ps)
		if 0 < len(products) {
			con.Println()
			con.PrintWarnf("Security Product(s): %s\n", strings.Join(products, ", "))
		}
	}
}

func getOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.OS
	} else if beacon != nil {
		return beacon.OS
	}
	return ""
}

// PrintPS - Prints the process list
// PrintPS - Prints 进程列表
func PrintPS(os string, ps *sliverpb.Ps, interactive bool, fullInfo bool, flags *pflag.FlagSet, con *console.SliverClient) {
	pidFilter, _ := flags.GetInt("pid")
	exeFilter, _ := flags.GetString("exe")
	ownerFilter, _ := flags.GetString("owner")
	overflow, _ := flags.GetBool("overflow")
	skipPages, _ := flags.GetInt("skip-pages")
	pstree, _ := flags.GetBool("tree")

	if pstree {
		var currentPID int32
		session, beacon := con.ActiveTarget.GetInteractive()
		if session != nil && session.PID != 0 {
			currentPID = session.PID
		} else if beacon != nil && beacon.PID != 0 {
			currentPID = beacon.PID
		}
		// Print the process tree
		// Print 进程树
		sorted := SortProcessesByPID(ps.Processes)
		tree := NewPsTree(currentPID)
		for _, p := range sorted {
			tree.AddProcess(p)
		}
		con.PrintInfof("Process Tree:\n%s", tree.String())
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	defaultInfo := table.Row{"pid", "ppid", "executable"}

	switch os {
	case "windows":
		if fullInfo {
			tw.AppendHeader(table.Row{"pid", "ppid", "owner", "arch", "executable", "session"})
		} else {
			tw.AppendHeader(defaultInfo)
		}
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		if fullInfo {
			tw.AppendHeader(table.Row{"pid", "ppid", "owner", "arch", "executable"})
		} else {
			tw.AppendHeader(defaultInfo)
		}
	}

	cmdLine, _ := flags.GetBool("print-cmdline")
	for _, proc := range ps.Processes {
		if pidFilter != -1 && proc.Pid != int32(pidFilter) {
			continue
		}
		if exeFilter != "" && !strings.Contains(strings.ToLower(proc.Executable), strings.ToLower(exeFilter)) {
			continue
		}
		if ownerFilter != "" && !strings.Contains(strings.ToLower(proc.Owner), strings.ToLower(ownerFilter)) {
			continue
		}
		row := procRow(proc, cmdLine, fullInfo, con)
		tw.AppendRow(row)
	}
	tw.SortBy([]table.SortBy{
		{Name: "pid", Mode: table.AscNumeric},
		{Name: "ppid", Mode: table.AscNumeric},
	})
	if !interactive {
		overflow = true
	}
	settings.PaginateTable(tw, skipPages, overflow, interactive, con)
}

func findKnownSecurityProducts(ps *sliverpb.Ps) []string {
	uniqProducts := map[string]struct{}{}
	for _, proc := range ps.Processes {
		if product, ok := knownSecurityTools[proc.Executable]; ok {
			uniqProducts[product] = struct{}{}
		}
	}
	products := make([]string, 0, len(uniqProducts))
	for name := range uniqProducts {
		products = append(products, name)
	}
	return products
}

// procRow - Stylizes the process information
// procRow - Stylizes 进程信息
func procRow(proc *commonpb.Process, cmdLine bool, fullInfo bool, con *console.SliverClient) table.Row {
	session, beacon := con.ActiveTarget.GetInteractive()

	style := console.StyleNormal
	if _, ok := knownSecurityTools[proc.Executable]; ok {
		style = console.StyleRed
	}
	if session != nil && proc.Pid == session.PID {
		style = console.StyleGreen
	}
	if beacon != nil && proc.Pid == beacon.PID {
		style = console.StyleGreen
	}

	var row table.Row
	switch session.GetOS() {
	case "windows":
		if cmdLine && fullInfo {
			var args string
			if len(proc.CmdLine) >= 1 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				style.Render(fmt.Sprintf("%d", proc.Pid)),
				style.Render(fmt.Sprintf("%d", proc.Ppid)),
				style.Render(proc.Owner),
				style.Render(proc.Architecture),
				style.Render(args),
				style.Render(fmt.Sprintf("%d", proc.SessionID)),
			}
		} else {
			if fullInfo {
				row = table.Row{
					style.Render(fmt.Sprintf("%d", proc.Pid)),
					style.Render(fmt.Sprintf("%d", proc.Ppid)),
					style.Render(proc.Owner),
					style.Render(proc.Architecture),
					style.Render(proc.Executable),
					style.Render(fmt.Sprintf("%d", proc.SessionID)),
				}
			} else {
				row = table.Row{
					style.Render(fmt.Sprintf("%d", proc.Pid)),
					style.Render(fmt.Sprintf("%d", proc.Ppid)),
					style.Render(proc.Executable),
				}
			}
		}
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		if cmdLine && fullInfo {
			var args string
			if len(proc.CmdLine) >= 2 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				style.Render(fmt.Sprintf("%d", proc.Pid)),
				style.Render(fmt.Sprintf("%d", proc.Ppid)),
				style.Render(proc.Owner),
				style.Render(proc.Architecture),
				style.Render(args),
			}
		} else {
			if fullInfo {
				row = table.Row{
					style.Render(fmt.Sprintf("%d", proc.Pid)),
					style.Render(fmt.Sprintf("%d", proc.Ppid)),
					style.Render(proc.Owner),
					style.Render(proc.Architecture),
					style.Render(proc.Executable),
				}
			} else {
				row = table.Row{
					style.Render(fmt.Sprintf("%d", proc.Pid)),
					style.Render(fmt.Sprintf("%d", proc.Ppid)),
					style.Render(proc.Executable),
				}
			}
		}
	}
	return row
}

// GetPIDByName - Get a PID by name from the active session
// GetPIDByName - Get 和 PID 的名称来自活动 session
func GetPIDByName(cmd *cobra.Command, name string, con *console.SliverClient) int {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		return -1
	}
	for _, proc := range ps.Processes {
		if proc.Executable == name {
			return int(proc.Pid)
		}
	}
	return -1
}

// SortProcessesByPID - Sorts a list of processes by PID
// SortProcessesByPID - Sorts PID 的进程列表
func SortProcessesByPID(ps []*commonpb.Process) []*commonpb.Process {
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Pid < ps[j].Pid
	})
	return ps
}
