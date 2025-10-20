package completion

import (
	"fmt"
	"os"

	"github.com/carapace-sh/carapace/pkg/style"
	"github.com/carapace-sh/carapace/pkg/xdg"
)

// DefaultStyleConfig sets some default styles for completion.
func DefaultStyleConfig() {
	// If carapace config file is found, just return.
	if dir, err := xdg.UserConfigDir(); err == nil {
		_, err := os.Stat(fmt.Sprintf("%v/carapace/styles.json", dir))
		if err == nil {
			return
		}
	}

	// Overwrite all default styles for color
	for i := 1; i < 13; i++ {
		styleStr := fmt.Sprintf("carapace.Highlight%d", i)
		style.Set(styleStr, "bright-white")
	}

	// Overwrite all default styles for flags
	style.Set("carapace.FlagArg", "bright-white")
	style.Set("carapace.FlagMultiArg", "bright-white")
	style.Set("carapace.FlagNoArg", "bright-white")
	style.Set("carapace.FlagOptArg", "bright-white")
}
