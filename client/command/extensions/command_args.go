package extensions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func parseExtensionCommandArgs(cmd *cobra.Command, args []string) ([]string, bool, error) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--":
			return append([]string(nil), args[index+1:]...), false, nil
		case arg == "--help" || arg == "-h":
			return nil, true, nil
		case arg == "--save" || arg == "-s":
			if err := setExtensionCommandFlag(cmd, "save", "true"); err != nil {
				return nil, false, err
			}
		case strings.HasPrefix(arg, "--save="):
			if err := setExtensionCommandFlag(cmd, "save", strings.TrimPrefix(arg, "--save=")); err != nil {
				return nil, false, err
			}
		case strings.HasPrefix(arg, "-s="):
			if err := setExtensionCommandFlag(cmd, "save", strings.TrimPrefix(arg, "-s=")); err != nil {
				return nil, false, err
			}
		case arg == "--timeout" || arg == "-t":
			if index+1 >= len(args) {
				return nil, false, fmt.Errorf("%s requires a value", arg)
			}
			index++
			if err := setExtensionCommandFlag(cmd, "timeout", args[index]); err != nil {
				return nil, false, err
			}
		case strings.HasPrefix(arg, "--timeout="):
			if err := setExtensionCommandFlag(cmd, "timeout", strings.TrimPrefix(arg, "--timeout=")); err != nil {
				return nil, false, err
			}
		case strings.HasPrefix(arg, "-t="):
			if err := setExtensionCommandFlag(cmd, "timeout", strings.TrimPrefix(arg, "-t=")); err != nil {
				return nil, false, err
			}
		case isAttachedTimeoutShorthand(arg):
			if err := setExtensionCommandFlag(cmd, "timeout", strings.TrimPrefix(arg, "-t")); err != nil {
				return nil, false, err
			}
		default:
			return append([]string(nil), args[index:]...), false, nil
		}
	}
	return []string{}, false, nil
}

func setExtensionCommandFlag(cmd *cobra.Command, name string, value string) error {
	if err := cmd.Flags().Set(name, value); err != nil {
		return fmt.Errorf("invalid Sliver --%s value %q: %w", name, value, err)
	}
	return nil
}

func isAttachedTimeoutShorthand(arg string) bool {
	if !strings.HasPrefix(arg, "-t") || len(arg) <= 2 {
		return false
	}
	_, err := strconv.ParseInt(strings.TrimPrefix(arg, "-t"), 10, 64)
	return err == nil
}

func normalizeBOFArguments(args []string, ext *ExtCommand) ([]string, error) {
	argumentsByName := make(map[string]*extensionArgument, len(ext.Arguments))
	for _, argument := range ext.Arguments {
		argumentsByName[argument.Name] = argument
	}

	normalized := make([]string, 0, len(args))
	provided := make(map[string]bool, len(ext.Arguments))
	positionalIndex := 0

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			continue
		}

		name, value, hasValue, isFlag := splitBOFFlag(arg)
		if isFlag {
			if _, ok := argumentsByName[name]; !ok {
				return nil, fmt.Errorf("flag provided but not defined: -%s", name)
			}
			if !hasValue {
				if index+1 >= len(args) {
					return nil, fmt.Errorf("flag needs an argument: -%s", name)
				}
				index++
				value = args[index]
			}
		} else {
			for positionalIndex < len(ext.Arguments) && provided[ext.Arguments[positionalIndex].Name] {
				positionalIndex++
			}
			if positionalIndex >= len(ext.Arguments) {
				return nil, fmt.Errorf("unexpected positional argument %q", arg)
			}
			name = ext.Arguments[positionalIndex].Name
			value = arg
			positionalIndex++
		}

		normalized = append(normalized, "-"+name, value)
		provided[name] = true
	}

	return normalized, nil
}

func splitBOFFlag(arg string) (name string, value string, hasValue bool, isFlag bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", "", false, false
	}

	nameValue := strings.TrimPrefix(arg, "-")
	nameValue = strings.TrimPrefix(nameValue, "-")
	if nameValue == "" {
		return "", "", false, false
	}

	parts := strings.SplitN(nameValue, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true, true
	}
	return parts[0], "", false, true
}
