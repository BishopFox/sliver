package triggers

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

	------------------------------------------------------------------------

	target.go -- client-side target IP storage for trigger implants.

	The server has no knowledge of where trigger implants are deployed,
	so we maintain a simple JSON mapping of implant-name -> target-IP on
	the operator's machine at ~/.sliver-client/triggers.json. This is
	used by "trigger send <index> <intent>" to auto-populate the target.
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

const targetStoreFileName = "triggers.json"

// TargetStore holds the client-side mapping of trigger implant names
// to their deployed target IPs.
type TargetStore struct {
	// Targets maps implant name -> target IP/hostname.
	Targets map[string]string `json:"targets"`
}

// targetStorePath returns the absolute path to the triggers.json file.
func targetStorePath() string {
	return filepath.Join(assets.GetRootAppDir(), targetStoreFileName)
}

// LoadTargetStore reads the target store from disk. Returns an empty
// store (not an error) if the file doesn't exist.
func LoadTargetStore() (*TargetStore, error) {
	store := &TargetStore{Targets: make(map[string]string)}
	data, err := os.ReadFile(targetStorePath())
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, fmt.Errorf("read target store: %w", err)
	}
	if err := json.Unmarshal(data, store); err != nil {
		return store, fmt.Errorf("parse target store: %w", err)
	}
	if store.Targets == nil {
		store.Targets = make(map[string]string)
	}
	return store, nil
}

// Save writes the target store to disk.
func (s *TargetStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal target store: %w", err)
	}
	return os.WriteFile(targetStorePath(), data, 0600)
}

// TriggerTargetCmd associates a target IP with a trigger implant by
// index. Usage: triggers target <index> <ip>
func TriggerTargetCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) < 2 {
		con.PrintErrorf("usage: triggers target <index> <ip>\n")
		return
	}

	index, err := strconv.Atoi(args[0])
	if err != nil || index < 1 {
		con.PrintErrorf("invalid trigger index %q (must be a positive integer)\n", args[0])
		return
	}
	targetIP := args[1]
	if targetIP == "" {
		con.PrintErrorf("target IP cannot be empty\n")
		return
	}

	// Look up the trigger implant by index
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("failed to fetch implant builds: %s\n", err)
		return
	}

	triggerBuilds := sortedTriggerBuilds(builds)
	if len(triggerBuilds) == 0 {
		con.PrintErrorf("no trigger implants found\n")
		return
	}
	if index < 1 || index > len(triggerBuilds) {
		con.PrintErrorf("trigger index %d out of range (have %d trigger implants)\n", index, len(triggerBuilds))
		return
	}

	name := triggerBuilds[index-1].Name

	store, err := LoadTargetStore()
	if err != nil {
		con.PrintErrorf("failed to load target store: %s\n", err)
		return
	}

	store.Targets[name] = targetIP
	if err := store.Save(); err != nil {
		con.PrintErrorf("failed to save target store: %s\n", err)
		return
	}

	con.PrintInfof("Target for %q (index %d) set to %s\n", name, index, targetIP)
	con.PrintInfof("Stored in %s\n", targetStorePath())
}

// TriggerBuildEntry holds a trigger implant name and its config for
// display and lookup purposes.
type TriggerBuildEntry struct {
	Name   string
	Config *clientpb.ImplantConfig
}

// sortedTriggerBuilds filters and sorts implant builds that have
// IncludeTriggerWake=true, sorted alphabetically by name.
func sortedTriggerBuilds(builds *clientpb.ImplantBuilds) []TriggerBuildEntry {
	var entries []TriggerBuildEntry
	for name, config := range builds.Configs {
		if config.GetIncludeTriggerWake() {
			entries = append(entries, TriggerBuildEntry{Name: name, Config: config})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}
