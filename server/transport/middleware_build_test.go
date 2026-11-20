package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestBuildStepMethodsAllowRestrictedBuilder(t *testing.T) {
	interceptor := permissionsUnaryServerInterceptor()
	builder := context.WithValue(context.Background(), Operator, &models.Operator{
		Name: "restricted-builder", PermissionBuilder: true,
	})
	for _, method := range []string{
		rpcpb.SliverRPC_BuilderUpdateStatus_FullMethodName,
		rpcpb.SliverRPC_BuilderGetStep_FullMethodName,
		rpcpb.SliverRPC_BuilderSubmitStep_FullMethodName,
	} {
		called := false
		_, err := interceptor(builder, &commonpb.Empty{}, &grpc.UnaryServerInfo{FullMethod: method},
			func(context.Context, interface{}) (interface{}, error) {
				called = true
				return &commonpb.Empty{}, nil
			})
		if err != nil || !called {
			t.Errorf("builder method %s denied before handler: %v", method, err)
		}
	}
	for _, method := range []string{
		rpcpb.SliverRPC_ExecuteBuild_FullMethodName,
		rpcpb.SliverRPC_GetBuildArtifact_FullMethodName,
	} {
		called := false
		_, err := interceptor(builder, &commonpb.Empty{}, &grpc.UnaryServerInfo{FullMethod: method},
			func(context.Context, interface{}) (interface{}, error) {
				called = true
				return &commonpb.Empty{}, nil
			})
		if status.Code(err) != codes.PermissionDenied || called {
			t.Errorf("restricted builder reached operator method %s: called=%v err=%v", method, called, err)
		}
	}
}

func TestBuildAuditRequestRedactsPrivateMaterial(t *testing.T) {
	const secret = "m7-audit-private-material"
	cases := []struct {
		method  string
		request proto.Message
	}{
		{rpcpb.SliverRPC_PlanBuild_FullMethodName, &clientpb.BuildRequest{
			Name: "audit-build", C2: []*clientpb.BuildEndpoint{{Address: "mtls://" + secret + ":8888"}},
			CoreOptions: []*clientpb.BuildOption{{Name: "key", ValueJSON: []byte(secret)}},
			Inputs:      []*clientpb.BuildInput{{Kind: "secret", Name: "key", Data: []byte(secret)}},
		}},
		{rpcpb.SliverRPC_ExecuteBuild_FullMethodName, &clientpb.BuildRequest{
			Name: "audit-build", Inputs: []*clientpb.BuildInput{{Kind: "secret", Data: []byte(secret)}},
		}},
		{rpcpb.SliverRPC_ExecuteProfileBuild_FullMethodName, &clientpb.BuildProfileExecuteRequest{
			Name: "audit-profile", Inputs: []*clientpb.BuildInput{{Kind: "secret", Data: []byte(secret)}},
		}},
		{rpcpb.SliverRPC_Generate_FullMethodName, &clientpb.GenerateReq{
			Name: "audit-legacy", Config: &clientpb.ImplantConfig{Assets: []*commonpb.File{{Data: []byte(secret)}}},
		}},
		{rpcpb.SliverRPC_GenerateExternal_FullMethodName, &clientpb.ExternalGenerateReq{
			Name: "audit-external", Config: &clientpb.ImplantConfig{Assets: []*commonpb.File{{Data: []byte(secret)}}},
		}},
		{rpcpb.SliverRPC_GenerateStage_FullMethodName, &clientpb.GenerateStageReq{
			Name: "audit-stage", AESEncryptKey: secret, RC4EncryptKey: secret,
		}},
		{rpcpb.SliverRPC_GenerateExternalSaveBuild_FullMethodName, &clientpb.ExternalImplantBinary{
			Name: "audit-external", File: &commonpb.File{Data: []byte(secret)},
		}},
		{rpcpb.SliverRPC_ValidateBuildProfile_FullMethodName, &clientpb.BuildProfileImportRequest{YAML: secret}},
		{rpcpb.SliverRPC_ImportBuildProfile_FullMethodName, &clientpb.BuildProfileImportRequest{YAML: secret}},
		{rpcpb.SliverRPC_BuilderUpdateStatus_FullMethodName, &clientpb.BuilderStatusReport{
			Name: "audit-builder", SessionToken: []byte(secret),
		}},
		{rpcpb.SliverRPC_BuilderGetStep_FullMethodName, &clientpb.BuilderStepRef{
			BuilderName: "audit-builder", AssignmentID: "assigned", SessionToken: []byte(secret),
		}},
		{rpcpb.SliverRPC_BuilderSubmitStep_FullMethodName, &clientpb.BuilderStepResult{
			BuilderName: "audit-builder", SessionToken: []byte(secret),
			Output: &clientpb.BuildArtifactPayload{Data: []byte(secret)},
		}},
	}
	for _, test := range cases {
		t.Run(test.method, func(t *testing.T) {
			original := proto.Clone(test.request)
			sanitized := sanitizeAuditRequest(test.method, test.request)
			encoded, err := json.Marshal(sanitized)
			if err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(test.request, original) {
				t.Fatal("audit sanitizer modified the handler request")
			}
			if strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), base64.StdEncoding.EncodeToString([]byte(secret))) {
				t.Fatalf("audit request exposed private material: %s", encoded)
			}
		})
	}
}

func TestBuildPayloadLoggingSuppressesPrivateMaterial(t *testing.T) {
	originalUnary, originalStream := serverConfig.Logs.GRPCUnaryPayloads, serverConfig.Logs.GRPCStreamPayloads
	serverConfig.Logs.GRPCUnaryPayloads, serverConfig.Logs.GRPCStreamPayloads = true, true
	t.Cleanup(func() {
		serverConfig.Logs.GRPCUnaryPayloads, serverConfig.Logs.GRPCStreamPayloads = originalUnary, originalStream
	})
	for _, method := range []string{
		rpcpb.SliverRPC_PlanBuild_FullMethodName,
		rpcpb.SliverRPC_ExecuteBuild_FullMethodName,
		rpcpb.SliverRPC_ExecuteProfileBuild_FullMethodName,
		rpcpb.SliverRPC_GetBuildArtifact_FullMethodName,
		rpcpb.SliverRPC_Generate_FullMethodName,
		rpcpb.SliverRPC_GenerateExternal_FullMethodName,
		rpcpb.SliverRPC_GenerateStage_FullMethodName,
		rpcpb.SliverRPC_GenerateExternalGetBuildConfig_FullMethodName,
		rpcpb.SliverRPC_GenerateExternalSaveBuild_FullMethodName,
		rpcpb.SliverRPC_GenerateWGClientConfig_FullMethodName,
		rpcpb.SliverRPC_BuilderUpdateStatus_FullMethodName,
		rpcpb.SliverRPC_BuilderGetStep_FullMethodName,
		rpcpb.SliverRPC_BuilderSubmitStep_FullMethodName,
	} {
		if deciderUnary(context.Background(), method, nil) {
			t.Errorf("payload logging enabled for %s", method)
		}
	}
	if deciderStream(context.Background(), rpcpb.SliverRPC_BuilderRegister_FullMethodName, nil) {
		t.Fatal("payload logging enabled for builder session token")
	}
}
