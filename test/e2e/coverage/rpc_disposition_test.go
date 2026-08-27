package coverage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestRPCDispositionRegistryMatchesGeneratedSliverRPC(t *testing.T) {
	service := rpcpb.File_rpcpb_services_proto.Services().ByName("SliverRPC")
	if service == nil {
		t.Fatal("generated rpcpb descriptor has no SliverRPC service")
	}

	registry := map[string]RPCDisposition{}
	for _, disposition := range ComprehensiveRPCDispositions() {
		if strings.TrimSpace(disposition.Method) == "" {
			t.Fatal("RPC disposition has a blank method")
		}
		if _, duplicate := registry[disposition.Method]; duplicate {
			t.Fatalf("duplicate RPC disposition for %s", disposition.Method)
		}
		if strings.TrimSpace(disposition.Reason) == "" {
			t.Fatalf("RPC disposition %s has no rationale", disposition.Method)
		}
		if !validRPCDispositionClass(disposition.Class) {
			t.Fatalf("RPC disposition %s has invalid class %q", disposition.Method, disposition.Class)
		}
		if disposition.Class == RPCServerOnly && disposition.ImplantTraffic {
			t.Fatalf("server-only RPC %s cannot declare implant traffic", disposition.Method)
		}
		if (disposition.Class == RPCCommandCovered || disposition.Class == RPCCommandDeferred || disposition.Class == RPCImplantLifecycle) && !disposition.ImplantTraffic {
			t.Fatalf("implant RPC %s must declare implant traffic", disposition.Method)
		}
		registry[disposition.Method] = disposition
	}

	classCounts := map[RPCDispositionClass]int{}
	requestMetadataCount := 0
	for index := 0; index < service.Methods().Len(); index++ {
		method := service.Methods().Get(index)
		name := string(method.Name())
		disposition, ok := registry[name]
		if !ok {
			t.Errorf("generated SliverRPC method %s is unclassified", name)
			continue
		}
		delete(registry, name)
		classCounts[disposition.Class]++

		if (method.IsStreamingClient() || method.IsStreamingServer()) && (disposition.Class == RPCCommandCovered || disposition.Class == RPCCommandDeferred || disposition.Class == RPCImplantLifecycle) {
			t.Errorf("finite implant RPC %s unexpectedly uses a stream", name)
		}

		hasRequest := hasTargetRequestMetadata(method.Input())
		if hasRequest {
			requestMetadataCount++
		}
		if want := dispositionCarriesTargetRequest(disposition); hasRequest != want {
			t.Errorf("SliverRPC.%s target Request metadata = %t, want %t for %s", name, hasRequest, want, disposition.Class)
		}

		messageType, err := protoregistry.GlobalTypes.FindMessageByName(method.Input().FullName())
		if err != nil {
			t.Errorf("resolve generated input type for SliverRPC.%s: %v", name, err)
			continue
		}
		directMessage := sliverpb.MsgNumber(messageType.New().Interface()) != 0
		if directMessage && !disposition.ImplantTraffic {
			t.Errorf("SliverRPC.%s has a direct implant message number but is classified without implant traffic", name)
		}
		if disposition.Class == RPCCommandCovered && !directMessage {
			t.Errorf("covered command SliverRPC.%s has no direct implant message number", name)
		}
	}
	for stale := range registry {
		t.Errorf("RPC disposition %s has no generated SliverRPC method", stale)
	}

	t.Logf("classified %d generated RPCs: %d server-only, %d covered commands, %d deferred commands, %d lifecycle, %d tunnel/interactive; %d carry target Request metadata",
		service.Methods().Len(),
		classCounts[RPCServerOnly],
		classCounts[RPCCommandCovered],
		classCounts[RPCCommandDeferred],
		classCounts[RPCImplantLifecycle],
		classCounts[RPCTunnelInteractive],
		requestMetadataCount,
	)
}

func TestRPCDispositionCoveredMethodsMatchCatalog(t *testing.T) {
	covered := map[string]struct{}{}
	for _, disposition := range ComprehensiveRPCDispositions() {
		if disposition.Class == RPCCommandCovered {
			covered[disposition.Method] = struct{}{}
		}
	}

	catalog := map[string]struct{}{}
	for _, expectation := range ComprehensiveCatalog() {
		method := expectation.GRPCMethod
		catalog[method] = struct{}{}
		if _, ok := covered[method]; !ok {
			t.Errorf("catalog method %s is not classified as a covered implant command", method)
		}
	}
	for method := range covered {
		if _, ok := catalog[method]; !ok {
			t.Errorf("covered implant command %s has no ComprehensiveCatalog scenario", method)
		}
	}
}

func TestGlobalMarkdownExposesFiniteRPCDenominator(t *testing.T) {
	markdown := string(renderGlobalMarkdown(GlobalReport{
		RPCDispositions: ComprehensiveRPCDispositions(),
	}))
	for _, required := range []string{
		"finite implant-command denominator is 67 unique methods: 41 covered",
		"26 explicitly deferred",
		"| Ping | COVERED |",
		"| Reconfigure | DEFERRED |",
		"| ExecWasmExtension | DEFERRED |",
	} {
		if !strings.Contains(markdown, required) {
			t.Errorf("global Markdown omitted %q", required)
		}
	}
}

func validRPCDispositionClass(class RPCDispositionClass) bool {
	switch class {
	case RPCServerOnly, RPCCommandCovered, RPCCommandDeferred, RPCImplantLifecycle, RPCTunnelInteractive:
		return true
	default:
		return false
	}
}

func hasTargetRequestMetadata(message protoreflect.MessageDescriptor) bool {
	request := message.Fields().ByNumber(9)
	return request != nil &&
		request.Kind() == protoreflect.MessageKind &&
		request.Message().FullName() == "commonpb.Request"
}

func dispositionCarriesTargetRequest(disposition RPCDisposition) bool {
	switch disposition.Class {
	case RPCCommandCovered, RPCCommandDeferred, RPCImplantLifecycle:
		return true
	case RPCServerOnly:
		// ShellcodeEncoder retains a common Request field for wire compatibility,
		// but its implementation encodes bytes entirely on the server.
		return disposition.Method == "ShellcodeEncoder"
	case RPCTunnelInteractive:
		switch disposition.Method {
		case "CreateSocks", "CloseSocks", "CreateTunnel", "CloseTunnel", "TunnelData":
			return false
		default:
			return true
		}
	default:
		panic(fmt.Sprintf("unknown RPC disposition class %q", disposition.Class))
	}
}
