package main

const (
	normal  = "\033[0m"
	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	orange  = "\033[33m"
	blue    = "\033[34m"
	purple  = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[37m"
	bold    = "\033[1m"
	clearln = "\r\x1b[2K"
	upN     = "\033[%dA"
	downN   = "\033[%dB"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)
