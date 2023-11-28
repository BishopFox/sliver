package console

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	// CommandFilterKey should be used as a key to in a cobra.Annotation map.
	// The value will be used as a filter to disable commands when the console
	// calls the Filter("name") method on the console.
	// The string value will be comma-splitted, with each split being a filter.
	CommandFilterKey = "console-hidden"
)

// Commands is a simple function a root cobra command containing an arbitrary tree
// of subcommands, along with any behavior parameters normally found in cobra.
// This function is used by each menu to produce a new, blank command tree after
// each execution run, as well as each command completion invocation.
type Commands func() *cobra.Command

// SetCommands requires a function returning a tree of cobra commands to be used.
func (m *Menu) SetCommands(cmds Commands) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.cmds = cmds
}

// HideCommands - Commands, in addition to their menus, can be shown/hidden based
// on a filter string. For example, some commands applying to a Windows host might
// be scattered around different groups, but, having all the filter "windows".
// If "windows" is used as the argument here, all windows commands for the current
// menu are subsequently hidden, until ShowCommands("windows") is called.
func (c *Console) HideCommands(filters ...string) {
next:
	for _, filt := range filters {
		for _, filter := range c.filters {
			if filt == filter {
				continue next
			}
		}
		if filt != "" {
			c.filters = append(c.filters, filt)
		}
	}
}

// ShowCommands - Commands, in addition to their menus, can be shown/hidden based
// on a filter string. For example, some commands applying to a Windows host might
// be scattered around different groups, but, having all the filter "windows".
// Use this function if you have previously called HideCommands("filter") and want
// these commands to be available back under their respective menu.
func (c *Console) ShowCommands(filters ...string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	updated := make([]string, 0)

	if len(filters) == 0 {
		c.filters = updated

		return
	}

next:
	for _, filt := range c.filters {
		for _, filter := range filters {
			if filt == filter {
				continue next
			}
		}
		updated = append(updated, filt)
	}

	c.filters = updated
}

func (c *Console) isFiltered(cmd *cobra.Command) bool {
	if cmd.Annotations == nil {
		return false
	}

	// Get the filters on the command
	filterStr := cmd.Annotations[CommandFilterKey]
	filters := strings.Split(filterStr, ",")

	for _, cmdFilter := range filters {
		for _, filter := range c.filters {
			if cmdFilter != "" && cmdFilter == filter {
				return true
			}
		}
	}

	return false
}

// hide commands that are filtered so that they are not
// shown in the help strings or proposed as completions.
func (c *Console) hideFilteredCommands() {
	for _, cmd := range c.activeMenu().Commands() {
		// Don't override commands if they are already hidden
		if cmd.Hidden {
			continue
		}

		if c.isFiltered(cmd) {
			cmd.Hidden = true
		}
	}
}
