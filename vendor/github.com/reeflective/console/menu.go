package console

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/reeflective/readline"
	"github.com/spf13/cobra"
)

// Menu - A menu is a simple way to seggregate commands based on
// the environment to which they belong. For instance, when using a menu
// specific to some host/user, or domain of activity, commands will vary.
type Menu struct {
	name    string
	active  bool
	prompt  *Prompt
	console *Console

	// Maps interrupt signals (CtrlC/IOF, etc) to specific error handlers.
	interruptHandlers map[error]func(c *Console)

	// Input/output channels
	out *bytes.Buffer

	// The root cobra command/parser is the one returned by the handler provided
	// through the `menu.SetCommands()` function. This command is thus renewed after
	// each command invocation/execution.
	// You can still use it as you want, for instance to introspect the current command
	// state of your menu.
	*cobra.Command

	// Command spawner
	cmds Commands

	// History sources peculiar to this menu.
	historyNames []string
	histories    map[string]readline.History

	// Concurrency management
	mutex *sync.RWMutex
}

func newMenu(name string, console *Console) *Menu {
	menu := &Menu{
		console:           console,
		name:              name,
		prompt:            newPrompt(console),
		Command:           &cobra.Command{},
		out:               bytes.NewBuffer(nil),
		interruptHandlers: make(map[error]func(c *Console)),
		histories:         make(map[string]readline.History),
		mutex:             &sync.RWMutex{},
	}

	// Add a default in memory history to each menu
	// This source is dropped if another source is added
	// to the menu via `AddHistorySource()`.
	histName := menu.defaultHistoryName()
	hist := readline.NewInMemoryHistory()

	menu.historyNames = append(menu.historyNames, histName)
	menu.histories[histName] = hist

	return menu
}

// Name returns the name of this menu.
func (m *Menu) Name() string {
	return m.name
}

// Prompt returns the prompt object for this menu.
func (m *Menu) Prompt() *Prompt {
	return m.prompt
}

// AddHistorySource adds a source of history commands that will
// be accessible to the shell when the menu is active.
func (m *Menu) AddHistorySource(name string, source readline.History) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.histories) == 1 && m.historyNames[0] == m.defaultHistoryName() {
		delete(m.histories, m.defaultHistoryName())
		m.historyNames = make([]string, 0)
	}

	m.historyNames = append(m.historyNames, name)
	m.histories[name] = source
}

// AddHistorySourceFile adds a new source of history populated from and writing
// to the specified "filepath" parameter. On the first call to this function,
// the default in-memory history source is removed.
func (m *Menu) AddHistorySourceFile(name string, filepath string) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.histories) == 1 && m.historyNames[0] == m.defaultHistoryName() {
		delete(m.histories, m.defaultHistoryName())
		m.historyNames = make([]string, 0)
	}

	m.historyNames = append(m.historyNames, name)
	m.histories[name], _ = readline.NewHistoryFromFile(filepath)
}

// DeleteHistorySource removes a history source from the menu.
// This normally should only be used in two cases:
// - You want to replace the default in-memory history with another one.
// - You want to replace one of your history sources for some reason.
func (m *Menu) DeleteHistorySource(name string) {
	if name == m.Name() {
		if name != "" {
			name = " (" + name + ")"
		}

		name = fmt.Sprintf("local history%s", name)
	}

	delete(m.histories, name)

	for i, hname := range m.historyNames {
		if hname == name {
			m.historyNames = append(m.historyNames[:i], m.historyNames[i+1:]...)

			break
		}
	}
}

// TransientPrintf prints a message to the console, but only if the current
// menu is active. If the menu is not active, the message is buffered and will
// be printed the next time the menu is active.
//
// The message is printed as a transient message, meaning that it will be
// printed above the current prompt, effectively "pushing" the prompt down.
//
// If this function is called while a command is running, the console
// will simply print the log below the current line, and will not print
// the prompt. In any other case this function will work normally.
func (m *Menu) TransientPrintf(msg string, args ...any) (n int, err error) {
	n, err = fmt.Fprintf(m.out, msg, args...)
	if err != nil {
		return
	}

	if !m.active {
		fmt.Fprintf(m.out, "\n")
		return
	}

	buf := m.out.String()
	m.out.Reset()

	return m.console.TransientPrintf(buf)
}

// Printf prints a message to the console, but only if the current menu
// is active. If the menu is not active, the message is buffered and will
// be printed the next time the menu is active.
//
// Unlike TransientPrintf, this function will not print the message above
// the current prompt, but will instead print it below it.
//
// If this function is called while a command is running, the console
// will simply print the log below the current line, and will not print
// the prompt. In any other case this function will work normally.
func (m *Menu) Printf(msg string, args ...any) (n int, err error) {
	n, err = fmt.Fprintf(m.out, msg, args...)
	if err != nil {
		return
	}

	if !m.active {
		fmt.Fprintf(m.out, "\n")
		return
	}

	buf := m.out.String()
	m.out.Reset()

	return m.console.Printf(buf)
}

// resetPreRun is called before each new read line loop and before arbitrary RunCommand() calls.
// This function is responsible for resetting the menu state to a clean state, regenerating the
// menu commands, and ensuring that the correct prompt is bound to the shell.
func (m *Menu) resetPreRun() {
	m.console.mutex.RLock()
	defer m.console.mutex.RUnlock()

	// Menu setup
	m.resetCommands()              // Regenerate the commands for the menu.
	m.resetCmdOutput()             // Reset or adjust any buffered command output.
	m.prompt.bind(m.console.shell) // Prompt binding

	// Console-wide setup.
	m.console.ensureNoRootRunner()   // Avoid printing any help when the command line is empty
	m.console.hideFilteredCommands() // Hide commands that are not available
}

func (m *Menu) resetCmdOutput() {
	buf := strings.TrimSpace(m.out.String())

	// If our command has printed everything to stdout, nothing to do.
	if len(buf) == 0 || buf == "" {
		m.out.Reset()
		return
	}

	// Add two newlines to the end of the buffer, so that the
	// next command will be printed slightly below the current one.
	m.out.WriteString("\n")
}

func (m *Menu) resetCommands() {
	if m.cmds != nil {
		m.Command = m.cmds()
	}

	if m.Command == nil {
		m.Command = &cobra.Command{
			Annotations: make(map[string]string),
		}
	}

	m.SilenceUsage = true
}

func (m *Menu) defaultHistoryName() string {
	var name string

	if m.name != "" {
		name = " (" + m.name + ")"
	}

	return fmt.Sprintf("local history%s", name)
}
