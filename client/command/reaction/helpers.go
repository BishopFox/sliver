package reaction

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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/rsteube/carapace"
)

const (
	ReactionFileName = "reactions.json"
)

// GetReactionFilePath - Get the.
func GetReactionFilePath() string {
	return path.Join(assets.GetRootAppDir(), ReactionFileName)
}

// SaveReactions - Save the reactions to the reaction file.
func SaveReactions(reactions []core.Reaction) error {
	reactionFilePath := GetReactionFilePath()
	data, err := json.MarshalIndent(reactions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reactionFilePath, data, 0o600)
}

// LoadReactions - Save the reactions to the reaction file.
func LoadReactions() (int, error) {
	reactionFilePath := GetReactionFilePath()
	data, err := os.ReadFile(reactionFilePath)
	if err != nil {
		return 0, err
	}
	reactions := []core.Reaction{}
	err = json.Unmarshal(data, &reactions)
	if err != nil {
		return 0, err
	}
	for _, oldReaction := range core.Reactions.All() {
		core.Reactions.Remove(oldReaction.ID)
	}
	for _, reaction := range reactions {
		if !isReactable(reaction) {
			continue
		}
		core.Reactions.Add(reaction)
	}
	return len(reactions), nil
}

func isReactable(reaction core.Reaction) bool {
	for _, eventType := range core.ReactableEvents {
		if reaction.EventType == eventType {
			return true
		}
	}
	return false
}

// ReactionIDCompleter completes saved/available reaction IDs.
func ReactionIDCompleter(_ *console.SliverClient) carapace.Action {
	results := make([]string, 0)

	for _, reaction := range core.Reactions.All() {
		results = append(results, strconv.Itoa(reaction.ID))
		results = append(results, fmt.Sprintf("[%s] %s", reaction.EventType, strings.Join(reaction.Commands, ",")))
	}

	if len(results) == 0 {
		return carapace.ActionMessage("no reactions available")
	}

	return carapace.ActionValuesDescribed(results...).Tag("reactions")
}
