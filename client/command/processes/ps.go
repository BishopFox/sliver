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

// Stylizes known processes in the `ps` command
var knownSecurityTools = map[string][]string{
	// Process Name -> [Color, Stylized Name]
	"ccSvcHst.exe":                    {console.Red, "Symantec Endpoint Protection"}, // Symantec Endpoint Protection (SEP)
	"cb.exe":                          {console.Red, "Carbon Black"},                 // Carbon Black
	"RepMgr.exe":                      {console.Red, "Carbon Black Cloud Sensor"},    // Carbon Black Cloud Sensor
	"RepUtils.exe":                    {console.Red, "Carbon Black Cloud Sensor"},    // Carbon Black Cloud Sensor
	"RepUx.exe":                       {console.Red, "Carbon Black Cloud Sensor"},    // Carbon Black Cloud Sensor
	"RepWSC.exe":                      {console.Red, "Carbon Black Cloud Sensor"},    // Carbon Black Cloud Sensor
	"scanhost.exe":                    {console.Red, "Carbon Black Cloud Sensor"},    // Carbon Black Cloud Sensor
	"elastic-agent.exe":               {console.Red, "Elastic Agent"},                // Elastic Agent
	"elastic-endpoint.exe":            {console.Red, "Elastic Agent"},                // Elastic Agent
	"filebeat.exe":                    {console.Red, "Elastic Agent"},                // Elastic Agent - log shipper
	"metricbeat.exe":                  {console.Red, "Elastic Agent"},                // Elastic Agent - metric shipper
	"smartscreen.exe":                 {console.Red, "Windows Smart Screen"},         // Windows Defender Smart Screen
	"MpCmdRun.exe":                    {console.Red, "Windows Defender"},             // Windows Defender Command-line
	"MonitoringHost.exe":              {console.Red, "Windows Defender"},             // Microsoft Monitoring Agent
	"HealthService.exe":               {console.Red, "Windows Defender"},             // Microsoft Monitoring Agent
	"MsMpEng.exe":                     {console.Red, "Windows Defender"},             // Windows Defender (Service Executable)
	"NisSrv.exe":                      {console.Red, "Windows Defender"},             // Windows Defender (Network Realtime Inspection)
	"SenseIR.exe":                     {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Live Response Session)
	"SenseNdr.exe":                    {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Network Detection and Response)
	"SenseSC.exe":                     {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Screenshot Capture Module)
	"SenseCE.exe":                     {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Classification Engine Module)
	"SenseCM.exe":                     {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Configuration Management Module)
	"SenseSampleUploader.exe":         {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Sample Uploader Module)
	"SenseCncProxy.exe":               {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Communication Module)
	"MsSense.exe":                     {console.Red, "Windows Defender MDE"},         // Windows Defender Endpoint (Service Executable)
	"CSFalconService.exe":             {console.Red, "CrowdStrike"},                  // Crowdstrike Falcon Service
	"CSFalconContainer.exe":           {console.Red, "CrowdStrike"},                  // CrowdStrike Falcon Container Security
	"bdservicehost.exe":               {console.Red, "Bitdefender"},                  // Bitdefender (Total Security)
	"bdagent.exe":                     {console.Red, "Bitdefender"},                  // Bitdefender (Total Security)
	"bdredline.exe":                   {console.Red, "Bitdefender"},                  // Bitdefender Redline Update Service (Source https://community.bitdefender.com/en/discussion/82135/bdredline-exe-bitdefender-total-security-2020)
	"Deep Security Manager.exe":       {console.Red, "Trend Micro"},                  // TM Deep Security Manager
	"coreServiceShell.exe":            {console.Red, "Trend Micro"},                  // TM Anti-malware scan process
	"ds_monitor.exe":                  {console.Red, "Trend Micro"},                  // TM Deep Security Monitor
	"Notifier.exe":                    {console.Red, "Trend Micro"},                  // TM Deep Security Notifier's process
	"dsa.exe":                         {console.Red, "Trend Micro"},                  // TM Agent's main process
	"ds_nuagent.exe":                  {console.Red, "Trend Micro"},                  // TM Advanced TLS traffic inspection
	"coreFrameworkHost.exe":           {console.Red, "Trend Micro"},                  // TM Anti-malware scan process
	"SentinelServiceHost.exe":         {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelStaticEngine.exe":        {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelStaticEngineScanner.exe": {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelAgent.exe":               {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelAgentWorker.exe":         {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelHelperService.exe":       {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelBrowserNativeHost.exe":   {console.Red, "SentinelOne"},                  // Sentinel One
	"SentinelUI.exe":                  {console.Red, "SentinelOne"},                  // Sentinel One
	"Sysmon.exe":                      {console.Red, "Sysmon"},                       // Sysmon
	"Sysmon64.exe":                    {console.Red, "Sysmon64"},                     // Sysmon64
	"CylanceSvc.exe":                  {console.Red, "Cylance"},                      // Cylance
	"CylanceUI.exe":                   {console.Red, "Cylance"},                      // Cylance
	"TaniumClient.exe":                {console.Red, "Tanium"},                       // Tanium
	"TaniumCX.exe":                    {console.Red, "Tanium"},                       // Tanium
	"TaniumDetectEngine.exe":          {console.Red, "Tanium"},                       // Tanium
	"collector.exe":                   {console.Red, "Rapid 7 Collector"},            // Rapid 7 Insight Platform Collector
	"ir_agent.exe":                    {console.Red, "Rapid 7 Insight Agent"},        // Rapid 7 Insight Agent
	"eguiproxy.exe":                   {console.Red, "ESET Security"},                // ESET Internet Security
	"ekrn.exe":                        {console.Red, "ESET Security"},                // ESET Internet Security
	"efwd.exe":                        {console.Red, "ESET Security"},                // ESET Internet Security
	"AmSvc.exe":                       {console.Red, "Cybereason ActiveProbe"},       // Cybereason ActiveProbe
	"CrAmTray.exe":                    {console.Red, "Cybereason ActiveProbe"},       // Cybereason ActiveProbe
	"CrsSvc.exe":                      {console.Red, "Cybereason ActiveProbe"},       // Cybereason ActiveProbe
	"CybereasonAV.exe":                {console.Red, "Cybereason ActiveProbe"},       // Cybereason ActiveProbe
}

// PsCmd - List processes on the remote system
func PsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	os := getOS(session, beacon)
	if ps.Response != nil && ps.Response.Async {
		con.AddBeaconCallback(ps.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, ps)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintPS(os, ps, false, cmd.Flags(), con)
			products := findKnownSecurityProducts(ps)
			if 0 < len(products) {
				con.Println()
				con.PrintWarnf("Security Product(s): %s\n", strings.Join(products, ", "))
			}
		})
		con.PrintAsyncResponse(ps.Response)
	} else {
		PrintPS(os, ps, true, cmd.Flags(), con)
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
func PrintPS(os string, ps *sliverpb.Ps, interactive bool, flags *pflag.FlagSet, con *console.SliverClient) {
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

	switch os {
	case "windows":
		tw.AppendHeader(table.Row{"pid", "ppid", "owner", "arch", "executable", "session"})
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		tw.AppendHeader(table.Row{"pid", "ppid", "owner", "arch", "executable"})
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
		row := procRow(tw, proc, cmdLine, con)
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
	uniqProducts := map[string]string{}
	for _, proc := range ps.Processes {
		if secTool, ok := knownSecurityTools[proc.Executable]; ok {
			uniqProducts[secTool[1]] = secTool[0]
		}
	}
	products := make([]string, 0, len(uniqProducts))
	for name := range uniqProducts {
		products = append(products, name)
	}
	return products
}

// procRow - Stylizes the process information
func procRow(tw table.Writer, proc *commonpb.Process, cmdLine bool, con *console.SliverClient) table.Row {
	session, beacon := con.ActiveTarget.GetInteractive()

	color := console.Normal
	if secTool, ok := knownSecurityTools[proc.Executable]; ok {
		color = secTool[0]
	}
	if session != nil && proc.Pid == session.PID {
		color = console.Green
	}
	if beacon != nil && proc.Pid == beacon.PID {
		color = console.Green
	}

	var row table.Row
	switch session.GetOS() {
	case "windows":
		if cmdLine {
			var args string
			if len(proc.CmdLine) >= 1 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Architecture),
				fmt.Sprintf(color+"%s"+console.Normal, args),
				fmt.Sprintf(color+"%d"+console.Normal, proc.SessionID),
			}
		} else {
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Architecture),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Executable),
				fmt.Sprintf(color+"%d"+console.Normal, proc.SessionID),
			}
		}
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		if cmdLine {
			var args string
			if len(proc.CmdLine) >= 2 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Architecture),
				fmt.Sprintf(color+"%s"+console.Normal, args),
			}
		} else {
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Architecture),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Executable),
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
