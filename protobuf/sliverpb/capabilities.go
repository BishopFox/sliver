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

// CapabilitySocksFlowControlV1 indicates that a SOCKS endpoint supports the
// cumulative acknowledgements and fixed send window carried by SocksData.
const CapabilitySocksFlowControlV1 uint64 = uint64(ImplantCapability_IMPLANT_CAPABILITY_SOCKS_FLOW_CONTROL_V1)

// SocksFlowControlWindowV1 is the maximum number of unacknowledged data frames
// allowed in either direction of a V1 flow-controlled SOCKS tunnel.
const SocksFlowControlWindowV1 = 64

// SocksFlowControlAckBatchV1 controls how frequently receivers emit cumulative
// acknowledgements while a stream remains active.
const SocksFlowControlAckBatchV1 = 16
