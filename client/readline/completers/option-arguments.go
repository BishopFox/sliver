package completers

import (
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// completeOptionArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeOptionArguments(cmd *flags.Command, opt *flags.Option, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// By default the last word is the prefix
	prefix = lastWord

	var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	// First of all: some options, no matter their contexts and subject, have default values.
	// When we have such an option, we don't bother analyzing context, we just build completions and return.
	if len(opt.Choices) > 0 {
		comp = &readline.CompletionGroup{
			Name:        opt.ValueName, // Value names are specified in struct metadata fields
			DisplayType: readline.TabDisplayGrid,
		}
		for _, choice := range opt.Choices {
			if strings.HasPrefix(choice, lastWord) {
				comp.Suggestions = append(comp.Suggestions, choice)
			}
		}
		completions = append(completions, comp)
		return
	}

	// EXAMPLE OF COMPLETING ARGUMENTS BASED ON THEIR NAMES -----------------------------------------------------------------------
	// We have 3 words, potentially different, with which we can filter:
	//
	// 1) '--option-name' is the string typed as input.
	// 2) 'OptionName' is the name of the struct/type for this option.
	// 3) 'ValueName' is the name of the value we expect.
	// var match = func(name string) bool {
	//         if strings.Contains(opt.Field().Name, name) {
	//                 return true
	//         }
	//         return false
	// }
	//
	//         // Sessions
	//         if match("ImplantID") || match("SessionID") {
	//                 completions = append(completions, sessionIDs(lastWord))
	//         }
	//
	//         // Any arguments with a path name. Often we "save" files that need paths, certificates, etc
	//         if match("Path") || match("Save") || match("Certificate") || match("PrivateKey") {
	//                 switch cmd.Name {
	//                 case constants.WebContentTypeStr, constants.WebUpdateStr, constants.AddWebContentStr, constants.RmWebContentStr:
	//                         // Make an exception for WebPath option in websites commands.
	//                 default:
	//                         switch opt.ValueName {
	//                         case "local-path", "path":
	//                                 prefix, comp = completeLocalPath(lastWord)
	//                                 completions = append(completions, comp)
	//                         case "local-file", "file":
	//                                 prefix, comp = completeLocalPathAndFiles(lastWord)
	//                                 completions = append(completions, comp)
	//                         default:
	//                                 // We always have a default searching for files, locally
	//                                 prefix, comp = completeLocalPathAndFiles(lastWord)
	//                                 completions = append(completions, comp)
	//                         }
	//
	//                 }
	//         }
	//
	return
}
