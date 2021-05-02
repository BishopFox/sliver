package completion

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func getHomeDirectory() string {
	switch core.ActiveSession.OS {
	case "windows":
		return "C:\\Users\\" + core.ActiveSession.Username
	case "darwin", "macos":
		return path.Join("/Users", core.ActiveSession.Username)
	default:
		return path.Join("/home", core.ActiveSession.Username)
	}
}

// CompleteRemotePath - Provides completion for the active session filesystem.
func CompleteRemotePath(last string) (string, []*readline.CompletionGroup) {

	// Completions
	completion := &readline.CompletionGroup{
		Name:        "remote path",
		MaxLength:   10, // The grid system is not yet able to roll on comps if > MaxLength
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true,
	}

	// Per-OS path separator
	if core.ActiveSession.OS == "windows" {
		completion.PathSeparator = '\\'
	} else {
		completion.PathSeparator = '/'
	}
	sep := completion.PathSeparator

	// Replace all session environment variables found.
	processedPath, _ := Console.ParseExpansionVariables([]string{last}, sep)

	// Normally only one string should return, because we passed one as arg.
	var inputPath string
	if len(processedPath) > 0 {
		inputPath = processedPath[0]
	}

	// If the env-processed array has no trailing path separator
	// but our raw user input has one, add it again.
	if len(last) > 0 && last[len(last)-1] == byte(sep) {
		inputPath += string(sep)
	}

	// 1) Get the absolute path. There are two cases:
	var linePath string // The path is "rounded" with a slash: no filter to keep.
	var absPath string  // The all-expanded, absolute path equivalent of linePath
	var rest string     // The rest is what we might have after expanding home.
	var lastPath string // The path is not a slash: a filter to keep for later.

	// Simple cases first: if the path is currently none (user has not entered anything)
	if inputPath == "" {
		linePath = "."
	}

	// Then, $HOME user expansions. If one is found, store it in both abs and line
	if strings.HasPrefix(inputPath, "~") {
		rest = strings.TrimPrefix(inputPath, "~")
		absPath = getHomeDirectory()
	} else {
		rest = inputPath
	}

	// Still working on the raw input, just check that
	// we need to store a base path, to filter against.
	if strings.HasSuffix(inputPath, string(sep)) {
		linePath = absPath + rest
	} else {
		linePath = absPath + filepath.Dir(rest) // We voluntarily do not add a path separator
		lastPath = filepath.Base(rest)
	}

	// Finally, the last path, which might be set only if the user
	// has entered something. This will always be the last bit of
	// uncurated path. The filepath.Base() returns a pdot when the
	// base directory is empty, so we remove it.
	if lastPath == "." && last == "" {
		lastPath = ""
	}

	// 2) We take the absolute path we found, and get all dirs in it.
	var dirs []string

	// Get the session completions cache
	sessCache := Cache.GetSessionCache(core.ActiveSession.ID)
	if sessCache == nil {
		return lastPath, []*readline.CompletionGroup{completion}
	}

	// Get files either from the cache itself, or through it requesting the implant.
	dirList := sessCache.GetDirectoryContents(linePath)
	if dirList == nil {
		return lastPath, []*readline.CompletionGroup{completion}
	}

	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			dirs = append(dirs, fileInfo.Name)
		}
	}

	// 3) Return only the items that match the current processed input line.
	switch lastPath {
	case "":
		for _, dir := range dirs {
			tokenized := addSpaceTokens(dir)
			search := tokenized + string(sep)
			if strings.HasPrefix(search, lastPath) {
				completion.Suggestions = append(completion.Suggestions, search)
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
			tokenized := addSpaceTokens(dir)
			search := tokenized + string(sep)
			if strings.HasPrefix(search, lastPath) {
				completion.Suggestions = append(completion.Suggestions, search)
			}
		}
	}

	return lastPath, []*readline.CompletionGroup{completion}
}

// CompleteRemotePathAndFiles - Provides completion for the active session filesystem, (directories and files)
func CompleteRemotePathAndFiles(last string) (string, []*readline.CompletionGroup) {
	// Completions
	completion := &readline.CompletionGroup{
		Name:        "remote path/files",
		MaxLength:   10, // The grid system is not yet able to roll on comps if > MaxLength
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true,
	}

	// Per-OS path separator
	if core.ActiveSession.OS == "windows" {
		completion.PathSeparator = '\\'
	} else {
		completion.PathSeparator = '/'
	}
	sep := completion.PathSeparator

	// Replace all session environment variables found.
	processedPath, _ := Console.ParseExpansionVariables([]string{last}, sep)

	// Normally only one string should return, because we passed one as arg.
	var inputPath string
	if len(processedPath) > 0 {
		inputPath = processedPath[0]
	}

	// If the env-processed array has no trailing path separator
	// but our raw user input has one, add it again.
	if len(last) > 0 && last[len(last)-1] == byte(sep) {
		inputPath += string(sep)
	}

	// 1) Get the absolute path. There are two cases:
	var linePath string // The path is "rounded" with a slash: no filter to keep.
	var absPath string  // The all-expanded, absolute path equivalent of linePath
	var rest string     // The rest is what we might have after expanding home.
	var lastPath string // The path is not a slash: a filter to keep for later.

	// Simple cases first: if the path is currently none (user has not entered anything)
	if inputPath == "" {
		linePath = "."
	}

	// Then, $HOME user expansions. If one is found, store it in both abs and line
	if strings.HasPrefix(inputPath, "~") {
		rest = strings.TrimPrefix(inputPath, "~")
		absPath = getHomeDirectory()
	} else {
		rest = inputPath
	}

	// Still working on the raw input, just check that
	// we need to store a base path, to filter against.
	if strings.HasSuffix(inputPath, string(sep)) {
		linePath = absPath + rest
	} else {
		linePath = absPath + filepath.Dir(rest) // We voluntarily do not add a path separator
		lastPath = filepath.Base(rest)
	}

	// 1) Get the absolute path. There are two cases:
	//      - The path is "rounded" with a slash: no filter to keep.
	//      - The path is not a slash: a filter to keep for later.
	// We keep a boolean for remembering which case we found
	// linePath := ""
	// lastPath := ""
	// switch core.ActiveSession.OS {
	// case "windows":
	//         if strings.HasSuffix(last, "\\") {
	//                 linePath = last // Trim the non needed slash
	//         } else if last == "" {
	//                 linePath = "."
	//         } else {
	//                 splitPath := strings.Split(last, "\\")
	//                 linePath = strings.Join(splitPath[:len(splitPath)-1], "\\") + "\\"
	//                 lastPath = splitPath[len(splitPath)-1]
	//         }
	// default:
	//         if strings.HasSuffix(last, "/") {
	//                 // If the the line is just "/", it means we start from filesystem root
	//                 if last == "/" {
	//                         linePath = "/"
	//                 } else if last == "~/" {
	//                         // If we look for "~", we need to build the path manually
	//                         linePath = filepath.Join("/home", core.ActiveSession.Username)
	//
	//                 } else if strings.HasPrefix(last, "~/") && last != "~/" {
	//                         // If we used the "~" at the beginning, we still need to build the path
	//                         homePath := filepath.Join("/home", core.ActiveSession.Username)
	//                         linePath = filepath.Join(homePath, strings.TrimPrefix(last, "~/"))
	//                 } else {
	//                         // Trim the non needed slash
	//                         linePath = strings.TrimSuffix(last, "/")
	//                 }
	//         } else if strings.HasPrefix(last, "~/") && last != "~/" {
	//                 // If we used the "~" at the beginning, we still need to build the path
	//                 homePath := filepath.Join("/home", core.ActiveSession.Username)
	//                 linePath = filepath.Join(homePath, filepath.Dir(strings.TrimPrefix(last, "~/")))
	//                 lastPath = filepath.Base(last)
	//
	//         } else if last == "" {
	//                 linePath = "."
	//         } else {
	//                 // linePath = last
	//                 linePath = filepath.Dir(last)
	//                 lastPath = filepath.Base(last)
	//         }
	// }

	// Get the session completions cache
	sessCache := Cache.GetSessionCache(core.ActiveSession.ID)
	if sessCache == nil {
		return lastPath, []*readline.CompletionGroup{completion}
	}

	// Get files either from the cache itself, or through it requesting the implant.
	dirList := sessCache.GetDirectoryContents(linePath)
	if dirList == nil {
		return lastPath, []*readline.CompletionGroup{completion}
	}
	if dirList == nil {
		return lastPath, []*readline.CompletionGroup{completion}
	}

	switch lastPath {
	case "":
		for _, f := range dirList.Files {
			tokenized := addSpaceTokens(f.Name)
			search := ""
			if f.IsDir {
				if core.ActiveSession.OS == "windows" {
					search = tokenized + "\\"
				} else {
					search = tokenized + "/"
				}
			} else {
				search = tokenized
			}
			if strings.HasPrefix(search, lastPath) {
				completion.Suggestions = append(completion.Suggestions, search)
			}
		}
	default:
		filtered := []*sliverpb.FileInfo{}
		for _, f := range dirList.Files {
			if strings.HasPrefix(f.Name, lastPath) {
				filtered = append(filtered, f)
			}
		}

		for _, f := range filtered {
			tokenized := addSpaceTokens(f.Name)
			search := ""
			if f.IsDir {
				if core.ActiveSession.OS == "windows" {
					search = tokenized + "\\"
				} else {
					search = tokenized + "/"
				}
			} else {
				search = tokenized
			}
			if strings.HasPrefix(search, lastPath) {
				completion.Suggestions = append(completion.Suggestions, search)
			}
		}
	}

	return lastPath, []*readline.CompletionGroup{completion}
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
