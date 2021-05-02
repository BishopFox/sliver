package gonsole

import (
	"github.com/jessevdk/go-flags"
)

// commandGroup - A group of commands, which might be by any motive: common domain,
// type, etc, as long as the group name is the same.
type commandGroup struct {
	Name string
	cmds []*Command
}

// GetCommandGroup - Get the group for a command.
func (c *Console) GetCommandGroup(cmd *flags.Command) string {

	// Sliver commands are searched for if we are in this menu
	for _, group := range c.current.cmd.groups {
		for _, c := range group.cmds {
			if c.Name == cmd.Name {
				// We don't return the name if the command is not generated
				if c.cmd != nil {
					return group.Name
				}
			}
		}
	}
	return ""
}

func (c *Console) bindCommandGroup(parent commandParser, grp *commandGroup) {

	// For each command in the group, yield a flags.Command
	for _, cmd := range grp.cmds {
		var skipHidden bool
	nextCommand:
		for _, cmdFilter := range cmd.Filters {
			// Do not generate the command if one of its
			// filters is active.
			for _, filter := range c.filters {
				if filter == cmdFilter {
					skipHidden = true
					continue nextCommand
				}
			}
			// Else, generate
			cmd.cmd = cmd.generator(parent, cmd.SubcommandsOptional)
		}

		// Bind any subcommands of this cmd if it was registered
		if skipHidden {
			skipHidden = false
		} else {
			for _, subgroup := range cmd.groups {
				c.bindCommandGroup(cmd.cmd, subgroup)
			}
		}
	}
}
