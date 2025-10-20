package utils

import (
	"bytes"
	"fmt"
	"os"
	"unicode"

	"github.com/fatih/color"
)

var Verbose = false

// checks if a byte array contains any element of another byte array
func containsBytes(data, any []byte) bool {
	for _, b := range any {
		if bytes.Contains(data, []byte{b}) {
			return true
		}
	}
	return false
}

// checks if s is ascii and printable, aka doesn't include tab, backspace, etc.
func IsASCIIPrintable(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func PrintStatus(format string, a ...interface{}) {
	color.New(color.Bold, color.FgYellow).Print("[*] ")
	fmt.Printf(format+"\n", a...)
}

func PrintSuccess(format string, a ...interface{}) {
	color.New(color.Bold, color.FgGreen).Print("[+] ")
	fmt.Printf(format+"\n", a...)
}

func PrintFatal(format string, a ...interface{}) {
	color.New(color.Bold, color.FgRed).Print("[-] ")
	fmt.Printf(format+"\n", a...)
	os.Exit(1)
}

func PrintVerbose(format string, a ...interface{}) {
	if !Verbose {
		return
	}
	yellow := color.New(color.Bold, color.FgYellow).PrintfFunc()
	yellow("[*] ")
	fmt.Printf(format+"\n", a...)
}
