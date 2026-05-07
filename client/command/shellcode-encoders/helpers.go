package shellcodeencoders

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/rsteube/carapace"
)

var shellcodeEncoderNameToEnum = map[string]clientpb.ShellcodeEncoder{
	"shikata_ga_nai": clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
	"xor":            clientpb.ShellcodeEncoder_XOR,
	"xor_dynamic":    clientpb.ShellcodeEncoder_XOR_DYNAMIC,
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

func shellcodeEncoderEnum(name string) (clientpb.ShellcodeEncoder, bool) {
	normalized := normalizeShellcodeEncoderName(name)
	encoder, ok := shellcodeEncoderNameToEnum[normalized]
	return encoder, ok
}

func fetchShellcodeEncoderMap(con *console.SliverClient) (*clientpb.ShellcodeEncoderMap, error) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	return con.Rpc.ShellcodeEncoderMap(grpcCtx, &commonpb.Empty{})
}

// ShellcodeEncoderNameCompleter returns available encoder names with supported arches.
func ShellcodeEncoderNameCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		encoderMap, err := fetchShellcodeEncoderMap(con)
		if err != nil {
			return carapace.ActionMessage("failed to fetch shellcode encoders: %s", err.Error())
		}

		nameArchs := map[string][]string{}
		for arch, archMap := range encoderMap.GetEncoders() {
			if archMap == nil {
				continue
			}
			for name := range archMap.GetEncoders() {
				nameArchs[name] = append(nameArchs[name], arch)
			}
		}

		names := make([]string, 0, len(nameArchs))
		for name := range nameArchs {
			names = append(names, name)
		}
		sort.Strings(names)

		results := []string{}
		for _, name := range names {
			arches := nameArchs[name]
			sort.Strings(arches)
			desc := fmt.Sprintf("(%s)", strings.Join(arches, ", "))
			results = append(results, name, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("shellcode encoders")
	})
}

// ShellcodeEncoderArchCompleter returns available architectures.
func ShellcodeEncoderArchCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		encoderMap, err := fetchShellcodeEncoderMap(con)
		if err != nil {
			return carapace.ActionMessage("failed to fetch shellcode encoders: %s", err.Error())
		}

		arches := make([]string, 0, len(encoderMap.GetEncoders()))
		for arch := range encoderMap.GetEncoders() {
			arches = append(arches, arch)
		}
		sort.Strings(arches)
		return carapace.ActionValues(arches...).Tag("shellcode architectures")
	})
}
