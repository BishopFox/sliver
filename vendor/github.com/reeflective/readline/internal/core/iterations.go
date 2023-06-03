package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/reeflective/readline/internal/color"
)

// Iterations manages iterations for commands.
type Iterations struct {
	times   string // Stores iteration value
	active  bool   // Are we currently setting the iterations.
	pending bool   // Has the last command been an iteration one (vi-pending style)
}

// Add accepts a string to be converted as an integer representing
// the number of times some action should be performed.
// The times parameter can also be a negative sign, in which case
// the iterations value will be negative until those are reset.
func (i *Iterations) Add(times string) {
	if times == "" {
		return
	}

	// Never accept non-digit values.
	if _, err := strconv.Atoi(times); err != nil && times != "-" {
		return
	}

	i.active = true
	i.pending = true

	switch {
	case times == "-":
		i.times = times + i.times
	case strings.HasPrefix(times, "-"):
		i.times = "-" + i.times + strings.TrimPrefix(times, "-")
	default:
		i.times += times
	}
}

// Get returns the number of iterations (possibly
// negative), and resets the iterations to 1.
func (i *Iterations) Get() int {
	times, err := strconv.Atoi(i.times)

	// Any invalid value is still one time.
	if err != nil && strings.HasPrefix(i.times, "-") {
		times = -1
	} else if err != nil && times == 0 {
		times = 1
	} else if times == 0 && strings.HasPrefix(i.times, "-") {
		times = -1
	}

	// At least one iteration
	if times == 0 {
		times++
	}

	i.times = ""

	return times
}

// IsSet returns true if an iteration/numeric argument is active.
func (i *Iterations) IsSet() bool {
	return i.active
}

// IsPending returns true if the very last command executed was an
// iteration one. This is only meant for the main readline loop/run.
func (i *Iterations) IsPending() bool {
	return i.pending
}

// Reset resets the iterations (drops them).
func (i *Iterations) Reset() {
	i.times = ""
	i.active = false
	i.pending = false
}

// ResetPostRunIterations resets the iterations if the last command didn't set them.
// If the reset operated on active iterations, this function returns true.
func ResetPostRunIterations(iter *Iterations) (hint string) {
	if iter.pending {
		hint = color.Dim + fmt.Sprintf("(arg: %s)", iter.times)
	}

	if iter.pending {
		iter.pending = false

		return
	}

	iter.active = false

	return
}
