package printer_test

import (
	"fmt"

	"github.com/mmcloughlin/avo/printer"
)

func ExampleConfig_GeneratedBy() {
	// Default configuration named "avo".
	cfg := printer.NewDefaultConfig()
	fmt.Println(cfg.GeneratedBy())

	// Name can be customized.
	cfg = printer.Config{
		Name: "mildred",
	}
	fmt.Println(cfg.GeneratedBy())

	// Argv takes precedence.
	cfg = printer.Config{
		Argv: []string{"echo", "hello", "world"},
		Name: "mildred",
	}
	fmt.Println(cfg.GeneratedBy())

	// Output:
	// avo
	// mildred
	// command: echo hello world
}
