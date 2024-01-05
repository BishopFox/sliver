package generate

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
	"math"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// ProfilesCmd - Display implant profiles.
func ProfilesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	profiles := getImplantProfiles(con)
	if profiles == nil {
		return
	}
	if len(profiles) == 0 {
		con.PrintInfof("No profiles, see `%s %s help`\n", consts.ProfilesStr, consts.NewStr)
		return
	} else {
		PrintProfiles(profiles, con)
	}
}

// PrintProfiles - Print the profiles.
func PrintProfiles(profiles []*clientpb.ImplantProfile, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Profile Name",
		"Implant Type",
		"Platform",
		"Command & Control",
		"Debug",
		"Format",
		"Obfuscation",
		"Limitations",
		"C2 Profile",
		// "Nonce",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Profile Name", Mode: table.Asc},
	})

	for _, profile := range profiles {
		config := profile.Config

		obfuscation := "disabled"
		if config.ObfuscateSymbols {
			obfuscation = "enabled"
		}
		implantType := "session"
		if config.IsBeacon {
			implantType = "beacon"
		}
		c2URLs := []string{}
		for index, c2 := range config.C2 {
			c2URLs = append(c2URLs, fmt.Sprintf("[%d] %s", index+1, c2.URL))
		}
		tw.AppendRow(table.Row{
			profile.Name,
			implantType,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			strings.Join(c2URLs, "\n"),
			fmt.Sprintf("%v", config.Debug),
			fmt.Sprintf("%v", config.Format),
			obfuscation,
			getLimitsString(config),
			config.HTTPC2ConfigName,
			// profile.ImplantID,
		})
	}

	con.Printf("%s\n", tw.Render())
}

func getImplantProfiles(con *console.SliverClient) []*clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return pbProfiles.Profiles
}

// GetImplantProfileByName - Get an implant profile by a specific name.
func GetImplantProfileByName(name string, con *console.SliverClient) *clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	for _, profile := range pbProfiles.Profiles {
		if profile.Name == name {
			return profile
		}
	}
	return nil
}

func populateProfileProperties(config *clientpb.ImplantConfig) map[string]string {
	properties := make(map[string]string)

	var plural string

	properties["osarch"] = fmt.Sprintf("%s %s", strings.Title(config.GOOS), strings.ToUpper(config.GOARCH))
	if config.IsBeacon {
		properties["implanttype"] = "Beacon"
		jitter := int(config.BeaconJitter / int64(math.Pow10(9)))
		plural = "s"
		if jitter == 1 {
			plural = ""
		}
		properties["beaconjitter"] = fmt.Sprintf("%d second%s", jitter, plural)
		interval := int(config.BeaconInterval / int64(math.Pow10(9)))
		plural = "s"
		if interval == 1 {
			plural = ""
		}
		properties["beaconinterval"] = fmt.Sprintf("%d second%s", interval, plural)
	} else {
		properties["implanttype"] = "Session"
	}
	if config.Debug {
		properties["debugging"] = "enabled"
	} else {
		properties["debugging"] = "disabled"
	}

	if config.Evasion {
		properties["evasion"] = "enabled"
	} else {
		properties["evasion"] = "disabled"
	}

	if config.ObfuscateSymbols {
		properties["obsymbols"] = "enabled"
	} else {
		properties["obsymbols"] = "disabled"
	}

	if config.SGNEnabled {
		properties["sgn"] = "enabled"
	} else {
		properties["sgn"] = "disabled"
	}

	reconnect := int(config.ReconnectInterval / int64(math.Pow10(9)))
	if reconnect == 1 {
		plural = ""
	} else {
		plural = "s"
	}
	properties["reconnect"] = fmt.Sprintf("%d second%s", reconnect, plural)

	properties["maxerrors"] = fmt.Sprintf("%d", config.MaxConnectionErrors)

	poll := int(config.PollTimeout / int64(math.Pow10(9)))
	if poll == 1 {
		plural = ""
	} else {
		plural = "s"
	}
	properties["polltimeout"] = fmt.Sprintf("%d second%s", poll, plural)

	c2URLs := []string{}
	for index, c2 := range config.C2 {
		c2URLs = append(c2URLs, fmt.Sprintf("[%d] %s", index+1, c2.URL))
	}
	properties["implantC2"] = strings.Join(c2URLs, "\n")
	properties["canary"] = strings.Join(config.CanaryDomains, "\n")
	if config.ConnectionStrategy == "" {
		properties["connectstrat"] = "Sequential"
	} else if config.ConnectionStrategy == "r" {
		properties["connectstrat"] = "Random"
	} else if config.ConnectionStrategy == "rd" {
		properties["connectstrat"] = "Random Domain"
	} else {
		properties["connectstrat"] = config.ConnectionStrategy
	}
	properties["outputlimits"] = "n"
	if config.LimitDomainJoined {
		properties["limit-domjoin"] = "Yes"
		properties["outputlimits"] = "y"
	} else {
		properties["limit-domjoin"] = "No"
	}

	if config.LimitDatetime != "" {
		properties["limit-dt"] = config.LimitDatetime
		properties["outputlimits"] = "y"
	} else {
		properties["limit-dt"] = "No restriction"
	}

	if config.LimitFileExists != "" {
		properties["limit-file"] = config.LimitFileExists
		properties["outputlimits"] = "y"
	} else {
		properties["limit-file"] = "No restriction"
	}

	if config.LimitHostname != "" {
		properties["limit-host"] = config.LimitHostname
		properties["outputlimits"] = "y"
	} else {
		properties["limit-host"] = "No restriction"
	}

	if config.LimitLocale != "" {
		properties["limit-locale"] = config.LimitLocale
		properties["outputlimits"] = "y"
	} else {
		properties["limit-locale"] = "No restriction"
	}

	if config.LimitUsername != "" {
		properties["limit-user"] = config.LimitUsername
		properties["outputlimits"] = "y"
	} else {
		properties["limit-user"] = "No restriction"
	}

	switch config.Format {
	case clientpb.OutputFormat_EXECUTABLE:
		properties["format"] = "Executable"
	case clientpb.OutputFormat_SHARED_LIB:
		properties["format"] = "Shared Library"
	case clientpb.OutputFormat_SERVICE:
		properties["format"] = "Service"
	case clientpb.OutputFormat_SHELLCODE:
		properties["format"] = "Shellcode"
	case clientpb.OutputFormat_THIRD_PARTY:
		properties["format"] = "Third Party"
	}
	if config.TrafficEncodersEnabled {
		properties["trafficencoders"] = strings.Join(config.TrafficEncoders, "\n")
	}

	return properties
}

// PrintProfileInfo - Print detailed information about a given profile.
func PrintProfileInfo(name string, con *console.SliverClient) {
	profile := GetImplantProfileByName(name, con)
	if profile == nil {
		con.PrintErrorf("Could not find a profile with the name \"%s\"", name)
		return
	}

	config := profile.Config
	properties := populateProfileProperties(config)

	tw := table.NewWriter()

	// Implant Basics
	tw.AppendRow(table.Row{
		"OS / Architecture",
		properties["osarch"],
	})
	tw.AppendRow(table.Row{
		"Implant Type",
		properties["implanttype"],
	})
	tw.AppendRow(table.Row{
		"Implant Format",
		properties["format"],
	})

	con.PrintInfof("Implant Basics\n")
	con.Printf("%s\n\n", tw.Render())

	tw.ResetRows()
	// Obfuscation Options
	tw.AppendRow(table.Row{
		"Evasion is",
		properties["evasion"],
	})
	tw.AppendRow(table.Row{
		"Debugging is",
		properties["debugging"],
	})
	tw.AppendRow(table.Row{
		"Obfuscation of symbols is",
		properties["obsymbols"],
	})
	tw.AppendRow(table.Row{
		"Shikata Ga Nai (SGN) is",
		properties["sgn"],
	})

	con.PrintInfof("Obfuscation\n")
	con.Printf("%s\n\n", tw.Render())

	// Timeouts and Intervals
	tw.ResetRows()
	if config.IsBeacon {
		tw.AppendRow(table.Row{
			"Beacon Interval",
			properties["beaconinterval"],
		})
		tw.AppendRow(table.Row{
			"Beacon Jitter",
			properties["beaconjitter"],
		})
	}
	tw.AppendRow(table.Row{
		"Reconnect Interval",
		properties["reconnect"],
	})
	tw.AppendRow(table.Row{
		"Maximum Connection Errors",
		properties["maxerrors"],
	})
	tw.AppendRow(table.Row{
		"Poll Timeout",
		properties["polltimeout"],
	})

	con.PrintInfof("Timeouts and Intervals\n")
	con.Printf("%s\n\n", tw.Render())

	// C2
	tw.ResetRows()
	tw.AppendRow(table.Row{
		"Endpoints",
		properties["implantC2"],
	})
	if len(config.CanaryDomains) > 0 {
		plural := "s"
		if len(config.CanaryDomains) == 1 {
			plural = ""
		}
		tw.AppendRow(table.Row{
			fmt.Sprintf("Canary Domain%s", plural),
			properties["canary"],
		})
	}
	tw.AppendRow(table.Row{
		"Connection Strategy",
		properties["connectstrat"],
	})

	con.PrintInfof("Command and Control\n")
	con.Printf("%s\n\n", tw.Render())

	// Connection Restrictions
	if properties["outputlimits"] == "y" {
		tw.ResetRows()
		tw.AppendRow(table.Row{
			"Device must be domain joined",
			properties["limit-domjoin"],
		})
		tw.AppendRow(table.Row{
			"Execution will only occur before the following date/time",
			properties["limit-dt"],
		})
		tw.AppendRow(table.Row{
			"Files that must be present",
			properties["limit-file"],
		})
		tw.AppendRow(table.Row{
			"Device has the hostname",
			properties["limit-host"],
		})
		tw.AppendRow(table.Row{
			"Device uses the locale",
			properties["limit-locale"],
		})
		tw.AppendRow(table.Row{
			"The implant must be running under the context of specified user",
			properties["limit-user"],
		})

		con.PrintInfof("Execution is subject to the following restrictions\n")
		con.Printf("%s\n\n", tw.Render())
	}

	// Traffic encoders
	if config.TrafficEncodersEnabled {
		tw.ResetRows()
		tw.AppendRow(table.Row{
			"Traffic encoders",
			properties["trafficencoders"],
		})
		con.PrintInfof("Traffic Encoders\n")
		con.Printf("%s\n\n", tw.Render())
	}

	// Output messages that would otherwise get lost in between the tables
	if properties["outputlimits"] == "n" {
		con.PrintInfof("Execution is not subject to any restrictions\n")
	}

	if !config.TrafficEncodersEnabled {
		con.PrintInfof("Traffic encoders are not enabled\n")
	}
}

// ProfileNameCompleter - Completer for implant build names.
func ProfileNameCompleter(con *console.SliverClient) carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage(fmt.Sprintf("No profiles, err: %s", err.Error()))
		}

		if len(pbProfiles.Profiles) == 0 {
			return carapace.ActionMessage("No saved implant profiles")
		}

		results := []string{}
		sessions := []string{}

		for _, profile := range pbProfiles.Profiles {

			osArch := fmt.Sprintf("[%s/%s]", profile.Config.GOOS, profile.Config.GOARCH)
			buildFormat := profile.Config.Format.String()

			profileType := ""
			if profile.Config.IsBeacon {
				profileType = "(B)"
			} else {
				profileType = "(S)"
			}

			var domains []string
			for _, c2 := range profile.Config.C2 {
				domains = append(domains, c2.GetURL())
			}

			desc := fmt.Sprintf("%s %s %s %s", profileType, osArch, buildFormat, strings.Join(domains, ","))

			if profile.Config.IsBeacon {
				results = append(results, profile.Name)
				results = append(results, desc)
			} else {
				sessions = append(sessions, profile.Name)
				sessions = append(sessions, desc)
			}
		}

		return action.Invoke(ctx).Merge(
			carapace.ActionValuesDescribed(sessions...).Tag("sessions").Invoke(ctx),
			carapace.ActionValuesDescribed(results...).Tag("beacons").Invoke(ctx),
		).ToA()
	}

	return carapace.ActionCallback(comps)
}
