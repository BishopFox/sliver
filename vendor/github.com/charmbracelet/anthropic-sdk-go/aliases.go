// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package anthropic

import (
	"github.com/charmbracelet/anthropic-sdk-go/internal/apierror"
	"github.com/charmbracelet/anthropic-sdk-go/packages/param"
	"github.com/charmbracelet/anthropic-sdk-go/shared"
)

// aliased to make [param.APIUnion] private when embedding
type paramUnion = param.APIUnion

// aliased to make [param.APIObject] private when embedding
type paramObj = param.APIObject

type Error = apierror.Error

// This is an alias to an internal type.
type APIErrorObject = shared.APIErrorObject

// This is an alias to an internal type.
type AuthenticationError = shared.AuthenticationError

// This is an alias to an internal type.
type BillingError = shared.BillingError

// This is an alias to an internal type.
type ErrorObjectUnion = shared.ErrorObjectUnion

// This is an alias to an internal type.
type ErrorResponse = shared.ErrorResponse

// This is an alias to an internal type.
type GatewayTimeoutError = shared.GatewayTimeoutError

// This is an alias to an internal type.
type InvalidRequestError = shared.InvalidRequestError

// This is an alias to an internal type.
type NotFoundError = shared.NotFoundError

// This is an alias to an internal type.
type OverloadedError = shared.OverloadedError

// This is an alias to an internal type.
type PermissionError = shared.PermissionError

// This is an alias to an internal type.
type RateLimitError = shared.RateLimitError

// Backward-compatible type aliases for renamed types.
// These ensure existing code continues to compile after
// UserLocation and ErrorCode types were consolidated.

type WebSearchToolRequestErrorErrorCode = WebSearchToolResultErrorCode
type WebSearchToolResultErrorErrorCode = WebSearchToolResultErrorCode
type WebSearchTool20250305UserLocationParam = UserLocationParam
type WebSearchTool20260209UserLocationParam = UserLocationParam

const WebSearchToolRequestErrorErrorCodeInvalidToolInput = WebSearchToolResultErrorCodeInvalidToolInput
const WebSearchToolRequestErrorErrorCodeUnavailable = WebSearchToolResultErrorCodeUnavailable
const WebSearchToolRequestErrorErrorCodeMaxUsesExceeded = WebSearchToolResultErrorCodeMaxUsesExceeded
const WebSearchToolRequestErrorErrorCodeTooManyRequests = WebSearchToolResultErrorCodeTooManyRequests
const WebSearchToolRequestErrorErrorCodeQueryTooLong = WebSearchToolResultErrorCodeQueryTooLong
const WebSearchToolRequestErrorErrorCodeRequestTooLarge = WebSearchToolResultErrorCodeRequestTooLarge
const WebSearchToolResultErrorErrorCodeInvalidToolInput = WebSearchToolResultErrorCodeInvalidToolInput
const WebSearchToolResultErrorErrorCodeUnavailable = WebSearchToolResultErrorCodeUnavailable
const WebSearchToolResultErrorErrorCodeMaxUsesExceeded = WebSearchToolResultErrorCodeMaxUsesExceeded
const WebSearchToolResultErrorErrorCodeTooManyRequests = WebSearchToolResultErrorCodeTooManyRequests
const WebSearchToolResultErrorErrorCodeQueryTooLong = WebSearchToolResultErrorCodeQueryTooLong
const WebSearchToolResultErrorErrorCodeRequestTooLarge = WebSearchToolResultErrorCodeRequestTooLarge
