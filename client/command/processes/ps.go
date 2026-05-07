package processes

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
// Process executable -> product name.
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
	"metricbeat.exe":                  "Elastic Agent",                // Elastic Agent - metric shipper
	"smartscreen.exe":                 "Windows Smart Screen",         // Windows Defender Smart Screen
	"MpCmdRun.exe":                    "Windows Defender",             // Windows Defender Command-line
	"MonitoringHost.exe":              "Windows Defender",             // Microsoft Monitoring Agent
	"HealthService.exe":               "Windows Defender",             // Microsoft Monitoring Agent
	"MsMpEng.exe":                     "Windows Defender",             // Windows Defender (Service Executable)
	"NisSrv.exe":                      "Windows Defender",             // Windows Defender (Network Realtime Inspection)
	"SenseIR.exe":                     "Windows Defender MDE",         // Windows Defender Endpoint (Live Response Session)
	"SenseNdr.exe":                    "Windows Defender MDE",         // Windows Defender Endpoint (Network Detection and Response)
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
	"Deep Security Manager.exe":       "Trend Micro",                  // TM Deep Security Manager
	"coreServiceShell.exe":            "Trend Micro",                  // TM Anti-malware scan process
	"ds_monitor.exe":                  "Trend Micro",                  // TM Deep Security Monitor
	"Notifier.exe":                    "Trend Micro",                  // TM Deep Security Notifier's process
	"dsa.exe":                         "Trend Micro",                  // TM Agent's main process
	"ds_nuagent.exe":                  "Trend Micro",                  // TM Advanced TLS traffic inspection
	"coreFrameworkHost.exe":           "Trend Micro",                  // TM Anti-malware scan process
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
	"cysandbox.exe":                   "Palo Alto Cortex",             // Cortex XDR - sandbox
	"cyuserservice.exe":               "Palo Alto Cortex",             // Cortex XDR - user service
	"cywscsvc.exe":                    "Palo Alto Cortex",             // Cortex XDR - security center service
	"tlaworker.exe":                   "Palo Alto Cortex",             // Cortex XDR - local analysis worker
	"AEEngine.exe":                    "Faronics Anti-Executable",     // Faronics Anti-Executable - security service
	"Antiexecutable.exe":              "Faronics Anti-Executable",     // Faronics Anti-Executable - gui and tray icon
}

// PsCmd - List processes on the remote system
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
	os := getOS(session, beacon)

	/*
		Because a full list can trigger EDR on some platforms,
		namely Windows, filtering of the process list must be
		done on the implant side.
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
func SortProcessesByPID(ps []*commonpb.Process) []*commonpb.Process {
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Pid < ps[j].Pid
	})
	return ps
}
