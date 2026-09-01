//go:build !sliver_lint && !((darwin && (amd64 || arm64)) || (linux && (386 || amd64 || arm64)) || (windows && (386 || amd64 || arm64)))

package runner

import "github.com/bishopfox/sliver/protobuf/sliverpb"

func implantCapabilities() uint64 {
	return uint64(sliverpb.ImplantCapability_IMPLANT_CAPABILITY_TUNNEL_TERMINAL_V1)
}
