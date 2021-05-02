package gonsole

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

var (
	commandError = fmt.Sprintf("%s[Command Error]%s ", readline.RED, readline.RESET)
	parserError  = fmt.Sprintf("%s[Parser Error]%s ", readline.RED, readline.RESET)

	info     = fmt.Sprintf("%s[-]%s ", readline.BLUE, readline.RESET)   // Info - All normal messages
	warn     = fmt.Sprintf("%s[!]%s ", readline.YELLOW, readline.RESET) // Warn - Errors in parameters, notifiable events in modules/sessions
	errorStr = fmt.Sprintf("%s[!]%s ", readline.RED, readline.RESET)    // Error - Error in commands, filters, modules and implants.
	Success  = fmt.Sprintf("%s[*]%s ", readline.GREEN, readline.RESET)  // Success - Success events

	infof   = fmt.Sprintf("%s[-] ", readline.BLUE)   // Infof - formatted
	warnf   = fmt.Sprintf("%s[!] ", readline.YELLOW) // Warnf - formatted
	errorf  = fmt.Sprintf("%s[!] ", readline.RED)    // Errorf - formatted
	sucessf = fmt.Sprintf("%s[*] ", readline.GREEN)  // Sucessf - formatted
)

// commandParser - Both flags.Command and flags.Parser can add commands in the same way,
// we need to be able to call the appropriate target no matter the level of command nesting.
type commandParser interface {
	AddCommand(name, short, long string, data interface{}) (cmd *flags.Command, err error)
}

// SetParserOptions - Set the general options that apply to the root command parser.
// Default options are:
// -h, --h options are available to all registered commands.
// Ignored option dashes are ignored and passed along the command tree.
// This function might be used by people who forked this library, and
// do not care about --help options triggering helps, but instead want
// lower level access to how arguments are parsed
// or on when/why/how errors should be raised.
func (c *Console) SetParserOptions(options flags.Options) {
	c.parserOpts = options
	if c.current.parser != nil {
		c.current.parser.Options = options
	}
	return
}

// CommandParser - Returns the root command parser of the console.
// You can use it to find and query about the CURRENT Context commands, options, etc.
// The only limitation is simple: anything you change to these commands, options will
// NOT persist across execution loops, because commands are reinstantiated each time.
// Please the documentation wiki of this project, to see how to use this: can be very
// handy and powerful, even when programming your commands.
func (c *Console) CommandParser() (parser *flags.Parser) {
	return c.current.parser
}

// optionGroup - Used to generate global option structs, bound to commands/parsers.
type optionGroup struct {
	short     string
	long      string
	generator func() interface{}
}

// AddGlobalOptions - Global options are available in all child commands of this command
// (or all commands of the parser). The data interface is a struct declared the same way
// as you'd declare a go-flags parsable option struct.
func (c *Command) AddGlobalOptions(shortDescription, longDescription string, data func() interface{}) {
	optGroup := &optionGroup{
		short:     shortDescription,
		long:      longDescription,
		generator: data,
	}
	c.opts = append(c.opts, optGroup)
}
