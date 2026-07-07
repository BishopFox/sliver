package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listCredentialsToolName = "list_credentials"
)

// list_credentials 结果
type credentialInfo struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Plaintext      string `json:"plaintext,omitempty"`
	Hash           string `json:"hash,omitempty"`
	HashType       string `json:"hash_type,omitempty"`
	IsCracked      bool   `json:"is_cracked"`
	OriginHostUUID string `json:"origin_host_uuid,omitempty"`
	Collection     string `json:"collection,omitempty"`
}

type listCredentialsResult struct {
	Credentials []credentialInfo `json:"credentials"`
	Count       int              `json:"count"`
}

// list_credentials 处理器
func (s *SliverMCPServer) listCredentialsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	return s.handleListCredentials(ctx)
}

func (s *SliverMCPServer) handleListCredentials(ctx context.Context) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(listCredentialsToolName, "", "")

	credsResp, err := s.Rpc.Creds(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list credentials", err), nil
	}

	result := listCredentialsResult{
		Credentials: make([]credentialInfo, 0, len(credsResp.Credentials)),
	}

	for _, cred := range credsResp.Credentials {
		if cred == nil {
			continue
		}
		result.Credentials = append(result.Credentials, credentialInfo{
			ID:             cred.ID,
			Username:       cred.Username,
			Plaintext:      cred.Plaintext,
			Hash:           cred.Hash,
			HashType:       formatHashType(cred.HashType),
			IsCracked:      cred.IsCracked,
			OriginHostUUID: cred.OriginHostUUID,
			Collection:     cred.Collection,
		})
	}

	result.Count = len(result.Credentials)
	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// formatHashType 将 HashType 枚举转换为字符串
func formatHashType(hashType clientpb.HashType) string {
	// 使用 clientpb 中定义的常量映射
	switch hashType {
	case clientpb.HashType_MD5:
		return "MD5"
	case clientpb.HashType_MD4:
		return "MD4"
	case clientpb.HashType_SHA1:
		return "SHA1"
	case clientpb.HashType_SHA2_224:
		return "SHA2-224"
	case clientpb.HashType_SHA2_256:
		return "SHA2-256"
	case clientpb.HashType_SHA2_384:
		return "SHA2-384"
	case clientpb.HashType_SHA2_512:
		return "SHA2-512"
	case clientpb.HashType_SHA3_224:
		return "SHA3-224"
	case clientpb.HashType_SHA3_256:
		return "SHA3-256"
	case clientpb.HashType_SHA3_384:
		return "SHA3-384"
	case clientpb.HashType_SHA3_512:
		return "SHA3-512"
	case clientpb.HashType_RIPEMD_160:
		return "RIPEMD-160"
	case clientpb.HashType_BLAKE2B_256:
		return "BLAKE2b-256"
	case clientpb.HashType_GOST_R_32_11_2012_256:
		return "GOST R 34.11-2012 256-bit"
	case clientpb.HashType_GOST_R_32_11_2012_512:
		return "GOST R 34.11-2012 512-bit"
	case clientpb.HashType_GOST_R_34_11_94:
		return "GOST R 34.11-94"
	case clientpb.HashType_GPG:
		return "GPG"
	case clientpb.HashType_HALF_MD5:
		return "Half MD5"
	case clientpb.HashType_KECCAK_224:
		return "Keccak-224"
	case clientpb.HashType_KECCAK_256:
		return "Keccak-256"
	case clientpb.HashType_KECCAK_384:
		return "Keccak-384"
	case clientpb.HashType_KECCAK_512:
		return "Keccak-512"
	case clientpb.HashType_WHIRLPOOL:
		return "Whirlpool"
	case clientpb.HashType_SIPHASH:
		return "SipHash"
	case clientpb.HashType_MD5_UTF16LE:
		return "MD5 (UTF-16LE)"
	case clientpb.HashType_SHA1_UTF16LE:
		return "SHA1 (UTF-16LE)"
	case clientpb.HashType_SHA256_UTF16LE:
		return "SHA256 (UTF-16LE)"
	case clientpb.HashType_SHA384_UTF16LE:
		return "SHA384 (UTF-16LE)"
	case clientpb.HashType_SHA512_UTF16LE:
		return "SHA512 (UTF-16LE)"
	case clientpb.HashType_NTLM:
		return "NTLM"
	case clientpb.HashType_KERBEROS_5_TGS:
		return "Kerberos 5 TGS"
	case clientpb.HashType_KERBEROS_5_TGS_3DES:
		return "Kerberos 5 TGS (3DES)"
	case clientpb.HashType_KERBEROS_5_PA_ETYPE_23:
		return "Kerberos 5 PA-ETYPE-23"
	case clientpb.HashType_KERBEROS_5_PA_ETYPE_17:
		return "Kerberos 5 PA-ETYPE-17"
	case clientpb.HashType_KERBEROS_5_PA_ETYPE_18:
		return "Kerberos 5 PA-ETYPE-18"
	default:
		return fmt.Sprintf("Unknown(%d)", hashType)
	}
}
