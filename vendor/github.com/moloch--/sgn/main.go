package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/moloch--/sgn/config"
	sgn "github.com/moloch--/sgn/pkg"
	"github.com/moloch--/sgn/utils"

	"github.com/fatih/color"
)

func main() {

	printBanner()
	// Configure the options from the flags/config file
	opts, err := config.ConfigureOptions()
	if err != nil {
		utils.PrintFatal("%s", err)
	}

	// Setup a encoder struct
	payload := []byte{}
	encoder, err := sgn.NewEncoder(opts.Arch)
	if err != nil {
		utils.PrintFatal("%s", err)
	}
	encoder.ObfuscationLimit = opts.ObsLevel
	encoder.PlainDecoder = opts.PlainDecoder
	encoder.EncodingCount = opts.EncCount
	encoder.SaveRegisters = opts.Safe
	file, err := os.ReadFile(opts.Input)
	if err != nil {
		utils.PrintFatal("%s", err)
	}

	spinr := spinner.New(spinner.CharSets[9], 50*time.Millisecond)
	if !opts.Verbose {
		spinr.Start()
	}

	// Print encoder params...
	utils.PrintVerbose("Architecture: x%d", encoder.GetArchitecture())
	utils.PrintVerbose("Encode Count: %d", encoder.EncodingCount)
	utils.PrintVerbose("Max. Obfuscation Size: %d", encoder.ObfuscationLimit)
	utils.PrintVerbose("Bad Characters: %x", opts.BadChars)
	utils.PrintVerbose("ASCII Mode: %t", opts.AsciiPayload)
	utils.PrintVerbose("Plain Decoder: %t", encoder.PlainDecoder)
	utils.PrintVerbose("Safe Registers: %t", encoder.SaveRegisters)
	// Calculate evarage garbage instrunction size
	average, err := encoder.CalculateAverageGarbageInstructionSize()
	if err != nil {
		utils.PrintFatal("%s", err)
	}
	utils.PrintVerbose("Avg. Garbage Size: %f", average)

	if opts.BadChars != "" || opts.AsciiPayload {

		// Need to disable verbosity now
		if utils.Verbose {
			spinr.Start()
			utils.Verbose = false
		}
		spinr.Suffix = " Bruteforcing bad characters..."

		badBytes, err := hex.DecodeString(strings.ReplaceAll(opts.BadChars, `\x`, ""))
		if err != nil {
			utils.PrintFatal("%s", err)
		}

		for {
			p, err := encode(encoder, file)
			if err != nil {
				utils.PrintFatal("%s", err)
			}

			if (opts.AsciiPayload && utils.IsASCIIPrintable(string(p))) || (len(badBytes) > 0 && !bytes.Contains(p, badBytes)) {
				payload = p
				break
			}
			encoder.Seed = (encoder.Seed + 1) % 255
		}
		spinr.Stop()
		utils.PrintStatus("Success ᕕ( ᐛ )ᕗ")
	} else {
		utils.PrintVerbose("Encoding payload...")
		payload, err = encode(encoder, file)
		if err != nil {
			utils.PrintFatal("%s", err)
		}
	}

	spinr.Stop()
	if opts.Output == "" {
		opts.Output = opts.Input + ".sgn"
	}

	utils.PrintStatus("Input: %s", opts.Input)
	utils.PrintStatus("Input Size: %d", len(file))
	utils.PrintStatus("Outfile: %s", opts.Output)
	out, err := os.OpenFile(opts.Output, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.PrintFatal("%s", err)
	}
	_, err = out.Write(payload)
	if err != nil {
		utils.PrintFatal("%s", err)
	}
	outputSize := len(payload)
	if utils.Verbose {
		color.Blue("\n" + hex.Dump(payload) + "\n")
	}

	utils.PrintVerbose("Total Garbage Size: %d", encoder.ObfuscationLimit)
	utils.PrintSuccess("Final size: %d", outputSize)
	utils.PrintSuccess("All done ＼(＾O＾)／")
}

// Encode function is the primary encode method for SGN
func encode(encoder *sgn.Encoder, payload []byte) ([]byte, error) {
	red := color.New(color.Bold, color.FgRed).SprintfFunc()
	green := color.New(color.Bold, color.FgGreen).SprintfFunc()
	var final []byte
	if encoder.SaveRegisters {
		utils.PrintVerbose("Adding safe register suffix...")
		payload = append(sgn.SafeRegisterSuffix[encoder.GetArchitecture()], payload...)
	}

	// Add garbage instrctions before the ciphered decoder stub
	garbage, err := encoder.GenerateGarbageInstructions()
	if err != nil {
		return nil, err
	}
	payload = append(garbage, payload...)
	encoder.ObfuscationLimit -= len(garbage)

	utils.PrintVerbose("Ciphering payload...")
	ciperedPayload := sgn.CipherADFL(payload, encoder.Seed)
	decoderAssembly, err := encoder.NewDecoderAssembly(len(ciperedPayload))
	if err != nil {
		utils.PrintFatal("%s", err)
	}
	utils.PrintVerbose("Selected decoder: %s", green("\n%s\n", decoderAssembly))
	decoder, ok := encoder.Assemble(decoderAssembly)
	if !ok {
		return nil, errors.New("decoder assembly failed")
	}

	encodedPayload := append(decoder, ciperedPayload...)
	if encoder.PlainDecoder {
		final = encodedPayload
	} else {
		schemaSize := ((len(encodedPayload) - len(ciperedPayload)) / (encoder.GetArchitecture() / 8)) + 1
		randomSchema := encoder.NewCipherSchema(schemaSize)
		utils.PrintVerbose("Cipher schema: %s", red("\n\n%s", sgn.GetSchemaTable(randomSchema)))
		obfuscatedEncodedPayload := encoder.SchemaCipher(encodedPayload, 0, randomSchema)
		final, err = encoder.AddSchemaDecoder(obfuscatedEncodedPayload, randomSchema)
		if err != nil {
			return nil, err
		}

	}

	if encoder.SaveRegisters {
		utils.PrintVerbose("Adding safe register prefix...")
		final = append(sgn.SafeRegisterPrefix[encoder.GetArchitecture()], final...)
	}

	if encoder.EncodingCount > 1 {
		encoder.EncodingCount--
		encoder.Seed = sgn.GetRandomByte()
		final, err = encode(encoder, final)
		if err != nil {
			return nil, err
		}
	}

	return final, nil
}

func printBanner() {
	banner, _ := base64.StdEncoding.DecodeString("ICAgICAgIF9fICAgXyBfXyAgICAgICAgX18gICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgXyAKICBfX18gLyAvICAoXykgL19fX19fIF8vIC9fX19fIF8gIF9fXyBfX19fIF8gIF9fXyAgX19fIF8oXykKIChfLTwvIF8gXC8gLyAgJ18vIF8gYC8gX18vIF8gYC8gLyBfIGAvIF8gYC8gLyBfIFwvIF8gYC8gLyAKL19fXy9fLy9fL18vXy9cX1xcXyxfL1xfXy9cXyxfLyAgXF8sIC9cXyxfLyAvXy8vXy9cXyxfL18vICAKPT09PT09PT1bQXV0aG9yOi1FZ2UtQmFsY8SxLV09PT09L19fXy89PT09PT09djIuMC4xPT09PT09PT09ICAKICAgIOKUu+KUgeKUuyDvuLXjg70oYNCUwrQp776J77i1IOKUu+KUgeKUuyAgICAgICAgICAgKOODjiDjgpzQlOOCnCnjg44g77i1IOS7leaWueOBjOOBquOBhAo=")
	fmt.Println(string(banner))
}
