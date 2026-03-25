// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
)

// BetaMessageService contains methods and other services that help with
// interacting with the anthropic API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewBetaMessageService] method instead.
type BetaMessageService struct {
	Options []option.RequestOption
	Batches BetaMessageBatchService
}

// NewBetaMessageService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewBetaMessageService(opts ...option.RequestOption) (r BetaMessageService) {
	r = BetaMessageService{}
	r.Options = opts
	r.Batches = NewBetaMessageBatchService(opts...)
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
func (r *BetaMessageService) New(ctx context.Context, params BetaMessageNewParams, opts ...option.RequestOption) (res *BetaMessage, err error) {
	for _, v := range params.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)

	// For non-streaming requests, calculate the appropriate timeout based on maxTokens
	// and check against model-specific limits
	timeout, timeoutErr := CalculateNonStreamingTimeout(int(params.MaxTokens), params.Model, opts)
	if timeoutErr != nil {
		return nil, timeoutErr
	}
	opts = append(opts, option.WithRequestTimeout(timeout))

	path := "v1/messages?beta=true"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
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
func (r *BetaMessageService) NewStreaming(ctx context.Context, params BetaMessageNewParams, opts ...option.RequestOption) (stream *ssestream.Stream[BetaRawMessageStreamEventUnion]) {
	var (
		raw *http.Response
		err error
	)
	for _, v := range params.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	opts = append(opts, option.WithJSONSet("stream", true))
	path := "v1/messages?beta=true"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &raw, opts...)
	return ssestream.NewStream[BetaRawMessageStreamEventUnion](ssestream.NewDecoder(raw), err)
}

// Count the number of tokens in a Message.
//
// The Token Count API can be used to count the number of tokens in a Message,
// including tools, images, and documents, without creating it.
//
// Learn more about token counting in our
// [user guide](https://docs.claude.com/en/docs/build-with-claude/token-counting)
func (r *BetaMessageService) CountTokens(ctx context.Context, params BetaMessageCountTokensParams, opts ...option.RequestOption) (res *BetaMessageTokensCount, err error) {
	for _, v := range params.Betas {
		opts = append(opts, option.WithHeaderAdd("anthropic-beta", fmt.Sprintf("%v", v)))
	}
	opts = slices.Concat(r.Options, opts)
	path := "v1/messages/count_tokens?beta=true"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

func NewBetaAllThinkingTurnsParam() BetaAllThinkingTurnsParam {
	return BetaAllThinkingTurnsParam{
		Type: "all",
	}
}

// This struct has a constant value, construct it with
// [NewBetaAllThinkingTurnsParam].
type BetaAllThinkingTurnsParam struct {
	Type constant.All `json:"type,required"`
	paramObj
}

func (r BetaAllThinkingTurnsParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaAllThinkingTurnsParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaAllThinkingTurnsParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Data, MediaType, Type are required.
type BetaBase64ImageSourceParam struct {
	Data string `json:"data,required" format:"byte"`
	// Any of "image/jpeg", "image/png", "image/gif", "image/webp".
	MediaType BetaBase64ImageSourceMediaType `json:"media_type,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "base64".
	Type constant.Base64 `json:"type,required"`
	paramObj
}

func (r BetaBase64ImageSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBase64ImageSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBase64ImageSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBase64ImageSourceMediaType string

const (
	BetaBase64ImageSourceMediaTypeImageJPEG BetaBase64ImageSourceMediaType = "image/jpeg"
	BetaBase64ImageSourceMediaTypeImagePNG  BetaBase64ImageSourceMediaType = "image/png"
	BetaBase64ImageSourceMediaTypeImageGIF  BetaBase64ImageSourceMediaType = "image/gif"
	BetaBase64ImageSourceMediaTypeImageWebP BetaBase64ImageSourceMediaType = "image/webp"
)

// The properties Source, Type are required.
type BetaBase64PDFBlockParam struct {
	Source  BetaBase64PDFBlockSourceUnionParam `json:"source,omitzero,required"`
	Context param.Opt[string]                  `json:"context,omitzero"`
	Title   param.Opt[string]                  `json:"title,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	Citations    BetaCitationsConfigParam       `json:"citations,omitzero"`
	// This field can be elided, and will marshal its zero value as "document".
	Type constant.Document `json:"type,required"`
	paramObj
}

func (r BetaBase64PDFBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBase64PDFBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBase64PDFBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaBase64PDFBlockSourceUnionParam struct {
	OfBase64  *BetaBase64PDFSourceParam    `json:",omitzero,inline"`
	OfText    *BetaPlainTextSourceParam    `json:",omitzero,inline"`
	OfContent *BetaContentBlockSourceParam `json:",omitzero,inline"`
	OfURL     *BetaURLPDFSourceParam       `json:",omitzero,inline"`
	OfFile    *BetaFileDocumentSourceParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaBase64PDFBlockSourceUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBase64,
		u.OfText,
		u.OfContent,
		u.OfURL,
		u.OfFile)
}
func (u *BetaBase64PDFBlockSourceUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaBase64PDFBlockSourceUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBase64) {
		return u.OfBase64
	} else if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfContent) {
		return u.OfContent
	} else if !param.IsOmitted(u.OfURL) {
		return u.OfURL
	} else if !param.IsOmitted(u.OfFile) {
		return u.OfFile
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetContent() *BetaContentBlockSourceContentUnionParam {
	if vt := u.OfContent; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetURL() *string {
	if vt := u.OfURL; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetFileID() *string {
	if vt := u.OfFile; vt != nil {
		return &vt.FileID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetData() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Data)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Data)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetMediaType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.MediaType)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.MediaType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBase64PDFBlockSourceUnionParam) GetType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContent; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfURL; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfFile; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[BetaBase64PDFBlockSourceUnionParam](
		"type",
		apijson.Discriminator[BetaBase64PDFSourceParam]("base64"),
		apijson.Discriminator[BetaPlainTextSourceParam]("text"),
		apijson.Discriminator[BetaContentBlockSourceParam]("content"),
		apijson.Discriminator[BetaURLPDFSourceParam]("url"),
		apijson.Discriminator[BetaFileDocumentSourceParam]("file"),
	)
}

func init() {
	apijson.RegisterUnion[BetaImageBlockParamSourceUnion](
		"type",
		apijson.Discriminator[BetaBase64ImageSourceParam]("base64"),
		apijson.Discriminator[BetaURLImageSourceParam]("url"),
		apijson.Discriminator[BetaFileImageSourceParam]("file"),
	)
}

func init() {
	apijson.RegisterUnion[BetaRequestDocumentBlockSourceUnionParam](
		"type",
		apijson.Discriminator[BetaBase64PDFSourceParam]("base64"),
		apijson.Discriminator[BetaPlainTextSourceParam]("text"),
		apijson.Discriminator[BetaContentBlockSourceParam]("content"),
		apijson.Discriminator[BetaURLPDFSourceParam]("url"),
		apijson.Discriminator[BetaFileDocumentSourceParam]("file"),
	)
}

func init() {
	apijson.RegisterUnion[BetaTextCitationParamUnion](
		"type",
		apijson.Discriminator[BetaCitationCharLocationParam]("char_location"),
		apijson.Discriminator[BetaCitationPageLocationParam]("page_location"),
		apijson.Discriminator[BetaCitationContentBlockLocationParam]("content_block_location"),
		apijson.Discriminator[BetaCitationWebSearchResultLocationParam]("web_search_result_location"),
		apijson.Discriminator[BetaCitationSearchResultLocationParam]("search_result_location"),
	)
}

func init() {
	apijson.RegisterUnion[BetaThinkingConfigParamUnion](
		"type",
		apijson.Discriminator[BetaThinkingConfigEnabledParam]("enabled"),
		apijson.Discriminator[BetaThinkingConfigDisabledParam]("disabled"),
	)
}

func init() {
	apijson.RegisterUnion[BetaToolChoiceUnionParam](
		"type",
		apijson.Discriminator[BetaToolChoiceAutoParam]("auto"),
		apijson.Discriminator[BetaToolChoiceAnyParam]("any"),
		apijson.Discriminator[BetaToolChoiceToolParam]("tool"),
		apijson.Discriminator[BetaToolChoiceNoneParam]("none"),
	)
}

func init() {
	apijson.RegisterUnion[BetaClearToolUses20250919EditTriggerUnionParam](
		"type",
		apijson.Discriminator[BetaInputTokensTriggerParam]("input_tokens"),
		apijson.Discriminator[BetaToolUsesTriggerParam]("tool_uses"),
	)
}

func init() {
	apijson.RegisterUnion[BetaContentBlockParamUnion](
		"type",
		apijson.Discriminator[BetaTextBlockParam]("text"),
		apijson.Discriminator[BetaImageBlockParam]("image"),
		apijson.Discriminator[BetaRequestDocumentBlockParam]("document"),
		apijson.Discriminator[BetaSearchResultBlockParam]("search_result"),
		apijson.Discriminator[BetaThinkingBlockParam]("thinking"),
		apijson.Discriminator[BetaRedactedThinkingBlockParam]("redacted_thinking"),
		apijson.Discriminator[BetaToolUseBlockParam]("tool_use"),
		apijson.Discriminator[BetaToolResultBlockParam]("tool_result"),
		apijson.Discriminator[BetaServerToolUseBlockParam]("server_tool_use"),
		apijson.Discriminator[BetaWebSearchToolResultBlockParam]("web_search_tool_result"),
		apijson.Discriminator[BetaWebFetchToolResultBlockParam]("web_fetch_tool_result"),
		apijson.Discriminator[BetaCodeExecutionToolResultBlockParam]("code_execution_tool_result"),
		apijson.Discriminator[BetaBashCodeExecutionToolResultBlockParam]("bash_code_execution_tool_result"),
		apijson.Discriminator[BetaTextEditorCodeExecutionToolResultBlockParam]("text_editor_code_execution_tool_result"),
		apijson.Discriminator[BetaToolSearchToolResultBlockParam]("tool_search_tool_result"),
		apijson.Discriminator[BetaMCPToolUseBlockParam]("mcp_tool_use"),
		apijson.Discriminator[BetaRequestMCPToolResultBlockParam]("mcp_tool_result"),
		apijson.Discriminator[BetaContainerUploadBlockParam]("container_upload"),
	)
}

func init() {
	apijson.RegisterUnion[BetaContextManagementConfigEditUnionParam](
		"type",
		apijson.Discriminator[BetaClearToolUses20250919EditParam]("clear_tool_uses_20250919"),
		apijson.Discriminator[BetaClearThinking20251015EditParam]("clear_thinking_20251015"),
	)
}

func init() {
	apijson.RegisterUnion[BetaServerToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
	)
}

func init() {
	apijson.RegisterUnion[BetaToolResultBlockParamContentUnion](
		"type",
		apijson.Discriminator[BetaTextBlockParam]("text"),
		apijson.Discriminator[BetaImageBlockParam]("image"),
		apijson.Discriminator[BetaSearchResultBlockParam]("search_result"),
		apijson.Discriminator[BetaRequestDocumentBlockParam]("document"),
		apijson.Discriminator[BetaToolReferenceBlockParam]("tool_reference"),
	)
}

func init() {
	apijson.RegisterUnion[BetaToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
	)
}

type BetaBase64PDFSource struct {
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
func (r BetaBase64PDFSource) RawJSON() string { return r.JSON.raw }
func (r *BetaBase64PDFSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BetaBase64PDFSource to a BetaBase64PDFSourceParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BetaBase64PDFSourceParam.Overrides()
func (r BetaBase64PDFSource) ToParam() BetaBase64PDFSourceParam {
	return param.Override[BetaBase64PDFSourceParam](json.RawMessage(r.RawJSON()))
}

// The properties Data, MediaType, Type are required.
type BetaBase64PDFSourceParam struct {
	Data string `json:"data,required" format:"byte"`
	// This field can be elided, and will marshal its zero value as "application/pdf".
	MediaType constant.ApplicationPDF `json:"media_type,required"`
	// This field can be elided, and will marshal its zero value as "base64".
	Type constant.Base64 `json:"type,required"`
	paramObj
}

func (r BetaBase64PDFSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBase64PDFSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBase64PDFSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBashCodeExecutionOutputBlock struct {
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
func (r BetaBashCodeExecutionOutputBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaBashCodeExecutionOutputBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type BetaBashCodeExecutionOutputBlockParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_output".
	Type constant.BashCodeExecutionOutput `json:"type,required"`
	paramObj
}

func (r BetaBashCodeExecutionOutputBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBashCodeExecutionOutputBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBashCodeExecutionOutputBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBashCodeExecutionResultBlock struct {
	Content    []BetaBashCodeExecutionOutputBlock `json:"content,required"`
	ReturnCode int64                              `json:"return_code,required"`
	Stderr     string                             `json:"stderr,required"`
	Stdout     string                             `json:"stdout,required"`
	Type       constant.BashCodeExecutionResult   `json:"type,required"`
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
func (r BetaBashCodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaBashCodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ReturnCode, Stderr, Stdout, Type are required.
type BetaBashCodeExecutionResultBlockParam struct {
	Content    []BetaBashCodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	ReturnCode int64                                   `json:"return_code,required"`
	Stderr     string                                  `json:"stderr,required"`
	Stdout     string                                  `json:"stdout,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_result".
	Type constant.BashCodeExecutionResult `json:"type,required"`
	paramObj
}

func (r BetaBashCodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBashCodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBashCodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBashCodeExecutionToolResultBlock struct {
	Content   BetaBashCodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                           `json:"tool_use_id,required"`
	Type      constant.BashCodeExecutionToolResult             `json:"type,required"`
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
func (r BetaBashCodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaBashCodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaBashCodeExecutionToolResultBlockContentUnion contains all possible
// properties and values from [BetaBashCodeExecutionToolResultError],
// [BetaBashCodeExecutionResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaBashCodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [BetaBashCodeExecutionToolResultError].
	ErrorCode BetaBashCodeExecutionToolResultErrorErrorCode `json:"error_code"`
	Type      string                                        `json:"type"`
	// This field is from variant [BetaBashCodeExecutionResultBlock].
	Content []BetaBashCodeExecutionOutputBlock `json:"content"`
	// This field is from variant [BetaBashCodeExecutionResultBlock].
	ReturnCode int64 `json:"return_code"`
	// This field is from variant [BetaBashCodeExecutionResultBlock].
	Stderr string `json:"stderr"`
	// This field is from variant [BetaBashCodeExecutionResultBlock].
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

func (u BetaBashCodeExecutionToolResultBlockContentUnion) AsResponseBashCodeExecutionToolResultError() (v BetaBashCodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaBashCodeExecutionToolResultBlockContentUnion) AsResponseBashCodeExecutionResultBlock() (v BetaBashCodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaBashCodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaBashCodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaBashCodeExecutionToolResultBlockParam struct {
	Content   BetaBashCodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                                `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_tool_result".
	Type constant.BashCodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r BetaBashCodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBashCodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBashCodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaBashCodeExecutionToolResultBlockParamContentUnion struct {
	OfRequestBashCodeExecutionToolResultError *BetaBashCodeExecutionToolResultErrorParam `json:",omitzero,inline"`
	OfRequestBashCodeExecutionResultBlock     *BetaBashCodeExecutionResultBlockParam     `json:",omitzero,inline"`
	paramUnion
}

func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestBashCodeExecutionToolResultError, u.OfRequestBashCodeExecutionResultBlock)
}
func (u *BetaBashCodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaBashCodeExecutionToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestBashCodeExecutionToolResultError) {
		return u.OfRequestBashCodeExecutionToolResultError
	} else if !param.IsOmitted(u.OfRequestBashCodeExecutionResultBlock) {
		return u.OfRequestBashCodeExecutionResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestBashCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetContent() []BetaBashCodeExecutionOutputBlockParam {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetReturnCode() *int64 {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.ReturnCode
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetStderr() *string {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.Stderr
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetStdout() *string {
	if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return &vt.Stdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaBashCodeExecutionToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestBashCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestBashCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BetaBashCodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "output_file_too_large".
	ErrorCode BetaBashCodeExecutionToolResultErrorErrorCode `json:"error_code,required"`
	Type      constant.BashCodeExecutionToolResultError     `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ErrorCode   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaBashCodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BetaBashCodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBashCodeExecutionToolResultErrorErrorCode string

const (
	BetaBashCodeExecutionToolResultErrorErrorCodeInvalidToolInput      BetaBashCodeExecutionToolResultErrorErrorCode = "invalid_tool_input"
	BetaBashCodeExecutionToolResultErrorErrorCodeUnavailable           BetaBashCodeExecutionToolResultErrorErrorCode = "unavailable"
	BetaBashCodeExecutionToolResultErrorErrorCodeTooManyRequests       BetaBashCodeExecutionToolResultErrorErrorCode = "too_many_requests"
	BetaBashCodeExecutionToolResultErrorErrorCodeExecutionTimeExceeded BetaBashCodeExecutionToolResultErrorErrorCode = "execution_time_exceeded"
	BetaBashCodeExecutionToolResultErrorErrorCodeOutputFileTooLarge    BetaBashCodeExecutionToolResultErrorErrorCode = "output_file_too_large"
)

// The properties ErrorCode, Type are required.
type BetaBashCodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "output_file_too_large".
	ErrorCode BetaBashCodeExecutionToolResultErrorParamErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "bash_code_execution_tool_result_error".
	Type constant.BashCodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r BetaBashCodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaBashCodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaBashCodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaBashCodeExecutionToolResultErrorParamErrorCode string

const (
	BetaBashCodeExecutionToolResultErrorParamErrorCodeInvalidToolInput      BetaBashCodeExecutionToolResultErrorParamErrorCode = "invalid_tool_input"
	BetaBashCodeExecutionToolResultErrorParamErrorCodeUnavailable           BetaBashCodeExecutionToolResultErrorParamErrorCode = "unavailable"
	BetaBashCodeExecutionToolResultErrorParamErrorCodeTooManyRequests       BetaBashCodeExecutionToolResultErrorParamErrorCode = "too_many_requests"
	BetaBashCodeExecutionToolResultErrorParamErrorCodeExecutionTimeExceeded BetaBashCodeExecutionToolResultErrorParamErrorCode = "execution_time_exceeded"
	BetaBashCodeExecutionToolResultErrorParamErrorCodeOutputFileTooLarge    BetaBashCodeExecutionToolResultErrorParamErrorCode = "output_file_too_large"
)

func NewBetaCacheControlEphemeralParam() BetaCacheControlEphemeralParam {
	return BetaCacheControlEphemeralParam{
		Type: "ephemeral",
	}
}

// This struct has a constant value, construct it with
// [NewBetaCacheControlEphemeralParam].
type BetaCacheControlEphemeralParam struct {
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
	TTL  BetaCacheControlEphemeralTTL `json:"ttl,omitzero"`
	Type constant.Ephemeral           `json:"type,required"`
	paramObj
}

func (r BetaCacheControlEphemeralParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCacheControlEphemeralParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCacheControlEphemeralParam) UnmarshalJSON(data []byte) error {
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
type BetaCacheControlEphemeralTTL string

const (
	BetaCacheControlEphemeralTTLTTL5m BetaCacheControlEphemeralTTL = "5m"
	BetaCacheControlEphemeralTTLTTL1h BetaCacheControlEphemeralTTL = "1h"
)

type BetaCacheCreation struct {
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
func (r BetaCacheCreation) RawJSON() string { return r.JSON.raw }
func (r *BetaCacheCreation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationCharLocation struct {
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
func (r BetaCitationCharLocation) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationCharLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndCharIndex,
// StartCharIndex, Type are required.
type BetaCitationCharLocationParam struct {
	DocumentTitle  param.Opt[string] `json:"document_title,omitzero,required"`
	CitedText      string            `json:"cited_text,required"`
	DocumentIndex  int64             `json:"document_index,required"`
	EndCharIndex   int64             `json:"end_char_index,required"`
	StartCharIndex int64             `json:"start_char_index,required"`
	// This field can be elided, and will marshal its zero value as "char_location".
	Type constant.CharLocation `json:"type,required"`
	paramObj
}

func (r BetaCitationCharLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationCharLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationCharLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationConfig struct {
	Enabled bool `json:"enabled,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Enabled     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCitationConfig) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationContentBlockLocation struct {
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
func (r BetaCitationContentBlockLocation) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationContentBlockLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndBlockIndex,
// StartBlockIndex, Type are required.
type BetaCitationContentBlockLocationParam struct {
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

func (r BetaCitationContentBlockLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationContentBlockLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationContentBlockLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationPageLocation struct {
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
func (r BetaCitationPageLocation) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationPageLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, DocumentIndex, DocumentTitle, EndPageNumber,
// StartPageNumber, Type are required.
type BetaCitationPageLocationParam struct {
	DocumentTitle   param.Opt[string] `json:"document_title,omitzero,required"`
	CitedText       string            `json:"cited_text,required"`
	DocumentIndex   int64             `json:"document_index,required"`
	EndPageNumber   int64             `json:"end_page_number,required"`
	StartPageNumber int64             `json:"start_page_number,required"`
	// This field can be elided, and will marshal its zero value as "page_location".
	Type constant.PageLocation `json:"type,required"`
	paramObj
}

func (r BetaCitationPageLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationPageLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationPageLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationSearchResultLocation struct {
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
func (r BetaCitationSearchResultLocation) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationSearchResultLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, EndBlockIndex, SearchResultIndex, Source,
// StartBlockIndex, Title, Type are required.
type BetaCitationSearchResultLocationParam struct {
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

func (r BetaCitationSearchResultLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationSearchResultLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationSearchResultLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties CitedText, EncryptedIndex, Title, Type, URL are required.
type BetaCitationWebSearchResultLocationParam struct {
	Title          param.Opt[string] `json:"title,omitzero,required"`
	CitedText      string            `json:"cited_text,required"`
	EncryptedIndex string            `json:"encrypted_index,required"`
	URL            string            `json:"url,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_result_location".
	Type constant.WebSearchResultLocation `json:"type,required"`
	paramObj
}

func (r BetaCitationWebSearchResultLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationWebSearchResultLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationWebSearchResultLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationsConfigParam struct {
	Enabled param.Opt[bool] `json:"enabled,omitzero"`
	paramObj
}

func (r BetaCitationsConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCitationsConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCitationsConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationsDelta struct {
	Citation BetaCitationsDeltaCitationUnion `json:"citation,required"`
	Type     constant.CitationsDelta         `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Citation    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCitationsDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationsDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaCitationsDeltaCitationUnion contains all possible properties and values from
// [BetaCitationCharLocation], [BetaCitationPageLocation],
// [BetaCitationContentBlockLocation], [BetaCitationsWebSearchResultLocation],
// [BetaCitationSearchResultLocation].
//
// Use the [BetaCitationsDeltaCitationUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaCitationsDeltaCitationUnion struct {
	CitedText     string `json:"cited_text"`
	DocumentIndex int64  `json:"document_index"`
	DocumentTitle string `json:"document_title"`
	// This field is from variant [BetaCitationCharLocation].
	EndCharIndex int64  `json:"end_char_index"`
	FileID       string `json:"file_id"`
	// This field is from variant [BetaCitationCharLocation].
	StartCharIndex int64 `json:"start_char_index"`
	// Any of "char_location", "page_location", "content_block_location",
	// "web_search_result_location", "search_result_location".
	Type string `json:"type"`
	// This field is from variant [BetaCitationPageLocation].
	EndPageNumber int64 `json:"end_page_number"`
	// This field is from variant [BetaCitationPageLocation].
	StartPageNumber int64 `json:"start_page_number"`
	EndBlockIndex   int64 `json:"end_block_index"`
	StartBlockIndex int64 `json:"start_block_index"`
	// This field is from variant [BetaCitationsWebSearchResultLocation].
	EncryptedIndex string `json:"encrypted_index"`
	Title          string `json:"title"`
	// This field is from variant [BetaCitationsWebSearchResultLocation].
	URL string `json:"url"`
	// This field is from variant [BetaCitationSearchResultLocation].
	SearchResultIndex int64 `json:"search_result_index"`
	// This field is from variant [BetaCitationSearchResultLocation].
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

// anyBetaCitationsDeltaCitation is implemented by each variant of
// [BetaCitationsDeltaCitationUnion] to add type safety for the return type of
// [BetaCitationsDeltaCitationUnion.AsAny]
type anyBetaCitationsDeltaCitation interface {
	implBetaCitationsDeltaCitationUnion()
}

func (BetaCitationCharLocation) implBetaCitationsDeltaCitationUnion()             {}
func (BetaCitationPageLocation) implBetaCitationsDeltaCitationUnion()             {}
func (BetaCitationContentBlockLocation) implBetaCitationsDeltaCitationUnion()     {}
func (BetaCitationsWebSearchResultLocation) implBetaCitationsDeltaCitationUnion() {}
func (BetaCitationSearchResultLocation) implBetaCitationsDeltaCitationUnion()     {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaCitationsDeltaCitationUnion.AsAny().(type) {
//	case anthropic.BetaCitationCharLocation:
//	case anthropic.BetaCitationPageLocation:
//	case anthropic.BetaCitationContentBlockLocation:
//	case anthropic.BetaCitationsWebSearchResultLocation:
//	case anthropic.BetaCitationSearchResultLocation:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaCitationsDeltaCitationUnion) AsAny() anyBetaCitationsDeltaCitation {
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

func (u BetaCitationsDeltaCitationUnion) AsCharLocation() (v BetaCitationCharLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCitationsDeltaCitationUnion) AsPageLocation() (v BetaCitationPageLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCitationsDeltaCitationUnion) AsContentBlockLocation() (v BetaCitationContentBlockLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCitationsDeltaCitationUnion) AsWebSearchResultLocation() (v BetaCitationsWebSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCitationsDeltaCitationUnion) AsSearchResultLocation() (v BetaCitationSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaCitationsDeltaCitationUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaCitationsDeltaCitationUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCitationsWebSearchResultLocation struct {
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
func (r BetaCitationsWebSearchResultLocation) RawJSON() string { return r.JSON.raw }
func (r *BetaCitationsWebSearchResultLocation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Type is required.
type BetaClearThinking20251015EditParam struct {
	// Number of most recent assistant turns to keep thinking blocks for. Older turns
	// will have their thinking blocks removed.
	Keep BetaClearThinking20251015EditKeepUnionParam `json:"keep,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "clear_thinking_20251015".
	Type constant.ClearThinking20251015 `json:"type,required"`
	paramObj
}

func (r BetaClearThinking20251015EditParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaClearThinking20251015EditParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaClearThinking20251015EditParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaClearThinking20251015EditKeepUnionParam struct {
	OfThinkingTurns    *BetaThinkingTurnsParam    `json:",omitzero,inline"`
	OfAllThinkingTurns *BetaAllThinkingTurnsParam `json:",omitzero,inline"`
	// Construct this variant with constant.ValueOf[constant.All]()
	OfAll constant.All `json:",omitzero,inline"`
	paramUnion
}

func (u BetaClearThinking20251015EditKeepUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfThinkingTurns, u.OfAllThinkingTurns, u.OfAll)
}
func (u *BetaClearThinking20251015EditKeepUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaClearThinking20251015EditKeepUnionParam) asAny() any {
	if !param.IsOmitted(u.OfThinkingTurns) {
		return u.OfThinkingTurns
	} else if !param.IsOmitted(u.OfAllThinkingTurns) {
		return u.OfAllThinkingTurns
	} else if !param.IsOmitted(u.OfAll) {
		return &u.OfAll
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaClearThinking20251015EditKeepUnionParam) GetValue() *int64 {
	if vt := u.OfThinkingTurns; vt != nil {
		return &vt.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaClearThinking20251015EditKeepUnionParam) GetType() *string {
	if vt := u.OfThinkingTurns; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAllThinkingTurns; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BetaClearThinking20251015EditResponse struct {
	// Number of input tokens cleared by this edit.
	ClearedInputTokens int64 `json:"cleared_input_tokens,required"`
	// Number of thinking turns that were cleared.
	ClearedThinkingTurns int64 `json:"cleared_thinking_turns,required"`
	// The type of context management edit applied.
	Type constant.ClearThinking20251015 `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClearedInputTokens   respjson.Field
		ClearedThinkingTurns respjson.Field
		Type                 respjson.Field
		ExtraFields          map[string]respjson.Field
		raw                  string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaClearThinking20251015EditResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaClearThinking20251015EditResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Type is required.
type BetaClearToolUses20250919EditParam struct {
	// Whether to clear all tool inputs (bool) or specific tool inputs to clear (list)
	ClearToolInputs BetaClearToolUses20250919EditClearToolInputsUnionParam `json:"clear_tool_inputs,omitzero"`
	// Tool names whose uses are preserved from clearing
	ExcludeTools []string `json:"exclude_tools,omitzero"`
	// Minimum number of tokens that must be cleared when triggered. Context will only
	// be modified if at least this many tokens can be removed.
	ClearAtLeast BetaInputTokensClearAtLeastParam `json:"clear_at_least,omitzero"`
	// Number of tool uses to retain in the conversation
	Keep BetaToolUsesKeepParam `json:"keep,omitzero"`
	// Condition that triggers the context management strategy
	Trigger BetaClearToolUses20250919EditTriggerUnionParam `json:"trigger,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "clear_tool_uses_20250919".
	Type constant.ClearToolUses20250919 `json:"type,required"`
	paramObj
}

func (r BetaClearToolUses20250919EditParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaClearToolUses20250919EditParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaClearToolUses20250919EditParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaClearToolUses20250919EditClearToolInputsUnionParam struct {
	OfBool        param.Opt[bool] `json:",omitzero,inline"`
	OfStringArray []string        `json:",omitzero,inline"`
	paramUnion
}

func (u BetaClearToolUses20250919EditClearToolInputsUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBool, u.OfStringArray)
}
func (u *BetaClearToolUses20250919EditClearToolInputsUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaClearToolUses20250919EditClearToolInputsUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBool) {
		return &u.OfBool.Value
	} else if !param.IsOmitted(u.OfStringArray) {
		return &u.OfStringArray
	}
	return nil
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaClearToolUses20250919EditTriggerUnionParam struct {
	OfInputTokens *BetaInputTokensTriggerParam `json:",omitzero,inline"`
	OfToolUses    *BetaToolUsesTriggerParam    `json:",omitzero,inline"`
	paramUnion
}

func (u BetaClearToolUses20250919EditTriggerUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfInputTokens, u.OfToolUses)
}
func (u *BetaClearToolUses20250919EditTriggerUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaClearToolUses20250919EditTriggerUnionParam) asAny() any {
	if !param.IsOmitted(u.OfInputTokens) {
		return u.OfInputTokens
	} else if !param.IsOmitted(u.OfToolUses) {
		return u.OfToolUses
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaClearToolUses20250919EditTriggerUnionParam) GetType() *string {
	if vt := u.OfInputTokens; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfToolUses; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaClearToolUses20250919EditTriggerUnionParam) GetValue() *int64 {
	if vt := u.OfInputTokens; vt != nil {
		return (*int64)(&vt.Value)
	} else if vt := u.OfToolUses; vt != nil {
		return (*int64)(&vt.Value)
	}
	return nil
}

type BetaClearToolUses20250919EditResponse struct {
	// Number of input tokens cleared by this edit.
	ClearedInputTokens int64 `json:"cleared_input_tokens,required"`
	// Number of tool uses that were cleared.
	ClearedToolUses int64 `json:"cleared_tool_uses,required"`
	// The type of context management edit applied.
	Type constant.ClearToolUses20250919 `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClearedInputTokens respjson.Field
		ClearedToolUses    respjson.Field
		Type               respjson.Field
		ExtraFields        map[string]respjson.Field
		raw                string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaClearToolUses20250919EditResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaClearToolUses20250919EditResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCodeExecutionOutputBlock struct {
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
func (r BetaCodeExecutionOutputBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaCodeExecutionOutputBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type BetaCodeExecutionOutputBlockParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_output".
	Type constant.CodeExecutionOutput `json:"type,required"`
	paramObj
}

func (r BetaCodeExecutionOutputBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionOutputBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionOutputBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCodeExecutionResultBlock struct {
	Content    []BetaCodeExecutionOutputBlock `json:"content,required"`
	ReturnCode int64                          `json:"return_code,required"`
	Stderr     string                         `json:"stderr,required"`
	Stdout     string                         `json:"stdout,required"`
	Type       constant.CodeExecutionResult   `json:"type,required"`
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
func (r BetaCodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaCodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ReturnCode, Stderr, Stdout, Type are required.
type BetaCodeExecutionResultBlockParam struct {
	Content    []BetaCodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	ReturnCode int64                               `json:"return_code,required"`
	Stderr     string                              `json:"stderr,required"`
	Stdout     string                              `json:"stdout,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_result".
	Type constant.CodeExecutionResult `json:"type,required"`
	paramObj
}

func (r BetaCodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaCodeExecutionTool20250522Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
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

func (r BetaCodeExecutionTool20250522Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionTool20250522Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionTool20250522Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaCodeExecutionTool20250825Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
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

func (r BetaCodeExecutionTool20250825Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionTool20250825Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionTool20250825Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Code execution tool with REPL state persistence (daemon mode + gVisor
// checkpoint).
//
// The properties Name, Type are required.
type BetaCodeExecutionTool20260120Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
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

func (r BetaCodeExecutionTool20260120Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionTool20260120Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionTool20260120Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCodeExecutionToolResultBlock struct {
	// Code execution result with encrypted stdout for PFC + web_search results.
	Content   BetaCodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                       `json:"tool_use_id,required"`
	Type      constant.CodeExecutionToolResult             `json:"type,required"`
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
func (r BetaCodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaCodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaCodeExecutionToolResultBlockContentUnion contains all possible properties
// and values from [BetaCodeExecutionToolResultError],
// [BetaCodeExecutionResultBlock], [BetaEncryptedCodeExecutionResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaCodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [BetaCodeExecutionToolResultError].
	ErrorCode  BetaCodeExecutionToolResultErrorCode `json:"error_code"`
	Type       string                               `json:"type"`
	Content    []BetaCodeExecutionOutputBlock       `json:"content"`
	ReturnCode int64                                `json:"return_code"`
	Stderr     string                               `json:"stderr"`
	// This field is from variant [BetaCodeExecutionResultBlock].
	Stdout string `json:"stdout"`
	// This field is from variant [BetaEncryptedCodeExecutionResultBlock].
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

func (u BetaCodeExecutionToolResultBlockContentUnion) AsResponseCodeExecutionToolResultError() (v BetaCodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCodeExecutionToolResultBlockContentUnion) AsResponseCodeExecutionResultBlock() (v BetaCodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaCodeExecutionToolResultBlockContentUnion) AsResponseEncryptedCodeExecutionResultBlock() (v BetaEncryptedCodeExecutionResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaCodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaCodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaCodeExecutionToolResultBlockParam struct {
	// Code execution result with encrypted stdout for PFC + web_search results.
	Content   BetaCodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                            `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_tool_result".
	Type constant.CodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r BetaCodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func BetaNewCodeExecutionToolRequestError(errorCode BetaCodeExecutionToolResultErrorCode) BetaCodeExecutionToolResultBlockParamContentUnion {
	var variant BetaCodeExecutionToolResultErrorParam
	variant.ErrorCode = errorCode
	return BetaCodeExecutionToolResultBlockParamContentUnion{OfError: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaCodeExecutionToolResultBlockParamContentUnion struct {
	OfError                                    *BetaCodeExecutionToolResultErrorParam      `json:",omitzero,inline"`
	OfResultBlock                              *BetaCodeExecutionResultBlockParam          `json:",omitzero,inline"`
	OfRequestEncryptedCodeExecutionResultBlock *BetaEncryptedCodeExecutionResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaCodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfError, u.OfResultBlock, u.OfRequestEncryptedCodeExecutionResultBlock)
}
func (u *BetaCodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaCodeExecutionToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfError) {
		return u.OfError
	} else if !param.IsOmitted(u.OfResultBlock) {
		return u.OfResultBlock
	} else if !param.IsOmitted(u.OfRequestEncryptedCodeExecutionResultBlock) {
		return u.OfRequestEncryptedCodeExecutionResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetStdout() *string {
	if vt := u.OfResultBlock; vt != nil {
		return &vt.Stdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetEncryptedStdout() *string {
	if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return &vt.EncryptedStdout
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfResultBlock; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetReturnCode() *int64 {
	if vt := u.OfResultBlock; vt != nil {
		return (*int64)(&vt.ReturnCode)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*int64)(&vt.ReturnCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetStderr() *string {
	if vt := u.OfResultBlock; vt != nil {
		return (*string)(&vt.Stderr)
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return (*string)(&vt.Stderr)
	}
	return nil
}

// Returns a pointer to the underlying variant's Content property, if present.
func (u BetaCodeExecutionToolResultBlockParamContentUnion) GetContent() []BetaCodeExecutionOutputBlockParam {
	if vt := u.OfResultBlock; vt != nil {
		return vt.Content
	} else if vt := u.OfRequestEncryptedCodeExecutionResultBlock; vt != nil {
		return vt.Content
	}
	return nil
}

type BetaCodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode BetaCodeExecutionToolResultErrorCode  `json:"error_code,required"`
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
func (r BetaCodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BetaCodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCodeExecutionToolResultErrorCode string

const (
	BetaCodeExecutionToolResultErrorCodeInvalidToolInput      BetaCodeExecutionToolResultErrorCode = "invalid_tool_input"
	BetaCodeExecutionToolResultErrorCodeUnavailable           BetaCodeExecutionToolResultErrorCode = "unavailable"
	BetaCodeExecutionToolResultErrorCodeTooManyRequests       BetaCodeExecutionToolResultErrorCode = "too_many_requests"
	BetaCodeExecutionToolResultErrorCodeExecutionTimeExceeded BetaCodeExecutionToolResultErrorCode = "execution_time_exceeded"
)

// The properties ErrorCode, Type are required.
type BetaCodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode BetaCodeExecutionToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_tool_result_error".
	Type constant.CodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r BetaCodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Automatically compact older context when reaching the configured trigger
// threshold.
//
// The property Type is required.
type BetaCompact20260112EditParam struct {
	// Additional instructions for summarization.
	Instructions param.Opt[string] `json:"instructions,omitzero"`
	// Whether to pause after compaction and return the compaction block to the user.
	PauseAfterCompaction param.Opt[bool] `json:"pause_after_compaction,omitzero"`
	// When to trigger compaction. Defaults to 150000 input tokens.
	Trigger BetaInputTokensTriggerParam `json:"trigger,omitzero"`
	// This field can be elided, and will marshal its zero value as "compact_20260112".
	Type constant.Compact20260112 `json:"type,required"`
	paramObj
}

func (r BetaCompact20260112EditParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCompact20260112EditParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCompact20260112EditParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A compaction block returned when autocompact is triggered.
//
// When content is None, it indicates the compaction failed to produce a valid
// summary (e.g., malformed output from the model). Clients may round-trip
// compaction blocks with null content; the server treats them as no-ops.
type BetaCompactionBlock struct {
	// Summary of compacted content, or null if compaction failed
	Content string              `json:"content,required"`
	Type    constant.Compaction `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCompactionBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaCompactionBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A compaction block containing summary of previous context.
//
// Users should round-trip these blocks from responses to subsequent requests to
// maintain context across compaction boundaries.
//
// When content is None, the block represents a failed compaction. The server
// treats these as no-ops. Empty string content is not allowed.
//
// The properties Content, Type are required.
type BetaCompactionBlockParam struct {
	// Summary of previously compacted content, or null if compaction failed
	Content param.Opt[string] `json:"content,omitzero,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "compaction".
	Type constant.Compaction `json:"type,required"`
	paramObj
}

func (r BetaCompactionBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaCompactionBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaCompactionBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCompactionContentBlockDelta struct {
	Content string                   `json:"content,required"`
	Type    constant.CompactionDelta `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCompactionContentBlockDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaCompactionContentBlockDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Token usage for a compaction iteration.
type BetaCompactionIterationUsage struct {
	// Breakdown of cached tokens by TTL
	CacheCreation BetaCacheCreation `json:"cache_creation,required"`
	// The number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// The number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// Usage for a compaction iteration
	Type constant.Compaction `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreation            respjson.Field
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InputTokens              respjson.Field
		OutputTokens             respjson.Field
		Type                     respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCompactionIterationUsage) RawJSON() string { return r.JSON.raw }
func (r *BetaCompactionIterationUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Information about the container used in the request (for the code execution
// tool)
type BetaContainer struct {
	// Identifier for the container used in this request
	ID string `json:"id,required"`
	// The time at which the container will expire.
	ExpiresAt time.Time `json:"expires_at,required" format:"date-time"`
	// Skills loaded in the container
	Skills []BetaSkill `json:"skills,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		ExpiresAt   respjson.Field
		Skills      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaContainer) RawJSON() string { return r.JSON.raw }
func (r *BetaContainer) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Container parameters with skills to be loaded.
type BetaContainerParams struct {
	// Container id
	ID param.Opt[string] `json:"id,omitzero"`
	// List of skills to load in the container
	Skills []BetaSkillParams `json:"skills,omitzero"`
	paramObj
}

func (r BetaContainerParams) MarshalJSON() (data []byte, err error) {
	type shadow BetaContainerParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaContainerParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Response model for a file uploaded to the container.
type BetaContainerUploadBlock struct {
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
func (r BetaContainerUploadBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaContainerUploadBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A content block that represents a file to be uploaded to the container Files
// uploaded via this block will be available in the container's input directory.
//
// The properties FileID, Type are required.
type BetaContainerUploadBlockParam struct {
	FileID string `json:"file_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "container_upload".
	Type constant.ContainerUpload `json:"type,required"`
	paramObj
}

func (r BetaContainerUploadBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaContainerUploadBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaContainerUploadBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaContentBlockUnion contains all possible properties and values from
// [BetaTextBlock], [BetaThinkingBlock], [BetaRedactedThinkingBlock],
// [BetaToolUseBlock], [BetaServerToolUseBlock], [BetaWebSearchToolResultBlock],
// [BetaWebFetchToolResultBlock], [BetaCodeExecutionToolResultBlock],
// [BetaBashCodeExecutionToolResultBlock],
// [BetaTextEditorCodeExecutionToolResultBlock], [BetaToolSearchToolResultBlock],
// [BetaMCPToolUseBlock], [BetaMCPToolResultBlock], [BetaContainerUploadBlock],
// [BetaCompactionBlock].
//
// Use the [BetaContentBlockUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaContentBlockUnion struct {
	// This field is from variant [BetaTextBlock].
	Citations []BetaTextCitationUnion `json:"citations"`
	// This field is from variant [BetaTextBlock].
	Text string `json:"text"`
	// Any of "text", "thinking", "redacted_thinking", "tool_use", "server_tool_use",
	// "web_search_tool_result", "web_fetch_tool_result", "code_execution_tool_result",
	// "bash_code_execution_tool_result", "text_editor_code_execution_tool_result",
	// "tool_search_tool_result", "mcp_tool_use", "mcp_tool_result",
	// "container_upload", "compaction".
	Type string `json:"type"`
	// This field is from variant [BetaThinkingBlock].
	Signature string `json:"signature"`
	// This field is from variant [BetaThinkingBlock].
	Thinking string `json:"thinking"`
	// This field is from variant [BetaRedactedThinkingBlock].
	Data string `json:"data"`
	ID   string `json:"id"`
	// necessary custom code modification
	Input json.RawMessage `json:"input"`
	Name  string          `json:"name"`
	// This field is a union of [BetaToolUseBlockCallerUnion],
	// [BetaServerToolUseBlockCallerUnion], [BetaWebSearchToolResultBlockCallerUnion],
	// [BetaWebFetchToolResultBlockCallerUnion]
	Caller BetaContentBlockUnionCaller `json:"caller"`
	// This field is a union of [BetaWebSearchToolResultBlockContentUnion],
	// [BetaWebFetchToolResultBlockContentUnion],
	// [BetaCodeExecutionToolResultBlockContentUnion],
	// [BetaBashCodeExecutionToolResultBlockContentUnion],
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion],
	// [BetaToolSearchToolResultBlockContentUnion],
	// [BetaMCPToolResultBlockContentUnion], [string]
	Content   BetaContentBlockUnionContent `json:"content"`
	ToolUseID string                       `json:"tool_use_id"`
	// This field is from variant [BetaMCPToolUseBlock].
	ServerName string `json:"server_name"`
	// This field is from variant [BetaMCPToolResultBlock].
	IsError bool `json:"is_error"`
	// This field is from variant [BetaContainerUploadBlock].
	FileID string `json:"file_id"`
	JSON   struct {
		Citations  respjson.Field
		Text       respjson.Field
		Type       respjson.Field
		Signature  respjson.Field
		Thinking   respjson.Field
		Data       respjson.Field
		ID         respjson.Field
		Input      respjson.Field
		Name       respjson.Field
		Caller     respjson.Field
		Content    respjson.Field
		ToolUseID  respjson.Field
		ServerName respjson.Field
		IsError    respjson.Field
		FileID     respjson.Field
		raw        string
	} `json:"-"`
}

// anyBetaContentBlock is implemented by each variant of [BetaContentBlockUnion] to
// add type safety for the return type of [BetaContentBlockUnion.AsAny]
type anyBetaContentBlock interface {
	implBetaContentBlockUnion()
	toParamUnion() BetaContentBlockParamUnion
}

func (BetaTextBlock) implBetaContentBlockUnion()                              {}
func (BetaThinkingBlock) implBetaContentBlockUnion()                          {}
func (BetaRedactedThinkingBlock) implBetaContentBlockUnion()                  {}
func (BetaToolUseBlock) implBetaContentBlockUnion()                           {}
func (BetaServerToolUseBlock) implBetaContentBlockUnion()                     {}
func (BetaWebSearchToolResultBlock) implBetaContentBlockUnion()               {}
func (BetaWebFetchToolResultBlock) implBetaContentBlockUnion()                {}
func (BetaCodeExecutionToolResultBlock) implBetaContentBlockUnion()           {}
func (BetaBashCodeExecutionToolResultBlock) implBetaContentBlockUnion()       {}
func (BetaTextEditorCodeExecutionToolResultBlock) implBetaContentBlockUnion() {}
func (BetaToolSearchToolResultBlock) implBetaContentBlockUnion()              {}
func (BetaMCPToolUseBlock) implBetaContentBlockUnion()                        {}
func (BetaMCPToolResultBlock) implBetaContentBlockUnion()                     {}
func (BetaContainerUploadBlock) implBetaContentBlockUnion()                   {}
func (BetaCompactionBlock) implBetaContentBlockUnion()                        {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaContentBlockUnion.AsAny().(type) {
//	case anthropic.BetaTextBlock:
//	case anthropic.BetaThinkingBlock:
//	case anthropic.BetaRedactedThinkingBlock:
//	case anthropic.BetaToolUseBlock:
//	case anthropic.BetaServerToolUseBlock:
//	case anthropic.BetaWebSearchToolResultBlock:
//	case anthropic.BetaWebFetchToolResultBlock:
//	case anthropic.BetaCodeExecutionToolResultBlock:
//	case anthropic.BetaBashCodeExecutionToolResultBlock:
//	case anthropic.BetaTextEditorCodeExecutionToolResultBlock:
//	case anthropic.BetaToolSearchToolResultBlock:
//	case anthropic.BetaMCPToolUseBlock:
//	case anthropic.BetaMCPToolResultBlock:
//	case anthropic.BetaContainerUploadBlock:
//	case anthropic.BetaCompactionBlock:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaContentBlockUnion) AsAny() anyBetaContentBlock {
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
	case "mcp_tool_use":
		return u.AsMCPToolUse()
	case "mcp_tool_result":
		return u.AsMCPToolResult()
	case "container_upload":
		return u.AsContainerUpload()
	case "compaction":
		return u.AsCompaction()
	}
	return nil
}

func (u BetaContentBlockUnion) AsText() (v BetaTextBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsThinking() (v BetaThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsRedactedThinking() (v BetaRedactedThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsToolUse() (v BetaToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsServerToolUse() (v BetaServerToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsWebSearchToolResult() (v BetaWebSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsWebFetchToolResult() (v BetaWebFetchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsCodeExecutionToolResult() (v BetaCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsBashCodeExecutionToolResult() (v BetaBashCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsTextEditorCodeExecutionToolResult() (v BetaTextEditorCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsToolSearchToolResult() (v BetaToolSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsMCPToolUse() (v BetaMCPToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsMCPToolResult() (v BetaMCPToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsContainerUpload() (v BetaContainerUploadBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContentBlockUnion) AsCompaction() (v BetaCompactionBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaContentBlockUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaContentBlockUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaContentBlockUnionCaller is an implicit subunion of [BetaContentBlockUnion].
// BetaContentBlockUnionCaller provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaContentBlockUnion].
type BetaContentBlockUnionCaller struct {
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

func (r *BetaContentBlockUnionCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaContentBlockUnionContent is an implicit subunion of [BetaContentBlockUnion].
// BetaContentBlockUnionContent provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBetaWebSearchResultBlockArray OfString
// OfBetaMCPToolResultBlockContent]
type BetaContentBlockUnionContent struct {
	// This field will be present if the value is a [[]BetaWebSearchResultBlock]
	// instead of an object.
	OfBetaWebSearchResultBlockArray []BetaWebSearchResultBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field will be present if the value is a [[]BetaTextBlock] instead of an
	// object.
	OfBetaMCPToolResultBlockContent []BetaTextBlock `json:",inline"`
	ErrorCode                       string          `json:"error_code"`
	Type                            string          `json:"type"`
	// This field is a union of [BetaDocumentBlock], [[]BetaCodeExecutionOutputBlock],
	// [[]BetaCodeExecutionOutputBlock], [[]BetaBashCodeExecutionOutputBlock], [string]
	Content BetaContentBlockUnionContentContent `json:"content"`
	// This field is from variant [BetaWebFetchToolResultBlockContentUnion].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [BetaWebFetchToolResultBlockContentUnion].
	URL        string `json:"url"`
	ReturnCode int64  `json:"return_code"`
	Stderr     string `json:"stderr"`
	Stdout     string `json:"stdout"`
	// This field is from variant [BetaCodeExecutionToolResultBlockContentUnion].
	EncryptedStdout string `json:"encrypted_stdout"`
	ErrorMessage    string `json:"error_message"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	FileType BetaTextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NumLines int64 `json:"num_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	StartLine int64 `json:"start_line"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	Lines []string `json:"lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NewLines int64 `json:"new_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NewStart int64 `json:"new_start"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	OldLines int64 `json:"old_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	OldStart int64 `json:"old_start"`
	// This field is from variant [BetaToolSearchToolResultBlockContentUnion].
	ToolReferences []BetaToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		OfBetaWebSearchResultBlockArray respjson.Field
		OfString                        respjson.Field
		OfBetaMCPToolResultBlockContent respjson.Field
		ErrorCode                       respjson.Field
		Type                            respjson.Field
		Content                         respjson.Field
		RetrievedAt                     respjson.Field
		URL                             respjson.Field
		ReturnCode                      respjson.Field
		Stderr                          respjson.Field
		Stdout                          respjson.Field
		EncryptedStdout                 respjson.Field
		ErrorMessage                    respjson.Field
		FileType                        respjson.Field
		NumLines                        respjson.Field
		StartLine                       respjson.Field
		TotalLines                      respjson.Field
		IsFileUpdate                    respjson.Field
		Lines                           respjson.Field
		NewLines                        respjson.Field
		NewStart                        respjson.Field
		OldLines                        respjson.Field
		OldStart                        respjson.Field
		ToolReferences                  respjson.Field
		raw                             string
	} `json:"-"`
}

func (r *BetaContentBlockUnionContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaContentBlockUnionContentContent is an implicit subunion of
// [BetaContentBlockUnion]. BetaContentBlockUnionContentContent provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfContent OfString]
type BetaContentBlockUnionContentContent struct {
	// This field will be present if the value is a [[]BetaCodeExecutionOutputBlock]
	// instead of an object.
	OfContent []BetaCodeExecutionOutputBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is from variant [BetaDocumentBlock].
	Citations BetaCitationConfig `json:"citations"`
	// This field is from variant [BetaDocumentBlock].
	Source BetaDocumentBlockSourceUnion `json:"source"`
	// This field is from variant [BetaDocumentBlock].
	Title string `json:"title"`
	// This field is from variant [BetaDocumentBlock].
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

func (r *BetaContentBlockUnionContentContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewBetaTextBlock(text string) BetaContentBlockParamUnion {
	var variant BetaTextBlockParam
	variant.Text = text
	return BetaContentBlockParamUnion{OfText: &variant}
}

func NewBetaImageBlock[
	T BetaBase64ImageSourceParam | BetaURLImageSourceParam | BetaFileImageSourceParam,
](source T) BetaContentBlockParamUnion {
	var image BetaImageBlockParam
	switch v := any(source).(type) {
	case BetaBase64ImageSourceParam:
		image.Source.OfBase64 = &v
	case BetaURLImageSourceParam:
		image.Source.OfURL = &v
	case BetaFileImageSourceParam:
		image.Source.OfFile = &v
	}
	return BetaContentBlockParamUnion{OfImage: &image}
}

func NewBetaDocumentBlock[
	T BetaBase64PDFSourceParam | BetaPlainTextSourceParam | BetaContentBlockSourceParam | BetaURLPDFSourceParam | BetaFileDocumentSourceParam,
](source T) BetaContentBlockParamUnion {
	var document BetaRequestDocumentBlockParam
	switch v := any(source).(type) {
	case BetaBase64PDFSourceParam:
		document.Source.OfBase64 = &v
	case BetaPlainTextSourceParam:
		document.Source.OfText = &v
	case BetaContentBlockSourceParam:
		document.Source.OfContent = &v
	case BetaURLPDFSourceParam:
		document.Source.OfURL = &v
	case BetaFileDocumentSourceParam:
		document.Source.OfFile = &v
	}
	return BetaContentBlockParamUnion{OfDocument: &document}
}

func NewBetaSearchResultBlock(content []BetaTextBlockParam, source string, title string) BetaContentBlockParamUnion {
	var searchResult BetaSearchResultBlockParam
	searchResult.Content = content
	searchResult.Source = source
	searchResult.Title = title
	return BetaContentBlockParamUnion{OfSearchResult: &searchResult}
}

func NewBetaThinkingBlock(signature string, thinking string) BetaContentBlockParamUnion {
	var variant BetaThinkingBlockParam
	variant.Signature = signature
	variant.Thinking = thinking
	return BetaContentBlockParamUnion{OfThinking: &variant}
}

func NewBetaRedactedThinkingBlock(data string) BetaContentBlockParamUnion {
	var redactedThinking BetaRedactedThinkingBlockParam
	redactedThinking.Data = data
	return BetaContentBlockParamUnion{OfRedactedThinking: &redactedThinking}
}

func NewBetaToolUseBlock(id string, input any, name string) BetaContentBlockParamUnion {
	var toolUse BetaToolUseBlockParam
	toolUse.ID = id
	toolUse.Input = input
	toolUse.Name = name
	return BetaContentBlockParamUnion{OfToolUse: &toolUse}
}

func NewBetaToolResultBlock(toolUseID string, content string, isError bool) BetaContentBlockParamUnion {
	var toolBlock BetaToolResultBlockParam
	toolBlock.ToolUseID = toolUseID
	toolBlock.Content = []BetaToolResultBlockParamContentUnion{
		{OfText: &BetaTextBlockParam{Text: content}},
	}
	toolBlock.IsError = Bool(isError)
	return BetaContentBlockParamUnion{OfToolResult: &toolBlock}
}

func NewBetaServerToolUseBlock(id string, input any, name BetaServerToolUseBlockParamName) BetaContentBlockParamUnion {
	var serverToolUse BetaServerToolUseBlockParam
	serverToolUse.ID = id
	serverToolUse.Input = input
	serverToolUse.Name = name
	return BetaContentBlockParamUnion{OfServerToolUse: &serverToolUse}
}

func NewBetaWebSearchToolResultBlock[
	T []BetaWebSearchResultBlockParam | BetaWebSearchToolRequestErrorParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var webSearchToolResult BetaWebSearchToolResultBlockParam
	switch v := any(content).(type) {
	case []BetaWebSearchResultBlockParam:
		webSearchToolResult.Content.OfResultBlock = v
	case BetaWebSearchToolRequestErrorParam:
		webSearchToolResult.Content.OfError = &v
	}
	webSearchToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfWebSearchToolResult: &webSearchToolResult}
}

func NewBetaWebFetchToolResultBlock[
	T BetaWebFetchToolResultErrorBlockParam | BetaWebFetchBlockParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var webFetchToolResult BetaWebFetchToolResultBlockParam
	switch v := any(content).(type) {
	case BetaWebFetchToolResultErrorBlockParam:
		webFetchToolResult.Content.OfRequestWebFetchToolResultError = &v
	case BetaWebFetchBlockParam:
		webFetchToolResult.Content.OfRequestWebFetchResultBlock = &v
	}
	webFetchToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfWebFetchToolResult: &webFetchToolResult}
}

func NewBetaCodeExecutionToolResultBlock[
	T BetaCodeExecutionToolResultErrorParam | BetaCodeExecutionResultBlockParam | BetaEncryptedCodeExecutionResultBlockParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var codeExecutionToolResult BetaCodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case BetaCodeExecutionToolResultErrorParam:
		codeExecutionToolResult.Content.OfError = &v
	case BetaCodeExecutionResultBlockParam:
		codeExecutionToolResult.Content.OfResultBlock = &v
	case BetaEncryptedCodeExecutionResultBlockParam:
		codeExecutionToolResult.Content.OfRequestEncryptedCodeExecutionResultBlock = &v
	}
	codeExecutionToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfCodeExecutionToolResult: &codeExecutionToolResult}
}

func NewBetaBashCodeExecutionToolResultBlock[
	T BetaBashCodeExecutionToolResultErrorParam | BetaBashCodeExecutionResultBlockParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var bashCodeExecutionToolResult BetaBashCodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case BetaBashCodeExecutionToolResultErrorParam:
		bashCodeExecutionToolResult.Content.OfRequestBashCodeExecutionToolResultError = &v
	case BetaBashCodeExecutionResultBlockParam:
		bashCodeExecutionToolResult.Content.OfRequestBashCodeExecutionResultBlock = &v
	}
	bashCodeExecutionToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfBashCodeExecutionToolResult: &bashCodeExecutionToolResult}
}

func NewBetaTextEditorCodeExecutionToolResultBlock[
	T BetaTextEditorCodeExecutionToolResultErrorParam | BetaTextEditorCodeExecutionViewResultBlockParam | BetaTextEditorCodeExecutionCreateResultBlockParam | BetaTextEditorCodeExecutionStrReplaceResultBlockParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var textEditorCodeExecutionToolResult BetaTextEditorCodeExecutionToolResultBlockParam
	switch v := any(content).(type) {
	case BetaTextEditorCodeExecutionToolResultErrorParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionToolResultError = &v
	case BetaTextEditorCodeExecutionViewResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionViewResultBlock = &v
	case BetaTextEditorCodeExecutionCreateResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionCreateResultBlock = &v
	case BetaTextEditorCodeExecutionStrReplaceResultBlockParam:
		textEditorCodeExecutionToolResult.Content.OfRequestTextEditorCodeExecutionStrReplaceResultBlock = &v
	}
	textEditorCodeExecutionToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfTextEditorCodeExecutionToolResult: &textEditorCodeExecutionToolResult}
}

func NewBetaToolSearchToolResultBlock[
	T BetaToolSearchToolResultErrorParam | BetaToolSearchToolSearchResultBlockParam,
](content T, toolUseID string) BetaContentBlockParamUnion {
	var toolSearchToolResult BetaToolSearchToolResultBlockParam
	switch v := any(content).(type) {
	case BetaToolSearchToolResultErrorParam:
		toolSearchToolResult.Content.OfRequestToolSearchToolResultError = &v
	case BetaToolSearchToolSearchResultBlockParam:
		toolSearchToolResult.Content.OfRequestToolSearchToolSearchResultBlock = &v
	}
	toolSearchToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfToolSearchToolResult: &toolSearchToolResult}
}

func NewBetaMCPToolResultBlock(toolUseID string) BetaContentBlockParamUnion {
	var mcpToolResult BetaRequestMCPToolResultBlockParam
	mcpToolResult.ToolUseID = toolUseID
	return BetaContentBlockParamUnion{OfMCPToolResult: &mcpToolResult}
}

func NewBetaContainerUploadBlock(fileID string) BetaContentBlockParamUnion {
	var containerUpload BetaContainerUploadBlockParam
	containerUpload.FileID = fileID
	return BetaContentBlockParamUnion{OfContainerUpload: &containerUpload}
}

func NewBetaCompactionBlock(content string) BetaContentBlockParamUnion {
	var compaction BetaCompactionBlockParam
	compaction.Content = param.NewOpt(content)
	return BetaContentBlockParamUnion{OfCompaction: &compaction}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaContentBlockParamUnion struct {
	OfText                              *BetaTextBlockParam                              `json:",omitzero,inline"`
	OfImage                             *BetaImageBlockParam                             `json:",omitzero,inline"`
	OfDocument                          *BetaRequestDocumentBlockParam                   `json:",omitzero,inline"`
	OfSearchResult                      *BetaSearchResultBlockParam                      `json:",omitzero,inline"`
	OfThinking                          *BetaThinkingBlockParam                          `json:",omitzero,inline"`
	OfRedactedThinking                  *BetaRedactedThinkingBlockParam                  `json:",omitzero,inline"`
	OfToolUse                           *BetaToolUseBlockParam                           `json:",omitzero,inline"`
	OfToolResult                        *BetaToolResultBlockParam                        `json:",omitzero,inline"`
	OfServerToolUse                     *BetaServerToolUseBlockParam                     `json:",omitzero,inline"`
	OfWebSearchToolResult               *BetaWebSearchToolResultBlockParam               `json:",omitzero,inline"`
	OfWebFetchToolResult                *BetaWebFetchToolResultBlockParam                `json:",omitzero,inline"`
	OfCodeExecutionToolResult           *BetaCodeExecutionToolResultBlockParam           `json:",omitzero,inline"`
	OfBashCodeExecutionToolResult       *BetaBashCodeExecutionToolResultBlockParam       `json:",omitzero,inline"`
	OfTextEditorCodeExecutionToolResult *BetaTextEditorCodeExecutionToolResultBlockParam `json:",omitzero,inline"`
	OfToolSearchToolResult              *BetaToolSearchToolResultBlockParam              `json:",omitzero,inline"`
	OfMCPToolUse                        *BetaMCPToolUseBlockParam                        `json:",omitzero,inline"`
	OfMCPToolResult                     *BetaRequestMCPToolResultBlockParam              `json:",omitzero,inline"`
	OfContainerUpload                   *BetaContainerUploadBlockParam                   `json:",omitzero,inline"`
	OfCompaction                        *BetaCompactionBlockParam                        `json:",omitzero,inline"`
	paramUnion
}

func (u BetaContentBlockParamUnion) MarshalJSON() ([]byte, error) {
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
		u.OfMCPToolUse,
		u.OfMCPToolResult,
		u.OfContainerUpload,
		u.OfCompaction)
}
func (u *BetaContentBlockParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaContentBlockParamUnion) asAny() any {
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
	} else if !param.IsOmitted(u.OfMCPToolUse) {
		return u.OfMCPToolUse
	} else if !param.IsOmitted(u.OfMCPToolResult) {
		return u.OfMCPToolResult
	} else if !param.IsOmitted(u.OfContainerUpload) {
		return u.OfContainerUpload
	} else if !param.IsOmitted(u.OfCompaction) {
		return u.OfCompaction
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetText() *string {
	if vt := u.OfText; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetContext() *string {
	if vt := u.OfDocument; vt != nil && vt.Context.Valid() {
		return &vt.Context.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetSignature() *string {
	if vt := u.OfThinking; vt != nil {
		return &vt.Signature
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetThinking() *string {
	if vt := u.OfThinking; vt != nil {
		return &vt.Thinking
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetData() *string {
	if vt := u.OfRedactedThinking; vt != nil {
		return &vt.Data
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetServerName() *string {
	if vt := u.OfMCPToolUse; vt != nil {
		return &vt.ServerName
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetFileID() *string {
	if vt := u.OfContainerUpload; vt != nil {
		return &vt.FileID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetType() *string {
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
	} else if vt := u.OfMCPToolUse; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMCPToolResult; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContainerUpload; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCompaction; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetTitle() *string {
	if vt := u.OfDocument; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	} else if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Title)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetID() *string {
	if vt := u.OfToolUse; vt != nil {
		return (*string)(&vt.ID)
	} else if vt := u.OfServerToolUse; vt != nil {
		return (*string)(&vt.ID)
	} else if vt := u.OfMCPToolUse; vt != nil {
		return (*string)(&vt.ID)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetName() *string {
	if vt := u.OfToolUse; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfServerToolUse; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMCPToolUse; vt != nil {
		return (*string)(&vt.Name)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetToolUseID() *string {
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
	} else if vt := u.OfMCPToolResult; vt != nil {
		return (*string)(&vt.ToolUseID)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContentBlockParamUnion) GetIsError() *bool {
	if vt := u.OfToolResult; vt != nil && vt.IsError.Valid() {
		return &vt.IsError.Value
	} else if vt := u.OfMCPToolResult; vt != nil && vt.IsError.Valid() {
		return &vt.IsError.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u BetaContentBlockParamUnion) GetCacheControl() *BetaCacheControlEphemeralParam {
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
	} else if vt := u.OfMCPToolUse; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfMCPToolResult; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfContainerUpload; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCompaction; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaContentBlockParamUnion) GetCitations() (res betaContentBlockParamUnionCitations) {
	if vt := u.OfText; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfDocument; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Citations
	}
	return
}

// Can have the runtime types [*[]BetaTextCitationParamUnion],
// [*BetaCitationsConfigParam]
type betaContentBlockParamUnionCitations struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.BetaTextCitationParamUnion:
//	case *anthropic.BetaCitationsConfigParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContentBlockParamUnionCitations) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionCitations) GetEnabled() *bool {
	switch vt := u.any.(type) {
	case *BetaCitationsConfigParam:
		return paramutil.AddrIfPresent(vt.Enabled)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaContentBlockParamUnion) GetSource() (res betaContentBlockParamUnionSource) {
	if vt := u.OfImage; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfDocument; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Source
	}
	return
}

// Can have the runtime types [*BetaBase64ImageSourceParam],
// [*BetaURLImageSourceParam], [*BetaFileImageSourceParam],
// [*BetaBase64PDFSourceParam], [*BetaPlainTextSourceParam],
// [*BetaContentBlockSourceParam], [*BetaURLPDFSourceParam],
// [*BetaFileDocumentSourceParam], [*string]
type betaContentBlockParamUnionSource struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.BetaBase64ImageSourceParam:
//	case *anthropic.BetaURLImageSourceParam:
//	case *anthropic.BetaFileImageSourceParam:
//	case *anthropic.BetaBase64PDFSourceParam:
//	case *anthropic.BetaPlainTextSourceParam:
//	case *anthropic.BetaContentBlockSourceParam:
//	case *anthropic.BetaURLPDFSourceParam:
//	case *anthropic.BetaFileDocumentSourceParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContentBlockParamUnionSource) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetContent() *BetaContentBlockSourceContentUnionParam {
	switch vt := u.any.(type) {
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetContent()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetData() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetData()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetData()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetMediaType() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetMediaType()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetMediaType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetType() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetType()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetURL() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetURL()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetURL()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionSource) GetFileID() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetFileID()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetFileID()
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaContentBlockParamUnion) GetContent() (res betaContentBlockParamUnionContent) {
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
	} else if vt := u.OfMCPToolResult; vt != nil {
		res.any = vt.Content.asAny()
	} else if vt := u.OfCompaction; vt != nil && vt.Content.Valid() {
		res.any = &vt.Content.Value
	}
	return
}

// Can have the runtime types [_[]BetaTextBlockParam],
// [_[]BetaToolResultBlockParamContentUnion], [*[]BetaWebSearchResultBlockParam],
// [*BetaWebFetchToolResultErrorBlockParam], [*BetaWebFetchBlockParam],
// [*BetaCodeExecutionToolResultErrorParam], [*BetaCodeExecutionResultBlockParam],
// [*BetaEncryptedCodeExecutionResultBlockParam],
// [*BetaBashCodeExecutionToolResultErrorParam],
// [*BetaBashCodeExecutionResultBlockParam],
// [*BetaTextEditorCodeExecutionToolResultErrorParam],
// [*BetaTextEditorCodeExecutionViewResultBlockParam],
// [*BetaTextEditorCodeExecutionCreateResultBlockParam],
// [*BetaTextEditorCodeExecutionStrReplaceResultBlockParam],
// [*BetaToolSearchToolResultErrorParam],
// [*BetaToolSearchToolSearchResultBlockParam], [*string]
type betaContentBlockParamUnionContent struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.BetaTextBlockParam:
//	case *[]anthropic.BetaToolResultBlockParamContentUnion:
//	case *[]anthropic.BetaWebSearchResultBlockParam:
//	case *anthropic.BetaWebFetchToolResultErrorBlockParam:
//	case *anthropic.BetaWebFetchBlockParam:
//	case *anthropic.BetaCodeExecutionToolResultErrorParam:
//	case *anthropic.BetaCodeExecutionResultBlockParam:
//	case *anthropic.BetaEncryptedCodeExecutionResultBlockParam:
//	case *anthropic.BetaBashCodeExecutionToolResultErrorParam:
//	case *anthropic.BetaBashCodeExecutionResultBlockParam:
//	case *anthropic.BetaTextEditorCodeExecutionToolResultErrorParam:
//	case *anthropic.BetaTextEditorCodeExecutionViewResultBlockParam:
//	case *anthropic.BetaTextEditorCodeExecutionCreateResultBlockParam:
//	case *anthropic.BetaTextEditorCodeExecutionStrReplaceResultBlockParam:
//	case *anthropic.BetaToolSearchToolResultErrorParam:
//	case *anthropic.BetaToolSearchToolSearchResultBlockParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContentBlockParamUnionContent) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetURL() *string {
	switch vt := u.any.(type) {
	case *BetaWebFetchToolResultBlockParamContentUnion:
		return vt.GetURL()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetRetrievedAt() *string {
	switch vt := u.any.(type) {
	case *BetaWebFetchToolResultBlockParamContentUnion:
		return vt.GetRetrievedAt()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetEncryptedStdout() *string {
	switch vt := u.any.(type) {
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetEncryptedStdout()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetErrorMessage() *string {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorMessage()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetFileType() *string {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetFileType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetNumLines() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNumLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetStartLine() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStartLine()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetTotalLines() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetTotalLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetIsFileUpdate() *bool {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetIsFileUpdate()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetLines() []string {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetNewLines() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNewLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetNewStart() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetNewStart()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetOldLines() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetOldLines()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetOldStart() *int64 {
	switch vt := u.any.(type) {
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetOldStart()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetToolReferences() []BetaToolReferenceBlockParam {
	switch vt := u.any.(type) {
	case *BetaToolSearchToolResultBlockParamContentUnion:
		return vt.GetToolReferences()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetErrorCode() *string {
	switch vt := u.any.(type) {
	case *BetaWebSearchToolResultBlockParamContentUnion:
		if vt.OfError != nil {
			return (*string)(&vt.OfError.ErrorCode)
		}
	case *BetaWebFetchToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	case *BetaToolSearchToolResultBlockParamContentUnion:
		return vt.GetErrorCode()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetType() *string {
	switch vt := u.any.(type) {
	case *BetaWebSearchToolResultBlockParamContentUnion:
		if vt.OfError != nil {
			return (*string)(&vt.OfError.Type)
		}
	case *BetaWebFetchToolResultBlockParamContentUnion:
		return vt.GetType()
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetType()
	case *BetaToolSearchToolResultBlockParamContentUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetReturnCode() *int64 {
	switch vt := u.any.(type) {
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetReturnCode()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetReturnCode()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetStderr() *string {
	switch vt := u.any.(type) {
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStderr()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStderr()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionContent) GetStdout() *string {
	switch vt := u.any.(type) {
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStdout()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		return vt.GetStdout()
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u betaContentBlockParamUnionContent) GetContent() (res betaContentBlockParamUnionContentContent) {
	switch vt := u.any.(type) {
	case *BetaWebFetchToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *BetaCodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *BetaBashCodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	case *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion:
		res.any = vt.GetContent()
	}
	return res
}

// Can have the runtime types [*BetaRequestDocumentBlockParam],
// [_[]BetaCodeExecutionOutputBlockParam],
// [_[]BetaBashCodeExecutionOutputBlockParam], [*string]
type betaContentBlockParamUnionContentContent struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.BetaRequestDocumentBlockParam:
//	case *[]anthropic.BetaCodeExecutionOutputBlockParam:
//	case *[]anthropic.BetaBashCodeExecutionOutputBlockParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContentBlockParamUnionContentContent) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's Input property, if present.
func (u BetaContentBlockParamUnion) GetInput() *any {
	if vt := u.OfToolUse; vt != nil {
		return &vt.Input
	} else if vt := u.OfServerToolUse; vt != nil {
		return &vt.Input
	} else if vt := u.OfMCPToolUse; vt != nil {
		return &vt.Input
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaContentBlockParamUnion) GetCaller() (res betaContentBlockParamUnionCaller) {
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

// Can have the runtime types [*BetaDirectCallerParam],
// [*BetaServerToolCallerParam], [*BetaServerToolCaller20260120Param]
type betaContentBlockParamUnionCaller struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.BetaDirectCallerParam:
//	case *anthropic.BetaServerToolCallerParam:
//	case *anthropic.BetaServerToolCaller20260120Param:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContentBlockParamUnionCaller) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionCaller) GetType() *string {
	switch vt := u.any.(type) {
	case *BetaToolUseBlockParamCallerUnion:
		return vt.GetType()
	case *BetaServerToolUseBlockParamCallerUnion:
		return vt.GetType()
	case *BetaWebSearchToolResultBlockParamCallerUnion:
		return vt.GetType()
	case *BetaWebFetchToolResultBlockParamCallerUnion:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContentBlockParamUnionCaller) GetToolID() *string {
	switch vt := u.any.(type) {
	case *BetaToolUseBlockParamCallerUnion:
		return vt.GetToolID()
	case *BetaServerToolUseBlockParamCallerUnion:
		return vt.GetToolID()
	case *BetaWebSearchToolResultBlockParamCallerUnion:
		return vt.GetToolID()
	case *BetaWebFetchToolResultBlockParamCallerUnion:
		return vt.GetToolID()
	}
	return nil
}


// The properties Content, Type are required.
type BetaContentBlockSourceParam struct {
	Content BetaContentBlockSourceContentUnionParam `json:"content,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "content".
	Type constant.Content `json:"type,required"`
	paramObj
}

func (r BetaContentBlockSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaContentBlockSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaContentBlockSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaContentBlockSourceContentUnionParam struct {
	OfString                        param.Opt[string]                         `json:",omitzero,inline"`
	OfBetaContentBlockSourceContent []BetaContentBlockSourceContentUnionParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaContentBlockSourceContentUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfBetaContentBlockSourceContent)
}
func (u *BetaContentBlockSourceContentUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaContentBlockSourceContentUnionParam) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfBetaContentBlockSourceContent) {
		return &u.OfBetaContentBlockSourceContent
	}
	return nil
}

type BetaContextManagementConfigParam struct {
	// List of context management edits to apply
	Edits []BetaContextManagementConfigEditUnionParam `json:"edits,omitzero"`
	paramObj
}

func (r BetaContextManagementConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaContextManagementConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaContextManagementConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaContextManagementConfigEditUnionParam struct {
	OfClearToolUses20250919 *BetaClearToolUses20250919EditParam `json:",omitzero,inline"`
	OfClearThinking20251015 *BetaClearThinking20251015EditParam `json:",omitzero,inline"`
	OfCompact20260112       *BetaCompact20260112EditParam       `json:",omitzero,inline"`
	paramUnion
}

func (u BetaContextManagementConfigEditUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfClearToolUses20250919, u.OfClearThinking20251015, u.OfCompact20260112)
}
func (u *BetaContextManagementConfigEditUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaContextManagementConfigEditUnionParam) asAny() any {
	if !param.IsOmitted(u.OfClearToolUses20250919) {
		return u.OfClearToolUses20250919
	} else if !param.IsOmitted(u.OfClearThinking20251015) {
		return u.OfClearThinking20251015
	} else if !param.IsOmitted(u.OfCompact20260112) {
		return u.OfCompact20260112
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetClearAtLeast() *BetaInputTokensClearAtLeastParam {
	if vt := u.OfClearToolUses20250919; vt != nil {
		return &vt.ClearAtLeast
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetClearToolInputs() *BetaClearToolUses20250919EditClearToolInputsUnionParam {
	if vt := u.OfClearToolUses20250919; vt != nil {
		return &vt.ClearToolInputs
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetExcludeTools() []string {
	if vt := u.OfClearToolUses20250919; vt != nil {
		return vt.ExcludeTools
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetInstructions() *string {
	if vt := u.OfCompact20260112; vt != nil && vt.Instructions.Valid() {
		return &vt.Instructions.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetPauseAfterCompaction() *bool {
	if vt := u.OfCompact20260112; vt != nil && vt.PauseAfterCompaction.Valid() {
		return &vt.PauseAfterCompaction.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaContextManagementConfigEditUnionParam) GetType() *string {
	if vt := u.OfClearToolUses20250919; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfClearThinking20251015; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCompact20260112; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaContextManagementConfigEditUnionParam) GetKeep() (res betaContextManagementConfigEditUnionParamKeep) {
	if vt := u.OfClearToolUses20250919; vt != nil {
		res.any = &vt.Keep
	} else if vt := u.OfClearThinking20251015; vt != nil {
		res.any = vt.Keep.asAny()
	}
	return
}

// Can have the runtime types [*BetaToolUsesKeepParam], [*string]
type betaContextManagementConfigEditUnionParamKeep struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.BetaToolUsesKeepParam:
//	case *string:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaContextManagementConfigEditUnionParamKeep) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaContextManagementConfigEditUnionParamKeep) GetType() *string {
	switch vt := u.any.(type) {
	case *BetaToolUsesKeepParam:
		return (*string)(&vt.Type)
	case *BetaClearThinking20251015EditKeepUnionParam:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaContextManagementConfigEditUnionParamKeep) GetValue() *int64 {
	switch vt := u.any.(type) {
	case *BetaToolUsesKeepParam:
		return (*int64)(&vt.Value)
	case *BetaClearThinking20251015EditKeepUnionParam:
		return vt.GetValue()
	}
	return nil
}


type BetaContextManagementResponse struct {
	// List of context management edits that were applied.
	AppliedEdits []BetaContextManagementResponseAppliedEditUnion `json:"applied_edits,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		AppliedEdits respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaContextManagementResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaContextManagementResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaContextManagementResponseAppliedEditUnion contains all possible properties
// and values from [BetaClearToolUses20250919EditResponse],
// [BetaClearThinking20251015EditResponse].
//
// Use the [BetaContextManagementResponseAppliedEditUnion.AsAny] method to switch
// on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaContextManagementResponseAppliedEditUnion struct {
	ClearedInputTokens int64 `json:"cleared_input_tokens"`
	// This field is from variant [BetaClearToolUses20250919EditResponse].
	ClearedToolUses int64 `json:"cleared_tool_uses"`
	// Any of "clear_tool_uses_20250919", "clear_thinking_20251015".
	Type string `json:"type"`
	// This field is from variant [BetaClearThinking20251015EditResponse].
	ClearedThinkingTurns int64 `json:"cleared_thinking_turns"`
	JSON                 struct {
		ClearedInputTokens   respjson.Field
		ClearedToolUses      respjson.Field
		Type                 respjson.Field
		ClearedThinkingTurns respjson.Field
		raw                  string
	} `json:"-"`
}

// anyBetaContextManagementResponseAppliedEdit is implemented by each variant of
// [BetaContextManagementResponseAppliedEditUnion] to add type safety for the
// return type of [BetaContextManagementResponseAppliedEditUnion.AsAny]
type anyBetaContextManagementResponseAppliedEdit interface {
	implBetaContextManagementResponseAppliedEditUnion()
}

func (BetaClearToolUses20250919EditResponse) implBetaContextManagementResponseAppliedEditUnion() {}
func (BetaClearThinking20251015EditResponse) implBetaContextManagementResponseAppliedEditUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaContextManagementResponseAppliedEditUnion.AsAny().(type) {
//	case anthropic.BetaClearToolUses20250919EditResponse:
//	case anthropic.BetaClearThinking20251015EditResponse:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaContextManagementResponseAppliedEditUnion) AsAny() anyBetaContextManagementResponseAppliedEdit {
	switch u.Type {
	case "clear_tool_uses_20250919":
		return u.AsClearToolUses20250919()
	case "clear_thinking_20251015":
		return u.AsClearThinking20251015()
	}
	return nil
}

func (u BetaContextManagementResponseAppliedEditUnion) AsClearToolUses20250919() (v BetaClearToolUses20250919EditResponse) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaContextManagementResponseAppliedEditUnion) AsClearThinking20251015() (v BetaClearThinking20251015EditResponse) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaContextManagementResponseAppliedEditUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaContextManagementResponseAppliedEditUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaCountTokensContextManagementResponse struct {
	// The original token count before context management was applied
	OriginalInputTokens int64 `json:"original_input_tokens,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		OriginalInputTokens respjson.Field
		ExtraFields         map[string]respjson.Field
		raw                 string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaCountTokensContextManagementResponse) RawJSON() string { return r.JSON.raw }
func (r *BetaCountTokensContextManagementResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Tool invocation directly from the model.
type BetaDirectCaller struct {
	Type constant.Direct `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaDirectCaller) RawJSON() string { return r.JSON.raw }
func (r *BetaDirectCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BetaDirectCaller to a BetaDirectCallerParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BetaDirectCallerParam.Overrides()
func (r BetaDirectCaller) ToParam() BetaDirectCallerParam {
	return param.Override[BetaDirectCallerParam](json.RawMessage(r.RawJSON()))
}

func NewBetaDirectCallerParam() BetaDirectCallerParam {
	return BetaDirectCallerParam{
		Type: "direct",
	}
}

// Tool invocation directly from the model.
//
// This struct has a constant value, construct it with [NewBetaDirectCallerParam].
type BetaDirectCallerParam struct {
	Type constant.Direct `json:"type,required"`
	paramObj
}

func (r BetaDirectCallerParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaDirectCallerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaDirectCallerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaDocumentBlock struct {
	// Citation configuration for the document
	Citations BetaCitationConfig           `json:"citations,required"`
	Source    BetaDocumentBlockSourceUnion `json:"source,required"`
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
func (r BetaDocumentBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaDocumentBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaDocumentBlockSourceUnion contains all possible properties and values from
// [BetaBase64PDFSource], [BetaPlainTextSource].
//
// Use the [BetaDocumentBlockSourceUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaDocumentBlockSourceUnion struct {
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

// anyBetaDocumentBlockSource is implemented by each variant of
// [BetaDocumentBlockSourceUnion] to add type safety for the return type of
// [BetaDocumentBlockSourceUnion.AsAny]
type anyBetaDocumentBlockSource interface {
	implBetaDocumentBlockSourceUnion()
}

func (BetaBase64PDFSource) implBetaDocumentBlockSourceUnion() {}
func (BetaPlainTextSource) implBetaDocumentBlockSourceUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaDocumentBlockSourceUnion.AsAny().(type) {
//	case anthropic.BetaBase64PDFSource:
//	case anthropic.BetaPlainTextSource:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaDocumentBlockSourceUnion) AsAny() anyBetaDocumentBlockSource {
	switch u.Type {
	case "base64":
		return u.AsBase64()
	case "text":
		return u.AsText()
	}
	return nil
}

func (u BetaDocumentBlockSourceUnion) AsBase64() (v BetaBase64PDFSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaDocumentBlockSourceUnion) AsText() (v BetaPlainTextSource) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaDocumentBlockSourceUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaDocumentBlockSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Code execution result with encrypted stdout for PFC + web_search results.
type BetaEncryptedCodeExecutionResultBlock struct {
	Content         []BetaCodeExecutionOutputBlock        `json:"content,required"`
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
func (r BetaEncryptedCodeExecutionResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaEncryptedCodeExecutionResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Code execution result with encrypted stdout for PFC + web_search results.
//
// The properties Content, EncryptedStdout, ReturnCode, Stderr, Type are required.
type BetaEncryptedCodeExecutionResultBlockParam struct {
	Content         []BetaCodeExecutionOutputBlockParam `json:"content,omitzero,required"`
	EncryptedStdout string                              `json:"encrypted_stdout,required"`
	ReturnCode      int64                               `json:"return_code,required"`
	Stderr          string                              `json:"stderr,required"`
	// This field can be elided, and will marshal its zero value as
	// "encrypted_code_execution_result".
	Type constant.EncryptedCodeExecutionResult `json:"type,required"`
	paramObj
}

func (r BetaEncryptedCodeExecutionResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaEncryptedCodeExecutionResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaEncryptedCodeExecutionResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type BetaFileDocumentSourceParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as "file".
	Type constant.File `json:"type,required"`
	paramObj
}

func (r BetaFileDocumentSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaFileDocumentSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaFileDocumentSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties FileID, Type are required.
type BetaFileImageSourceParam struct {
	FileID string `json:"file_id,required"`
	// This field can be elided, and will marshal its zero value as "file".
	Type constant.File `json:"type,required"`
	paramObj
}

func (r BetaFileImageSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaFileImageSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaFileImageSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Source, Type are required.
type BetaImageBlockParam struct {
	Source BetaImageBlockParamSourceUnion `json:"source,omitzero,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "image".
	Type constant.Image `json:"type,required"`
	paramObj
}

func (r BetaImageBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaImageBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaImageBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaImageBlockParamSourceUnion struct {
	OfBase64 *BetaBase64ImageSourceParam `json:",omitzero,inline"`
	OfURL    *BetaURLImageSourceParam    `json:",omitzero,inline"`
	OfFile   *BetaFileImageSourceParam   `json:",omitzero,inline"`
	paramUnion
}

func (u BetaImageBlockParamSourceUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBase64, u.OfURL, u.OfFile)
}
func (u *BetaImageBlockParamSourceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaImageBlockParamSourceUnion) asAny() any {
	if !param.IsOmitted(u.OfBase64) {
		return u.OfBase64
	} else if !param.IsOmitted(u.OfURL) {
		return u.OfURL
	} else if !param.IsOmitted(u.OfFile) {
		return u.OfFile
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaImageBlockParamSourceUnion) GetData() *string {
	if vt := u.OfBase64; vt != nil {
		return &vt.Data
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaImageBlockParamSourceUnion) GetMediaType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.MediaType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaImageBlockParamSourceUnion) GetURL() *string {
	if vt := u.OfURL; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaImageBlockParamSourceUnion) GetFileID() *string {
	if vt := u.OfFile; vt != nil {
		return &vt.FileID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaImageBlockParamSourceUnion) GetType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfURL; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfFile; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BetaInputJSONDelta struct {
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
func (r BetaInputJSONDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaInputJSONDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, Value are required.
type BetaInputTokensClearAtLeastParam struct {
	Value int64 `json:"value,required"`
	// This field can be elided, and will marshal its zero value as "input_tokens".
	Type constant.InputTokens `json:"type,required"`
	paramObj
}

func (r BetaInputTokensClearAtLeastParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaInputTokensClearAtLeastParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaInputTokensClearAtLeastParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, Value are required.
type BetaInputTokensTriggerParam struct {
	Value int64 `json:"value,required"`
	// This field can be elided, and will marshal its zero value as "input_tokens".
	Type constant.InputTokens `json:"type,required"`
	paramObj
}

func (r BetaInputTokensTriggerParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaInputTokensTriggerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaInputTokensTriggerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaIterationsUsage []BetaIterationsUsageItemUnion

// BetaIterationsUsageItemUnion contains all possible properties and values from
// [BetaMessageIterationUsage], [BetaCompactionIterationUsage].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaIterationsUsageItemUnion struct {
	// This field is from variant [BetaMessageIterationUsage].
	CacheCreation            BetaCacheCreation `json:"cache_creation"`
	CacheCreationInputTokens int64             `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64             `json:"cache_read_input_tokens"`
	InputTokens              int64             `json:"input_tokens"`
	OutputTokens             int64             `json:"output_tokens"`
	Type                     string            `json:"type"`
	JSON                     struct {
		CacheCreation            respjson.Field
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InputTokens              respjson.Field
		OutputTokens             respjson.Field
		Type                     respjson.Field
		raw                      string
	} `json:"-"`
}

func (u BetaIterationsUsageItemUnion) AsMessageIterationUsage() (v BetaMessageIterationUsage) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaIterationsUsageItemUnion) AsCompactionIterationUsage() (v BetaCompactionIterationUsage) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaIterationsUsageItemUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaIterationsUsageItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Schema, Type are required.
type BetaJSONOutputFormatParam struct {
	// The JSON schema of the format
	Schema map[string]any `json:"schema,omitzero,required"`
	// This field can be elided, and will marshal its zero value as "json_schema".
	Type constant.JSONSchema `json:"type,required"`
	paramObj
}

func (r BetaJSONOutputFormatParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaJSONOutputFormatParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaJSONOutputFormatParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for a specific tool in an MCP toolset.
type BetaMCPToolConfigParam struct {
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	Enabled      param.Opt[bool] `json:"enabled,omitzero"`
	paramObj
}

func (r BetaMCPToolConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMCPToolConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMCPToolConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Default configuration for tools in an MCP toolset.
type BetaMCPToolDefaultConfigParam struct {
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	Enabled      param.Opt[bool] `json:"enabled,omitzero"`
	paramObj
}

func (r BetaMCPToolDefaultConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMCPToolDefaultConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMCPToolDefaultConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMCPToolResultBlock struct {
	Content   BetaMCPToolResultBlockContentUnion `json:"content,required"`
	IsError   bool                               `json:"is_error,required"`
	ToolUseID string                             `json:"tool_use_id,required"`
	Type      constant.MCPToolResult             `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		IsError     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMCPToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaMCPToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaMCPToolResultBlockContentUnion contains all possible properties and values
// from [string], [[]BetaTextBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfString OfBetaMCPToolResultBlockContent]
type BetaMCPToolResultBlockContentUnion struct {
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field will be present if the value is a [[]BetaTextBlock] instead of an
	// object.
	OfBetaMCPToolResultBlockContent []BetaTextBlock `json:",inline"`
	JSON                            struct {
		OfString                        respjson.Field
		OfBetaMCPToolResultBlockContent respjson.Field
		raw                             string
	} `json:"-"`
}

func (u BetaMCPToolResultBlockContentUnion) AsString() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMCPToolResultBlockContentUnion) AsBetaMCPToolResultBlockContent() (v []BetaTextBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaMCPToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaMCPToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMCPToolUseBlock struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,required"`
	// The name of the MCP tool
	Name string `json:"name,required"`
	// The name of the MCP server
	ServerName string              `json:"server_name,required"`
	Type       constant.MCPToolUse `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Input       respjson.Field
		Name        respjson.Field
		ServerName  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMCPToolUseBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaMCPToolUseBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ID, Input, Name, ServerName, Type are required.
type BetaMCPToolUseBlockParam struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,omitzero,required"`
	Name  string `json:"name,required"`
	// The name of the MCP server
	ServerName string `json:"server_name,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "mcp_tool_use".
	Type constant.MCPToolUse `json:"type,required"`
	paramObj
}

func (r BetaMCPToolUseBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMCPToolUseBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMCPToolUseBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for a group of tools from an MCP server.
//
// Allows configuring enabled status and defer_loading for all tools from an MCP
// server, with optional per-tool overrides.
//
// The properties MCPServerName, Type are required.
type BetaMCPToolsetParam struct {
	// Name of the MCP server to configure tools for
	MCPServerName string `json:"mcp_server_name,required"`
	// Configuration overrides for specific tools, keyed by tool name
	Configs map[string]BetaMCPToolConfigParam `json:"configs,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Default configuration applied to all tools from this server
	DefaultConfig BetaMCPToolDefaultConfigParam `json:"default_config,omitzero"`
	// This field can be elided, and will marshal its zero value as "mcp_toolset".
	Type constant.MCPToolset `json:"type,required"`
	paramObj
}

func (r BetaMCPToolsetParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMCPToolsetParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMCPToolsetParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaMemoryTool20250818Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
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

func (r BetaMemoryTool20250818Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaMemoryTool20250818Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMemoryTool20250818Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaMemoryTool20250818CommandUnion contains all possible properties and values
// from [BetaMemoryTool20250818ViewCommand], [BetaMemoryTool20250818CreateCommand],
// [BetaMemoryTool20250818StrReplaceCommand],
// [BetaMemoryTool20250818InsertCommand], [BetaMemoryTool20250818DeleteCommand],
// [BetaMemoryTool20250818RenameCommand].
//
// Use the [BetaMemoryTool20250818CommandUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaMemoryTool20250818CommandUnion struct {
	// Any of "view", "create", "str_replace", "insert", "delete", "rename".
	Command string `json:"command"`
	Path    string `json:"path"`
	// This field is from variant [BetaMemoryTool20250818ViewCommand].
	ViewRange []int64 `json:"view_range"`
	// This field is from variant [BetaMemoryTool20250818CreateCommand].
	FileText string `json:"file_text"`
	// This field is from variant [BetaMemoryTool20250818StrReplaceCommand].
	NewStr string `json:"new_str"`
	// This field is from variant [BetaMemoryTool20250818StrReplaceCommand].
	OldStr string `json:"old_str"`
	// This field is from variant [BetaMemoryTool20250818InsertCommand].
	InsertLine int64 `json:"insert_line"`
	// This field is from variant [BetaMemoryTool20250818InsertCommand].
	InsertText string `json:"insert_text"`
	// This field is from variant [BetaMemoryTool20250818RenameCommand].
	NewPath string `json:"new_path"`
	// This field is from variant [BetaMemoryTool20250818RenameCommand].
	OldPath string `json:"old_path"`
	JSON    struct {
		Command    respjson.Field
		Path       respjson.Field
		ViewRange  respjson.Field
		FileText   respjson.Field
		NewStr     respjson.Field
		OldStr     respjson.Field
		InsertLine respjson.Field
		InsertText respjson.Field
		NewPath    respjson.Field
		OldPath    respjson.Field
		raw        string
	} `json:"-"`
}

// anyBetaMemoryTool20250818Command is implemented by each variant of
// [BetaMemoryTool20250818CommandUnion] to add type safety for the return type of
// [BetaMemoryTool20250818CommandUnion.AsAny]
type anyBetaMemoryTool20250818Command interface {
	implBetaMemoryTool20250818CommandUnion()
}

func (BetaMemoryTool20250818ViewCommand) implBetaMemoryTool20250818CommandUnion()       {}
func (BetaMemoryTool20250818CreateCommand) implBetaMemoryTool20250818CommandUnion()     {}
func (BetaMemoryTool20250818StrReplaceCommand) implBetaMemoryTool20250818CommandUnion() {}
func (BetaMemoryTool20250818InsertCommand) implBetaMemoryTool20250818CommandUnion()     {}
func (BetaMemoryTool20250818DeleteCommand) implBetaMemoryTool20250818CommandUnion()     {}
func (BetaMemoryTool20250818RenameCommand) implBetaMemoryTool20250818CommandUnion()     {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaMemoryTool20250818CommandUnion.AsAny().(type) {
//	case anthropic.BetaMemoryTool20250818ViewCommand:
//	case anthropic.BetaMemoryTool20250818CreateCommand:
//	case anthropic.BetaMemoryTool20250818StrReplaceCommand:
//	case anthropic.BetaMemoryTool20250818InsertCommand:
//	case anthropic.BetaMemoryTool20250818DeleteCommand:
//	case anthropic.BetaMemoryTool20250818RenameCommand:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaMemoryTool20250818CommandUnion) AsAny() anyBetaMemoryTool20250818Command {
	switch u.Command {
	case "view":
		return u.AsView()
	case "create":
		return u.AsCreate()
	case "str_replace":
		return u.AsStrReplace()
	case "insert":
		return u.AsInsert()
	case "delete":
		return u.AsDelete()
	case "rename":
		return u.AsRename()
	}
	return nil
}

func (u BetaMemoryTool20250818CommandUnion) AsView() (v BetaMemoryTool20250818ViewCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMemoryTool20250818CommandUnion) AsCreate() (v BetaMemoryTool20250818CreateCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMemoryTool20250818CommandUnion) AsStrReplace() (v BetaMemoryTool20250818StrReplaceCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMemoryTool20250818CommandUnion) AsInsert() (v BetaMemoryTool20250818InsertCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMemoryTool20250818CommandUnion) AsDelete() (v BetaMemoryTool20250818DeleteCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaMemoryTool20250818CommandUnion) AsRename() (v BetaMemoryTool20250818RenameCommand) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaMemoryTool20250818CommandUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaMemoryTool20250818CommandUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818CreateCommand struct {
	// Command type identifier
	Command constant.Create `json:"command,required"`
	// Content to write to the file
	FileText string `json:"file_text,required"`
	// Path where the file should be created
	Path string `json:"path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		FileText    respjson.Field
		Path        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818CreateCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818CreateCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818DeleteCommand struct {
	// Command type identifier
	Command constant.Delete `json:"command,required"`
	// Path to the file or directory to delete
	Path string `json:"path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Path        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818DeleteCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818DeleteCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818InsertCommand struct {
	// Command type identifier
	Command constant.Insert `json:"command,required"`
	// Line number where text should be inserted
	InsertLine int64 `json:"insert_line,required"`
	// Text to insert at the specified line
	InsertText string `json:"insert_text,required"`
	// Path to the file where text should be inserted
	Path string `json:"path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		InsertLine  respjson.Field
		InsertText  respjson.Field
		Path        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818InsertCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818InsertCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818RenameCommand struct {
	// Command type identifier
	Command constant.Rename `json:"command,required"`
	// New path for the file or directory
	NewPath string `json:"new_path,required"`
	// Current path of the file or directory
	OldPath string `json:"old_path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		NewPath     respjson.Field
		OldPath     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818RenameCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818RenameCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818StrReplaceCommand struct {
	// Command type identifier
	Command constant.StrReplace `json:"command,required"`
	// Text to replace with
	NewStr string `json:"new_str,required"`
	// Text to search for and replace
	OldStr string `json:"old_str,required"`
	// Path to the file where text should be replaced
	Path string `json:"path,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		NewStr      respjson.Field
		OldStr      respjson.Field
		Path        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818StrReplaceCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818StrReplaceCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMemoryTool20250818ViewCommand struct {
	// Command type identifier
	Command constant.View `json:"command,required"`
	// Path to directory or file to view
	Path string `json:"path,required"`
	// Optional line range for viewing specific lines
	ViewRange []int64 `json:"view_range"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command     respjson.Field
		Path        respjson.Field
		ViewRange   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMemoryTool20250818ViewCommand) RawJSON() string { return r.JSON.raw }
func (r *BetaMemoryTool20250818ViewCommand) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMessage struct {
	// Unique object identifier.
	//
	// The format and length of IDs may change over time.
	ID string `json:"id,required"`
	// Information about the container used in the request (for the code execution
	// tool)
	Container BetaContainer `json:"container,required"`
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
	Content []BetaContentBlockUnion `json:"content,required"`
	// Context management response.
	//
	// Information about context management strategies applied during the request.
	ContextManagement BetaContextManagementResponse `json:"context_management,required"`
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
	// "compaction", "refusal", "model_context_window_exceeded".
	StopReason BetaStopReason `json:"stop_reason,required"`
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
	Usage BetaUsage `json:"usage,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID                respjson.Field
		Container         respjson.Field
		Content           respjson.Field
		ContextManagement respjson.Field
		Model             respjson.Field
		Role              respjson.Field
		StopReason        respjson.Field
		StopSequence      respjson.Field
		Type              respjson.Field
		Usage             respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMessage) RawJSON() string { return r.JSON.raw }
func (r *BetaMessage) UnmarshalJSON(data []byte) error {
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
type BetaMessageStopReason string

const (
	BetaMessageStopReasonEndTurn      BetaMessageStopReason = "end_turn"
	BetaMessageStopReasonMaxTokens    BetaMessageStopReason = "max_tokens"
	BetaMessageStopReasonStopSequence BetaMessageStopReason = "stop_sequence"
	BetaMessageStopReasonToolUse      BetaMessageStopReason = "tool_use"
)

type BetaMessageDeltaUsage struct {
	// The cumulative number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The cumulative number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The cumulative number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// Per-iteration token usage breakdown.
	//
	// Each entry represents one sampling iteration, with its own input/output token
	// counts and cache statistics. This allows you to:
	//
	// - Determine which iterations exceeded long context thresholds (>=200k tokens)
	// - Calculate the true context window size from the last iteration
	// - Understand token accumulation across server-side tool use loops
	Iterations BetaIterationsUsage `json:"iterations,required"`
	// The cumulative number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// The number of server tool requests.
	ServerToolUse BetaServerToolUsage `json:"server_tool_use,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InputTokens              respjson.Field
		Iterations               respjson.Field
		OutputTokens             respjson.Field
		ServerToolUse            respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMessageDeltaUsage) RawJSON() string { return r.JSON.raw }
func (r *BetaMessageDeltaUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Token usage for a sampling iteration.
type BetaMessageIterationUsage struct {
	// Breakdown of cached tokens by TTL
	CacheCreation BetaCacheCreation `json:"cache_creation,required"`
	// The number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// The number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// Usage for a sampling iteration
	Type constant.Message `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreation            respjson.Field
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InputTokens              respjson.Field
		OutputTokens             respjson.Field
		Type                     respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMessageIterationUsage) RawJSON() string { return r.JSON.raw }
func (r *BetaMessageIterationUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, Role are required.
type BetaMessageParam struct {
	Content []BetaContentBlockParamUnion `json:"content,omitzero,required"`
	// Any of "user", "assistant".
	Role BetaMessageParamRole `json:"role,omitzero,required"`
	paramObj
}

func NewBetaUserMessage(blocks ...BetaContentBlockParamUnion) BetaMessageParam {
	return BetaMessageParam{
		Role:    BetaMessageParamRoleUser,
		Content: blocks,
	}
}

func (r BetaMessageParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMessageParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMessageParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMessageParamRole string

const (
	BetaMessageParamRoleUser      BetaMessageParamRole = "user"
	BetaMessageParamRoleAssistant BetaMessageParamRole = "assistant"
)

type BetaMessageTokensCount struct {
	// Information about context management applied to the message.
	ContextManagement BetaCountTokensContextManagementResponse `json:"context_management,required"`
	// The total number of tokens across the provided list of messages, system prompt,
	// and tools.
	InputTokens int64 `json:"input_tokens,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ContextManagement respjson.Field
		InputTokens       respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaMessageTokensCount) RawJSON() string { return r.JSON.raw }
func (r *BetaMessageTokensCount) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaMetadataParam struct {
	// An external identifier for the user who is associated with the request.
	//
	// This should be a uuid, hash value, or other opaque identifier. Anthropic may use
	// this id to help detect abuse. Do not include any identifying information such as
	// name, email address, or phone number.
	UserID param.Opt[string] `json:"user_id,omitzero"`
	paramObj
}

func (r BetaMetadataParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaMetadataParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMetadataParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaOutputConfigParam struct {
	// All possible effort levels.
	//
	// Any of "low", "medium", "high", "max".
	Effort BetaOutputConfigEffort `json:"effort,omitzero"`
	// A schema to specify Claude's output format in responses. See
	// [structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
	Format BetaJSONOutputFormatParam `json:"format,omitzero"`
	paramObj
}

func (r BetaOutputConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaOutputConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaOutputConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// All possible effort levels.
type BetaOutputConfigEffort string

const (
	BetaOutputConfigEffortLow    BetaOutputConfigEffort = "low"
	BetaOutputConfigEffortMedium BetaOutputConfigEffort = "medium"
	BetaOutputConfigEffortHigh   BetaOutputConfigEffort = "high"
	BetaOutputConfigEffortMax    BetaOutputConfigEffort = "max"
)

type BetaPlainTextSource struct {
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
func (r BetaPlainTextSource) RawJSON() string { return r.JSON.raw }
func (r *BetaPlainTextSource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BetaPlainTextSource to a BetaPlainTextSourceParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BetaPlainTextSourceParam.Overrides()
func (r BetaPlainTextSource) ToParam() BetaPlainTextSourceParam {
	return param.Override[BetaPlainTextSourceParam](json.RawMessage(r.RawJSON()))
}

// The properties Data, MediaType, Type are required.
type BetaPlainTextSourceParam struct {
	Data string `json:"data,required"`
	// This field can be elided, and will marshal its zero value as "text/plain".
	MediaType constant.TextPlain `json:"media_type,required"`
	// This field can be elided, and will marshal its zero value as "text".
	Type constant.Text `json:"type,required"`
	paramObj
}

func (r BetaPlainTextSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaPlainTextSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaPlainTextSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawContentBlockDeltaUnion contains all possible properties and values from
// [BetaTextDelta], [BetaInputJSONDelta], [BetaCitationsDelta],
// [BetaThinkingDelta], [BetaSignatureDelta], [BetaCompactionContentBlockDelta].
//
// Use the [BetaRawContentBlockDeltaUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaRawContentBlockDeltaUnion struct {
	// This field is from variant [BetaTextDelta].
	Text string `json:"text"`
	// Any of "text_delta", "input_json_delta", "citations_delta", "thinking_delta",
	// "signature_delta", "compaction_delta".
	Type string `json:"type"`
	// This field is from variant [BetaInputJSONDelta].
	PartialJSON string `json:"partial_json"`
	// This field is from variant [BetaCitationsDelta].
	Citation BetaCitationsDeltaCitationUnion `json:"citation"`
	// This field is from variant [BetaThinkingDelta].
	Thinking string `json:"thinking"`
	// This field is from variant [BetaSignatureDelta].
	Signature string `json:"signature"`
	// This field is from variant [BetaCompactionContentBlockDelta].
	Content string `json:"content"`
	JSON    struct {
		Text        respjson.Field
		Type        respjson.Field
		PartialJSON respjson.Field
		Citation    respjson.Field
		Thinking    respjson.Field
		Signature   respjson.Field
		Content     respjson.Field
		raw         string
	} `json:"-"`
}

// anyBetaRawContentBlockDelta is implemented by each variant of
// [BetaRawContentBlockDeltaUnion] to add type safety for the return type of
// [BetaRawContentBlockDeltaUnion.AsAny]
type anyBetaRawContentBlockDelta interface {
	implBetaRawContentBlockDeltaUnion()
}

func (BetaTextDelta) implBetaRawContentBlockDeltaUnion()                   {}
func (BetaInputJSONDelta) implBetaRawContentBlockDeltaUnion()              {}
func (BetaCitationsDelta) implBetaRawContentBlockDeltaUnion()              {}
func (BetaThinkingDelta) implBetaRawContentBlockDeltaUnion()               {}
func (BetaSignatureDelta) implBetaRawContentBlockDeltaUnion()              {}
func (BetaCompactionContentBlockDelta) implBetaRawContentBlockDeltaUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaRawContentBlockDeltaUnion.AsAny().(type) {
//	case anthropic.BetaTextDelta:
//	case anthropic.BetaInputJSONDelta:
//	case anthropic.BetaCitationsDelta:
//	case anthropic.BetaThinkingDelta:
//	case anthropic.BetaSignatureDelta:
//	case anthropic.BetaCompactionContentBlockDelta:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaRawContentBlockDeltaUnion) AsAny() anyBetaRawContentBlockDelta {
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
	case "compaction_delta":
		return u.AsCompactionDelta()
	}
	return nil
}

func (u BetaRawContentBlockDeltaUnion) AsTextDelta() (v BetaTextDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockDeltaUnion) AsInputJSONDelta() (v BetaInputJSONDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockDeltaUnion) AsCitationsDelta() (v BetaCitationsDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockDeltaUnion) AsThinkingDelta() (v BetaThinkingDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockDeltaUnion) AsSignatureDelta() (v BetaSignatureDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockDeltaUnion) AsCompactionDelta() (v BetaCompactionContentBlockDelta) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaRawContentBlockDeltaUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaRawContentBlockDeltaUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawContentBlockDeltaEvent struct {
	Delta BetaRawContentBlockDeltaUnion `json:"delta,required"`
	Index int64                         `json:"index,required"`
	Type  constant.ContentBlockDelta    `json:"type,required"`
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
func (r BetaRawContentBlockDeltaEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawContentBlockDeltaEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawContentBlockStartEvent struct {
	// Response model for a file uploaded to the container.
	ContentBlock BetaRawContentBlockStartEventContentBlockUnion `json:"content_block,required"`
	Index        int64                                          `json:"index,required"`
	Type         constant.ContentBlockStart                     `json:"type,required"`
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
func (r BetaRawContentBlockStartEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawContentBlockStartEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawContentBlockStartEventContentBlockUnion contains all possible properties
// and values from [BetaTextBlock], [BetaThinkingBlock],
// [BetaRedactedThinkingBlock], [BetaToolUseBlock], [BetaServerToolUseBlock],
// [BetaWebSearchToolResultBlock], [BetaWebFetchToolResultBlock],
// [BetaCodeExecutionToolResultBlock], [BetaBashCodeExecutionToolResultBlock],
// [BetaTextEditorCodeExecutionToolResultBlock], [BetaToolSearchToolResultBlock],
// [BetaMCPToolUseBlock], [BetaMCPToolResultBlock], [BetaContainerUploadBlock],
// [BetaCompactionBlock].
//
// Use the [BetaRawContentBlockStartEventContentBlockUnion.AsAny] method to switch
// on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaRawContentBlockStartEventContentBlockUnion struct {
	// This field is from variant [BetaTextBlock].
	Citations []BetaTextCitationUnion `json:"citations"`
	// This field is from variant [BetaTextBlock].
	Text string `json:"text"`
	// Any of "text", "thinking", "redacted_thinking", "tool_use", "server_tool_use",
	// "web_search_tool_result", "web_fetch_tool_result", "code_execution_tool_result",
	// "bash_code_execution_tool_result", "text_editor_code_execution_tool_result",
	// "tool_search_tool_result", "mcp_tool_use", "mcp_tool_result",
	// "container_upload", "compaction".
	Type string `json:"type"`
	// This field is from variant [BetaThinkingBlock].
	Signature string `json:"signature"`
	// This field is from variant [BetaThinkingBlock].
	Thinking string `json:"thinking"`
	// This field is from variant [BetaRedactedThinkingBlock].
	Data  string `json:"data"`
	ID    string `json:"id"`
	Input any    `json:"input"`
	Name  string `json:"name"`
	// This field is a union of [BetaToolUseBlockCallerUnion],
	// [BetaServerToolUseBlockCallerUnion], [BetaWebSearchToolResultBlockCallerUnion],
	// [BetaWebFetchToolResultBlockCallerUnion]
	Caller BetaRawContentBlockStartEventContentBlockUnionCaller `json:"caller"`
	// This field is a union of [BetaWebSearchToolResultBlockContentUnion],
	// [BetaWebFetchToolResultBlockContentUnion],
	// [BetaCodeExecutionToolResultBlockContentUnion],
	// [BetaBashCodeExecutionToolResultBlockContentUnion],
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion],
	// [BetaToolSearchToolResultBlockContentUnion],
	// [BetaMCPToolResultBlockContentUnion], [string]
	Content   BetaRawContentBlockStartEventContentBlockUnionContent `json:"content"`
	ToolUseID string                                                `json:"tool_use_id"`
	// This field is from variant [BetaMCPToolUseBlock].
	ServerName string `json:"server_name"`
	// This field is from variant [BetaMCPToolResultBlock].
	IsError bool `json:"is_error"`
	// This field is from variant [BetaContainerUploadBlock].
	FileID string `json:"file_id"`
	JSON   struct {
		Citations  respjson.Field
		Text       respjson.Field
		Type       respjson.Field
		Signature  respjson.Field
		Thinking   respjson.Field
		Data       respjson.Field
		ID         respjson.Field
		Input      respjson.Field
		Name       respjson.Field
		Caller     respjson.Field
		Content    respjson.Field
		ToolUseID  respjson.Field
		ServerName respjson.Field
		IsError    respjson.Field
		FileID     respjson.Field
		raw        string
	} `json:"-"`
}

// anyBetaRawContentBlockStartEventContentBlock is implemented by each variant of
// [BetaRawContentBlockStartEventContentBlockUnion] to add type safety for the
// return type of [BetaRawContentBlockStartEventContentBlockUnion.AsAny]
type anyBetaRawContentBlockStartEventContentBlock interface {
	implBetaRawContentBlockStartEventContentBlockUnion()
}

func (BetaTextBlock) implBetaRawContentBlockStartEventContentBlockUnion()                        {}
func (BetaThinkingBlock) implBetaRawContentBlockStartEventContentBlockUnion()                    {}
func (BetaRedactedThinkingBlock) implBetaRawContentBlockStartEventContentBlockUnion()            {}
func (BetaToolUseBlock) implBetaRawContentBlockStartEventContentBlockUnion()                     {}
func (BetaServerToolUseBlock) implBetaRawContentBlockStartEventContentBlockUnion()               {}
func (BetaWebSearchToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion()         {}
func (BetaWebFetchToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion()          {}
func (BetaCodeExecutionToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion()     {}
func (BetaBashCodeExecutionToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion() {}
func (BetaTextEditorCodeExecutionToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion() {
}
func (BetaToolSearchToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion() {}
func (BetaMCPToolUseBlock) implBetaRawContentBlockStartEventContentBlockUnion()           {}
func (BetaMCPToolResultBlock) implBetaRawContentBlockStartEventContentBlockUnion()        {}
func (BetaContainerUploadBlock) implBetaRawContentBlockStartEventContentBlockUnion()      {}
func (BetaCompactionBlock) implBetaRawContentBlockStartEventContentBlockUnion()           {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaRawContentBlockStartEventContentBlockUnion.AsAny().(type) {
//	case anthropic.BetaTextBlock:
//	case anthropic.BetaThinkingBlock:
//	case anthropic.BetaRedactedThinkingBlock:
//	case anthropic.BetaToolUseBlock:
//	case anthropic.BetaServerToolUseBlock:
//	case anthropic.BetaWebSearchToolResultBlock:
//	case anthropic.BetaWebFetchToolResultBlock:
//	case anthropic.BetaCodeExecutionToolResultBlock:
//	case anthropic.BetaBashCodeExecutionToolResultBlock:
//	case anthropic.BetaTextEditorCodeExecutionToolResultBlock:
//	case anthropic.BetaToolSearchToolResultBlock:
//	case anthropic.BetaMCPToolUseBlock:
//	case anthropic.BetaMCPToolResultBlock:
//	case anthropic.BetaContainerUploadBlock:
//	case anthropic.BetaCompactionBlock:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaRawContentBlockStartEventContentBlockUnion) AsAny() anyBetaRawContentBlockStartEventContentBlock {
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
	case "mcp_tool_use":
		return u.AsMCPToolUse()
	case "mcp_tool_result":
		return u.AsMCPToolResult()
	case "container_upload":
		return u.AsContainerUpload()
	case "compaction":
		return u.AsCompaction()
	}
	return nil
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsText() (v BetaTextBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsThinking() (v BetaThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsRedactedThinking() (v BetaRedactedThinkingBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsToolUse() (v BetaToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsServerToolUse() (v BetaServerToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsWebSearchToolResult() (v BetaWebSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsWebFetchToolResult() (v BetaWebFetchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsCodeExecutionToolResult() (v BetaCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsBashCodeExecutionToolResult() (v BetaBashCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsTextEditorCodeExecutionToolResult() (v BetaTextEditorCodeExecutionToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsToolSearchToolResult() (v BetaToolSearchToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsMCPToolUse() (v BetaMCPToolUseBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsMCPToolResult() (v BetaMCPToolResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsContainerUpload() (v BetaContainerUploadBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawContentBlockStartEventContentBlockUnion) AsCompaction() (v BetaCompactionBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaRawContentBlockStartEventContentBlockUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaRawContentBlockStartEventContentBlockUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawContentBlockStartEventContentBlockUnionCaller is an implicit subunion of
// [BetaRawContentBlockStartEventContentBlockUnion].
// BetaRawContentBlockStartEventContentBlockUnionCaller provides convenient access
// to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaRawContentBlockStartEventContentBlockUnion].
type BetaRawContentBlockStartEventContentBlockUnionCaller struct {
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

func (r *BetaRawContentBlockStartEventContentBlockUnionCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawContentBlockStartEventContentBlockUnionContent is an implicit subunion of
// [BetaRawContentBlockStartEventContentBlockUnion].
// BetaRawContentBlockStartEventContentBlockUnionContent provides convenient access
// to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaRawContentBlockStartEventContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBetaWebSearchResultBlockArray OfString
// OfBetaMCPToolResultBlockContent]
type BetaRawContentBlockStartEventContentBlockUnionContent struct {
	// This field will be present if the value is a [[]BetaWebSearchResultBlock]
	// instead of an object.
	OfBetaWebSearchResultBlockArray []BetaWebSearchResultBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field will be present if the value is a [[]BetaTextBlock] instead of an
	// object.
	OfBetaMCPToolResultBlockContent []BetaTextBlock `json:",inline"`
	ErrorCode                       string          `json:"error_code"`
	Type                            string          `json:"type"`
	// This field is a union of [BetaDocumentBlock], [[]BetaCodeExecutionOutputBlock],
	// [[]BetaCodeExecutionOutputBlock], [[]BetaBashCodeExecutionOutputBlock], [string]
	Content BetaRawContentBlockStartEventContentBlockUnionContentContent `json:"content"`
	// This field is from variant [BetaWebFetchToolResultBlockContentUnion].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [BetaWebFetchToolResultBlockContentUnion].
	URL        string `json:"url"`
	ReturnCode int64  `json:"return_code"`
	Stderr     string `json:"stderr"`
	Stdout     string `json:"stdout"`
	// This field is from variant [BetaCodeExecutionToolResultBlockContentUnion].
	EncryptedStdout string `json:"encrypted_stdout"`
	ErrorMessage    string `json:"error_message"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	FileType BetaTextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NumLines int64 `json:"num_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	StartLine int64 `json:"start_line"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	Lines []string `json:"lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NewLines int64 `json:"new_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	NewStart int64 `json:"new_start"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	OldLines int64 `json:"old_lines"`
	// This field is from variant
	// [BetaTextEditorCodeExecutionToolResultBlockContentUnion].
	OldStart int64 `json:"old_start"`
	// This field is from variant [BetaToolSearchToolResultBlockContentUnion].
	ToolReferences []BetaToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		OfBetaWebSearchResultBlockArray respjson.Field
		OfString                        respjson.Field
		OfBetaMCPToolResultBlockContent respjson.Field
		ErrorCode                       respjson.Field
		Type                            respjson.Field
		Content                         respjson.Field
		RetrievedAt                     respjson.Field
		URL                             respjson.Field
		ReturnCode                      respjson.Field
		Stderr                          respjson.Field
		Stdout                          respjson.Field
		EncryptedStdout                 respjson.Field
		ErrorMessage                    respjson.Field
		FileType                        respjson.Field
		NumLines                        respjson.Field
		StartLine                       respjson.Field
		TotalLines                      respjson.Field
		IsFileUpdate                    respjson.Field
		Lines                           respjson.Field
		NewLines                        respjson.Field
		NewStart                        respjson.Field
		OldLines                        respjson.Field
		OldStart                        respjson.Field
		ToolReferences                  respjson.Field
		raw                             string
	} `json:"-"`
}

func (r *BetaRawContentBlockStartEventContentBlockUnionContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawContentBlockStartEventContentBlockUnionContentContent is an implicit
// subunion of [BetaRawContentBlockStartEventContentBlockUnion].
// BetaRawContentBlockStartEventContentBlockUnionContentContent provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaRawContentBlockStartEventContentBlockUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfContent OfString]
type BetaRawContentBlockStartEventContentBlockUnionContentContent struct {
	// This field will be present if the value is a [[]BetaCodeExecutionOutputBlock]
	// instead of an object.
	OfContent []BetaCodeExecutionOutputBlock `json:",inline"`
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is from variant [BetaDocumentBlock].
	Citations BetaCitationConfig `json:"citations"`
	// This field is from variant [BetaDocumentBlock].
	Source BetaDocumentBlockSourceUnion `json:"source"`
	// This field is from variant [BetaDocumentBlock].
	Title string `json:"title"`
	// This field is from variant [BetaDocumentBlock].
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

func (r *BetaRawContentBlockStartEventContentBlockUnionContentContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawContentBlockStopEvent struct {
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
func (r BetaRawContentBlockStopEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawContentBlockStopEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawMessageDeltaEvent struct {
	// Information about context management strategies applied during the request
	ContextManagement BetaContextManagementResponse `json:"context_management,required"`
	Delta             BetaRawMessageDeltaEventDelta `json:"delta,required"`
	Type              constant.MessageDelta         `json:"type,required"`
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
	Usage BetaMessageDeltaUsage `json:"usage,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ContextManagement respjson.Field
		Delta             respjson.Field
		Type              respjson.Field
		Usage             respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaRawMessageDeltaEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawMessageDeltaEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawMessageDeltaEventDelta struct {
	// Information about the container used in the request (for the code execution
	// tool)
	Container BetaContainer `json:"container,required"`
	// Any of "end_turn", "max_tokens", "stop_sequence", "tool_use", "pause_turn",
	// "compaction", "refusal", "model_context_window_exceeded".
	StopReason   BetaStopReason `json:"stop_reason,required"`
	StopSequence string         `json:"stop_sequence,required"`
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
func (r BetaRawMessageDeltaEventDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaRawMessageDeltaEventDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawMessageStartEvent struct {
	Message BetaMessage           `json:"message,required"`
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
func (r BetaRawMessageStartEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawMessageStartEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRawMessageStopEvent struct {
	Type constant.MessageStop `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaRawMessageStopEvent) RawJSON() string { return r.JSON.raw }
func (r *BetaRawMessageStopEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawMessageStreamEventUnion contains all possible properties and values from
// [BetaRawMessageStartEvent], [BetaRawMessageDeltaEvent],
// [BetaRawMessageStopEvent], [BetaRawContentBlockStartEvent],
// [BetaRawContentBlockDeltaEvent], [BetaRawContentBlockStopEvent].
//
// Use the [BetaRawMessageStreamEventUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaRawMessageStreamEventUnion struct {
	// This field is from variant [BetaRawMessageStartEvent].
	Message BetaMessage `json:"message"`
	// Any of "message_start", "message_delta", "message_stop", "content_block_start",
	// "content_block_delta", "content_block_stop".
	Type string `json:"type"`
	// This field is from variant [BetaRawMessageDeltaEvent].
	ContextManagement BetaContextManagementResponse `json:"context_management"`
	// This field is a union of [BetaRawMessageDeltaEventDelta],
	// [BetaRawContentBlockDeltaUnion]
	Delta BetaRawMessageStreamEventUnionDelta `json:"delta"`
	// This field is from variant [BetaRawMessageDeltaEvent].
	Usage BetaMessageDeltaUsage `json:"usage"`
	// This field is from variant [BetaRawContentBlockStartEvent].
	ContentBlock BetaRawContentBlockStartEventContentBlockUnion `json:"content_block"`
	Index        int64                                          `json:"index"`
	JSON         struct {
		Message           respjson.Field
		Type              respjson.Field
		ContextManagement respjson.Field
		Delta             respjson.Field
		Usage             respjson.Field
		ContentBlock      respjson.Field
		Index             respjson.Field
		raw               string
	} `json:"-"`
}

// anyBetaRawMessageStreamEvent is implemented by each variant of
// [BetaRawMessageStreamEventUnion] to add type safety for the return type of
// [BetaRawMessageStreamEventUnion.AsAny]
type anyBetaRawMessageStreamEvent interface {
	implBetaRawMessageStreamEventUnion()
}

func (BetaRawMessageStartEvent) implBetaRawMessageStreamEventUnion()      {}
func (BetaRawMessageDeltaEvent) implBetaRawMessageStreamEventUnion()      {}
func (BetaRawMessageStopEvent) implBetaRawMessageStreamEventUnion()       {}
func (BetaRawContentBlockStartEvent) implBetaRawMessageStreamEventUnion() {}
func (BetaRawContentBlockDeltaEvent) implBetaRawMessageStreamEventUnion() {}
func (BetaRawContentBlockStopEvent) implBetaRawMessageStreamEventUnion()  {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaRawMessageStreamEventUnion.AsAny().(type) {
//	case anthropic.BetaRawMessageStartEvent:
//	case anthropic.BetaRawMessageDeltaEvent:
//	case anthropic.BetaRawMessageStopEvent:
//	case anthropic.BetaRawContentBlockStartEvent:
//	case anthropic.BetaRawContentBlockDeltaEvent:
//	case anthropic.BetaRawContentBlockStopEvent:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaRawMessageStreamEventUnion) AsAny() anyBetaRawMessageStreamEvent {
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

func (u BetaRawMessageStreamEventUnion) AsMessageStart() (v BetaRawMessageStartEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawMessageStreamEventUnion) AsMessageDelta() (v BetaRawMessageDeltaEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawMessageStreamEventUnion) AsMessageStop() (v BetaRawMessageStopEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawMessageStreamEventUnion) AsContentBlockStart() (v BetaRawContentBlockStartEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawMessageStreamEventUnion) AsContentBlockDelta() (v BetaRawContentBlockDeltaEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaRawMessageStreamEventUnion) AsContentBlockStop() (v BetaRawContentBlockStopEvent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaRawMessageStreamEventUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaRawMessageStreamEventUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaRawMessageStreamEventUnionDelta is an implicit subunion of
// [BetaRawMessageStreamEventUnion]. BetaRawMessageStreamEventUnionDelta provides
// convenient access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [BetaRawMessageStreamEventUnion].
type BetaRawMessageStreamEventUnionDelta struct {
	// This field is from variant [BetaRawMessageDeltaEventDelta].
	Container BetaContainer `json:"container"`
	// This field is from variant [BetaRawMessageDeltaEventDelta].
	StopReason BetaStopReason `json:"stop_reason"`
	// This field is from variant [BetaRawMessageDeltaEventDelta].
	StopSequence string `json:"stop_sequence"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	Text string `json:"text"`
	Type string `json:"type"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	PartialJSON string `json:"partial_json"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	Citation BetaCitationsDeltaCitationUnion `json:"citation"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	Thinking string `json:"thinking"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	Signature string `json:"signature"`
	// This field is from variant [BetaRawContentBlockDeltaUnion].
	Content string `json:"content"`
	JSON    struct {
		Container    respjson.Field
		StopReason   respjson.Field
		StopSequence respjson.Field
		Text         respjson.Field
		Type         respjson.Field
		PartialJSON  respjson.Field
		Citation     respjson.Field
		Thinking     respjson.Field
		Signature    respjson.Field
		Content      respjson.Field
		raw          string
	} `json:"-"`
}

func (r *BetaRawMessageStreamEventUnionDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaRedactedThinkingBlock struct {
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
func (r BetaRedactedThinkingBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaRedactedThinkingBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Data, Type are required.
type BetaRedactedThinkingBlockParam struct {
	Data string `json:"data,required"`
	// This field can be elided, and will marshal its zero value as
	// "redacted_thinking".
	Type constant.RedactedThinking `json:"type,required"`
	paramObj
}

func (r BetaRedactedThinkingBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaRedactedThinkingBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaRedactedThinkingBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Source, Type are required.
type BetaRequestDocumentBlockParam struct {
	Source  BetaRequestDocumentBlockSourceUnionParam `json:"source,omitzero,required"`
	Context param.Opt[string]                        `json:"context,omitzero"`
	Title   param.Opt[string]                        `json:"title,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	Citations    BetaCitationsConfigParam       `json:"citations,omitzero"`
	// This field can be elided, and will marshal its zero value as "document".
	Type constant.Document `json:"type,required"`
	paramObj
}

func (r BetaRequestDocumentBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaRequestDocumentBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaRequestDocumentBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaRequestDocumentBlockSourceUnionParam struct {
	OfBase64  *BetaBase64PDFSourceParam    `json:",omitzero,inline"`
	OfText    *BetaPlainTextSourceParam    `json:",omitzero,inline"`
	OfContent *BetaContentBlockSourceParam `json:",omitzero,inline"`
	OfURL     *BetaURLPDFSourceParam       `json:",omitzero,inline"`
	OfFile    *BetaFileDocumentSourceParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaRequestDocumentBlockSourceUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfBase64,
		u.OfText,
		u.OfContent,
		u.OfURL,
		u.OfFile)
}
func (u *BetaRequestDocumentBlockSourceUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaRequestDocumentBlockSourceUnionParam) asAny() any {
	if !param.IsOmitted(u.OfBase64) {
		return u.OfBase64
	} else if !param.IsOmitted(u.OfText) {
		return u.OfText
	} else if !param.IsOmitted(u.OfContent) {
		return u.OfContent
	} else if !param.IsOmitted(u.OfURL) {
		return u.OfURL
	} else if !param.IsOmitted(u.OfFile) {
		return u.OfFile
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetContent() *BetaContentBlockSourceContentUnionParam {
	if vt := u.OfContent; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetURL() *string {
	if vt := u.OfURL; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetFileID() *string {
	if vt := u.OfFile; vt != nil {
		return &vt.FileID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetData() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Data)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Data)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetMediaType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.MediaType)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.MediaType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaRequestDocumentBlockSourceUnionParam) GetType() *string {
	if vt := u.OfBase64; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfText; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfContent; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfURL; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfFile; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BetaRequestMCPServerToolConfigurationParam struct {
	Enabled      param.Opt[bool] `json:"enabled,omitzero"`
	AllowedTools []string        `json:"allowed_tools,omitzero"`
	paramObj
}

func (r BetaRequestMCPServerToolConfigurationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaRequestMCPServerToolConfigurationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaRequestMCPServerToolConfigurationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type, URL are required.
type BetaRequestMCPServerURLDefinitionParam struct {
	Name               string                                     `json:"name,required"`
	URL                string                                     `json:"url,required"`
	AuthorizationToken param.Opt[string]                          `json:"authorization_token,omitzero"`
	ToolConfiguration  BetaRequestMCPServerToolConfigurationParam `json:"tool_configuration,omitzero"`
	// This field can be elided, and will marshal its zero value as "url".
	Type constant.URL `json:"type,required"`
	paramObj
}

func (r BetaRequestMCPServerURLDefinitionParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaRequestMCPServerURLDefinitionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaRequestMCPServerURLDefinitionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ToolUseID, Type are required.
type BetaRequestMCPToolResultBlockParam struct {
	ToolUseID string          `json:"tool_use_id,required"`
	IsError   param.Opt[bool] `json:"is_error,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam                 `json:"cache_control,omitzero"`
	Content      BetaRequestMCPToolResultBlockParamContentUnion `json:"content,omitzero"`
	// This field can be elided, and will marshal its zero value as "mcp_tool_result".
	Type constant.MCPToolResult `json:"type,required"`
	paramObj
}

func (r BetaRequestMCPToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaRequestMCPToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaRequestMCPToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaRequestMCPToolResultBlockParamContentUnion struct {
	OfString                        param.Opt[string]    `json:",omitzero,inline"`
	OfBetaMCPToolResultBlockContent []BetaTextBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaRequestMCPToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfBetaMCPToolResultBlockContent)
}
func (u *BetaRequestMCPToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaRequestMCPToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfBetaMCPToolResultBlockContent) {
		return &u.OfBetaMCPToolResultBlockContent
	}
	return nil
}

// The properties Content, Source, Title, Type are required.
type BetaSearchResultBlockParam struct {
	Content []BetaTextBlockParam `json:"content,omitzero,required"`
	Source  string               `json:"source,required"`
	Title   string               `json:"title,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	Citations    BetaCitationsConfigParam       `json:"citations,omitzero"`
	// This field can be elided, and will marshal its zero value as "search_result".
	Type constant.SearchResult `json:"type,required"`
	paramObj
}

func (r BetaSearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaSearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaSearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Tool invocation generated by a server-side tool.
type BetaServerToolCaller struct {
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
func (r BetaServerToolCaller) RawJSON() string { return r.JSON.raw }
func (r *BetaServerToolCaller) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BetaServerToolCaller to a BetaServerToolCallerParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BetaServerToolCallerParam.Overrides()
func (r BetaServerToolCaller) ToParam() BetaServerToolCallerParam {
	return param.Override[BetaServerToolCallerParam](json.RawMessage(r.RawJSON()))
}

// Tool invocation generated by a server-side tool.
//
// The properties ToolID, Type are required.
type BetaServerToolCallerParam struct {
	ToolID string `json:"tool_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20250825".
	Type constant.CodeExecution20250825 `json:"type,required"`
	paramObj
}

func (r BetaServerToolCallerParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaServerToolCallerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaServerToolCallerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaServerToolCaller20260120 struct {
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
func (r BetaServerToolCaller20260120) RawJSON() string { return r.JSON.raw }
func (r *BetaServerToolCaller20260120) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BetaServerToolCaller20260120 to a
// BetaServerToolCaller20260120Param.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BetaServerToolCaller20260120Param.Overrides()
func (r BetaServerToolCaller20260120) ToParam() BetaServerToolCaller20260120Param {
	return param.Override[BetaServerToolCaller20260120Param](json.RawMessage(r.RawJSON()))
}

// The properties ToolID, Type are required.
type BetaServerToolCaller20260120Param struct {
	ToolID string `json:"tool_id,required"`
	// This field can be elided, and will marshal its zero value as
	// "code_execution_20260120".
	Type constant.CodeExecution20260120 `json:"type,required"`
	paramObj
}

func (r BetaServerToolCaller20260120Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaServerToolCaller20260120Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaServerToolCaller20260120Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaServerToolUsage struct {
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
func (r BetaServerToolUsage) RawJSON() string { return r.JSON.raw }
func (r *BetaServerToolUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaServerToolUseBlock struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,required"`
	// Any of "web_search", "web_fetch", "code_execution", "bash_code_execution",
	// "text_editor_code_execution", "tool_search_tool_regex", "tool_search_tool_bm25".
	Name BetaServerToolUseBlockName `json:"name,required"`
	Type constant.ServerToolUse     `json:"type,required"`
	// Tool invocation directly from the model.
	Caller BetaServerToolUseBlockCallerUnion `json:"caller"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Input       respjson.Field
		Name        respjson.Field
		Type        respjson.Field
		Caller      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaServerToolUseBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaServerToolUseBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaServerToolUseBlockName string

const (
	BetaServerToolUseBlockNameWebSearch               BetaServerToolUseBlockName = "web_search"
	BetaServerToolUseBlockNameWebFetch                BetaServerToolUseBlockName = "web_fetch"
	BetaServerToolUseBlockNameCodeExecution           BetaServerToolUseBlockName = "code_execution"
	BetaServerToolUseBlockNameBashCodeExecution       BetaServerToolUseBlockName = "bash_code_execution"
	BetaServerToolUseBlockNameTextEditorCodeExecution BetaServerToolUseBlockName = "text_editor_code_execution"
	BetaServerToolUseBlockNameToolSearchToolRegex     BetaServerToolUseBlockName = "tool_search_tool_regex"
	BetaServerToolUseBlockNameToolSearchToolBm25      BetaServerToolUseBlockName = "tool_search_tool_bm25"
)

// BetaServerToolUseBlockCallerUnion contains all possible properties and values
// from [BetaDirectCaller], [BetaServerToolCaller], [BetaServerToolCaller20260120].
//
// Use the [BetaServerToolUseBlockCallerUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaServerToolUseBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyBetaServerToolUseBlockCaller is implemented by each variant of
// [BetaServerToolUseBlockCallerUnion] to add type safety for the return type of
// [BetaServerToolUseBlockCallerUnion.AsAny]
type anyBetaServerToolUseBlockCaller interface {
	implBetaServerToolUseBlockCallerUnion()
}

func (BetaDirectCaller) implBetaServerToolUseBlockCallerUnion()             {}
func (BetaServerToolCaller) implBetaServerToolUseBlockCallerUnion()         {}
func (BetaServerToolCaller20260120) implBetaServerToolUseBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaServerToolUseBlockCallerUnion.AsAny().(type) {
//	case anthropic.BetaDirectCaller:
//	case anthropic.BetaServerToolCaller:
//	case anthropic.BetaServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaServerToolUseBlockCallerUnion) AsAny() anyBetaServerToolUseBlockCaller {
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

func (u BetaServerToolUseBlockCallerUnion) AsDirect() (v BetaDirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaServerToolUseBlockCallerUnion) AsCodeExecution20250825() (v BetaServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaServerToolUseBlockCallerUnion) AsCodeExecution20260120() (v BetaServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaServerToolUseBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaServerToolUseBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ID, Input, Name, Type are required.
type BetaServerToolUseBlockParam struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,omitzero,required"`
	// Any of "web_search", "web_fetch", "code_execution", "bash_code_execution",
	// "text_editor_code_execution", "tool_search_tool_regex", "tool_search_tool_bm25".
	Name BetaServerToolUseBlockParamName `json:"name,omitzero,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller BetaServerToolUseBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as "server_tool_use".
	Type constant.ServerToolUse `json:"type,required"`
	paramObj
}

func (r BetaServerToolUseBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaServerToolUseBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaServerToolUseBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaServerToolUseBlockParamName string

const (
	BetaServerToolUseBlockParamNameWebSearch               BetaServerToolUseBlockParamName = "web_search"
	BetaServerToolUseBlockParamNameWebFetch                BetaServerToolUseBlockParamName = "web_fetch"
	BetaServerToolUseBlockParamNameCodeExecution           BetaServerToolUseBlockParamName = "code_execution"
	BetaServerToolUseBlockParamNameBashCodeExecution       BetaServerToolUseBlockParamName = "bash_code_execution"
	BetaServerToolUseBlockParamNameTextEditorCodeExecution BetaServerToolUseBlockParamName = "text_editor_code_execution"
	BetaServerToolUseBlockParamNameToolSearchToolRegex     BetaServerToolUseBlockParamName = "tool_search_tool_regex"
	BetaServerToolUseBlockParamNameToolSearchToolBm25      BetaServerToolUseBlockParamName = "tool_search_tool_bm25"
)

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaServerToolUseBlockParamCallerUnion struct {
	OfDirect                *BetaDirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *BetaServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *BetaServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u BetaServerToolUseBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *BetaServerToolUseBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaServerToolUseBlockParamCallerUnion) asAny() any {
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
func (u BetaServerToolUseBlockParamCallerUnion) GetType() *string {
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
func (u BetaServerToolUseBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[BetaServerToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[BetaServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

type BetaSignatureDelta struct {
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
func (r BetaSignatureDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaSignatureDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A skill that was loaded in a container (response model).
type BetaSkill struct {
	// Skill ID
	SkillID string `json:"skill_id,required"`
	// Type of skill - either 'anthropic' (built-in) or 'custom' (user-defined)
	//
	// Any of "anthropic", "custom".
	Type BetaSkillType `json:"type,required"`
	// Skill version or 'latest' for most recent version
	Version string `json:"version,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		SkillID     respjson.Field
		Type        respjson.Field
		Version     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaSkill) RawJSON() string { return r.JSON.raw }
func (r *BetaSkill) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Type of skill - either 'anthropic' (built-in) or 'custom' (user-defined)
type BetaSkillType string

const (
	BetaSkillTypeAnthropic BetaSkillType = "anthropic"
	BetaSkillTypeCustom    BetaSkillType = "custom"
)

// Specification for a skill to be loaded in a container (request model).
//
// The properties SkillID, Type are required.
type BetaSkillParams struct {
	// Skill ID
	SkillID string `json:"skill_id,required"`
	// Type of skill - either 'anthropic' (built-in) or 'custom' (user-defined)
	//
	// Any of "anthropic", "custom".
	Type BetaSkillParamsType `json:"type,omitzero,required"`
	// Skill version or 'latest' for most recent version
	Version param.Opt[string] `json:"version,omitzero"`
	paramObj
}

func (r BetaSkillParams) MarshalJSON() (data []byte, err error) {
	type shadow BetaSkillParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaSkillParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Type of skill - either 'anthropic' (built-in) or 'custom' (user-defined)
type BetaSkillParamsType string

const (
	BetaSkillParamsTypeAnthropic BetaSkillParamsType = "anthropic"
	BetaSkillParamsTypeCustom    BetaSkillParamsType = "custom"
)

type BetaStopReason string

const (
	BetaStopReasonEndTurn                    BetaStopReason = "end_turn"
	BetaStopReasonMaxTokens                  BetaStopReason = "max_tokens"
	BetaStopReasonStopSequence               BetaStopReason = "stop_sequence"
	BetaStopReasonToolUse                    BetaStopReason = "tool_use"
	BetaStopReasonPauseTurn                  BetaStopReason = "pause_turn"
	BetaStopReasonCompaction                 BetaStopReason = "compaction"
	BetaStopReasonRefusal                    BetaStopReason = "refusal"
	BetaStopReasonModelContextWindowExceeded BetaStopReason = "model_context_window_exceeded"
)

type BetaTextBlock struct {
	// Citations supporting the text block.
	//
	// The type of citation returned will depend on the type of document being cited.
	// Citing a PDF results in `page_location`, plain text results in `char_location`,
	// and content document results in `content_block_location`.
	Citations []BetaTextCitationUnion `json:"citations,required"`
	Text      string                  `json:"text,required"`
	Type      constant.Text           `json:"type,required"`
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
func (r BetaTextBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaTextBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Text, Type are required.
type BetaTextBlockParam struct {
	Text      string                       `json:"text,required"`
	Citations []BetaTextCitationParamUnion `json:"citations,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "text".
	Type constant.Text `json:"type,required"`
	paramObj
}

func (r BetaTextBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaTextCitationUnion contains all possible properties and values from
// [BetaCitationCharLocation], [BetaCitationPageLocation],
// [BetaCitationContentBlockLocation], [BetaCitationsWebSearchResultLocation],
// [BetaCitationSearchResultLocation].
//
// Use the [BetaTextCitationUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaTextCitationUnion struct {
	CitedText     string `json:"cited_text"`
	DocumentIndex int64  `json:"document_index"`
	DocumentTitle string `json:"document_title"`
	// This field is from variant [BetaCitationCharLocation].
	EndCharIndex int64  `json:"end_char_index"`
	FileID       string `json:"file_id"`
	// This field is from variant [BetaCitationCharLocation].
	StartCharIndex int64 `json:"start_char_index"`
	// Any of "char_location", "page_location", "content_block_location",
	// "web_search_result_location", "search_result_location".
	Type string `json:"type"`
	// This field is from variant [BetaCitationPageLocation].
	EndPageNumber int64 `json:"end_page_number"`
	// This field is from variant [BetaCitationPageLocation].
	StartPageNumber int64 `json:"start_page_number"`
	EndBlockIndex   int64 `json:"end_block_index"`
	StartBlockIndex int64 `json:"start_block_index"`
	// This field is from variant [BetaCitationsWebSearchResultLocation].
	EncryptedIndex string `json:"encrypted_index"`
	Title          string `json:"title"`
	// This field is from variant [BetaCitationsWebSearchResultLocation].
	URL string `json:"url"`
	// This field is from variant [BetaCitationSearchResultLocation].
	SearchResultIndex int64 `json:"search_result_index"`
	// This field is from variant [BetaCitationSearchResultLocation].
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

// anyBetaTextCitation is implemented by each variant of [BetaTextCitationUnion] to
// add type safety for the return type of [BetaTextCitationUnion.AsAny]
type anyBetaTextCitation interface {
	implBetaTextCitationUnion()
	toParamUnion() BetaTextCitationParamUnion
}

func (BetaCitationCharLocation) implBetaTextCitationUnion()             {}
func (BetaCitationPageLocation) implBetaTextCitationUnion()             {}
func (BetaCitationContentBlockLocation) implBetaTextCitationUnion()     {}
func (BetaCitationsWebSearchResultLocation) implBetaTextCitationUnion() {}
func (BetaCitationSearchResultLocation) implBetaTextCitationUnion()     {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaTextCitationUnion.AsAny().(type) {
//	case anthropic.BetaCitationCharLocation:
//	case anthropic.BetaCitationPageLocation:
//	case anthropic.BetaCitationContentBlockLocation:
//	case anthropic.BetaCitationsWebSearchResultLocation:
//	case anthropic.BetaCitationSearchResultLocation:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaTextCitationUnion) AsAny() anyBetaTextCitation {
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

func (u BetaTextCitationUnion) AsCharLocation() (v BetaCitationCharLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextCitationUnion) AsPageLocation() (v BetaCitationPageLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextCitationUnion) AsContentBlockLocation() (v BetaCitationContentBlockLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextCitationUnion) AsWebSearchResultLocation() (v BetaCitationsWebSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextCitationUnion) AsSearchResultLocation() (v BetaCitationSearchResultLocation) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaTextCitationUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaTextCitationUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaTextCitationParamUnion struct {
	OfCharLocation            *BetaCitationCharLocationParam            `json:",omitzero,inline"`
	OfPageLocation            *BetaCitationPageLocationParam            `json:",omitzero,inline"`
	OfContentBlockLocation    *BetaCitationContentBlockLocationParam    `json:",omitzero,inline"`
	OfWebSearchResultLocation *BetaCitationWebSearchResultLocationParam `json:",omitzero,inline"`
	OfSearchResultLocation    *BetaCitationSearchResultLocationParam    `json:",omitzero,inline"`
	paramUnion
}

func (u BetaTextCitationParamUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfCharLocation,
		u.OfPageLocation,
		u.OfContentBlockLocation,
		u.OfWebSearchResultLocation,
		u.OfSearchResultLocation)
}
func (u *BetaTextCitationParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaTextCitationParamUnion) asAny() any {
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
func (u BetaTextCitationParamUnion) GetEndCharIndex() *int64 {
	if vt := u.OfCharLocation; vt != nil {
		return &vt.EndCharIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetStartCharIndex() *int64 {
	if vt := u.OfCharLocation; vt != nil {
		return &vt.StartCharIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetEndPageNumber() *int64 {
	if vt := u.OfPageLocation; vt != nil {
		return &vt.EndPageNumber
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetStartPageNumber() *int64 {
	if vt := u.OfPageLocation; vt != nil {
		return &vt.StartPageNumber
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetEncryptedIndex() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil {
		return &vt.EncryptedIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetURL() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetSearchResultIndex() *int64 {
	if vt := u.OfSearchResultLocation; vt != nil {
		return &vt.SearchResultIndex
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetSource() *string {
	if vt := u.OfSearchResultLocation; vt != nil {
		return &vt.Source
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetCitedText() *string {
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
func (u BetaTextCitationParamUnion) GetDocumentIndex() *int64 {
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
func (u BetaTextCitationParamUnion) GetDocumentTitle() *string {
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
func (u BetaTextCitationParamUnion) GetType() *string {
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
func (u BetaTextCitationParamUnion) GetEndBlockIndex() *int64 {
	if vt := u.OfContentBlockLocation; vt != nil {
		return (*int64)(&vt.EndBlockIndex)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*int64)(&vt.EndBlockIndex)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetStartBlockIndex() *int64 {
	if vt := u.OfContentBlockLocation; vt != nil {
		return (*int64)(&vt.StartBlockIndex)
	} else if vt := u.OfSearchResultLocation; vt != nil {
		return (*int64)(&vt.StartBlockIndex)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextCitationParamUnion) GetTitle() *string {
	if vt := u.OfWebSearchResultLocation; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	} else if vt := u.OfSearchResultLocation; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	}
	return nil
}

type BetaTextDelta struct {
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
func (r BetaTextDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaTextDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionCreateResultBlock struct {
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
func (r BetaTextEditorCodeExecutionCreateResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaTextEditorCodeExecutionCreateResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties IsFileUpdate, Type are required.
type BetaTextEditorCodeExecutionCreateResultBlockParam struct {
	IsFileUpdate bool `json:"is_file_update,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_create_result".
	Type constant.TextEditorCodeExecutionCreateResult `json:"type,required"`
	paramObj
}

func (r BetaTextEditorCodeExecutionCreateResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextEditorCodeExecutionCreateResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextEditorCodeExecutionCreateResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionStrReplaceResultBlock struct {
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
func (r BetaTextEditorCodeExecutionStrReplaceResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaTextEditorCodeExecutionStrReplaceResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The property Type is required.
type BetaTextEditorCodeExecutionStrReplaceResultBlockParam struct {
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

func (r BetaTextEditorCodeExecutionStrReplaceResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextEditorCodeExecutionStrReplaceResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextEditorCodeExecutionStrReplaceResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionToolResultBlock struct {
	Content   BetaTextEditorCodeExecutionToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                                 `json:"tool_use_id,required"`
	Type      constant.TextEditorCodeExecutionToolResult             `json:"type,required"`
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
func (r BetaTextEditorCodeExecutionToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaTextEditorCodeExecutionToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaTextEditorCodeExecutionToolResultBlockContentUnion contains all possible
// properties and values from [BetaTextEditorCodeExecutionToolResultError],
// [BetaTextEditorCodeExecutionViewResultBlock],
// [BetaTextEditorCodeExecutionCreateResultBlock],
// [BetaTextEditorCodeExecutionStrReplaceResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaTextEditorCodeExecutionToolResultBlockContentUnion struct {
	// This field is from variant [BetaTextEditorCodeExecutionToolResultError].
	ErrorCode BetaTextEditorCodeExecutionToolResultErrorErrorCode `json:"error_code"`
	// This field is from variant [BetaTextEditorCodeExecutionToolResultError].
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
	// This field is from variant [BetaTextEditorCodeExecutionViewResultBlock].
	Content string `json:"content"`
	// This field is from variant [BetaTextEditorCodeExecutionViewResultBlock].
	FileType BetaTextEditorCodeExecutionViewResultBlockFileType `json:"file_type"`
	// This field is from variant [BetaTextEditorCodeExecutionViewResultBlock].
	NumLines int64 `json:"num_lines"`
	// This field is from variant [BetaTextEditorCodeExecutionViewResultBlock].
	StartLine int64 `json:"start_line"`
	// This field is from variant [BetaTextEditorCodeExecutionViewResultBlock].
	TotalLines int64 `json:"total_lines"`
	// This field is from variant [BetaTextEditorCodeExecutionCreateResultBlock].
	IsFileUpdate bool `json:"is_file_update"`
	// This field is from variant [BetaTextEditorCodeExecutionStrReplaceResultBlock].
	Lines []string `json:"lines"`
	// This field is from variant [BetaTextEditorCodeExecutionStrReplaceResultBlock].
	NewLines int64 `json:"new_lines"`
	// This field is from variant [BetaTextEditorCodeExecutionStrReplaceResultBlock].
	NewStart int64 `json:"new_start"`
	// This field is from variant [BetaTextEditorCodeExecutionStrReplaceResultBlock].
	OldLines int64 `json:"old_lines"`
	// This field is from variant [BetaTextEditorCodeExecutionStrReplaceResultBlock].
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

func (u BetaTextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionToolResultError() (v BetaTextEditorCodeExecutionToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionViewResultBlock() (v BetaTextEditorCodeExecutionViewResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionCreateResultBlock() (v BetaTextEditorCodeExecutionCreateResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaTextEditorCodeExecutionToolResultBlockContentUnion) AsResponseTextEditorCodeExecutionStrReplaceResultBlock() (v BetaTextEditorCodeExecutionStrReplaceResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaTextEditorCodeExecutionToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaTextEditorCodeExecutionToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaTextEditorCodeExecutionToolResultBlockParam struct {
	Content   BetaTextEditorCodeExecutionToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                                      `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_tool_result".
	Type constant.TextEditorCodeExecutionToolResult `json:"type,required"`
	paramObj
}

func (r BetaTextEditorCodeExecutionToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextEditorCodeExecutionToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextEditorCodeExecutionToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaTextEditorCodeExecutionToolResultBlockParamContentUnion struct {
	OfRequestTextEditorCodeExecutionToolResultError       *BetaTextEditorCodeExecutionToolResultErrorParam       `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionViewResultBlock       *BetaTextEditorCodeExecutionViewResultBlockParam       `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionCreateResultBlock     *BetaTextEditorCodeExecutionCreateResultBlockParam     `json:",omitzero,inline"`
	OfRequestTextEditorCodeExecutionStrReplaceResultBlock *BetaTextEditorCodeExecutionStrReplaceResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestTextEditorCodeExecutionToolResultError, u.OfRequestTextEditorCodeExecutionViewResultBlock, u.OfRequestTextEditorCodeExecutionCreateResultBlock, u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock)
}
func (u *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) asAny() any {
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
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestTextEditorCodeExecutionToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetErrorMessage() *string {
	if vt := u.OfRequestTextEditorCodeExecutionToolResultError; vt != nil && vt.ErrorMessage.Valid() {
		return &vt.ErrorMessage.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetContent() *string {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetFileType() *string {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil {
		return (*string)(&vt.FileType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetNumLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.NumLines.Valid() {
		return &vt.NumLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetStartLine() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.StartLine.Valid() {
		return &vt.StartLine.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetTotalLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionViewResultBlock; vt != nil && vt.TotalLines.Valid() {
		return &vt.TotalLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetIsFileUpdate() *bool {
	if vt := u.OfRequestTextEditorCodeExecutionCreateResultBlock; vt != nil {
		return &vt.IsFileUpdate
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetLines() []string {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil {
		return vt.Lines
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetNewLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.NewLines.Valid() {
		return &vt.NewLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetNewStart() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.NewStart.Valid() {
		return &vt.NewStart.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetOldLines() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.OldLines.Valid() {
		return &vt.OldLines.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetOldStart() *int64 {
	if vt := u.OfRequestTextEditorCodeExecutionStrReplaceResultBlock; vt != nil && vt.OldStart.Valid() {
		return &vt.OldStart.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaTextEditorCodeExecutionToolResultBlockParamContentUnion) GetType() *string {
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

type BetaTextEditorCodeExecutionToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "file_not_found".
	ErrorCode    BetaTextEditorCodeExecutionToolResultErrorErrorCode `json:"error_code,required"`
	ErrorMessage string                                              `json:"error_message,required"`
	Type         constant.TextEditorCodeExecutionToolResultError     `json:"type,required"`
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
func (r BetaTextEditorCodeExecutionToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BetaTextEditorCodeExecutionToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionToolResultErrorErrorCode string

const (
	BetaTextEditorCodeExecutionToolResultErrorErrorCodeInvalidToolInput      BetaTextEditorCodeExecutionToolResultErrorErrorCode = "invalid_tool_input"
	BetaTextEditorCodeExecutionToolResultErrorErrorCodeUnavailable           BetaTextEditorCodeExecutionToolResultErrorErrorCode = "unavailable"
	BetaTextEditorCodeExecutionToolResultErrorErrorCodeTooManyRequests       BetaTextEditorCodeExecutionToolResultErrorErrorCode = "too_many_requests"
	BetaTextEditorCodeExecutionToolResultErrorErrorCodeExecutionTimeExceeded BetaTextEditorCodeExecutionToolResultErrorErrorCode = "execution_time_exceeded"
	BetaTextEditorCodeExecutionToolResultErrorErrorCodeFileNotFound          BetaTextEditorCodeExecutionToolResultErrorErrorCode = "file_not_found"
)

// The properties ErrorCode, Type are required.
type BetaTextEditorCodeExecutionToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded", "file_not_found".
	ErrorCode    BetaTextEditorCodeExecutionToolResultErrorParamErrorCode `json:"error_code,omitzero,required"`
	ErrorMessage param.Opt[string]                                        `json:"error_message,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_tool_result_error".
	Type constant.TextEditorCodeExecutionToolResultError `json:"type,required"`
	paramObj
}

func (r BetaTextEditorCodeExecutionToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextEditorCodeExecutionToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextEditorCodeExecutionToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionToolResultErrorParamErrorCode string

const (
	BetaTextEditorCodeExecutionToolResultErrorParamErrorCodeInvalidToolInput      BetaTextEditorCodeExecutionToolResultErrorParamErrorCode = "invalid_tool_input"
	BetaTextEditorCodeExecutionToolResultErrorParamErrorCodeUnavailable           BetaTextEditorCodeExecutionToolResultErrorParamErrorCode = "unavailable"
	BetaTextEditorCodeExecutionToolResultErrorParamErrorCodeTooManyRequests       BetaTextEditorCodeExecutionToolResultErrorParamErrorCode = "too_many_requests"
	BetaTextEditorCodeExecutionToolResultErrorParamErrorCodeExecutionTimeExceeded BetaTextEditorCodeExecutionToolResultErrorParamErrorCode = "execution_time_exceeded"
	BetaTextEditorCodeExecutionToolResultErrorParamErrorCodeFileNotFound          BetaTextEditorCodeExecutionToolResultErrorParamErrorCode = "file_not_found"
)

type BetaTextEditorCodeExecutionViewResultBlock struct {
	Content string `json:"content,required"`
	// Any of "text", "image", "pdf".
	FileType   BetaTextEditorCodeExecutionViewResultBlockFileType `json:"file_type,required"`
	NumLines   int64                                              `json:"num_lines,required"`
	StartLine  int64                                              `json:"start_line,required"`
	TotalLines int64                                              `json:"total_lines,required"`
	Type       constant.TextEditorCodeExecutionViewResult         `json:"type,required"`
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
func (r BetaTextEditorCodeExecutionViewResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaTextEditorCodeExecutionViewResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionViewResultBlockFileType string

const (
	BetaTextEditorCodeExecutionViewResultBlockFileTypeText  BetaTextEditorCodeExecutionViewResultBlockFileType = "text"
	BetaTextEditorCodeExecutionViewResultBlockFileTypeImage BetaTextEditorCodeExecutionViewResultBlockFileType = "image"
	BetaTextEditorCodeExecutionViewResultBlockFileTypePDF   BetaTextEditorCodeExecutionViewResultBlockFileType = "pdf"
)

// The properties Content, FileType, Type are required.
type BetaTextEditorCodeExecutionViewResultBlockParam struct {
	Content string `json:"content,required"`
	// Any of "text", "image", "pdf".
	FileType   BetaTextEditorCodeExecutionViewResultBlockParamFileType `json:"file_type,omitzero,required"`
	NumLines   param.Opt[int64]                                        `json:"num_lines,omitzero"`
	StartLine  param.Opt[int64]                                        `json:"start_line,omitzero"`
	TotalLines param.Opt[int64]                                        `json:"total_lines,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_code_execution_view_result".
	Type constant.TextEditorCodeExecutionViewResult `json:"type,required"`
	paramObj
}

func (r BetaTextEditorCodeExecutionViewResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaTextEditorCodeExecutionViewResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaTextEditorCodeExecutionViewResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaTextEditorCodeExecutionViewResultBlockParamFileType string

const (
	BetaTextEditorCodeExecutionViewResultBlockParamFileTypeText  BetaTextEditorCodeExecutionViewResultBlockParamFileType = "text"
	BetaTextEditorCodeExecutionViewResultBlockParamFileTypeImage BetaTextEditorCodeExecutionViewResultBlockParamFileType = "image"
	BetaTextEditorCodeExecutionViewResultBlockParamFileTypePDF   BetaTextEditorCodeExecutionViewResultBlockParamFileType = "pdf"
)

type BetaThinkingBlock struct {
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
func (r BetaThinkingBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaThinkingBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Signature, Thinking, Type are required.
type BetaThinkingBlockParam struct {
	Signature string `json:"signature,required"`
	Thinking  string `json:"thinking,required"`
	// This field can be elided, and will marshal its zero value as "thinking".
	Type constant.Thinking `json:"type,required"`
	paramObj
}

func (r BetaThinkingBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaThinkingBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaThinkingBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewBetaThinkingConfigAdaptiveParam() BetaThinkingConfigAdaptiveParam {
	return BetaThinkingConfigAdaptiveParam{
		Type: "adaptive",
	}
}

// This struct has a constant value, construct it with
// [NewBetaThinkingConfigAdaptiveParam].
type BetaThinkingConfigAdaptiveParam struct {
	Type constant.Adaptive `json:"type,required"`
	paramObj
}

func (r BetaThinkingConfigAdaptiveParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaThinkingConfigAdaptiveParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaThinkingConfigAdaptiveParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewBetaThinkingConfigDisabledParam() BetaThinkingConfigDisabledParam {
	return BetaThinkingConfigDisabledParam{
		Type: "disabled",
	}
}

// This struct has a constant value, construct it with
// [NewBetaThinkingConfigDisabledParam].
type BetaThinkingConfigDisabledParam struct {
	Type constant.Disabled `json:"type,required"`
	paramObj
}

func (r BetaThinkingConfigDisabledParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaThinkingConfigDisabledParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaThinkingConfigDisabledParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties BudgetTokens, Type are required.
type BetaThinkingConfigEnabledParam struct {
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

func (r BetaThinkingConfigEnabledParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaThinkingConfigEnabledParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaThinkingConfigEnabledParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func BetaThinkingConfigParamOfEnabled(budgetTokens int64) BetaThinkingConfigParamUnion {
	var enabled BetaThinkingConfigEnabledParam
	enabled.BudgetTokens = budgetTokens
	return BetaThinkingConfigParamUnion{OfEnabled: &enabled}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaThinkingConfigParamUnion struct {
	OfEnabled  *BetaThinkingConfigEnabledParam  `json:",omitzero,inline"`
	OfDisabled *BetaThinkingConfigDisabledParam `json:",omitzero,inline"`
	OfAdaptive *BetaThinkingConfigAdaptiveParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaThinkingConfigParamUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfEnabled, u.OfDisabled, u.OfAdaptive)
}
func (u *BetaThinkingConfigParamUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaThinkingConfigParamUnion) asAny() any {
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
func (u BetaThinkingConfigParamUnion) GetBudgetTokens() *int64 {
	if vt := u.OfEnabled; vt != nil {
		return &vt.BudgetTokens
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaThinkingConfigParamUnion) GetType() *string {
	if vt := u.OfEnabled; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfDisabled; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAdaptive; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}


type BetaThinkingDelta struct {
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
func (r BetaThinkingDelta) RawJSON() string { return r.JSON.raw }
func (r *BetaThinkingDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, Value are required.
type BetaThinkingTurnsParam struct {
	Value int64 `json:"value,required"`
	// This field can be elided, and will marshal its zero value as "thinking_turns".
	Type constant.ThinkingTurns `json:"type,required"`
	paramObj
}

func (r BetaThinkingTurnsParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaThinkingTurnsParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaThinkingTurnsParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties InputSchema, Name are required.
type BetaToolParam struct {
	// [JSON schema](https://json-schema.org/draft/2020-12) for this tool's input.
	//
	// This defines the shape of the `input` that your tool accepts and that the model
	// will produce.
	InputSchema BetaToolInputSchemaParam `json:"input_schema,omitzero,required"`
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
	Type BetaToolType `json:"type,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	paramObj
}

func (r BetaToolParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// [JSON schema](https://json-schema.org/draft/2020-12) for this tool's input.
//
// This defines the shape of the `input` that your tool accepts and that the model
// will produce.
//
// The property Type is required.
type BetaToolInputSchemaParam struct {
	Properties any      `json:"properties,omitzero"`
	Required   []string `json:"required,omitzero"`
	// This field can be elided, and will marshal its zero value as "object".
	Type        constant.Object `json:"type,required"`
	ExtraFields map[string]any  `json:"-"`
	paramObj
}

func (r BetaToolInputSchemaParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolInputSchemaParam
	return param.MarshalWithExtras(r, (*shadow)(&r), r.ExtraFields)
}
func (r *BetaToolInputSchemaParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolType string

const (
	BetaToolTypeCustom BetaToolType = "custom"
)

// The properties Name, Type are required.
type BetaToolBash20241022Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "bash".
	Name constant.Bash `json:"name,required"`
	// This field can be elided, and will marshal its zero value as "bash_20241022".
	Type constant.Bash20241022 `json:"type,required"`
	paramObj
}

func (r BetaToolBash20241022Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolBash20241022Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolBash20241022Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaToolBash20250124Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
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

func (r BetaToolBash20250124Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolBash20250124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolBash20250124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func BetaToolChoiceParamOfTool(name string) BetaToolChoiceUnionParam {
	var tool BetaToolChoiceToolParam
	tool.Name = name
	return BetaToolChoiceUnionParam{OfTool: &tool}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaToolChoiceUnionParam struct {
	OfAuto *BetaToolChoiceAutoParam `json:",omitzero,inline"`
	OfAny  *BetaToolChoiceAnyParam  `json:",omitzero,inline"`
	OfTool *BetaToolChoiceToolParam `json:",omitzero,inline"`
	OfNone *BetaToolChoiceNoneParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaToolChoiceUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfAuto, u.OfAny, u.OfTool, u.OfNone)
}
func (u *BetaToolChoiceUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaToolChoiceUnionParam) asAny() any {
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
func (u BetaToolChoiceUnionParam) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return &vt.Name
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolChoiceUnionParam) GetType() *string {
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
func (u BetaToolChoiceUnionParam) GetDisableParallelToolUse() *bool {
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
type BetaToolChoiceAnyParam struct {
	// Whether to disable parallel tool use.
	//
	// Defaults to `false`. If set to `true`, the model will output exactly one tool
	// use.
	DisableParallelToolUse param.Opt[bool] `json:"disable_parallel_tool_use,omitzero"`
	// This field can be elided, and will marshal its zero value as "any".
	Type constant.Any `json:"type,required"`
	paramObj
}

func (r BetaToolChoiceAnyParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolChoiceAnyParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolChoiceAnyParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The model will automatically decide whether to use tools.
//
// The property Type is required.
type BetaToolChoiceAutoParam struct {
	// Whether to disable parallel tool use.
	//
	// Defaults to `false`. If set to `true`, the model will output at most one tool
	// use.
	DisableParallelToolUse param.Opt[bool] `json:"disable_parallel_tool_use,omitzero"`
	// This field can be elided, and will marshal its zero value as "auto".
	Type constant.Auto `json:"type,required"`
	paramObj
}

func (r BetaToolChoiceAutoParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolChoiceAutoParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolChoiceAutoParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewBetaToolChoiceNoneParam() BetaToolChoiceNoneParam {
	return BetaToolChoiceNoneParam{
		Type: "none",
	}
}

// The model will not be allowed to use tools.
//
// This struct has a constant value, construct it with
// [NewBetaToolChoiceNoneParam].
type BetaToolChoiceNoneParam struct {
	Type constant.None `json:"type,required"`
	paramObj
}

func (r BetaToolChoiceNoneParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolChoiceNoneParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolChoiceNoneParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The model will use the specified tool with `tool_choice.name`.
//
// The properties Name, Type are required.
type BetaToolChoiceToolParam struct {
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

func (r BetaToolChoiceToolParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolChoiceToolParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolChoiceToolParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties DisplayHeightPx, DisplayWidthPx, Name, Type are required.
type BetaToolComputerUse20241022Param struct {
	// The height of the display in pixels.
	DisplayHeightPx int64 `json:"display_height_px,required"`
	// The width of the display in pixels.
	DisplayWidthPx int64 `json:"display_width_px,required"`
	// The X11 display number (e.g. 0, 1) for the display.
	DisplayNumber param.Opt[int64] `json:"display_number,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "computer".
	Name constant.Computer `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "computer_20241022".
	Type constant.Computer20241022 `json:"type,required"`
	paramObj
}

func (r BetaToolComputerUse20241022Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolComputerUse20241022Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolComputerUse20241022Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties DisplayHeightPx, DisplayWidthPx, Name, Type are required.
type BetaToolComputerUse20250124Param struct {
	// The height of the display in pixels.
	DisplayHeightPx int64 `json:"display_height_px,required"`
	// The width of the display in pixels.
	DisplayWidthPx int64 `json:"display_width_px,required"`
	// The X11 display number (e.g. 0, 1) for the display.
	DisplayNumber param.Opt[int64] `json:"display_number,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "computer".
	Name constant.Computer `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "computer_20250124".
	Type constant.Computer20250124 `json:"type,required"`
	paramObj
}

func (r BetaToolComputerUse20250124Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolComputerUse20250124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolComputerUse20250124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties DisplayHeightPx, DisplayWidthPx, Name, Type are required.
type BetaToolComputerUse20251124Param struct {
	// The height of the display in pixels.
	DisplayHeightPx int64 `json:"display_height_px,required"`
	// The width of the display in pixels.
	DisplayWidthPx int64 `json:"display_width_px,required"`
	// The X11 display number (e.g. 0, 1) for the display.
	DisplayNumber param.Opt[int64] `json:"display_number,omitzero"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// Whether to enable an action to take a zoomed-in screenshot of the screen.
	EnableZoom param.Opt[bool] `json:"enable_zoom,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as "computer".
	Name constant.Computer `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "computer_20251124".
	Type constant.Computer20251124 `json:"type,required"`
	paramObj
}

func (r BetaToolComputerUse20251124Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolComputerUse20251124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolComputerUse20251124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolReferenceBlock struct {
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
func (r BetaToolReferenceBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaToolReferenceBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Tool reference block that can be included in tool_result content.
//
// The properties ToolName, Type are required.
type BetaToolReferenceBlockParam struct {
	ToolName string `json:"tool_name,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_reference".
	Type constant.ToolReference `json:"type,required"`
	paramObj
}

func (r BetaToolReferenceBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolReferenceBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolReferenceBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ToolUseID, Type are required.
type BetaToolResultBlockParam struct {
	ToolUseID string          `json:"tool_use_id,required"`
	IsError   param.Opt[bool] `json:"is_error,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam         `json:"cache_control,omitzero"`
	Content      []BetaToolResultBlockParamContentUnion `json:"content,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_result".
	Type constant.ToolResult `json:"type,required"`
	paramObj
}

func (r BetaToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func NewBetaToolResultTextBlockParam(toolUseID string, text string, isError bool) BetaToolResultBlockParam {
	var p BetaToolResultBlockParam
	p.ToolUseID = toolUseID
	p.IsError = param.Opt[bool]{Value: isError}
	p.Content = []BetaToolResultBlockParamContentUnion{
		{OfText: &BetaTextBlockParam{Text: text}},
	}
	return p
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaToolResultBlockParamContentUnion struct {
	OfText          *BetaTextBlockParam            `json:",omitzero,inline"`
	OfImage         *BetaImageBlockParam           `json:",omitzero,inline"`
	OfSearchResult  *BetaSearchResultBlockParam    `json:",omitzero,inline"`
	OfDocument      *BetaRequestDocumentBlockParam `json:",omitzero,inline"`
	OfToolReference *BetaToolReferenceBlockParam   `json:",omitzero,inline"`
	paramUnion
}

func (u BetaToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfText,
		u.OfImage,
		u.OfSearchResult,
		u.OfDocument,
		u.OfToolReference)
}
func (u *BetaToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaToolResultBlockParamContentUnion) asAny() any {
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
func (u BetaToolResultBlockParamContentUnion) GetText() *string {
	if vt := u.OfText; vt != nil {
		return &vt.Text
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolResultBlockParamContentUnion) GetContent() []BetaTextBlockParam {
	if vt := u.OfSearchResult; vt != nil {
		return vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolResultBlockParamContentUnion) GetContext() *string {
	if vt := u.OfDocument; vt != nil && vt.Context.Valid() {
		return &vt.Context.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolResultBlockParamContentUnion) GetToolName() *string {
	if vt := u.OfToolReference; vt != nil {
		return &vt.ToolName
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolResultBlockParamContentUnion) GetType() *string {
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
func (u BetaToolResultBlockParamContentUnion) GetTitle() *string {
	if vt := u.OfSearchResult; vt != nil {
		return (*string)(&vt.Title)
	} else if vt := u.OfDocument; vt != nil && vt.Title.Valid() {
		return &vt.Title.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's CacheControl property, if present.
func (u BetaToolResultBlockParamContentUnion) GetCacheControl() *BetaCacheControlEphemeralParam {
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
func (u BetaToolResultBlockParamContentUnion) GetCitations() (res betaToolResultBlockParamContentUnionCitations) {
	if vt := u.OfText; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Citations
	} else if vt := u.OfDocument; vt != nil {
		res.any = &vt.Citations
	}
	return
}

// Can have the runtime types [*[]BetaTextCitationParamUnion],
// [*BetaCitationsConfigParam]
type betaToolResultBlockParamContentUnionCitations struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *[]anthropic.BetaTextCitationParamUnion:
//	case *anthropic.BetaCitationsConfigParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaToolResultBlockParamContentUnionCitations) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionCitations) GetEnabled() *bool {
	switch vt := u.any.(type) {
	case *BetaCitationsConfigParam:
		return paramutil.AddrIfPresent(vt.Enabled)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u BetaToolResultBlockParamContentUnion) GetSource() (res betaToolResultBlockParamContentUnionSource) {
	if vt := u.OfImage; vt != nil {
		res.any = vt.Source.asAny()
	} else if vt := u.OfSearchResult; vt != nil {
		res.any = &vt.Source
	} else if vt := u.OfDocument; vt != nil {
		res.any = vt.Source.asAny()
	}
	return
}

// Can have the runtime types [*BetaBase64ImageSourceParam],
// [*BetaURLImageSourceParam], [*BetaFileImageSourceParam], [*string],
// [*BetaBase64PDFSourceParam], [*BetaPlainTextSourceParam],
// [*BetaContentBlockSourceParam], [*BetaURLPDFSourceParam],
// [*BetaFileDocumentSourceParam]
type betaToolResultBlockParamContentUnionSource struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *anthropic.BetaBase64ImageSourceParam:
//	case *anthropic.BetaURLImageSourceParam:
//	case *anthropic.BetaFileImageSourceParam:
//	case *string:
//	case *anthropic.BetaBase64PDFSourceParam:
//	case *anthropic.BetaPlainTextSourceParam:
//	case *anthropic.BetaContentBlockSourceParam:
//	case *anthropic.BetaURLPDFSourceParam:
//	case *anthropic.BetaFileDocumentSourceParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u betaToolResultBlockParamContentUnionSource) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetContent() *BetaContentBlockSourceContentUnionParam {
	switch vt := u.any.(type) {
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetContent()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetData() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetData()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetData()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetMediaType() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetMediaType()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetMediaType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetType() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetType()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetURL() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetURL()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetURL()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u betaToolResultBlockParamContentUnionSource) GetFileID() *string {
	switch vt := u.any.(type) {
	case *BetaImageBlockParamSourceUnion:
		return vt.GetFileID()
	case *BetaRequestDocumentBlockSourceUnionParam:
		return vt.GetFileID()
	}
	return nil
}

// The properties Name, Type are required.
type BetaToolSearchToolBm25_20251119Param struct {
	// Any of "tool_search_tool_bm25_20251119", "tool_search_tool_bm25".
	Type BetaToolSearchToolBm25_20251119Type `json:"type,omitzero,required"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_bm25".
	Name constant.ToolSearchToolBm25 `json:"name,required"`
	paramObj
}

func (r BetaToolSearchToolBm25_20251119Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolSearchToolBm25_20251119Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolSearchToolBm25_20251119Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolSearchToolBm25_20251119Type string

const (
	BetaToolSearchToolBm25_20251119TypeToolSearchToolBm25_20251119 BetaToolSearchToolBm25_20251119Type = "tool_search_tool_bm25_20251119"
	BetaToolSearchToolBm25_20251119TypeToolSearchToolBm25          BetaToolSearchToolBm25_20251119Type = "tool_search_tool_bm25"
)

// The properties Name, Type are required.
type BetaToolSearchToolRegex20251119Param struct {
	// Any of "tool_search_tool_regex_20251119", "tool_search_tool_regex".
	Type BetaToolSearchToolRegex20251119Type `json:"type,omitzero,required"`
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_regex".
	Name constant.ToolSearchToolRegex `json:"name,required"`
	paramObj
}

func (r BetaToolSearchToolRegex20251119Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolSearchToolRegex20251119Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolSearchToolRegex20251119Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolSearchToolRegex20251119Type string

const (
	BetaToolSearchToolRegex20251119TypeToolSearchToolRegex20251119 BetaToolSearchToolRegex20251119Type = "tool_search_tool_regex_20251119"
	BetaToolSearchToolRegex20251119TypeToolSearchToolRegex         BetaToolSearchToolRegex20251119Type = "tool_search_tool_regex"
)

type BetaToolSearchToolResultBlock struct {
	Content   BetaToolSearchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                    `json:"tool_use_id,required"`
	Type      constant.ToolSearchToolResult             `json:"type,required"`
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
func (r BetaToolSearchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaToolSearchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaToolSearchToolResultBlockContentUnion contains all possible properties and
// values from [BetaToolSearchToolResultError],
// [BetaToolSearchToolSearchResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaToolSearchToolResultBlockContentUnion struct {
	// This field is from variant [BetaToolSearchToolResultError].
	ErrorCode BetaToolSearchToolResultErrorErrorCode `json:"error_code"`
	// This field is from variant [BetaToolSearchToolResultError].
	ErrorMessage string `json:"error_message"`
	Type         string `json:"type"`
	// This field is from variant [BetaToolSearchToolSearchResultBlock].
	ToolReferences []BetaToolReferenceBlock `json:"tool_references"`
	JSON           struct {
		ErrorCode      respjson.Field
		ErrorMessage   respjson.Field
		Type           respjson.Field
		ToolReferences respjson.Field
		raw            string
	} `json:"-"`
}

func (u BetaToolSearchToolResultBlockContentUnion) AsResponseToolSearchToolResultError() (v BetaToolSearchToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaToolSearchToolResultBlockContentUnion) AsResponseToolSearchToolSearchResultBlock() (v BetaToolSearchToolSearchResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaToolSearchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaToolSearchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaToolSearchToolResultBlockParam struct {
	Content   BetaToolSearchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                         `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_result".
	Type constant.ToolSearchToolResult `json:"type,required"`
	paramObj
}

func (r BetaToolSearchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolSearchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolSearchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaToolSearchToolResultBlockParamContentUnion struct {
	OfRequestToolSearchToolResultError       *BetaToolSearchToolResultErrorParam       `json:",omitzero,inline"`
	OfRequestToolSearchToolSearchResultBlock *BetaToolSearchToolSearchResultBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaToolSearchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestToolSearchToolResultError, u.OfRequestToolSearchToolSearchResultBlock)
}
func (u *BetaToolSearchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaToolSearchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestToolSearchToolResultError) {
		return u.OfRequestToolSearchToolResultError
	} else if !param.IsOmitted(u.OfRequestToolSearchToolSearchResultBlock) {
		return u.OfRequestToolSearchToolSearchResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolSearchToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestToolSearchToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolSearchToolResultBlockParamContentUnion) GetToolReferences() []BetaToolReferenceBlockParam {
	if vt := u.OfRequestToolSearchToolSearchResultBlock; vt != nil {
		return vt.ToolReferences
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolSearchToolResultBlockParamContentUnion) GetType() *string {
	if vt := u.OfRequestToolSearchToolResultError; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfRequestToolSearchToolSearchResultBlock; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type BetaToolSearchToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode    BetaToolSearchToolResultErrorErrorCode `json:"error_code,required"`
	ErrorMessage string                                 `json:"error_message,required"`
	Type         constant.ToolSearchToolResultError     `json:"type,required"`
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
func (r BetaToolSearchToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BetaToolSearchToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolSearchToolResultErrorErrorCode string

const (
	BetaToolSearchToolResultErrorErrorCodeInvalidToolInput      BetaToolSearchToolResultErrorErrorCode = "invalid_tool_input"
	BetaToolSearchToolResultErrorErrorCodeUnavailable           BetaToolSearchToolResultErrorErrorCode = "unavailable"
	BetaToolSearchToolResultErrorErrorCodeTooManyRequests       BetaToolSearchToolResultErrorErrorCode = "too_many_requests"
	BetaToolSearchToolResultErrorErrorCodeExecutionTimeExceeded BetaToolSearchToolResultErrorErrorCode = "execution_time_exceeded"
)

// The properties ErrorCode, Type are required.
type BetaToolSearchToolResultErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "too_many_requests",
	// "execution_time_exceeded".
	ErrorCode BetaToolSearchToolResultErrorParamErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_result_error".
	Type constant.ToolSearchToolResultError `json:"type,required"`
	paramObj
}

func (r BetaToolSearchToolResultErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolSearchToolResultErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolSearchToolResultErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaToolSearchToolResultErrorParamErrorCode string

const (
	BetaToolSearchToolResultErrorParamErrorCodeInvalidToolInput      BetaToolSearchToolResultErrorParamErrorCode = "invalid_tool_input"
	BetaToolSearchToolResultErrorParamErrorCodeUnavailable           BetaToolSearchToolResultErrorParamErrorCode = "unavailable"
	BetaToolSearchToolResultErrorParamErrorCodeTooManyRequests       BetaToolSearchToolResultErrorParamErrorCode = "too_many_requests"
	BetaToolSearchToolResultErrorParamErrorCodeExecutionTimeExceeded BetaToolSearchToolResultErrorParamErrorCode = "execution_time_exceeded"
)

type BetaToolSearchToolSearchResultBlock struct {
	ToolReferences []BetaToolReferenceBlock            `json:"tool_references,required"`
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
func (r BetaToolSearchToolSearchResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaToolSearchToolSearchResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ToolReferences, Type are required.
type BetaToolSearchToolSearchResultBlockParam struct {
	ToolReferences []BetaToolReferenceBlockParam `json:"tool_references,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "tool_search_tool_search_result".
	Type constant.ToolSearchToolSearchResult `json:"type,required"`
	paramObj
}

func (r BetaToolSearchToolSearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolSearchToolSearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolSearchToolSearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaToolTextEditor20241022Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	//
	// This field can be elided, and will marshal its zero value as
	// "str_replace_editor".
	Name constant.StrReplaceEditor `json:"name,required"`
	// This field can be elided, and will marshal its zero value as
	// "text_editor_20241022".
	Type constant.TextEditor20241022 `json:"type,required"`
	paramObj
}

func (r BetaToolTextEditor20241022Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolTextEditor20241022Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolTextEditor20241022Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaToolTextEditor20250124Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
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

func (r BetaToolTextEditor20250124Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolTextEditor20250124Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolTextEditor20250124Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaToolTextEditor20250429Param struct {
	// If true, tool will not be included in initial system prompt. Only loaded when
	// returned via tool_reference from tool search.
	DeferLoading param.Opt[bool] `json:"defer_loading,omitzero"`
	// When true, guarantees schema validation on tool names and inputs
	Strict param.Opt[bool] `json:"strict,omitzero"`
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	AllowedCallers []string `json:"allowed_callers,omitzero"`
	// Create a cache control breakpoint at this content block.
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
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

func (r BetaToolTextEditor20250429Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolTextEditor20250429Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolTextEditor20250429Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaToolTextEditor20250728Param struct {
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
	CacheControl  BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	InputExamples []map[string]any               `json:"input_examples,omitzero"`
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

func (r BetaToolTextEditor20250728Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolTextEditor20250728Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolTextEditor20250728Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func BetaToolUnionParamOfTool(inputSchema BetaToolInputSchemaParam, name string) BetaToolUnionParam {
	var variant BetaToolParam
	variant.InputSchema = inputSchema
	variant.Name = name
	return BetaToolUnionParam{OfTool: &variant}
}

func BetaToolUnionParamOfComputerUseTool20241022(displayHeightPx int64, displayWidthPx int64) BetaToolUnionParam {
	var variant BetaToolComputerUse20241022Param
	variant.DisplayHeightPx = displayHeightPx
	variant.DisplayWidthPx = displayWidthPx
	return BetaToolUnionParam{OfComputerUseTool20241022: &variant}
}

func BetaToolUnionParamOfComputerUseTool20250124(displayHeightPx int64, displayWidthPx int64) BetaToolUnionParam {
	var variant BetaToolComputerUse20250124Param
	variant.DisplayHeightPx = displayHeightPx
	variant.DisplayWidthPx = displayWidthPx
	return BetaToolUnionParam{OfComputerUseTool20250124: &variant}
}

func BetaToolUnionParamOfComputerUseTool20251124(displayHeightPx int64, displayWidthPx int64) BetaToolUnionParam {
	var variant BetaToolComputerUse20251124Param
	variant.DisplayHeightPx = displayHeightPx
	variant.DisplayWidthPx = displayWidthPx
	return BetaToolUnionParam{OfComputerUseTool20251124: &variant}
}

func BetaToolUnionParamOfToolSearchToolBm25_20251119(type_ BetaToolSearchToolBm25_20251119Type) BetaToolUnionParam {
	var variant BetaToolSearchToolBm25_20251119Param
	variant.Type = type_
	return BetaToolUnionParam{OfToolSearchToolBm25_20251119: &variant}
}

func BetaToolUnionParamOfToolSearchToolRegex20251119(type_ BetaToolSearchToolRegex20251119Type) BetaToolUnionParam {
	var variant BetaToolSearchToolRegex20251119Param
	variant.Type = type_
	return BetaToolUnionParam{OfToolSearchToolRegex20251119: &variant}
}

func BetaToolUnionParamOfMCPToolset(mcpServerName string) BetaToolUnionParam {
	var variant BetaMCPToolsetParam
	variant.MCPServerName = mcpServerName
	return BetaToolUnionParam{OfMCPToolset: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaToolUnionParam struct {
	OfTool                        *BetaToolParam                        `json:",omitzero,inline"`
	OfBashTool20241022            *BetaToolBash20241022Param            `json:",omitzero,inline"`
	OfBashTool20250124            *BetaToolBash20250124Param            `json:",omitzero,inline"`
	OfCodeExecutionTool20250522   *BetaCodeExecutionTool20250522Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20250825   *BetaCodeExecutionTool20250825Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20260120   *BetaCodeExecutionTool20260120Param   `json:",omitzero,inline"`
	OfComputerUseTool20241022     *BetaToolComputerUse20241022Param     `json:",omitzero,inline"`
	OfMemoryTool20250818          *BetaMemoryTool20250818Param          `json:",omitzero,inline"`
	OfComputerUseTool20250124     *BetaToolComputerUse20250124Param     `json:",omitzero,inline"`
	OfTextEditor20241022          *BetaToolTextEditor20241022Param      `json:",omitzero,inline"`
	OfComputerUseTool20251124     *BetaToolComputerUse20251124Param     `json:",omitzero,inline"`
	OfTextEditor20250124          *BetaToolTextEditor20250124Param      `json:",omitzero,inline"`
	OfTextEditor20250429          *BetaToolTextEditor20250429Param      `json:",omitzero,inline"`
	OfTextEditor20250728          *BetaToolTextEditor20250728Param      `json:",omitzero,inline"`
	OfWebSearchTool20250305       *BetaWebSearchTool20250305Param       `json:",omitzero,inline"`
	OfWebFetchTool20250910        *BetaWebFetchTool20250910Param        `json:",omitzero,inline"`
	OfWebSearchTool20260209       *BetaWebSearchTool20260209Param       `json:",omitzero,inline"`
	OfWebFetchTool20260209        *BetaWebFetchTool20260209Param        `json:",omitzero,inline"`
	OfToolSearchToolBm25_20251119 *BetaToolSearchToolBm25_20251119Param `json:",omitzero,inline"`
	OfToolSearchToolRegex20251119 *BetaToolSearchToolRegex20251119Param `json:",omitzero,inline"`
	OfMCPToolset                  *BetaMCPToolsetParam                  `json:",omitzero,inline"`
	paramUnion
}

func (u BetaToolUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfTool,
		u.OfBashTool20241022,
		u.OfBashTool20250124,
		u.OfCodeExecutionTool20250522,
		u.OfCodeExecutionTool20250825,
		u.OfCodeExecutionTool20260120,
		u.OfComputerUseTool20241022,
		u.OfMemoryTool20250818,
		u.OfComputerUseTool20250124,
		u.OfTextEditor20241022,
		u.OfComputerUseTool20251124,
		u.OfTextEditor20250124,
		u.OfTextEditor20250429,
		u.OfTextEditor20250728,
		u.OfWebSearchTool20250305,
		u.OfWebFetchTool20250910,
		u.OfWebSearchTool20260209,
		u.OfWebFetchTool20260209,
		u.OfToolSearchToolBm25_20251119,
		u.OfToolSearchToolRegex20251119,
		u.OfMCPToolset)
}
func (u *BetaToolUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaToolUnionParam) asAny() any {
	if !param.IsOmitted(u.OfTool) {
		return u.OfTool
	} else if !param.IsOmitted(u.OfBashTool20241022) {
		return u.OfBashTool20241022
	} else if !param.IsOmitted(u.OfBashTool20250124) {
		return u.OfBashTool20250124
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250522) {
		return u.OfCodeExecutionTool20250522
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250825) {
		return u.OfCodeExecutionTool20250825
	} else if !param.IsOmitted(u.OfCodeExecutionTool20260120) {
		return u.OfCodeExecutionTool20260120
	} else if !param.IsOmitted(u.OfComputerUseTool20241022) {
		return u.OfComputerUseTool20241022
	} else if !param.IsOmitted(u.OfMemoryTool20250818) {
		return u.OfMemoryTool20250818
	} else if !param.IsOmitted(u.OfComputerUseTool20250124) {
		return u.OfComputerUseTool20250124
	} else if !param.IsOmitted(u.OfTextEditor20241022) {
		return u.OfTextEditor20241022
	} else if !param.IsOmitted(u.OfComputerUseTool20251124) {
		return u.OfComputerUseTool20251124
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
	} else if !param.IsOmitted(u.OfMCPToolset) {
		return u.OfMCPToolset
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetInputSchema() *BetaToolInputSchemaParam {
	if vt := u.OfTool; vt != nil {
		return &vt.InputSchema
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetDescription() *string {
	if vt := u.OfTool; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetEagerInputStreaming() *bool {
	if vt := u.OfTool; vt != nil && vt.EagerInputStreaming.Valid() {
		return &vt.EagerInputStreaming.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetEnableZoom() *bool {
	if vt := u.OfComputerUseTool20251124; vt != nil && vt.EnableZoom.Valid() {
		return &vt.EnableZoom.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetMaxCharacters() *int64 {
	if vt := u.OfTextEditor20250728; vt != nil && vt.MaxCharacters.Valid() {
		return &vt.MaxCharacters.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetMCPServerName() *string {
	if vt := u.OfMCPToolset; vt != nil {
		return &vt.MCPServerName
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetConfigs() map[string]BetaMCPToolConfigParam {
	if vt := u.OfMCPToolset; vt != nil {
		return vt.Configs
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetDefaultConfig() *BetaMCPToolDefaultConfigParam {
	if vt := u.OfMCPToolset; vt != nil {
		return &vt.DefaultConfig
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaToolUnionParam) GetDeferLoading() *bool {
	if vt := u.OfTool; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.DeferLoading.Valid() {
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
func (u BetaToolUnionParam) GetStrict() *bool {
	if vt := u.OfTool; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.Strict.Valid() {
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
func (u BetaToolUnionParam) GetType() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
	} else if vt := u.OfMCPToolset; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetDisplayHeightPx() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetDisplayWidthPx() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetDisplayNumber() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaToolUnionParam) GetMaxUses() *int64 {
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
func (u BetaToolUnionParam) GetMaxContentTokens() *int64 {
	if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedCallers property, if
// present.
func (u BetaToolUnionParam) GetAllowedCallers() []string {
	if vt := u.OfTool; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaToolUnionParam) GetCacheControl() *BetaCacheControlEphemeralParam {
	if vt := u.OfTool; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
	} else if vt := u.OfMCPToolset; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a pointer to the underlying variant's InputExamples property, if
// present.
func (u BetaToolUnionParam) GetInputExamples() []map[string]any {
	if vt := u.OfTool; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaToolUnionParam) GetAllowedDomains() []string {
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
func (u BetaToolUnionParam) GetBlockedDomains() []string {
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
func (u BetaToolUnionParam) GetUserLocation() *BetaUserLocationParam {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.UserLocation
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.UserLocation
	}
	return nil
}

// Returns a pointer to the underlying variant's Citations property, if present.
func (u BetaToolUnionParam) GetCitations() *BetaCitationsConfigParam {
	if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.Citations
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.Citations
	}
	return nil
}

type BetaToolUseBlock struct {
	ID    string           `json:"id,required"`
	Input any              `json:"input,required"`
	Name  string           `json:"name,required"`
	Type  constant.ToolUse `json:"type,required"`
	// Tool invocation directly from the model.
	Caller BetaToolUseBlockCallerUnion `json:"caller"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Input       respjson.Field
		Name        respjson.Field
		Type        respjson.Field
		Caller      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaToolUseBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaToolUseBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaToolUseBlockCallerUnion contains all possible properties and values from
// [BetaDirectCaller], [BetaServerToolCaller], [BetaServerToolCaller20260120].
//
// Use the [BetaToolUseBlockCallerUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaToolUseBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyBetaToolUseBlockCaller is implemented by each variant of
// [BetaToolUseBlockCallerUnion] to add type safety for the return type of
// [BetaToolUseBlockCallerUnion.AsAny]
type anyBetaToolUseBlockCaller interface {
	implBetaToolUseBlockCallerUnion()
}

func (BetaDirectCaller) implBetaToolUseBlockCallerUnion()             {}
func (BetaServerToolCaller) implBetaToolUseBlockCallerUnion()         {}
func (BetaServerToolCaller20260120) implBetaToolUseBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaToolUseBlockCallerUnion.AsAny().(type) {
//	case anthropic.BetaDirectCaller:
//	case anthropic.BetaServerToolCaller:
//	case anthropic.BetaServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaToolUseBlockCallerUnion) AsAny() anyBetaToolUseBlockCaller {
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

func (u BetaToolUseBlockCallerUnion) AsDirect() (v BetaDirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaToolUseBlockCallerUnion) AsCodeExecution20250825() (v BetaServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaToolUseBlockCallerUnion) AsCodeExecution20260120() (v BetaServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaToolUseBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaToolUseBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ID, Input, Name, Type are required.
type BetaToolUseBlockParam struct {
	ID    string `json:"id,required"`
	Input any    `json:"input,omitzero,required"`
	Name  string `json:"name,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller BetaToolUseBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as "tool_use".
	Type constant.ToolUse `json:"type,required"`
	paramObj
}

func (r BetaToolUseBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolUseBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolUseBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaToolUseBlockParamCallerUnion struct {
	OfDirect                *BetaDirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *BetaServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *BetaServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u BetaToolUseBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *BetaToolUseBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaToolUseBlockParamCallerUnion) asAny() any {
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
func (u BetaToolUseBlockParamCallerUnion) GetType() *string {
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
func (u BetaToolUseBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[BetaToolUseBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[BetaServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

// The properties Type, Value are required.
type BetaToolUsesKeepParam struct {
	Value int64 `json:"value,required"`
	// This field can be elided, and will marshal its zero value as "tool_uses".
	Type constant.ToolUses `json:"type,required"`
	paramObj
}

func (r BetaToolUsesKeepParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolUsesKeepParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolUsesKeepParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, Value are required.
type BetaToolUsesTriggerParam struct {
	Value int64 `json:"value,required"`
	// This field can be elided, and will marshal its zero value as "tool_uses".
	Type constant.ToolUses `json:"type,required"`
	paramObj
}

func (r BetaToolUsesTriggerParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaToolUsesTriggerParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaToolUsesTriggerParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, URL are required.
type BetaURLImageSourceParam struct {
	URL string `json:"url,required"`
	// This field can be elided, and will marshal its zero value as "url".
	Type constant.URL `json:"type,required"`
	paramObj
}

func (r BetaURLImageSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaURLImageSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaURLImageSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Type, URL are required.
type BetaURLPDFSourceParam struct {
	URL string `json:"url,required"`
	// This field can be elided, and will marshal its zero value as "url".
	Type constant.URL `json:"type,required"`
	paramObj
}

func (r BetaURLPDFSourceParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaURLPDFSourceParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaURLPDFSourceParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaUsage struct {
	// Breakdown of cached tokens by TTL
	CacheCreation BetaCacheCreation `json:"cache_creation,required"`
	// The number of input tokens used to create the cache entry.
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,required"`
	// The number of input tokens read from the cache.
	CacheReadInputTokens int64 `json:"cache_read_input_tokens,required"`
	// The geographic region where inference was performed for this request.
	InferenceGeo string `json:"inference_geo,required"`
	// The number of input tokens which were used.
	InputTokens int64 `json:"input_tokens,required"`
	// Per-iteration token usage breakdown.
	//
	// Each entry represents one sampling iteration, with its own input/output token
	// counts and cache statistics. This allows you to:
	//
	// - Determine which iterations exceeded long context thresholds (>=200k tokens)
	// - Calculate the true context window size from the last iteration
	// - Understand token accumulation across server-side tool use loops
	Iterations BetaIterationsUsage `json:"iterations,required"`
	// The number of output tokens which were used.
	OutputTokens int64 `json:"output_tokens,required"`
	// The number of server tool requests.
	ServerToolUse BetaServerToolUsage `json:"server_tool_use,required"`
	// If the request used the priority, standard, or batch tier.
	//
	// Any of "standard", "priority", "batch".
	ServiceTier BetaUsageServiceTier `json:"service_tier,required"`
	// The inference speed mode used for this request.
	//
	// Any of "standard", "fast".
	Speed BetaUsageSpeed `json:"speed,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		CacheCreation            respjson.Field
		CacheCreationInputTokens respjson.Field
		CacheReadInputTokens     respjson.Field
		InferenceGeo             respjson.Field
		InputTokens              respjson.Field
		Iterations               respjson.Field
		OutputTokens             respjson.Field
		ServerToolUse            respjson.Field
		ServiceTier              respjson.Field
		Speed                    respjson.Field
		ExtraFields              map[string]respjson.Field
		raw                      string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaUsage) RawJSON() string { return r.JSON.raw }
func (r *BetaUsage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// If the request used the priority, standard, or batch tier.
type BetaUsageServiceTier string

const (
	BetaUsageServiceTierStandard BetaUsageServiceTier = "standard"
	BetaUsageServiceTierPriority BetaUsageServiceTier = "priority"
	BetaUsageServiceTierBatch    BetaUsageServiceTier = "batch"
)

// The inference speed mode used for this request.
type BetaUsageSpeed string

const (
	BetaUsageSpeedStandard BetaUsageSpeed = "standard"
	BetaUsageSpeedFast     BetaUsageSpeed = "fast"
)

// The property Type is required.
type BetaUserLocationParam struct {
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

func (r BetaUserLocationParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaUserLocationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaUserLocationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaWebFetchBlock struct {
	Content BetaDocumentBlock `json:"content,required"`
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
func (r BetaWebFetchBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaWebFetchBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, Type, URL are required.
type BetaWebFetchBlockParam struct {
	Content BetaRequestDocumentBlockParam `json:"content,omitzero,required"`
	// Fetched content URL
	URL string `json:"url,required"`
	// ISO 8601 timestamp when the content was retrieved
	RetrievedAt param.Opt[string] `json:"retrieved_at,omitzero"`
	// This field can be elided, and will marshal its zero value as "web_fetch_result".
	Type constant.WebFetchResult `json:"type,required"`
	paramObj
}

func (r BetaWebFetchBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebFetchBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebFetchBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaWebFetchTool20250910Param struct {
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
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Citations configuration for fetched documents. Citations are disabled by
	// default.
	Citations BetaCitationsConfigParam `json:"citations,omitzero"`
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

func (r BetaWebFetchTool20250910Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebFetchTool20250910Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebFetchTool20250910Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaWebFetchTool20260209Param struct {
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
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Citations configuration for fetched documents. Citations are disabled by
	// default.
	Citations BetaCitationsConfigParam `json:"citations,omitzero"`
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

func (r BetaWebFetchTool20260209Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebFetchTool20260209Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebFetchTool20260209Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaWebFetchToolResultBlock struct {
	Content   BetaWebFetchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                  `json:"tool_use_id,required"`
	Type      constant.WebFetchToolResult             `json:"type,required"`
	// Tool invocation directly from the model.
	Caller BetaWebFetchToolResultBlockCallerUnion `json:"caller"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		Caller      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaWebFetchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaWebFetchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaWebFetchToolResultBlockContentUnion contains all possible properties and
// values from [BetaWebFetchToolResultErrorBlock], [BetaWebFetchBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaWebFetchToolResultBlockContentUnion struct {
	// This field is from variant [BetaWebFetchToolResultErrorBlock].
	ErrorCode BetaWebFetchToolResultErrorCode `json:"error_code"`
	Type      string                          `json:"type"`
	// This field is from variant [BetaWebFetchBlock].
	Content BetaDocumentBlock `json:"content"`
	// This field is from variant [BetaWebFetchBlock].
	RetrievedAt string `json:"retrieved_at"`
	// This field is from variant [BetaWebFetchBlock].
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

func (u BetaWebFetchToolResultBlockContentUnion) AsResponseWebFetchToolResultError() (v BetaWebFetchToolResultErrorBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebFetchToolResultBlockContentUnion) AsResponseWebFetchResultBlock() (v BetaWebFetchBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaWebFetchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaWebFetchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaWebFetchToolResultBlockCallerUnion contains all possible properties and
// values from [BetaDirectCaller], [BetaServerToolCaller],
// [BetaServerToolCaller20260120].
//
// Use the [BetaWebFetchToolResultBlockCallerUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaWebFetchToolResultBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyBetaWebFetchToolResultBlockCaller is implemented by each variant of
// [BetaWebFetchToolResultBlockCallerUnion] to add type safety for the return type
// of [BetaWebFetchToolResultBlockCallerUnion.AsAny]
type anyBetaWebFetchToolResultBlockCaller interface {
	implBetaWebFetchToolResultBlockCallerUnion()
}

func (BetaDirectCaller) implBetaWebFetchToolResultBlockCallerUnion()             {}
func (BetaServerToolCaller) implBetaWebFetchToolResultBlockCallerUnion()         {}
func (BetaServerToolCaller20260120) implBetaWebFetchToolResultBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaWebFetchToolResultBlockCallerUnion.AsAny().(type) {
//	case anthropic.BetaDirectCaller:
//	case anthropic.BetaServerToolCaller:
//	case anthropic.BetaServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaWebFetchToolResultBlockCallerUnion) AsAny() anyBetaWebFetchToolResultBlockCaller {
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

func (u BetaWebFetchToolResultBlockCallerUnion) AsDirect() (v BetaDirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebFetchToolResultBlockCallerUnion) AsCodeExecution20250825() (v BetaServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebFetchToolResultBlockCallerUnion) AsCodeExecution20260120() (v BetaServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaWebFetchToolResultBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaWebFetchToolResultBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaWebFetchToolResultBlockParam struct {
	Content   BetaWebFetchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                       `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller BetaWebFetchToolResultBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_tool_result".
	Type constant.WebFetchToolResult `json:"type,required"`
	paramObj
}

func (r BetaWebFetchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebFetchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebFetchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaWebFetchToolResultBlockParamContentUnion struct {
	OfRequestWebFetchToolResultError *BetaWebFetchToolResultErrorBlockParam `json:",omitzero,inline"`
	OfRequestWebFetchResultBlock     *BetaWebFetchBlockParam                `json:",omitzero,inline"`
	paramUnion
}

func (u BetaWebFetchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRequestWebFetchToolResultError, u.OfRequestWebFetchResultBlock)
}
func (u *BetaWebFetchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaWebFetchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfRequestWebFetchToolResultError) {
		return u.OfRequestWebFetchToolResultError
	} else if !param.IsOmitted(u.OfRequestWebFetchResultBlock) {
		return u.OfRequestWebFetchResultBlock
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaWebFetchToolResultBlockParamContentUnion) GetErrorCode() *string {
	if vt := u.OfRequestWebFetchToolResultError; vt != nil {
		return (*string)(&vt.ErrorCode)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaWebFetchToolResultBlockParamContentUnion) GetContent() *BetaRequestDocumentBlockParam {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil {
		return &vt.Content
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaWebFetchToolResultBlockParamContentUnion) GetURL() *string {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil {
		return &vt.URL
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaWebFetchToolResultBlockParamContentUnion) GetRetrievedAt() *string {
	if vt := u.OfRequestWebFetchResultBlock; vt != nil && vt.RetrievedAt.Valid() {
		return &vt.RetrievedAt.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaWebFetchToolResultBlockParamContentUnion) GetType() *string {
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
type BetaWebFetchToolResultBlockParamCallerUnion struct {
	OfDirect                *BetaDirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *BetaServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *BetaServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u BetaWebFetchToolResultBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *BetaWebFetchToolResultBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaWebFetchToolResultBlockParamCallerUnion) asAny() any {
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
func (u BetaWebFetchToolResultBlockParamCallerUnion) GetType() *string {
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
func (u BetaWebFetchToolResultBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[BetaWebFetchToolResultBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[BetaServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

type BetaWebFetchToolResultErrorBlock struct {
	// Any of "invalid_tool_input", "url_too_long", "url_not_allowed",
	// "url_not_accessible", "unsupported_content_type", "too_many_requests",
	// "max_uses_exceeded", "unavailable".
	ErrorCode BetaWebFetchToolResultErrorCode  `json:"error_code,required"`
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
func (r BetaWebFetchToolResultErrorBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaWebFetchToolResultErrorBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ErrorCode, Type are required.
type BetaWebFetchToolResultErrorBlockParam struct {
	// Any of "invalid_tool_input", "url_too_long", "url_not_allowed",
	// "url_not_accessible", "unsupported_content_type", "too_many_requests",
	// "max_uses_exceeded", "unavailable".
	ErrorCode BetaWebFetchToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_fetch_tool_result_error".
	Type constant.WebFetchToolResultError `json:"type,required"`
	paramObj
}

func (r BetaWebFetchToolResultErrorBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebFetchToolResultErrorBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebFetchToolResultErrorBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaWebFetchToolResultErrorCode string

const (
	BetaWebFetchToolResultErrorCodeInvalidToolInput       BetaWebFetchToolResultErrorCode = "invalid_tool_input"
	BetaWebFetchToolResultErrorCodeURLTooLong             BetaWebFetchToolResultErrorCode = "url_too_long"
	BetaWebFetchToolResultErrorCodeURLNotAllowed          BetaWebFetchToolResultErrorCode = "url_not_allowed"
	BetaWebFetchToolResultErrorCodeURLNotAccessible       BetaWebFetchToolResultErrorCode = "url_not_accessible"
	BetaWebFetchToolResultErrorCodeUnsupportedContentType BetaWebFetchToolResultErrorCode = "unsupported_content_type"
	BetaWebFetchToolResultErrorCodeTooManyRequests        BetaWebFetchToolResultErrorCode = "too_many_requests"
	BetaWebFetchToolResultErrorCodeMaxUsesExceeded        BetaWebFetchToolResultErrorCode = "max_uses_exceeded"
	BetaWebFetchToolResultErrorCodeUnavailable            BetaWebFetchToolResultErrorCode = "unavailable"
)

type BetaWebSearchResultBlock struct {
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
func (r BetaWebSearchResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaWebSearchResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties EncryptedContent, Title, Type, URL are required.
type BetaWebSearchResultBlockParam struct {
	EncryptedContent string            `json:"encrypted_content,required"`
	Title            string            `json:"title,required"`
	URL              string            `json:"url,required"`
	PageAge          param.Opt[string] `json:"page_age,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_result".
	Type constant.WebSearchResult `json:"type,required"`
	paramObj
}

func (r BetaWebSearchResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebSearchResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebSearchResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaWebSearchTool20250305Param struct {
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
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Parameters for the user's location. Used to provide more relevant search
	// results.
	UserLocation BetaUserLocationParam `json:"user_location,omitzero"`
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

func (r BetaWebSearchTool20250305Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebSearchTool20250305Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebSearchTool20250305Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Name, Type are required.
type BetaWebSearchTool20260209Param struct {
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
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Parameters for the user's location. Used to provide more relevant search
	// results.
	UserLocation BetaUserLocationParam `json:"user_location,omitzero"`
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

func (r BetaWebSearchTool20260209Param) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebSearchTool20260209Param
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebSearchTool20260209Param) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties ErrorCode, Type are required.
type BetaWebSearchToolRequestErrorParam struct {
	// Any of "invalid_tool_input", "unavailable", "max_uses_exceeded",
	// "too_many_requests", "query_too_long", "request_too_large".
	ErrorCode BetaWebSearchToolResultErrorCode `json:"error_code,omitzero,required"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_tool_result_error".
	Type constant.WebSearchToolResultError `json:"type,required"`
	paramObj
}

func (r BetaWebSearchToolRequestErrorParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebSearchToolRequestErrorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebSearchToolRequestErrorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaWebSearchToolResultBlock struct {
	Content   BetaWebSearchToolResultBlockContentUnion `json:"content,required"`
	ToolUseID string                                   `json:"tool_use_id,required"`
	Type      constant.WebSearchToolResult             `json:"type,required"`
	// Tool invocation directly from the model.
	Caller BetaWebSearchToolResultBlockCallerUnion `json:"caller"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Content     respjson.Field
		ToolUseID   respjson.Field
		Type        respjson.Field
		Caller      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BetaWebSearchToolResultBlock) RawJSON() string { return r.JSON.raw }
func (r *BetaWebSearchToolResultBlock) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaWebSearchToolResultBlockCallerUnion contains all possible properties and
// values from [BetaDirectCaller], [BetaServerToolCaller],
// [BetaServerToolCaller20260120].
//
// Use the [BetaWebSearchToolResultBlockCallerUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type BetaWebSearchToolResultBlockCallerUnion struct {
	// Any of "direct", "code_execution_20250825", "code_execution_20260120".
	Type   string `json:"type"`
	ToolID string `json:"tool_id"`
	JSON   struct {
		Type   respjson.Field
		ToolID respjson.Field
		raw    string
	} `json:"-"`
}

// anyBetaWebSearchToolResultBlockCaller is implemented by each variant of
// [BetaWebSearchToolResultBlockCallerUnion] to add type safety for the return type
// of [BetaWebSearchToolResultBlockCallerUnion.AsAny]
type anyBetaWebSearchToolResultBlockCaller interface {
	implBetaWebSearchToolResultBlockCallerUnion()
}

func (BetaDirectCaller) implBetaWebSearchToolResultBlockCallerUnion()             {}
func (BetaServerToolCaller) implBetaWebSearchToolResultBlockCallerUnion()         {}
func (BetaServerToolCaller20260120) implBetaWebSearchToolResultBlockCallerUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := BetaWebSearchToolResultBlockCallerUnion.AsAny().(type) {
//	case anthropic.BetaDirectCaller:
//	case anthropic.BetaServerToolCaller:
//	case anthropic.BetaServerToolCaller20260120:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u BetaWebSearchToolResultBlockCallerUnion) AsAny() anyBetaWebSearchToolResultBlockCaller {
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

func (u BetaWebSearchToolResultBlockCallerUnion) AsDirect() (v BetaDirectCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebSearchToolResultBlockCallerUnion) AsCodeExecution20250825() (v BetaServerToolCaller) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebSearchToolResultBlockCallerUnion) AsCodeExecution20260120() (v BetaServerToolCaller20260120) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaWebSearchToolResultBlockCallerUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaWebSearchToolResultBlockCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// BetaWebSearchToolResultBlockContentUnion contains all possible properties and
// values from [BetaWebSearchToolResultError], [[]BetaWebSearchResultBlock].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfBetaWebSearchResultBlockArray]
type BetaWebSearchToolResultBlockContentUnion struct {
	// This field will be present if the value is a [[]BetaWebSearchResultBlock]
	// instead of an object.
	OfBetaWebSearchResultBlockArray []BetaWebSearchResultBlock `json:",inline"`
	// This field is from variant [BetaWebSearchToolResultError].
	ErrorCode BetaWebSearchToolResultErrorCode `json:"error_code"`
	// This field is from variant [BetaWebSearchToolResultError].
	Type constant.WebSearchToolResultError `json:"type"`
	JSON struct {
		OfBetaWebSearchResultBlockArray respjson.Field
		ErrorCode                       respjson.Field
		Type                            respjson.Field
		raw                             string
	} `json:"-"`
}

func (u BetaWebSearchToolResultBlockContentUnion) AsResponseWebSearchToolResultError() (v BetaWebSearchToolResultError) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u BetaWebSearchToolResultBlockContentUnion) AsBetaWebSearchResultBlockArray() (v []BetaWebSearchResultBlock) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u BetaWebSearchToolResultBlockContentUnion) RawJSON() string { return u.JSON.raw }

func (r *BetaWebSearchToolResultBlockContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The properties Content, ToolUseID, Type are required.
type BetaWebSearchToolResultBlockParam struct {
	Content   BetaWebSearchToolResultBlockParamContentUnion `json:"content,omitzero,required"`
	ToolUseID string                                        `json:"tool_use_id,required"`
	// Create a cache control breakpoint at this content block.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Tool invocation directly from the model.
	Caller BetaWebSearchToolResultBlockParamCallerUnion `json:"caller,omitzero"`
	// This field can be elided, and will marshal its zero value as
	// "web_search_tool_result".
	Type constant.WebSearchToolResult `json:"type,required"`
	paramObj
}

func (r BetaWebSearchToolResultBlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BetaWebSearchToolResultBlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaWebSearchToolResultBlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaWebSearchToolResultBlockParamCallerUnion struct {
	OfDirect                *BetaDirectCallerParam             `json:",omitzero,inline"`
	OfCodeExecution20250825 *BetaServerToolCallerParam         `json:",omitzero,inline"`
	OfCodeExecution20260120 *BetaServerToolCaller20260120Param `json:",omitzero,inline"`
	paramUnion
}

func (u BetaWebSearchToolResultBlockParamCallerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfDirect, u.OfCodeExecution20250825, u.OfCodeExecution20260120)
}
func (u *BetaWebSearchToolResultBlockParamCallerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaWebSearchToolResultBlockParamCallerUnion) asAny() any {
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
func (u BetaWebSearchToolResultBlockParamCallerUnion) GetType() *string {
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
func (u BetaWebSearchToolResultBlockParamCallerUnion) GetToolID() *string {
	if vt := u.OfCodeExecution20250825; vt != nil {
		return (*string)(&vt.ToolID)
	} else if vt := u.OfCodeExecution20260120; vt != nil {
		return (*string)(&vt.ToolID)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[BetaWebSearchToolResultBlockParamCallerUnion](
		"type",
		apijson.Discriminator[BetaDirectCallerParam]("direct"),
		apijson.Discriminator[BetaServerToolCallerParam]("code_execution_20250825"),
		apijson.Discriminator[BetaServerToolCaller20260120Param]("code_execution_20260120"),
	)
}

func BetaNewWebSearchToolRequestError(errorCode BetaWebSearchToolResultErrorCode) BetaWebSearchToolResultBlockParamContentUnion {
	var variant BetaWebSearchToolRequestErrorParam
	variant.ErrorCode = errorCode
	return BetaWebSearchToolResultBlockParamContentUnion{OfError: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaWebSearchToolResultBlockParamContentUnion struct {
	OfResultBlock []BetaWebSearchResultBlockParam     `json:",omitzero,inline"`
	OfError       *BetaWebSearchToolRequestErrorParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaWebSearchToolResultBlockParamContentUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfResultBlock, u.OfError)
}
func (u *BetaWebSearchToolResultBlockParamContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaWebSearchToolResultBlockParamContentUnion) asAny() any {
	if !param.IsOmitted(u.OfResultBlock) {
		return &u.OfResultBlock
	} else if !param.IsOmitted(u.OfError) {
		return u.OfError
	}
	return nil
}

type BetaWebSearchToolResultError struct {
	// Any of "invalid_tool_input", "unavailable", "max_uses_exceeded",
	// "too_many_requests", "query_too_long", "request_too_large".
	ErrorCode BetaWebSearchToolResultErrorCode  `json:"error_code,required"`
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
func (r BetaWebSearchToolResultError) RawJSON() string { return r.JSON.raw }
func (r *BetaWebSearchToolResultError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BetaWebSearchToolResultErrorCode string

const (
	BetaWebSearchToolResultErrorCodeInvalidToolInput BetaWebSearchToolResultErrorCode = "invalid_tool_input"
	BetaWebSearchToolResultErrorCodeUnavailable      BetaWebSearchToolResultErrorCode = "unavailable"
	BetaWebSearchToolResultErrorCodeMaxUsesExceeded  BetaWebSearchToolResultErrorCode = "max_uses_exceeded"
	BetaWebSearchToolResultErrorCodeTooManyRequests  BetaWebSearchToolResultErrorCode = "too_many_requests"
	BetaWebSearchToolResultErrorCodeQueryTooLong     BetaWebSearchToolResultErrorCode = "query_too_long"
	BetaWebSearchToolResultErrorCodeRequestTooLarge  BetaWebSearchToolResultErrorCode = "request_too_large"
)

type BetaMessageNewParams struct {
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
	Messages []BetaMessageParam `json:"messages,omitzero,required"`
	// The model that will complete your prompt.\n\nSee
	// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
	// details and options.
	Model Model `json:"model,omitzero,required"`
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
	// Container identifier for reuse across requests.
	Container BetaMessageNewParamsContainerUnion `json:"container,omitzero"`
	// The inference speed mode for this request. `"fast"` enables high
	// output-tokens-per-second inference.
	//
	// Any of "standard", "fast".
	Speed BetaMessageNewParamsSpeed `json:"speed,omitzero"`
	// Top-level cache control automatically applies a cache_control marker to the last
	// cacheable block in the request.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Context management configuration.
	//
	// This allows you to control how Claude manages context across multiple requests,
	// such as whether to clear function results or not.
	ContextManagement BetaContextManagementConfigParam `json:"context_management,omitzero"`
	// MCP servers to be utilized in this request
	MCPServers []BetaRequestMCPServerURLDefinitionParam `json:"mcp_servers,omitzero"`
	// An object describing metadata about the request.
	Metadata BetaMetadataParam `json:"metadata,omitzero"`
	// Configuration options for the model's output, such as the output format.
	OutputConfig BetaOutputConfigParam `json:"output_config,omitzero"`
	// Deprecated: Use `output_config.format` instead. See
	// [structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
	//
	// A schema to specify Claude's output format in responses. This parameter will be
	// removed in a future release.
	OutputFormat BetaJSONOutputFormatParam `json:"output_format,omitzero"`
	// Determines whether to use priority capacity (if available) or standard capacity
	// for this request.
	//
	// Anthropic offers different levels of service for your API requests. See
	// [service-tiers](https://docs.claude.com/en/api/service-tiers) for details.
	//
	// Any of "auto", "standard_only".
	ServiceTier BetaMessageNewParamsServiceTier `json:"service_tier,omitzero"`
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
	System []BetaTextBlockParam `json:"system,omitzero"`
	// Configuration for enabling Claude's extended thinking.
	//
	// When enabled, responses include `thinking` content blocks showing Claude's
	// thinking process before the final answer. Requires a minimum budget of 1,024
	// tokens and counts towards your `max_tokens` limit.
	//
	// See
	// [extended thinking](https://docs.claude.com/en/docs/build-with-claude/extended-thinking)
	// for details.
	Thinking BetaThinkingConfigParamUnion `json:"thinking,omitzero"`
	// How the model should use the provided tools. The model can use a specific tool,
	// any available tool, decide by itself, or not use tools at all.
	ToolChoice BetaToolChoiceUnionParam `json:"tool_choice,omitzero"`
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
	Tools []BetaToolUnionParam `json:"tools,omitzero"`
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}

func (r BetaMessageNewParams) MarshalJSON() (data []byte, err error) {
	type shadow BetaMessageNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMessageNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaMessageNewParamsContainerUnion struct {
	OfContainers *BetaContainerParams `json:",omitzero,inline"`
	OfString     param.Opt[string]    `json:",omitzero,inline"`
	paramUnion
}

func (u BetaMessageNewParamsContainerUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfContainers, u.OfString)
}
func (u *BetaMessageNewParamsContainerUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaMessageNewParamsContainerUnion) asAny() any {
	if !param.IsOmitted(u.OfContainers) {
		return u.OfContainers
	} else if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	}
	return nil
}

// Determines whether to use priority capacity (if available) or standard capacity
// for this request.
//
// Anthropic offers different levels of service for your API requests. See
// [service-tiers](https://docs.claude.com/en/api/service-tiers) for details.
type BetaMessageNewParamsServiceTier string

const (
	BetaMessageNewParamsServiceTierAuto         BetaMessageNewParamsServiceTier = "auto"
	BetaMessageNewParamsServiceTierStandardOnly BetaMessageNewParamsServiceTier = "standard_only"
)

// The inference speed mode for this request. `"fast"` enables high
// output-tokens-per-second inference.
type BetaMessageNewParamsSpeed string

const (
	BetaMessageNewParamsSpeedStandard BetaMessageNewParamsSpeed = "standard"
	BetaMessageNewParamsSpeedFast     BetaMessageNewParamsSpeed = "fast"
)

type BetaMessageCountTokensParams struct {
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
	Messages []BetaMessageParam `json:"messages,omitzero,required"`
	// The model that will complete your prompt.\n\nSee
	// [models](https://docs.anthropic.com/en/docs/models-overview) for additional
	// details and options.
	Model Model `json:"model,omitzero,required"`
	// The inference speed mode for this request. `"fast"` enables high
	// output-tokens-per-second inference.
	//
	// Any of "standard", "fast".
	Speed BetaMessageCountTokensParamsSpeed `json:"speed,omitzero"`
	// Top-level cache control automatically applies a cache_control marker to the last
	// cacheable block in the request.
	CacheControl BetaCacheControlEphemeralParam `json:"cache_control,omitzero"`
	// Context management configuration.
	//
	// This allows you to control how Claude manages context across multiple requests,
	// such as whether to clear function results or not.
	ContextManagement BetaContextManagementConfigParam `json:"context_management,omitzero"`
	// MCP servers to be utilized in this request
	MCPServers []BetaRequestMCPServerURLDefinitionParam `json:"mcp_servers,omitzero"`
	// Configuration options for the model's output, such as the output format.
	OutputConfig BetaOutputConfigParam `json:"output_config,omitzero"`
	// Deprecated: Use `output_config.format` instead. See
	// [structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
	//
	// A schema to specify Claude's output format in responses. This parameter will be
	// removed in a future release.
	OutputFormat BetaJSONOutputFormatParam `json:"output_format,omitzero"`
	// System prompt.
	//
	// A system prompt is a way of providing context and instructions to Claude, such
	// as specifying a particular goal or role. See our
	// [guide to system prompts](https://docs.claude.com/en/docs/system-prompts).
	System BetaMessageCountTokensParamsSystemUnion `json:"system,omitzero"`
	// Configuration for enabling Claude's extended thinking.
	//
	// When enabled, responses include `thinking` content blocks showing Claude's
	// thinking process before the final answer. Requires a minimum budget of 1,024
	// tokens and counts towards your `max_tokens` limit.
	//
	// See
	// [extended thinking](https://docs.claude.com/en/docs/build-with-claude/extended-thinking)
	// for details.
	Thinking BetaThinkingConfigParamUnion `json:"thinking,omitzero"`
	// How the model should use the provided tools. The model can use a specific tool,
	// any available tool, decide by itself, or not use tools at all.
	ToolChoice BetaToolChoiceUnionParam `json:"tool_choice,omitzero"`
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
	Tools []BetaMessageCountTokensParamsToolUnion `json:"tools,omitzero"`
	// Optional header to specify the beta version(s) you want to use.
	Betas []AnthropicBeta `header:"anthropic-beta,omitzero" json:"-"`
	paramObj
}

func (r BetaMessageCountTokensParams) MarshalJSON() (data []byte, err error) {
	type shadow BetaMessageCountTokensParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BetaMessageCountTokensParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The inference speed mode for this request. `"fast"` enables high
// output-tokens-per-second inference.
type BetaMessageCountTokensParamsSpeed string

const (
	BetaMessageCountTokensParamsSpeedStandard BetaMessageCountTokensParamsSpeed = "standard"
	BetaMessageCountTokensParamsSpeedFast     BetaMessageCountTokensParamsSpeed = "fast"
)

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaMessageCountTokensParamsSystemUnion struct {
	OfString             param.Opt[string]    `json:",omitzero,inline"`
	OfBetaTextBlockArray []BetaTextBlockParam `json:",omitzero,inline"`
	paramUnion
}

func (u BetaMessageCountTokensParamsSystemUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfString, u.OfBetaTextBlockArray)
}
func (u *BetaMessageCountTokensParamsSystemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaMessageCountTokensParamsSystemUnion) asAny() any {
	if !param.IsOmitted(u.OfString) {
		return &u.OfString.Value
	} else if !param.IsOmitted(u.OfBetaTextBlockArray) {
		return &u.OfBetaTextBlockArray
	}
	return nil
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type BetaMessageCountTokensParamsToolUnion struct {
	OfTool                        *BetaToolParam                        `json:",omitzero,inline"`
	OfBashTool20241022            *BetaToolBash20241022Param            `json:",omitzero,inline"`
	OfBashTool20250124            *BetaToolBash20250124Param            `json:",omitzero,inline"`
	OfCodeExecutionTool20250522   *BetaCodeExecutionTool20250522Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20250825   *BetaCodeExecutionTool20250825Param   `json:",omitzero,inline"`
	OfCodeExecutionTool20260120   *BetaCodeExecutionTool20260120Param   `json:",omitzero,inline"`
	OfComputerUseTool20241022     *BetaToolComputerUse20241022Param     `json:",omitzero,inline"`
	OfMemoryTool20250818          *BetaMemoryTool20250818Param          `json:",omitzero,inline"`
	OfComputerUseTool20250124     *BetaToolComputerUse20250124Param     `json:",omitzero,inline"`
	OfTextEditor20241022          *BetaToolTextEditor20241022Param      `json:",omitzero,inline"`
	OfComputerUseTool20251124     *BetaToolComputerUse20251124Param     `json:",omitzero,inline"`
	OfTextEditor20250124          *BetaToolTextEditor20250124Param      `json:",omitzero,inline"`
	OfTextEditor20250429          *BetaToolTextEditor20250429Param      `json:",omitzero,inline"`
	OfTextEditor20250728          *BetaToolTextEditor20250728Param      `json:",omitzero,inline"`
	OfWebSearchTool20250305       *BetaWebSearchTool20250305Param       `json:",omitzero,inline"`
	OfWebFetchTool20250910        *BetaWebFetchTool20250910Param        `json:",omitzero,inline"`
	OfWebSearchTool20260209       *BetaWebSearchTool20260209Param       `json:",omitzero,inline"`
	OfWebFetchTool20260209        *BetaWebFetchTool20260209Param        `json:",omitzero,inline"`
	OfToolSearchToolBm25_20251119 *BetaToolSearchToolBm25_20251119Param `json:",omitzero,inline"`
	OfToolSearchToolRegex20251119 *BetaToolSearchToolRegex20251119Param `json:",omitzero,inline"`
	OfMCPToolset                  *BetaMCPToolsetParam                  `json:",omitzero,inline"`
	paramUnion
}

func (u BetaMessageCountTokensParamsToolUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfTool,
		u.OfBashTool20241022,
		u.OfBashTool20250124,
		u.OfCodeExecutionTool20250522,
		u.OfCodeExecutionTool20250825,
		u.OfCodeExecutionTool20260120,
		u.OfComputerUseTool20241022,
		u.OfMemoryTool20250818,
		u.OfComputerUseTool20250124,
		u.OfTextEditor20241022,
		u.OfComputerUseTool20251124,
		u.OfTextEditor20250124,
		u.OfTextEditor20250429,
		u.OfTextEditor20250728,
		u.OfWebSearchTool20250305,
		u.OfWebFetchTool20250910,
		u.OfWebSearchTool20260209,
		u.OfWebFetchTool20260209,
		u.OfToolSearchToolBm25_20251119,
		u.OfToolSearchToolRegex20251119,
		u.OfMCPToolset)
}
func (u *BetaMessageCountTokensParamsToolUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *BetaMessageCountTokensParamsToolUnion) asAny() any {
	if !param.IsOmitted(u.OfTool) {
		return u.OfTool
	} else if !param.IsOmitted(u.OfBashTool20241022) {
		return u.OfBashTool20241022
	} else if !param.IsOmitted(u.OfBashTool20250124) {
		return u.OfBashTool20250124
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250522) {
		return u.OfCodeExecutionTool20250522
	} else if !param.IsOmitted(u.OfCodeExecutionTool20250825) {
		return u.OfCodeExecutionTool20250825
	} else if !param.IsOmitted(u.OfCodeExecutionTool20260120) {
		return u.OfCodeExecutionTool20260120
	} else if !param.IsOmitted(u.OfComputerUseTool20241022) {
		return u.OfComputerUseTool20241022
	} else if !param.IsOmitted(u.OfMemoryTool20250818) {
		return u.OfMemoryTool20250818
	} else if !param.IsOmitted(u.OfComputerUseTool20250124) {
		return u.OfComputerUseTool20250124
	} else if !param.IsOmitted(u.OfTextEditor20241022) {
		return u.OfTextEditor20241022
	} else if !param.IsOmitted(u.OfComputerUseTool20251124) {
		return u.OfComputerUseTool20251124
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
	} else if !param.IsOmitted(u.OfMCPToolset) {
		return u.OfMCPToolset
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetInputSchema() *BetaToolInputSchemaParam {
	if vt := u.OfTool; vt != nil {
		return &vt.InputSchema
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetDescription() *string {
	if vt := u.OfTool; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetEagerInputStreaming() *bool {
	if vt := u.OfTool; vt != nil && vt.EagerInputStreaming.Valid() {
		return &vt.EagerInputStreaming.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetEnableZoom() *bool {
	if vt := u.OfComputerUseTool20251124; vt != nil && vt.EnableZoom.Valid() {
		return &vt.EnableZoom.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetMaxCharacters() *int64 {
	if vt := u.OfTextEditor20250728; vt != nil && vt.MaxCharacters.Valid() {
		return &vt.MaxCharacters.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetMCPServerName() *string {
	if vt := u.OfMCPToolset; vt != nil {
		return &vt.MCPServerName
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetConfigs() map[string]BetaMCPToolConfigParam {
	if vt := u.OfMCPToolset; vt != nil {
		return vt.Configs
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetDefaultConfig() *BetaMCPToolDefaultConfigParam {
	if vt := u.OfMCPToolset; vt != nil {
		return &vt.DefaultConfig
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetName() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaMessageCountTokensParamsToolUnion) GetDeferLoading() *bool {
	if vt := u.OfTool; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfTextEditor20241022; vt != nil && vt.DeferLoading.Valid() {
		return &vt.DeferLoading.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.DeferLoading.Valid() {
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
func (u BetaMessageCountTokensParamsToolUnion) GetStrict() *bool {
	if vt := u.OfTool; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfBashTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfMemoryTool20250818; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfTextEditor20241022; vt != nil && vt.Strict.Valid() {
		return &vt.Strict.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.Strict.Valid() {
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
func (u BetaMessageCountTokensParamsToolUnion) GetType() *string {
	if vt := u.OfTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfBashTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
	} else if vt := u.OfMCPToolset; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetDisplayHeightPx() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
		return (*int64)(&vt.DisplayHeightPx)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetDisplayWidthPx() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
		return (*int64)(&vt.DisplayWidthPx)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetDisplayNumber() *int64 {
	if vt := u.OfComputerUseTool20241022; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	} else if vt := u.OfComputerUseTool20250124; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	} else if vt := u.OfComputerUseTool20251124; vt != nil && vt.DisplayNumber.Valid() {
		return &vt.DisplayNumber.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetMaxUses() *int64 {
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
func (u BetaMessageCountTokensParamsToolUnion) GetMaxContentTokens() *int64 {
	if vt := u.OfWebFetchTool20250910; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	} else if vt := u.OfWebFetchTool20260209; vt != nil && vt.MaxContentTokens.Valid() {
		return &vt.MaxContentTokens.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's AllowedCallers property, if
// present.
func (u BetaMessageCountTokensParamsToolUnion) GetAllowedCallers() []string {
	if vt := u.OfTool; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return vt.AllowedCallers
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaMessageCountTokensParamsToolUnion) GetCacheControl() *BetaCacheControlEphemeralParam {
	if vt := u.OfTool; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfBashTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250522; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20250825; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfCodeExecutionTool20260120; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return &vt.CacheControl
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
	} else if vt := u.OfMCPToolset; vt != nil {
		return &vt.CacheControl
	}
	return nil
}

// Returns a pointer to the underlying variant's InputExamples property, if
// present.
func (u BetaMessageCountTokensParamsToolUnion) GetInputExamples() []map[string]any {
	if vt := u.OfTool; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfBashTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfMemoryTool20250818; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20250124; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfTextEditor20241022; vt != nil {
		return vt.InputExamples
	} else if vt := u.OfComputerUseTool20251124; vt != nil {
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
func (u BetaMessageCountTokensParamsToolUnion) GetAllowedDomains() []string {
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
func (u BetaMessageCountTokensParamsToolUnion) GetBlockedDomains() []string {
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
func (u BetaMessageCountTokensParamsToolUnion) GetUserLocation() *BetaUserLocationParam {
	if vt := u.OfWebSearchTool20250305; vt != nil {
		return &vt.UserLocation
	} else if vt := u.OfWebSearchTool20260209; vt != nil {
		return &vt.UserLocation
	}
	return nil
}

// Returns a pointer to the underlying variant's Citations property, if present.
func (u BetaMessageCountTokensParamsToolUnion) GetCitations() *BetaCitationsConfigParam {
	if vt := u.OfWebFetchTool20250910; vt != nil {
		return &vt.Citations
	} else if vt := u.OfWebFetchTool20260209; vt != nil {
		return &vt.Citations
	}
	return nil
}
