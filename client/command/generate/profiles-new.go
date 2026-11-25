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
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if err := checkNewProfileName(context.Background(), name, con); err != nil {
		con.PrintErrorf("%s\n", err)
		return
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
		reportSavedProfile(resp.Name, false, con)
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
	if err := checkNewProfileName(context.Background(), name, con); err != nil {
		con.PrintErrorf("%s\n", err)
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
		reportSavedProfile(resp.Name, true, con)
	}
}

func checkNewProfileName(ctx context.Context, name string, con *console.SliverClient) error {
	if name == "" {
		return nil // Preserve the legacy server's missing-name behavior.
	}
	doc, err := GetProfileDefinition(ctx, name, con)
	if status.Code(err) == codes.NotFound || status.Code(err) == codes.Unimplemented {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check existing profile %q: %w", name, err)
	}
	if doc == nil {
		return fmt.Errorf("check existing profile %q: server returned no profile status", name)
	}
	if !doc.Migrated {
		return fmt.Errorf("profile %q is an authored YAML profile; export and edit its YAML, then use profiles import", name)
	}
	return nil
}

func reportSavedProfile(name string, beacon bool, con *console.SliverClient) {
	label := "implant profile"
	if beacon {
		label = "implant profile (beacon)"
	}
	doc, err := GetProfileDefinition(context.Background(), name, con)
	if status.Code(err) == codes.Unimplemented {
		con.PrintInfof("Saved new %s %s (legacy server; canonical YAML unavailable)\n", label, name)
		return
	}
	if err != nil {
		con.PrintWarnf("Saved new %s %s, but could not verify canonical YAML: %s\n", label, name, err)
		return
	}
	if doc == nil {
		con.PrintWarnf("Saved new %s %s, but the server returned no profile status\n", label, name)
		return
	}
	if doc.MigrationError != "" {
		con.PrintWarnf("Saved new %s %s as a legacy profile; canonical YAML migration failed: %s\n",
			label, name, doc.MigrationError)
		return
	}
	if doc.YAML == "" || doc.FinalArtifact == nil {
		con.PrintWarnf("Saved new %s %s, but the server returned no canonical YAML definition\n", label, name)
		return
	}
	con.PrintInfof("Saved new %s %s with canonical YAML revision %d (%s/%s)\n",
		label, name, doc.Revision, doc.FinalArtifact.Kind, doc.FinalArtifact.Container)
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
