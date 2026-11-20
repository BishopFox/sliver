package rpc

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// profileArtifactMatches checks the final output that a consumer can use.
// The build's completed record is checked again because a profile may be
// edited between the initial lookup and execution.
func profileArtifactMatches(artifact *clientpb.BuildArtifactType, kind, container, goos string) bool {
	return artifact != nil && artifact.Kind == kind && artifact.Container == container && artifact.GOOS == goos && artifact.GOARCH != ""
}

func sameConsumerArtifact(a, b *clientpb.BuildArtifactType) bool {
	return a != nil && b != nil && proto.Equal(a, b)
}

//nolint:gocyclo // Profile resolution, execution, and artifact ownership share one outcome.
func (rpc *Server) buildProfileBytesForConsumer(ctx context.Context, profileName, buildName,
	kind, container, goos string) ([]byte, error) {
	if profileName == "" {
		return nil, status.Error(codes.InvalidArgument, "missing build profile name")
	}
	document, err := rpc.GetBuildProfile(ctx, &clientpb.BuildProfileRef{Name: profileName})
	if err != nil {
		return nil, err
	}
	if document.GetMigrationError() != "" {
		return nil, status.Errorf(codes.FailedPrecondition, "profile %q cannot be represented as a build pipeline: %s", profileName, document.GetMigrationError())
	}
	if !profileArtifactMatches(document.GetFinalArtifact(), kind, container, goos) {
		return nil, status.Errorf(codes.InvalidArgument, "profile %q must end in %s/%s for %s", profileName, kind, container, goos)
	}
	record, err := rpc.ExecuteProfileBuild(ctx, &clientpb.BuildProfileExecuteRequest{Name: profileName, BuildName: buildName})
	if err != nil {
		return nil, err
	}
	if record == nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.FinalArtifact == nil {
		failure := "profile build did not complete"
		if record != nil && record.Failure != nil && record.Failure.Message != "" {
			failure = record.Failure.Message
		}
		return nil, status.Error(codes.FailedPrecondition, failure)
	}
	if !profileArtifactMatches(record.FinalArtifact.Type, kind, container, goos) ||
		!sameConsumerArtifact(document.FinalArtifact, record.FinalArtifact.Type) {
		return nil, status.Error(codes.FailedPrecondition, "profile final artifact changed during build")
	}
	response, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		return nil, err
	}
	if response == nil || response.File == nil || response.File.Name == "" || len(response.File.Data) == 0 || response.Metadata == nil ||
		!sameConsumerArtifact(response.Metadata.Type, record.FinalArtifact.Type) ||
		response.File.Name != response.Metadata.FileName ||
		response.File.Name != record.FinalArtifact.FileName ||
		response.Metadata.SizeBytes != uint64(len(response.File.Data)) ||
		response.Metadata.SizeBytes != record.FinalArtifact.SizeBytes ||
		response.Metadata.SHA256 != record.FinalArtifact.SHA256 {
		return nil, status.Error(codes.Internal, fmt.Sprintf("profile %q stored artifact differs from its build record", profileName))
	}
	return response.File.Data, nil
}
