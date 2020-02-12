package main

import (
	"regexp"
	"strings"
)

// AllCaps is a regex to test if a string identifier is made of
// all upper case letters.
var allCaps = regexp.MustCompile("^[A-Z0-9]+$")

// popCount counts number of bits 'set' in mask.
func popCount(mask uint) uint {
	m := uint32(mask)
	n := uint(0)
	for i := uint32(0); i < 32; i++ {
		if m&(1<<i) != 0 {
			n++
		}
	}
	return n
}

// pad makes sure 'n' aligns on 4 bytes.
func pad(n int) int {
	return (n + 3) & ^3
}

// splitAndTitle takes a string, splits it by underscores, capitalizes the
// first letter of each chunk, and smushes'em back together.
func splitAndTitle(s string) string {
	// If the string is all caps, lower it and capitalize first letter.
	if allCaps.MatchString(s) {
		return strings.Title(strings.ToLower(s))
	}

	// If the string has no underscores, capitalize it and leave it be.
	if i := strings.Index(s, "_"); i == -1 {
		return strings.Title(s)
	}

	// Now split the name at underscores, capitalize the first
	// letter of each chunk, and smush'em back together.
	chunks := strings.Split(s, "_")
	for i, chunk := range chunks {
		chunks[i] = strings.Title(strings.ToLower(chunk))
	}
	return strings.Join(chunks, "")
}
