package console

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

// Start - Start the console application (readline loop). Blocking.
// The error returned will always be an error that the console
// application does not understand or cannot handle.
func (c *Console) Start() error {
	c.loadActiveHistories()

	// Print the console logo
	if c.printLogo != nil {
		c.printLogo(c)
	}

	for {
		// Identical to printing it at the end of the loop, and
		// leaves some space between the logo and the first prompt.
		if c.NewlineAfter {
			fmt.Println()
		}

		// Always ensure we work with the active menu, with freshly
		// generated commands, bound prompts and some other things.
		menu := c.activeMenu()
		menu.resetPreRun()

		c.printed = false

		if err := c.runAllE(c.PreReadlineHooks); err != nil {
			fmt.Printf("Pre-read error: %s\n", err.Error())
			continue
		}

		// Block and read user input.
		line, err := c.shell.Readline()

		if c.NewlineBefore {
			fmt.Println()
		}

		if err != nil {
			menu.handleInterrupt(err)
			continue
		}

		// Any call to the SwitchMenu() while we were reading user
		// input (through an interrupt handler) might have changed it,
		// so we must be sure we use the good one.
		menu = c.activeMenu()

		// Parse the line with bash-syntax, removing comments.
		args, err := c.parse(line)
		if err != nil {
			fmt.Printf("Parsing error: %s\n", err.Error())
			continue
		}

		if len(args) == 0 {
			continue
		}

		// Run user-provided pre-run line hooks,
		// which may modify the input line args.
		args, err = c.runLineHooks(args)
		if err != nil {
			fmt.Printf("Line error: %s\n", err.Error())
			continue
		}

		// Run all pre-run hooks and the command itself
		// Don't check the error: if its a cobra error,
		// the library user is responsible for setting
		// the cobra behavior.
		// If it's an interrupt, we take care of it.
		if err := c.execute(menu, args, false); err != nil {
			fmt.Println(err)
		}
	}
}

// RunCommandArgs is a convenience function to run a command line in a given menu.
// After running, the menu's commands are reset, and the prompts reloaded, therefore
// mostly mimicking the behavior that is the one of the normal readline/run/readline
// workflow.
// Although state segregation is a priority for this library to be ensured as much
// as possible, you should be cautious when using this function to run commands.
func (m *Menu) RunCommandArgs(args []string) (err error) {
	// The menu used and reset is the active menu.
	// Prepare its output buffer for the command.
	m.resetPreRun()

	// Run the command and associated helpers.
	return m.console.execute(m, args, !m.console.isExecuting)
}

// RunCommandLine is the equivalent of menu.RunCommandArgs(), but accepts
// an unsplit command line to execute. This line is split and processed in
// *sh-compliant form, identically to how lines are in normal console usage.
func (m *Menu) RunCommandLine(line string) (err error) {
	if len(line) == 0 {
		return
	}

	// Split the line into shell words.
	args, err := shellquote.Split(line)
	if err != nil {
		return fmt.Errorf("line error: %w", err)
	}

	return m.RunCommandArgs(args)
}

// execute - The user has entered a command input line, the arguments have been processed:
// we synchronize a few elements of the console, then pass these arguments to the command
// parser for execution and error handling.
// Our main object of interest is the menu's root command, and we explicitly use this reference
// instead of the menu itself, because if RunCommand() is asynchronously triggered while another
// command is running, the menu's root command will be overwritten.
func (c *Console) execute(menu *Menu, args []string, async bool) (err error) {
	if !async {
		c.mutex.RLock()
		c.isExecuting = true
		c.mutex.RUnlock()
	}

	defer func() {
		c.mutex.RLock()
		c.isExecuting = false
		c.mutex.RUnlock()
	}()

	// Our root command of interest, used throughout this function.
	cmd := menu.Command

	// Find the target command: if this command is filtered, don't run it.
	target, _, _ := cmd.Find(args)

	if err = menu.CheckIsAvailable(target); err != nil {
		return err
	}

	// Console-wide pre-run hooks, cannot.
	if err = c.runAllE(c.PreCmdRunHooks); err != nil {
		return fmt.Errorf("pre-run error: %s", err.Error())
	}

	// Assign those arguments to our parser.
	cmd.SetArgs(args)

	// The command execution should happen in a separate goroutine,
	// and should notify the main goroutine when it is done.
	ctx, cancel := context.WithCancelCause(context.Background())

	cmd.SetContext(ctx)

	// Start monitoring keyboard and OS signals.
	sigchan := c.monitorSignals()

	// And start the command execution.
	go c.executeCommand(cmd, cancel)

	// Wait for the command to finish, or for an OS signal to be caught.
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			err = ctx.Err()
		}

	case signal := <-sigchan:
		cancel(errors.New(signal.String()))
		menu.handleInterrupt(errors.New(signal.String()))
	}

	return err
}

// Run the command in a separate goroutine, and cancel the context when done.
func (c *Console) executeCommand(cmd *cobra.Command, cancel context.CancelCauseFunc) {
	if err := cmd.Execute(); err != nil {
		cancel(err)
		return
	}

	// And the post-run hooks in the same goroutine,
	// because they should not be skipped even if
	// the command is backgrounded by the user.
	if err := c.runAllE(c.PostCmdRunHooks); err != nil {
		cancel(err)
		return
	}

	// Command successfully executed, cancel the context.
	cancel(nil)
}

func (c *Console) loadActiveHistories() {
	c.shell.History.Delete()

	for _, name := range c.activeMenu().historyNames {
		c.shell.History.Add(name, c.activeMenu().histories[name])
	}
}

func (c *Console) runAllE(hooks []func() error) error {
	for _, hook := range hooks {
		if err := hook(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Console) runLineHooks(args []string) ([]string, error) {
	processed := args

	// Or modify them again
	for _, hook := range c.PreCmdRunLineHooks {
		var err error

		if processed, err = hook(processed); err != nil {
			return nil, err
		}
	}

	return processed, nil
}

// monitorSignals - Monitor the signals that can be sent to the process
// while a command is running. We want to be able to cancel the command.
func (c *Console) monitorSignals() <-chan os.Signal {
	sigchan := make(chan os.Signal, 1)

	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		// syscall.SIGKILL,
	)

	return sigchan
}
