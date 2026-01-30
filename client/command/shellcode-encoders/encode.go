package shellcodeencoders

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ShellcodeEncodersEncodeCmd - Encode shellcode files with server encoders.
func ShellcodeEncodersEncodeCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	encoderFlag, _ := cmd.Flags().GetString("encoder")
	if strings.TrimSpace(encoderFlag) == "" {
		con.PrintErrorf("Encoder is required (use --encoder). See %s for options.\n", consts.ShellcodeEncodersStr)
		return
	}

	encoderName := normalizeShellcodeEncoderName(encoderFlag)
	encoderEnum, ok := shellcodeEncoderEnum(encoderName)
	if !ok {
		con.PrintErrorf("Unknown encoder %q (see %s for supported encoders)\n", encoderFlag, consts.ShellcodeEncodersStr)
		return
	}

	archFlag, _ := cmd.Flags().GetString("arch")
	arch := normalizeShellcodeArch(archFlag)
	if arch == "" {
		arch = "amd64"
	}

	iterations, _ := cmd.Flags().GetInt("iterations")
	if iterations <= 0 {
		iterations = 1
	}

	badCharsRaw, _ := cmd.Flags().GetString("bad-chars")
	badChars, err := decodeHexBytes("bad-chars", badCharsRaw)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	output, _ := cmd.Flags().GetString("output")
	outputInfo, err := outputInfoForArgs(output, len(args))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()

	hadError := false
	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			con.PrintErrorf("Failed to read %s: %s\n", path, err)
			hadError = true
			continue
		}

		resp, err := con.Rpc.ShellcodeEncoder(grpcCtx, &clientpb.ShellcodeEncodeReq{
			Encoder:      encoderEnum,
			Architecture: arch,
			Iterations:   uint32(iterations),
			BadChars:     badChars,
			Data:         data,
			Request:      con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("Failed to encode %s: %s\n", path, err)
			hadError = true
			continue
		}
		if resp.GetResponse() != nil && resp.GetResponse().GetErr() != "" {
			con.PrintErrorf("Failed to encode %s: %s\n", path, resp.GetResponse().GetErr())
			hadError = true
			continue
		}

		outputPath, err := resolveOutputPath(path, output, outputInfo, encoderName, len(args) > 1)
		if err != nil {
			con.PrintErrorf("Failed to write %s: %s\n", path, err)
			hadError = true
			continue
		}

		if err := os.WriteFile(outputPath, resp.Data, 0o644); err != nil {
			con.PrintErrorf("Failed to write %s: %s\n", outputPath, err)
			hadError = true
			continue
		}

		con.PrintInfof("Shellcode written to %s (%d bytes)\n", outputPath, len(resp.Data))
	}

	if hadError {
		con.PrintWarnf("One or more files failed to encode\n")
	}
}

func decodeHexBytes(label string, raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "0x")
	trimmed = strings.TrimPrefix(trimmed, "0X")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	if trimmed == "" {
		return nil, nil
	}
	data, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be hex encoded: %w", label, err)
	}
	return data, nil
}

func outputInfoForArgs(output string, argCount int) (os.FileInfo, error) {
	if output == "" {
		return nil, nil
	}
	info, err := os.Stat(output)
	if err != nil {
		if os.IsNotExist(err) {
			if argCount > 1 {
				if mkErr := os.MkdirAll(output, 0o755); mkErr != nil {
					return nil, fmt.Errorf("failed to create output directory %s: %w", output, mkErr)
				}
				info, err = os.Stat(output)
				if err != nil {
					return nil, fmt.Errorf("failed to stat output directory %s: %w", output, err)
				}
				return info, nil
			}
			return nil, nil
		}
		return nil, err
	}
	if argCount > 1 && !info.IsDir() {
		return nil, fmt.Errorf("output must be a directory when encoding multiple files")
	}
	return info, nil
}

func resolveOutputPath(inputPath string, output string, outputInfo os.FileInfo, encoderName string, multiple bool) (string, error) {
	suffix := "." + strings.ReplaceAll(encoderName, "_", "-")
	base := filepath.Base(inputPath)
	if output == "" {
		return filepath.Join(filepath.Dir(inputPath), base+suffix), nil
	}
	if multiple {
		if outputInfo == nil || !outputInfo.IsDir() {
			return "", fmt.Errorf("output must be a directory when encoding multiple files")
		}
		return filepath.Join(output, base+suffix), nil
	}
	if outputInfo != nil && outputInfo.IsDir() {
		return filepath.Join(output, base+suffix), nil
	}
	return output, nil
}
