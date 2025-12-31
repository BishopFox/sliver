package generate

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesNewCmd - Create a new implant profile.
func ProfilesNewCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if shouldRunGenerateProfilesNewForm(cmd, con, args) {
		result, err := forms.GenerateProfilesNewForm()
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Profiles new form failed: %s\n", err)
			return
		}
		profileName, err := applyGenerateProfilesNewForm(cmd, result)
		if err != nil {
			con.PrintErrorf("Failed to apply profiles new form values: %s\n", err)
			return
		}
		name = profileName
	} else if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	_, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := con.Rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Saved new implant profile %s\n", resp.Name)
	}
}

// ProfilesNewBeaconCmd - Create a new beacon profile.
func ProfilesNewBeaconCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if shouldRunGenerateProfilesNewBeaconForm(cmd, con, args) {
		result, err := forms.GenerateProfilesNewBeaconForm()
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Profiles new beacon form failed: %s\n", err)
			return
		}
		profileName, err := applyGenerateProfilesNewBeaconForm(cmd, result)
		if err != nil {
			con.PrintErrorf("Failed to apply profiles new beacon form values: %s\n", err)
			return
		}
		name = profileName
	} else if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}
	_, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	config.IsBeacon = true
	err := parseBeaconFlags(cmd, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := con.Rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Saved new implant profile (beacon) %s\n", resp.Name)
	}
}

func shouldRunGenerateProfilesNewForm(cmd *cobra.Command, con *console.SliverClient, args []string) bool {
	if con == nil || con.IsCLI {
		return false
	}
	if len(args) != 0 {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func applyGenerateProfilesNewForm(cmd *cobra.Command, result *forms.GenerateProfilesNewFormResult) (string, error) {
	if err := cmd.Flags().Set("os", result.OS); err != nil {
		return "", err
	}
	if err := cmd.Flags().Set("arch", result.Arch); err != nil {
		return "", err
	}
	if err := cmd.Flags().Set("format", result.Format); err != nil {
		return "", err
	}

	profileName := strings.TrimSpace(result.ProfileName)
	if profileName == "" {
		return "", errors.New("profile name required")
	}

	implantName := strings.TrimSpace(result.Name)
	if implantName != "" {
		if err := cmd.Flags().Set("name", implantName); err != nil {
			return "", err
		}
	}

	c2Value := strings.TrimSpace(result.C2Value)
	switch result.C2Type {
	case "mtls":
		if err := cmd.Flags().Set("mtls", c2Value); err != nil {
			return "", err
		}
	case "wg":
		if err := cmd.Flags().Set("wg", c2Value); err != nil {
			return "", err
		}
	case "http":
		if err := cmd.Flags().Set("http", c2Value); err != nil {
			return "", err
		}
	case "dns":
		if err := cmd.Flags().Set("dns", c2Value); err != nil {
			return "", err
		}
	case "named-pipe":
		if err := cmd.Flags().Set("named-pipe", c2Value); err != nil {
			return "", err
		}
	case "tcp-pivot":
		if err := cmd.Flags().Set("tcp-pivot", c2Value); err != nil {
			return "", err
		}
	default:
		return "", errors.New("unsupported C2 transport selection")
	}

	return profileName, nil
}

func shouldRunGenerateProfilesNewBeaconForm(cmd *cobra.Command, con *console.SliverClient, args []string) bool {
	if con == nil || con.IsCLI {
		return false
	}
	if len(args) != 0 {
		return false
	}
	return cmd.Flags().NFlag() == 0
}

func applyGenerateProfilesNewBeaconForm(cmd *cobra.Command, result *forms.GenerateProfilesNewBeaconFormResult) (string, error) {
	if err := cmd.Flags().Set("os", result.OS); err != nil {
		return "", err
	}
	if err := cmd.Flags().Set("arch", result.Arch); err != nil {
		return "", err
	}
	if err := cmd.Flags().Set("format", result.Format); err != nil {
		return "", err
	}

	profileName := strings.TrimSpace(result.ProfileName)
	if profileName == "" {
		return "", errors.New("profile name required")
	}

	implantName := strings.TrimSpace(result.Name)
	if implantName != "" {
		if err := cmd.Flags().Set("name", implantName); err != nil {
			return "", err
		}
	}

	c2Value := strings.TrimSpace(result.C2Value)
	switch result.C2Type {
	case "mtls":
		if err := cmd.Flags().Set("mtls", c2Value); err != nil {
			return "", err
		}
	case "wg":
		if err := cmd.Flags().Set("wg", c2Value); err != nil {
			return "", err
		}
	case "http":
		if err := cmd.Flags().Set("http", c2Value); err != nil {
			return "", err
		}
	case "dns":
		if err := cmd.Flags().Set("dns", c2Value); err != nil {
			return "", err
		}
	case "named-pipe":
		if err := cmd.Flags().Set("named-pipe", c2Value); err != nil {
			return "", err
		}
	case "tcp-pivot":
		if err := cmd.Flags().Set("tcp-pivot", c2Value); err != nil {
			return "", err
		}
	default:
		return "", errors.New("unsupported C2 transport selection")
	}

	if err := setOptionalInt64Flag(cmd, "days", result.Days); err != nil {
		return "", err
	}
	if err := setOptionalInt64Flag(cmd, "hours", result.Hours); err != nil {
		return "", err
	}
	if err := setOptionalInt64Flag(cmd, "minutes", result.Minutes); err != nil {
		return "", err
	}
	if err := setOptionalInt64Flag(cmd, "seconds", result.Seconds); err != nil {
		return "", err
	}
	if err := setOptionalInt64Flag(cmd, "jitter", result.Jitter); err != nil {
		return "", err
	}

	return profileName, nil
}
