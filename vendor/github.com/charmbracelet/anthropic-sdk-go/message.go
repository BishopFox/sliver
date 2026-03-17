// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"time"

	"github.com/charmbracelet/anthropic-sdk-go/internal/apijson"
	"github.com/charmbracelet/anthropic-sdk-go/internal/paramutil"
	"github.com/charmbracelet/anthropic-sdk-go/internal/requestconfig"
	"github.com/charmbracelet/anthropic-sdk-go/option"
	"github.com/charmbracelet/anthropic-sdk-go/packages/param"
	"github.com/charmbracelet/anthropic-sdk-go/packages/respjson"
	"github.com/charmbracelet/anthropic-sdk-go/packages/ssestream"
	"github.com/charmbracelet/anthropic-sdk-go/shared/constant"
	"github.com/tidwall/gjson"
)

// MessageService contains methods and other services that help with interacting
// with the anthropic API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewMessageService] method instead.
type MessageService struct {
	Options []option.RequestOption
	Batches MessageBatchService
}

// NewMessageService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewMessageService(opts ...option.RequestOption) (r MessageService) {
	r = MessageService{}
	r.Options = opts
	r.Batches = NewMessageBatchService(opts...)
	return
}

// Send a structured list of input messages with text and/or image content, and the
// model will generate the next message in the conversation.
//
// The Messages API can be used for either single queries or stateless multi-turn
// conversations.
//
// Learn more about the Messages API in our
// [user guide](https://docs.claude.com/en/docs/initial-setup)
//
// Note: If you choose to set a timeout for this request, we recommend 10 minutes.
func (r *MessageService) New(ctx context.Context, body MessageNewParams, opts ...option.RequestOption) (res *Message, err error) {
	opts = slices.Concat(r.Options, opts)

	// For non-streaming requests, calculate the appropriate timeout based on maxTokens
	// and check against model-specific limits
	timeout, timeoutErr := CalculateNonStreamingTimeout(int(body.MaxTokens), body.Model, opts)
	if timeoutErr != nil {
		return nil, timeoutErr
	}
	opts = append(opts, option.WithRequestTimeout(timeout))

	path := "v1/messages"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Send a structured list of input messages with text and/or image content, and the
// model will generate the next message in the conversation.
//
// The Messages API can be used for either single queries or stateless multi-turn
// conversations.
//
// Learn more about the Messages API in our
// [user guide](https://docs.claude.com/en/docs/initial-setup)
//
// Note: If you choose to set a timeout for this request, we recommend 10 minutes.
func (r *MessageService) NewStreaming(ctx context.Context, body MessageNewParams, opts ...option.RequestOption) (stream *ssestream.Stream[MessageStreamEventUnion]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append(opts, option.WithJSONSet("stream", true))
	path := "v1/messages"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &raw, opts...)
	return ssestream.NewStream[MessageStreamEventUnion](ssestream.NewDecoder(raw), err)
}

// Count the number of tokens in a Message.
//
// The Token Count API can be used to count the number of tokens in a Message,
// including tools, images, and documents, without creating it.
//
// Learn more about token counting in our
// [user guide](https://docs.claude.com/en/docs/build-with-claude/token-counting)
func (r *MessageService) CountTokens(ctx context.Context, body MessageCountTokensParams, opts ...option.RequestOption) (res *MessageTokensCount, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "v1/messages/count_tokens"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// The properties Data, MediaType, Type are required.
type Base64ImageSourceParam struct {
	Data string `json:"data,required" format:"byte"`
	// Any of "image/jpeg", "image/png", "image/gif", "image/webp".
	MediaType Base64ImageSourceMediaType `json:"media_type,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "base64".
	Type constant.Base64 `json:"type,required"`
	paramObj
}

func (r Base64ImageSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow Base64ImageSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *Base64ImageSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Base64ImageSourceMediaType string

const (
	Base64ImageSourceMediaTypeImageJPEG Base64ImageSourceMediaType = "image/jpeg"
	Base64ImageSourceMediaTypeImagePNG  Base64ImageSourceMediaType = "image/png"
	Base64ImageSourceMediaTypeImageGIF  Base64ImageSourceMediaType = "image/gif"
	Base64ImageSourceMediaTypeImageWebP Base64ImageSourceMediaType = "image/webp"
)

type Base64PDFSource struct {
	Data      string                  `json:"data,required" format:"byte"`
	MediaType constant.ApplicationPDF `json:"media_type,required"`
	Type      constant.Base64         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		MediaType   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Base64PDFSource) RawJSON() string { return r.JSON.raw }
func (r *Base64PDFSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this Base64PDFSource to a Base64PDFSourceParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// Base64PDFSourceParam.Overrides()
func (r Base64PDFSource) ToParam() Base64PDFSourceParam {
	return param.Override[Base64PDFSourceParam](json.RawMessage(r.RawJSON()))
}

// The properties Data, MediaType, Type are required.
type Base64PDFSourceParam struct {
	Data string `json:"data,required" format:"byte"`
	// This field can be elided, and will marshal its zero value as "application/pdf".
	MediaType constant.ApplicationPDF `json:"media_type,required"`
	// This field can be elided, and will marshal its zero value as "base64".
	Type constant.Base64 `json:"type,required"`
	paramObj
}

func (r Base64PDFSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow Base64PDFSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *Base64PDFSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BashCodeExecutionOutputBlock struct {
	FileID string                           `json:"file_id,required"`
	Type   constant.BashCodeExecutionOutput `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		FileID      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BashCodeExecutionOutputBlock) RawJSON() string { return r.JSON.raw }
func (r *BashCodeExecutionOutputBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type BashCodeExecutionOutputBlockParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_output".
	Type constant.BashCodeExecutionOutput `json:"type,required"`
	paramObj
}

func (r BashCodeExecutionOutputBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BashCodeExecutionOutputBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BashCodeExecutionOutputBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BashCodeExecutionResultBlock struct {
	Content    []BashCodeExecutionOutputBlock   `json:"content,required"`
	ReturnCode int64                            `json:"return_code,required"`
	Stderr     string                           `json:"stderr,required"`
	Stdout     string                           `json:"stdout,required"`
	Type       constant.BashCodeExecutionResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ReturnCode  respjson.Field
		Stderr      respjson.Field
		Stdout      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BashCodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BashCodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ReturnCode, Stderr, Stdout, Type are required.
type BashCodeExecutionResultBlockParam struct {
	Content    []BashCodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	ReturnCode int64                               `json:"return_code,required"`
	Stderr     string                              `json:"stderr,required"`
	Stdout     string                              `json:"stdout,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_result".
	Type constant.BashCodeExecutionResult `json:"type,required"`
	paramObj
}

func (r BashCodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BashCodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BashCodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BashCodeExecutionToolResultBlock struct {
	Content   BashCodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                       `json:"tool_use_id,required"`
	Type      constant.BashCodeExecutionToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BashCodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BashCodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BashCodeExecutionToolResultBlockContentUnion contains all possible properties
// and values from [BashCodeExecutionToolResultError],
// [BashCodeExecutionResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BashCodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [BashCodeExecutionToolResultError].
	ErrorCode BashCodeExecutionToolResultErrorCode `json:"error_code"`
	Type      string                               `json:"type"`
	// This field is from variant [BashCodeExecutionResultBlock].
	Content []BashCodeExecutionOutputBlock `json:"content"`
	// This field is from variant [BashCodeExecutionResultBlock].
	ReturnCode int64 `json:"return_code"`
	// This field is from variant [BashCodeExecutionResultBlock].
	Stderr string `json:"stderr"`
	// This field is from variant [BashCodeExecutionResultBlock].
	Stdout string `json:"stdout"`
	JSON   struct {
		ErrorCode  respjson.Field
		Type       respjson.Field
		Content    respjson.Field
		ReturnCode respjson.Field
		Stderr     respjson.Field
		Stdout     respjson.Field
		raw        string
	} `json:"-"`
}

func (u BashCodeExecutionToolResultBlockContentUnion) AsResponseBashCodeExecutionToolResultError() (v BashCodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BashCodeExecutionToolResultBlockContentUnion) AsResponseBashCodeExecutionResultBlock() (v BashCodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BashCodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BashCodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BashCodeExecutionToolResultBlockParam struct {
	Content   BashCodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                            `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_tool_result".
	Type constant.BashCodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r BashCodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BashCodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BashCodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BashCodeExecutionToolResultBlockParamContentUnion struct {
	OfRequestBashCodeExecutionToolResultError *BashCodeExecutionToolResultErrorParam `json:",omitzero,inline"`
	OfRequestBashCodeExecutionResultBlock     *BashCodeExecutionResultBlockParam     `json:",omitzero,inline"`
	paramUnion
}

func (u BashCodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestBashCodeExecutionToolResultError, u.OfRequestBashCodeExecutionResultBlock)
}
func (u *BashCodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BashCodeExecutionToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestBashCodeExecutionToolResultError) {
		return u.OfRequestBashCodeExecutionToolResultError
	} else if !param.IsOmitted(u.OfRequestBashCodeExecutionResultBlock) {
		return u.OfRequestBashCodeExecutionResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestBashCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetContent() []BashCodeExecutionOutputBlockParam {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetReturnCode() *int64 {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.ReturnCode
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetStderr() *string {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.Stderr
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetStdout() *string {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.Stdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BashCodeExecutionToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestBashCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BashCodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "output_file_too_large".
	ErrorCode BashCodeExecutionToolResultErrorCode      `json:"error_code,required"`
	Type      constant.BashCodeExecutionToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BashCodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BashCodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BashCodeExecutionToolResultErrorCode string

const (
	BashCodeExecutionToolResultErrorCodeInvalidToolInput      BashCodeExecutionToolResultErrorCode = "invalid_tool_input"
	BashCodeExecutionToolResultErrorCodeUnavailable           BashCodeExecutionToolResultErrorCode = "unavailable"
	BashCodeExecutionToolResultErrorCodeTooManyRequests       BashCodeExecutionToolResultErrorCode = "too_many_requests"
	BashCodeExecutionToolResultErrorCodeExecutionTimeExceeded BashCodeExecutionToolResultErrorCode = "execution_time_exceeded"
	BashCodeExecutionToolResultErrorCodeOutputFileTooLarge    BashCodeExecutionToolResultErrorCode = "output_file_too_large"
)

// The properties ErrorCode, Type are required.
type BashCodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "output_file_too_large".
	ErrorCode BashCodeExecutionToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_tool_result_error".
	Type constant.BashCodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r BashCodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BashCodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BashCodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewCacheControlEphemeralParam() CacheControlEphemeralParam {
	return CacheControlEphemeralParam{
		Type: "ephemeral",
	}
}

// This struct has a constant value, construct it with
// [NewCacheControlEphemeralParam].
type CacheControlEphemeralParam struct {
	// The time-to-live for the cache control breakpoint.
	//
	// This may be one the following values:
	//
	// - `5m`: 5 minutes
	// - `1h`: 1 hour
	//
	// Defaults to `5m`.
	//
	// Any of "5m", "1h".
	TTL  CacheControlEphemeralTTL `json:"ttl,omitzero"`
	Type constant.Ephemeral       `json:"type,required"`
	paramObj
}

func (r CacheControlEphemeralParam) MarshalJSON() (data []byte, err error) {
	type shadow CacheControlEphemeralParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CacheControlEphemeralParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The time-to-live for the cache control breakpoint.
//
// This may be one the following values:
//
// - `5m`: 5 minutes
// - `1h`: 1 hour
//
// Defaults to `5m`.
type CacheControlEphemeralTTL string

const (
	CacheControlEphemeralTTLTTL5m CacheControlEphemeralTTL = "5m"
	CacheControlEphemeralTTLTTL1h CacheControlEphemeralTTL = "1h"
)

type CacheCreation struct {
	// The number of input tokens used to create the 1 hour cache entry.
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens,required"`
	// The number of input tokens used to create the 5 minute cache entry.
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Ephemeral1hInputTokens respjson.Field
		Ephemeral5mInputTokens respjson.Field
		ExtraFields            map[string]respjson.Field
		raw                    string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CacheCreation) RawJSON() string { return r.JSON.raw }
func (r *CacheCreation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationCharLocation struct {
	CitedText      string                `json:"cited_text,required"`
	DocumentIndex  int64                 `json:"document_index,required"`
	DocumentTitle  string                `json:"document_title,required"`
	EndCharIndex   int64                 `json:"end_char_index,required"`
	FileID         string                `json:"file_id,required"`
	StartCharIndex int64                 `json:"start_char_index,required"`
	Type           constant.CharLocation `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CitedText      respjson.Field
		DocumentIndex  respjson.Field
		DocumentTitle  respjson.Field
		EndCharIndex   respjson.Field
		FileID         respjson.Field
		StartCharIndex respjson.Field
		Type           respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationCharLocation) RawJSON() string { return r.JSON.raw }
func (r *CitationCharLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndCharIndex,
// StartCharIndex, Type are required.
type CitationCharLocationParam struct {
	DocumentTitle  param.Opt[string] `json:"document_title,omitzero,required"`
	CitedText      string            `json:"cited_text,required"`
	DocumentIndex  int64             `json:"document_index,required"`
	EndCharIndex   int64             `json:"end_char_index,required"`
	StartCharIndex int64             `json:"start_char_index,required"`
	// This field can be elided, and will marshal its zero value as "char_location".
	Type constant.CharLocation `json:"type,required"`
	paramObj
}

func (r CitationCharLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationCharLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationCharLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationContentBlockLocation struct {
	CitedText       string                        `json:"cited_text,required"`
	DocumentIndex   int64                         `json:"document_index,required"`
	DocumentTitle   string                        `json:"document_title,required"`
	EndBlockIndex   int64                         `json:"end_block_index,required"`
	FileID          string                        `json:"file_id,required"`
	StartBlockIndex int64                         `json:"start_block_index,required"`
	Type            constant.ContentBlockLocation `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CitedText       respjson.Field
		DocumentIndex   respjson.Field
		DocumentTitle   respjson.Field
		EndBlockIndex   respjson.Field
		FileID          respjson.Field
		StartBlockIndex respjson.Field
		Type            respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationContentBlockLocation) RawJSON() string { return r.JSON.raw }
func (r *CitationContentBlockLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndBlockIndex,
// StartBlockIndex, Type are required.
type CitationContentBlockLocationParam struct {
	DocumentTitle   param.Opt[string] `json:"document_title,omitzero,required"`
	CitedText       string            `json:"cited_text,required"`
	DocumentIndex   int64             `json:"document_index,required"`
	EndBlockIndex   int64             `json:"end_block_index,required"`
	StartBlockIndex int64             `json:"start_block_index,required"`
	// This field can be elided, and will marshal its zero value as
	// "content_block_location".
	Type constant.ContentBlockLocation `json:"type,required"`
	paramObj
}

func (r CitationContentBlockLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationContentBlockLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationContentBlockLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationPageLocation struct {
	CitedText       string                `json:"cited_text,required"`
	DocumentIndex   int64                 `json:"document_index,required"`
	DocumentTitle   string                `json:"document_title,required"`
	EndPageNumber   int64                 `json:"end_page_number,required"`
	FileID          string                `json:"file_id,required"`
	StartPageNumber int64                 `json:"start_page_number,required"`
	Type            constant.PageLocation `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CitedText       respjson.Field
		DocumentIndex   respjson.Field
		DocumentTitle   respjson.Field
		EndPageNumber   respjson.Field
		FileID          respjson.Field
		StartPageNumber respjson.Field
		Type            respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationPageLocation) RawJSON() string { return r.JSON.raw }
func (r *CitationPageLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndPageNumber,
// StartPageNumber, Type are required.
type CitationPageLocationParam struct {
	DocumentTitle   param.Opt[string] `json:"document_title,omitzero,required"`
	CitedText       string            `json:"cited_text,required"`
	DocumentIndex   int64             `json:"document_index,required"`
	EndPageNumber   int64             `json:"end_page_number,required"`
	StartPageNumber int64             `json:"start_page_number,required"`
	// This field can be elided, and will marshal its zero value as "page_location".
	Type constant.PageLocation `json:"type,required"`
	paramObj
}

func (r CitationPageLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationPageLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationPageLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, EndBlockIndex, SearchResultIndex, Source,
// StartBlockIndex, Title, Type are required.
type CitationSearchResultLocationParam struct {
	Title             param.Opt[string] `json:"title,omitzero,required"`
	CitedText         string            `json:"cited_text,required"`
	EndBlockIndex     int64             `json:"end_block_index,required"`
	SearchResultIndex int64             `json:"search_result_index,required"`
	Source            string            `json:"source,required"`
	StartBlockIndex   int64             `json:"start_block_index,required"`
	// This field can be elided, and will marshal its zero value as
	// "search_result_location".
	Type constant.SearchResultLocation `json:"type,required"`
	paramObj
}

func (r CitationSearchResultLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationSearchResultLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationSearchResultLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, EncryptedIndex, Title, Type, URL are required.
type CitationWebSearchResultLocationParam struct {
	Title          param.Opt[string] `json:"title,omitzero,required"`
	CitedText      string            `json:"cited_text,required"`
	EncryptedIndex string            `json:"encrypted_index,required"`
	URL            string            `json:"url,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_result_location".
	Type constant.WebSearchResultLocation `json:"type,required"`
	paramObj
}

func (r CitationWebSearchResultLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationWebSearchResultLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationWebSearchResultLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationsConfig struct {
	Enabled bool `json:"enabled,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Enabled     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationsConfig) RawJSON() string { return r.JSON.raw }
func (r *CitationsConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationsConfigParam struct {
	Enabled param.Opt[bool] `json:"enabled,omitzero"`
	paramObj
}

func (r CitationsConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow CitationsConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CitationsConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationsDelta struct {
	Citation CitationsDeltaCitationUnion `json:"citation,required"`
	Type     constant.CitationsDelta     `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Citation    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationsDelta) RawJSON() string { return r.JSON.raw }
func (r *CitationsDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// CitationsDeltaCitationUnion contains all possible properties and values from
// [CitationCharLocation], [CitationPageLocation], [CitationContentBlockLocation],
// [CitationsWebSearchResultLocation], [CitationsSearchResultLocation].
//
// Use the [CitationsDeltaCitationUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type CitationsDeltaCitationUnion struct {
	CitedText     string `json:"cited_text"`
	DocumentIndex int64  `json:"document_index"`
	DocumentTitle string `json:"document_title"`
	// This field is from variant [CitationCharLocation].
	EndCharIndex int64  `json:"end_char_index"`
	FileID       string `json:"file_id"`
	// This field is from variant [CitationCharLocation].
	StartCharIndex int64 `json:"start_char_index"`
	// Any of "char_location", "page_location", "content_block_location",
	// "web_search_result_location", "search_result_location".
	Type string `json:"type"`
	// This field is from variant [CitationPageLocation].
	EndPageNumber int64 `json:"end_page_number"`
	// This field is from variant [CitationPageLocation].
	StartPageNumber int64 `json:"start_page_number"`
	EndBlockIndex   int64 `json:"end_block_index"`
	StartBlockIndex int64 `json:"start_block_index"`
	// This field is from variant [CitationsWebSearchResultLocation].
	EncryptedIndex string `json:"encrypted_index"`
	Title          string `json:"title"`
	// This field is from variant [CitationsWebSearchResultLocation].
	URL string `json:"url"`
	// This field is from variant [CitationsSearchResultLocation].
	SearchResultIndex int64 `json:"search_result_index"`
	// This field is from variant [CitationsSearchResultLocation].
	Source string `json:"source"`
	JSON   struct {
		CitedText         respjson.Field
		DocumentIndex     respjson.Field
		DocumentTitle     respjson.Field
		EndCharIndex      respjson.Field
		FileID            respjson.Field
		StartCharIndex    respjson.Field
		Type              respjson.Field
		EndPageNumber     respjson.Field
		StartPageNumber   respjson.Field
		EndBlockIndex     respjson.Field
		StartBlockIndex   respjson.Field
		EncryptedIndex    respjson.Field
		Title             respjson.Field
		URL               respjson.Field
		SearchResultIndex respjson.Field
		Source            respjson.Field
		raw               string
	} `json:"-"`
}

// anyCitationsDeltaCitation is implemented by each variant of
// [CitationsDeltaCitationUnion] to add type safety for the return type of
// [CitationsDeltaCitationUnion.AsAny]
type anyCitationsDeltaCitation interface {
	implCitationsDeltaCitationUnion()
}

func (CitationCharLocation) implCitationsDeltaCitationUnion()             {}
func (CitationPageLocation) implCitationsDeltaCitationUnion()             {}
func (CitationContentBlockLocation) implCitationsDeltaCitationUnion()     {}
func (CitationsWebSearchResultLocation) implCitationsDeltaCitationUnion() {}
func (CitationsSearchResultLocation) implCitationsDeltaCitationUnion()    {}

// Use the following switch statement to find the correct variant
//
//	switch variant := CitationsDeltaCitationUnion.AsAny().(type) {
//	case anthropic.CitationCharLocation:
//	case anthropic.CitationPageLocation:
//	case anthropic.CitationContentBlockLocation:
//	case anthropic.CitationsWebSearchResultLocation:
//	case anthropic.CitationsSearchResultLocation:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u CitationsDeltaCitationUnion) AsAny() anyCitationsDeltaCitation {
	switch u.Type {
	case "char_location":
		return u.AsCharLocation()
	case "page_location":
		return u.AsPageLocation()
	case "content_block_location":
		return u.AsContentBlockLocation()
	case "web_search_result_location":
		return u.AsWebSearchResultLocation()
	case "search_result_location":
		return u.AsSearchResultLocation()
	}
	return nil
}

func (u CitationsDeltaCitationUnion) AsCharLocation() (v CitationCharLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CitationsDeltaCitationUnion) AsPageLocation() (v CitationPageLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CitationsDeltaCitationUnion) AsContentBlockLocation() (v CitationContentBlockLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CitationsDeltaCitationUnion) AsWebSearchResultLocation() (v CitationsWebSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CitationsDeltaCitationUnion) AsSearchResultLocation() (v CitationsSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u CitationsDeltaCitationUnion) RawJSON() string { return u.JSON.raw }

func (r *CitationsDeltaCitationUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationsSearchResultLocation struct {
	CitedText         string                        `json:"cited_text,required"`
	EndBlockIndex     int64                         `json:"end_block_index,required"`
	SearchResultIndex int64                         `json:"search_result_index,required"`
	Source            string                        `json:"source,required"`
	StartBlockIndex   int64                         `json:"start_block_index,required"`
	Title             string                        `json:"title,required"`
	Type              constant.SearchResultLocation `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CitedText         respjson.Field
		EndBlockIndex     respjson.Field
		SearchResultIndex respjson.Field
		Source            respjson.Field
		StartBlockIndex   respjson.Field
		Title             respjson.Field
		Type              respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationsSearchResultLocation) RawJSON() string { return r.JSON.raw }
func (r *CitationsSearchResultLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CitationsWebSearchResultLocation struct {
	CitedText      string                           `json:"cited_text,required"`
	EncryptedIndex string                           `json:"encrypted_index,required"`
	Title          string                           `json:"title,required"`
	Type           constant.WebSearchResultLocation `json:"type,required"`
	URL            string                           `json:"url,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CitedText      respjson.Field
		EncryptedIndex respjson.Field
		Title          respjson.Field
		Type           respjson.Field
		URL            respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CitationsWebSearchResultLocation) RawJSON() string { return r.JSON.raw }
func (r *CitationsWebSearchResultLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CodeExecutionOutputBlock struct {
	FileID string                       `json:"file_id,required"`
	Type   constant.CodeExecutionOutput `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		FileID      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CodeExecutionOutputBlock) RawJSON() string { return r.JSON.raw }
func (r *CodeExecutionOutputBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type CodeExecutionOutputBlockParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_output".
	Type constant.CodeExecutionOutput `json:"type,required"`
	paramObj
}

func (r CodeExecutionOutputBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionOutputBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionOutputBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CodeExecutionResultBlock struct {
	Content    []CodeExecutionOutputBlock   `json:"content,required"`
	ReturnCode int64                        `json:"return_code,required"`
	Stderr     string                       `json:"stderr,required"`
	Stdout     string                       `json:"stdout,required"`
	Type       constant.CodeExecutionResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ReturnCode  respjson.Field
		Stderr      respjson.Field
		Stdout      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *CodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ReturnCode, Stderr, Stdout, Type are required.
type CodeExecutionResultBlockParam struct {
	Content    []CodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	ReturnCode int64                           `json:"return_code,required"`
	Stderr     string                          `json:"stderr,required"`
	Stdout     string                          `json:"stdout,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_result".
	Type constant.CodeExecutionResult `json:"type,required"`
	paramObj
}

func (r CodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type CodeExecutionTool20250522Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "code_execution".
	Name constant.CodeExecution `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20250522".
	Type constant.CodeExecution20250522 `json:"type,required"`
	paramObj
}

func (r CodeExecutionTool20250522Param) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionTool20250522Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionTool20250522Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type CodeExecutionTool20250825Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "code_execution".
	Name constant.CodeExecution `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20250825".
	Type constant.CodeExecution20250825 `json:"type,required"`
	paramObj
}

func (r CodeExecutionTool20250825Param) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionTool20250825Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionTool20250825Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Code execution tool with REPL state persistence (daemon mode + gVisor
// checkpoint).
//
// The properties Name, Type are required.
type CodeExecutionTool20260120Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "code_execution".
	Name constant.CodeExecution `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20260120".
	Type constant.CodeExecution20260120 `json:"type,required"`
	paramObj
}

func (r CodeExecutionTool20260120Param) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionTool20260120Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionTool20260120Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CodeExecutionToolResultBlock struct {
	// Code execution result with encrypted stdout for PFC + web_search results.
	Content   CodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                   `json:"tool_use_id,required"`
	Type      constant.CodeExecutionToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *CodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// CodeExecutionToolResultBlockContentUnion contains all possible properties and
// values from [CodeExecutionToolResultError], [CodeExecutionResultBlock],
// [EncryptedCodeExecutionResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type CodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [CodeExecutionToolResultError].
	ErrorCode  CodeExecutionToolResultErrorCode `json:"error_code"`
	Type       string                           `json:"type"`
	Content    []CodeExecutionOutputBlock       `json:"content"`
	ReturnCode int64                            `json:"return_code"`
	Stderr     string                           `json:"stderr"`
	// This field is from variant [CodeExecutionResultBlock].
	Stdout string `json:"stdout"`
	// This field is from variant [EncryptedCodeExecutionResultBlock].
	EncryptedStdout string `json:"encrypted_stdout"`
	JSON            struct {
		ErrorCode       respjson.Field
		Type            respjson.Field
		Content         respjson.Field
		ReturnCode      respjson.Field
		Stderr          respjson.Field
		Stdout          respjson.Field
		EncryptedStdout respjson.Field
		raw             string
	} `json:"-"`
}

func (u CodeExecutionToolResultBlockContentUnion) AsResponseCodeExecutionToolResultError() (v CodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CodeExecutionToolResultBlockContentUnion) AsResponseCodeExecutionResultBlock() (v CodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CodeExecutionToolResultBlockContentUnion) AsResponseEncryptedCodeExecutionResultBlock() (v EncryptedCodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u CodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *CodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type CodeExecutionToolResultBlockParam struct {
	// Code execution result with encrypted stdout for PFC + web_search results.
	Content   CodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                        `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_tool_result".
	Type constant.CodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r CodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func CodeExecutionToolResultBlockParamContentOfRequestCodeExecutionToolResultError(errorCode CodeExecutionToolResultErrorCode) CodeExecutionToolResultBlockParamContentUnion {
	var variant CodeExecutionToolResultErrorParam
	variant.ErrorCode = errorCode
	return CodeExecutionToolResultBlockParamContentUnion{OfRequestCodeExecutionToolResultError: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type CodeExecutionToolResultBlockParamContentUnion struct {
	OfRequestCodeExecutionToolResultError      *CodeExecutionToolResultErrorParam      `json:",omitzero,inline"`
	OfRequestCodeExecutionResultBlock          *CodeExecutionResultBlockParam          `json:",omitzero,inline"`
	OfRequestEncryptedCodeExecutionResultBlock *EncryptedCodeExecutionResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u CodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestCodeExecutionToolResultError, u.OfRequestCodeExecutionResultBlock, u.OfRequestEncryptedCodeExecutionResultBlock)
}
func (u *CodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *CodeExecutionToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestCodeExecutionToolResultError) {
		return u.OfRequestCodeExecutionToolResultError
	} else if !param.IsOmitted(u.OfRequestCodeExecutionResultBlock) {
		return u.OfRequestCodeExecutionResultBlock
	} else if !param.IsOmitted(u.OfRequestEncryptedCodeExecutionResultBlock) {
		return u.OfRequestEncryptedCodeExecutionResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetStdout() *string {
	if vt := u.OfRequestCodeExecutionResultBlock; vt != nil {
		return &vt.Stdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetEncryptedStdout() *string {
	if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return &vt.EncryptedStdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetReturnCode() *int64 {
	if vt := u.OfRequestCodeExecutionResultBlock; vt != nil {
		return (*int64)(&vt.ReturnCode)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*int64)(&vt.ReturnCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetStderr() *string {
	if vt := u.OfRequestCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Stderr)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Stderr)
	}
	return nil
}

// Returns a pointer to the underlying variant's Content property, if present.
func (u CodeExecutionToolResultBlockParamContentUnion) GetContent() []CodeExecutionOutputBlockParam {
	if vt := u.OfRequestCodeExecutionResultBlock; vt != nil {
		return vt.Content
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return vt.Content
	}
	return nil
}

type CodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode CodeExecutionToolResultErrorCode      `json:"error_code,required"`
	Type      constant.CodeExecutionToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *CodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CodeExecutionToolResultErrorCode string

const (
	CodeExecutionToolResultErrorCodeInvalidToolInput      CodeExecutionToolResultErrorCode = "invalid_tool_input"
	CodeExecutionToolResultErrorCodeUnavailable           CodeExecutionToolResultErrorCode = "unavailable"
	CodeExecutionToolResultErrorCodeTooManyRequests       CodeExecutionToolResultErrorCode = "too_many_requests"
	CodeExecutionToolResultErrorCodeExecutionTimeExceeded CodeExecutionToolResultErrorCode = "execution_time_exceeded"
)

// The properties ErrorCode, Type are required.
type CodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode CodeExecutionToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_tool_result_error".
	Type constant.CodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r CodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow CodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Information about the container used in the request (for the code execution
// tool)
type Container struct {
	// Identifier for the container used in this request
	ID string `json:"id,required"`
	// The time at which the container will expire.
	ExpiresAt time.Time `json:"expires_at,required" format:"date-time"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		ExpiresAt   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Container) RawJSON() string { return r.JSON.raw }
func (r *Container) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Response model for a file uploaded to the container.
type ContainerUploadBlock struct {
	FileID string                   `json:"file_id,required"`
	Type   constant.ContainerUpload `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		FileID      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ContainerUploadBlock) RawJSON() string { return r.JSON.raw }
func (r *ContainerUploadBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A content block that represents a file to be uploaded to the container Files
// uploaded via this block will be available in the container's input directory.
//
// The properties FileID, Type are required.
type ContainerUploadBlockParam struct {
	FileID string `json:"file_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "container_upload".
	Type constant.ContainerUpload `json:"type,required"`
	paramObj
}

func (r ContainerUploadBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ContainerUploadBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ContainerUploadBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockUnion contains all possible properties and values from [TextBlock],
// [ThinkingBlock], [RedactedThinkingBlock], [ToolUseBlock], [ServerToolUseBlock],
// [WebSearchToolResultBlock], [WebFetchToolResultBlock],
// [CodeExecutionToolResultBlock], [BashCodeExecutionToolResultBlock],
// [TextEditorCodeExecutionToolResultBlock], [ToolSearchToolResultBlock],
// [ContainerUploadBlock].
//
// Use the [ContentBlockUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ContentBlockUnion struct {
	// This field is from variant [TextBlock].
	Citations []TextCitationUnion `json:"citations"`
	// This field is from variant [TextBlock].
	Text string `json:"text"`
	// Any of "text", "thinking", "redacted_thinking", "tool_use", "server_tool_use",
	// "web_search_tool_result", "web_fetch_tool_result", "code_execution_tool_result",
	// "bash_code_execution_tool_result", "text_editor_code_execution_tool_result",
	// "tool_search_tool_result", "container_upload".
	Type string `json:"type"`
	// This field is from variant [ThinkingBlock].
	Signature string `json:"signature"`
	// This field is from variant [ThinkingBlock].
	Thinking string `json:"thinking"`
	// This field is from variant [RedactedThinkingBlock].
	Data string `json:"data"`
	ID   string `json:"id"`
	// necessary custom code modification
	Input json.RawMessage `json:"input"`
	Name  string          `json:"name"`
	// This field is from variant [WebSearchToolResultBlock].
	Content WebSearchToolResultBlockContentUnion `json:"content"`
	// This field is from variant [WebSearchToolResultBlock].
	ToolUseID string `json:"tool_use_id"`
	JSON      struct {
		Citations respjson.Field
		Text      respjson.Field
		Type      respjson.Field
		Signature respjson.Field
		Thinking  respjson.Field
		Data      respjson.Field
		ID        respjson.Field
		Caller    respjson.Field
		Input     respjson.Field
		Name      respjson.Field
		Content   respjson.Field
		ToolUseID respjson.Field
		FileID    respjson.Field
		raw       string
	} `json:"-"`
}

// anyContentBlock is implemented by each variant of [ContentBlockUnion] to add
// type safety for the return type of [ContentBlockUnion.AsAny]
type anyContentBlock interface {
	implContentBlockUnion()
	toParamUnion() ContentBlockParamUnion
}

func (TextBlock) implContentBlockUnion()                              {}
func (ThinkingBlock) implContentBlockUnion()                          {}
func (RedactedThinkingBlock) implContentBlockUnion()                  {}
func (ToolUseBlock) implContentBlockUnion()                           {}
func (ServerToolUseBlock) implContentBlockUnion()                     {}
func (WebSearchToolResultBlock) implContentBlockUnion()               {}
func (WebFetchToolResultBlock) implContentBlockUnion()                {}
func (CodeExecutionToolResultBlock) implContentBlockUnion()           {}
func (BashCodeExecutionToolResultBlock) implContentBlockUnion()       {}
func (TextEditorCodeExecutionToolResultBlock) implContentBlockUnion() {}
func (ToolSearchToolResultBlock) implContentBlockUnion()              {}
func (ContainerUploadBlock) implContentBlockUnion()                   {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ContentBlockUnion.AsAny().(type) {
//	case anthropic.TextBlock:
//	case anthropic.ThinkingBlock:
//	case anthropic.RedactedThinkingBlock:
//	case anthropic.ToolUseBlock:
//	case anthropic.ServerToolUseBlock:
//	case anthropic.WebSearchToolResultBlock:
//	case anthropic.WebFetchToolResultBlock:
//	case anthropic.CodeExecutionToolResultBlock:
//	case anthropic.BashCodeExecutionToolResultBlock:
//	case anthropic.TextEditorCodeExecutionToolResultBlock:
//	case anthropic.ToolSearchToolResultBlock:
//	case anthropic.ContainerUploadBlock:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ContentBlockUnion) AsAny() anyContentBlock {
	switch u.Type {
	case "text":
		return u.AsText()
	case "thinking":
		return u.AsThinking()
	case "redacted_thinking":
		return u.AsRedactedThinking()
	case "tool_use":
		return u.AsToolUse()
	case "server_tool_use":
		return u.AsServerToolUse()
	case "web_search_tool_result":
		return u.AsWebSearchToolResult()
	case "web_fetch_tool_result":
		return u.AsWebFetchToolResult()
	case "code_execution_tool_result":
		return u.AsCodeExecutionToolResult()
	case "bash_code_execution_tool_result":
		return u.AsBashCodeExecutionToolResult()
	case "text_editor_code_execution_tool_result":
		return u.AsTextEditorCodeExecutionToolResult()
	case "tool_search_tool_result":
		return u.AsToolSearchToolResult()
	case "container_upload":
		return u.AsContainerUpload()
	}
	return nil
}

func (u ContentBlockUnion) AsText() (v TextBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsThinking() (v ThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsRedactedThinking() (v RedactedThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsToolUse() (v ToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsServerToolUse() (v ServerToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsWebSearchToolResult() (v WebSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsWebFetchToolResult() (v WebFetchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsCodeExecutionToolResult() (v CodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsBashCodeExecutionToolResult() (v BashCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsTextEditorCodeExecutionToolResult() (v TextEditorCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsToolSearchToolResult() (v ToolSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockUnion) AsContainerUpload() (v ContainerUploadBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ContentBlockUnion) RawJSON() string { return u.JSON.raw }

func (r *ContentBlockUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockUnionCaller is an implicit subunion of [ContentBlockUnion].
// ContentBlockUnionCaller provides convenient access to the sub-properties of the
// union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockUnion].
type ContentBlockUnionCaller struct {
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

func (r *ContentBlockUnionCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockUnionContent is an implicit subunion of [ContentBlockUnion].
// ContentBlockUnionContent provides convenient access to the sub-properties of the
// union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfWebSearchResultBlockArray]
type ContentBlockUnionContent struct {
	// This field will be present if the value is a [[]WebSearchResultBlock] instead of
	// an object.
	OfWebSearchResultBlockArray []WebSearchResultBlock `json:",inline"`
	ErrorCode                   string                 `json:"error_code"`
	Type                        string                 `json:"type"`
	// This field is a union of [DocumentBlock], [[]CodeExecutionOutputBlock],
	// [[]CodeExecutionOutputBlock], [[]BashCodeExecutionOutputBlock], [string]
	Content ContentBlockUnionContentContent `json:"content"`
	// This field is from variant [WebFetchToolResultBlockContentUnion].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [WebFetchToolResultBlockContentUnion].
	URL        string `json:"url"`
	ReturnCode int64  `json:"return_code"`
	Stderr     string `json:"stderr"`
	Stdout     string `json:"stdout"`
	// This field is from variant [CodeExecutionToolResultBlockContentUnion].
	EncryptedStdout string `json:"encrypted_stdout"`
	ErrorMessage    string `json:"error_message"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	FileType TextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NumLines int64 `json:"num_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	StartLine int64 `json:"start_line"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	Lines []string `json:"lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NewLines int64 `json:"new_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NewStart int64 `json:"new_start"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	OldLines int64 `json:"old_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	OldStart int64 `json:"old_start"`
	// This field is from variant [ToolSearchToolResultBlockContentUnion].
	ToolReferences []ToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		OfWebSearchResultBlockArray respjson.Field
		ErrorCode                   respjson.Field
		Type                        respjson.Field
		Content                     respjson.Field
		RetrievedAt                 respjson.Field
		URL                         respjson.Field
		ReturnCode                  respjson.Field
		Stderr                      respjson.Field
		Stdout                      respjson.Field
		EncryptedStdout             respjson.Field
		ErrorMessage                respjson.Field
		FileType                    respjson.Field
		NumLines                    respjson.Field
		StartLine                   respjson.Field
		TotalLines                  respjson.Field
		IsFileUpdate                respjson.Field
		Lines                       respjson.Field
		NewLines                    respjson.Field
		NewStart                    respjson.Field
		OldLines                    respjson.Field
		OldStart                    respjson.Field
		ToolReferences              respjson.Field
		raw                         string
	} `json:"-"`
}

func (r *ContentBlockUnionContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockUnionContentContent is an implicit subunion of [ContentBlockUnion].
// ContentBlockUnionContentContent provides convenient access to the sub-properties
// of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfContent OfString]
type ContentBlockUnionContentContent struct {
	// This field will be present if the value is a [[]CodeExecutionOutputBlock]
	// instead of an object.
	OfContent []CodeExecutionOutputBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is from variant [DocumentBlock].
	Citations CitationsConfig `json:"citations"`
	// This field is from variant [DocumentBlock].
	Source DocumentBlockSourceUnion `json:"source"`
	// This field is from variant [DocumentBlock].
	Title string `json:"title"`
	// This field is from variant [DocumentBlock].
	Type constant.Document `json:"type"`
	JSON struct {
		OfContent respjson.Field
		OfString  respjson.Field
		Citations respjson.Field
		Source    respjson.Field
		Title     respjson.Field
		Type      respjson.Field
		raw       string
	} `json:"-"`
}

func (r *ContentBlockUnionContentContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewTextBlock(text string) ContentBlockParamUnion {
	var variant TextBlockParam
	variant.Text = text
	return ContentBlockParamUnion{OfText: &variant}
}

func NewImageBlock[T Base64ImageSourceParam | URLImageSourceParam](source T) ContentBlockParamUnion {
	var image ImageBlockParam
	switch v := any(source).(type) {
	case Base64ImageSourceParam:
		image.Source.OfBase64 = &v
	case URLImageSourceParam:
		image.Source.OfURL = &v
	}
	return ContentBlockParamUnion{OfImage: &image}
}

func NewImageBlockBase64(mediaType string, encodedData string) ContentBlockParamUnion {
	return ContentBlockParamUnion{
		OfImage: &ImageBlockParam{
			Source: ImageBlockParamSourceUnion{
				OfBase64: &Base64ImageSourceParam{
					Data:      encodedData,
					MediaType: Base64ImageSourceMediaType(mediaType),
				},
			},
		},
	}
}

func NewDocumentBlock[
	T Base64PDFSourceParam | PlainTextSourceParam | ContentBlockSourceParam | URLPDFSourceParam,
](source T) ContentBlockParamUnion {
	var document DocumentBlockParam
	switch v := any(source).(type) {
	case Base64PDFSourceParam:
		document.Source.OfBase64 = &v
	case PlainTextSourceParam:
		document.Source.OfText = &v
	case ContentBlockSourceParam:
		document.Source.OfContent = &v
	case URLPDFSourceParam:
		document.Source.OfURL = &v
	}
	return ContentBlockParamUnion{OfDocument: &document}
}

func NewSearchResultBlock(content []TextBlockParam, source string, title string) ContentBlockParamUnion {
	var searchResult SearchResultBlockParam
	searchResult.Content = content
	searchResult.Source = source
	searchResult.Title = title
	return ContentBlockParamUnion{OfSearchResult: &searchResult}
}

func NewThinkingBlock(signature string, thinking string) ContentBlockParamUnion {
	var variant ThinkingBlockParam
	variant.Signature = signature
	variant.Thinking = thinking
	return ContentBlockParamUnion{OfThinking: &variant}
}

func NewRedactedThinkingBlock(data string) ContentBlockParamUnion {
	var redactedThinking RedactedThinkingBlockParam
	redactedThinking.Data = data
	return ContentBlockParamUnion{OfRedactedThinking: &redactedThinking}
}

func NewToolUseBlock(id string, input any, name string) ContentBlockParamUnion {
	var toolUse ToolUseBlockParam
	toolUse.ID = id
	toolUse.Input = input
	toolUse.Name = name
	return ContentBlockParamUnion{OfToolUse: &toolUse}
}

func NewToolResultBlock(toolUseID string, content string, isError bool) ContentBlockParamUnion {
	toolBlock := ToolResultBlockParam{
		ToolUseID: toolUseID,
		Content: []ToolResultBlockParamContentUnion{
			{OfText: &TextBlockParam{Text: content}},
		},
		IsError: Bool(isError),
	}
	return ContentBlockParamUnion{OfToolResult: &toolBlock}
}

func NewServerToolUseBlock(id string, input any, name ServerToolUseBlockParamName) ContentBlockParamUnion {
	var serverToolUse ServerToolUseBlockParam
	serverToolUse.ID = id
	serverToolUse.Input = input
	serverToolUse.Name = name
	return ContentBlockParamUnion{OfServerToolUse: &serverToolUse}
}

func NewWebSearchToolResultBlock[
	T []WebSearchResultBlockParam | WebSearchToolRequestErrorParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var webSearchToolResult WebSearchToolResultBlockParam
	switch v := any(content).(type) {
	case []WebSearchResultBlockParam:
		webSearchToolResult.Content.OfWebSearchToolResultBlockItem = v
	case WebSearchToolRequestErrorParam:
		webSearchToolResult.Content.OfRequestWebSearchToolResultError = &v
	}
	webSearchToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfWebSearchToolResult: &webSearchToolResult}
}

func NewWebFetchToolResultBlock[
	T WebFetchToolResultErrorBlockParam | WebFetchBlockParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var webFetchToolResult WebFetchToolResultBlockParam
	switch v := any(content).(type) {
	case WebFetchToolResultErrorBlockParam:
		webFetchToolResult.Content.OfRequestWebFetchToolResultError = &v
	case WebFetchBlockParam:
		webFetchToolResult.Content.OfRequestWebFetchResultBlock = &v
	}
	webFetchToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfWebFetchToolResult: &webFetchToolResult}
}

func NewCodeExecutionToolResultBlock[
	T CodeExecutionToolResultErrorParam | CodeExecutionResultBlockParam | EncryptedCodeExecutionResultBlockParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var codeExecutionToolResult CodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case CodeExecutionToolResultErrorParam:
		codeExecutionToolResult.Content.OfRequestCodeExecutionToolResultError = &v
	case CodeExecutionResultBlockParam:
		codeExecutionToolResult.Content.OfRequestCodeExecutionResultBlock = &v
	case EncryptedCodeExecutionResultBlockParam:
		codeExecutionToolResult.Content.OfRequestEncryptedCodeExecutionResultBlock = &v
	}
	codeExecutionToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfCodeExecutionToolResult: &codeExecutionToolResult}
}

func NewBashCodeExecutionToolResultBlock[
	T BashCodeExecutionToolResultErrorParam | BashCodeExecutionResultBlockParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var bashCodeExecutionToolResult BashCodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case BashCodeExecutionToolResultErrorParam:
		bashCodeExecutionToolResult.Content.OfRequestBashCodeExecutionToolResultError = &v
	case BashCodeExecutionResultBlockParam:
		bashCodeExecutionToolResult.Content.OfRequestBashCodeExecutionResultBlock = &v
	}
	bashCodeExecutionToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfBashCodeExecutionToolResult: &bashCodeExecutionToolResult}
}

func NewTextEditorCodeExecutionToolResultBlock[
	T TextEditorCodeExecutionToolResultErrorParam | TextEditorCodeExecutionViewResultBlockParam | TextEditorCodeExecutionCreateResultBlockParam | TextEditorCodeExecutionStrReplaceResultBlockParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var textEditorCodeExecutionToolResult TextEditorCodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case TextEditorCodeExecutionToolResultErrorParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionToolResultError = &v
	case TextEditorCodeExecutionViewResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionViewResultBlock = &v
	case TextEditorCodeExecutionCreateResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionCreateResultBlock = &v
	case TextEditorCodeExecutionStrReplaceResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionStrReplaceResultBlock = &v
	}
	textEditorCodeExecutionToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfTextEditorCodeExecutionToolResult: &textEditorCodeExecutionToolResult}
}

func NewToolSearchToolResultBlock[
	T ToolSearchToolResultErrorParam | ToolSearchToolSearchResultBlockParam,
](content T, toolUseID string) ContentBlockParamUnion {
	var toolSearchToolResult ToolSearchToolResultBlockParam
	switch v := any(content).(type) {
	case ToolSearchToolResultErrorParam:
		toolSearchToolResult.Content.OfRequestToolSearchToolResultError = &v
	case ToolSearchToolSearchResultBlockParam:
		toolSearchToolResult.Content.OfRequestToolSearchToolSearchResultBlock = &v
	}
	toolSearchToolResult.ToolUseID = toolUseID
	return ContentBlockParamUnion{OfToolSearchToolResult: &toolSearchToolResult}
}

func NewContainerUploadBlock(fileID string) ContentBlockParamUnion {
	var containerUpload ContainerUploadBlockParam
	containerUpload.FileID = fileID
	return ContentBlockParamUnion{OfContainerUpload: &containerUpload}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ContentBlockParamUnion struct {
	OfText                              *TextBlockParam                              `json:",omitzero,inline"`
	OfImage                             *ImageBlockParam                             `json:",omitzero,inline"`
	OfDocument                          *DocumentBlockParam                          `json:",omitzero,inline"`
	OfSearchResult                      *SearchResultBlockParam                      `json:",omitzero,inline"`
	OfThinking                          *ThinkingBlockParam                          `json:",omitzero,inline"`
	OfRedactedThinking                  *RedactedThinkingBlockParam                  `json:",omitzero,inline"`
	OfToolUse                           *ToolUseBlockParam                           `json:",omitzero,inline"`
	OfToolResult                        *ToolResultBlockParam                        `json:",omitzero,inline"`
	OfServerToolUse                     *ServerToolUseBlockParam                     `json:",omitzero,inline"`
	OfWebSearchToolResult               *WebSearchToolResultBlockParam               `json:",omitzero,inline"`
	OfWebFetchToolResult                *WebFetchToolResultBlockParam                `json:",omitzero,inline"`
	OfCodeExecutionToolResult           *CodeExecutionToolResultBlockParam           `json:",omitzero,inline"`
	OfBashCodeExecutionToolResult       *BashCodeExecutionToolResultBlockParam       `json:",omitzero,inline"`
	OfTextEditorCodeExecutionToolResult *TextEditorCodeExecutionToolResultBlockParam `json:",omitzero,inline"`
	OfToolSearchToolResult              *ToolSearchToolResultBlockParam              `json:",omitzero,inline"`
	OfContainerUpload                   *ContainerUploadBlockParam                   `json:",omitzero,inline"`
	paramUnion
}

func (u ContentBlockParamUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfText,
		u.OfImage,
		u.OfDocument,
		u.OfSearchResult,
		u.OfThinking,
		u.OfRedactedThinking,
		u.OfToolUse,
		u.OfToolResult,
		u.OfServerToolUse,
		u.OfWebSearchToolResult,
		u.OfWebFetchToolResult,
		u.OfCodeExecutionToolResult,
		u.OfBashCodeExecutionToolResult,
		u.OfTextEditorCodeExecutionToolResult,
		u.OfToolSearchToolResult,
		u.OfContainerUpload)
}
func (u *ContentBlockParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ContentBlockParamUnion) asAny() any {
	if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfImage) {
		return u.OfImage
	} else if !param.IsOmitted(u.OfDocument) {
		return u.OfDocument
	} else if !param.IsOmitted(u.OfSearchResult) {
		return u.OfSearchResult
	} else if !param.IsOmitted(u.OfThinking) {
		return u.OfThinking
	} else if !param.IsOmitted(u.OfRedactedThinking) {
		return u.OfRedactedThinking
	} else if !param.IsOmitted(u.OfToolUse) {
		return u.OfToolUse
	} else if !param.IsOmitted(u.OfToolResult) {
		return u.OfToolResult
	} else if !param.IsOmitted(u.OfServerToolUse) {
		return u.OfServerToolUse
	} else if !param.IsOmitted(u.OfWebSearchToolResult) {
		return u.OfWebSearchToolResult
	} else if !param.IsOmitted(u.OfWebFetchToolResult) {
		return u.OfWebFetchToolResult
	} else if !param.IsOmitted(u.OfCodeExecutionToolResult) {
		return u.OfCodeExecutionToolResult
	} else if !param.IsOmitted(u.OfBashCodeExecutionToolResult) {
		return u.OfBashCodeExecutionToolResult
	} else if !param.IsOmitted(u.OfTextEditorCodeExecutionToolResult) {
		return u.OfTextEditorCodeExecutionToolResult
	} else if !param.IsOmitted(u.OfToolSearchToolResult) {
		return u.OfToolSearchToolResult
	} else if !param.IsOmitted(u.OfContainerUpload) {
		return u.OfContainerUpload
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetText() *string {
	if vt := u.OfText; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetContext() *string {
	if vt := u.OfDocument; vt != nil && vt.Context.Valid() {
		return &vt.Context.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetSignature() *string {
	if vt := u.OfThinking; vt != nil {
		return &vt.Signature
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetThinking() *string {
	if vt := u.OfThinking; vt != nil {
		return &vt.Thinking
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetData() *string {
	if vt := u.OfRedactedThinking; vt != nil {
		return &vt.Data
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetIsError() *bool {
	if vt := u.OfToolResult; vt != nil && vt.IsError.Valid() {
		return &vt.IsError.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetFileID() *string {
	if vt := u.OfContainerUpload; vt != nil {
		return &vt.FileID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetType() *string {
	if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfImage; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfDocument; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfThinking; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRedactedThinking; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolUse; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfServerToolUse; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebFetchToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditorCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolSearchToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContainerUpload; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetTitle() *string {
	if vt := u.OfDocument; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	} else if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Title)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetID() *string {
	if vt := u.OfToolUse; vt != nil {
		return (*string)(&vt.ID)
	} else if vt := u.OfServerToolUse; vt != nil {
		return (*string)(&vt.ID)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetName() *string {
	if vt := u.OfToolUse; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfServerToolUse; vt != nil {
		return (*string)(&vt.Name)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockParamUnion) GetToolUseID() *string {
	if vt := u.OfToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfWebSearchToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfWebFetchToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfBashCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfTextEditorCodeExecutionToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	} else if vt := u.OfToolSearchToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u ContentBlockParamUnion) GetCacheControl() *CacheControlEphemeralParam {
	if vt := u.OfText; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfImage; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfDocument; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfSearchResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolUse; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfServerToolUse; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebSearchToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebFetchToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashCodeExecutionToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditorCodeExecutionToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolSearchToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfContainerUpload; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ContentBlockParamUnion) GetCitations() (res contentBlockParamUnionCitations) {
	if vt := u.OfText; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfDocument; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Citations
	}
	return
}

// Can have the runtime types [*[]TextCitationParamUnion], [*CitationsConfigParam]
type contentBlockParamUnionCitations struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.TextCitationParamUnion:
//	case *anthropic.CitationsConfigParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u contentBlockParamUnionCitations) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionCitations) GetEnabled() *bool {
	switch vt := u.any.(type) {
	case *CitationsConfigParam:
		return paramutil.AddrIfPresent(vt.Enabled)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ContentBlockParamUnion) GetSource() (res contentBlockParamUnionSource) {
	if vt := u.OfImage; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfDocument; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Source
	}
	return
}

// Can have the runtime types [*Base64ImageSourceParam], [*URLImageSourceParam],
// [*Base64PDFSourceParam], [*PlainTextSourceParam], [*ContentBlockSourceParam],
// [*URLPDFSourceParam], [*string]
type contentBlockParamUnionSource struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.Base64ImageSourceParam:
//	case *anthropic.URLImageSourceParam:
//	case *anthropic.Base64PDFSourceParam:
//	case *anthropic.PlainTextSourceParam:
//	case *anthropic.ContentBlockSourceParam:
//	case *anthropic.URLPDFSourceParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u contentBlockParamUnionSource) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionSource) GetContent() *ContentBlockSourceContentUnionParam {
	switch vt := u.any.(type) {
	case *DocumentBlockParamSourceUnion:
		return vt.GetContent()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionSource) GetData() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetData()
	case *DocumentBlockParamSourceUnion:
		return vt.GetData()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionSource) GetMediaType() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetMediaType()
	case *DocumentBlockParamSourceUnion:
		return vt.GetMediaType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionSource) GetType() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetType()
	case *DocumentBlockParamSourceUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionSource) GetURL() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetURL()
	case *DocumentBlockParamSourceUnion:
		return vt.GetURL()
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ContentBlockParamUnion) GetContent() (res contentBlockParamUnionContent) {
	if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Content
	} else if vt := u.OfToolResult; vt != nil {
		res.any = &vt.Content
	} else if vt := u.OfWebSearchToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfWebFetchToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfCodeExecutionToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfBashCodeExecutionToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfTextEditorCodeExecutionToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfToolSearchToolResult; vt != nil {
		res.any = vt.Content.asAny()
	}
	return
}

// Can have the runtime types [_[]TextBlockParam],
// [_[]ToolResultBlockParamContentUnion], [*[]WebSearchResultBlockParam],
// [*WebFetchToolResultErrorBlockParam], [*WebFetchBlockParam],
// [*CodeExecutionToolResultErrorParam], [*CodeExecutionResultBlockParam],
// [*EncryptedCodeExecutionResultBlockParam],
// [*BashCodeExecutionToolResultErrorParam], [*BashCodeExecutionResultBlockParam],
// [*TextEditorCodeExecutionToolResultErrorParam],
// [*TextEditorCodeExecutionViewResultBlockParam],
// [*TextEditorCodeExecutionCreateResultBlockParam],
// [*TextEditorCodeExecutionStrReplaceResultBlockParam],
// [*ToolSearchToolResultErrorParam], [*ToolSearchToolSearchResultBlockParam]
type contentBlockParamUnionContent struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.TextBlockParam:
//	case *[]anthropic.ToolResultBlockParamContentUnion:
//	case *[]anthropic.WebSearchResultBlockParam:
//	case *anthropic.WebFetchToolResultErrorBlockParam:
//	case *anthropic.WebFetchBlockParam:
//	case *anthropic.CodeExecutionToolResultErrorParam:
//	case *anthropic.CodeExecutionResultBlockParam:
//	case *anthropic.EncryptedCodeExecutionResultBlockParam:
//	case *anthropic.BashCodeExecutionToolResultErrorParam:
//	case *anthropic.BashCodeExecutionResultBlockParam:
//	case *anthropic.TextEditorCodeExecutionToolResultErrorParam:
//	case *anthropic.TextEditorCodeExecutionViewResultBlockParam:
//	case *anthropic.TextEditorCodeExecutionCreateResultBlockParam:
//	case *anthropic.TextEditorCodeExecutionStrReplaceResultBlockParam:
//	case *anthropic.ToolSearchToolResultErrorParam:
//	case *anthropic.ToolSearchToolSearchResultBlockParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u contentBlockParamUnionContent) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetURL() *string {
	switch vt := u.any.(type) {
	case *WebFetchToolResultBlockParamContentUnion:
		return vt.GetURL()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetRetrievedAt() *string {
	switch vt := u.any.(type) {
	case *WebFetchToolResultBlockParamContentUnion:
		return vt.GetRetrievedAt()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetEncryptedStdout() *string {
	switch vt := u.any.(type) {
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetEncryptedStdout()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetErrorMessage() *string {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorMessage()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetFileType() *string {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetFileType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetNumLines() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNumLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetStartLine() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStartLine()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetTotalLines() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetTotalLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetIsFileUpdate() *bool {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetIsFileUpdate()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetLines() []string {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetNewLines() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNewLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetNewStart() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNewStart()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetOldLines() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetOldLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetOldStart() *int64 {
	switch vt := u.any.(type) {
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetOldStart()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetToolReferences() []ToolReferenceBlockParam {
	switch vt := u.any.(type) {
	case *ToolSearchToolResultBlockParamContentUnion:
		return vt.GetToolReferences()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetErrorCode() *string {
	switch vt := u.any.(type) {
	case *WebSearchToolResultBlockParamContentUnion:
		if vt.OfRequestWebSearchToolResultError != nil {
			return (*string)(&vt.OfRequestWebSearchToolResultError.ErrorCode)
		}
	case *WebFetchToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *ToolSearchToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetType() *string {
	switch vt := u.any.(type) {
	case *WebSearchToolResultBlockParamContentUnion:
		if vt.OfRequestWebSearchToolResultError != nil {
			return (*string)(&vt.OfRequestWebSearchToolResultError.Type)
		}
	case *WebFetchToolResultBlockParamContentUnion:
		return vt.GetType()
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *ToolSearchToolResultBlockParamContentUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetReturnCode() *int64 {
	switch vt := u.any.(type) {
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetReturnCode()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetReturnCode()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetStderr() *string {
	switch vt := u.any.(type) {
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStderr()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStderr()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionContent) GetStdout() *string {
	switch vt := u.any.(type) {
	case *CodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStdout()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStdout()
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u contentBlockParamUnionContent) GetContent() (res contentBlockParamUnionContentContent) {
	switch vt := u.any.(type) {
	case *WebFetchToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *CodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *BashCodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *TextEditorCodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	}
	return res
}

// Can have the runtime types [*DocumentBlockParam],
// [_[]CodeExecutionOutputBlockParam], [_[]BashCodeExecutionOutputBlockParam],
// [*string]
type contentBlockParamUnionContentContent struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.DocumentBlockParam:
//	case *[]anthropic.CodeExecutionOutputBlockParam:
//	case *[]anthropic.BashCodeExecutionOutputBlockParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u contentBlockParamUnionContentContent) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's Input property, if present.
func (u ContentBlockParamUnion) GetInput() *any {
	if vt := u.OfToolUse; vt != nil {
		return &vt.Input
	} else if vt := u.OfServerToolUse; vt != nil {
		return &vt.Input
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ContentBlockParamUnion) GetCaller() (res contentBlockParamUnionCaller) {
	if vt := u.OfToolUse; vt != nil {
		res.any = vt.Caller.asAny()
	} else if vt := u.OfServerToolUse; vt != nil {
		res.any = vt.Caller.asAny()
	} else if vt := u.OfWebSearchToolResult; vt != nil {
		res.any = vt.Caller.asAny()
	} else if vt := u.OfWebFetchToolResult; vt != nil {
		res.any = vt.Caller.asAny()
	}
	return
}

// Can have the runtime types [*DirectCallerParam], [*ServerToolCallerParam],
// [*ServerToolCaller20260120Param]
type contentBlockParamUnionCaller struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.DirectCallerParam:
//	case *anthropic.ServerToolCallerParam:
//	case *anthropic.ServerToolCaller20260120Param:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u contentBlockParamUnionCaller) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionCaller) GetType() *string {
	switch vt := u.any.(type) {
	case *ToolUseBlockParamCallerUnion:
		return vt.GetType()
	case *ServerToolUseBlockParamCallerUnion:
		return vt.GetType()
	case *WebSearchToolResultBlockParamCallerUnion:
		return vt.GetType()
	case *WebFetchToolResultBlockParamCallerUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u contentBlockParamUnionCaller) GetToolID() *string {
	switch vt := u.any.(type) {
	case *ToolUseBlockParamCallerUnion:
		return vt.GetToolID()
	case *ServerToolUseBlockParamCallerUnion:
		return vt.GetToolID()
	case *WebSearchToolResultBlockParamCallerUnion:
		return vt.GetToolID()
	case *WebFetchToolResultBlockParamCallerUnion:
		return vt.GetToolID()
	}
	return nil
}

func init() {
	apijson.RegisterUnion[ContentBlockParamUnion](
		"type",
		apijson.Discriminator[TextBlockParam]("text"),
		apijson.Discriminator[ImageBlockParam]("image"),
		apijson.Discriminator[DocumentBlockParam]("document"),
		apijson.Discriminator[SearchResultBlockParam]("search_result"),
		apijson.Discriminator[ThinkingBlockParam]("thinking"),
		apijson.Discriminator[RedactedThinkingBlockParam]("redacted_thinking"),
		apijson.Discriminator[ToolUseBlockParam]("tool_use"),
		apijson.Discriminator[ToolResultBlockParam]("tool_result"),
		apijson.Discriminator[ServerToolUseBlockParam]("server_tool_use"),
		apijson.Discriminator[WebSearchToolResultBlockParam]("web_search_tool_result"),
		apijson.Discriminator[WebFetchToolResultBlockParam]("web_fetch_tool_result"),
		apijson.Discriminator[CodeExecutionToolResultBlockParam]("code_execution_tool_result"),
		apijson.Discriminator[BashCodeExecutionToolResultBlockParam]("bash_code_execution_tool_result"),
		apijson.Discriminator[TextEditorCodeExecutionToolResultBlockParam]("text_editor_code_execution_tool_result"),
		apijson.Discriminator[ToolSearchToolResultBlockParam]("tool_search_tool_result"),
		apijson.Discriminator[ContainerUploadBlockParam]("container_upload"),
	)

	// Register custom decoder for []ContentBlockParamUnion to handle string content
	apijson.RegisterCustomDecoder[[]ContentBlockParamUnion](func(node gjson.Result, value reflect.Value, defaultDecoder func(gjson.Result, reflect.Value) error) error {
		// If it's a string, convert it to a TextBlock automatically
		if node.Type == gjson.String {
			textBlock := TextBlockParam{
				Text: node.String(),
				Type: "text",
			}
			contentUnion := ContentBlockParamUnion{
				OfText: &textBlock,
			}
			arrayValue := reflect.MakeSlice(value.Type(), 1, 1)
			arrayValue.Index(0).Set(reflect.ValueOf(contentUnion))
			value.Set(arrayValue)
			return nil
		}

		return defaultDecoder(node, value)
	})
}

func init() {
	apijson.RegisterUnion[ContentBlockSourceContentItemUnionParam](
		"type",
		apijson.Discriminator[TextBlockParam]("text"),
		apijson.Discriminator[ImageBlockParam]("image"),
	)
}

func init() {
	apijson.RegisterUnion[DocumentBlockParamSourceUnion](
		"type",
		apijson.Discriminator[Base64PDFSourceParam]("base64"),
		apijson.Discriminator[PlainTextSourceParam]("text"),
		apijson.Discriminator[ContentBlockSourceParam]("content"),
		apijson.Discriminator[URLPDFSourceParam]("url"),
	)
}

func init() {
	apijson.RegisterUnion[ImageBlockParamSourceUnion](
		"type",
		apijson.Discriminator[Base64ImageSourceParam]("base64"),
		apijson.Discriminator[URLImageSourceParam]("url"),
	)
}

func init() {
	apijson.RegisterUnion[TextCitationParamUnion](
		"type",
		apijson.Discriminator[CitationCharLocationParam]("char_location"),
		apijson.Discriminator[CitationPageLocationParam]("page_location"),
		apijson.Discriminator[CitationContentBlockLocationParam]("content_block_location"),
		apijson.Discriminator[CitationWebSearchResultLocationParam]("web_search_result_location"),
		apijson.Discriminator[CitationSearchResultLocationParam]("search_result_location"),
	)
}

func init() {
	apijson.RegisterUnion[ThinkingConfigParamUnion](
		"type",
		apijson.Discriminator[ThinkingConfigEnabledParam]("enabled"),
		apijson.Discriminator[ThinkingConfigDisabledParam]("disabled"),
	)
}

func init() {
	apijson.RegisterUnion[ToolChoiceUnionParam](
		"type",
		apijson.Discriminator[ToolChoiceAutoParam]("auto"),
		apijson.Discriminator[ToolChoiceAnyParam]("any"),
		apijson.Discriminator[ToolChoiceToolParam]("tool"),
		apijson.Discriminator[ToolChoiceNoneParam]("none"),
	)
}

func init() {
	apijson.RegisterUnion[ToolResultBlockParamContentUnion](
		"type",
		apijson.Discriminator[TextBlockParam]("text"),
		apijson.Discriminator[ImageBlockParam]("image"),
		apijson.Discriminator[SearchResultBlockParam]("search_result"),
		apijson.Discriminator[DocumentBlockParam]("document"),
	)

	// Register custom decoder for []ToolResultBlockParamContentUnion to handle string content
	apijson.RegisterCustomDecoder[[]ToolResultBlockParamContentUnion](func(node gjson.Result, value reflect.Value, defaultDecoder func(gjson.Result, reflect.Value) error) error {
		// If it's a string, convert it to a TextBlock automatically
		if node.Type == gjson.String {
			textBlock := TextBlockParam{
				Text: node.String(),
				Type: "text",
			}
			contentUnion := ToolResultBlockParamContentUnion{
				OfText: &textBlock,
			}
			arrayValue := reflect.MakeSlice(value.Type(), 1, 1)
			arrayValue.Index(0).Set(reflect.ValueOf(contentUnion))
			value.Set(arrayValue)
			return nil
		}

		// If it's already an array, use the default decoder
		return defaultDecoder(node, value)
	})
}

// The properties Content, Type are required.
type ContentBlockSourceParam struct {
	Content ContentBlockSourceContentUnionParam `json:"content,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "content".
	Type constant.Content `json:"type,required"`
	paramObj
}

func (r ContentBlockSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow ContentBlockSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ContentBlockSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ContentBlockSourceContentUnionParam struct {
	OfString                    param.Opt[string]                         `json:",omitzero,inline"`
	OfContentBlockSourceContent []ContentBlockSourceContentItemUnionParam `json:",omitzero,inline"`
	paramUnion
}

func (u ContentBlockSourceContentUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfContentBlockSourceContent)
}
func (u *ContentBlockSourceContentUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ContentBlockSourceContentUnionParam) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfContentBlockSourceContent) {
		return &u.OfContentBlockSourceContent
	}
	return nil
}

func ContentBlockSourceContentItemParamOfText(text string) ContentBlockSourceContentItemUnionParam {
	var variant TextBlockParam
	variant.Text = text
	return ContentBlockSourceContentItemUnionParam{OfText: &variant}
}

func ContentBlockSourceContentItemParamOfImage[T Base64ImageSourceParam | URLImageSourceParam](source T) ContentBlockSourceContentItemUnionParam {
	var image ImageBlockParam
	switch v := any(source).(type) {
	case Base64ImageSourceParam:
		image.Source.OfBase64 = &v
	case URLImageSourceParam:
		image.Source.OfURL = &v
	}
	return ContentBlockSourceContentItemUnionParam{OfImage: &image}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ContentBlockSourceContentItemUnionParam struct {
	OfText  *TextBlockParam  `json:",omitzero,inline"`
	OfImage *ImageBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u ContentBlockSourceContentItemUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfText, u.OfImage)
}
func (u *ContentBlockSourceContentItemUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ContentBlockSourceContentItemUnionParam) asAny() any {
	if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfImage) {
		return u.OfImage
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockSourceContentItemUnionParam) GetText() *string {
	if vt := u.OfText; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockSourceContentItemUnionParam) GetCitations() []TextCitationParamUnion {
	if vt := u.OfText; vt != nil {
		return vt.Citations
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockSourceContentItemUnionParam) GetSource() *ImageBlockParamSourceUnion {
	if vt := u.OfImage; vt != nil {
		return &vt.Source
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ContentBlockSourceContentItemUnionParam) GetType() *string {
	if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfImage; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u ContentBlockSourceContentItemUnionParam) GetCacheControl() *CacheControlEphemeralParam {
	if vt := u.OfText; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfImage; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

func init() {
	apijson.RegisterUnion[ContentBlockSourceContentItemUnionParam](
		"type",
		apijson.Discriminator[TextBlockParam]("text"),
		apijson.Discriminator[ImageBlockParam]("image"),
	)
}

// Tool invocation directly from the model.
type DirectCaller struct {
	Type constant.Direct `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r DirectCaller) RawJSON() string { return r.JSON.raw }
func (r *DirectCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this DirectCaller to a DirectCallerParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// DirectCallerParam.Overrides()
func (r DirectCaller) ToParam() DirectCallerParam {
	return param.Override[DirectCallerParam](json.RawMessage(r.RawJSON()))
}

func NewDirectCallerParam() DirectCallerParam {
	return DirectCallerParam{
		Type: "direct",
	}
}

// Tool invocation directly from the model.
//
// This struct has a constant value, construct it with [NewDirectCallerParam].
type DirectCallerParam struct {
	Type constant.Direct `json:"type,required"`
	paramObj
}

func (r DirectCallerParam) MarshalJSON() (data []byte, err error) {
	type shadow DirectCallerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *DirectCallerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type DocumentBlock struct {
	// Citation configuration for the document
	Citations CitationsConfig          `json:"citations,required"`
	Source    DocumentBlockSourceUnion `json:"source,required"`
	// The title of the document
	Title string            `json:"title,required"`
	Type  constant.Document `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Citations   respjson.Field
		Source      respjson.Field
		Title       respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r DocumentBlock) RawJSON() string { return r.JSON.raw }
func (r *DocumentBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// DocumentBlockSourceUnion contains all possible properties and values from
// [Base64PDFSource], [PlainTextSource].
//
// Use the [DocumentBlockSourceUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type DocumentBlockSourceUnion struct {
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
	// Any of "base64", "text".
	Type string `json:"type"`
	JSON struct {
		Data      respjson.Field
		MediaType respjson.Field
		Type      respjson.Field
		raw       string
	} `json:"-"`
}

// anyDocumentBlockSource is implemented by each variant of
// [DocumentBlockSourceUnion] to add type safety for the return type of
// [DocumentBlockSourceUnion.AsAny]
type anyDocumentBlockSource interface {
	implDocumentBlockSourceUnion()
}

func (Base64PDFSource) implDocumentBlockSourceUnion() {}
func (PlainTextSource) implDocumentBlockSourceUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := DocumentBlockSourceUnion.AsAny().(type) {
//	case anthropic.Base64PDFSource:
//	case anthropic.PlainTextSource:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u DocumentBlockSourceUnion) AsAny() anyDocumentBlockSource {
	switch u.Type {
	case "base64":
		return u.AsBase64()
	case "text":
		return u.AsText()
	}
	return nil
}

func (u DocumentBlockSourceUnion) AsBase64() (v Base64PDFSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u DocumentBlockSourceUnion) AsText() (v PlainTextSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u DocumentBlockSourceUnion) RawJSON() string { return u.JSON.raw }

func (r *DocumentBlockSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Source, Type are required.
type DocumentBlockParam struct {
	Source  DocumentBlockParamSourceUnion `json:"source,omitzero,required"`
	Context param.Opt[string]             `json:"context,omitzero"`
	Title   param.Opt[string]             `json:"title,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	Citations    CitationsConfigParam       `json:"citations,omitzero"`
	// This field can be elided, and will marshal its zero value as "document".
	Type constant.Document `json:"type,required"`
	paramObj
}

func (r DocumentBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow DocumentBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *DocumentBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type DocumentBlockParamSourceUnion struct {
	OfBase64  *Base64PDFSourceParam    `json:",omitzero,inline"`
	OfText    *PlainTextSourceParam    `json:",omitzero,inline"`
	OfContent *ContentBlockSourceParam `json:",omitzero,inline"`
	OfURL     *URLPDFSourceParam       `json:",omitzero,inline"`
	paramUnion
}

func (u DocumentBlockParamSourceUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBase64, u.OfText, u.OfContent, u.OfURL)
}
func (u *DocumentBlockParamSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *DocumentBlockParamSourceUnion) asAny() any {
	if !param.IsOmitted(u.OfBase64) {
		return u.OfBase64
	} else if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfContent) {
		return u.OfContent
	} else if !param.IsOmitted(u.OfURL) {
		return u.OfURL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u DocumentBlockParamSourceUnion) GetContent() *ContentBlockSourceContentUnionParam {
	if vt := u.OfContent; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u DocumentBlockParamSourceUnion) GetURL() *string {
	if vt := u.OfURL; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u DocumentBlockParamSourceUnion) GetData() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Data)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Data)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u DocumentBlockParamSourceUnion) GetMediaType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.MediaType)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.MediaType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u DocumentBlockParamSourceUnion) GetType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContent; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfURL; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[DocumentBlockParamSourceUnion](
		"type",
		apijson.Discriminator[Base64PDFSourceParam]("base64"),
		apijson.Discriminator[PlainTextSourceParam]("text"),
		apijson.Discriminator[ContentBlockSourceParam]("content"),
		apijson.Discriminator[URLPDFSourceParam]("url"),
	)
}

// Code execution result with encrypted stdout for PFC + web_search results.
type EncryptedCodeExecutionResultBlock struct {
	Content         []CodeExecutionOutputBlock            `json:"content,required"`
	EncryptedStdout string                                `json:"encrypted_stdout,required"`
	ReturnCode      int64                                 `json:"return_code,required"`
	Stderr          string                                `json:"stderr,required"`
	Type            constant.EncryptedCodeExecutionResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content         respjson.Field
		EncryptedStdout respjson.Field
		ReturnCode      respjson.Field
		Stderr          respjson.Field
		Type            respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r EncryptedCodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *EncryptedCodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Code execution result with encrypted stdout for PFC + web_search results.
//
// The properties Content, EncryptedStdout, ReturnCode, Stderr, Type are required.
type EncryptedCodeExecutionResultBlockParam struct {
	Content         []CodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	EncryptedStdout string                          `json:"encrypted_stdout,required"`
	ReturnCode      int64                           `json:"return_code,required"`
	Stderr          string                          `json:"stderr,required"`
	// This field can be elided, and will marshal its zero value as
	// "encrypted_code_execution_result".
	Type constant.EncryptedCodeExecutionResult `json:"type,required"`
	paramObj
}

func (r EncryptedCodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow EncryptedCodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *EncryptedCodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Source, Type are required.
type ImageBlockParam struct {
	Source ImageBlockParamSourceUnion `json:"source,omitzero,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "image".
	Type constant.Image `json:"type,required"`
	paramObj
}

func (r ImageBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ImageBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ImageBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ImageBlockParamSourceUnion struct {
	OfBase64 *Base64ImageSourceParam `json:",omitzero,inline"`
	OfURL    *URLImageSourceParam    `json:",omitzero,inline"`
	paramUnion
}

func (u ImageBlockParamSourceUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBase64, u.OfURL)
}
func (u *ImageBlockParamSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ImageBlockParamSourceUnion) asAny() any {
	if !param.IsOmitted(u.OfBase64) {
		return u.OfBase64
	} else if !param.IsOmitted(u.OfURL) {
		return u.OfURL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ImageBlockParamSourceUnion) GetData() *string {
	if vt := u.OfBase64; vt != nil {
		return &vt.Data
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ImageBlockParamSourceUnion) GetMediaType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.MediaType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ImageBlockParamSourceUnion) GetURL() *string {
	if vt := u.OfURL; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ImageBlockParamSourceUnion) GetType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfURL; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type InputJSONDelta struct {
	PartialJSON string                  `json:"partial_json,required"`
	Type        constant.InputJSONDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		PartialJSON respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r InputJSONDelta) RawJSON() string { return r.JSON.raw }
func (r *InputJSONDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Schema, Type are required.
type JSONOutputFormatParam struct {
	// The JSON schema of the format
	Schema map[string]any `json:"schema,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "json_schema".
	Type constant.JSONSchema `json:"type,required"`
	paramObj
}

func (r JSONOutputFormatParam) MarshalJSON() (data []byte, err error) {
	type shadow JSONOutputFormatParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *JSONOutputFormatParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type MemoryTool20250818Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "memory".
	Name constant.Memory `json:"name,required"`
	// This field can be elided, and will marshal its zero value as "memory_20250818".
	Type constant.Memory20250818 `json:"type,required"`
	paramObj
}

func (r MemoryTool20250818Param) MarshalJSON() (data []byte, err error) {
	type shadow MemoryTool20250818Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *MemoryTool20250818Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Message struct {
	// Unique object identifier.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// Information about the container used in the request (for the code execution
	// tool)
	Container Container `json:"container,required"`
	// Content generated by the model.
	//
	// This is an array of content blocks, each of which has a `type` that determines
	// its shape.
	//
	// Example:
	//
	// ```json
	// [{ "type": "text", "text": "Hi, I'm Claude." }]
	// ```
	//
	// If the request input `messages` ended with an `assistant` turn, then the
	// response `content` will continue directly from that last turn. You can use this
	// to constrain the model's output.
	//
	// For example, if the input `messages` were:
	//
	// ```json
	// [
	//
	//	{
	//	  "role": "user",
	//	  "content": "What's the Greek name for Sun? (A) Sol (B) Helios (C) Sun"
	//	},
	//	{ "role": "assistant", "content": "The best answer is (" }
	//
	// ]
	// ```
	//
	// Then the response `content` might be:
	//
	// ```json
	// [{ "type": "text", "text": "B)" }]
	// ```
	Content []ContentBlockUnion `json:"content,required"`
	// The model that will complete your prompt.\n\nSee
	// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
	// details and options.
	Model Model `json:"model,required"`
	// Conversational role of the generated message.
	//
	// This will always be `"assistant"`.
	Role constant.Assistant `json:"role,required"`
	// The reason that we stopped.
	//
	// This may be one the following values:
	//
	//   - `"end_turn"`: the model reached a natural stopping point
	//   - `"max_tokens"`: we exceeded the requested `max_tokens` or the model's maximum
	//   - `"stop_sequence"`: one of your provided custom `stop_sequences` was generated
	//   - `"tool_use"`: the model invoked one or more tools
	//   - `"pause_turn"`: we paused a long-running turn. You may provide the response
	//     back as-is in a subsequent request to let the model continue.
	//   - `"refusal"`: when streaming classifiers intervene to handle potential policy
	//     violations
	//
	// In non-streaming mode this value is always non-null. In streaming mode, it is
	// null in the `message_start` event and non-null otherwise.
	//
	// Any of "end_turn", "max_tokens", "stop_sequence", "tool_use", "pause_turn",
	// "refusal".
	StopReason StopReason `json:"stop_reason,required"`
	// Which custom stop sequence was generated, if any.
	//
	// This value will be a non-null string if one of your custom stop sequences was
	// generated.
	StopSequence string `json:"stop_sequence,required"`
	// Object type.
	//
	// For Messages, this is always `"message"`.
	Type constant.Message `json:"type,required"`
	// Billing and rate-limit usage.
	//
	// Anthropic's API bills and rate-limits by token counts, as tokens represent the
	// underlying cost to our systems.
	//
	// Under the hood, the API transforms requests into a format suitable for the
	// model. The model's output then goes through a parsing stage before becoming an
	// API response. As a result, the token counts in `usage` will not match one-to-one
	// with the exact visible content of an API request or response.
	//
	// For example, `output_tokens` will be non-zero, even for an empty string response
	// from Claude.
	//
	// Total input tokens in a request is the summation of `input_tokens`,
	// `cache_creation_input_tokens`, and `cache_read_input_tokens`.
	Usage Usage `json:"usage,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID           respjson.Field
		Container    respjson.Field
		Content      respjson.Field
		Model        respjson.Field
		Role         respjson.Field
		StopReason   respjson.Field
		StopSequence respjson.Field
		Type         respjson.Field
		Usage        respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Message) RawJSON() string { return r.JSON.raw }
func (r *Message) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The reason that we stopped.
//
// This may be one the following values:
//
// - `"end_turn"`: the model reached a natural stopping point
// - `"max_tokens"`: we exceeded the requested `max_tokens` or the model's maximum
// - `"stop_sequence"`: one of your provided custom `stop_sequences` was generated
// - `"tool_use"`: the model invoked one or more tools
//
// In non-streaming mode this value is always non-null. In streaming mode, it is
// null in the `message_start` event and non-null otherwise.
type MessageStopReason string

const (
	MessageStopReasonEndTurn      MessageStopReason = "end_turn"
	MessageStopReasonMaxTokens    MessageStopReason = "max_tokens"
	MessageStopReasonStopSequence MessageStopReason = "stop_sequence"
	MessageStopReasonToolUse      MessageStopReason = "tool_use"
)

func MessageCountTokensToolParamOfTool(inputSchema ToolInputSchemaParam, name string) MessageCountTokensToolUnionParam {
	var variant ToolParam
	variant.InputSchema = inputSchema
	variant.Name = name
	return MessageCountTokensToolUnionParam{OfTool: &variant}
}

func MessageCountTokensToolParamOfToolSearchToolBm25_20251119(type_ ToolSearchToolBm25_20251119Type) MessageCountTokensToolUnionParam {
	var variant ToolSearchToolBm25_20251119Param
	variant.Type = type_
	return MessageCountTokensToolUnionParam{OfToolSearchToolBm25_20251119: &variant}
}

func MessageCountTokensToolParamOfToolSearchToolRegex20251119(type_ ToolSearchToolRegex20251119Type) MessageCountTokensToolUnionParam {
	var variant ToolSearchToolRegex20251119Param
	variant.Type = type_
	return MessageCountTokensToolUnionParam{OfToolSearchToolRegex20251119: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type MessageCountTokensToolUnionParam struct {
	OfTool                        *ToolParam                        `json:",omitzero,inline"`
	OfBashTool20250124            *ToolBash20250124Param            `json:",omitzero,inline"`
	OfCodeExecutionTool20250522   *CodeExecutionTool20250522Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20250825   *CodeExecutionTool20250825Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20260120   *CodeExecutionTool20260120Param   `json:",omitzero,inline"`
	OfMemoryTool20250818          *MemoryTool20250818Param          `json:",omitzero,inline"`
	OfTextEditor20250124          *ToolTextEditor20250124Param      `json:",omitzero,inline"`
	OfTextEditor20250429          *ToolTextEditor20250429Param      `json:",omitzero,inline"`
	OfTextEditor20250728          *ToolTextEditor20250728Param      `json:",omitzero,inline"`
	OfWebSearchTool20250305       *WebSearchTool20250305Param       `json:",omitzero,inline"`
	OfWebFetchTool20250910        *WebFetchTool20250910Param        `json:",omitzero,inline"`
	OfWebSearchTool20260209       *WebSearchTool20260209Param       `json:",omitzero,inline"`
	OfWebFetchTool20260209        *WebFetchTool20260209Param        `json:",omitzero,inline"`
	OfToolSearchToolBm25_20251119 *ToolSearchToolBm25_20251119Param `json:",omitzero,inline"`
	OfToolSearchToolRegex20251119 *ToolSearchToolRegex20251119Param `json:",omitzero,inline"`
	paramUnion
}

func (u MessageCountTokensToolUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfTool,
		u.OfBashTool20250124,
		u.OfCodeExecutionTool20250522,
		u.OfCodeExecutionTool20250825,
		u.OfCodeExecutionTool20260120,
		u.OfMemoryTool20250818,
		u.OfTextEditor20250124,
		u.OfTextEditor20250429,
		u.OfTextEditor20250728,
		u.OfWebSearchTool20250305,
		u.OfWebFetchTool20250910,
		u.OfWebSearchTool20260209,
		u.OfWebFetchTool20260209,
		u.OfToolSearchToolBm25_20251119,
		u.OfToolSearchToolRegex20251119)
}
func (u *MessageCountTokensToolUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *MessageCountTokensToolUnionParam) asAny() any {
	if !param.IsOmitted(u.OfTool) {
		return u.OfTool
	} else if !param.IsOmitted(u.OfBashTool20250124) {
		return u.OfBashTool20250124
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250522) {
		return u.OfCodeExecutionTool20250522
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250825) {
		return u.OfCodeExecutionTool20250825
	} else if !param.IsOmitted(u.OfCodeExecutionTool20260120) {
		return u.OfCodeExecutionTool20260120
	} else if !param.IsOmitted(u.OfMemoryTool20250818) {
		return u.OfMemoryTool20250818
	} else if !param.IsOmitted(u.OfTextEditor20250124) {
		return u.OfTextEditor20250124
	} else if !param.IsOmitted(u.OfTextEditor20250429) {
		return u.OfTextEditor20250429
	} else if !param.IsOmitted(u.OfTextEditor20250728) {
		return u.OfTextEditor20250728
	} else if !param.IsOmitted(u.OfWebSearchTool20250305) {
		return u.OfWebSearchTool20250305
	} else if !param.IsOmitted(u.OfWebFetchTool20250910) {
		return u.OfWebFetchTool20250910
	} else if !param.IsOmitted(u.OfWebSearchTool20260209) {
		return u.OfWebSearchTool20260209
	} else if !param.IsOmitted(u.OfWebFetchTool20260209) {
		return u.OfWebFetchTool20260209
	} else if !param.IsOmitted(u.OfToolSearchToolBm25_20251119) {
		return u.OfToolSearchToolBm25_20251119
	} else if !param.IsOmitted(u.OfToolSearchToolRegex20251119) {
		return u.OfToolSearchToolRegex20251119
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetInputSchema() *ToolInputSchemaParam {
	if vt := u.OfTool; vt != nil {
		return &vt.InputSchema
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetDescription() *string {
	if vt := u.OfTool; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetEagerInputStreaming() *bool {
	if vt := u.OfTool; vt != nil && vt.EagerInputStreaming.Valid() {
		return &vt.EagerInputStreaming.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetMaxCharacters() *int64 {
	if vt := u.OfTextEditor20250728; vt != nil && vt.MaxCharacters.Valid() {
		return &vt.MaxCharacters.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return (*string)(&vt.Name)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetDeferLoading() *bool {
	if vt := u.OfTool; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250429; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250728; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebSearchTool20250305; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetStrict() *bool {
	if vt := u.OfTool; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250429; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250728; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebSearchTool20250305; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetType() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetMaxUses() *int64 {
	if vt := u.OfWebSearchTool20250305; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u MessageCountTokensToolUnionParam) GetMaxContentTokens() *int64 {
	if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedCallers property, if
// present.
func (u MessageCountTokensToolUnionParam) GetAllowedCallers() []string {
	if vt := u.OfTool; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return vt.AllowedCallers
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u MessageCountTokensToolUnionParam) GetCacheControl() *CacheControlEphemeralParam {
	if vt := u.OfTool; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a pointer to the underlying variant's InputExamples property, if
// present.
func (u MessageCountTokensToolUnionParam) GetInputExamples() []map[string]any {
	if vt := u.OfTool; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return vt.InputExamples
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedDomains property, if
// present.
func (u MessageCountTokensToolUnionParam) GetAllowedDomains() []string {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.AllowedDomains
	}
	return nil
}

// Returns a pointer to the underlying variant's BlockedDomains property, if
// present.
func (u MessageCountTokensToolUnionParam) GetBlockedDomains() []string {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.BlockedDomains
	}
	return nil
}

// Returns a pointer to the underlying variant's UserLocation property, if present.
func (u MessageCountTokensToolUnionParam) GetUserLocation() *UserLocationParam {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.UserLocation
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.UserLocation
	}
	return nil
}

// Returns a pointer to the underlying variant's Citations property, if present.
func (u MessageCountTokensToolUnionParam) GetCitations() *CitationsConfigParam {
	if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.Citations
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.Citations
	}
	return nil
}

type MessageDeltaUsage struct {
	// The cumulative number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The cumulative number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The cumulative number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// The cumulative number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// The number of server tool requests.
	ServerToolUse ServerToolUsage `json:"server_tool_use,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InputTokens              respjson.Field
		OutputTokens             respjson.Field
		ServerToolUse            respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageDeltaUsage) RawJSON() string { return r.JSON.raw }
func (r *MessageDeltaUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, Role are required.
type MessageParam struct {
	Content []ContentBlockParamUnion `json:"content,omitzero,required"`
	// Any of "user", "assistant".
	Role MessageParamRole `json:"role,omitzero,required"`
	paramObj
}

func NewUserMessage(blocks ...ContentBlockParamUnion) MessageParam {
	return MessageParam{
		Role:    MessageParamRoleUser,
		Content: blocks,
	}
}

func NewAssistantMessage(blocks ...ContentBlockParamUnion) MessageParam {
	return MessageParam{
		Role:    MessageParamRoleAssistant,
		Content: blocks,
	}
}

func (r MessageParam) MarshalJSON() (data []byte, err error) {
	type shadow MessageParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *MessageParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageParamRole string

const (
	MessageParamRoleUser      MessageParamRole = "user"
	MessageParamRoleAssistant MessageParamRole = "assistant"
)

type MessageTokensCount struct {
	// The total number of tokens across the provided list of messages, system prompt,
	// and tools.
	InputTokens int64 `json:"input_tokens,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		InputTokens respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageTokensCount) RawJSON() string { return r.JSON.raw }
func (r *MessageTokensCount) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MetadataParam struct {
	// An external identifier for the user who is associated with the request.
	//
	// This should be a uuid, hash value, or other opaque identifier. Anthropic may use
	// this id to help detect abuse. Do not include any identifying information such as
	// name, email address, or phone number.
	UserID param.Opt[string] `json:"user_id,omitzero"`
	paramObj
}

func (r MetadataParam) MarshalJSON() (data []byte, err error) {
	type shadow MetadataParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *MetadataParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The model that will complete your prompt.\n\nSee
// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
// details and options.
type Model string

const (
	ModelClaudeOpus4_6          Model = "claude-opus-4-6"
	ModelClaudeSonnet4_6        Model = "claude-sonnet-4-6"
	ModelClaudeOpus4_5_20251101 Model = "claude-opus-4-5-20251101"
	ModelClaudeOpus4_5          Model = "claude-opus-4-5"
	// Deprecated: Will reach end-of-life on February 19th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude3_7SonnetLatest Model = "claude-3-7-sonnet-latest"
	// Deprecated: Will reach end-of-life on February 19th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude3_7Sonnet20250219 Model = "claude-3-7-sonnet-20250219"
	// Deprecated: Will reach end-of-life on February 19th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude3_5HaikuLatest Model = "claude-3-5-haiku-latest"
	// Deprecated: Will reach end-of-life on February 19th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude3_5Haiku20241022   Model = "claude-3-5-haiku-20241022"
	ModelClaudeHaiku4_5           Model = "claude-haiku-4-5"
	ModelClaudeHaiku4_5_20251001  Model = "claude-haiku-4-5-20251001"
	ModelClaudeSonnet4_20250514   Model = "claude-sonnet-4-20250514"
	ModelClaudeSonnet4_0          Model = "claude-sonnet-4-0"
	ModelClaude4Sonnet20250514    Model = "claude-4-sonnet-20250514"
	ModelClaudeSonnet4_5          Model = "claude-sonnet-4-5"
	ModelClaudeSonnet4_5_20250929 Model = "claude-sonnet-4-5-20250929"
	ModelClaudeOpus4_0            Model = "claude-opus-4-0"
	ModelClaudeOpus4_20250514     Model = "claude-opus-4-20250514"
	ModelClaude4Opus20250514      Model = "claude-4-opus-20250514"
	ModelClaudeOpus4_1_20250805   Model = "claude-opus-4-1-20250805"
	// Deprecated: Will reach end-of-life on January 5th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude3OpusLatest Model = "claude-3-opus-latest"
	// Deprecated: Will reach end-of-life on January 5th, 2026. Please migrate to a
	// newer model. Visit
	// https://docs.anthropic.com/en/docs/resources/model-deprecations for more
	// information.
	ModelClaude_3_Opus_20240229  Model = "claude-3-opus-20240229"
	ModelClaude_3_Haiku_20240307 Model = "claude-3-haiku-20240307"
)

type OutputConfigParam struct {
	// All possible effort levels.
	//
	// Any of "low", "medium", "high", "max".
	Effort OutputConfigEffort `json:"effort,omitzero"`
	// A schema to specify Claude's output format in responses. See
	// [structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
	Format JSONOutputFormatParam `json:"format,omitzero"`
	paramObj
}

func (r OutputConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow OutputConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *OutputConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// All possible effort levels.
type OutputConfigEffort string

const (
	OutputConfigEffortLow    OutputConfigEffort = "low"
	OutputConfigEffortMedium OutputConfigEffort = "medium"
	OutputConfigEffortHigh   OutputConfigEffort = "high"
	OutputConfigEffortMax    OutputConfigEffort = "max"
)

type PlainTextSource struct {
	Data      string             `json:"data,required"`
	MediaType constant.TextPlain `json:"media_type,required"`
	Type      constant.Text      `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		MediaType   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PlainTextSource) RawJSON() string { return r.JSON.raw }
func (r *PlainTextSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this PlainTextSource to a PlainTextSourceParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// PlainTextSourceParam.Overrides()
func (r PlainTextSource) ToParam() PlainTextSourceParam {
	return param.Override[PlainTextSourceParam](json.RawMessage(r.RawJSON()))
}

// The properties Data, MediaType, Type are required.
type PlainTextSourceParam struct {
	Data string `json:"data,required"`
	// This field can be elided, and will marshal its zero value as "text/plain".
	MediaType constant.TextPlain `json:"media_type,required"`
	// This field can be elided, and will marshal its zero value as "text".
	Type constant.Text `json:"type,required"`
	paramObj
}

func (r PlainTextSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow PlainTextSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *PlainTextSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RawContentBlockDeltaUnion contains all possible properties and values from
// [TextDelta], [InputJSONDelta], [CitationsDelta], [ThinkingDelta],
// [SignatureDelta].
//
// Use the [RawContentBlockDeltaUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type RawContentBlockDeltaUnion struct {
	// This field is from variant [TextDelta].
	Text string `json:"text"`
	// Any of "text_delta", "input_json_delta", "citations_delta", "thinking_delta",
	// "signature_delta".
	Type string `json:"type"`
	// This field is from variant [InputJSONDelta].
	PartialJSON string `json:"partial_json"`
	// This field is from variant [CitationsDelta].
	Citation CitationsDeltaCitationUnion `json:"citation"`
	// This field is from variant [ThinkingDelta].
	Thinking string `json:"thinking"`
	// This field is from variant [SignatureDelta].
	Signature string `json:"signature"`
	JSON      struct {
		Text        respjson.Field
		Type        respjson.Field
		PartialJSON respjson.Field
		Citation    respjson.Field
		Thinking    respjson.Field
		Signature   respjson.Field
		raw         string
	} `json:"-"`
}

// anyRawContentBlockDelta is implemented by each variant of
// [RawContentBlockDeltaUnion] to add type safety for the return type of
// [RawContentBlockDeltaUnion.AsAny]
type anyRawContentBlockDelta interface {
	implRawContentBlockDeltaUnion()
}

func (TextDelta) implRawContentBlockDeltaUnion()      {}
func (InputJSONDelta) implRawContentBlockDeltaUnion() {}
func (CitationsDelta) implRawContentBlockDeltaUnion() {}
func (ThinkingDelta) implRawContentBlockDeltaUnion()  {}
func (SignatureDelta) implRawContentBlockDeltaUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := RawContentBlockDeltaUnion.AsAny().(type) {
//	case anthropic.TextDelta:
//	case anthropic.InputJSONDelta:
//	case anthropic.CitationsDelta:
//	case anthropic.ThinkingDelta:
//	case anthropic.SignatureDelta:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u RawContentBlockDeltaUnion) AsAny() anyRawContentBlockDelta {
	switch u.Type {
	case "text_delta":
		return u.AsTextDelta()
	case "input_json_delta":
		return u.AsInputJSONDelta()
	case "citations_delta":
		return u.AsCitationsDelta()
	case "thinking_delta":
		return u.AsThinkingDelta()
	case "signature_delta":
		return u.AsSignatureDelta()
	}
	return nil
}

func (u RawContentBlockDeltaUnion) AsTextDelta() (v TextDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RawContentBlockDeltaUnion) AsInputJSONDelta() (v InputJSONDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RawContentBlockDeltaUnion) AsCitationsDelta() (v CitationsDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RawContentBlockDeltaUnion) AsThinkingDelta() (v ThinkingDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RawContentBlockDeltaUnion) AsSignatureDelta() (v SignatureDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RawContentBlockDeltaUnion) RawJSON() string { return u.JSON.raw }

func (r *RawContentBlockDeltaUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ContentBlockDeltaEvent struct {
	Delta RawContentBlockDeltaUnion  `json:"delta,required"`
	Index int64                      `json:"index,required"`
	Type  constant.ContentBlockDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Delta       respjson.Field
		Index       respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ContentBlockDeltaEvent) RawJSON() string { return r.JSON.raw }
func (r *ContentBlockDeltaEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ContentBlockStartEvent struct {
	// Response model for a file uploaded to the container.
	ContentBlock ContentBlockStartEventContentBlockUnion `json:"content_block,required"`
	Index        int64                                   `json:"index,required"`
	Type         constant.ContentBlockStart              `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ContentBlock respjson.Field
		Index        respjson.Field
		Type         respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ContentBlockStartEvent) RawJSON() string { return r.JSON.raw }
func (r *ContentBlockStartEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockStartEventContentBlockUnion contains all possible properties and
// values from [TextBlock], [ThinkingBlock], [RedactedThinkingBlock],
// [ToolUseBlock], [ServerToolUseBlock], [WebSearchToolResultBlock],
// [WebFetchToolResultBlock], [CodeExecutionToolResultBlock],
// [BashCodeExecutionToolResultBlock], [TextEditorCodeExecutionToolResultBlock],
// [ToolSearchToolResultBlock], [ContainerUploadBlock].
//
// Use the [ContentBlockStartEventContentBlockUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ContentBlockStartEventContentBlockUnion struct {
	// This field is from variant [TextBlock].
	Citations []TextCitationUnion `json:"citations"`
	// This field is from variant [TextBlock].
	Text string `json:"text"`
	// Any of "text", "thinking", "redacted_thinking", "tool_use", "server_tool_use",
	// "web_search_tool_result", "web_fetch_tool_result", "code_execution_tool_result",
	// "bash_code_execution_tool_result", "text_editor_code_execution_tool_result",
	// "tool_search_tool_result", "container_upload".
	Type string `json:"type"`
	// This field is from variant [ThinkingBlock].
	Signature string `json:"signature"`
	// This field is from variant [ThinkingBlock].
	Thinking string `json:"thinking"`
	// This field is from variant [RedactedThinkingBlock].
	Data string `json:"data"`
	ID   string `json:"id"`
	// This field is a union of [ToolUseBlockCallerUnion],
	// [ServerToolUseBlockCallerUnion], [WebSearchToolResultBlockCallerUnion],
	// [WebFetchToolResultBlockCallerUnion]
	Caller ContentBlockStartEventContentBlockUnionCaller `json:"caller"`
	Input  any                                           `json:"input"`
	Name   string                                        `json:"name"`
	// This field is a union of [WebSearchToolResultBlockContentUnion],
	// [WebFetchToolResultBlockContentUnion],
	// [CodeExecutionToolResultBlockContentUnion],
	// [BashCodeExecutionToolResultBlockContentUnion],
	// [TextEditorCodeExecutionToolResultBlockContentUnion],
	// [ToolSearchToolResultBlockContentUnion]
	Content   ContentBlockStartEventContentBlockUnionContent `json:"content"`
	ToolUseID string                                         `json:"tool_use_id"`
	// This field is from variant [ContainerUploadBlock].
	FileID string `json:"file_id"`
	JSON   struct {
		Citations respjson.Field
		Text      respjson.Field
		Type      respjson.Field
		Signature respjson.Field
		Thinking  respjson.Field
		Data      respjson.Field
		ID        respjson.Field
		Caller    respjson.Field
		Input     respjson.Field
		Name      respjson.Field
		Content   respjson.Field
		ToolUseID respjson.Field
		FileID    respjson.Field
		raw       string
	} `json:"-"`
}

// anyContentBlockStartEventContentBlock is implemented by each variant of
// [ContentBlockStartEventContentBlockUnion] to add type safety for the return type
// of [ContentBlockStartEventContentBlockUnion.AsAny]
type anyContentBlockStartEventContentBlock interface {
	implContentBlockStartEventContentBlockUnion()
}

func (TextBlock) implContentBlockStartEventContentBlockUnion()                              {}
func (ThinkingBlock) implContentBlockStartEventContentBlockUnion()                          {}
func (RedactedThinkingBlock) implContentBlockStartEventContentBlockUnion()                  {}
func (ToolUseBlock) implContentBlockStartEventContentBlockUnion()                           {}
func (ServerToolUseBlock) implContentBlockStartEventContentBlockUnion()                     {}
func (WebSearchToolResultBlock) implContentBlockStartEventContentBlockUnion()               {}
func (WebFetchToolResultBlock) implContentBlockStartEventContentBlockUnion()                {}
func (CodeExecutionToolResultBlock) implContentBlockStartEventContentBlockUnion()           {}
func (BashCodeExecutionToolResultBlock) implContentBlockStartEventContentBlockUnion()       {}
func (TextEditorCodeExecutionToolResultBlock) implContentBlockStartEventContentBlockUnion() {}
func (ToolSearchToolResultBlock) implContentBlockStartEventContentBlockUnion()              {}
func (ContainerUploadBlock) implContentBlockStartEventContentBlockUnion()                   {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ContentBlockStartEventContentBlockUnion.AsAny().(type) {
//	case anthropic.TextBlock:
//	case anthropic.ThinkingBlock:
//	case anthropic.RedactedThinkingBlock:
//	case anthropic.ToolUseBlock:
//	case anthropic.ServerToolUseBlock:
//	case anthropic.WebSearchToolResultBlock:
//	case anthropic.WebFetchToolResultBlock:
//	case anthropic.CodeExecutionToolResultBlock:
//	case anthropic.BashCodeExecutionToolResultBlock:
//	case anthropic.TextEditorCodeExecutionToolResultBlock:
//	case anthropic.ToolSearchToolResultBlock:
//	case anthropic.ContainerUploadBlock:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ContentBlockStartEventContentBlockUnion) AsAny() anyContentBlockStartEventContentBlock {
	switch u.Type {
	case "text":
		return u.AsText()
	case "thinking":
		return u.AsThinking()
	case "redacted_thinking":
		return u.AsRedactedThinking()
	case "tool_use":
		return u.AsToolUse()
	case "server_tool_use":
		return u.AsServerToolUse()
	case "web_search_tool_result":
		return u.AsWebSearchToolResult()
	case "web_fetch_tool_result":
		return u.AsWebFetchToolResult()
	case "code_execution_tool_result":
		return u.AsCodeExecutionToolResult()
	case "bash_code_execution_tool_result":
		return u.AsBashCodeExecutionToolResult()
	case "text_editor_code_execution_tool_result":
		return u.AsTextEditorCodeExecutionToolResult()
	case "tool_search_tool_result":
		return u.AsToolSearchToolResult()
	case "container_upload":
		return u.AsContainerUpload()
	}
	return nil
}

func (u ContentBlockStartEventContentBlockUnion) AsText() (v TextBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsThinking() (v ThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsRedactedThinking() (v RedactedThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsToolUse() (v ToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsServerToolUse() (v ServerToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsWebSearchToolResult() (v WebSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsWebFetchToolResult() (v WebFetchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsCodeExecutionToolResult() (v CodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsBashCodeExecutionToolResult() (v BashCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsTextEditorCodeExecutionToolResult() (v TextEditorCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsToolSearchToolResult() (v ToolSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ContentBlockStartEventContentBlockUnion) AsContainerUpload() (v ContainerUploadBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ContentBlockStartEventContentBlockUnion) RawJSON() string { return u.JSON.raw }

func (r *ContentBlockStartEventContentBlockUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockStartEventContentBlockUnionCaller is an implicit subunion of
// [ContentBlockStartEventContentBlockUnion].
// ContentBlockStartEventContentBlockUnionCaller provides convenient access to the
// sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockStartEventContentBlockUnion].
type ContentBlockStartEventContentBlockUnionCaller struct {
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

func (r *ContentBlockStartEventContentBlockUnionCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockStartEventContentBlockUnionContent is an implicit subunion of
// [ContentBlockStartEventContentBlockUnion].
// ContentBlockStartEventContentBlockUnionContent provides convenient access to the
// sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockStartEventContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfWebSearchResultBlockArray]
type ContentBlockStartEventContentBlockUnionContent struct {
	// This field will be present if the value is a [[]WebSearchResultBlock] instead of
	// an object.
	OfWebSearchResultBlockArray []WebSearchResultBlock `json:",inline"`
	ErrorCode                   string                 `json:"error_code"`
	Type                        string                 `json:"type"`
	// This field is a union of [DocumentBlock], [[]CodeExecutionOutputBlock],
	// [[]CodeExecutionOutputBlock], [[]BashCodeExecutionOutputBlock], [string]
	Content ContentBlockStartEventContentBlockUnionContentContent `json:"content"`
	// This field is from variant [WebFetchToolResultBlockContentUnion].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [WebFetchToolResultBlockContentUnion].
	URL        string `json:"url"`
	ReturnCode int64  `json:"return_code"`
	Stderr     string `json:"stderr"`
	Stdout     string `json:"stdout"`
	// This field is from variant [CodeExecutionToolResultBlockContentUnion].
	EncryptedStdout string `json:"encrypted_stdout"`
	ErrorMessage    string `json:"error_message"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	FileType TextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NumLines int64 `json:"num_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	StartLine int64 `json:"start_line"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	Lines []string `json:"lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NewLines int64 `json:"new_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	NewStart int64 `json:"new_start"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	OldLines int64 `json:"old_lines"`
	// This field is from variant [TextEditorCodeExecutionToolResultBlockContentUnion].
	OldStart int64 `json:"old_start"`
	// This field is from variant [ToolSearchToolResultBlockContentUnion].
	ToolReferences []ToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		OfWebSearchResultBlockArray respjson.Field
		ErrorCode                   respjson.Field
		Type                        respjson.Field
		Content                     respjson.Field
		RetrievedAt                 respjson.Field
		URL                         respjson.Field
		ReturnCode                  respjson.Field
		Stderr                      respjson.Field
		Stdout                      respjson.Field
		EncryptedStdout             respjson.Field
		ErrorMessage                respjson.Field
		FileType                    respjson.Field
		NumLines                    respjson.Field
		StartLine                   respjson.Field
		TotalLines                  respjson.Field
		IsFileUpdate                respjson.Field
		Lines                       respjson.Field
		NewLines                    respjson.Field
		NewStart                    respjson.Field
		OldLines                    respjson.Field
		OldStart                    respjson.Field
		ToolReferences              respjson.Field
		raw                         string
	} `json:"-"`
}

func (r *ContentBlockStartEventContentBlockUnionContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ContentBlockStartEventContentBlockUnionContentContent is an implicit subunion of
// [ContentBlockStartEventContentBlockUnion].
// ContentBlockStartEventContentBlockUnionContentContent provides convenient access
// to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ContentBlockStartEventContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfContent OfString]
type ContentBlockStartEventContentBlockUnionContentContent struct {
	// This field will be present if the value is a [[]CodeExecutionOutputBlock]
	// instead of an object.
	OfContent []CodeExecutionOutputBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is from variant [DocumentBlock].
	Citations CitationsConfig `json:"citations"`
	// This field is from variant [DocumentBlock].
	Source DocumentBlockSourceUnion `json:"source"`
	// This field is from variant [DocumentBlock].
	Title string `json:"title"`
	// This field is from variant [DocumentBlock].
	Type constant.Document `json:"type"`
	JSON struct {
		OfContent respjson.Field
		OfString  respjson.Field
		Citations respjson.Field
		Source    respjson.Field
		Title     respjson.Field
		Type      respjson.Field
		raw       string
	} `json:"-"`
}

func (r *ContentBlockStartEventContentBlockUnionContentContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ContentBlockStopEvent struct {
	Index int64                     `json:"index,required"`
	Type  constant.ContentBlockStop `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Index       respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ContentBlockStopEvent) RawJSON() string { return r.JSON.raw }
func (r *ContentBlockStopEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageDeltaEvent struct {
	Delta MessageDeltaEventDelta `json:"delta,required"`
	Type  constant.MessageDelta  `json:"type,required"`
	// Billing and rate-limit usage.
	//
	// Anthropic's API bills and rate-limits by token counts, as tokens represent the
	// underlying cost to our systems.
	//
	// Under the hood, the API transforms requests into a format suitable for the
	// model. The model's output then goes through a parsing stage before becoming an
	// API response. As a result, the token counts in `usage` will not match one-to-one
	// with the exact visible content of an API request or response.
	//
	// For example, `output_tokens` will be non-zero, even for an empty string response
	// from Claude.
	//
	// Total input tokens in a request is the summation of `input_tokens`,
	// `cache_creation_input_tokens`, and `cache_read_input_tokens`.
	Usage MessageDeltaUsage `json:"usage,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Delta       respjson.Field
		Type        respjson.Field
		Usage       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageDeltaEvent) RawJSON() string { return r.JSON.raw }
func (r *MessageDeltaEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageDeltaEventDelta struct {
	// Information about the container used in the request (for the code execution
	// tool)
	Container Container `json:"container,required"`
	// Any of "end_turn", "max_tokens", "stop_sequence", "tool_use", "pause_turn",
	// "refusal".
	StopReason   StopReason `json:"stop_reason,required"`
	StopSequence string     `json:"stop_sequence,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Container    respjson.Field
		StopReason   respjson.Field
		StopSequence respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageDeltaEventDelta) RawJSON() string { return r.JSON.raw }
func (r *MessageDeltaEventDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageStartEvent struct {
	Message Message               `json:"message,required"`
	Type    constant.MessageStart `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageStartEvent) RawJSON() string { return r.JSON.raw }
func (r *MessageStartEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type MessageStopEvent struct {
	Type constant.MessageStop `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageStopEvent) RawJSON() string { return r.JSON.raw }
func (r *MessageStopEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageStreamEventUnion contains all possible properties and values from
// [MessageStartEvent], [MessageDeltaEvent], [MessageStopEvent],
// [ContentBlockStartEvent], [ContentBlockDeltaEvent], [ContentBlockStopEvent].
//
// Use the [MessageStreamEventUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type MessageStreamEventUnion struct {
	// This field is from variant [MessageStartEvent].
	Message Message `json:"message"`
	// Any of "message_start", "message_delta", "message_stop", "content_block_start",
	// "content_block_delta", "content_block_stop".
	Type string `json:"type"`
	// This field is a union of [MessageDeltaEventDelta], [RawContentBlockDeltaUnion]
	Delta MessageStreamEventUnionDelta `json:"delta"`
	// This field is from variant [MessageDeltaEvent].
	Usage MessageDeltaUsage `json:"usage"`
	// This field is from variant [ContentBlockStartEvent].
	ContentBlock ContentBlockStartEventContentBlockUnion `json:"content_block"`
	Index        int64                                   `json:"index"`
	JSON         struct {
		Message      respjson.Field
		Type         respjson.Field
		Delta        respjson.Field
		Usage        respjson.Field
		ContentBlock respjson.Field
		Index        respjson.Field
		raw          string
	} `json:"-"`
}

// anyMessageStreamEvent is implemented by each variant of
// [MessageStreamEventUnion] to add type safety for the return type of
// [MessageStreamEventUnion.AsAny]
type anyMessageStreamEvent interface {
	implMessageStreamEventUnion()
}

func (MessageStartEvent) implMessageStreamEventUnion()      {}
func (MessageDeltaEvent) implMessageStreamEventUnion()      {}
func (MessageStopEvent) implMessageStreamEventUnion()       {}
func (ContentBlockStartEvent) implMessageStreamEventUnion() {}
func (ContentBlockDeltaEvent) implMessageStreamEventUnion() {}
func (ContentBlockStopEvent) implMessageStreamEventUnion()  {}

// Use the following switch statement to find the correct variant
//
//	switch variant := MessageStreamEventUnion.AsAny().(type) {
//	case anthropic.MessageStartEvent:
//	case anthropic.MessageDeltaEvent:
//	case anthropic.MessageStopEvent:
//	case anthropic.ContentBlockStartEvent:
//	case anthropic.ContentBlockDeltaEvent:
//	case anthropic.ContentBlockStopEvent:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u MessageStreamEventUnion) AsAny() anyMessageStreamEvent {
	switch u.Type {
	case "message_start":
		return u.AsMessageStart()
	case "message_delta":
		return u.AsMessageDelta()
	case "message_stop":
		return u.AsMessageStop()
	case "content_block_start":
		return u.AsContentBlockStart()
	case "content_block_delta":
		return u.AsContentBlockDelta()
	case "content_block_stop":
		return u.AsContentBlockStop()
	}
	return nil
}

func (u MessageStreamEventUnion) AsMessageStart() (v MessageStartEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageStreamEventUnion) AsMessageDelta() (v MessageDeltaEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageStreamEventUnion) AsMessageStop() (v MessageStopEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageStreamEventUnion) AsContentBlockStart() (v ContentBlockStartEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageStreamEventUnion) AsContentBlockDelta() (v ContentBlockDeltaEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageStreamEventUnion) AsContentBlockStop() (v ContentBlockStopEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u MessageStreamEventUnion) RawJSON() string { return u.JSON.raw }

func (r *MessageStreamEventUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageStreamEventUnionDelta is an implicit subunion of
// [MessageStreamEventUnion]. MessageStreamEventUnionDelta provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [MessageStreamEventUnion].
type MessageStreamEventUnionDelta struct {
	// This field is from variant [MessageDeltaEventDelta].
	Container Container `json:"container"`
	// This field is from variant [MessageDeltaEventDelta].
	StopReason StopReason `json:"stop_reason"`
	// This field is from variant [MessageDeltaEventDelta].
	StopSequence string `json:"stop_sequence"`
	// This field is from variant [RawContentBlockDeltaUnion].
	Text string `json:"text"`
	Type string `json:"type"`
	// This field is from variant [RawContentBlockDeltaUnion].
	PartialJSON string `json:"partial_json"`
	// This field is from variant [RawContentBlockDeltaUnion].
	Citation CitationsDeltaCitationUnion `json:"citation"`
	// This field is from variant [RawContentBlockDeltaUnion].
	Thinking string `json:"thinking"`
	// This field is from variant [RawContentBlockDeltaUnion].
	Signature string `json:"signature"`
	JSON      struct {
		Container    respjson.Field
		StopReason   respjson.Field
		StopSequence respjson.Field
		Text         respjson.Field
		Type         respjson.Field
		PartialJSON  respjson.Field
		Citation     respjson.Field
		Thinking     respjson.Field
		Signature    respjson.Field
		raw          string
	} `json:"-"`
}

func (r *MessageStreamEventUnionDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RedactedThinkingBlock struct {
	Data string                    `json:"data,required"`
	Type constant.RedactedThinking `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RedactedThinkingBlock) RawJSON() string { return r.JSON.raw }
func (r *RedactedThinkingBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Data, Type are required.
type RedactedThinkingBlockParam struct {
	Data string `json:"data,required"`
	// This field can be elided, and will marshal its zero value as
	// "redacted_thinking".
	Type constant.RedactedThinking `json:"type,required"`
	paramObj
}

func (r RedactedThinkingBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow RedactedThinkingBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RedactedThinkingBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, Source, Title, Type are required.
type SearchResultBlockParam struct {
	Content []TextBlockParam `json:"content,omitzero,required"`
	Source  string           `json:"source,required"`
	Title   string           `json:"title,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	Citations    CitationsConfigParam       `json:"citations,omitzero"`
	// This field can be elided, and will marshal its zero value as "search_result".
	Type constant.SearchResult `json:"type,required"`
	paramObj
}

func (r SearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow SearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *SearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Tool invocation generated by a server-side tool.
type ServerToolCaller struct {
	ToolID string                         `json:"tool_id,required"`
	Type   constant.CodeExecution20250825 `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ToolID      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ServerToolCaller) RawJSON() string { return r.JSON.raw }
func (r *ServerToolCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this ServerToolCaller to a ServerToolCallerParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// ServerToolCallerParam.Overrides()
func (r ServerToolCaller) ToParam() ServerToolCallerParam {
	return param.Override[ServerToolCallerParam](json.RawMessage(r.RawJSON()))
}

// Tool invocation generated by a server-side tool.
//
// The properties ToolID, Type are required.
type ServerToolCallerParam struct {
	ToolID string `json:"tool_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20250825".
	Type constant.CodeExecution20250825 `json:"type,required"`
	paramObj
}

func (r ServerToolCallerParam) MarshalJSON() (data []byte, err error) {
	type shadow ServerToolCallerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ServerToolCallerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ServerToolCaller20260120 struct {
	ToolID string                         `json:"tool_id,required"`
	Type   constant.CodeExecution20260120 `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ToolID      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ServerToolCaller20260120) RawJSON() string { return r.JSON.raw }
func (r *ServerToolCaller20260120) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this ServerToolCaller20260120 to a
// ServerToolCaller20260120Param.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// ServerToolCaller20260120Param.Overrides()
func (r ServerToolCaller20260120) ToParam() ServerToolCaller20260120Param {
	return param.Override[ServerToolCaller20260120Param](json.RawMessage(r.RawJSON()))
}

// The properties ToolID, Type are required.
type ServerToolCaller20260120Param struct {
	ToolID string `json:"tool_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20260120".
	Type constant.CodeExecution20260120 `json:"type,required"`
	paramObj
}

func (r ServerToolCaller20260120Param) MarshalJSON() (data []byte, err error) {
	type shadow ServerToolCaller20260120Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ServerToolCaller20260120Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ServerToolUsage struct {
	// The number of web fetch tool requests.
	WebFetchRequests int64 `json:"web_fetch_requests,required"`
	// The number of web search tool requests.
	WebSearchRequests int64 `json:"web_search_requests,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		WebFetchRequests  respjson.Field
		WebSearchRequests respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ServerToolUsage) RawJSON() string { return r.JSON.raw }
func (r *ServerToolUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ServerToolUseBlock struct {
	ID string `json:"id,required"`
	// Tool invocation directly from the model.
	Caller ServerToolUseBlockCallerUnion `json:"caller,required"`
	Input  any                           `json:"input,required"`
	// Any of "web_search", "web_fetch", "code_execution", "bash_code_execution",
	// "text_editor_code_execution", "tool_search_tool_regex", "tool_search_tool_bm25".
	Name ServerToolUseBlockName `json:"name,required"`
	Type constant.ServerToolUse `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Caller      respjson.Field
		Input       respjson.Field
		Name        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ServerToolUseBlock) RawJSON() string { return r.JSON.raw }
func (r *ServerToolUseBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ServerToolUseBlockCallerUnion contains all possible properties and values from
// [DirectCaller], [ServerToolCaller], [ServerToolCaller20260120].
//
// Use the [ServerToolUseBlockCallerUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ServerToolUseBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyServerToolUseBlockCaller is implemented by each variant of
// [ServerToolUseBlockCallerUnion] to add type safety for the return type of
// [ServerToolUseBlockCallerUnion.AsAny]
type anyServerToolUseBlockCaller interface {
	implServerToolUseBlockCallerUnion()
}

func (DirectCaller) implServerToolUseBlockCallerUnion()             {}
func (ServerToolCaller) implServerToolUseBlockCallerUnion()         {}
func (ServerToolCaller20260120) implServerToolUseBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ServerToolUseBlockCallerUnion.AsAny().(type) {
//	case anthropic.DirectCaller:
//	case anthropic.ServerToolCaller:
//	case anthropic.ServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ServerToolUseBlockCallerUnion) AsAny() anyServerToolUseBlockCaller {
	switch u.Type {
	case "direct":
		return u.AsDirect()
	case "code_execution_20250825":
		return u.AsCodeExecution20250825()
	case "code_execution_20260120":
		return u.AsCodeExecution20260120()
	}
	return nil
}

func (u ServerToolUseBlockCallerUnion) AsDirect() (v DirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ServerToolUseBlockCallerUnion) AsCodeExecution20250825() (v ServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ServerToolUseBlockCallerUnion) AsCodeExecution20260120() (v ServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ServerToolUseBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *ServerToolUseBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ServerToolUseBlockName string

const (
	ServerToolUseBlockNameWebSearch               ServerToolUseBlockName = "web_search"
	ServerToolUseBlockNameWebFetch                ServerToolUseBlockName = "web_fetch"
	ServerToolUseBlockNameCodeExecution           ServerToolUseBlockName = "code_execution"
	ServerToolUseBlockNameBashCodeExecution       ServerToolUseBlockName = "bash_code_execution"
	ServerToolUseBlockNameTextEditorCodeExecution ServerToolUseBlockName = "text_editor_code_execution"
	ServerToolUseBlockNameToolSearchToolRegex     ServerToolUseBlockName = "tool_search_tool_regex"
	ServerToolUseBlockNameToolSearchToolBm25      ServerToolUseBlockName = "tool_search_tool_bm25"
)

// The properties ID, Input, Name, Type are required.
type ServerToolUseBlockParam struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,omitzero,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller ServerToolUseBlockParamCallerUnion `json:"caller,omitzero"`
	// Any of "web_search", "web_fetch", "code_execution", "bash_code_execution",
	// "text_editor_code_execution", "tool_search_tool_regex", "tool_search_tool_bm25".
	Name ServerToolUseBlockParamName `json:"name,required"`
	// This field can be elided, and will marshal its zero value as "server_tool_use".
	Type constant.ServerToolUse `json:"type,required"`
	paramObj
}

func (r ServerToolUseBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ServerToolUseBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ServerToolUseBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ServerToolUseBlockParamName string

const (
	ServerToolUseBlockParamNameWebSearch               ServerToolUseBlockParamName = "web_search"
	ServerToolUseBlockParamNameWebFetch                ServerToolUseBlockParamName = "web_fetch"
	ServerToolUseBlockParamNameCodeExecution           ServerToolUseBlockParamName = "code_execution"
	ServerToolUseBlockParamNameBashCodeExecution       ServerToolUseBlockParamName = "bash_code_execution"
	ServerToolUseBlockParamNameTextEditorCodeExecution ServerToolUseBlockParamName = "text_editor_code_execution"
	ServerToolUseBlockParamNameToolSearchToolRegex     ServerToolUseBlockParamName = "tool_search_tool_regex"
	ServerToolUseBlockParamNameToolSearchToolBm25      ServerToolUseBlockParamName = "tool_search_tool_bm25"
)

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ServerToolUseBlockParamCallerUnion struct {
	OfDirect                *DirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *ServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *ServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u ServerToolUseBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *ServerToolUseBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ServerToolUseBlockParamCallerUnion) asAny() any {
	if !param.IsOmitted(u.OfDirect) {
		return u.OfDirect
	} else if !param.IsOmitted(u.OfCodeExecution20250825) {
		return u.OfCodeExecution20250825
	} else if !param.IsOmitted(u.OfCodeExecution20260120) {
		return u.OfCodeExecution20260120
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ServerToolUseBlockParamCallerUnion) GetType() *string {
	if vt := u.OfDirect; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ServerToolUseBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[ServerToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[DirectCallerParam]("direct"),
		apijson.Discriminator[ServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[ServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

type SignatureDelta struct {
	Signature string                  `json:"signature,required"`
	Type      constant.SignatureDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Signature   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SignatureDelta) RawJSON() string { return r.JSON.raw }
func (r *SignatureDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type StopReason string

const (
	StopReasonEndTurn      StopReason = "end_turn"
	StopReasonMaxTokens    StopReason = "max_tokens"
	StopReasonStopSequence StopReason = "stop_sequence"
	StopReasonToolUse      StopReason = "tool_use"
	StopReasonPauseTurn    StopReason = "pause_turn"
	StopReasonRefusal      StopReason = "refusal"
)

type TextBlock struct {
	// Citations supporting the text block.
	//
	// The type of citation returned will depend on the type of document being cited.
	// Citing a PDF results in `page_location`, plain text results in `char_location`,
	// and content document results in `content_block_location`.
	Citations []TextCitationUnion `json:"citations,required"`
	Text      string              `json:"text,required"`
	Type      constant.Text       `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Citations   respjson.Field
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextBlock) RawJSON() string { return r.JSON.raw }
func (r *TextBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Text, Type are required.
type TextBlockParam struct {
	Text      string                   `json:"text,required"`
	Citations []TextCitationParamUnion `json:"citations,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "text".
	Type constant.Text `json:"type,required"`
	paramObj
}

func (r TextBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow TextBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// TextCitationUnion contains all possible properties and values from
// [CitationCharLocation], [CitationPageLocation], [CitationContentBlockLocation],
// [CitationsWebSearchResultLocation], [CitationsSearchResultLocation].
//
// Use the [TextCitationUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type TextCitationUnion struct {
	CitedText     string `json:"cited_text"`
	DocumentIndex int64  `json:"document_index"`
	DocumentTitle string `json:"document_title"`
	// This field is from variant [CitationCharLocation].
	EndCharIndex int64  `json:"end_char_index"`
	FileID       string `json:"file_id"`
	// This field is from variant [CitationCharLocation].
	StartCharIndex int64 `json:"start_char_index"`
	// Any of "char_location", "page_location", "content_block_location",
	// "web_search_result_location", "search_result_location".
	Type string `json:"type"`
	// This field is from variant [CitationPageLocation].
	EndPageNumber int64 `json:"end_page_number"`
	// This field is from variant [CitationPageLocation].
	StartPageNumber int64 `json:"start_page_number"`
	EndBlockIndex   int64 `json:"end_block_index"`
	StartBlockIndex int64 `json:"start_block_index"`
	// This field is from variant [CitationsWebSearchResultLocation].
	EncryptedIndex string `json:"encrypted_index"`
	Title          string `json:"title"`
	// This field is from variant [CitationsWebSearchResultLocation].
	URL string `json:"url"`
	// This field is from variant [CitationsSearchResultLocation].
	SearchResultIndex int64 `json:"search_result_index"`
	// This field is from variant [CitationsSearchResultLocation].
	Source string `json:"source"`
	JSON   struct {
		CitedText         respjson.Field
		DocumentIndex     respjson.Field
		DocumentTitle     respjson.Field
		EndCharIndex      respjson.Field
		FileID            respjson.Field
		StartCharIndex    respjson.Field
		Type              respjson.Field
		EndPageNumber     respjson.Field
		StartPageNumber   respjson.Field
		EndBlockIndex     respjson.Field
		StartBlockIndex   respjson.Field
		EncryptedIndex    respjson.Field
		Title             respjson.Field
		URL               respjson.Field
		SearchResultIndex respjson.Field
		Source            respjson.Field
		raw               string
	} `json:"-"`
}

// anyTextCitation is implemented by each variant of [TextCitationUnion] to add
// type safety for the return type of [TextCitationUnion.AsAny]
type anyTextCitation interface {
	implTextCitationUnion()
	toParamUnion() TextCitationParamUnion
}

func (CitationCharLocation) implTextCitationUnion()             {}
func (CitationPageLocation) implTextCitationUnion()             {}
func (CitationContentBlockLocation) implTextCitationUnion()     {}
func (CitationsWebSearchResultLocation) implTextCitationUnion() {}
func (CitationsSearchResultLocation) implTextCitationUnion()    {}

// Use the following switch statement to find the correct variant
//
//	switch variant := TextCitationUnion.AsAny().(type) {
//	case anthropic.CitationCharLocation:
//	case anthropic.CitationPageLocation:
//	case anthropic.CitationContentBlockLocation:
//	case anthropic.CitationsWebSearchResultLocation:
//	case anthropic.CitationsSearchResultLocation:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u TextCitationUnion) AsAny() anyTextCitation {
	switch u.Type {
	case "char_location":
		return u.AsCharLocation()
	case "page_location":
		return u.AsPageLocation()
	case "content_block_location":
		return u.AsContentBlockLocation()
	case "web_search_result_location":
		return u.AsWebSearchResultLocation()
	case "search_result_location":
		return u.AsSearchResultLocation()
	}
	return nil
}

func (u TextCitationUnion) AsCharLocation() (v CitationCharLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextCitationUnion) AsPageLocation() (v CitationPageLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextCitationUnion) AsContentBlockLocation() (v CitationContentBlockLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextCitationUnion) AsWebSearchResultLocation() (v CitationsWebSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextCitationUnion) AsSearchResultLocation() (v CitationsSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u TextCitationUnion) RawJSON() string { return u.JSON.raw }

func (r *TextCitationUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type TextCitationParamUnion struct {
	OfCharLocation            *CitationCharLocationParam            `json:",omitzero,inline"`
	OfPageLocation            *CitationPageLocationParam            `json:",omitzero,inline"`
	OfContentBlockLocation    *CitationContentBlockLocationParam    `json:",omitzero,inline"`
	OfWebSearchResultLocation *CitationWebSearchResultLocationParam `json:",omitzero,inline"`
	OfSearchResultLocation    *CitationSearchResultLocationParam    `json:",omitzero,inline"`
	paramUnion
}

func (u TextCitationParamUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfCharLocation,
		u.OfPageLocation,
		u.OfContentBlockLocation,
		u.OfWebSearchResultLocation,
		u.OfSearchResultLocation)
}
func (u *TextCitationParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *TextCitationParamUnion) asAny() any {
	if !param.IsOmitted(u.OfCharLocation) {
		return u.OfCharLocation
	} else if !param.IsOmitted(u.OfPageLocation) {
		return u.OfPageLocation
	} else if !param.IsOmitted(u.OfContentBlockLocation) {
		return u.OfContentBlockLocation
	} else if !param.IsOmitted(u.OfWebSearchResultLocation) {
		return u.OfWebSearchResultLocation
	} else if !param.IsOmitted(u.OfSearchResultLocation) {
		return u.OfSearchResultLocation
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetEndCharIndex() *int64 {
	if vt := u.OfCharLocation; vt != nil {
		return &vt.EndCharIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetStartCharIndex() *int64 {
	if vt := u.OfCharLocation; vt != nil {
		return &vt.StartCharIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetEndPageNumber() *int64 {
	if vt := u.OfPageLocation; vt != nil {
		return &vt.EndPageNumber
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetStartPageNumber() *int64 {
	if vt := u.OfPageLocation; vt != nil {
		return &vt.StartPageNumber
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetEncryptedIndex() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil {
		return &vt.EncryptedIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetURL() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetSearchResultIndex() *int64 {
	if vt := u.OfSearchResultLocation; vt != nil {
		return &vt.SearchResultIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetSource() *string {
	if vt := u.OfSearchResultLocation; vt != nil {
		return &vt.Source
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetCitedText() *string {
	if vt := u.OfCharLocation; vt != nil {
		return (*string)(&vt.CitedText)
	} else if vt := u.OfPageLocation; vt != nil {
		return (*string)(&vt.CitedText)
	} else if vt := u.OfContentBlockLocation; vt != nil {
		return (*string)(&vt.CitedText)
	} else if vt := u.OfWebSearchResultLocation; vt != nil {
		return (*string)(&vt.CitedText)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*string)(&vt.CitedText)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetDocumentIndex() *int64 {
	if vt := u.OfCharLocation; vt != nil {
		return (*int64)(&vt.DocumentIndex)
	} else if vt := u.OfPageLocation; vt != nil {
		return (*int64)(&vt.DocumentIndex)
	} else if vt := u.OfContentBlockLocation; vt != nil {
		return (*int64)(&vt.DocumentIndex)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetDocumentTitle() *string {
	if vt := u.OfCharLocation; vt != nil && vt.DocumentTitle.Valid() {
		return &vt.DocumentTitle.Value
	} else if vt := u.OfPageLocation; vt != nil && vt.DocumentTitle.Valid() {
		return &vt.DocumentTitle.Value
	} else if vt := u.OfContentBlockLocation; vt != nil && vt.DocumentTitle.Valid() {
		return &vt.DocumentTitle.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetType() *string {
	if vt := u.OfCharLocation; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfPageLocation; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContentBlockLocation; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchResultLocation; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetEndBlockIndex() *int64 {
	if vt := u.OfContentBlockLocation; vt != nil {
		return (*int64)(&vt.EndBlockIndex)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*int64)(&vt.EndBlockIndex)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetStartBlockIndex() *int64 {
	if vt := u.OfContentBlockLocation; vt != nil {
		return (*int64)(&vt.StartBlockIndex)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*int64)(&vt.StartBlockIndex)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextCitationParamUnion) GetTitle() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	} else if vt := u.OfSearchResultLocation; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	}
	return nil
}

type TextDelta struct {
	Text string             `json:"text,required"`
	Type constant.TextDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextDelta) RawJSON() string { return r.JSON.raw }
func (r *TextDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionCreateResultBlock struct {
	IsFileUpdate bool                                         `json:"is_file_update,required"`
	Type         constant.TextEditorCodeExecutionCreateResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		IsFileUpdate respjson.Field
		Type         respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextEditorCodeExecutionCreateResultBlock) RawJSON() string { return r.JSON.raw }
func (r *TextEditorCodeExecutionCreateResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties IsFileUpdate, Type are required.
type TextEditorCodeExecutionCreateResultBlockParam struct {
	IsFileUpdate bool `json:"is_file_update,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_create_result".
	Type constant.TextEditorCodeExecutionCreateResult `json:"type,required"`
	paramObj
}

func (r TextEditorCodeExecutionCreateResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow TextEditorCodeExecutionCreateResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextEditorCodeExecutionCreateResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionStrReplaceResultBlock struct {
	Lines    []string                                         `json:"lines,required"`
	NewLines int64                                            `json:"new_lines,required"`
	NewStart int64                                            `json:"new_start,required"`
	OldLines int64                                            `json:"old_lines,required"`
	OldStart int64                                            `json:"old_start,required"`
	Type     constant.TextEditorCodeExecutionStrReplaceResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Lines       respjson.Field
		NewLines    respjson.Field
		NewStart    respjson.Field
		OldLines    respjson.Field
		OldStart    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextEditorCodeExecutionStrReplaceResultBlock) RawJSON() string { return r.JSON.raw }
func (r *TextEditorCodeExecutionStrReplaceResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Type is required.
type TextEditorCodeExecutionStrReplaceResultBlockParam struct {
	NewLines param.Opt[int64] `json:"new_lines,omitzero"`
	NewStart param.Opt[int64] `json:"new_start,omitzero"`
	OldLines param.Opt[int64] `json:"old_lines,omitzero"`
	OldStart param.Opt[int64] `json:"old_start,omitzero"`
	Lines    []string         `json:"lines,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_str_replace_result".
	Type constant.TextEditorCodeExecutionStrReplaceResult `json:"type,required"`
	paramObj
}

func (r TextEditorCodeExecutionStrReplaceResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow TextEditorCodeExecutionStrReplaceResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextEditorCodeExecutionStrReplaceResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionToolResultBlock struct {
	Content   TextEditorCodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                             `json:"tool_use_id,required"`
	Type      constant.TextEditorCodeExecutionToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextEditorCodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *TextEditorCodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// TextEditorCodeExecutionToolResultBlockContentUnion contains all possible
// properties and values from [TextEditorCodeExecutionToolResultError],
// [TextEditorCodeExecutionViewResultBlock],
// [TextEditorCodeExecutionCreateResultBlock],
// [TextEditorCodeExecutionStrReplaceResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type TextEditorCodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [TextEditorCodeExecutionToolResultError].
	ErrorCode TextEditorCodeExecutionToolResultErrorCode `json:"error_code"`
	// This field is from variant [TextEditorCodeExecutionToolResultError].
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
	// This field is from variant [TextEditorCodeExecutionViewResultBlock].
	Content string `json:"content"`
	// This field is from variant [TextEditorCodeExecutionViewResultBlock].
	FileType TextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant [TextEditorCodeExecutionViewResultBlock].
	NumLines int64 `json:"num_lines"`
	// This field is from variant [TextEditorCodeExecutionViewResultBlock].
	StartLine int64 `json:"start_line"`
	// This field is from variant [TextEditorCodeExecutionViewResultBlock].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant [TextEditorCodeExecutionCreateResultBlock].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant [TextEditorCodeExecutionStrReplaceResultBlock].
	Lines []string `json:"lines"`
	// This field is from variant [TextEditorCodeExecutionStrReplaceResultBlock].
	NewLines int64 `json:"new_lines"`
	// This field is from variant [TextEditorCodeExecutionStrReplaceResultBlock].
	NewStart int64 `json:"new_start"`
	// This field is from variant [TextEditorCodeExecutionStrReplaceResultBlock].
	OldLines int64 `json:"old_lines"`
	// This field is from variant [TextEditorCodeExecutionStrReplaceResultBlock].
	OldStart int64 `json:"old_start"`
	JSON     struct {
		ErrorCode    respjson.Field
		ErrorMessage respjson.Field
		Type         respjson.Field
		Content      respjson.Field
		FileType     respjson.Field
		NumLines     respjson.Field
		StartLine    respjson.Field
		TotalLines   respjson.Field
		IsFileUpdate respjson.Field
		Lines        respjson.Field
		NewLines     respjson.Field
		NewStart     respjson.Field
		OldLines     respjson.Field
		OldStart     respjson.Field
		raw          string
	} `json:"-"`
}

func (u TextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionToolResultError() (v TextEditorCodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionViewResultBlock() (v TextEditorCodeExecutionViewResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionCreateResultBlock() (v TextEditorCodeExecutionCreateResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u TextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionStrReplaceResultBlock() (v TextEditorCodeExecutionStrReplaceResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u TextEditorCodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *TextEditorCodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type TextEditorCodeExecutionToolResultBlockParam struct {
	Content   TextEditorCodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                                  `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_tool_result".
	Type constant.TextEditorCodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r TextEditorCodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow TextEditorCodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextEditorCodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type TextEditorCodeExecutionToolResultBlockParamContentUnion struct {
	OfRequestTextEditorCodeExecutionToolResultError       *TextEditorCodeExecutionToolResultErrorParam       `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionViewResultBlock       *TextEditorCodeExecutionViewResultBlockParam       `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionCreateResultBlock     *TextEditorCodeExecutionCreateResultBlockParam     `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionStrReplaceResultBlock *TextEditorCodeExecutionStrReplaceResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestTextEditorCodeExecutionToolResultError, u.OfRequestTextEditorCodeExecutionViewResultBlock, u.OfRequestTextEditorCodeExecutionCreateResultBlock, u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock)
}
func (u *TextEditorCodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *TextEditorCodeExecutionToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestTextEditorCodeExecutionToolResultError) {
		return u.OfRequestTextEditorCodeExecutionToolResultError
	} else if !param.IsOmitted(u.OfRequestTextEditorCodeExecutionViewResultBlock) {
		return u.OfRequestTextEditorCodeExecutionViewResultBlock
	} else if !param.IsOmitted(u.OfRequestTextEditorCodeExecutionCreateResultBlock) {
		return u.OfRequestTextEditorCodeExecutionCreateResultBlock
	} else if !param.IsOmitted(u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock) {
		return u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestTextEditorCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetErrorMessage() *string {
	if vt := u.OfRequestTextEditorCodeExecutionToolResultError; vt != nil && vt.ErrorMessage.Valid() {
		return &vt.ErrorMessage.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetContent() *string {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetFileType() *string {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil {
		return (*string)(&vt.FileType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetNumLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.NumLines.Valid() {
		return &vt.NumLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetStartLine() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.StartLine.Valid() {
		return &vt.StartLine.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetTotalLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.TotalLines.Valid() {
		return &vt.TotalLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetIsFileUpdate() *bool {
	if vt := u.OfRequestTextEditorCodeExecutionCreateResultBlock; vt != nil {
		return &vt.IsFileUpdate
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetLines() []string {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil {
		return vt.Lines
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetNewLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.NewLines.Valid() {
		return &vt.NewLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetNewStart() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.NewStart.Valid() {
		return &vt.NewStart.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetOldLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.OldLines.Valid() {
		return &vt.OldLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetOldStart() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.OldStart.Valid() {
		return &vt.OldStart.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u TextEditorCodeExecutionToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestTextEditorCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestTextEditorCodeExecutionCreateResultBlock; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type TextEditorCodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "file_not_found".
	ErrorCode    TextEditorCodeExecutionToolResultErrorCode      `json:"error_code,required"`
	ErrorMessage string                                          `json:"error_message,required"`
	Type         constant.TextEditorCodeExecutionToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode    respjson.Field
		ErrorMessage respjson.Field
		Type         respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextEditorCodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *TextEditorCodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionToolResultErrorCode string

const (
	TextEditorCodeExecutionToolResultErrorCodeInvalidToolInput      TextEditorCodeExecutionToolResultErrorCode = "invalid_tool_input"
	TextEditorCodeExecutionToolResultErrorCodeUnavailable           TextEditorCodeExecutionToolResultErrorCode = "unavailable"
	TextEditorCodeExecutionToolResultErrorCodeTooManyRequests       TextEditorCodeExecutionToolResultErrorCode = "too_many_requests"
	TextEditorCodeExecutionToolResultErrorCodeExecutionTimeExceeded TextEditorCodeExecutionToolResultErrorCode = "execution_time_exceeded"
	TextEditorCodeExecutionToolResultErrorCodeFileNotFound          TextEditorCodeExecutionToolResultErrorCode = "file_not_found"
)

// The properties ErrorCode, Type are required.
type TextEditorCodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "file_not_found".
	ErrorCode    TextEditorCodeExecutionToolResultErrorCode `json:"error_code,omitzero,required"`
	ErrorMessage param.Opt[string]                          `json:"error_message,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_tool_result_error".
	Type constant.TextEditorCodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r TextEditorCodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow TextEditorCodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextEditorCodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionViewResultBlock struct {
	Content string `json:"content,required"`
	// Any of "text", "image", "pdf".
	FileType   TextEditorCodeExecutionViewResultBlockFileType `json:"file_type,required"`
	NumLines   int64                                          `json:"num_lines,required"`
	StartLine  int64                                          `json:"start_line,required"`
	TotalLines int64                                          `json:"total_lines,required"`
	Type       constant.TextEditorCodeExecutionViewResult     `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		FileType    respjson.Field
		NumLines    respjson.Field
		StartLine   respjson.Field
		TotalLines  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextEditorCodeExecutionViewResultBlock) RawJSON() string { return r.JSON.raw }
func (r *TextEditorCodeExecutionViewResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionViewResultBlockFileType string

const (
	TextEditorCodeExecutionViewResultBlockFileTypeText  TextEditorCodeExecutionViewResultBlockFileType = "text"
	TextEditorCodeExecutionViewResultBlockFileTypeImage TextEditorCodeExecutionViewResultBlockFileType = "image"
	TextEditorCodeExecutionViewResultBlockFileTypePDF   TextEditorCodeExecutionViewResultBlockFileType = "pdf"
)

// The properties Content, FileType, Type are required.
type TextEditorCodeExecutionViewResultBlockParam struct {
	Content string `json:"content,required"`
	// Any of "text", "image", "pdf".
	FileType   TextEditorCodeExecutionViewResultBlockParamFileType `json:"file_type,omitzero,required"`
	NumLines   param.Opt[int64]                                    `json:"num_lines,omitzero"`
	StartLine  param.Opt[int64]                                    `json:"start_line,omitzero"`
	TotalLines param.Opt[int64]                                    `json:"total_lines,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_view_result".
	Type constant.TextEditorCodeExecutionViewResult `json:"type,required"`
	paramObj
}

func (r TextEditorCodeExecutionViewResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow TextEditorCodeExecutionViewResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TextEditorCodeExecutionViewResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TextEditorCodeExecutionViewResultBlockParamFileType string

const (
	TextEditorCodeExecutionViewResultBlockParamFileTypeText  TextEditorCodeExecutionViewResultBlockParamFileType = "text"
	TextEditorCodeExecutionViewResultBlockParamFileTypeImage TextEditorCodeExecutionViewResultBlockParamFileType = "image"
	TextEditorCodeExecutionViewResultBlockParamFileTypePDF   TextEditorCodeExecutionViewResultBlockParamFileType = "pdf"
)

type ThinkingBlock struct {
	Signature string            `json:"signature,required"`
	Thinking  string            `json:"thinking,required"`
	Type      constant.Thinking `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Signature   respjson.Field
		Thinking    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ThinkingBlock) RawJSON() string { return r.JSON.raw }
func (r *ThinkingBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Signature, Thinking, Type are required.
type ThinkingBlockParam struct {
	Signature string `json:"signature,required"`
	Thinking  string `json:"thinking,required"`
	// This field can be elided, and will marshal its zero value as "thinking".
	Type constant.Thinking `json:"type,required"`
	paramObj
}

func (r ThinkingBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ThinkingBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ThinkingBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewThinkingConfigAdaptiveParam() ThinkingConfigAdaptiveParam {
	return ThinkingConfigAdaptiveParam{
		Type: "adaptive",
	}
}

// This struct has a constant value, construct it with
// [NewThinkingConfigAdaptiveParam].
type ThinkingConfigAdaptiveParam struct {
	Type constant.Adaptive `json:"type,required"`
	paramObj
}

func (r ThinkingConfigAdaptiveParam) MarshalJSON() (data []byte, err error) {
	type shadow ThinkingConfigAdaptiveParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ThinkingConfigAdaptiveParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewThinkingConfigDisabledParam() ThinkingConfigDisabledParam {
	return ThinkingConfigDisabledParam{
		Type: "disabled",
	}
}

// This struct has a constant value, construct it with
// [NewThinkingConfigDisabledParam].
type ThinkingConfigDisabledParam struct {
	Type constant.Disabled `json:"type,required"`
	paramObj
}

func (r ThinkingConfigDisabledParam) MarshalJSON() (data []byte, err error) {
	type shadow ThinkingConfigDisabledParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ThinkingConfigDisabledParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties BudgetTokens, Type are required.
type ThinkingConfigEnabledParam struct {
	// Determines how many tokens Claude can use for its internal reasoning process.
	// Larger budgets can enable more thorough analysis for complex problems, improving
	// response quality.
	//
	// Must be ≥1024 and less than `max_tokens`.
	//
	// See
	// [extended thinking](https://docs.claude.com/en/docs/build-with-claude/extended-thinking)
	// for details.
	BudgetTokens int64 `json:"budget_tokens,required"`
	// This field can be elided, and will marshal its zero value as "enabled".
	Type constant.Enabled `json:"type,required"`
	paramObj
}

func (r ThinkingConfigEnabledParam) MarshalJSON() (data []byte, err error) {
	type shadow ThinkingConfigEnabledParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ThinkingConfigEnabledParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func ThinkingConfigParamOfEnabled(budgetTokens int64) ThinkingConfigParamUnion {
	var enabled ThinkingConfigEnabledParam
	enabled.BudgetTokens = budgetTokens
	return ThinkingConfigParamUnion{OfEnabled: &enabled}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ThinkingConfigParamUnion struct {
	OfEnabled  *ThinkingConfigEnabledParam  `json:",omitzero,inline"`
	OfDisabled *ThinkingConfigDisabledParam `json:",omitzero,inline"`
	OfAdaptive *ThinkingConfigAdaptiveParam `json:",omitzero,inline"`
	paramUnion
}

func (u ThinkingConfigParamUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfEnabled, u.OfDisabled, u.OfAdaptive)
}
func (u *ThinkingConfigParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ThinkingConfigParamUnion) asAny() any {
	if !param.IsOmitted(u.OfEnabled) {
		return u.OfEnabled
	} else if !param.IsOmitted(u.OfDisabled) {
		return u.OfDisabled
	} else if !param.IsOmitted(u.OfAdaptive) {
		return u.OfAdaptive
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ThinkingConfigParamUnion) GetBudgetTokens() *int64 {
	if vt := u.OfEnabled; vt != nil {
		return &vt.BudgetTokens
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ThinkingConfigParamUnion) GetType() *string {
	if vt := u.OfEnabled; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfDisabled; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAdaptive; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}


type ThinkingDelta struct {
	Thinking string                 `json:"thinking,required"`
	Type     constant.ThinkingDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Thinking    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ThinkingDelta) RawJSON() string { return r.JSON.raw }
func (r *ThinkingDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties InputSchema, Name are required.
type ToolParam struct {
	// [JSON schema](https://json-schema.org/draft/2020-12) for this tool's input.
	//
	// This defines the shape of the `input` that your tool accepts and that the model
	// will produce.
	InputSchema ToolInputSchemaParam `json:"input_schema,omitzero,required"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	Name string `json:"name,required"`
	// Enable eager input streaming for this tool. When true, tool input parameters
	// will be streamed incrementally as they are generated, and types will be inferred
	// on-the-fly rather than buffering the full JSON output. When false, streaming is
	// disabled for this tool even if the fine-grained-tool-streaming beta is active.
	// When null (default), uses the default behavior based on beta headers.
	EagerInputStreaming param.Opt[bool] `json:"eager_input_streaming,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// Description of what this tool does.
	//
	// Tool descriptions should be as detailed as possible. The more information that
	// the model has about what the tool is and how to use it, the better it will
	// perform. You can use natural language descriptions to reinforce important
	// aspects of the tool input JSON schema.
	Description param.Opt[string] `json:"description,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "custom".
	Type ToolType `json:"type,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	paramObj
}

func (r ToolParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// [JSON schema](https://json-schema.org/draft/2020-12) for this tool's input.
//
// This defines the shape of the `input` that your tool accepts and that the model
// will produce.
//
// The property Type is required.
type ToolInputSchemaParam struct {
	Properties any      `json:"properties,omitzero"`
	Required   []string `json:"required,omitzero"`
	// This field can be elided, and will marshal its zero value as "object".
	Type        constant.Object `json:"type,required"`
	ExtraFields map[string]any  `json:"-"`
	paramObj
}

func (r ToolInputSchemaParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolInputSchemaParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *ToolInputSchemaParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolType string

const (
	ToolTypeCustom ToolType = "custom"
)

// The properties Name, Type are required.
type ToolBash20250124Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "bash".
	Name constant.Bash `json:"name,required"`
	// This field can be elided, and will marshal its zero value as "bash_20250124".
	Type constant.Bash20250124 `json:"type,required"`
	paramObj
}

func (r ToolBash20250124Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolBash20250124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolBash20250124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func ToolChoiceParamOfTool(name string) ToolChoiceUnionParam {
	var tool ToolChoiceToolParam
	tool.Name = name
	return ToolChoiceUnionParam{OfTool: &tool}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ToolChoiceUnionParam struct {
	OfAuto *ToolChoiceAutoParam `json:",omitzero,inline"`
	OfAny  *ToolChoiceAnyParam  `json:",omitzero,inline"`
	OfTool *ToolChoiceToolParam `json:",omitzero,inline"`
	OfNone *ToolChoiceNoneParam `json:",omitzero,inline"`
	paramUnion
}

func (u ToolChoiceUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfAuto, u.OfAny, u.OfTool, u.OfNone)
}
func (u *ToolChoiceUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ToolChoiceUnionParam) asAny() any {
	if !param.IsOmitted(u.OfAuto) {
		return u.OfAuto
	} else if !param.IsOmitted(u.OfAny) {
		return u.OfAny
	} else if !param.IsOmitted(u.OfTool) {
		return u.OfTool
	} else if !param.IsOmitted(u.OfNone) {
		return u.OfNone
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolChoiceUnionParam) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return &vt.Name
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolChoiceUnionParam) GetType() *string {
	if vt := u.OfAuto; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAny; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfNone; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolChoiceUnionParam) GetDisableParallelToolUse() *bool {
	if vt := u.OfAuto; vt != nil && vt.DisableParallelToolUse.Valid() {
		return &vt.DisableParallelToolUse.Value
	} else if vt := u.OfAny; vt != nil && vt.DisableParallelToolUse.Valid() {
		return &vt.DisableParallelToolUse.Value
	} else if vt := u.OfTool; vt != nil && vt.DisableParallelToolUse.Valid() {
		return &vt.DisableParallelToolUse.Value
	}
	return nil
}

// The model will use any available tools.
//
// The property Type is required.
type ToolChoiceAnyParam struct {
	// Whether to disable parallel tool use.
	//
	// Defaults to `false`. If set to `true`, the model will output exactly one tool
	// use.
	DisableParallelToolUse param.Opt[bool] `json:"disable_parallel_tool_use,omitzero"`
	// This field can be elided, and will marshal its zero value as "any".
	Type constant.Any `json:"type,required"`
	paramObj
}

func (r ToolChoiceAnyParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolChoiceAnyParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolChoiceAnyParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The model will automatically decide whether to use tools.
//
// The property Type is required.
type ToolChoiceAutoParam struct {
	// Whether to disable parallel tool use.
	//
	// Defaults to `false`. If set to `true`, the model will output at most one tool
	// use.
	DisableParallelToolUse param.Opt[bool] `json:"disable_parallel_tool_use,omitzero"`
	// This field can be elided, and will marshal its zero value as "auto".
	Type constant.Auto `json:"type,required"`
	paramObj
}

func (r ToolChoiceAutoParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolChoiceAutoParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolChoiceAutoParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewToolChoiceNoneParam() ToolChoiceNoneParam {
	return ToolChoiceNoneParam{
		Type: "none",
	}
}

// The model will not be allowed to use tools.
//
// This struct has a constant value, construct it with [NewToolChoiceNoneParam].
type ToolChoiceNoneParam struct {
	Type constant.None `json:"type,required"`
	paramObj
}

func (r ToolChoiceNoneParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolChoiceNoneParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolChoiceNoneParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The model will use the specified tool with `tool_choice.name`.
//
// The properties Name, Type are required.
type ToolChoiceToolParam struct {
	// The name of the tool to use.
	Name string `json:"name,required"`
	// Whether to disable parallel tool use.
	//
	// Defaults to `false`. If set to `true`, the model will output exactly one tool
	// use.
	DisableParallelToolUse param.Opt[bool] `json:"disable_parallel_tool_use,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool".
	Type constant.Tool `json:"type,required"`
	paramObj
}

func (r ToolChoiceToolParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolChoiceToolParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolChoiceToolParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolReferenceBlock struct {
	ToolName string                 `json:"tool_name,required"`
	Type     constant.ToolReference `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ToolName    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ToolReferenceBlock) RawJSON() string { return r.JSON.raw }
func (r *ToolReferenceBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Tool reference block that can be included in tool_result content.
//
// The properties ToolName, Type are required.
type ToolReferenceBlockParam struct {
	ToolName string `json:"tool_name,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_reference".
	Type constant.ToolReference `json:"type,required"`
	paramObj
}

func (r ToolReferenceBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolReferenceBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolReferenceBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ToolUseID, Type are required.
type ToolResultBlockParam struct {
	ToolUseID string          `json:"tool_use_id,required"`
	IsError   param.Opt[bool] `json:"is_error,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam         `json:"cache_control,omitzero"`
	Content      []ToolResultBlockParamContentUnion `json:"content,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_result".
	Type constant.ToolResult `json:"type,required"`
	paramObj
}

func (r ToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ToolResultBlockParamContentUnion struct {
	OfText          *TextBlockParam          `json:",omitzero,inline"`
	OfImage         *ImageBlockParam         `json:",omitzero,inline"`
	OfSearchResult  *SearchResultBlockParam  `json:",omitzero,inline"`
	OfDocument      *DocumentBlockParam      `json:",omitzero,inline"`
	OfToolReference *ToolReferenceBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u ToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfText,
		u.OfImage,
		u.OfSearchResult,
		u.OfDocument,
		u.OfToolReference)
}
func (u *ToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfImage) {
		return u.OfImage
	} else if !param.IsOmitted(u.OfSearchResult) {
		return u.OfSearchResult
	} else if !param.IsOmitted(u.OfDocument) {
		return u.OfDocument
	} else if !param.IsOmitted(u.OfToolReference) {
		return u.OfToolReference
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetText() *string {
	if vt := u.OfText; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetContent() []TextBlockParam {
	if vt := u.OfSearchResult; vt != nil {
		return vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetContext() *string {
	if vt := u.OfDocument; vt != nil && vt.Context.Valid() {
		return &vt.Context.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetToolName() *string {
	if vt := u.OfToolReference; vt != nil {
		return &vt.ToolName
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfImage; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfDocument; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolReference; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolResultBlockParamContentUnion) GetTitle() *string {
	if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Title)
	} else if vt := u.OfDocument; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u ToolResultBlockParamContentUnion) GetCacheControl() *CacheControlEphemeralParam {
	if vt := u.OfText; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfImage; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfSearchResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfDocument; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolReference; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ToolResultBlockParamContentUnion) GetCitations() (res toolResultBlockParamContentUnionCitations) {
	if vt := u.OfText; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfDocument; vt != nil {
		res.any = &vt.Citations
	}
	return
}

// Can have the runtime types [*[]TextCitationParamUnion], [*CitationsConfigParam]
type toolResultBlockParamContentUnionCitations struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.TextCitationParamUnion:
//	case *anthropic.CitationsConfigParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u toolResultBlockParamContentUnionCitations) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionCitations) GetEnabled() *bool {
	switch vt := u.any.(type) {
	case *CitationsConfigParam:
		return paramutil.AddrIfPresent(vt.Enabled)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ToolResultBlockParamContentUnion) GetSource() (res toolResultBlockParamContentUnionSource) {
	if vt := u.OfImage; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Source
	} else if vt := u.OfDocument; vt != nil {
		res.any = vt.Source.asAny()
	}
	return
}

// Can have the runtime types [*Base64ImageSourceParam], [*URLImageSourceParam],
// [*string], [*Base64PDFSourceParam], [*PlainTextSourceParam],
// [*ContentBlockSourceParam], [*URLPDFSourceParam]
type toolResultBlockParamContentUnionSource struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.Base64ImageSourceParam:
//	case *anthropic.URLImageSourceParam:
//	case *string:
//	case *anthropic.Base64PDFSourceParam:
//	case *anthropic.PlainTextSourceParam:
//	case *anthropic.ContentBlockSourceParam:
//	case *anthropic.URLPDFSourceParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u toolResultBlockParamContentUnionSource) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionSource) GetContent() *ContentBlockSourceContentUnionParam {
	switch vt := u.any.(type) {
	case *DocumentBlockParamSourceUnion:
		return vt.GetContent()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionSource) GetData() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetData()
	case *DocumentBlockParamSourceUnion:
		return vt.GetData()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionSource) GetMediaType() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetMediaType()
	case *DocumentBlockParamSourceUnion:
		return vt.GetMediaType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionSource) GetType() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetType()
	case *DocumentBlockParamSourceUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u toolResultBlockParamContentUnionSource) GetURL() *string {
	switch vt := u.any.(type) {
	case *ImageBlockParamSourceUnion:
		return vt.GetURL()
	case *DocumentBlockParamSourceUnion:
		return vt.GetURL()
	}
	return nil
}


// The properties Name, Type are required.
type ToolSearchToolBm25_20251119Param struct {
	// Any of "tool_search_tool_bm25_20251119", "tool_search_tool_bm25".
	Type ToolSearchToolBm25_20251119Type `json:"type,omitzero,required"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_bm25".
	Name constant.ToolSearchToolBm25 `json:"name,required"`
	paramObj
}

func (r ToolSearchToolBm25_20251119Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolSearchToolBm25_20251119Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolSearchToolBm25_20251119Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolSearchToolBm25_20251119Type string

const (
	ToolSearchToolBm25_20251119TypeToolSearchToolBm25_20251119 ToolSearchToolBm25_20251119Type = "tool_search_tool_bm25_20251119"
	ToolSearchToolBm25_20251119TypeToolSearchToolBm25          ToolSearchToolBm25_20251119Type = "tool_search_tool_bm25"
)

// The properties Name, Type are required.
type ToolSearchToolRegex20251119Param struct {
	// Any of "tool_search_tool_regex_20251119", "tool_search_tool_regex".
	Type ToolSearchToolRegex20251119Type `json:"type,omitzero,required"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_regex".
	Name constant.ToolSearchToolRegex `json:"name,required"`
	paramObj
}

func (r ToolSearchToolRegex20251119Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolSearchToolRegex20251119Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolSearchToolRegex20251119Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolSearchToolRegex20251119Type string

const (
	ToolSearchToolRegex20251119TypeToolSearchToolRegex20251119 ToolSearchToolRegex20251119Type = "tool_search_tool_regex_20251119"
	ToolSearchToolRegex20251119TypeToolSearchToolRegex         ToolSearchToolRegex20251119Type = "tool_search_tool_regex"
)

type ToolSearchToolResultBlock struct {
	Content   ToolSearchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                `json:"tool_use_id,required"`
	Type      constant.ToolSearchToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ToolSearchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *ToolSearchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToolSearchToolResultBlockContentUnion contains all possible properties and
// values from [ToolSearchToolResultError], [ToolSearchToolSearchResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ToolSearchToolResultBlockContentUnion struct {
	// This field is from variant [ToolSearchToolResultError].
	ErrorCode ToolSearchToolResultErrorCode `json:"error_code"`
	// This field is from variant [ToolSearchToolResultError].
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
	// This field is from variant [ToolSearchToolSearchResultBlock].
	ToolReferences []ToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		ErrorCode      respjson.Field
		ErrorMessage   respjson.Field
		Type           respjson.Field
		ToolReferences respjson.Field
		raw            string
	} `json:"-"`
}

func (u ToolSearchToolResultBlockContentUnion) AsResponseToolSearchToolResultError() (v ToolSearchToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ToolSearchToolResultBlockContentUnion) AsResponseToolSearchToolSearchResultBlock() (v ToolSearchToolSearchResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ToolSearchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *ToolSearchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type ToolSearchToolResultBlockParam struct {
	Content   ToolSearchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                     `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_result".
	Type constant.ToolSearchToolResult `json:"type,required"`
	paramObj
}

func (r ToolSearchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolSearchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolSearchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ToolSearchToolResultBlockParamContentUnion struct {
	OfRequestToolSearchToolResultError       *ToolSearchToolResultErrorParam       `json:",omitzero,inline"`
	OfRequestToolSearchToolSearchResultBlock *ToolSearchToolSearchResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u ToolSearchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestToolSearchToolResultError, u.OfRequestToolSearchToolSearchResultBlock)
}
func (u *ToolSearchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ToolSearchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestToolSearchToolResultError) {
		return u.OfRequestToolSearchToolResultError
	} else if !param.IsOmitted(u.OfRequestToolSearchToolSearchResultBlock) {
		return u.OfRequestToolSearchToolSearchResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolSearchToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestToolSearchToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolSearchToolResultBlockParamContentUnion) GetToolReferences() []ToolReferenceBlockParam {
	if vt := u.OfRequestToolSearchToolSearchResultBlock; vt != nil {
		return vt.ToolReferences
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolSearchToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestToolSearchToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestToolSearchToolSearchResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type ToolSearchToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode    ToolSearchToolResultErrorCode      `json:"error_code,required"`
	ErrorMessage string                             `json:"error_message,required"`
	Type         constant.ToolSearchToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode    respjson.Field
		ErrorMessage respjson.Field
		Type         respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ToolSearchToolResultError) RawJSON() string { return r.JSON.raw }
func (r *ToolSearchToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolSearchToolResultErrorCode string

const (
	ToolSearchToolResultErrorCodeInvalidToolInput      ToolSearchToolResultErrorCode = "invalid_tool_input"
	ToolSearchToolResultErrorCodeUnavailable           ToolSearchToolResultErrorCode = "unavailable"
	ToolSearchToolResultErrorCodeTooManyRequests       ToolSearchToolResultErrorCode = "too_many_requests"
	ToolSearchToolResultErrorCodeExecutionTimeExceeded ToolSearchToolResultErrorCode = "execution_time_exceeded"
)

// The properties ErrorCode, Type are required.
type ToolSearchToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode ToolSearchToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_result_error".
	Type constant.ToolSearchToolResultError `json:"type,required"`
	paramObj
}

func (r ToolSearchToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolSearchToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolSearchToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ToolSearchToolSearchResultBlock struct {
	ToolReferences []ToolReferenceBlock                `json:"tool_references,required"`
	Type           constant.ToolSearchToolSearchResult `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ToolReferences respjson.Field
		Type           respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ToolSearchToolSearchResultBlock) RawJSON() string { return r.JSON.raw }
func (r *ToolSearchToolSearchResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ToolReferences, Type are required.
type ToolSearchToolSearchResultBlockParam struct {
	ToolReferences []ToolReferenceBlockParam `json:"tool_references,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_search_result".
	Type constant.ToolSearchToolSearchResult `json:"type,required"`
	paramObj
}

func (r ToolSearchToolSearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolSearchToolSearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolSearchToolSearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type ToolTextEditor20250124Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "str_replace_editor".
	Name constant.StrReplaceEditor `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_20250124".
	Type constant.TextEditor20250124 `json:"type,required"`
	paramObj
}

func (r ToolTextEditor20250124Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolTextEditor20250124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolTextEditor20250124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type ToolTextEditor20250429Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "str_replace_based_edit_tool".
	Name constant.StrReplaceBasedEditTool `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_20250429".
	Type constant.TextEditor20250429 `json:"type,required"`
	paramObj
}

func (r ToolTextEditor20250429Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolTextEditor20250429Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolTextEditor20250429Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type ToolTextEditor20250728Param struct {
	// Maximum number of characters to display when viewing a file. If not specified,
	// defaults to displaying the full file.
	MaxCharacters param.Opt[int64] `json:"max_characters,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  CacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any           `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "str_replace_based_edit_tool".
	Name constant.StrReplaceBasedEditTool `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_20250728".
	Type constant.TextEditor20250728 `json:"type,required"`
	paramObj
}

func (r ToolTextEditor20250728Param) MarshalJSON() (data []byte, err error) {
	type shadow ToolTextEditor20250728Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolTextEditor20250728Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func ToolUnionParamOfTool(inputSchema ToolInputSchemaParam, name string) ToolUnionParam {
	var variant ToolParam
	variant.InputSchema = inputSchema
	variant.Name = name
	return ToolUnionParam{OfTool: &variant}
}

func ToolUnionParamOfToolSearchToolBm25_20251119(type_ ToolSearchToolBm25_20251119Type) ToolUnionParam {
	var variant ToolSearchToolBm25_20251119Param
	variant.Type = type_
	return ToolUnionParam{OfToolSearchToolBm25_20251119: &variant}
}

func ToolUnionParamOfToolSearchToolRegex20251119(type_ ToolSearchToolRegex20251119Type) ToolUnionParam {
	var variant ToolSearchToolRegex20251119Param
	variant.Type = type_
	return ToolUnionParam{OfToolSearchToolRegex20251119: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ToolUnionParam struct {
	OfTool                        *ToolParam                        `json:",omitzero,inline"`
	OfBashTool20250124            *ToolBash20250124Param            `json:",omitzero,inline"`
	OfCodeExecutionTool20250522   *CodeExecutionTool20250522Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20250825   *CodeExecutionTool20250825Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20260120   *CodeExecutionTool20260120Param   `json:",omitzero,inline"`
	OfMemoryTool20250818          *MemoryTool20250818Param          `json:",omitzero,inline"`
	OfTextEditor20250124          *ToolTextEditor20250124Param      `json:",omitzero,inline"`
	OfTextEditor20250429          *ToolTextEditor20250429Param      `json:",omitzero,inline"`
	OfTextEditor20250728          *ToolTextEditor20250728Param      `json:",omitzero,inline"`
	OfWebSearchTool20250305       *WebSearchTool20250305Param       `json:",omitzero,inline"`
	OfWebFetchTool20250910        *WebFetchTool20250910Param        `json:",omitzero,inline"`
	OfWebSearchTool20260209       *WebSearchTool20260209Param       `json:",omitzero,inline"`
	OfWebFetchTool20260209        *WebFetchTool20260209Param        `json:",omitzero,inline"`
	OfToolSearchToolBm25_20251119 *ToolSearchToolBm25_20251119Param `json:",omitzero,inline"`
	OfToolSearchToolRegex20251119 *ToolSearchToolRegex20251119Param `json:",omitzero,inline"`
	paramUnion
}

func (u ToolUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfTool,
		u.OfBashTool20250124,
		u.OfCodeExecutionTool20250522,
		u.OfCodeExecutionTool20250825,
		u.OfCodeExecutionTool20260120,
		u.OfMemoryTool20250818,
		u.OfTextEditor20250124,
		u.OfTextEditor20250429,
		u.OfTextEditor20250728,
		u.OfWebSearchTool20250305,
		u.OfWebFetchTool20250910,
		u.OfWebSearchTool20260209,
		u.OfWebFetchTool20260209,
		u.OfToolSearchToolBm25_20251119,
		u.OfToolSearchToolRegex20251119)
}
func (u *ToolUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ToolUnionParam) asAny() any {
	if !param.IsOmitted(u.OfTool) {
		return u.OfTool
	} else if !param.IsOmitted(u.OfBashTool20250124) {
		return u.OfBashTool20250124
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250522) {
		return u.OfCodeExecutionTool20250522
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250825) {
		return u.OfCodeExecutionTool20250825
	} else if !param.IsOmitted(u.OfCodeExecutionTool20260120) {
		return u.OfCodeExecutionTool20260120
	} else if !param.IsOmitted(u.OfMemoryTool20250818) {
		return u.OfMemoryTool20250818
	} else if !param.IsOmitted(u.OfTextEditor20250124) {
		return u.OfTextEditor20250124
	} else if !param.IsOmitted(u.OfTextEditor20250429) {
		return u.OfTextEditor20250429
	} else if !param.IsOmitted(u.OfTextEditor20250728) {
		return u.OfTextEditor20250728
	} else if !param.IsOmitted(u.OfWebSearchTool20250305) {
		return u.OfWebSearchTool20250305
	} else if !param.IsOmitted(u.OfWebFetchTool20250910) {
		return u.OfWebFetchTool20250910
	} else if !param.IsOmitted(u.OfWebSearchTool20260209) {
		return u.OfWebSearchTool20260209
	} else if !param.IsOmitted(u.OfWebFetchTool20260209) {
		return u.OfWebFetchTool20260209
	} else if !param.IsOmitted(u.OfToolSearchToolBm25_20251119) {
		return u.OfToolSearchToolBm25_20251119
	} else if !param.IsOmitted(u.OfToolSearchToolRegex20251119) {
		return u.OfToolSearchToolRegex20251119
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetInputSchema() *ToolInputSchemaParam {
	if vt := u.OfTool; vt != nil {
		return &vt.InputSchema
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetDescription() *string {
	if vt := u.OfTool; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetEagerInputStreaming() *bool {
	if vt := u.OfTool; vt != nil && vt.EagerInputStreaming.Valid() {
		return &vt.EagerInputStreaming.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetMaxCharacters() *int64 {
	if vt := u.OfTextEditor20250728; vt != nil && vt.MaxCharacters.Valid() {
		return &vt.MaxCharacters.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return (*string)(&vt.Name)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetDeferLoading() *bool {
	if vt := u.OfTool; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250429; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20250728; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebSearchTool20250305; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetStrict() *bool {
	if vt := u.OfTool; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250429; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20250728; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebSearchTool20250305; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetType() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetMaxUses() *int64 {
	if vt := u.OfWebSearchTool20250305; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebSearchTool20260209; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxUses.Valid() {
		return &vt.MaxUses.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUnionParam) GetMaxContentTokens() *int64 {
	if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedCallers property, if
// present.
func (u ToolUnionParam) GetAllowedCallers() []string {
	if vt := u.OfTool; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return vt.AllowedCallers
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u ToolUnionParam) GetCacheControl() *CacheControlEphemeralParam {
	if vt := u.OfTool; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolSearchToolBm25_20251119; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfToolSearchToolRegex20251119; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a pointer to the underlying variant's InputExamples property, if
// present.
func (u ToolUnionParam) GetInputExamples() []map[string]any {
	if vt := u.OfTool; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250429; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20250728; vt != nil {
		return vt.InputExamples
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedDomains property, if
// present.
func (u ToolUnionParam) GetAllowedDomains() []string {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.AllowedDomains
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.AllowedDomains
	}
	return nil
}

// Returns a pointer to the underlying variant's BlockedDomains property, if
// present.
func (u ToolUnionParam) GetBlockedDomains() []string {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebFetchTool20250910; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return vt.BlockedDomains
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return vt.BlockedDomains
	}
	return nil
}

// Returns a pointer to the underlying variant's UserLocation property, if present.
func (u ToolUnionParam) GetUserLocation() *UserLocationParam {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.UserLocation
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.UserLocation
	}
	return nil
}

// Returns a pointer to the underlying variant's Citations property, if present.
func (u ToolUnionParam) GetCitations() *CitationsConfigParam {
	if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.Citations
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.Citations
	}
	return nil
}

type ToolUseBlock struct {
	ID string `json:"id,required"`
	// necessary custom code modification
	Input json.RawMessage  `json:"input,required"`
	Name  string           `json:"name,required"`
	Type  constant.ToolUse `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Caller      respjson.Field
		Input       respjson.Field
		Name        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ToolUseBlock) RawJSON() string { return r.JSON.raw }
func (r *ToolUseBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToolUseBlockCallerUnion contains all possible properties and values from
// [DirectCaller], [ServerToolCaller], [ServerToolCaller20260120].
//
// Use the [ToolUseBlockCallerUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ToolUseBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyToolUseBlockCaller is implemented by each variant of
// [ToolUseBlockCallerUnion] to add type safety for the return type of
// [ToolUseBlockCallerUnion.AsAny]
type anyToolUseBlockCaller interface {
	implToolUseBlockCallerUnion()
}

func (DirectCaller) implToolUseBlockCallerUnion()             {}
func (ServerToolCaller) implToolUseBlockCallerUnion()         {}
func (ServerToolCaller20260120) implToolUseBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ToolUseBlockCallerUnion.AsAny().(type) {
//	case anthropic.DirectCaller:
//	case anthropic.ServerToolCaller:
//	case anthropic.ServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ToolUseBlockCallerUnion) AsAny() anyToolUseBlockCaller {
	switch u.Type {
	case "direct":
		return u.AsDirect()
	case "code_execution_20250825":
		return u.AsCodeExecution20250825()
	case "code_execution_20260120":
		return u.AsCodeExecution20260120()
	}
	return nil
}

func (u ToolUseBlockCallerUnion) AsDirect() (v DirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ToolUseBlockCallerUnion) AsCodeExecution20250825() (v ServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ToolUseBlockCallerUnion) AsCodeExecution20260120() (v ServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ToolUseBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *ToolUseBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ID, Input, Name, Type are required.
type ToolUseBlockParam struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,omitzero,required"`
	Name  string `json:"name,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller ToolUseBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_use".
	Type constant.ToolUse `json:"type,required"`
	paramObj
}

func (r ToolUseBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow ToolUseBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ToolUseBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ToolUseBlockParamCallerUnion struct {
	OfDirect                *DirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *ServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *ServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u ToolUseBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *ToolUseBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ToolUseBlockParamCallerUnion) asAny() any {
	if !param.IsOmitted(u.OfDirect) {
		return u.OfDirect
	} else if !param.IsOmitted(u.OfCodeExecution20250825) {
		return u.OfCodeExecution20250825
	} else if !param.IsOmitted(u.OfCodeExecution20260120) {
		return u.OfCodeExecution20260120
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUseBlockParamCallerUnion) GetType() *string {
	if vt := u.OfDirect; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ToolUseBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[ToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[DirectCallerParam]("direct"),
		apijson.Discriminator[ServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[ServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

// The properties Type, URL are required.
type URLImageSourceParam struct {
	URL string `json:"url,required"`
	// This field can be elided, and will marshal its zero value as "url".
	Type constant.URL `json:"type,required"`
	paramObj
}

func (r URLImageSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow URLImageSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *URLImageSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, URL are required.
type URLPDFSourceParam struct {
	URL string `json:"url,required"`
	// This field can be elided, and will marshal its zero value as "url".
	Type constant.URL `json:"type,required"`
	paramObj
}

func (r URLPDFSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow URLPDFSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *URLPDFSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Usage struct {
	// Breakdown of cached tokens by TTL
	CacheCreation CacheCreation `json:"cache_creation,required"`
	// The number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The geographic region where inference was performed for this request.
	InferenceGeo string `json:"inference_geo,required"`
	// The number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// The number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// The number of server tool requests.
	ServerToolUse ServerToolUsage `json:"server_tool_use,required"`
	// If the request used the priority, standard, or batch tier.
	//
	// Any of "standard", "priority", "batch".
	ServiceTier UsageServiceTier `json:"service_tier,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreation            respjson.Field
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InferenceGeo             respjson.Field
		InputTokens              respjson.Field
		OutputTokens             respjson.Field
		ServerToolUse            respjson.Field
		ServiceTier              respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Usage) RawJSON() string { return r.JSON.raw }
func (r *Usage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// If the request used the priority, standard, or batch tier.
type UsageServiceTier string

const (
	UsageServiceTierStandard UsageServiceTier = "standard"
	UsageServiceTierPriority UsageServiceTier = "priority"
	UsageServiceTierBatch    UsageServiceTier = "batch"
)

// The property Type is required.
type UserLocationParam struct {
	// The city of the user.
	City param.Opt[string] `json:"city,omitzero"`
	// The two letter
	// [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2) of the
	// user.
	Country param.Opt[string] `json:"country,omitzero"`
	// The region of the user.
	Region param.Opt[string] `json:"region,omitzero"`
	// The [IANA timezone](https://nodatime.org/TimeZones) of the user.
	Timezone param.Opt[string] `json:"timezone,omitzero"`
	// This field can be elided, and will marshal its zero value as "approximate".
	Type constant.Approximate `json:"type,required"`
	paramObj
}

func (r UserLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow UserLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *UserLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type WebFetchBlock struct {
	Content DocumentBlock `json:"content,required"`
	// ISO 8601 timestamp when the content was retrieved
	RetrievedAt string                  `json:"retrieved_at,required"`
	Type        constant.WebFetchResult `json:"type,required"`
	// Fetched content URL
	URL string `json:"url,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		RetrievedAt respjson.Field
		Type        respjson.Field
		URL         respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebFetchBlock) RawJSON() string { return r.JSON.raw }
func (r *WebFetchBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, Type, URL are required.
type WebFetchBlockParam struct {
	Content DocumentBlockParam `json:"content,omitzero,required"`
	// Fetched content URL
	URL string `json:"url,required"`
	// ISO 8601 timestamp when the content was retrieved
	RetrievedAt param.Opt[string] `json:"retrieved_at,omitzero"`
	// This field can be elided, and will marshal its zero value as "web_fetch_result".
	Type constant.WebFetchResult `json:"type,required"`
	paramObj
}

func (r WebFetchBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow WebFetchBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebFetchBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type WebFetchTool20250910Param struct {
	// Maximum number of tokens used by including web page text content in the context.
	// The limit is approximate and does not apply to binary content such as PDFs.
	MaxContentTokens param.Opt[int64] `json:"max_content_tokens,omitzero"`
	// Maximum number of times the tool can be used in the API request.
	MaxUses param.Opt[int64] `json:"max_uses,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// List of domains to allow fetching from
	AllowedDomains []string `json:"allowed_domains,omitzero"`
	// List of domains to block fetching from
	BlockedDomains []string `json:"blocked_domains,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Citations configuration for fetched documents. Citations are disabled by
	// default.
	Citations CitationsConfigParam `json:"citations,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "web_fetch".
	Name constant.WebFetch `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_20250910".
	Type constant.WebFetch20250910 `json:"type,required"`
	paramObj
}

func (r WebFetchTool20250910Param) MarshalJSON() (data []byte, err error) {
	type shadow WebFetchTool20250910Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebFetchTool20250910Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type WebFetchTool20260209Param struct {
	// Maximum number of tokens used by including web page text content in the context.
	// The limit is approximate and does not apply to binary content such as PDFs.
	MaxContentTokens param.Opt[int64] `json:"max_content_tokens,omitzero"`
	// Maximum number of times the tool can be used in the API request.
	MaxUses param.Opt[int64] `json:"max_uses,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// List of domains to allow fetching from
	AllowedDomains []string `json:"allowed_domains,omitzero"`
	// List of domains to block fetching from
	BlockedDomains []string `json:"blocked_domains,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Citations configuration for fetched documents. Citations are disabled by
	// default.
	Citations CitationsConfigParam `json:"citations,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "web_fetch".
	Name constant.WebFetch `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_20260209".
	Type constant.WebFetch20260209 `json:"type,required"`
	paramObj
}

func (r WebFetchTool20260209Param) MarshalJSON() (data []byte, err error) {
	type shadow WebFetchTool20260209Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebFetchTool20260209Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type WebFetchToolResultBlock struct {
	// Tool invocation directly from the model.
	Caller    WebFetchToolResultBlockCallerUnion  `json:"caller,required"`
	Content   WebFetchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                              `json:"tool_use_id,required"`
	Type      constant.WebFetchToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Caller      respjson.Field
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebFetchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *WebFetchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// WebFetchToolResultBlockCallerUnion contains all possible properties and values
// from [DirectCaller], [ServerToolCaller], [ServerToolCaller20260120].
//
// Use the [WebFetchToolResultBlockCallerUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type WebFetchToolResultBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyWebFetchToolResultBlockCaller is implemented by each variant of
// [WebFetchToolResultBlockCallerUnion] to add type safety for the return type of
// [WebFetchToolResultBlockCallerUnion.AsAny]
type anyWebFetchToolResultBlockCaller interface {
	implWebFetchToolResultBlockCallerUnion()
}

func (DirectCaller) implWebFetchToolResultBlockCallerUnion()             {}
func (ServerToolCaller) implWebFetchToolResultBlockCallerUnion()         {}
func (ServerToolCaller20260120) implWebFetchToolResultBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := WebFetchToolResultBlockCallerUnion.AsAny().(type) {
//	case anthropic.DirectCaller:
//	case anthropic.ServerToolCaller:
//	case anthropic.ServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u WebFetchToolResultBlockCallerUnion) AsAny() anyWebFetchToolResultBlockCaller {
	switch u.Type {
	case "direct":
		return u.AsDirect()
	case "code_execution_20250825":
		return u.AsCodeExecution20250825()
	case "code_execution_20260120":
		return u.AsCodeExecution20260120()
	}
	return nil
}

func (u WebFetchToolResultBlockCallerUnion) AsDirect() (v DirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebFetchToolResultBlockCallerUnion) AsCodeExecution20250825() (v ServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebFetchToolResultBlockCallerUnion) AsCodeExecution20260120() (v ServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u WebFetchToolResultBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *WebFetchToolResultBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// WebFetchToolResultBlockContentUnion contains all possible properties and values
// from [WebFetchToolResultErrorBlock], [WebFetchBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type WebFetchToolResultBlockContentUnion struct {
	// This field is from variant [WebFetchToolResultErrorBlock].
	ErrorCode WebFetchToolResultErrorCode `json:"error_code"`
	Type      string                      `json:"type"`
	// This field is from variant [WebFetchBlock].
	Content DocumentBlock `json:"content"`
	// This field is from variant [WebFetchBlock].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [WebFetchBlock].
	URL  string `json:"url"`
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		Content     respjson.Field
		RetrievedAt respjson.Field
		URL         respjson.Field
		raw         string
	} `json:"-"`
}

func (u WebFetchToolResultBlockContentUnion) AsResponseWebFetchToolResultError() (v WebFetchToolResultErrorBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebFetchToolResultBlockContentUnion) AsResponseWebFetchResultBlock() (v WebFetchBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u WebFetchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *WebFetchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type WebFetchToolResultBlockParam struct {
	Content   WebFetchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                   `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller WebFetchToolResultBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_tool_result".
	Type constant.WebFetchToolResult `json:"type,required"`
	paramObj
}

func (r WebFetchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow WebFetchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebFetchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type WebFetchToolResultBlockParamContentUnion struct {
	OfRequestWebFetchToolResultError *WebFetchToolResultErrorBlockParam `json:",omitzero,inline"`
	OfRequestWebFetchResultBlock     *WebFetchBlockParam                `json:",omitzero,inline"`
	paramUnion
}

func (u WebFetchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestWebFetchToolResultError, u.OfRequestWebFetchResultBlock)
}
func (u *WebFetchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *WebFetchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestWebFetchToolResultError) {
		return u.OfRequestWebFetchToolResultError
	} else if !param.IsOmitted(u.OfRequestWebFetchResultBlock) {
		return u.OfRequestWebFetchResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestWebFetchToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamContentUnion) GetContent() *DocumentBlockParam {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamContentUnion) GetURL() *string {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamContentUnion) GetRetrievedAt() *string {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil && vt.RetrievedAt.Valid() {
		return &vt.RetrievedAt.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestWebFetchToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestWebFetchResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type WebFetchToolResultBlockParamCallerUnion struct {
	OfDirect                *DirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *ServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *ServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u WebFetchToolResultBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *WebFetchToolResultBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *WebFetchToolResultBlockParamCallerUnion) asAny() any {
	if !param.IsOmitted(u.OfDirect) {
		return u.OfDirect
	} else if !param.IsOmitted(u.OfCodeExecution20250825) {
		return u.OfCodeExecution20250825
	} else if !param.IsOmitted(u.OfCodeExecution20260120) {
		return u.OfCodeExecution20260120
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamCallerUnion) GetType() *string {
	if vt := u.OfDirect; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebFetchToolResultBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[WebFetchToolResultBlockParamCallerUnion](
		"type",
		apijson.Discriminator[DirectCallerParam]("direct"),
		apijson.Discriminator[ServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[ServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

type WebFetchToolResultErrorBlock struct {
	// Any of "invalid_tool_input", "url_too_long", "url_not_allowed",
	// "url_not_accessible", "unsupported_content_type", "too_many_requests",
	// "max_uses_exceeded", "unavailable".
	ErrorCode WebFetchToolResultErrorCode      `json:"error_code,required"`
	Type      constant.WebFetchToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebFetchToolResultErrorBlock) RawJSON() string { return r.JSON.raw }
func (r *WebFetchToolResultErrorBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ErrorCode, Type are required.
type WebFetchToolResultErrorBlockParam struct {
	// Any of "invalid_tool_input", "url_too_long", "url_not_allowed",
	// "url_not_accessible", "unsupported_content_type", "too_many_requests",
	// "max_uses_exceeded", "unavailable".
	ErrorCode WebFetchToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_tool_result_error".
	Type constant.WebFetchToolResultError `json:"type,required"`
	paramObj
}

func (r WebFetchToolResultErrorBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow WebFetchToolResultErrorBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebFetchToolResultErrorBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type WebFetchToolResultErrorCode string

const (
	WebFetchToolResultErrorCodeInvalidToolInput       WebFetchToolResultErrorCode = "invalid_tool_input"
	WebFetchToolResultErrorCodeURLTooLong             WebFetchToolResultErrorCode = "url_too_long"
	WebFetchToolResultErrorCodeURLNotAllowed          WebFetchToolResultErrorCode = "url_not_allowed"
	WebFetchToolResultErrorCodeURLNotAccessible       WebFetchToolResultErrorCode = "url_not_accessible"
	WebFetchToolResultErrorCodeUnsupportedContentType WebFetchToolResultErrorCode = "unsupported_content_type"
	WebFetchToolResultErrorCodeTooManyRequests        WebFetchToolResultErrorCode = "too_many_requests"
	WebFetchToolResultErrorCodeMaxUsesExceeded        WebFetchToolResultErrorCode = "max_uses_exceeded"
	WebFetchToolResultErrorCodeUnavailable            WebFetchToolResultErrorCode = "unavailable"
)

type WebSearchResultBlock struct {
	EncryptedContent string                   `json:"encrypted_content,required"`
	PageAge          string                   `json:"page_age,required"`
	Title            string                   `json:"title,required"`
	Type             constant.WebSearchResult `json:"type,required"`
	URL              string                   `json:"url,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		EncryptedContent respjson.Field
		PageAge          respjson.Field
		Title            respjson.Field
		Type             respjson.Field
		URL              respjson.Field
		ExtraFields      map[string]respjson.Field
		raw              string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebSearchResultBlock) RawJSON() string { return r.JSON.raw }
func (r *WebSearchResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties EncryptedContent, Title, Type, URL are required.
type WebSearchResultBlockParam struct {
	EncryptedContent string            `json:"encrypted_content,required"`
	Title            string            `json:"title,required"`
	URL              string            `json:"url,required"`
	PageAge          param.Opt[string] `json:"page_age,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_result".
	Type constant.WebSearchResult `json:"type,required"`
	paramObj
}

func (r WebSearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow WebSearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebSearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type WebSearchTool20250305Param struct {
	// Maximum number of times the tool can be used in the API request.
	MaxUses param.Opt[int64] `json:"max_uses,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// If provided, only these domains will be included in results. Cannot be used
	// alongside `blocked_domains`.
	AllowedDomains []string `json:"allowed_domains,omitzero"`
	// If provided, these domains will never appear in results. Cannot be used
	// alongside `allowed_domains`.
	BlockedDomains []string `json:"blocked_domains,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Parameters for the user's location. Used to provide more relevant search
	// results.
	UserLocation UserLocationParam `json:"user_location,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "web_search".
	Name constant.WebSearch `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_20250305".
	Type constant.WebSearch20250305 `json:"type,required"`
	paramObj
}

func (r WebSearchTool20250305Param) MarshalJSON() (data []byte, err error) {
	type shadow WebSearchTool20250305Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebSearchTool20250305Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type WebSearchTool20260209Param struct {
	// Maximum number of times the tool can be used in the API request.
	MaxUses param.Opt[int64] `json:"max_uses,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// If provided, only these domains will be included in results. Cannot be used
	// alongside `blocked_domains`.
	AllowedDomains []string `json:"allowed_domains,omitzero"`
	// If provided, these domains will never appear in results. Cannot be used
	// alongside `allowed_domains`.
	BlockedDomains []string `json:"blocked_domains,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Parameters for the user's location. Used to provide more relevant search
	// results.
	UserLocation UserLocationParam `json:"user_location,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "web_search".
	Name constant.WebSearch `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_20260209".
	Type constant.WebSearch20260209 `json:"type,required"`
	paramObj
}

func (r WebSearchTool20260209Param) MarshalJSON() (data []byte, err error) {
	type shadow WebSearchTool20260209Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebSearchTool20260209Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ErrorCode, Type are required.
type WebSearchToolRequestErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "max_uses_exceeded",
	// "too_many_requests", "query_too_long", "request_too_large".
	ErrorCode WebSearchToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_tool_result_error".
	Type constant.WebSearchToolResultError `json:"type,required"`
	paramObj
}

func (r WebSearchToolRequestErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow WebSearchToolRequestErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebSearchToolRequestErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type WebSearchToolResultBlock struct {
	// Tool invocation directly from the model.
	Caller    WebSearchToolResultBlockCallerUnion  `json:"caller,required"`
	Content   WebSearchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                               `json:"tool_use_id,required"`
	Type      constant.WebSearchToolResult         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Caller      respjson.Field
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebSearchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *WebSearchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// WebSearchToolResultBlockCallerUnion contains all possible properties and values
// from [DirectCaller], [ServerToolCaller], [ServerToolCaller20260120].
//
// Use the [WebSearchToolResultBlockCallerUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type WebSearchToolResultBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyWebSearchToolResultBlockCaller is implemented by each variant of
// [WebSearchToolResultBlockCallerUnion] to add type safety for the return type of
// [WebSearchToolResultBlockCallerUnion.AsAny]
type anyWebSearchToolResultBlockCaller interface {
	implWebSearchToolResultBlockCallerUnion()
}

func (DirectCaller) implWebSearchToolResultBlockCallerUnion()             {}
func (ServerToolCaller) implWebSearchToolResultBlockCallerUnion()         {}
func (ServerToolCaller20260120) implWebSearchToolResultBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := WebSearchToolResultBlockCallerUnion.AsAny().(type) {
//	case anthropic.DirectCaller:
//	case anthropic.ServerToolCaller:
//	case anthropic.ServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u WebSearchToolResultBlockCallerUnion) AsAny() anyWebSearchToolResultBlockCaller {
	switch u.Type {
	case "direct":
		return u.AsDirect()
	case "code_execution_20250825":
		return u.AsCodeExecution20250825()
	case "code_execution_20260120":
		return u.AsCodeExecution20260120()
	}
	return nil
}

func (u WebSearchToolResultBlockCallerUnion) AsDirect() (v DirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebSearchToolResultBlockCallerUnion) AsCodeExecution20250825() (v ServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebSearchToolResultBlockCallerUnion) AsCodeExecution20260120() (v ServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u WebSearchToolResultBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *WebSearchToolResultBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// WebSearchToolResultBlockContentUnion contains all possible properties and values
// from [WebSearchToolResultError], [[]WebSearchResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfWebSearchResultBlockArray]
type WebSearchToolResultBlockContentUnion struct {
	// This field will be present if the value is a [[]WebSearchResultBlock] instead of
	// an object.
	OfWebSearchResultBlockArray []WebSearchResultBlock `json:",inline"`
	// This field is from variant [WebSearchToolResultError].
	ErrorCode WebSearchToolResultErrorCode `json:"error_code"`
	// This field is from variant [WebSearchToolResultError].
	Type constant.WebSearchToolResultError `json:"type"`
	JSON struct {
		OfWebSearchResultBlockArray respjson.Field
		ErrorCode                   respjson.Field
		Type                        respjson.Field
		raw                         string
	} `json:"-"`
}

func (u WebSearchToolResultBlockContentUnion) AsResponseWebSearchToolResultError() (v WebSearchToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u WebSearchToolResultBlockContentUnion) AsWebSearchResultBlockArray() (v []WebSearchResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u WebSearchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *WebSearchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type WebSearchToolResultBlockParam struct {
	Content   WebSearchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                    `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller WebSearchToolResultBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_tool_result".
	Type constant.WebSearchToolResult `json:"type,required"`
	paramObj
}

func (r WebSearchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow WebSearchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *WebSearchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type WebSearchToolResultBlockParamCallerUnion struct {
	OfDirect                *DirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *ServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *ServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u WebSearchToolResultBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *WebSearchToolResultBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *WebSearchToolResultBlockParamCallerUnion) asAny() any {
	if !param.IsOmitted(u.OfDirect) {
		return u.OfDirect
	} else if !param.IsOmitted(u.OfCodeExecution20250825) {
		return u.OfCodeExecution20250825
	} else if !param.IsOmitted(u.OfCodeExecution20260120) {
		return u.OfCodeExecution20260120
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebSearchToolResultBlockParamCallerUnion) GetType() *string {
	if vt := u.OfDirect; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u WebSearchToolResultBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[WebSearchToolResultBlockParamCallerUnion](
		"type",
		apijson.Discriminator[DirectCallerParam]("direct"),
		apijson.Discriminator[ServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[ServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

func NewWebSearchToolRequestError(errorCode WebSearchToolResultErrorCode) WebSearchToolResultBlockParamContentUnion {
	var variant WebSearchToolRequestErrorParam
	variant.ErrorCode = errorCode
	return WebSearchToolResultBlockParamContentUnion{OfRequestWebSearchToolResultError: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type WebSearchToolResultBlockParamContentUnion struct {
	OfWebSearchToolResultBlockItem    []WebSearchResultBlockParam     `json:",omitzero,inline"`
	OfRequestWebSearchToolResultError *WebSearchToolRequestErrorParam `json:",omitzero,inline"`
	paramUnion
}

func (u WebSearchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfWebSearchToolResultBlockItem, u.OfRequestWebSearchToolResultError)
}
func (u *WebSearchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *WebSearchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfWebSearchToolResultBlockItem) {
		return &u.OfWebSearchToolResultBlockItem
	} else if !param.IsOmitted(u.OfRequestWebSearchToolResultError) {
		return u.OfRequestWebSearchToolResultError
	}
	return nil
}

type WebSearchToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "max_uses_exceeded",
	// "too_many_requests", "query_too_long", "request_too_large".
	ErrorCode WebSearchToolResultErrorCode      `json:"error_code,required"`
	Type      constant.WebSearchToolResultError `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r WebSearchToolResultError) RawJSON() string { return r.JSON.raw }
func (r *WebSearchToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type WebSearchToolResultErrorCode string

const (
	WebSearchToolResultErrorCodeInvalidToolInput WebSearchToolResultErrorCode = "invalid_tool_input"
	WebSearchToolResultErrorCodeUnavailable      WebSearchToolResultErrorCode = "unavailable"
	WebSearchToolResultErrorCodeMaxUsesExceeded  WebSearchToolResultErrorCode = "max_uses_exceeded"
	WebSearchToolResultErrorCodeTooManyRequests  WebSearchToolResultErrorCode = "too_many_requests"
	WebSearchToolResultErrorCodeQueryTooLong     WebSearchToolResultErrorCode = "query_too_long"
	WebSearchToolResultErrorCodeRequestTooLarge  WebSearchToolResultErrorCode = "request_too_large"
)

type MessageNewParams struct {
	// The maximum number of tokens to generate before stopping.
	//
	// Note that our models may stop _before_ reaching this maximum. This parameter
	// only specifies the absolute maximum number of tokens to generate.
	//
	// Different models have different maximum values for this parameter. See
	// [models](https://docs.claude.com/en/docs/models-overview) for details.
	MaxTokens int64 `json:"max_tokens,required"`
	// Input messages.
	//
	// Our models are trained to operate on alternating `user` and `assistant`
	// conversational turns. When creating a new `Message`, you specify the prior
	// conversational turns with the `messages` parameter, and the model then generates
	// the next `Message` in the conversation. Consecutive `user` or `assistant` turns
	// in your request will be combined into a single turn.
	//
	// Each input message must be an object with a `role` and `content`. You can
	// specify a single `user`-role message, or you can include multiple `user` and
	// `assistant` messages.
	//
	// If the final message uses the `assistant` role, the response content will
	// continue immediately from the content in that message. This can be used to
	// constrain part of the model's response.
	//
	// Example with a single `user` message:
	//
	// ```json
	// [{ "role": "user", "content": "Hello, Claude" }]
	// ```
	//
	// Example with multiple conversational turns:
	//
	// ```json
	// [
	//
	//	{ "role": "user", "content": "Hello there." },
	//	{ "role": "assistant", "content": "Hi, I'm Claude. How can I help you?" },
	//	{ "role": "user", "content": "Can you explain LLMs in plain English?" }
	//
	// ]
	// ```
	//
	// Example with a partially-filled response from Claude:
	//
	// ```json
	// [
	//
	//	{
	//	  "role": "user",
	//	  "content": "What's the Greek name for Sun? (A) Sol (B) Helios (C) Sun"
	//	},
	//	{ "role": "assistant", "content": "The best answer is (" }
	//
	// ]
	// ```
	//
	// Each input message `content` may be either a single `string` or an array of
	// content blocks, where each block has a specific `type`. Using a `string` for
	// `content` is shorthand for an array of one content block of type `"text"`. The
	// following input messages are equivalent:
	//
	// ```json
	// { "role": "user", "content": "Hello, Claude" }
	// ```
	//
	// ```json
	// { "role": "user", "content": [{ "type": "text", "text": "Hello, Claude" }] }
	// ```
	//
	// See [input examples](https://docs.claude.com/en/api/messages-examples).
	//
	// Note that if you want to include a
	// [system prompt](https://docs.claude.com/en/docs/system-prompts), you can use the
	// top-level `system` parameter — there is no `"system"` role for input messages in
	// the Messages API.
	//
	// There is a limit of 100,000 messages in a single request.
	Messages []MessageParam `json:"messages,omitzero,required"`
	// The model that will complete your prompt.\n\nSee
	// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
	// details and options.
	Model Model `json:"model,omitzero,required"`
	// Container identifier for reuse across requests.
	Container param.Opt[string] `json:"container,omitzero"`
	// Specifies the geographic region for inference processing. If not specified, the
	// workspace's `default_inference_geo` is used.
	InferenceGeo param.Opt[string] `json:"inference_geo,omitzero"`
	// Amount of randomness injected into the response.
	//
	// Defaults to `1.0`. Ranges from `0.0` to `1.0`. Use `temperature` closer to `0.0`
	// for analytical / multiple choice, and closer to `1.0` for creative and
	// generative tasks.
	//
	// Note that even with `temperature` of `0.0`, the results will not be fully
	// deterministic.
	Temperature param.Opt[float64] `json:"temperature,omitzero"`
	// Only sample from the top K options for each subsequent token.
	//
	// Used to remove "long tail" low probability responses.
	// [Learn more technical details here](https://towardsdatascience.com/how-to-sample-from-language-models-682bceb97277).
	//
	// Recommended for advanced use cases only. You usually only need to use
	// `temperature`.
	TopK param.Opt[int64] `json:"top_k,omitzero"`
	// Use nucleus sampling.
	//
	// In nucleus sampling, we compute the cumulative distribution over all the options
	// for each subsequent token in decreasing probability order and cut it off once it
	// reaches a particular probability specified by `top_p`. You should either alter
	// `temperature` or `top_p`, but not both.
	//
	// Recommended for advanced use cases only. You usually only need to use
	// `temperature`.
	TopP param.Opt[float64] `json:"top_p,omitzero"`
	// Top-level cache control automatically applies a cache_control marker to the last
	// cacheable block in the request.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// An object describing metadata about the request.
	Metadata MetadataParam `json:"metadata,omitzero"`
	// Configuration options for the model's output, such as the output format.
	OutputConfig OutputConfigParam `json:"output_config,omitzero"`
	// Determines whether to use priority capacity (if available) or standard capacity
	// for this request.
	//
	// Anthropic offers different levels of service for your API requests. See
	// [service-tiers](https://docs.claude.com/en/api/service-tiers) for details.
	//
	// Any of "auto", "standard_only".
	ServiceTier MessageNewParamsServiceTier `json:"service_tier,omitzero"`
	// Custom text sequences that will cause the model to stop generating.
	//
	// Our models will normally stop when they have naturally completed their turn,
	// which will result in a response `stop_reason` of `"end_turn"`.
	//
	// If you want the model to stop generating when it encounters custom strings of
	// text, you can use the `stop_sequences` parameter. If the model encounters one of
	// the custom sequences, the response `stop_reason` value will be `"stop_sequence"`
	// and the response `stop_sequence` value will contain the matched stop sequence.
	StopSequences []string `json:"stop_sequences,omitzero"`
	// System prompt.
	//
	// A system prompt is a way of providing context and instructions to Claude, such
	// as specifying a particular goal or role. See our
	// [guide to system prompts](https://docs.claude.com/en/docs/system-prompts).
	System []TextBlockParam `json:"system,omitzero"`
	// Configuration for enabling Claude's extended thinking.
	//
	// When enabled, responses include `thinking` content blocks showing Claude's
	// thinking process before the final answer. Requires a minimum budget of 1,024
	// tokens and counts towards your `max_tokens` limit.
	//
	// See
	// [extended thinking](https://docs.claude.com/en/docs/build-with-claude/extended-thinking)
	// for details.
	Thinking ThinkingConfigParamUnion `json:"thinking,omitzero"`
	// How the model should use the provided tools. The model can use a specific tool,
	// any available tool, decide by itself, or not use tools at all.
	ToolChoice ToolChoiceUnionParam `json:"tool_choice,omitzero"`
	// Definitions of tools that the model may use.
	//
	// If you include `tools` in your API request, the model may return `tool_use`
	// content blocks that represent the model's use of those tools. You can then run
	// those tools using the tool input generated by the model and then optionally
	// return results back to the model using `tool_result` content blocks.
	//
	// There are two types of tools: **client tools** and **server tools**. The
	// behavior described below applies to client tools. For
	// [server tools](https://docs.claude.com/en/docs/agents-and-tools/tool-use/overview#server-tools),
	// see their individual documentation as each has its own behavior (e.g., the
	// [web search tool](https://docs.claude.com/en/docs/agents-and-tools/tool-use/web-search-tool)).
	//
	// Each tool definition includes:
	//
	//   - `name`: Name of the tool.
	//   - `description`: Optional, but strongly-recommended description of the tool.
	//   - `input_schema`: [JSON schema](https://json-schema.org/draft/2020-12) for the
	//     tool `input` shape that the model will produce in `tool_use` output content
	//     blocks.
	//
	// For example, if you defined `tools` as:
	//
	// ```json
	// [
	//
	//	{
	//	  "name": "get_stock_price",
	//	  "description": "Get the current stock price for a given ticker symbol.",
	//	  "input_schema": {
	//	    "type": "object",
	//	    "properties": {
	//	      "ticker": {
	//	        "type": "string",
	//	        "description": "The stock ticker symbol, e.g. AAPL for Apple Inc."
	//	      }
	//	    },
	//	    "required": ["ticker"]
	//	  }
	//	}
	//
	// ]
	// ```
	//
	// And then asked the model "What's the S&P 500 at today?", the model might produce
	// `tool_use` content blocks in the response like this:
	//
	// ```json
	// [
	//
	//	{
	//	  "type": "tool_use",
	//	  "id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
	//	  "name": "get_stock_price",
	//	  "input": { "ticker": "^GSPC" }
	//	}
	//
	// ]
	// ```
	//
	// You might then run your `get_stock_price` tool with `{"ticker": "^GSPC"}` as an
	// input, and return the following back to the model in a subsequent `user`
	// message:
	//
	// ```json
	// [
	//
	//	{
	//	  "type": "tool_result",
	//	  "tool_use_id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
	//	  "content": "259.75 USD"
	//	}
	//
	// ]
	// ```
	//
	// Tools can be used for workflows that include running client-side tools and
	// functions, or more generally whenever you want the model to produce a particular
	// JSON structure of output.
	//
	// See our [guide](https://docs.claude.com/en/docs/tool-use) for more details.
	Tools []ToolUnionParam `json:"tools,omitzero"`
	paramObj
}

func (r MessageNewParams) MarshalJSON() (data []byte, err error) {
	type shadow MessageNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *MessageNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Determines whether to use priority capacity (if available) or standard capacity
// for this request.
//
// Anthropic offers different levels of service for your API requests. See
// [service-tiers](https://docs.claude.com/en/api/service-tiers) for details.
type MessageNewParamsServiceTier string

const (
	MessageNewParamsServiceTierAuto         MessageNewParamsServiceTier = "auto"
	MessageNewParamsServiceTierStandardOnly MessageNewParamsServiceTier = "standard_only"
)

type MessageCountTokensParams struct {
	// Input messages.
	//
	// Our models are trained to operate on alternating `user` and `assistant`
	// conversational turns. When creating a new `Message`, you specify the prior
	// conversational turns with the `messages` parameter, and the model then generates
	// the next `Message` in the conversation. Consecutive `user` or `assistant` turns
	// in your request will be combined into a single turn.
	//
	// Each input message must be an object with a `role` and `content`. You can
	// specify a single `user`-role message, or you can include multiple `user` and
	// `assistant` messages.
	//
	// If the final message uses the `assistant` role, the response content will
	// continue immediately from the content in that message. This can be used to
	// constrain part of the model's response.
	//
	// Example with a single `user` message:
	//
	// ```json
	// [{ "role": "user", "content": "Hello, Claude" }]
	// ```
	//
	// Example with multiple conversational turns:
	//
	// ```json
	// [
	//
	//	{ "role": "user", "content": "Hello there." },
	//	{ "role": "assistant", "content": "Hi, I'm Claude. How can I help you?" },
	//	{ "role": "user", "content": "Can you explain LLMs in plain English?" }
	//
	// ]
	// ```
	//
	// Example with a partially-filled response from Claude:
	//
	// ```json
	// [
	//
	//	{
	//	  "role": "user",
	//	  "content": "What's the Greek name for Sun? (A) Sol (B) Helios (C) Sun"
	//	},
	//	{ "role": "assistant", "content": "The best answer is (" }
	//
	// ]
	// ```
	//
	// Each input message `content` may be either a single `string` or an array of
	// content blocks, where each block has a specific `type`. Using a `string` for
	// `content` is shorthand for an array of one content block of type `"text"`. The
	// following input messages are equivalent:
	//
	// ```json
	// { "role": "user", "content": "Hello, Claude" }
	// ```
	//
	// ```json
	// { "role": "user", "content": [{ "type": "text", "text": "Hello, Claude" }] }
	// ```
	//
	// See [input examples](https://docs.claude.com/en/api/messages-examples).
	//
	// Note that if you want to include a
	// [system prompt](https://docs.claude.com/en/docs/system-prompts), you can use the
	// top-level `system` parameter — there is no `"system"` role for input messages in
	// the Messages API.
	//
	// There is a limit of 100,000 messages in a single request.
	Messages []MessageParam `json:"messages,omitzero,required"`
	// The model that will complete your prompt.\n\nSee
	// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
	// details and options.
	Model Model `json:"model,omitzero,required"`
	// Top-level cache control automatically applies a cache_control marker to the last
	// cacheable block in the request.
	CacheControl CacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Configuration options for the model's output, such as the output format.
	OutputConfig OutputConfigParam `json:"output_config,omitzero"`
	// System prompt.
	//
	// A system prompt is a way of providing context and instructions to Claude, such
	// as specifying a particular goal or role. See our
	// [guide to system prompts](https://docs.claude.com/en/docs/system-prompts).
	System MessageCountTokensParamsSystemUnion `json:"system,omitzero"`
	// Configuration for enabling Claude's extended thinking.
	//
	// When enabled, responses include `thinking` content blocks showing Claude's
	// thinking process before the final answer. Requires a minimum budget of 1,024
	// tokens and counts towards your `max_tokens` limit.
	//
	// See
	// [extended thinking](https://docs.claude.com/en/docs/build-with-claude/extended-thinking)
	// for details.
	Thinking ThinkingConfigParamUnion `json:"thinking,omitzero"`
	// How the model should use the provided tools. The model can use a specific tool,
	// any available tool, decide by itself, or not use tools at all.
	ToolChoice ToolChoiceUnionParam `json:"tool_choice,omitzero"`
	// Definitions of tools that the model may use.
	//
	// If you include `tools` in your API request, the model may return `tool_use`
	// content blocks that represent the model's use of those tools. You can then run
	// those tools using the tool input generated by the model and then optionally
	// return results back to the model using `tool_result` content blocks.
	//
	// There are two types of tools: **client tools** and **server tools**. The
	// behavior described below applies to client tools. For
	// [server tools](https://docs.claude.com/en/docs/agents-and-tools/tool-use/overview#server-tools),
	// see their individual documentation as each has its own behavior (e.g., the
	// [web search tool](https://docs.claude.com/en/docs/agents-and-tools/tool-use/web-search-tool)).
	//
	// Each tool definition includes:
	//
	//   - `name`: Name of the tool.
	//   - `description`: Optional, but strongly-recommended description of the tool.
	//   - `input_schema`: [JSON schema](https://json-schema.org/draft/2020-12) for the
	//     tool `input` shape that the model will produce in `tool_use` output content
	//     blocks.
	//
	// For example, if you defined `tools` as:
	//
	// ```json
	// [
	//
	//	{
	//	  "name": "get_stock_price",
	//	  "description": "Get the current stock price for a given ticker symbol.",
	//	  "input_schema": {
	//	    "type": "object",
	//	    "properties": {
	//	      "ticker": {
	//	        "type": "string",
	//	        "description": "The stock ticker symbol, e.g. AAPL for Apple Inc."
	//	      }
	//	    },
	//	    "required": ["ticker"]
	//	  }
	//	}
	//
	// ]
	// ```
	//
	// And then asked the model "What's the S&P 500 at today?", the model might produce
	// `tool_use` content blocks in the response like this:
	//
	// ```json
	// [
	//
	//	{
	//	  "type": "tool_use",
	//	  "id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
	//	  "name": "get_stock_price",
	//	  "input": { "ticker": "^GSPC" }
	//	}
	//
	// ]
	// ```
	//
	// You might then run your `get_stock_price` tool with `{"ticker": "^GSPC"}` as an
	// input, and return the following back to the model in a subsequent `user`
	// message:
	//
	// ```json
	// [
	//
	//	{
	//	  "type": "tool_result",
	//	  "tool_use_id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
	//	  "content": "259.75 USD"
	//	}
	//
	// ]
	// ```
	//
	// Tools can be used for workflows that include running client-side tools and
	// functions, or more generally whenever you want the model to produce a particular
	// JSON structure of output.
	//
	// See our [guide](https://docs.claude.com/en/docs/tool-use) for more details.
	Tools []MessageCountTokensToolUnionParam `json:"tools,omitzero"`
	paramObj
}

func (r MessageCountTokensParams) MarshalJSON() (data []byte, err error) {
	type shadow MessageCountTokensParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *MessageCountTokensParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type MessageCountTokensParamsSystemUnion struct {
	OfString         param.Opt[string] `json:",omitzero,inline"`
	OfTextBlockArray []TextBlockParam  `json:",omitzero,inline"`
	paramUnion
}

func (u MessageCountTokensParamsSystemUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfTextBlockArray)
}
func (u *MessageCountTokensParamsSystemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *MessageCountTokensParamsSystemUnion) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfTextBlockArray) {
		return &u.OfTextBlockArray
	}
	return nil
}
