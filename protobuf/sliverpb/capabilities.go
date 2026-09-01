// Package sliverpb contains protobuf messages and protocol constants shared by
// Sliver clients, servers, and implants.
package sliverpb

// CapabilityBOFV1 indicates that an implant can execute BOFs using the
// built-in BOF v1 request fields on CallExtensionReq.
const CapabilityBOFV1 uint64 = uint64(ImplantCapability_IMPLANT_CAPABILITY_BOF_V1)

// CapabilityTunnelTerminalV1 indicates that an implant waits for every data
// frame below a TunnelClose.Sequence exclusive terminal before detaching the
// tunnel generation.
const CapabilityTunnelTerminalV1 uint64 = uint64(ImplantCapability_IMPLANT_CAPABILITY_TUNNEL_TERMINAL_V1)
