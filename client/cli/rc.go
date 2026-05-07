package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const RCFlagName = "rc"

func ReadRCScript(cmd *cobra.Command) (string, error) {
	if cmd == nil {
		return "", nil
	}
	if cmd.Flags().Lookup(RCFlagName) == nil {
		return "", nil
	}
	rcPath, err := cmd.Flags().GetString(RCFlagName)
	if err != nil {
		return "", err
	}
	if rcPath == "" {
		return "", nil
	}
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return "", fmt.Errorf("read rc script %q: %w", rcPath, err)
	}
	return string(data), nil
}
