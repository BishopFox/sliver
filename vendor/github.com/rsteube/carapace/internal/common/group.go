package common

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
)

type Group struct {
	Cmd *cobra.Command
}

func (g Group) Tag() string {
	tag := "commands"
	if id := g.Cmd.GroupID; id != "" {
		if strings.HasSuffix(id, "commands") {
			tag = id
		} else {
			tag = fmt.Sprintf("%v %v", id, tag)
		}
	} else if len(g.Cmd.Parent().Groups()) != 0 {
		tag = "other commands"
	}
	return tag
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
