package generate

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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ProfilesStageCmd - Generate an encrypted/compressed implant binary based on a profile
func ProfilesStageCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")
	profileName := args[0]
	if handled, err := canonicalProfileStage(cmd, con, profileName); handled {
		if err != nil {
			con.PrintErrorf("Profile stage build failed: %s\n", err)
		}
		return
	}
	aesEncryptKey, _ := cmd.Flags().GetString("aes-encrypt-key")
	aesEncryptIv, _ := cmd.Flags().GetString("aes-encrypt-iv")
	rc4EncryptKey, _ := cmd.Flags().GetString("rc4-encrypt-key")
	prependSize, _ := cmd.Flags().GetBool("prepend-size")
	compressF, _ := cmd.Flags().GetString("compress")
	compress := strings.ToLower(compressF)
	save, _ := cmd.Flags().GetString("save")

	profile := GetImplantProfileByName(profileName, con)
	if profile == nil {
		con.PrintErrorf("Profile not found\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Compiling, please wait ...", ctrl)
	stage2, err := con.Rpc.GenerateStage(context.Background(), &clientpb.GenerateStageReq{
		Profile:       profileName,
		Name:          name,
		AESEncryptKey: aesEncryptKey,
		AESEncryptIv:  aesEncryptIv,
		RC4EncryptKey: rc4EncryptKey,
		PrependSize:   prependSize,
		Compress:      compress,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	saveTo, err := saveLocation(save, stage2.File.Name, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	err = os.WriteFile(saveTo, stage2.File.Data, 0o700)
	if err != nil {
		con.PrintErrorf("Failed to write to: %s\n", saveTo)
		return
	}
	con.PrintInfof("Implant saved to %s\n", saveTo)
}

func canonicalProfileStage(cmd *cobra.Command, con *console.SliverClient, name string) (bool, error) {
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	doc, err := GetProfileDefinition(ctx, name, con)
	if status.Code(err) == codes.Unimplemented {
		if stageHasTypedInputs(cmd) {
			return true, errors.New("the server does not support YAML per-build inputs")
		}
		return false, nil
	}
	if err != nil {
		return true, err
	}
	if doc.Migrated {
		if stageHasTypedInputs(cmd) {
			return true, errors.New("migrated legacy profiles do not accept --input-artifact or --secret-file with profiles stage")
		}
		if doc.MigrationError != "" {
			con.PrintWarnf("Profile %s requires legacy stage generation: %s\n", name, doc.MigrationError)
		}
		return false, nil
	}
	if err := validateAuthoredStageCommand(cmd, doc.FinalArtifact); err != nil {
		return true, err
	}
	request, err := profileExecuteRequestFromFlags(cmd, name)
	if err != nil {
		return true, err
	}
	if err := validateDeclaredProfileInputs(doc.YAML, request.Inputs); err != nil {
		return true, err
	}
	record, file, err := ExecuteProfile(ctx, request, con)
	if err != nil {
		return true, err
	}
	if !proto.Equal(doc.FinalArtifact, record.GetFinalArtifact().GetType()) {
		return true, errors.New("completed artifact type differs from the profile definition")
	}
	if !isRawStageArtifact(record.GetFinalArtifact().GetType()) {
		return true, fmt.Errorf("profile build %s did not produce a stage/raw artifact", record.GetID())
	}
	save, _ := cmd.Flags().GetString("save")
	path, err := saveLocation(save, file.Name, con)
	if err != nil {
		return true, err
	}
	if err := os.WriteFile(path, file.Data, 0700); err != nil {
		return true, err
	}
	con.PrintInfof("Profile %s stage build %s saved to %s\n", name, record.ID, path)
	return true, nil
}

func stageHasTypedInputs(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("input-artifact") || cmd.Flags().Changed("secret-file")
}

func validateAuthoredStageCommand(cmd *cobra.Command, artifact *clientpb.BuildArtifactType) error {
	if !isRawStageArtifact(artifact) {
		return fmt.Errorf("profile final artifact is %s/%s; profiles stage requires stage/raw", artifact.GetKind(), artifact.GetContainer())
	}
	for _, flag := range []string{"aes-encrypt-key", "aes-encrypt-iv", "rc4-encrypt-key", "prepend-size", "compress"} {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("--%s cannot override an authored YAML profile; declare stage transforms in the profile YAML", flag)
		}
	}
	return nil
}

func isRawStageArtifact(artifact *clientpb.BuildArtifactType) bool {
	return artifact.GetKind() == "stage" && artifact.GetContainer() == "raw"
}
