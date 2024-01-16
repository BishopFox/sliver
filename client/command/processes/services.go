package processes

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/golang/protobuf/proto"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	serviceStopped         = 1
	serviceStartPending    = 2
	serviceStopPending     = 3
	serviceRunning         = 4
	serviceContinuePending = 5
	servicePausePending    = 6
	servicePaused          = 7
	serviceBootStart       = 0
	serviceSystemStart     = 1
	serviceAutoStart       = 2
	serviceDemandStart     = 3
	serviceDisabled        = 4
)

func ServicesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Hopefully this command being Windows only is temporary
	activeOS := getOS(session, beacon)
	if activeOS != "windows" {
		con.PrintErrorf("The services command is currently only available on Windows")
		return
	}

	hostname, _ := cmd.Flags().GetString("host")

	services, err := con.Rpc.Services(context.Background(), &sliverpb.ServicesReq{
		Hostname: hostname,
		Request:  con.ActiveTarget.Request(cmd),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if services.Response != nil && services.Response.Async {
		con.AddBeaconCallback(services.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, services)
			if err != nil {
				con.PrintErrorf("Failed to decode response: %s\n", err)
				return
			}
			PrintServices(services, con)
		})
		con.PrintAsyncResponse(services.Response)
	} else {
		PrintServices(services, con)
	}
}

func ServiceInfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Hopefully this command being Windows only is temporary
	activeOS := getOS(session, beacon)
	if activeOS != "windows" {
		con.PrintErrorf("The services command is currently only available on Windows")
		return
	}

	hostname, _ := cmd.Flags().GetString("host")
	serviceName := args[0]
	if serviceName == "" {
		con.PrintErrorf("A service name is required")
		return
	}

	serviceInfo, err := con.Rpc.ServiceDetail(context.Background(), &sliverpb.ServiceDetailReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{Hostname: hostname, ServiceName: serviceName},
		Request:     con.ActiveTarget.Request(cmd),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if serviceInfo.Response != nil && serviceInfo.Response.Async {
		con.AddBeaconCallback(serviceInfo.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, serviceInfo)
			if err != nil {
				con.PrintErrorf("Failed to decode response: %s\n", err)
				return
			}
			PrintServiceDetail(serviceInfo, con)
		})
		con.PrintAsyncResponse(serviceInfo.Response)
	} else {
		PrintServiceDetail(serviceInfo, con)
	}
}

func ServiceStopCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Hopefully this command being Windows only is temporary
	activeOS := getOS(session, beacon)
	if activeOS != "windows" {
		con.PrintErrorf("The services command is currently only available on Windows")
		return
	}

	hostname, _ := cmd.Flags().GetString("host")
	serviceName := args[0]
	if serviceName == "" {
		con.PrintErrorf("A service name is required")
		return
	}

	stopService, err := con.Rpc.StopService(context.Background(), &sliverpb.StopServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{Hostname: hostname, ServiceName: serviceName},
		Request:     con.ActiveTarget.Request(cmd),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if stopService.Response != nil && stopService.Response.Async {
		con.AddBeaconCallback(stopService.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, stopService)
			if err != nil {
				con.PrintErrorf("Failed to decode response: %s\n", err)
				return
			}
			// We only get a response with content if there is an error
			if stopService.Response != nil {
				con.PrintErrorf("Error when stopping %s on %s: %s", serviceName, hostname, stopService.Response.Err)
			} else {
				displayName := hostname
				if hostname == "localhost" {
					displayName = beacon.Name
				}
				con.PrintSuccessf("%s on %s stopped successfully", serviceName, displayName)
			}
		})
		con.PrintAsyncResponse(stopService.Response)
	} else {
		if stopService.Response != nil {
			con.PrintErrorf("Error when stopping %s on %s: %s", serviceName, hostname, stopService.Response.Err)
		} else {
			displayName := hostname
			if hostname == "localhost" {
				displayName = session.Name
			}
			con.PrintSuccessf("%s on %s stopped successfully", serviceName, displayName)
		}
	}
}

func ServiceStartCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Hopefully this command being Windows only is temporary
	activeOS := getOS(session, beacon)
	if activeOS != "windows" {
		con.PrintErrorf("The services command is currently only available on Windows")
		return
	}

	hostname, _ := cmd.Flags().GetString("host")
	serviceName := args[0]
	if serviceName == "" {
		con.PrintErrorf("A service name is required")
		return
	}

	startService, err := con.Rpc.StartServiceByName(context.Background(), &sliverpb.StartServiceByNameReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{Hostname: hostname, ServiceName: serviceName},
		Request:     con.ActiveTarget.Request(cmd),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if startService.Response != nil && startService.Response.Async {
		con.AddBeaconCallback(startService.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, startService)
			if err != nil {
				con.PrintErrorf("Failed to decode response: %s\n", err)
				return
			}
			// We only get a response with content if there is an error
			if startService.Response != nil {
				con.PrintErrorf("Error when starting %s on %s: %s", serviceName, hostname, startService.Response.Err)
			} else {
				displayName := hostname
				if hostname == "localhost" {
					displayName = beacon.Name
				}
				con.PrintSuccessf("%s on %s started successfully", serviceName, displayName)
			}
		})
		con.PrintAsyncResponse(startService.Response)
	} else {
		if startService.Response != nil {
			con.PrintErrorf("Error when starting %s on %s: %s", serviceName, hostname, startService.Response.Err)
		} else {
			displayName := hostname
			if hostname == "localhost" {
				displayName = session.Name
			}
			con.PrintSuccessf("%s on %s started successfully", serviceName, displayName)
		}
	}
}

func translateServiceStatus(status uint32) string {
	switch status {
	case serviceStopped:
		return "Stopped"
	case serviceStartPending:
		return "Start Pending"
	case serviceStopPending:
		return "Stop Pending"
	case serviceRunning:
		return "Running"
	case serviceContinuePending:
		return "Continue Pending"
	case servicePausePending:
		return "Pause Pending"
	case servicePaused:
		return "Paused"
	default:
		return fmt.Sprintf("Unknown (status type: %d)", status)
	}
}

func translateServiceStartup(startup uint32) string {
	switch startup {
	case serviceBootStart:
		return "Device (System Loader; Boot)"
	case serviceSystemStart:
		return "Device (IOInitSystem; System)"
	case serviceAutoStart:
		return "Automatic"
	case serviceDemandStart:
		return "Manual"
	case serviceDisabled:
		return "Disabled"
	default:
		return fmt.Sprintf("Unknown (Type %d)", startup)
	}
}

func PrintServices(serviceInfo *sliverpb.Services, con *console.SliverClient) {
	// Get terminal width
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	if serviceInfo.Details == nil {
		if serviceInfo.Error != "" {
			con.PrintErrorf("Encountered the following errors while trying to retrieve service info:\n%s\n", serviceInfo.Error)
		}
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width

	if wideTermWidth {
		tw.AppendHeader(table.Row{
			"Service Name",
			"Display Name",
			"Status",
			"Startup Type",
			"Binary Path",
			"Account",
		})
	} else {
		tw.AppendHeader(table.Row{
			"Service Name",
			"Display Name",
			"Status",
		})
	}

	tw.SortBy([]table.SortBy{
		{Name: "Service Name", Mode: table.Asc},
	})

	for _, service := range serviceInfo.Details {
		status := translateServiceStatus(service.Status)
		var row table.Row
		if wideTermWidth {
			startupType := translateServiceStartup(service.StartupType)
			row = table.Row{
				service.Name,
				service.DisplayName,
				status,
				startupType,
				service.BinPath,
				service.Account,
			}
		} else {
			row = table.Row{
				service.Name,
				service.DisplayName,
				status,
			}
		}
		tw.AppendRow(row)
	}

	con.Printf("%s\n", tw.Render())
	if serviceInfo.Error != "" {
		con.Println()
		con.PrintWarnf("Service info may not be complete. Encountered the following errors while trying to retrieve service info:\n%s\n", serviceInfo.Error)
	}
}

func PrintServiceDetail(serviceDetail *sliverpb.ServiceDetail, con *console.SliverClient) {
	if serviceDetail.Response != nil && serviceDetail.Response.Err != "" {
		con.PrintErrorf("%s\n", serviceDetail.Response.Err)
		return
	}

	if serviceDetail.Detail == nil {
		return
	}

	detail := serviceDetail.Detail

	header := fmt.Sprintf("Service information for %s", detail.Name)

	con.Printf("\n%s\n", header)
	con.Printf("%s\n\n", strings.Repeat("-", len(header)))
	con.Println("Display Name: ", detail.DisplayName)
	con.Println("Description: ", detail.Description)
	con.Println("Account the service runs under: ", detail.Account)
	con.Println("Binary Path: ", detail.BinPath)
	con.Println("Startup type: ", translateServiceStartup(detail.StartupType))
	if serviceDetail.Message != "" {
		con.Println("Status: ", serviceDetail.Message)
	} else {
		con.Println("Status: ", translateServiceStatus(detail.Status))
	}
}
