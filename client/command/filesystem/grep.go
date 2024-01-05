package filesystem

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

const maxLinesDisplayed = 50

func processFlags(searchPattern string, insensitive bool, exact bool) string {
	if !insensitive && !exact {
		return searchPattern
	}

	var processedSearchPattern = searchPattern

	flagsAtBeginning, _ := regexp.Compile(`^\(\?.*\)`)
	flagsSpecifiedIndex := flagsAtBeginning.FindStringIndex(searchPattern)

	if insensitive {
		if flagsSpecifiedIndex != nil {
			// If we matched, flagsSpecifiedIndex[0] will always be 0 (the start of the string)
			flags := searchPattern[:flagsSpecifiedIndex[1]]
			if !strings.Contains(flags, "i") {
				processedSearchPattern = "(?i" + searchPattern[2:]
			}
		} else {
			processedSearchPattern = "(?i)" + processedSearchPattern
		}
	}

	// For exact matches, we will replace any start and end of line anchors with (^|\s) and (\s|$) respectively
	flagsSpecifiedIndex = flagsAtBeginning.FindStringIndex(processedSearchPattern)

	if exact {
		var endIndexOfFlags = 0

		if flagsSpecifiedIndex != nil {
			endIndexOfFlags = flagsSpecifiedIndex[1]
		}
		if strings.HasPrefix(processedSearchPattern[endIndexOfFlags:], "^") {
			processedSearchPattern = processedSearchPattern[:endIndexOfFlags] + strings.Replace(processedSearchPattern[endIndexOfFlags:], "^", `(^|\s)`, 1)
		} else {
			processedSearchPattern = `(^|\s)` + processedSearchPattern
		}

		if strings.HasSuffix(processedSearchPattern, "$") {
			processedSearchPattern = processedSearchPattern[:len(processedSearchPattern)-1] + `(\s|$)`
		} else {
			processedSearchPattern = processedSearchPattern + `(\s|$)`
		}
	}

	return processedSearchPattern
}

func GrepCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	searchPattern := args[0]
	searchPath := args[1]

	recursive, _ := cmd.Flags().GetBool("recursive")
	insensitive, _ := cmd.Flags().GetBool("insensitive")
	exact, _ := cmd.Flags().GetBool("exact")

	searchPatternProcessed := processFlags(searchPattern, insensitive, exact)

	// Sanity check the search pattern to validate that it is a valid regex
	_, err := regexp.Compile(searchPattern)
	if err != nil {
		con.PrintErrorf("%s is not a valid regex: %s\n", searchPattern, err)
		return
	}

	// Context overrides individual values for before and after
	var linesBefore int32 = 0
	var linesAfter int32 = 0

	if cmd.Flags().Changed("context") {
		linesBefore, _ = cmd.Flags().GetInt32("context")
		linesAfter = linesBefore
	} else {
		linesBefore, _ = cmd.Flags().GetInt32("before")
		linesAfter, _ = cmd.Flags().GetInt32("after")
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Searching for %s in %s ...", searchPattern, searchPath), ctrl)

	grep, err := con.Rpc.Grep(context.Background(), &sliverpb.GrepReq{
		Request:       con.ActiveTarget.Request(cmd),
		SearchPattern: searchPatternProcessed,
		Path:          searchPath,
		Recursive:     recursive,
		LinesBefore:   linesBefore,
		LinesAfter:    linesAfter,
	})

	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if grep.Response != nil && grep.Response.Async {
		con.AddBeaconCallback(grep.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, grep)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			// Using args[0] so that the operator's original input is preserved
			printGrep(grep, args[0], searchPath, cmd, con)
		})
		con.PrintAsyncResponse(grep.Response)
	} else {
		printGrep(grep, args[0], searchPath, cmd, con)
	}
}

// printGrep - Print the results from the grep operation to stdout
func printGrep(grep *sliverpb.Grep, searchPattern string, searchPath string, cmd *cobra.Command, con *console.SliverClient) {
	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("name")
	colorize, _ := cmd.Flags().GetBool("colorize-output")
	userLootFileType, _ := cmd.Flags().GetString("file-type")
	currentDateTime := time.Now().Format("2006-01-02_150405")
	if !cmd.Flags().Changed("lootName") {
		// If the loot name has not been specified by the operator, generate one
		con.GetActiveSessionConfig()
		lootName = fmt.Sprintf("grep from %s on %s (%s)", searchPath, con.GetActiveSessionConfig().ID, currentDateTime)
	}
	if grep.Response != nil && grep.Response.Err != "" {
		con.PrintErrorf("%s\n", grep.Response.Err)
		return
	}

	grepResults, numberOfResults, binaryFilesMatched := printGrepResults(grep.Results, colorize, true)
	if len(grepResults) > maxLinesDisplayed {
		// If there are more than maxResultsDisplayed results, then loot the rest so that we do not overfill the console
		saveLoot = true
	}

	var resultQualifier string
	var displayLimit string

	if numberOfResults == 1 {
		resultQualifier = "line"
	} else {
		resultQualifier = "lines"
	}

	if len(grepResults) > maxLinesDisplayed {
		displayLimit = fmt.Sprintf(" (%d displayed, including context if available)", maxLinesDisplayed)
	}

	con.Printf("Found %d matching %s for %s in %s%s\n\n", numberOfResults, resultQualifier, searchPattern, searchPath, displayLimit)
	if len(grepResults) > maxLinesDisplayed {
		con.Println(strings.Join(grepResults[:maxLinesDisplayed+1], "\n"))
	} else {
		con.Println(strings.Join(grepResults, "\n"))
	}

	if len(grepResults) > maxLinesDisplayed {
		con.PrintWarnf("Number of output lines exceed %d. The full output is available in the loot file \"%s\"\n", maxLinesDisplayed, lootName)
	}
	if len(binaryFilesMatched) > 0 {
		con.PrintInfof("The following binary files contained one or more matches that may not be reflected above:\n")
		for _, fileName := range binaryFilesMatched {
			con.Printf("\t%s\n", fileName)
		}
	}

	if saveLoot {
		// Do not allow escape sequences in the output when looting
		grepResultsForLoot, numberOfResults, binaryFilesMatched := printGrepResults(grep.Results, false, false)

		lootFileName := fmt.Sprintf("grep_%s_%s.txt", con.GetActiveSessionConfig().ID, currentDateTime)
		flags := []string{}
		if recursive, _ := cmd.Flags().GetBool("recursive"); recursive {
			flags = append(flags, "recursive")
		}
		if exact, _ := cmd.Flags().GetBool("exact"); exact {
			flags = append(flags, "exact match")
		}
		if insensitive, _ := cmd.Flags().GetBool("insensitive"); insensitive {
			flags = append(flags, "case insensitive search")
		}

		// Add a header to the looted grep results to indicate what the pattern was and what was searched
		grepResultsString := fmt.Sprintf("Search pattern: %s\nSearch location: %s\n", searchPattern, grep.SearchPathAbsolute)
		if len(flags) > 0 {
			grepResultsString += fmt.Sprintf("Search flags: %s\n", strings.Join(flags, ", "))
		}
		grepResultsString += fmt.Sprintf("Number of Results: %d\n\n%s", numberOfResults, strings.Join(grepResultsForLoot, "\n"))
		if len(binaryFilesMatched) > 0 {
			grepResultsString += "\nThe following binary files contained one or more matches:\n"
			grepResultsString += strings.Join(binaryFilesMatched, "\n")
		}
		fileType := loot.ValidateLootFileType(userLootFileType, []byte(grepResultsString))
		loot.LootText(grepResultsString, lootName, lootFileName, fileType, con)
	}

}

// grepLineResult - Add color or formatting for results for console output
func grepLineResult(positions []*sliverpb.GrepLinePosition, line string, colorize bool, allowFormatting bool) string {
	var result string = ""
	var matchOutput func(a ...interface{}) string
	var previousPositionEnd int32 = 0

	if colorize {
		matchOutput = color.New(color.FgRed, color.Bold).SprintFunc()
	} else if allowFormatting {
		matchOutput = color.New(color.Bold).SprintFunc()
	}

	for idx, position := range positions {
		if colorize || allowFormatting {
			result += line[previousPositionEnd:position.Start]
			result += matchOutput(line[position.Start:position.End])
			if idx == len(positions)-1 {
				result += line[position.End:]
			} else {
				previousPositionEnd = position.End
			}
		} else {
			result = line
		}
	}

	return result
}

// printGrepResults - Take the results from the implant and put them together for output to the console or loot
func printGrepResults(results map[string]*sliverpb.GrepResultsForFile, colorize bool, allowFormatting bool) ([]string, int, []string) {
	var resultOutput []string
	var numberOfResults = 0
	binaryFilesMatched := []string{}

	for fileName, result := range results {
		if result.IsBinary {
			if len(result.FileResults) > 0 {
				binaryFilesMatched = append(binaryFilesMatched, fileName)
			}
		}
		for _, lineResult := range result.FileResults {
			line := lineResult.Line
			if len(lineResult.LinesBefore) > 0 || len(lineResult.LinesAfter) > 0 {
				resultOutput = append(resultOutput, fmt.Sprintf("%s (match on line %d)\n", fileName, lineResult.LineNumber))
				if len(lineResult.LinesBefore) > 0 {
					resultOutput = append(resultOutput, strings.Join(lineResult.LinesBefore, "\n"))
				}
				resultOutput = append(resultOutput, grepLineResult(lineResult.Positions, line, colorize, allowFormatting))
				if len(lineResult.LinesAfter) > 0 {
					resultOutput = append(resultOutput, strings.Join(lineResult.LinesAfter, "\n"))
				}
				resultOutput = append(resultOutput, "\n")
			} else {
				if lineResult.LineNumber == -1 {
					resultOutput = append(resultOutput, fmt.Sprintf("Error when reading file %s: %s", fileName, line))
				} else {
					resultOutput = append(resultOutput, fmt.Sprintf("%s: Line %d: %s", fileName, lineResult.LineNumber, grepLineResult(lineResult.Positions, line, colorize, allowFormatting)))
				}
			}
			numberOfResults += 1
		}
	}

	return resultOutput, numberOfResults, binaryFilesMatched
}
