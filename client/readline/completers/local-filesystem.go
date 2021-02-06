package completers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/evilsocket/islazy/fs"

	"github.com/maxlandon/readline"
)

func completeLocalPath(last string) (string, *readline.CompletionGroup) {

	// Completions
	completion := &readline.CompletionGroup{
		Name:        "(console) local path",
		MaxLength:   10, // The grid system is not yet able to roll on comps if > MaxLength
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true,
	}
	var suggestions []string

	// Any parsing error is silently ignored, for not messing the prompt
	processedPath, _ := ParseEnvironmentVariables([]string{last})

	// Check if processed input is empty
	var inputPath string
	if len(processedPath) == 1 {
		inputPath = processedPath[0]
	}

	// Add a slash if the raw input has one but not the processed input
	if len(last) > 0 && last[len(last)-1] == '/' {
		inputPath += "/"
	}

	var linePath string // curated version of the inputPath
	var absPath string  // absolute path (excluding suffix) of the inputPath
	var lastPath string // last directory in the input path

	if strings.HasSuffix(string(inputPath), "/") {
		linePath = filepath.Dir(string(inputPath))
		absPath, _ = fs.Expand(string(linePath)) // Get absolute path

	} else if string(inputPath) == "" {
		linePath = "."
		absPath, _ = fs.Expand(string(linePath))
	} else {
		linePath = filepath.Dir(string(inputPath))
		absPath, _ = fs.Expand(string(linePath))    // Get absolute path
		lastPath = filepath.Base(string(inputPath)) // Save filter
	}

	// 2) We take the absolute path we found, and get all dirs in it.
	var dirs []string
	files, _ := ioutil.ReadDir(absPath)
	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	switch lastPath {
	case "":
		for _, dir := range dirs {
			if strings.HasPrefix(dir, lastPath) || lastPath == dir {
				tokenized := addSpaceTokens(dir)
				suggestions = append(suggestions, tokenized+"/")
			}
		}
	default:
		filtered := []string{}
		for _, dir := range dirs {
			if strings.HasPrefix(dir, lastPath) {
				filtered = append(filtered, dir)
			}
		}

		for _, dir := range filtered {
			if !hasPrefix([]rune(lastPath), []rune(dir)) || lastPath == dir {
				tokenized := addSpaceTokens(dir)
				suggestions = append(suggestions, tokenized+"/")
			}
		}

	}

	completion.Suggestions = suggestions
	return string(lastPath), completion
}

func addSpaceTokens(in string) (path string) {
	items := strings.Split(in, " ")
	for i := range items {
		if len(items) == i+1 { // If last one, no char, add and return
			path += items[i]
			return
		}
		path += items[i] + "\\ " // By default add space char and roll
	}
	return
}

func completeLocalPathAndFiles(last string) (string, *readline.CompletionGroup) {

	// Completions
	completion := &readline.CompletionGroup{
		Name:        "(console) local directory/files",
		MaxLength:   10, // The grid system is not yet able to roll on comps if > MaxLength
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true,
	}
	var suggestions []string

	// Any parsing error is silently ignored, for not messing the prompt
	processedPath, _ := ParseEnvironmentVariables([]string{last})

	// Check if processed input is empty
	var inputPath string
	if len(processedPath) == 1 {
		inputPath = processedPath[0]
	}

	// Add a slash if the raw input has one but not the processed input
	if len(last) > 0 && last[len(last)-1] == '/' {
		inputPath += "/"
	}

	var linePath string // curated version of the inputPath
	var absPath string  // absolute path (excluding suffix) of the inputPath
	var lastPath string // last directory in the input path

	if strings.HasSuffix(string(inputPath), "/") {
		linePath = filepath.Dir(string(inputPath)) // Trim the non needed slash
		absPath, _ = fs.Expand(string(linePath))   // Get absolute path

	} else if string(inputPath) == "" {
		linePath = "."
		absPath, _ = fs.Expand(string(linePath))
	} else {
		linePath = filepath.Dir(string(inputPath))
		absPath, _ = fs.Expand(string(linePath))    // Get absolute path
		lastPath = filepath.Base(string(inputPath)) // Save filter
	}

	// 2) We take the absolute path we found, and get all dirs in it.
	var dirs []string
	files, _ := ioutil.ReadDir(absPath)
	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	switch lastPath {
	case "":
		for _, file := range files {
			if strings.HasPrefix(file.Name(), lastPath) || lastPath == file.Name() {
				if file.IsDir() {
					suggestions = append(suggestions, file.Name()+"/")
				} else {
					suggestions = append(suggestions, file.Name())
				}
			}
		}
	default:
		filtered := []os.FileInfo{}
		for _, file := range files {
			if strings.HasPrefix(file.Name(), lastPath) {
				filtered = append(filtered, file)
			}
		}

		for _, file := range filtered {
			if !hasPrefix([]rune(lastPath), []rune(file.Name())) || lastPath == file.Name() {
				if file.IsDir() {
					suggestions = append(suggestions, file.Name()+"/")
				} else {
					suggestions = append(suggestions, file.Name())
				}
			}
		}

	}

	completion.Suggestions = suggestions
	return string(lastPath), completion
}
