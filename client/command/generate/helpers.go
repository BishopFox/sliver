package generate

import (
	"context"
	"fmt"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/cache"
	"github.com/rsteube/carapace/pkg/style"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// GetSliverBinary - Get the binary of an implant based on it's profile.
func GetSliverBinary(profile *clientpb.ImplantProfile, con *console.SliverClient) ([]byte, error) {
	var data []byte
	// get implant builds
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return data, con.UnwrapServerErr(err)
	}

	implantName := buildImplantName(profile.GetConfig().GetFileName())
	_, ok := builds.GetConfigs()[implantName]
	if implantName == "" || !ok {
		// no built implant found for profile, generate a new one
		con.PrintInfof("No builds found for profile %s, generating a new one\n", profile.GetName())
		ctrl := make(chan bool)
		con.SpinUntil("Compiling, please wait ...", ctrl)

		generated, err := con.Rpc.Generate(context.Background(), &clientpb.GenerateReq{
			Config: profile.Config,
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			con.PrintErrorf("Error generating implant\n")
			return data, con.UnwrapServerErr(err)
		}
		data = generated.GetFile().GetData()
		profile.Config.FileName = generated.File.Name
		_, err = con.Rpc.SaveImplantProfile(context.Background(), profile)
		if err != nil {
			con.PrintErrorf("Error updating implant profile\n")
			return data, con.UnwrapServerErr(err)
		}
	} else {
		// Found a build, reuse that one
		con.PrintInfof("Sliver name for profile: %s\n", implantName)
		regenerate, err := con.Rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
			ImplantName: implantName,
		})
		if err != nil {
			return data, con.UnwrapServerErr(err)
		}
		data = regenerate.GetFile().GetData()
	}
	return data, err
}

// FormatCompleter completes builds' architectures.
func ArchCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", con.UnwrapServerErr(err))
		}

		var results []string

	nextTarget:
		for _, target := range compiler.Targets {
			for _, res := range results {
				if res == target.GOARCH {
					continue nextTarget
				}
			}
			results = append(results, target.GOARCH)
		}

	nextUnsupported:
		for _, target := range compiler.UnsupportedTargets {
			for _, res := range results {
				if res == target.GOARCH {
					continue nextUnsupported
				}
			}
			results = append(results, target.GOARCH)
		}

		return carapace.ActionValues(results...).Tag("architectures")
	}).Cache(completers.CacheCompilerInfo)
}

// FormatCompleter completes build operating systems.
func OSCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", con.UnwrapServerErr(err))
		}

		var results []string

	nextTarget:
		for _, target := range compiler.Targets {
			for _, res := range results {
				if res == target.GOOS {
					continue nextTarget
				}
			}
			results = append(results, target.GOOS)
		}

	nextUnsupported:
		for _, target := range compiler.UnsupportedTargets {
			for _, res := range results {
				if res == target.GOOS {
					continue nextUnsupported
				}
			}
			results = append(results, target.GOOS)
		}

		return carapace.ActionValues(results...).Tag("operating systems")
	}).Cache(completers.CacheCompilerInfo)
}

// FormatCompleter completes build formats.
func FormatCompleter() carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		return carapace.ActionValues([]string{
			"exe", "shared", "service", "shellcode",
		}...).Tag("implant format")
	})
}

// MsfFormatCompleter completes MsfVenom stager formats.
func MsfFormatCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		info, err := con.Rpc.GetMetasploitCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch Metasploit info: %s", con.UnwrapServerErr(err))
		}

		var results []string

		for _, fmt := range info.Formats {
			fmt = strings.TrimSpace(fmt)
			if fmt == "" {
				continue
			}

			results = append(results, fmt)

		}

		return carapace.ActionValues(results...).Tag("msfvenom formats")
	}).Cache(completers.CacheMsf)
}

// MsfArchCompleter completes MsfVenom stager architectures.
func MsfArchCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		info, err := con.Rpc.GetMetasploitCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch Metasploit info: %s", con.UnwrapServerErr(err))
		}

		var results []string

		for _, arch := range info.Archs {
			arch = strings.TrimSpace(arch)
			if arch == "" {
				continue
			}

			results = append(results, arch)
		}

		return carapace.ActionValues(results...).Tag("msfvenom archs")
	}).Cache(completers.CacheMsf)
}

// MsfFormatCompleter completes MsfVenom stager encoders.
func MsfEncoderCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		info, err := con.Rpc.GetMetasploitCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch Metasploit info: %s", con.UnwrapServerErr(err))
		}

		var results []string

		for _, mod := range info.Encoders {
			results = append(results, mod.FullName)

			level := fmt.Sprintf("%-10s", "["+mod.Quality+"]")
			desc := fmt.Sprintf("%s %s", level, mod.Description)

			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("msfvenom encoders")
	}).Cache(completers.CacheMsf)
}

// MsfPayloadCompleter completes Metasploit payloads.
func MsfPayloadCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if msg, err := con.PreRunComplete(); err != nil {
			return msg
		}

		info, err := con.Rpc.GetMetasploitCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch Metasploit info: %s", con.UnwrapServerErr(err))
		}

		var results []string

		for _, mod := range info.Payloads {
			if mod.FullName == "" && mod.Name == "" {
				continue
			}

			results = append(results, mod.FullName)
			results = append(results, mod.Description)
		}

		return carapace.ActionValuesDescribed(results...)
	}).Cache(completers.CacheMsf, cache.String("payloads")).MultiParts("/").StyleF(style.ForPath)
}
