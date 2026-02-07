package rpc

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
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/encoders/shellcode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var shellcodeEncoderNames = map[clientpb.ShellcodeEncoder]string{
	clientpb.ShellcodeEncoder_SHIKATA_GA_NAI: "shikata_ga_nai",
	clientpb.ShellcodeEncoder_XOR:            "xor",
	clientpb.ShellcodeEncoder_XOR_DYNAMIC:    "xor_dynamic",
}

var shellcodeEncoderEnums = map[string]clientpb.ShellcodeEncoder{
	"shikata_ga_nai": clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
	"xor":            clientpb.ShellcodeEncoder_XOR,
	"xor_dynamic":    clientpb.ShellcodeEncoder_XOR_DYNAMIC,
}

// ShellcodeEncode - Encode a piece shellcode
func (rpc *Server) ShellcodeEncoder(ctx context.Context, req *clientpb.ShellcodeEncodeReq) (*clientpb.ShellcodeEncode, error) {

	resp := &clientpb.ShellcodeEncode{Response: &commonpb.Response{}}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	if req.Encoder == clientpb.ShellcodeEncoder_NONE {
		resp.Data = req.Data
		return resp, nil
	}

	encoderName, ok := shellcodeEncoderNames[req.Encoder]
	if !ok {
		resp.Response.Err = "Unknown encoder"
		return resp, nil
	}

	arch := normalizeShellcodeArch(req.Architecture)
	if arch == "" {
		arch = defaultShellcodeArch(encoderName)
	}
	if arch == "" {
		resp.Response.Err = "Unknown architecture"
		return resp, nil
	}

	encoderArchs := shellcode.ShellcodeEncoders[arch]
	if encoderArchs == nil {
		resp.Response.Err = fmt.Sprintf("Unknown architecture: %s", arch)
		return resp, nil
	}
	encoder := encoderArchs[encoderName]
	if encoder == nil {
		resp.Response.Err = fmt.Sprintf("Encoder %s not supported for architecture %s", encoderName, arch)
		return resp, nil
	}

	iterations := int(req.Iterations)
	if iterations <= 0 {
		iterations = 1
	}

	rpcLog.Infof("[rpc] Shellcode encoder request for: %s (%s)", encoderName, arch)
	rpcLog.Infof("[rpc] Encoding shellcode (%d bytes) for architecture %s with %d iterations and badchars: %v", len(req.Data), arch, iterations, req.BadChars)
	encoded, err := encoder.Encode(req.Data, shellcode.ShellcodeEncoderArgs{
		Iterations: iterations,
		BadChars:   req.BadChars,
	})
	if err != nil {
		rpcLog.Errorf("[rpc] Failed to encode shellcode: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to encode shellcode (%s)", err))
	}
	resp.Data = encoded

	rpcLog.Infof("[rpc] Successfully encoded shellcode (%d bytes)", len(resp.Data))

	return resp, nil
}

// ShellcodeEncoderMap - Get a map of support shellcode encoders <human readable/enum>
func (rpc *Server) ShellcodeEncoderMap(ctx context.Context, _ *commonpb.Empty) (*clientpb.ShellcodeEncoderMap, error) {
	resp := &clientpb.ShellcodeEncoderMap{
		Encoders: map[string]*clientpb.ShellcodeEncoderArchMap{},
	}

	arches := make([]string, 0, len(shellcode.ShellcodeEncoders))
	for arch := range shellcode.ShellcodeEncoders {
		arches = append(arches, arch)
	}
	sort.Strings(arches)

	for _, arch := range arches {
		encoderMap := shellcode.ShellcodeEncoders[arch]
		if encoderMap == nil {
			continue
		}
		resp.Encoders[arch] = &clientpb.ShellcodeEncoderArchMap{
			Encoders:     map[string]clientpb.ShellcodeEncoder{},
			Descriptions: map[string]string{},
		}
		names := make([]string, 0, len(encoderMap))
		for name := range encoderMap {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if enumVal, ok := shellcodeEncoderEnums[name]; ok {
				resp.Encoders[arch].Encoders[name] = enumVal
			}
			resp.Encoders[arch].Descriptions[name] = encoderMap[name].Description()
		}
	}
	return resp, nil
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

func defaultShellcodeArch(encoderName string) string {
	var candidates []string
	for arch, encoderMap := range shellcode.ShellcodeEncoders {
		if _, ok := encoderMap[encoderName]; ok {
			candidates = append(candidates, arch)
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	for _, arch := range candidates {
		if arch == "amd64" {
			return arch
		}
	}
	if len(candidates) > 0 {
		sort.Strings(candidates)
		return candidates[0]
	}
	return ""
}
