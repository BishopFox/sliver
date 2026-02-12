package generate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// GetSliverBinary - Get the binary of an implant based on it's profile.
// GetSliverBinary - Get 基于 profile. 的 implant 的二进制文件
func GetSliverBinary(profile *clientpb.ImplantProfile, con *console.SliverClient) ([]byte, error) {
	var data []byte

	ctrl := make(chan bool)
	con.SpinUntil("Compiling, please wait ...", ctrl)

	generated, err := con.Rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Config: profile.Config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error generating implant\n")
		return data, err
	}
	data = generated.GetFile().GetData()

	if profile.Config.Format == clientpb.OutputFormat_SHELLCODE {
		encoder := profile.Config.ShellcodeEncoder
		legacySGN := false
		if encoder == clientpb.ShellcodeEncoder_NONE && profile.Config.SGNEnabled {
			encoder = clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
			legacySGN = true
		}
		if encoder != clientpb.ShellcodeEncoder_NONE {
			encoderName := map[clientpb.ShellcodeEncoder]string{
				clientpb.ShellcodeEncoder_SHIKATA_GA_NAI: "shikata_ga_nai",
				clientpb.ShellcodeEncoder_XOR:            "xor",
				clientpb.ShellcodeEncoder_XOR_DYNAMIC:    "xor_dynamic",
			}[encoder]
			if encoderName == "" {
				return nil, fmt.Errorf("unknown shellcode encoder enum %d", int32(encoder))
			}

			encoderMap, err := fetchShellcodeEncoderMap(con)
			if err != nil {
				return nil, err
			}
			if _, ok := shellcodeEncoderEnumForArch(encoderMap, profile.Config.GOARCH, encoderName); !ok {
				msg := fmt.Sprintf("shellcode encoder %q is not supported for arch %s", encoderName, normalizeShellcodeArch(profile.Config.GOARCH))
				if legacySGN {
					con.PrintWarnf("%s, skipping.\n", msg)
				} else {
					return nil, errors.New(msg)
				}
			} else {
				encodeResp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
					Encoder:      encoder,
					Architecture: profile.Config.GOARCH,
					Iterations:   1,
					BadChars:     []byte{},
					Data:         data,
				})
				if err != nil {
					con.PrintErrorf("Error encoding shellcode")
					return nil, err
				}
				data = encodeResp.Data
			}
		}
	}

	_, err = con.Rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		con.PrintErrorf("Error updating implant profile\n")
		return data, err
	}
	return data, err
}

func compilerTargets(con *console.SliverClient) (*clientpb.Compiler, error) {
	if con == nil || con.Rpc == nil {
		return nil, fmt.Errorf("no compiler info")
	}
	return con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func compilerArchValues(compiler *clientpb.Compiler) []string {
	results := []string{}

	for _, target := range compiler.Targets {
		results = appendUnique(results, target.GOARCH)
	}

	for _, target := range compiler.UnsupportedTargets {
		results = appendUnique(results, target.GOARCH)
	}

	return results
}

func compilerOSValues(compiler *clientpb.Compiler) []string {
	results := []string{}

	for _, target := range compiler.Targets {
		results = appendUnique(results, target.GOOS)
	}

	for _, target := range compiler.UnsupportedTargets {
		results = appendUnique(results, target.GOOS)
	}

	return results
}

func filterByPrefix(values []string, prefix string) []string {
	if prefix == "" {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

func registerTargetFlagCompletion(cmd *cobra.Command, name string, valuesFn func(*clientpb.Compiler) []string, con *console.SliverClient) {
	if cmd == nil {
		return
	}
	if _, ok := cmd.GetFlagCompletionFunc(name); ok {
		return
	}
	if cmd.Flags().Lookup(name) == nil && cmd.PersistentFlags().Lookup(name) == nil {
		return
	}
	_ = cmd.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		compiler, err := compilerTargets(con)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		values := valuesFn(compiler)
		return filterByPrefix(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func registerImplantTargetFlagCompletions(cmd *cobra.Command, con *console.SliverClient) {
	registerTargetFlagCompletion(cmd, "os", compilerOSValues, con)
	registerTargetFlagCompletion(cmd, "arch", compilerArchValues, con)
}

// FormatCompleter completes builds' architectures.
// FormatCompleter 完成构建' architectures.
func ArchCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		compiler, err := compilerTargets(con)
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", err.Error())
		}

		return carapace.ActionValues(compilerArchValues(compiler)...).Tag("architectures")
	})
}

// FormatCompleter completes build operating systems.
// FormatCompleter 完成构建操作 systems.
func OSCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		compiler, err := compilerTargets(con)
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", err.Error())
		}

		return carapace.ActionValues(compilerOSValues(compiler)...).Tag("operating systems")
	})
}

// FormatCompleter completes build formats.
// FormatCompleter 完成构建 formats.
func FormatCompleter() carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		return carapace.ActionValues([]string{
			"exe", "shared", "service", "shellcode",
		}...).Tag("implant format")
	})
}

// HTTPC2Completer - Completes the HTTP C2 PROFILES
// HTTPC2Completer - Completes HTTP C2 PROFILES
func HTTPC2Completer(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()
		httpC2Profiles, err := con.Rpc.GetHTTPC2Profiles(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch HTTP C2 profiles: %s", err.Error())
		}

		var results []string
		for _, profile := range httpC2Profiles.Configs {
			results = append(results, profile.Name)
		}
		return carapace.ActionValues(results...).Tag("HTTP C2 Profiles")
	})
}

// TrafficEncoderCompleter - Completes the names of traffic encoders.
// TrafficEncoderCompleter - Completes 流量名称 encoders.
func TrafficEncodersCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()
		trafficEncoders, err := con.Rpc.TrafficEncoderMap(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch traffic encoders: %s", err.Error())
		}

		results := []string{}
		for _, encoder := range trafficEncoders.Encoders {
			results = append(results, encoder.Wasm.Name)
			skipTests := ""
			if encoder.SkipTests {
				skipTests = "[skip-tests]"
			}
			desc := fmt.Sprintf("(Wasm: %s) %s", encoder.Wasm.Name, skipTests)
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("traffic encoders")
	})
}
