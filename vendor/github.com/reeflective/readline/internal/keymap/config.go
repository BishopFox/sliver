package keymap

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/reeflective/readline/inputrc"
)

// readline global options specific to this library.
var readlineOptions = map[string]interface{}{
	// General edition
	"autopairs": false,

	// Completion
	"autocomplete":               false,
	"completion-list-separator":  "--",
	"completion-selection-style": "\x1b[1;30m",

	// Prompt & General UI
	"transient-prompt":          false,
	"usage-hint-always":         false,
	"history-autosuggest":       false,
	"multiline-column":          true,
	"multiline-column-numbered": false,
}

// ReloadConfig parses all valid .inputrc configurations and immediately
// updates/reloads all related settings (editing mode, variables behavior, etc.)
func (m *Engine) ReloadConfig(opts ...inputrc.Option) (err error) {
	// Builtin Go binds (in addition to default readline binds)
	m.loadBuiltinOptions()
	m.loadBuiltinBinds()

	user, _ := user.Current()

	// Parse library-specific configurations.
	//
	// This library implements various additional commands and keymaps.
	// Parse the configuration with a specific App name, ignoring errors.
	inputrc.UserDefault(user, m.config, inputrc.WithApp("go"))

	// Parse user configurations.
	//
	// Those default settings are the base options often needed
	// by /etc/inputrc on various Linux distros (for special keys).
	defaults := []inputrc.Option{
		inputrc.WithMode("emacs"),
		inputrc.WithTerm(os.Getenv("TERM")),
	}

	opts = append(defaults, opts...)

	// This will only overwrite binds that have been
	// set in those configs, and leave the default ones
	// (those just set above), so as to keep most of the
	// default functionality working out of the box.
	err = inputrc.UserDefault(user, m.config, opts...)
	if err != nil {
		return err
	}

	// Some configuration variables might have an
	// effect on our various keymaps and bindings.
	m.overrideBindsSpecial()

	// Startup editing mode
	switch m.config.GetString("editing-mode") {
	case "emacs":
		m.main = Emacs
	case "vi":
		m.main = ViInsert
	}

	return nil
}

// loadBuiltinOptions loads some options specific to
// this library, if they are not loaded already.
func (m *Engine) loadBuiltinOptions() {
	for name, value := range readlineOptions {
		if val := m.config.Get(name); val == nil {
			m.config.Set(name, value)
		}
	}
}

// loadBuiltinBinds adds additional command mappings that are not part
// of the standard C readline configuration: those binds therefore can
// reference commands or keymaps only implemented/used in this library.
func (m *Engine) loadBuiltinBinds() {
	// Emacs specials
	for seq, bind := range emacsKeys {
		m.config.Binds[string(Emacs)][seq] = bind
	}

	// Vim main keymaps
	for seq, bind := range vicmdKeys {
		m.config.Binds[string(ViCommand)][seq] = bind
		m.config.Binds[string(ViMove)][seq] = bind
		m.config.Binds[string(Vi)][seq] = bind
	}

	for seq, bind := range viinsKeys {
		m.config.Binds[string(ViInsert)][seq] = bind
	}

	// Vim local keymaps
	m.config.Binds[string(Visual)] = visualKeys
	m.config.Binds[string(ViOpp)] = vioppKeys
	m.config.Binds[string(MenuSelect)] = menuselectKeys
	m.config.Binds[string(Isearch)] = menuselectKeys

	// Default TTY binds
	for _, keymap := range m.config.Binds {
		keymap[inputrc.Unescape(`\C-C`)] = inputrc.Bind{Action: "abort"}
	}
}

// overrideBindsSpecial overwrites some binds as dictated by the configuration variables.
func (m *Engine) overrideBindsSpecial() {
	// Disable completion functions if required
	if m.config.GetBool("disable-completion") {
		for _, keymap := range m.config.Binds {
			for seq, bind := range keymap {
				switch bind.Action {
				case "complete", "menu-complete", "possible-completions":
					keymap[seq] = inputrc.Bind{Action: "self-insert"}
				}
			}
		}
	}
}

func printBindsReadable(commands []string, all map[string][]string) {
	for _, command := range commands {
		commandBinds := all[command]
		sort.Strings(commandBinds)

		switch {
		case len(commandBinds) == 0:
		case len(commandBinds) > 5:
			var firstBinds []string

			for i := 0; i < 5; i++ {
				firstBinds = append(firstBinds, "\""+commandBinds[i]+"\"")
			}

			bindsStr := strings.Join(firstBinds, ", ")
			fmt.Printf("%s can be found on %s ...\n", command, bindsStr)

		default:
			var firstBinds []string

			for _, bind := range commandBinds {
				firstBinds = append(firstBinds, "\""+bind+"\"")
			}

			bindsStr := strings.Join(firstBinds, ", ")
			fmt.Printf("%s can be found on %s\n", command, bindsStr)
		}
	}
}

func printBindsInputrc(commands []string, all map[string][]string) {
	for _, command := range commands {
		commandBinds := all[command]
		sort.Strings(commandBinds)

		if len(commandBinds) > 0 {
			for _, bind := range commandBinds {
				fmt.Printf("\"%s\": %s\n", bind, command)
			}
		}
	}
}
