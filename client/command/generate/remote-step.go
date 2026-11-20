package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// remoteStepRequested treats -E, --remote-step, and a builder pin as requests
// for the assigned-step remote pipeline.
func remoteStepRequested(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	external, _ := cmd.Flags().GetBool("external-builder")
	remote, _ := cmd.Flags().GetBool("remote-step")
	pin, _ := cmd.Flags().GetString("builder")
	return external || remote || pin != ""
}

// The neutral core spec does not yet model every compatibility flag. Reject
// remote requests that would change their meaning instead of silently
// dropping settings.
func remoteStepCompatibility(cmd *cobra.Command, config *clientpb.ImplantConfig) error {
	if cmd == nil || config == nil {
		return errors.New("missing generate command or config")
	}
	for _, name := range []string{"reconnect", "poll-timeout", "max-errors", "strategy", "c2profile", "key-exchange", "tcp-comms"} {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--%s is not represented in the neutral core spec", name)
		}
	}
	if config.IncludeWG || hasWGEndpoint(config) {
		return errors.New("WireGuard peer settings are not represented in the neutral core spec")
	}
	if config.HTTPC2ConfigName != "" && config.HTTPC2ConfigName != consts.DefaultC2Profile {
		return errors.New("custom HTTP C2 profiles are not represented in the neutral core spec")
	}
	trafficEncoders, _ := cmd.Flags().GetString("traffic-encoders")
	if config.TrafficEncodersEnabled || cmd.Flags().Changed("traffic-encoders") && trafficEncoders != "" {
		return errors.New("traffic encoder assets are not represented in the neutral core spec")
	}
	return nil
}

func hasWGEndpoint(config *clientpb.ImplantConfig) bool {
	for _, endpoint := range config.GetC2() {
		if endpoint != nil && (strings.HasPrefix(endpoint.URL, "wg://") || strings.HasPrefix(endpoint.URL, "wireguard://")) {
			return true
		}
	}
	return false
}

// remoteStepBuild uses the server's assigned-step executor and retrieves the
// exact final artifact recorded by the server. Metadata donor bytes are
// carried as bounded per-build inputs, never as plan options.
func remoteStepBuild(cmd *cobra.Command, name string, config *clientpb.ImplantConfig,
	metadata *clientpb.SpoofMetadataConfig, save string, con *console.SliverClient) (*commonpb.File, error) {
	request, err := planRequestFromFlags(cmd, config.GetIsBeacon())
	if err != nil {
		return nil, err
	}
	// An interactive encoder choice lives in config, while plan previews read
	// flags only. Carry the selected transform into this actual build request.
	if config.GetFormat() == clientpb.OutputFormat_SHELLCODE && !cmd.Flags().Changed("shellcode-encoder") {
		encoder := ""
		switch config.GetShellcodeEncoder() {
		case clientpb.ShellcodeEncoder_NONE:
		case clientpb.ShellcodeEncoder_SHIKATA_GA_NAI:
			encoder = "builtin/shikata-ga-nai"
		case clientpb.ShellcodeEncoder_XOR:
			encoder = "builtin/xor"
		case clientpb.ShellcodeEncoder_XOR_DYNAMIC:
			encoder = "builtin/xor-dynamic"
		default:
			return nil, fmt.Errorf("unsupported selected shellcode encoder %d", config.GetShellcodeEncoder())
		}
		if encoder != "" {
			request.Transforms = append(request.Transforms, &clientpb.BuildTransformRequest{
				Spec: &clientpb.BuildSpecRef{Name: encoder},
			})
		}
	}
	if name != "" {
		request.Name = name
	}
	if metadata != nil {
		if err := addRemoteMetadataInputs(request, metadata); err != nil {
			return nil, err
		}
	}
	if request.Provider == nil || request.Provider.Mode == "automatic" {
		request.Provider = &clientpb.BuildProviderPolicy{Mode: "remote"}
	}
	con.PrintInfof("Using assigned-step remote build pipeline\n")
	start := time.Now()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	record, err := con.Rpc.ExecuteBuild(ctx, request)
	if err != nil {
		return nil, err
	}
	if record.GetState() != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.GetFinalArtifact() == nil {
		if failure := record.GetFailure(); failure != nil && failure.Message != "" {
			return nil, fmt.Errorf("remote build failed: %s", failure.Message)
		}
		return nil, errors.New("remote build did not produce a final artifact")
	}
	response, err := con.Rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.GetID()})
	if err != nil {
		return nil, err
	}
	file, err := verifiedRemoteStepArtifact(record, response)
	if err != nil {
		return nil, err
	}
	saveTo, err := outputSavePath(save, file.Name, config, con)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(saveTo, file.Data, 0o700); err != nil {
		return nil, err
	}
	con.PrintInfof("Build completed in %s; artifact %s saved to %s\n", time.Since(start).Round(time.Second), record.GetID(), saveTo)
	return file, nil
}

func verifiedRemoteStepArtifact(record *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) (*commonpb.File, error) {
	if record == nil || record.FinalArtifact == nil || record.FinalArtifact.Type == nil || response == nil ||
		response.Metadata == nil || response.Metadata.Type == nil {
		return nil, errors.New("remote build returned an empty or untyped artifact")
	}
	file := response.File
	if file == nil || !util.ValidPortableFileName(file.Name) || len(file.Data) == 0 {
		return nil, errors.New("remote build returned an empty artifact or unsafe filename")
	}
	hash := sha256.Sum256(file.Data)
	checksum := hex.EncodeToString(hash[:])
	if checksum != record.FinalArtifact.SHA256 || checksum != response.Metadata.SHA256 ||
		uint64(len(file.Data)) != record.FinalArtifact.SizeBytes || uint64(len(file.Data)) != response.Metadata.SizeBytes ||
		!proto.Equal(record.FinalArtifact.Type, response.Metadata.Type) ||
		file.Name != record.FinalArtifact.FileName || file.Name != response.Metadata.FileName {
		return nil, errors.New("remote build artifact differs from its frozen record")
	}
	return file, nil
}

func addRemoteMetadataInputs(request *clientpb.BuildRequest, metadata *clientpb.SpoofMetadataConfig) error {
	if request == nil || metadata == nil || metadata.GetPE() == nil || metadata.GetPE().GetSource() == nil {
		return errors.New("missing PE metadata donor")
	}
	serialized, err := proto.MarshalOptions{Deterministic: true}.Marshal(metadata)
	if err != nil {
		return err
	}
	if len(serialized) == 0 || len(serialized) > 128<<20 || len(metadata.GetPE().GetSource().GetData()) == 0 ||
		len(metadata.GetPE().GetSource().GetData()) > 64<<20 {
		return errors.New("PE metadata input is empty or exceeds its size limit")
	}
	hash := sha256.Sum256(serialized)
	refName := "metadata-sha256:" + hex.EncodeToString(hash[:])
	matched := false
	for _, transform := range request.Transforms {
		if transform.GetSpec().GetName() != "builtin/pe-metadata" {
			continue
		}
		for _, option := range transform.Options {
			if option.GetName() == "donorArtifact" && strings.Contains(string(option.GetValueJSON()), refName) {
				matched = true
			}
		}
	}
	if !matched {
		return errors.New("PE metadata changed while preparing the remote build")
	}
	request.Inputs = append(request.Inputs,
		&clientpb.BuildInput{Kind: "input-artifact", Name: refName, Data: metadata.GetPE().GetSource().GetData()},
		&clientpb.BuildInput{Kind: "metadata-config", Name: refName, Data: serialized},
	)
	return nil
}
