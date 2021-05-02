package gonsole

import (
	"fmt"

	"github.com/jessevdk/go-flags"
)

// execute - The user has entered a command input line, the arguments
// have been processed: we synchronize a few elements of the console,
// then pass these arguments to the command parser for execution and error handling.
func (c *Console) execute(args []string) {

	// Asynchronous messages do not mess with the prompt from now on,
	// until end of execution. Once we are done executing the command,
	// they can again.
	c.mutex.RLock()
	c.isExecuting = true
	c.mutex.RUnlock()
	defer func() {
		c.mutex.RLock()
		c.isExecuting = false
		c.mutex.RUnlock()
	}()

	// Execute the command line, with the current menu' parser.
	_, err := c.current.parser.ParseArgs(args)

	// Process the errors raised by the parser.
	// A few of them are not really errors, and trigger some stuff.
	if err != nil {

		// Cast the error raised by the parser.
		parserErr, ok := err.(*flags.Error)
		if !ok {
			return
		}

		// If the command was not recognized and the current
		// menu has a user-specified unknown command handler, execute it.
		if parserErr.Type == flags.ErrUnknownCommand {
			if c.current.UnknownCommandHandler != nil {
				err = c.current.UnknownCommandHandler(args)
				if err != nil {
					fmt.Println(err.Error())
				}
				return
			}
			fmt.Println(commandError + parserErr.Error())
			return
		}

		// If the error type is a detected -h, --help flag, print custom help.
		if parserErr.Type == flags.ErrHelp {
			c.handleHelpFlag(args)
			return
		}

		// Else, we print the raw parser error
		fmt.Println(parserError + parserErr.Error())
	}

	return
}
