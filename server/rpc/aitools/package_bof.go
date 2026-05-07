package aitools

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/core"
)

func buildBOFExtensionArgs(command *aiExtensionCommand, bofData []byte, args []string) ([]byte, error) {
	argsBuffer := core.BOFArgsBuffer{Buffer: new(bytes.Buffer)}
	if err := argsBuffer.AddString(command.Entrypoint); err != nil {
		return nil, err
	}
	if err := argsBuffer.AddData(bofData); err != nil {
		return nil, err
	}

	parsedArgs, err := parseBOFArguments(command, args)
	if err != nil {
		return nil, err
	}
	if err := argsBuffer.AddData(parsedArgs); err != nil {
		return nil, err
	}
	return argsBuffer.GetBuffer()
}

func parseBOFArguments(command *aiExtensionCommand, args []string) ([]byte, error) {
	fs, stringValues, wstringValues, intValues, shortValues, fileValues, err := setupBOFFlags(args, command)
	if err != nil {
		return nil, err
	}

	argsBuffer := core.BOFArgsBuffer{Buffer: new(bytes.Buffer)}
	missingRequired := []string{}

	for _, argDef := range command.Arguments {
		if argDef == nil {
			continue
		}

		var provided bool
		switch argDef.Type {
		case "string":
			provided, err = processBOFStringArg(fs, argDef, stringValues, argsBuffer)
		case "wstring":
			provided, err = processBOFWStringArg(fs, argDef, wstringValues, argsBuffer)
		case "int", "integer":
			provided, err = processBOFIntArg(fs, argDef, intValues, argsBuffer)
		case "short":
			provided, err = processBOFShortArg(fs, argDef, shortValues, argsBuffer)
		case "file":
			provided, err = processBOFFileArg(fs, argDef, fileValues, argsBuffer)
		default:
			return nil, fmt.Errorf("unsupported BOF argument type %q", argDef.Type)
		}
		if err != nil {
			return nil, err
		}

		if provided {
			continue
		}
		if argDef.Optional {
			if err := applyBOFDefaultValue(argDef, argsBuffer); err != nil {
				return nil, err
			}
			continue
		}
		missingRequired = append(missingRequired, argDef.Name)
	}

	if len(missingRequired) > 0 {
		return nil, fmt.Errorf("required BOF arguments were not provided: %s", strings.Join(missingRequired, ", "))
	}
	return argsBuffer.GetBuffer()
}

func setupBOFFlags(args []string, command *aiExtensionCommand) (
	*flag.FlagSet,
	map[string]*string,
	map[string]*string,
	map[string]*int,
	map[string]*int,
	map[string]*string,
	error,
) {
	fs := flag.NewFlagSet("sliver-ai-bof", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	stringValues := map[string]*string{}
	wstringValues := map[string]*string{}
	intValues := map[string]*int{}
	shortValues := map[string]*int{}
	fileValues := map[string]*string{}

	for _, arg := range command.Arguments {
		if arg == nil {
			continue
		}
		switch arg.Type {
		case "string":
			stringValues[arg.Name] = fs.String(arg.Name, "", arg.Desc)
		case "wstring":
			wstringValues[arg.Name] = fs.String(arg.Name, "", arg.Desc)
		case "int", "integer":
			intValues[arg.Name] = fs.Int(arg.Name, 0, arg.Desc)
		case "short":
			shortValues[arg.Name] = fs.Int(arg.Name, 0, arg.Desc)
		case "file":
			fileValues[arg.Name] = fs.String(arg.Name, "", arg.Desc)
		default:
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported BOF argument type %q", arg.Type)
		}
	}

	if err := fs.Parse(args); err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return fs, stringValues, wstringValues, intValues, shortValues, fileValues, nil
}

func processBOFStringArg(fs *flag.FlagSet, argDef *aiExtensionArgument, values map[string]*string, argsBuffer core.BOFArgsBuffer) (bool, error) {
	ptr := values[argDef.Name]
	if ptr == nil || !bofFlagWasProvided(fs, argDef.Name) {
		return false, nil
	}
	if err := argsBuffer.AddString(*ptr); err != nil {
		return false, fmt.Errorf("failed to add string argument %s: %w", argDef.Name, err)
	}
	return true, nil
}

func processBOFWStringArg(fs *flag.FlagSet, argDef *aiExtensionArgument, values map[string]*string, argsBuffer core.BOFArgsBuffer) (bool, error) {
	ptr := values[argDef.Name]
	if ptr == nil || !bofFlagWasProvided(fs, argDef.Name) {
		return false, nil
	}
	if err := argsBuffer.AddWString(*ptr); err != nil {
		return false, fmt.Errorf("failed to add wstring argument %s: %w", argDef.Name, err)
	}
	return true, nil
}

func processBOFIntArg(fs *flag.FlagSet, argDef *aiExtensionArgument, values map[string]*int, argsBuffer core.BOFArgsBuffer) (bool, error) {
	ptr := values[argDef.Name]
	if ptr == nil || !bofFlagWasProvided(fs, argDef.Name) {
		return false, nil
	}
	if err := argsBuffer.AddInt(uint32(*ptr)); err != nil {
		return false, fmt.Errorf("failed to add integer argument %s: %w", argDef.Name, err)
	}
	return true, nil
}

func processBOFShortArg(fs *flag.FlagSet, argDef *aiExtensionArgument, values map[string]*int, argsBuffer core.BOFArgsBuffer) (bool, error) {
	ptr := values[argDef.Name]
	if ptr == nil || !bofFlagWasProvided(fs, argDef.Name) {
		return false, nil
	}
	if err := argsBuffer.AddShort(uint16(*ptr)); err != nil {
		return false, fmt.Errorf("failed to add short argument %s: %w", argDef.Name, err)
	}
	return true, nil
}

func processBOFFileArg(fs *flag.FlagSet, argDef *aiExtensionArgument, values map[string]*string, argsBuffer core.BOFArgsBuffer) (bool, error) {
	ptr := values[argDef.Name]
	if ptr == nil || !bofFlagWasProvided(fs, argDef.Name) {
		return false, nil
	}
	data, err := os.ReadFile(*ptr)
	if err != nil {
		return false, fmt.Errorf("failed to read file argument %s: %w", argDef.Name, err)
	}
	if err := argsBuffer.AddData(data); err != nil {
		return false, fmt.Errorf("failed to add file argument %s: %w", argDef.Name, err)
	}
	return true, nil
}

func applyBOFDefaultValue(argDef *aiExtensionArgument, argsBuffer core.BOFArgsBuffer) error {
	if argDef == nil {
		return nil
	}
	if argDef.Default == nil {
		return addBOFZeroValue(argDef.Type, argsBuffer)
	}

	switch argDef.Type {
	case "string":
		return argsBuffer.AddString(fmt.Sprint(argDef.Default))
	case "wstring":
		return argsBuffer.AddWString(fmt.Sprint(argDef.Default))
	case "int", "integer":
		value, err := bofDefaultIntValue(argDef.Default)
		if err != nil {
			return fmt.Errorf("invalid default integer value for %s: %w", argDef.Name, err)
		}
		return argsBuffer.AddInt(uint32(value))
	case "short":
		value, err := bofDefaultIntValue(argDef.Default)
		if err != nil {
			return fmt.Errorf("invalid default short value for %s: %w", argDef.Name, err)
		}
		return argsBuffer.AddShort(uint16(value))
	case "file":
		path := fmt.Sprint(argDef.Default)
		if path == "" {
			return addBOFZeroValue(argDef.Type, argsBuffer)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read default file for %s: %w", argDef.Name, err)
		}
		return argsBuffer.AddData(data)
	default:
		return fmt.Errorf("unsupported BOF argument type %q", argDef.Type)
	}
}

func addBOFZeroValue(argType string, argsBuffer core.BOFArgsBuffer) error {
	switch argType {
	case "string":
		return argsBuffer.AddString("")
	case "wstring":
		return argsBuffer.AddWString("")
	case "int", "integer":
		return argsBuffer.AddInt(0)
	case "short":
		return argsBuffer.AddShort(0)
	case "file":
		return argsBuffer.AddData([]byte{})
	default:
		return fmt.Errorf("unsupported BOF argument type %q", argType)
	}
}

func bofDefaultIntValue(raw any) (int64, error) {
	switch value := raw.(type) {
	case float64:
		return int64(value), nil
	case float32:
		return int64(value), nil
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case uint:
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		return int64(value), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric default type %T", raw)
	}
}

func bofFlagWasProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
