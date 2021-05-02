package gonsole

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evilsocket/islazy/fs"
	"github.com/maxlandon/readline"
	"gopkg.in/AlecAivazis/survey.v1"
)

// AddConfigCommand - The console will add a command used to manage all elements of the console
// for any menu, and to save such elements into configurations, ready for export. You can
// choose both the command name and the group, for avoiding command collision with your owns.
func (c *Console) AddConfigCommand(name, group string) {
	c.configCommandName = name

	for _, cc := range c.menus {

		// Root
		conf := cc.AddCommand(name,
			"manage the console configuration elements and exports/imports",
			"",
			group,
			[]string{""},
			func() interface{} { return &config{console: c} })
		conf.SubcommandsOptional = true

		// Set values
		set := conf.AddCommand("set",
			"set elements of the console, stored in a configuration",
			"",
			"builtin",
			[]string{""},
			func() interface{} { return &configSet{console: c} })

		input := set.AddCommand("input",
			"set the input editing mode of the console (Vim/Emacs) (all menus)",
			"",
			"",
			[]string{""},
			func() interface{} { return &inputMode{console: c} })
		input.AddArgumentCompletion("Input", c.Completer.inputModes)

		hints := set.AddCommand("hints",
			"turn the console hints on/off (all menus)",
			"",
			"",
			[]string{""},
			func() interface{} { return &hintsDisplay{console: c} })
		hints.AddArgumentCompletion("Display", c.Completer.hints)

		set.AddCommand("max-tab-completer-rows",
			"set the maximum number of completion rows (all menus)",
			"",
			"",
			[]string{""},
			func() interface{} { return &maxTabCompleterRows{console: c} })

		prompt := set.AddCommand("prompt",
			"set right/left prompt strings (per menu, default is current)",
			"",
			"",
			[]string{""},
			func() interface{} { return &promptSet{console: c} })
		prompt.AddArgumentCompletionDynamic("Prompt", c.Completer.promptItems)
		prompt.AddOptionCompletion("Context", c.Completer.menus)

		multiline := set.AddCommand("prompt-multiline",
			"set/enable/disable multiline prompt strings for one of the available menus",
			"",
			"",
			[]string{""},
			func() interface{} { return &promptSetMultiline{console: c} })
		multiline.AddArgumentCompletionDynamic("Prompt", c.Completer.promptItems)

		highlight := set.AddCommand("highlight",
			"set the highlighting of tokens in the command line (all menus)",
			"",
			"",
			[]string{""},
			func() interface{} { return &highlightSyntax{console: c} })
		highlight.AddArgumentCompletion("Color", c.Completer.promptColors)
		highlight.AddArgumentCompletion("Token", c.Completer.highlightTokens)

		// Export configuration
		export := conf.AddCommand("export",
			"export the current console configuration as a JSON object in a file, or STDOUT",
			"",
			"builtin",
			[]string{""},
			func() interface{} { return &configExport{console: c} })
		export.AddOptionCompletionDynamic("Save", c.Completer.LocalPath)

	}
}

// AddConfigSubCommand - Allows the user to bind specialized subcommands to the config root command. This is useful if, for
// example, you want to save the console configuration on a remote server.
func (c *Console) AddConfigSubCommand(name, short, long, group string, filters []string, data func() interface{}) {
	for _, cc := range c.menus {
		for _, cmd := range cc.Commands() {
			if cmd.Name == c.configCommandName {
				cmd.AddCommand(name, short, long, group, filters, data)
			}
		}
	}
	return
}

// config - Manage console configuration. Prints current by default
type config struct {
	console *Console
}

// Execute - Manage console configuration. Prints current by default
func (c *config) Execute(args []string) (err error) {
	conf := c.console.config

	fmt.Println(readline.Bold(readline.Blue(" Console configuration\n")))

	// Elements applying to all menus.
	fmt.Println(readline.Yellow("Global"))

	var input string
	if conf.InputMode == InputVim {
		input = readline.Bold("Vim")
	} else {
		input = readline.Bold("Emacs")
	}
	pad := fmt.Sprintf("%-15s", "Input mode")
	fmt.Printf(" "+pad+"    %s%s%s\n", readline.BOLD, input, readline.RESET)
	pad = fmt.Sprintf("%-15s", "Console hints")
	fmt.Printf(" "+pad+"    %s%t%s\n", readline.BOLD, conf.Hints, readline.RESET)
	fmt.Println()

	// Print menu-specific configuration elements
	cc := c.console.current
	promptConf := conf.Prompts[cc.Name]

	fmt.Println(readline.Yellow(" " + cc.Name))

	pad = fmt.Sprintf("%-15s", "Prompt (left)")
	fmt.Printf(" "+pad+"    %s%s%s\n", readline.BOLD, promptConf.Left, readline.RESET)
	pad = fmt.Sprintf("%-15s", "Prompt (right)")
	fmt.Printf(" "+pad+"    %s%s%s\n", readline.BOLD, promptConf.Right, readline.RESET)
	pad = fmt.Sprintf("%-15s", "Multiline")
	fmt.Printf(" "+pad+"    %s%t%s\n", readline.BOLD, promptConf.Multiline, readline.RESET)
	pad = fmt.Sprintf("%-15s", "Multiline prompt")
	fmt.Printf(" "+pad+"    %s%s%s\n", readline.BOLD, promptConf.MultilinePrompt, readline.RESET)
	pad = fmt.Sprintf("%-15s", "Newline")
	fmt.Printf(" "+pad+"    %s%t%s\n", readline.BOLD, promptConf.NewlineAfter, readline.RESET)

	return
}

// configSet - Set configuration elements of the console
type configSet struct {
	console *Console
}

// Execute - Set configuration elements of the console
func (c *configSet) Execute(args []string) (err error) {
	return
}

// inputMode - Set the input editing mode of the console
type inputMode struct {
	Positional struct {
		Input string `description:"Input/editing mode"`
	} `positional-args:"true"`
	console *Console
}

// Execute - Set the input editing mode of the console
func (i *inputMode) Execute(args []string) (err error) {
	conf := i.console.config

	switch i.Positional.Input {
	case "vi", "vim":
		conf.InputMode = InputVim
		i.console.shell.InputMode = readline.Vim
	case "emacs":
		conf.InputMode = InputEmacs
		i.console.shell.InputMode = readline.Emacs
	default:
		fmt.Printf(errorStr+"Invalid argument: %s (must be 'vim'/'vi' or 'emacs')\n", i.Positional.Input)
	}
	fmt.Printf(info+"Console input mode: %s\n", readline.Yellow(i.Positional.Input))

	return
}

// hintsDisplay - Turn the hints on/off
type hintsDisplay struct {
	Positional struct {
		Display string `description:"show / hide command hints" required:"yes"`
	} `positional-args:"yes" required:"yes"`
	console *Console
}

// Execute - Turn the hints on/off
func (c *hintsDisplay) Execute(args []string) (err error) {
	conf := c.console.config

	switch c.Positional.Display {
	case "show", "on":
		conf.Hints = true
		c.console.shell.HintText = c.console.Completer.hintCompleter
		fmt.Printf(info+"Console hints: %s\n", readline.Yellow(c.Positional.Display))
	case "hide", "off":
		conf.Hints = false
		c.console.shell.HintText = nil
		fmt.Printf(info+"Console hints: %s\n", readline.Yellow(c.Positional.Display))
	default:
		fmt.Printf(errorStr+"Invalid argument: %s (must be 'hide'/'on' or 'show'/'off')\n", c.Positional.Display)
		return nil
	}
	return
}

// maxTabCompleterRows - Set the maximum number of completion rows
type maxTabCompleterRows struct {
	Positional struct {
		Rows int `description:"maximum number of completion rows to print" required:"yes"`
	} `positional-args:"yes" required:"yes"`
	console *Console
}

// Execute - Set the maximum number of completion rows
func (m *maxTabCompleterRows) Execute(args []string) (err error) {
	conf := m.console.config
	conf.MaxTabCompleterRows = m.Positional.Rows
	fmt.Printf(info+"Max tab completer rows: %d\n", m.Positional.Rows)
	return
}

// configExport - Export the current console configuration as a JSON object in a file.
type configExport struct {
	Options struct {
		Save   string `long:"save" short:"s" description:"path to save the configuration (default: working dir)"`
		Output bool   `long:"output" short:"o" description:"if set, only print the JSON config to STDOUT"`
	} `group:"export options"`
	console *Console
}

// Execute - Export the current console configuration as a JSON object in a file.
func (c *configExport) Execute(args []string) (err error) {
	conf := c.console.config

	// Pretty-print format marshaling
	configBytes, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		fmt.Printf(errorStr+"Error marshaling config to JSON: %s .\n", err.Error())
		return
	}

	// Print to STDOUT if asked
	if c.Options.Output {
		var config = &Config{}
		err = json.Unmarshal(configBytes, config)
		if err != nil {
			fmt.Printf(errorStr+"Failed to unmarshal config: %s\n", err.Error())
		} else {
			fmt.Println(string(configBytes))
		}
		return
	}

	// Else save to file
	save := c.Options.Save
	fullPath, _ := fs.Expand(save) // Get absolute path
	if save == "" {
		save, _ = os.Getwd()
	}
	shaID := md5.New()
	saveTo, err := saveLocation(fullPath, fmt.Sprintf("gonsole_%x.cfg", hex.EncodeToString(shaID.Sum([]byte{})[:5])))
	if err != nil {
		fmt.Printf(errorStr+"%s\n", err)
		return
	}
	err = ioutil.WriteFile(saveTo, configBytes, 0600)
	if err != nil {
		fmt.Printf(errorStr+"Failed to write to %s\n", err)
		return
	}
	fmt.Printf(errorStr+"Console configuration JSON saved to: %s\n", saveTo)

	return
}

func saveLocation(save, defaultName string) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		log.Printf(info+"%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			log.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0700)
			saveTo, _ = filepath.Abs(path.Join(saveTo, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			saveDir := filepath.Dir(save)
			_, err := os.Stat(saveTo)
			if os.IsNotExist(err) {
				os.MkdirAll(saveDir, 0700)
			}
			saveTo, _ = filepath.Abs(save)
		}
	} else {
		log.Printf("%s does exist\n", save)
		if fi.IsDir() {
			log.Printf("%s is dir\n", save)
			saveTo, _ = filepath.Abs(path.Join(save, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			prompt := &survey.Confirm{Message: "Overwrite existing file?"}
			var confirm bool
			survey.AskOne(prompt, &confirm, nil)
			if !confirm {
				return "", errors.New("File already exists")
			}
			saveTo, _ = filepath.Abs(save)
		}
	}
	return saveTo, nil
}

// highlightSyntax - Set the highlighting of tokens in the command line.
type highlightSyntax struct {
	Positional struct {
		Color string `description:"color to use for highlighting. Can be anything (some defaults colors/effects are completed)" required:"1-1"`
		Token string `description:"token (word type) to highlight with the given color (completed)" required:"1-1"`
	} `positional-args:"true" required:"yes"`
	console *Console
}

// Execute - Set the highlighting of tokens in the command line.
func (h *highlightSyntax) Execute(args []string) (err error) {
	for token := range h.console.config.Highlighting {
		if token == h.Positional.Token {
			h.console.config.Highlighting[token] = h.Positional.Color
		}
	}
	return
}

// promptSet - Set prompt strings for one of the available menus.
type promptSet struct {
	Positional struct {
		Prompt []string `description:"prompt string. Pass an empty '' to deactivate it (default colors/effect/items completed)" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Right         bool   `long:"right" short:"r" description:"apply changes to the right-side prompt"`
		Left          bool   `long:"left" short:"l" description:"apply changes to the left-side prompt"`
		NewlineBefore bool   `long:"newline-before" short:"b" description:"if true, a blank line is left before the prompt is printed"`
		NewlineAfter  bool   `long:"newline-after" short:"a" description:"if true, a blank line is left before the command output is printed"`
		Context       string `long:"menu" short:"c" description:"name of the menu for which to set the prompt (completed)" default:"current"`
	} `group:"export options"`
	console *Console
}

// Execute - Set prompt strings for one of the available menus.
func (p *promptSet) Execute(args []string) (err error) {
	if len(args) > 0 {
		fmt.Printf(warn+"Detected undesired remaining arguments: %s\n", readline.Bold(strings.Join(args, " ")))
		fmt.Printf("    Please use \\ dashes for each space in prompt string (input readline doesn't detect them)\n")
		fmt.Printf(readline.Yellow("    The current value has therefore not been saved.\n"))
		return
	}

	// By default, we only keep existing spaces, but any redundance will go away...
	prompt := strings.Join(p.Positional.Prompt, " ")

	var cc *Menu
	if p.Options.Context == "current" {
		cc = p.console.current
	} else {
		cc = p.console.GetMenu(p.Options.Context)
	}
	if cc == nil {
		fmt.Printf(errorStr+"Invalid menu/menu name: %s .\n", p.Options.Context)
		return
	}

	conf := p.console.config

	// Which prompt side did we set
	var side string
	if p.Options.Right {
		side = "(right)"
		cc.Prompt.right = prompt
		conf.Prompts[cc.Name].Right = prompt
	}
	if p.Options.Left {
		side = "(left)"
		cc.Prompt.left = prompt
		conf.Prompts[cc.Name].Left = prompt
	}
	if !p.Options.Left && !p.Options.Right {
		side = "(left)"
		cc.Prompt.left = prompt
		conf.Prompts[cc.Name].Left = prompt
	}

	// TODO: should be changed because not handy to use like this
	if p.Options.NewlineAfter {
		cc.Prompt.newline = true
		cc.console.PreOutputNewline = true
		conf.Prompts[cc.Name].NewlineAfter = true
	}

	if p.Options.NewlineBefore {
		p.console.LeaveNewline = true
		conf.Prompts[cc.Name].NewlineBefore = true
	}

	if prompt == "\"\"" || prompt == "''" {
		fmt.Printf(info + "Detected empty prompt string: deactivating the corresponding prompt.\n")
		return
	}

	fmt.Printf(info+"Server prompt %s : %s\n", side, readline.Bold(prompt))
	return
}

// promptSetMultiline - Set multiline prompt strings for one of the available menus.
type promptSetMultiline struct {
	Positional struct {
		Prompt string `description:"multine prompt string"`
	} `positional-args:"yes"`
	Options struct {
		Enable  bool   `long:"enable" short:"e" description:"if true, the prompt will be a 2-line prompt"`
		Disable bool   `long:"disable" short:"d" description:"if true, disable the multiline prompt"`
		Context string `long:"menu" short:"c" description:"name of the menu for which to set the prompt (completed)" default:"current"`
	} `group:"export options"`
	console *Console
}

// Execute - Set multiline prompt strings for one of the available menus.
func (p *promptSetMultiline) Execute(args []string) (err error) {
	if len(args) > 0 {
		fmt.Printf(warn+"Detected undesired remaining arguments: %s\n", readline.Bold(strings.Join(args, " ")))
		fmt.Printf("    Please use \\ dashes for each space in prompt string (input readline doesn't detect them)\n")
		fmt.Printf(readline.Yellow("    The current value has therefore not been saved.\n"))
		return
	}

	var cc *Menu
	if p.Options.Context == "current" {
		cc = p.console.current
	} else {
		cc = p.console.GetMenu(p.Options.Context)
	}
	if cc == nil {
		fmt.Printf(errorStr+"Invalid menu/menu name: %s .\n", p.Options.Context)
		return
	}

	conf := p.console.config

	if p.Positional.Prompt != "" {
		p.console.shell.MultilinePrompt = p.Positional.Prompt
		conf.Prompts[cc.Name].MultilinePrompt = p.Positional.Prompt
		fmt.Printf(info+"Setting raw multiline prompt to: %s", p.Positional.Prompt)
	}

	if p.Options.Enable {
		p.console.shell.Multiline = true
		conf.Prompts[cc.Name].Multiline = true
		return
	}
	if p.Options.Disable {
		p.console.shell.Multiline = false
		conf.Prompts[cc.Name].Multiline = false
		return
	}

	return
}
