// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package adaptivecard

import "strings"

//  - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#newlines-for-adaptive-cards
//  - https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features

// Newline and break statement patterns stripped out of text content sent to
// Microsoft Teams (by request).
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

	// Used with MessageCard format to emulate newlines, incompatible with
	// Adaptive Card format (displays as literal values).
	breakStatement = "<br>"
)

// ConvertEOL converts \r\n (windows), \r (mac) and \n (unix) into \n\n.
//
// This function is intended for processing text for use in an Adaptive Card
// TextBlock element. The goal is to provide spacing in rendered text display
// comparable to native display.
//
// NOTE: There are known discrepancies in the way that Microsoft Teams renders
// text in desktop, web and mobile, so even with using this helper function
// some differences are to be expected.
//
//   - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#newlines-for-adaptive-cards
//   - https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features
func ConvertEOL(s string) string {
	s = strings.ReplaceAll(s, windowsEOLEscaped, unixEOLActual+unixEOLActual)
	s = strings.ReplaceAll(s, windowsEOLActual, unixEOLActual+unixEOLActual)
	s = strings.ReplaceAll(s, macEOLActual, unixEOLActual+unixEOLActual)
	s = strings.ReplaceAll(s, macEOLEscaped, unixEOLActual+unixEOLActual)
	s = strings.ReplaceAll(s, unixEOLEscaped, unixEOLActual+unixEOLActual)

	return s
}

// ConvertBreakToEOL converts <br> statements into \n\n to provide comparable
// spacing in Adaptive Card TextBlock elements.
//
// This function is intended for processing text for use in an Adaptive Card
// TextBlock element. The goal is to provide spacing in rendered text display
// comparable to native display.
//
// The primary use case of this function is to process text that was
// previously formatted in preparation for use in a MessageCard; the
// MessageCard format supports <br> statements for text spacing/formatting
// where the Adaptive Card format does not.
//
//   - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#newlines-for-adaptive-cards
//   - https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features
func ConvertBreakToEOL(s string) string {
	return strings.ReplaceAll(s, breakStatement, unixEOLActual+unixEOLActual)
}
