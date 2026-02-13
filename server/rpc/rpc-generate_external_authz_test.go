package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestAuthorizeBuilderForExternalBuildAllowsAssignedBuilder(t *testing.T) {
	const (
		buildID      = "build-authz-allow"
		builderName  = "builder-authz-allow"
		operatorName = "operator-authz-allow"
	)
	err := core.AddBuilder(&clientpb.Builder{Name: builderName, OperatorName: operatorName})
	if err != nil {
		t.Fatalf("add builder: %v", err)
	}
	t.Cleanup(func() {
		core.RemoveBuilder(builderName)
		core.RemoveExternalBuildAssignment(buildID)
	})
	core.TrackExternalBuildAssignment(buildID, builderName, operatorName)

	rpc := &Server{}
	err = rpc.authorizeBuilderForExternalBuild(contextWithCommonName(operatorName), builderName, buildID)
	if err != nil {
		t.Fatalf("authorizeBuilderForExternalBuild() unexpected error: %v", err)
	}
}

func TestAuthorizeBuilderForExternalBuildRejectsForeignBuilder(t *testing.T) {
	const (
		buildID      = "build-authz-foreign"
		ownerBuilder = "builder-owner"
		otherBuilder = "builder-other"
		operatorName = "operator-shared"
	)
	if err := core.AddBuilder(&clientpb.Builder{Name: ownerBuilder, OperatorName: operatorName}); err != nil {
		t.Fatalf("add owner builder: %v", err)
	}
	if err := core.AddBuilder(&clientpb.Builder{Name: otherBuilder, OperatorName: operatorName}); err != nil {
		t.Fatalf("add other builder: %v", err)
	}
	t.Cleanup(func() {
		core.RemoveBuilder(ownerBuilder)
		core.RemoveBuilder(otherBuilder)
		core.RemoveExternalBuildAssignment(buildID)
	})
	core.TrackExternalBuildAssignment(buildID, ownerBuilder, operatorName)

	rpc := &Server{}
	err := rpc.authorizeBuilderForExternalBuild(contextWithCommonName(operatorName), otherBuilder, buildID)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v (%v)", status.Code(err), err)
	}
}

func TestAuthorizeBuilderForExternalBuildRejectsWrongOperator(t *testing.T) {
	const (
		buildID          = "build-authz-wrong-op"
		builderName      = "builder-authz-wrong-op"
		ownerOperator    = "operator-owner"
		attackerOperator = "operator-attacker"
	)
	err := core.AddBuilder(&clientpb.Builder{Name: builderName, OperatorName: ownerOperator})
	if err != nil {
		t.Fatalf("add builder: %v", err)
	}
	t.Cleanup(func() {
		core.RemoveBuilder(builderName)
		core.RemoveExternalBuildAssignment(buildID)
	})
	core.TrackExternalBuildAssignment(buildID, builderName, ownerOperator)

	rpc := &Server{}
	err = rpc.authorizeBuilderForExternalBuild(contextWithCommonName(attackerOperator), builderName, buildID)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v (%v)", status.Code(err), err)
	}
}

func TestAuthorizeBuilderOperatorForExternalBuildRejectsForeignOperator(t *testing.T) {
	const (
		buildID       = "build-op-authz"
		builderName   = "builder-op-authz"
		ownerOperator = "operator-a"
		otherOperator = "operator-b"
	)
	core.TrackExternalBuildAssignment(buildID, builderName, ownerOperator)
	t.Cleanup(func() {
		core.RemoveExternalBuildAssignment(buildID)
	})

	rpc := &Server{}
	err := rpc.authorizeBuilderOperatorForExternalBuild(contextWithCommonName(otherOperator), buildID)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v (%v)", status.Code(err), err)
	}
}

func TestSplitExternalBuildEventData(t *testing.T) {
	builderName, buildID, err := splitExternalBuildEventData([]byte("builder:name:abc123"))
	if err != nil {
		t.Fatalf("splitExternalBuildEventData() unexpected error: %v", err)
	}
	if builderName != "builder:name" {
		t.Fatalf("expected builder:name, got %s", builderName)
	}
	if buildID != "abc123" {
		t.Fatalf("expected abc123, got %s", buildID)
	}
}

func TestSplitExternalBuildProgressData(t *testing.T) {
	buildID, details, err := splitExternalBuildProgressData([]byte("abc123:done"))
	if err != nil {
		t.Fatalf("splitExternalBuildProgressData() unexpected error: %v", err)
	}
	if buildID != "abc123" {
		t.Fatalf("expected abc123, got %s", buildID)
	}
	if details != "done" {
		t.Fatalf("expected done, got %s", details)
	}
}

func contextWithCommonName(commonName string) context.Context {
	cert := &x509.Certificate{
		Subject: pkix.Name{CommonName: commonName},
	}
	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		},
	}
	return peer.NewContext(context.Background(), &peer.Peer{AuthInfo: tlsInfo})
}
