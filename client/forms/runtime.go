package forms

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/bishopfox/sliver/client/termio"
	"golang.org/x/term"
)

var openFormTTY = tea.OpenTTY

func runForm(form *huh.Form) error {
	cleanup := configureFormTTY(form)
	if cleanup != nil {
		defer cleanup()
	}
	return form.Run()
}

// Console logging swaps stdout/stderr to pipes so the session can be tee'd to
// disk. Bubble Tea forms need a real TTY output, so temporarily bind them back
// to the controlling terminal when the interactive console has piped stdio.
func configureFormTTY(form *huh.Form) func() {
	if form == nil {
		return nil
	}

	stdinTTY := isTTY(os.Stdin)
	stdoutTTY := isTTY(os.Stdout)
	stderrTTY := isTTY(os.Stderr)

	switch {
	case shouldUseStdoutTTYForForm(stdinTTY, stdoutTTY, stderrTTY):
		form.WithInput(os.Stdin)
		form.WithOutput(os.Stdout)
		return nil

	case needsDedicatedTTYForForm(stdinTTY, stdoutTTY, stderrTTY):
		if isTTY(termio.InteractiveInput()) && isTTY(termio.InteractiveOutput()) {
			form.WithInput(termio.InteractiveInput())
			form.WithOutput(termio.InteractiveOutput())
			return nil
		}

		inTTY, outTTY, err := openFormTTY()
		if err != nil {
			return nil
		}

		form.WithInput(inTTY)
		form.WithOutput(outTTY)
		return func() {
			_ = inTTY.Close()
			if outTTY != inTTY {
				_ = outTTY.Close()
			}
		}

	default:
		return nil
	}
}

func isTTY(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}

func shouldUseStdoutTTYForForm(stdinTTY, stdoutTTY, stderrTTY bool) bool {
	return stdinTTY && stdoutTTY && !stderrTTY
}

func needsDedicatedTTYForForm(stdinTTY, stdoutTTY, stderrTTY bool) bool {
	return stdinTTY && !stdoutTTY && !stderrTTY
}
