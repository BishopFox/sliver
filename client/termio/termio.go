package termio

import "os"

var (
	interactiveInput  = os.Stdin
	interactiveOutput = os.Stdout
)

// InteractiveInput returns the process's original stdin handle before any
// console logging hooks swapped stdio to pipes.
func InteractiveInput() *os.File {
	return interactiveInput
}

// InteractiveOutput returns the process's original stdout handle before any
// console logging hooks swapped stdio to pipes.
func InteractiveOutput() *os.File {
	return interactiveOutput
}
