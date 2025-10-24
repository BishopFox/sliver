package extensions

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/spf13/cobra"
)

// ParseFlagArgumentsToBuffer parses flag-style arguments based on extension manifest
// and converts them to a BOF-compatible binary buffer
func ParseFlagArgumentsToBuffer(_ *cobra.Command, args []string, _ string, ext *ExtCommand) ([]byte, error) {
	// Create a flag set and parse the arguments
	fs, stringValues, wstringValues, intValues, shortValues, fileValues, err := bofSetupAndParseFlags(args, ext)
	if err != nil {
		return nil, err
	}

	// Print debug information about parsed flags
	//bofDebugPrintParsedFlags(fs, ext, stringValues, wstringValues, intValues, shortValues, fileValues)

	// Initialize the BOF arguments buffer
	argsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	// Process arguments and build the buffer
	missingRequiredArgs, err := bofProcessArgumentsToBuffer(fs, ext, argsBuffer,
		stringValues, wstringValues, intValues, shortValues, fileValues)
	if err != nil {
		return nil, err
	}

	// Return error if we have missing required arguments
	if len(missingRequiredArgs) > 0 {
		return nil, fmt.Errorf("required arguments %s were not provided", strings.Join(missingRequiredArgs, ", "))
	}

	// Get the final binary buffer
	parsedArgs, err := argsBuffer.GetBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to get buffer: %v", err)
	}

	return parsedArgs, nil
}

// bofSetupAndParseFlags creates a flag set, defines expected flags based on the extension manifest,
// and parses the provided arguments
func bofSetupAndParseFlags(args []string, ext *ExtCommand) (*flag.FlagSet,
	map[string]*string, map[string]*string, map[string]*int, map[string]*int, map[string]*string, error) {

	// Create a new FlagSet
	fs := flag.NewFlagSet("sliver-bof", flag.ContinueOnError)

	// Maps to store flag value pointers
	stringValues := make(map[string]*string)
	intValues := make(map[string]*int)
	shortValues := make(map[string]*int)
	fileValues := make(map[string]*string) // Path to file data
	wstringValues := make(map[string]*string)

	// Define expected flags based on manifest arguments
	for _, arg := range ext.Arguments {
		flagName := arg.Name
		flagDesc := arg.Desc

		switch arg.Type {
		case "string":
			stringValues[flagName] = fs.String(flagName, "", flagDesc)
		case "wstring":
			wstringValues[flagName] = fs.String(flagName, "", flagDesc)
		case "int", "integer":
			intValues[flagName] = fs.Int(flagName, 0, flagDesc)
		case "short":
			shortValues[flagName] = fs.Int(flagName, 0, flagDesc)
		case "file":
			fileValues[flagName] = fs.String(flagName, "", flagDesc)
		default:
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported argument type: %s", arg.Type)
		}
	}

	// Parse the arguments
	if err := fs.Parse(args); err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return fs, stringValues, wstringValues, intValues, shortValues, fileValues, nil
}

// bofProcessArgumentsToBuffer processes each argument and adds its value to the buffer
// Returns a list of missing required arguments and any error encountered
func bofProcessArgumentsToBuffer(fs *flag.FlagSet, ext *ExtCommand, argsBuffer core.BOFArgsBuffer,
	stringValues, wstringValues map[string]*string,
	intValues, shortValues map[string]*int,
	fileValues map[string]*string) ([]string, error) {

	missingRequiredArgs := make([]string, 0)

	for _, argDef := range ext.Arguments {
		var provided bool
		var err error

		// Process based on argument type
		switch argDef.Type {
		case "string":
			provided, err = bofProcessStringArg(fs, argDef, stringValues, argsBuffer)
		case "wstring":
			provided, err = bofProcessWStringArg(fs, argDef, wstringValues, argsBuffer)
		case "int", "integer":
			provided, err = bofProcessIntArg(fs, argDef, intValues, argsBuffer)
		case "short":
			provided, err = bofProcessShortArg(fs, argDef, shortValues, argsBuffer)
		case "file":
			provided, err = bofProcessFileArg(fs, argDef, fileValues, argsBuffer)
		}

		if err != nil {
			return nil, err
		}

		// If argument not provided, handle according to rules
		if !provided {
			if argDef.Optional {
				// Try to apply default or type-appropriate zero value
				err = bofApplyDefaultValue(argDef, argsBuffer)
				if err != nil {
					return nil, err
				}
			} else {
				// Required argument was not provided
				missingRequiredArgs = append(missingRequiredArgs, "`"+argDef.Name+"`")
			}
		}
	}

	return missingRequiredArgs, nil
}

// bofProcessStringArg processes a string argument
func bofProcessStringArg(fs *flag.FlagSet, argDef *extensionArgument, stringValues map[string]*string,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := stringValues[argDef.Name]
	flagWasSet := *ptr != "" && bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		err := argsBuffer.AddString(*ptr)
		if err != nil {
			return false, fmt.Errorf("failed to add string argument %s: %v", argDef.Name, err)
		}
		return true, nil
	}

	return false, nil
}

// bofProcessWStringArg processes a wide string argument
func bofProcessWStringArg(fs *flag.FlagSet, argDef *extensionArgument, wstringValues map[string]*string,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := wstringValues[argDef.Name]
	flagWasSet := *ptr != "" && bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		err := argsBuffer.AddWString(*ptr)
		if err != nil {
			return false, fmt.Errorf("failed to add wstring argument %s: %v", argDef.Name, err)
		}
		return true, nil
	}

	return false, nil
}

// bofProcessIntArg processes an integer argument
func bofProcessIntArg(fs *flag.FlagSet, argDef *extensionArgument, intValues map[string]*int,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := intValues[argDef.Name]
	flagWasSet := bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		err := argsBuffer.AddInt(uint32(*ptr))
		if err != nil {
			return false, fmt.Errorf("failed to add int argument %s: %v", argDef.Name, err)
		}
		return true, nil
	}

	return false, nil
}

// bofProcessShortArg processes a short integer argument
func bofProcessShortArg(fs *flag.FlagSet, argDef *extensionArgument, shortValues map[string]*int,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := shortValues[argDef.Name]
	flagWasSet := bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		err := argsBuffer.AddShort(uint16(*ptr))
		if err != nil {
			return false, fmt.Errorf("failed to add short argument %s: %v", argDef.Name, err)
		}
		return true, nil
	}

	return false, nil
}

// bofProcessFileArg processes a file argument
func bofProcessFileArg(fs *flag.FlagSet, argDef *extensionArgument, fileValues map[string]*string,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := fileValues[argDef.Name]
	flagWasSet := *ptr != "" && bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		// Validate file exists and read it
		data, err := os.ReadFile(*ptr)
		if err != nil {
			return false, fmt.Errorf("error reading file for argument %s: %v", argDef.Name, err)
		}
		err = argsBuffer.AddData(data)
		if err != nil {
			return false, fmt.Errorf("failed to add file data for argument %s: %v", argDef.Name, err)
		}
		return true, nil
	}

	return false, nil
}

// bofApplyDefaultValue applies a default value based on the argument definition
func bofApplyDefaultValue(argDef *extensionArgument, argsBuffer core.BOFArgsBuffer) error {
	// Try to use default value from manifest if available
	if argDef.Default != nil {
		switch argDef.Type {
		case "string":
			// Handle string default - could be string literal or numeric in JSON
			defaultVal, ok := argDef.Default.(string)
			if !ok {
				// Try to convert from other types
				defaultVal = fmt.Sprintf("%v", argDef.Default)
			}
			err := argsBuffer.AddString(defaultVal)
			if err != nil {
				return fmt.Errorf("failed to add default string: %v", err)
			}
			fmt.Printf("  -%s:%s (default)\n", argDef.Name, defaultVal)

		case "wstring":
			// Handle wstring default
			defaultVal, ok := argDef.Default.(string)
			if !ok {
				defaultVal = fmt.Sprintf("%v", argDef.Default)
			}
			err := argsBuffer.AddWString(defaultVal)
			if err != nil {
				return fmt.Errorf("failed to add default wstring: %v", err)
			}
			fmt.Printf("  -%s:%s (default)\n", argDef.Name, defaultVal)

		case "int", "integer":
			// Handle int default - could be number or string in JSON
			var defaultInt uint32

			// Try as number first
			numVal, ok := argDef.Default.(float64) // JSON unmarshals numbers as float64
			if ok {
				defaultInt = uint32(numVal)
			} else {
				// Try as string
				strVal, ok := argDef.Default.(string)
				if ok {
					val, err := strconv.ParseUint(strVal, 10, 32)
					if err == nil {
						defaultInt = uint32(val)
					}
				}
			}

			err := argsBuffer.AddInt(defaultInt)
			if err != nil {
				return fmt.Errorf("failed to add default int: %v", err)
			}
			fmt.Printf("  -%s:%d (default)\n", argDef.Name, defaultInt)

		case "short":
			// Handle short default - could be number or string in JSON
			var defaultShort uint16

			// Try as number first
			numVal, ok := argDef.Default.(float64) // JSON unmarshals numbers as float64
			if ok {
				defaultShort = uint16(numVal)
			} else {
				// Try as string
				strVal, ok := argDef.Default.(string)
				if ok {
					val, err := strconv.ParseUint(strVal, 10, 16)
					if err == nil {
						defaultShort = uint16(val)
					}
				}
			}

			err := argsBuffer.AddShort(defaultShort)
			if err != nil {
				return fmt.Errorf("failed to add default short: %v", err)
			}
			fmt.Printf("  -%s:%d (default)\n", argDef.Name, defaultShort)

		case "file":
			// Default for file doesn't really make sense, but handle for completeness
			err := argsBuffer.AddData([]byte{})
			if err != nil {
				return fmt.Errorf("failed to add default file data: %v", err)
			}
			fmt.Printf("  -%s:<empty> (default)\n", argDef.Name)
		}
	} else {
		// No default specified in manifest, use type-appropriate zero values
		switch argDef.Type {
		case "string":
			err := argsBuffer.AddString("")
			if err != nil {
				return fmt.Errorf("failed to add default string: %v", err)
			}
			fmt.Printf("  -%s:<empty> (default)\n", argDef.Name)

		case "wstring":
			err := argsBuffer.AddWString("")
			if err != nil {
				return fmt.Errorf("failed to add default wstring: %v", err)
			}
			fmt.Printf("  -%s:<empty> (default)\n", argDef.Name)

		case "int", "integer":
			err := argsBuffer.AddInt(0)
			if err != nil {
				return fmt.Errorf("failed to add default int: %v", err)
			}
			fmt.Printf("  -%s:0 (default)\n", argDef.Name)

		case "short":
			err := argsBuffer.AddShort(0)
			if err != nil {
				return fmt.Errorf("failed to add default short: %v", err)
			}
			fmt.Printf("  -%s:0 (default)\n", argDef.Name)

		case "file":
			// Empty data for optional file
			err := argsBuffer.AddData([]byte{})
			if err != nil {
				return fmt.Errorf("failed to add default file data: %v", err)
			}
			fmt.Printf("  -%s:<empty> (default)\n", argDef.Name)
		}
	}

	return nil
}

// bofDebugPrintParsedFlags prints information about the parsed flags
func bofDebugPrintParsedFlags(fs *flag.FlagSet, ext *ExtCommand,
	stringValues map[string]*string,
	wstringValues map[string]*string,
	intValues map[string]*int,
	shortValues map[string]*int,
	fileValues map[string]*string) {

	fmt.Println("parsed flags:")

	for _, arg := range ext.Arguments {
		flagName := arg.Name
		wasProvided := bofFlagWasProvided(fs, flagName)

		if wasProvided {
			switch arg.Type {
			case "string":
				fmt.Printf("  -%s:%s\n", flagName, *stringValues[flagName])
			case "wstring":
				fmt.Printf("  -%s:%s\n", flagName, *wstringValues[flagName])
			case "int", "integer":
				fmt.Printf("  -%s:%d\n", flagName, *intValues[flagName])
			case "short":
				fmt.Printf("  -%s:%d\n", flagName, *shortValues[flagName])
			case "file":
				fmt.Printf("  -%s:%s\n", flagName, *fileValues[flagName])
			}
		}
	}
}

// bofFlagWasProvided checks if a flag was explicitly set
func bofFlagWasProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
