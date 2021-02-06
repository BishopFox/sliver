package completers

import (
	"os"
	"strings"

	"github.com/maxlandon/readline"
)

// completeEnvironmentVariables - Returns all environment variables as suggestions
func completeEnvironmentVariables(lastWord string) (last string, completions []*readline.CompletionGroup) {

	// Check if last input is made of several different variables
	allVars := strings.Split(lastWord, "/")
	lastVar := allVars[len(allVars)-1]

	var evaluated = map[string]string{}

	grp := &readline.CompletionGroup{
		Name:        "console OS environment",
		MaxLength:   5, // Should be plenty enough
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true, // Some variables can be paths
	}

	for k, v := range clientEnv {
		if strings.HasPrefix("$"+k, lastVar) {
			grp.Suggestions = append(grp.Suggestions, "$"+k+"/")
			evaluated[k] = v
		}
	}

	completions = append(completions, grp)

	return lastVar, completions
}

// clientEnv - Contains all OS environment variables, client-side.
// This is used for things like downloading/uploading files from localhost, etc.,
// therefore we need completion and parsing stuff, sometimes.
var clientEnv = map[string]string{}

// ParseEnvironmentVariables - Parses a line of input and replace detected environment variables with their values.
func ParseEnvironmentVariables(args []string) (processed []string, err error) {

	for _, arg := range args {

		// Anywhere a $ is assigned means there is an env variable
		if strings.Contains(arg, "$") || strings.Contains(arg, "~") {

			//Split in case env is embedded in path
			envArgs := strings.Split(arg, "/")

			// If its not a path
			if len(envArgs) == 1 {
				processed = append(processed, handleCuratedVar(arg))
			}

			// If len of the env var split is > 1, its a path
			if len(envArgs) > 1 {
				processed = append(processed, handleEmbeddedVar(arg))
			}
		} else if arg != "" && arg != " " {
			// Else, if arg is not an environment variable, return it as is
			processed = append(processed, arg)
		}

	}
	return
}

// handleCuratedVar - Replace an environment variable alone and without any undesired characters attached
func handleCuratedVar(arg string) (value string) {
	if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
		envVar := strings.TrimPrefix(arg, "$")
		val, ok := clientEnv[envVar]
		if !ok {
			return envVar
		}
		return val
	}
	if arg != "" && arg == "~" {
		return clientEnv["HOME"]
	}

	return arg
}

// handleEmbeddedVar - Replace an environment variable that is in the middle of a path, or other one-string combination
func handleEmbeddedVar(arg string) (value string) {

	envArgs := strings.Split(arg, "/")
	var path []string

	for _, arg := range envArgs {
		if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
			envVar := strings.TrimPrefix(arg, "$")
			val, ok := clientEnv[envVar]
			if !ok {
				// Err will be caught when command is ran anyway, or completion will stop...
				path = append(path, arg)
			}
			path = append(path, val)
		} else if arg != "" && arg == "~" {
			path = append(path, clientEnv["HOME"])
		} else if arg != " " && arg != "" {
			path = append(path, arg)
		}
	}

	return strings.Join(path, "/")
}

// loadClientEnv - Loads all user environment variables
func loadClientEnv() error {
	env := os.Environ()

	for _, kv := range env {
		key := strings.Split(kv, "=")[0]
		value := strings.Split(kv, "=")[1]
		clientEnv[key] = value
	}
	return nil
}
