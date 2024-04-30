package common

import (
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

type Group struct {
	Cmd *cobra.Command
}

func (g Group) Tag() string {
	id := strings.ToLower(g.Cmd.GroupID)
	switch {
	case strings.HasSuffix(id, " commands"):
		return id
	case id != "":
		return id + " commands"
	case len(g.Cmd.Parent().Groups()) != 0:
		return "other commands"
	default:
		return "commands"
	}
}

func (g Group) Style() string {
	if g.Cmd.Parent() == nil || g.Cmd.Parent().Groups() == nil {
		return style.Default
	}

	for index, group := range g.Cmd.Parent().Groups() {
		if group.ID == g.Cmd.GroupID {
			return style.Carapace.Highlight(index)
		}
	}
	return style.Default
}
