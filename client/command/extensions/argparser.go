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
// ParseFlagArgumentsToBuffer 根据扩展清单解析 flag__PH0__ 参数
// and converts them to a BOF-compatible binary buffer
// 并将它们转换为 BOF__PH0__ 二进制缓冲区
func ParseFlagArgumentsToBuffer(_ *cobra.Command, args []string, _ string, ext *ExtCommand) ([]byte, error) {
	// Create a flag set and parse the arguments
	// Create 设置标志并解析参数
	fs, stringValues, wstringValues, intValues, shortValues, fileValues, err := bofSetupAndParseFlags(args, ext)
	if err != nil {
		return nil, err
	}

	// Print debug information about parsed flags
	// Print 有关已解析标志的调试信息
	//bofDebugPrintParsedFlags(fs, ext, stringValues, wstringValues, intValues, shortValues, fileValues)
	//bofDebugPrintParsedFlags（文件系统，外部，stringValues，wstringValues，intValues，shortValues，fileValues）

	// Initialize the BOF arguments buffer
	// Initialize BOF 参数缓冲区
	argsBuffer := core.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	// Process arguments and build the buffer
	// Process 参数并构建缓冲区
	missingRequiredArgs, err := bofProcessArgumentsToBuffer(fs, ext, argsBuffer,
		stringValues, wstringValues, intValues, shortValues, fileValues)
	if err != nil {
		return nil, err
	}

	// Return error if we have missing required arguments
	// 如果我们缺少必需的参数，则会出现 Return 错误
	if len(missingRequiredArgs) > 0 {
		return nil, fmt.Errorf("required arguments %s were not provided", strings.Join(missingRequiredArgs, ", "))
	}

	// Get the final binary buffer
	// Get 最终的二进制缓冲区
	parsedArgs, err := argsBuffer.GetBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to get buffer: %v", err)
	}

	return parsedArgs, nil
}

// bofSetupAndParseFlags creates a flag set, defines expected flags based on the extension manifest,
// bofSetupAndParseFlags 创建一个标志集，根据扩展清单定义预期标志，
// and parses the provided arguments
// 并解析提供的参数
func bofSetupAndParseFlags(args []string, ext *ExtCommand) (*flag.FlagSet,
	map[string]*string, map[string]*string, map[string]*int, map[string]*int, map[string]*string, error) {

	// Create a new FlagSet
	// Create 一个新的 FlagSet
	fs := flag.NewFlagSet("sliver-bof", flag.ContinueOnError)

	// Maps to store flag value pointers
	// Maps 存储标志值指针
	stringValues := make(map[string]*string)
	intValues := make(map[string]*int)
	shortValues := make(map[string]*int)
	fileValues := make(map[string]*string) // Path to file data
	fileValues := make(map[string]*string) // Path 到文件数据
	wstringValues := make(map[string]*string)

	// Define expected flags based on manifest arguments
	// Define 基于清单参数的预期标志
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
	// Parse 参数
	if err := fs.Parse(args); err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return fs, stringValues, wstringValues, intValues, shortValues, fileValues, nil
}

// bofProcessArgumentsToBuffer processes each argument and adds its value to the buffer
// bofProcessArgumentsToBuffer 处理每个参数并将其值添加到缓冲区
// Returns a list of missing required arguments and any error encountered
// Returns 缺少必需参数和遇到的任何错误的列表
func bofProcessArgumentsToBuffer(fs *flag.FlagSet, ext *ExtCommand, argsBuffer core.BOFArgsBuffer,
	stringValues, wstringValues map[string]*string,
	intValues, shortValues map[string]*int,
	fileValues map[string]*string) ([]string, error) {

	missingRequiredArgs := make([]string, 0)

	for _, argDef := range ext.Arguments {
		var provided bool
		var err error

		// Process based on argument type
		// Process 基于参数类型
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
		// 未提供If参数，按规则处理
		if !provided {
			if argDef.Optional {
				// Try to apply default or type-appropriate zero value
				// Try 应用默认值或 type__PH0__ 零值
				err = bofApplyDefaultValue(argDef, argsBuffer)
				if err != nil {
					return nil, err
				}
			} else {
				// Required argument was not provided
				// 未提供 Required 参数
				missingRequiredArgs = append(missingRequiredArgs, "`"+argDef.Name+"`")
			}
		}
	}

	return missingRequiredArgs, nil
}

// bofProcessStringArg processes a string argument
// bofProcessStringArg 处理字符串参数
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
// bofProcessWStringArg 处理宽字符串参数
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
// bofProcessIntArg 处理整数参数
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
// bofProcessShortArg 处理短整数参数
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
// bofProcessFileArg 处理文件参数
func bofProcessFileArg(fs *flag.FlagSet, argDef *extensionArgument, fileValues map[string]*string,
	argsBuffer core.BOFArgsBuffer) (bool, error) {

	ptr := fileValues[argDef.Name]
	flagWasSet := *ptr != "" && bofFlagWasProvided(fs, argDef.Name)

	if flagWasSet {
		// Validate file exists and read it
		// Validate 文件存在并读取它
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
// bofApplyDefaultValue 根据参数定义应用默认值
func bofApplyDefaultValue(argDef *extensionArgument, argsBuffer core.BOFArgsBuffer) error {
	// Try to use default value from manifest if available
	// Try 使用清单中的默认值（如果可用）
	if argDef.Default != nil {
		switch argDef.Type {
		case "string":
			// Handle string default - could be string literal or numeric in JSON
			// Handle 字符串默认值 - 可以是 JSON 中的字符串文字或数字
			defaultVal, ok := argDef.Default.(string)
			if !ok {
				// Try to convert from other types
				// Try 从其他类型转换
				defaultVal = fmt.Sprintf("%v", argDef.Default)
			}
			err := argsBuffer.AddString(defaultVal)
			if err != nil {
				return fmt.Errorf("failed to add default string: %v", err)
			}
			fmt.Printf("  -%s:%s (default)\n", argDef.Name, defaultVal)

		case "wstring":
			// Handle wstring default
			// Handle wstring 默认值
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
			// Handle int 默认值 - 可以是 JSON 中的数字或字符串
			var defaultInt uint32

			// Try as number first
			// Try 作为数字优先
			numVal, ok := argDef.Default.(float64) // JSON unmarshals numbers as float64
			numVal, ok := argDef.Default.(float64) // JSON 将数字解组为 float64
			if ok {
				defaultInt = uint32(numVal)
			} else {
				// Try as string
				// Try 作为字符串
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
			// Handle 短默认值 - 可以是 JSON 中的数字或字符串
			var defaultShort uint16

			// Try as number first
			// Try 作为数字优先
			numVal, ok := argDef.Default.(float64) // JSON unmarshals numbers as float64
			numVal, ok := argDef.Default.(float64) // JSON 将数字解组为 float64
			if ok {
				defaultShort = uint16(numVal)
			} else {
				// Try as string
				// Try 作为字符串
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
			// Default 对于文件来说并没有什么意义，但是为了完整性而处理
			err := argsBuffer.AddData([]byte{})
			if err != nil {
				return fmt.Errorf("failed to add default file data: %v", err)
			}
			fmt.Printf("  -%s:<empty> (default)\n", argDef.Name)
		}
	} else {
		// No default specified in manifest, use type-appropriate zero values
		// No 默认在清单中指定，使用 type__PH0__ 零值
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
			// Empty 可选文件的数据
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
// bofDebugPrintParsedFlags 打印有关已解析标志的信息
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
// bofFlagWasProvided 检查是否显式设置了标志
func bofFlagWasProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
