package jobs

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// WebsiteNameCompleter completes the names of available websites.
func WebsiteNameCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		websites, err := websiteNameValues(con)
		if err != nil {
			return carapace.ActionMessage("Failed to list websites %s", err)
		}
		if len(websites) == 0 {
			return carapace.ActionMessage("no available websites")
		}

		return carapace.ActionValues(websites...).Tag("websites").Usage("website name")
	})
}

func registerWebsiteFlagCompletion(cmd *cobra.Command, name string, con *console.SliverClient) {
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
		values, err := websiteNameValues(con)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return filterCompletionValues(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func websiteNameValues(con *console.SliverClient) ([]string, error) {
	results := make([]string, 0)
	if con == nil || con.Rpc == nil {
		return results, nil
	}

	websites, err := con.Rpc.Websites(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, ws := range websites.Websites {
		results = append(results, ws.Name)
	}
	sort.Strings(results)
	return results, nil
}

func filterCompletionValues(values []string, prefix string) []string {
	if prefix == "" {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		candidate := value
		if tab := strings.IndexByte(value, '\t'); tab >= 0 {
			candidate = value[:tab]
		}
		if strings.HasPrefix(candidate, prefix) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}
