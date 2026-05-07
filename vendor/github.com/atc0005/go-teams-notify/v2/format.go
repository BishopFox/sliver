// Copyright 2021 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package goteamsnotify

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

/////////////////////////////////////////////////////////////////////////
// NOTE: The contents of this file are deprecated. See the Deprecated
// indicators in this file for intended replacements.
//
// Please submit a bug report if you find exported code in this file which
// does *not* already have a replacement elsewhere in this library.
/////////////////////////////////////////////////////////////////////////

// Newline patterns stripped out of text content sent to Microsoft Teams (by
// request) and replacement break value used to provide equivalent formatting
// for MessageCard payloads in Microsoft Teams.
const (

	// CR LF \r\n (windows)
	windowsEOLActual  = "\r\n"
	windowsEOLEscaped = `\r\n`

	// CF \r (mac)
	macEOLActual  = "\r"
	macEOLEscaped = `\r`

	// LF \n (unix)
	unixEOLActual  = "\n"
	unixEOLEscaped = `\n`

	// Used by Teams to separate lines
	breakStatement = "<br>"
)

// Even though Microsoft Teams doesn't show the additional newlines,
// https://messagecardplayground.azurewebsites.net/ DOES show the results
// as a formatted code block. Including the newlines now is an attempt at
// "future proofing" the codeblock support in MessageCard values sent to
// Microsoft Teams.
const (

	// msTeamsCodeBlockSubmissionPrefix is the prefix appended to text input
	// to indicate that the text should be displayed as a codeblock by
	// Microsoft Teams for MessageCard payloads.
	msTeamsCodeBlockSubmissionPrefix string = "\n```\n"
	// msTeamsCodeBlockSubmissionPrefix string = "```"

	// msTeamsCodeBlockSubmissionSuffix is the suffix appended to text input
	// to indicate that the text should be displayed as a codeblock by
	// Microsoft Teams for MessageCard payloads.
	msTeamsCodeBlockSubmissionSuffix string = "```\n"
	// msTeamsCodeBlockSubmissionSuffix string = "```"

	// msTeamsCodeSnippetSubmissionPrefix is the prefix appended to text input
	// to indicate that the text should be displayed as a code formatted
	// string of text by Microsoft Teams for MessageCard payloads.
	msTeamsCodeSnippetSubmissionPrefix string = "`"

	// msTeamsCodeSnippetSubmissionSuffix is the suffix appended to text input
	// to indicate that the text should be displayed as a code formatted
	// string of text by Microsoft Teams for MessageCard payloads.
	msTeamsCodeSnippetSubmissionSuffix string = "`"
)

// TryToFormatAsCodeBlock acts as a wrapper for FormatAsCodeBlock. If an
// error is encountered in the FormatAsCodeBlock function, this function will
// return the original string, otherwise if no errors occur the newly formatted
// string will be returned.
//
// This function is intended for processing text intended for a MessageCard.
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
//
// Deprecated: use messagecard.TryToFormatAsCodeBlock instead.
func TryToFormatAsCodeBlock(input string) string {
	result, err := FormatAsCodeBlock(input)
	if err != nil {
		logger.Printf("TryToFormatAsCodeBlock: error occurred when calling FormatAsCodeBlock: %v\n", err)
		logger.Println("TryToFormatAsCodeBlock: returning original string")
		return input
	}

	logger.Println("TryToFormatAsCodeBlock: no errors occurred when calling FormatAsCodeBlock")
	return result
}

// TryToFormatAsCodeSnippet acts as a wrapper for FormatAsCodeSnippet. If an
// error is encountered in the FormatAsCodeSnippet function, this function
// will return the original string, otherwise if no errors occur the newly
// formatted string will be returned.
//
// This function is intended for processing text intended for a MessageCard.
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
//
// Deprecated: use messagecard.TryToFormatAsCodeSnippet instead.
func TryToFormatAsCodeSnippet(input string) string {
	result, err := FormatAsCodeSnippet(input)
	if err != nil {
		logger.Printf("TryToFormatAsCodeSnippet: error occurred when calling FormatAsCodeBlock: %v\n", err)
		logger.Println("TryToFormatAsCodeSnippet: returning original string")
		return input
	}

	logger.Println("TryToFormatAsCodeSnippet: no errors occurred when calling FormatAsCodeSnippet")
	return result
}

// FormatAsCodeBlock accepts an arbitrary string, quoted or not, and calls a
// helper function which attempts to format as a valid Markdown code block for
// submission to Microsoft Teams.
//
// This function is intended for processing text intended for a MessageCard.
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
//
// Deprecated: use messagecard.FormatAsCodeBlock instead.
func FormatAsCodeBlock(input string) (string, error) {
	if input == "" {
		return "", errors.New("received empty string, refusing to format")
	}

	result, err := formatAsCode(
		input,
		msTeamsCodeBlockSubmissionPrefix,
		msTeamsCodeBlockSubmissionSuffix,
	)

	return result, err
}

// FormatAsCodeSnippet accepts an arbitrary string, quoted or not, and calls a
// helper function which attempts to format as a single-line valid Markdown
// code snippet for submission to Microsoft Teams.
//
// This function is intended for processing text intended for a MessageCard.
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
//
// Deprecated: use messagecard.FormatAsCodeSnippet instead.
func FormatAsCodeSnippet(input string) (string, error) {
	if input == "" {
		return "", errors.New("received empty string, refusing to format")
	}

	result, err := formatAsCode(
		input,
		msTeamsCodeSnippetSubmissionPrefix,
		msTeamsCodeSnippetSubmissionSuffix,
	)

	return result, err
}

// formatAsCode is a helper function which accepts an arbitrary string, quoted
// or not, a desired prefix and a suffix for the string and attempts to format
// as a valid Markdown formatted code sample for submission to Microsoft
// Teams. This helper function is intended for processing text intended for a
// MessageCard.
//
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
func formatAsCode(input string, prefix string, suffix string) (string, error) {
	var err error
	var byteSlice []byte

	switch {
	// required; protects against slice out of range panics
	case input == "":
		return "", errors.New("received empty string, refusing to format as code block")

	// If the input string is already valid JSON, don't double-encode and
	// escape the content
	case json.Valid([]byte(input)):
		logger.Printf("formatAsCode: input string already valid JSON; input: %+v", input)
		logger.Printf("formatAsCode: Calling json.RawMessage([]byte(input)); input: %+v", input)

		// FIXME: Is json.RawMessage() really needed if the input string is
		// *already* JSON? https://golang.org/pkg/encoding/json/#RawMessage
		// seems to imply a different use case.
		byteSlice = json.RawMessage([]byte(input))
		//
		// From light testing, it appears to not be necessary:
		//
		// logger.Printf("formatAsCode: Skipping json.RawMessage, converting string directly to byte slice; input: %+v", input)
		// byteSlice = []byte(input)

	default:
		logger.Printf("formatAsCode: input string not valid JSON; input: %+v", input)
		logger.Printf("formatAsCode: Calling json.Marshal(input); input: %+v", input)
		byteSlice, err = json.Marshal(input)
		if err != nil {
			return "", err
		}
	}

	logger.Println("formatAsCode: byteSlice as string:", string(byteSlice))

	var prettyJSON bytes.Buffer

	logger.Println("formatAsCode: calling json.Indent")
	err = json.Indent(&prettyJSON, byteSlice, "", "\t")
	if err != nil {
		return "", err
	}
	formattedJSON := prettyJSON.String()

	logger.Println("formatAsCode: Formatted JSON:", formattedJSON)

	// handle both cases: where the formatted JSON string was not wrapped with
	// double-quotes and when it was
	codeContentForSubmission := prefix + strings.Trim(formattedJSON, "\"") + suffix

	logger.Printf("formatAsCode: formatted JSON as-is:\n%s\n\n", formattedJSON)
	logger.Printf("formatAsCode: formatted JSON wrapped with code prefix/suffix: \n%s\n\n", codeContentForSubmission)

	// err should be nil if everything worked as expected
	return codeContentForSubmission, err
}

// ConvertEOLToBreak converts \r\n (windows), \r (mac) and \n (unix) into <br>
// statements.
//
// This function is intended for processing text intended for a MessageCard.
// Using this helper function for text intended for an Adaptive Card is
// unsupported and unlikely to produce the desired results.
//
// Deprecated: use messagecard.ConvertEOLToBreak instead.
func ConvertEOLToBreak(s string) string {
	logger.Printf("ConvertEOLToBreak: Received %#v", s)

	s = strings.ReplaceAll(s, windowsEOLActual, breakStatement)
	s = strings.ReplaceAll(s, windowsEOLEscaped, breakStatement)
	s = strings.ReplaceAll(s, macEOLActual, breakStatement)
	s = strings.ReplaceAll(s, macEOLEscaped, breakStatement)
	s = strings.ReplaceAll(s, unixEOLActual, breakStatement)
	s = strings.ReplaceAll(s, unixEOLEscaped, breakStatement)

	logger.Printf("ConvertEOLToBreak: Returning %#v", s)

	return s
}
