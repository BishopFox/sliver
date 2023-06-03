package generate

import (
	"context"
	"fmt"

	"github.com/rsteube/carapace"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// GetSliverBinary - Get the binary of an implant based on it's profile
func GetSliverBinary(profile *clientpb.ImplantProfile, con *console.SliverConsoleClient) ([]byte, error) {
	var data []byte
	// get implant builds
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return data, err
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
			return data, err
		}
		data = generated.GetFile().GetData()
		profile.Config.FileName = generated.File.Name
		_, err = con.Rpc.SaveImplantProfile(context.Background(), profile)
		if err != nil {
			con.PrintErrorf("Error updating implant profile\n")
			return data, err
		}
	} else {
		// Found a build, reuse that one
		con.PrintInfof("Sliver name for profile: %s\n", implantName)
		regenerate, err := con.Rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
			ImplantName: implantName,
		})
		if err != nil {
			return data, err
		}
		data = regenerate.GetFile().GetData()
	}
	return data, err
}

// FormatCompleter completes builds' architectures.
func ArchCompleter(con *console.SliverConsoleClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", err.Error())
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
	})
}

// FormatCompleter completes build operating systems
func OSCompleter(con *console.SliverConsoleClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		compiler, err := con.Rpc.GetCompiler(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("No compiler info: %s", err.Error())
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
	})
}

// FormatCompleter completes build formats
func FormatCompleter() carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		return carapace.ActionValues([]string{
			"exe", "shared", "service", "shellcode",
		}...).Tag("implant format")
	})
}

// TrafficEncoderCompleter - Completes the names of traffic encoders
func TrafficEncodersCompleter(con *console.SliverConsoleClient) carapace.Action {
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
