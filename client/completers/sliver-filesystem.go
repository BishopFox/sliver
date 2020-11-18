package completers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/connection"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/maxlandon/readline"
)

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

func completeRemotePath(last string) (string, *readline.CompletionGroup) {

	// Completions
	completion := &readline.CompletionGroup{
		Name:        "(sliver) local path",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}

	// 1) Get the absolute path. There are two cases:
	//      - The path is "rounded" with a slash: no filter to keep.
	//      - The path is not a slash: a filter to keep for later.
	// We keep a boolean for remembering which case we found
	linePath := ""
	lastPath := ""
	switch cctx.Context.Sliver.OS {
	case "windows":
		if strings.HasSuffix(last, "\\") {
			linePath = last // Trim the non needed slash
		} else if last == "" {
			linePath = "."
		} else {
			splitPath := strings.Split(last, "\\")
			linePath = strings.Join(splitPath[:len(splitPath)-1], "\\") + "\\"
			lastPath = splitPath[len(splitPath)-1]
		}
	default:
		if strings.HasSuffix(last, "/") {
			// If the the line is just "/", it means we start from filesystem root
			if last == "/" {
				linePath = "/"
			} else if last == "~/" {
				// If we look for "~", we need to build the path manually
				linePath = filepath.Join("/home", cctx.Context.Sliver.Username)

			} else if strings.HasPrefix(last, "~/") && last != "~/" {
				// If we used the "~" at the beginning, we still need to build the path
				homePath := filepath.Join("/home", cctx.Context.Sliver.Username)
				linePath = filepath.Join(homePath, strings.TrimPrefix(last, "~/"))
			} else {
				// Trim the non needed slash
				linePath = strings.TrimSuffix(last, "/")
			}
		} else if strings.HasPrefix(last, "~/") && last != "~/" {
			// If we used the "~" at the beginning, we still need to build the path
			homePath := filepath.Join("/home", cctx.Context.Sliver.Username)
			linePath = filepath.Join(homePath, filepath.Dir(strings.TrimPrefix(last, "~/")))
			lastPath = filepath.Base(last)

		} else if last == "" {
			linePath = "."
		} else {
			// linePath = last
			linePath = filepath.Dir(last)
			lastPath = filepath.Base(last)
		}
	}

	// 2) We take the absolute path we found, and get all dirs in it.
	var dirs []string

	dirList, err := connection.RPC.Ls(context.Background(), &sliverpb.LsReq{
		Request: &commonpb.Request{
			SessionID: cctx.Context.Sliver.ID,
		},
		Path: linePath,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	}

	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			dirs = append(dirs, fileInfo.Name)
		}
	}

	switch lastPath {
	case "":
		for _, dir := range dirs {
			tokenized := addSpaceTokens(dir)
			search := ""
			if cctx.Context.Sliver.OS == "windows" {
				search = tokenized + "\\"
			} else {
				search = tokenized + "/"
			}
			if strings.HasPrefix(search, lastPath) {
				tokenized := addSpaceTokens(search)
				completion.Suggestions = append(completion.Suggestions, tokenized)
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
			search := ""
			if cctx.Context.Sliver.OS == "windows" {
				search = tokenized + "\\"
			} else {
				search = tokenized + "/"
			}
			if strings.HasPrefix(search, lastPath) {
				completion.Suggestions = append(completion.Suggestions, tokenized)
			}
		}
	}

	return lastPath, completion
}

func completeRemotePathAndFiles(last string) (string, *readline.CompletionGroup) {
	// Completions
	completion := &readline.CompletionGroup{
		Name:        "(console) local directory/files)",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}

	// 1) Get the absolute path. There are two cases:
	//      - The path is "rounded" with a slash: no filter to keep.
	//      - The path is not a slash: a filter to keep for later.
	// We keep a boolean for remembering which case we found
	linePath := ""
	lastPath := ""
	switch cctx.Context.Sliver.OS {
	case "windows":
		if strings.HasSuffix(last, "\\") {
			linePath = last // Trim the non needed slash
		} else if last == "" {
			linePath = "."
		} else {
			splitPath := strings.Split(last, "\\")
			linePath = strings.Join(splitPath[:len(splitPath)-1], "\\") + "\\"
			lastPath = splitPath[len(splitPath)-1]
		}
	default:
		if strings.HasSuffix(last, "/") {
			// If the the line is just "/", it means we start from filesystem root
			if last == "/" {
				linePath = "/"
			} else if last == "~/" {
				// If we look for "~", we need to build the path manually
				linePath = filepath.Join("/home", cctx.Context.Sliver.Username)

			} else if strings.HasPrefix(last, "~/") && last != "~/" {
				// If we used the "~" at the beginning, we still need to build the path
				homePath := filepath.Join("/home", cctx.Context.Sliver.Username)
				linePath = filepath.Join(homePath, strings.TrimPrefix(last, "~/"))
			} else {
				// Trim the non needed slash
				linePath = strings.TrimSuffix(last, "/")
			}
		} else if strings.HasPrefix(last, "~/") && last != "~/" {
			// If we used the "~" at the beginning, we still need to build the path
			homePath := filepath.Join("/home", cctx.Context.Sliver.Username)
			linePath = filepath.Join(homePath, filepath.Dir(strings.TrimPrefix(last, "~/")))
			lastPath = filepath.Base(last)

		} else if last == "" {
			linePath = "."
		} else {
			// linePath = last
			linePath = filepath.Dir(last)
			lastPath = filepath.Base(last)
		}
	}

	dirList, err := connection.RPC.Ls(context.Background(), &sliverpb.LsReq{
		Request: &commonpb.Request{
			SessionID: cctx.Context.Sliver.ID,
		},
		Path: linePath,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	}

	switch lastPath {
	case "":
		for _, f := range dirList.Files {
			tokenized := addSpaceTokens(f.Name)
			search := ""
			if f.IsDir {
				if cctx.Context.Sliver.OS == "windows" {
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
				if cctx.Context.Sliver.OS == "windows" {
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

	return lastPath, completion
}
