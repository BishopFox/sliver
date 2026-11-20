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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// remoteStepFlags keeps the existing -E choice and treats a builder pin as a
// remote request. --remote-step requires the neutral pipeline even when a
// legacy config cannot be represented yet.
func remoteStepFlags(cmd *cobra.Command) (requested, explicit bool, pin string) {
	if cmd == nil {
		return false, false, ""
	}
	external, _ := cmd.Flags().GetBool("external-builder")
	explicit, _ = cmd.Flags().GetBool("remote-step")
	pin, _ = cmd.Flags().GetString("builder")
	return external || explicit || pin != "", explicit, pin
}

// The neutral core spec does not yet model every compatibility flag. Only
// configurations with equivalent defaults run through assigned steps; all
// other -E requests retain their existing external build behavior.
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
	if config.IsBeacon && (config.BeaconInterval != int64(60*time.Second) || config.BeaconJitter != 0) {
		return errors.New("custom beacon timing is not represented in the neutral core spec")
	}
	if config.HTTPC2ConfigName != "" && config.HTTPC2ConfigName != consts.DefaultC2Profile {
		return errors.New("custom HTTP C2 profiles are not represented in the neutral core spec")
	}
	if config.TrafficEncodersEnabled {
		return errors.New("traffic encoder assets require the legacy build material path")
	}
	if config.Format == clientpb.OutputFormat_SHELLCODE && !explicitRemoteStep(cmd) {
		return errors.New("shellcode compatibility options use the legacy external build path")
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

func explicitRemoteStep(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("remote-step")
	return value
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
	file := response.GetFile()
	if file == nil || file.GetName() == "" || len(file.GetData()) == 0 || response.GetMetadata() == nil {
		return nil, errors.New("remote build returned an empty artifact")
	}
	hash := sha256.Sum256(file.Data)
	checksum := hex.EncodeToString(hash[:])
	if checksum != record.FinalArtifact.SHA256 || checksum != response.Metadata.SHA256 ||
		uint64(len(file.Data)) != record.FinalArtifact.SizeBytes || uint64(len(file.Data)) != response.Metadata.SizeBytes {
		return nil, errors.New("remote build artifact differs from its recorded checksum or size")
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
