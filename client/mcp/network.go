package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	networkInterfacesToolName = "network_interfaces"
	netstatToolName           = "netstat"
)

// network_interfaces 工具参数
type networkInterfacesArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

// netstat 工具参数
type netstatArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	TCP            bool   `json:"tcp,omitempty"`
	UDP            bool   `json:"udp,omitempty"`
	IP4            bool   `json:"ip4,omitempty"`
	IP6            bool   `json:"ip6,omitempty"`
	Listening      bool   `json:"listening,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

// network_interfaces 结果
type netInterfaceInfo struct {
	Index       int32    `json:"index"`
	Name        string   `json:"name"`
	MAC         string   `json:"mac"`
	IPAddresses []string `json:"ip_addresses"`
}

type networkInterfacesResult struct {
	Interfaces []netInterfaceInfo `json:"interfaces"`
	Count      int                `json:"count"`
}

// netstat 结果
type sockAddr struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

type netstatEntry struct {
	Protocol   string      `json:"protocol"`
	LocalAddr  sockAddr    `json:"local_addr"`
	RemoteAddr sockAddr    `json:"remote_addr"`
	State      string      `json:"state"`
	UID        uint32      `json:"uid"`
	Process    *processRef `json:"process,omitempty"`
}

type processRef struct {
	PID        int32  `json:"pid"`
	Executable string `json:"executable,omitempty"`
	Owner      string `json:"owner,omitempty"`
}

type netstatResult struct {
	Entries []netstatEntry `json:"entries"`
	Count   int            `json:"count"`
}

// network_interfaces 处理器
func (s *SliverMCPServer) networkInterfacesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args networkInterfacesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handleNetworkInterfaces(ctx, args)
}

func (s *SliverMCPServer) handleNetworkInterfaces(ctx context.Context, args networkInterfacesArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(networkInterfacesToolName, args.SessionID, args.BeaconID)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	ifconfigResp, err := s.Rpc.Ifconfig(ctx, &sliverpb.IfconfigReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get network interfaces", err), nil
	}

	if ifconfigResp.Response != nil && ifconfigResp.Response.Err != "" {
		return mcpapi.NewToolResultError(ifconfigResp.Response.Err), nil
	}

	if isBeacon && ifconfigResp.Response != nil && ifconfigResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("network_interfaces", ifconfigResp.Response.TaskID, ifconfigResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Ifconfig{}
		if err := s.waitForBeaconTaskResponse(ctx, ifconfigResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await network interfaces task", err), nil
		}
		ifconfigResp = resolved
		if ifconfigResp.Response != nil && ifconfigResp.Response.Err != "" {
			return mcpapi.NewToolResultError(ifconfigResp.Response.Err), nil
		}
	}

	result := networkInterfacesResult{
		Interfaces: make([]netInterfaceInfo, 0, len(ifconfigResp.NetInterfaces)),
	}

	for _, iface := range ifconfigResp.NetInterfaces {
		if iface == nil {
			continue
		}
		result.Interfaces = append(result.Interfaces, netInterfaceInfo{
			Index:       iface.Index,
			Name:        iface.Name,
			MAC:         iface.MAC,
			IPAddresses: iface.IPAddresses,
		})
	}

	result.Count = len(result.Interfaces)
	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// netstat 处理器
func (s *SliverMCPServer) netstatHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args netstatArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handleNetstat(ctx, args)
}

func (s *SliverMCPServer) handleNetstat(ctx context.Context, args netstatArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		netstatToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("tcp=%t", args.TCP),
		fmt.Sprintf("udp=%t", args.UDP),
		fmt.Sprintf("ip4=%t", args.IP4),
		fmt.Sprintf("ip6=%t", args.IP6),
		fmt.Sprintf("listening=%t", args.Listening),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	netstatResp, err := s.Rpc.Netstat(ctx, &sliverpb.NetstatReq{
		Request:   req,
		TCP:       args.TCP,
		UDP:       args.UDP,
		IP4:       args.IP4,
		IP6:       args.IP6,
		Listening: args.Listening,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get netstat", err), nil
	}

	if netstatResp.Response != nil && netstatResp.Response.Err != "" {
		return mcpapi.NewToolResultError(netstatResp.Response.Err), nil
	}

	if isBeacon && netstatResp.Response != nil && netstatResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("netstat", netstatResp.Response.TaskID, netstatResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Netstat{}
		if err := s.waitForBeaconTaskResponse(ctx, netstatResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await netstat task", err), nil
		}
		netstatResp = resolved
		if netstatResp.Response != nil && netstatResp.Response.Err != "" {
			return mcpapi.NewToolResultError(netstatResp.Response.Err), nil
		}
	}

	result := netstatResult{
		Entries: make([]netstatEntry, 0, len(netstatResp.Entries)),
	}

	for _, entry := range netstatResp.Entries {
		if entry == nil {
			continue
		}

		netEntry := netstatEntry{
			Protocol: entry.Protocol,
			State:    entry.SkState,
			UID:      entry.UID,
		}

		if entry.LocalAddr != nil {
			netEntry.LocalAddr = sockAddr{
				IP:   entry.LocalAddr.Ip,
				Port: entry.LocalAddr.Port,
			}
		}

		if entry.RemoteAddr != nil {
			netEntry.RemoteAddr = sockAddr{
				IP:   entry.RemoteAddr.Ip,
				Port: entry.RemoteAddr.Port,
			}
		}

		if entry.Process != nil {
			netEntry.Process = &processRef{
				PID:        entry.Process.Pid,
				Executable: entry.Process.Executable,
				Owner:      entry.Process.Owner,
			}
		}

		result.Entries = append(result.Entries, netEntry)
	}

	result.Count = len(result.Entries)
	return mcpapi.NewToolResultStructuredOnly(result), nil
}
