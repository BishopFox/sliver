package commands

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
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/evilsocket/islazy/tui"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// NewProfile - Configure and save a new implant profile.
type NewProfile struct {
	StageOptions // This commands works the same as generate, and needs full options.
}

// Execute - Configure and save a new implant profile.
func (p *NewProfile) Execute(args []string) (err error) {

	name := p.CoreOptions.Profile
	if name == "" {
		fmt.Printf(util.Error + "Invalid profile name\n")
		return
	}

	config := parseCompileFlags(p.StageOptions)
	if config == nil {
		return
	}

	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := transport.RPC.SaveImplantProfile(context.Background(), profile)

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"Saved new profile %s\n", resp.Name)
	}

	return
}

// Profiles - List saved implant profiles.
type Profiles struct{}

// Execute - List saved implant profiles.
func (p *Profiles) Execute(args []string) (err error) {
	profiles := getSliverProfiles()
	if profiles == nil {
		return
	}
	if len(*profiles) == 0 {
		fmt.Printf(util.Info+"No profiles, create one with `%s`\n", constants.NewProfileStr)
		return
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Name\tPlatform\tCommand & Control\tDebug\tFormat\tObfuscation\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("Command & Control")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Format")),
		strings.Repeat("=", len("Obfuscation")),
		strings.Repeat("=", len("Limitations")))

	for name, profile := range *profiles {
		config := profile.Config
		if 0 < len(config.C2) {
			obfuscation := "strings only"
			if config.ObfuscateSymbols {
				obfuscation = "symbols obfuscation"
			}
			if config.Debug {
				obfuscation = "none"
			}
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				name,
				fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
				fmt.Sprintf("[1] %s", config.C2[0].URL),
				fmt.Sprintf("%v", config.Debug),
				fmt.Sprintf("%v", config.Format),
				fmt.Sprintf("%s", obfuscation),
				getLimitsString(config),
			)
		}
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
					"",
					"",
					"",
					"",
				)
			}
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "", "", "", "", "", "", "")
	}
	table.Flush()
	return
}

// ProfileGenerate - Generate implant from a profile given as argment (completed)
type ProfileGenerate struct {
	Positional struct {
		Profile string `description:"name of profile to use"`
	} `required:"true"`
	Save string `long:"save" short:"s" description:"directory/file where to save binary"`
}

// Execute - Generate implant from a profile given as argment (completed)
func (p *ProfileGenerate) Execute(args []string) (err error) {
	name := p.Positional.Profile
	save := p.Save
	if save == "" {
		save, _ = os.Getwd()
	}
	profiles := getSliverProfiles()
	if profile, ok := (*profiles)[name]; ok {
		implantFile, err := compile(profile.Config, save)
		if err != nil {
			return err
		}
		profile.Config.Name = buildImplantName(implantFile.Name)
		_, err = transport.RPC.SaveImplantProfile(context.Background(), profile)
		if err != nil {
			fmt.Printf(util.Error+"could not update implant profile: %v\n", err)
			return err
		}
	} else {
		fmt.Printf(util.Error+"No profile with name '%s'", name)
	}
	return
}

func getSliverProfiles() *map[string]*clientpb.ImplantProfile {
	pbProfiles, err := transport.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"Error %s", err)
		return nil
	}
	profiles := &map[string]*clientpb.ImplantProfile{}
	for _, profile := range pbProfiles.Profiles {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}

func getLimitsString(config *clientpb.ImplantConfig) string {
	limits := []string{}
	if config.LimitDatetime != "" {
		limits = append(limits, fmt.Sprintf("datetime=%s", config.LimitDatetime))
	}
	if config.LimitDomainJoined {
		limits = append(limits, fmt.Sprintf("domainjoined=%v", config.LimitDomainJoined))
	}
	if config.LimitUsername != "" {
		limits = append(limits, fmt.Sprintf("username=%s", config.LimitUsername))
	}
	if config.LimitHostname != "" {
		limits = append(limits, fmt.Sprintf("hostname=%s", config.LimitHostname))
	}
	if config.LimitFileExists != "" {
		limits = append(limits, fmt.Sprintf("fileexists=%s", config.LimitFileExists))
	}
	return strings.Join(limits, "; ")
}

// ProfileDelete - Delete one or more profiles from the server
type ProfileDelete struct {
	Positional struct {
		Profiles []string `description:"name of profile to delete" required:"1"`
	} `positional-args:"yes" required:"true"`
}

// Execute - Command
func (pd *ProfileDelete) Execute(args []string) (err error) {
	for _, p := range pd.Positional.Profiles {
		_, err := transport.RPC.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
			Name: p,
		})
		if err != nil {
			fmt.Printf(util.Warn+"Failed to delete profile %s\n", err)
			continue
		} else {
			fmt.Printf(util.Info+"Delete profile %s\n", tui.Bold(p))
		}
	}
	return
}
