// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/openai/openai-go/v2/internal/apijson"
	"github.com/openai/openai-go/v2/internal/apiquery"
	"github.com/openai/openai-go/v2/internal/requestconfig"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/pagination"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/packages/respjson"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

// ItemService contains methods and other services that help with interacting with
// the openai API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewItemService] method instead.
type ItemService struct {
	Options []option.RequestOption
}

// NewItemService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewItemService(opts ...option.RequestOption) (r ItemService) {
	r = ItemService{}
	r.Options = opts
	return
}

// Create items in a conversation with the given ID.
func (r *ItemService) New(ctx context.Context, conversationID string, params ItemNewParams, opts ...option.RequestOption) (res *ConversationItemList, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s/items", conversationID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return
}

// Get a single item from a conversation with the given IDs.
func (r *ItemService) Get(ctx context.Context, conversationID string, itemID string, query ItemGetParams, opts ...option.RequestOption) (res *ConversationItemUnion, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	if itemID == "" {
		err = errors.New("missing required item_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s/items/%s", conversationID, itemID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// List all items for a conversation with the given ID.
func (r *ItemService) List(ctx context.Context, conversationID string, query ItemListParams, opts ...option.RequestOption) (res *pagination.ConversationCursorPage[ConversationItemUnion], err error) {
	var raw *http.Response
	opts = slices.Concat(r.Options, opts)
	opts = append([]option.RequestOption{option.WithResponseInto(&raw)}, opts...)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s/items", conversationID)
	cfg, err := requestconfig.NewRequestConfig(ctx, http.MethodGet, path, query, &res, opts...)
	if err != nil {
		return nil, err
	}
	err = cfg.Execute()
	if err != nil {
		return nil, err
	}
	res.SetPageConfig(cfg, raw)
	return res, nil
}

// List all items for a conversation with the given ID.
func (r *ItemService) ListAutoPaging(ctx context.Context, conversationID string, query ItemListParams, opts ...option.RequestOption) *pagination.ConversationCursorPageAutoPager[ConversationItemUnion] {
	return pagination.NewConversationCursorPageAutoPager(r.List(ctx, conversationID, query, opts...))
}

// Delete an item from a conversation with the given IDs.
func (r *ItemService) Delete(ctx context.Context, conversationID string, itemID string, opts ...option.RequestOption) (res *Conversation, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	if itemID == "" {
		err = errors.New("missing required item_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s/items/%s", conversationID, itemID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, nil, &res, opts...)
	return
}

// ConversationItemUnion contains all possible properties and values from
// [Message], [responses.ResponseFunctionToolCallItem],
// [responses.ResponseFunctionToolCallOutputItem],
// [responses.ResponseFileSearchToolCall], [responses.ResponseFunctionWebSearch],
// [ConversationItemImageGenerationCall], [responses.ResponseComputerToolCall],
// [responses.ResponseComputerToolCallOutputItem],
// [responses.ResponseReasoningItem], [responses.ResponseCodeInterpreterToolCall],
// [ConversationItemLocalShellCall], [ConversationItemLocalShellCallOutput],
// [ConversationItemMcpListTools], [ConversationItemMcpApprovalRequest],
// [ConversationItemMcpApprovalResponse], [ConversationItemMcpCall],
// [responses.ResponseCustomToolCall], [responses.ResponseCustomToolCallOutput].
//
// Use the [ConversationItemUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ConversationItemUnion struct {
	ID string `json:"id"`
	// This field is a union of [[]MessageContentUnion],
	// [[]responses.ResponseReasoningItemContent]
	Content ConversationItemUnionContent `json:"content"`
	// This field is from variant [Message].
	Role   MessageRole `json:"role"`
	Status string      `json:"status"`
	// Any of "message", "function_call", "function_call_output", "file_search_call",
	// "web_search_call", "image_generation_call", "computer_call",
	// "computer_call_output", "reasoning", "code_interpreter_call",
	// "local_shell_call", "local_shell_call_output", "mcp_list_tools",
	// "mcp_approval_request", "mcp_approval_response", "mcp_call", "custom_tool_call",
	// "custom_tool_call_output".
	Type      string `json:"type"`
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	// This field is a union of [string],
	// [responses.ResponseComputerToolCallOutputScreenshot], [string], [string],
	// [string]
	Output ConversationItemUnionOutput `json:"output"`
	// This field is from variant [responses.ResponseFileSearchToolCall].
	Queries []string `json:"queries"`
	// This field is from variant [responses.ResponseFileSearchToolCall].
	Results []responses.ResponseFileSearchToolCallResult `json:"results"`
	// This field is a union of [responses.ResponseFunctionWebSearchActionUnion],
	// [responses.ResponseComputerToolCallActionUnion],
	// [ConversationItemLocalShellCallAction]
	Action ConversationItemUnionAction `json:"action"`
	// This field is from variant [ConversationItemImageGenerationCall].
	Result string `json:"result"`
	// This field is from variant [responses.ResponseComputerToolCall].
	PendingSafetyChecks []responses.ResponseComputerToolCallPendingSafetyCheck `json:"pending_safety_checks"`
	// This field is from variant [responses.ResponseComputerToolCallOutputItem].
	AcknowledgedSafetyChecks []responses.ResponseComputerToolCallOutputItemAcknowledgedSafetyCheck `json:"acknowledged_safety_checks"`
	// This field is from variant [responses.ResponseReasoningItem].
	Summary []responses.ResponseReasoningItemSummary `json:"summary"`
	// This field is from variant [responses.ResponseReasoningItem].
	EncryptedContent string `json:"encrypted_content"`
	// This field is from variant [responses.ResponseCodeInterpreterToolCall].
	Code string `json:"code"`
	// This field is from variant [responses.ResponseCodeInterpreterToolCall].
	ContainerID string `json:"container_id"`
	// This field is from variant [responses.ResponseCodeInterpreterToolCall].
	Outputs     []responses.ResponseCodeInterpreterToolCallOutputUnion `json:"outputs"`
	ServerLabel string                                                 `json:"server_label"`
	// This field is from variant [ConversationItemMcpListTools].
	Tools []ConversationItemMcpListToolsTool `json:"tools"`
	Error string                             `json:"error"`
	// This field is from variant [ConversationItemMcpApprovalResponse].
	ApprovalRequestID string `json:"approval_request_id"`
	// This field is from variant [ConversationItemMcpApprovalResponse].
	Approve bool `json:"approve"`
	// This field is from variant [ConversationItemMcpApprovalResponse].
	Reason string `json:"reason"`
	// This field is from variant [responses.ResponseCustomToolCall].
	Input string `json:"input"`
	JSON  struct {
		ID                       respjson.Field
		Content                  respjson.Field
		Role                     respjson.Field
		Status                   respjson.Field
		Type                     respjson.Field
		Arguments                respjson.Field
		CallID                   respjson.Field
		Name                     respjson.Field
		Output                   respjson.Field
		Queries                  respjson.Field
		Results                  respjson.Field
		Action                   respjson.Field
		Result                   respjson.Field
		PendingSafetyChecks      respjson.Field
		AcknowledgedSafetyChecks respjson.Field
		Summary                  respjson.Field
		EncryptedContent         respjson.Field
		Code                     respjson.Field
		ContainerID              respjson.Field
		Outputs                  respjson.Field
		ServerLabel              respjson.Field
		Tools                    respjson.Field
		Error                    respjson.Field
		ApprovalRequestID        respjson.Field
		Approve                  respjson.Field
		Reason                   respjson.Field
		Input                    respjson.Field
		raw                      string
	} `json:"-"`
}

// anyConversationItem is implemented by each variant of [ConversationItemUnion] to
// add type safety for the return type of [ConversationItemUnion.AsAny]
type anyConversationItem interface {
	ImplConversationItemUnion()
}

func (Message) ImplConversationItemUnion()                              {}
func (ConversationItemImageGenerationCall) ImplConversationItemUnion()  {}
func (ConversationItemLocalShellCall) ImplConversationItemUnion()       {}
func (ConversationItemLocalShellCallOutput) ImplConversationItemUnion() {}
func (ConversationItemMcpListTools) ImplConversationItemUnion()         {}
func (ConversationItemMcpApprovalRequest) ImplConversationItemUnion()   {}
func (ConversationItemMcpApprovalResponse) ImplConversationItemUnion()  {}
func (ConversationItemMcpCall) ImplConversationItemUnion()              {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ConversationItemUnion.AsAny().(type) {
//	case conversations.Message:
//	case responses.ResponseFunctionToolCallItem:
//	case responses.ResponseFunctionToolCallOutputItem:
//	case responses.ResponseFileSearchToolCall:
//	case responses.ResponseFunctionWebSearch:
//	case conversations.ConversationItemImageGenerationCall:
//	case responses.ResponseComputerToolCall:
//	case responses.ResponseComputerToolCallOutputItem:
//	case responses.ResponseReasoningItem:
//	case responses.ResponseCodeInterpreterToolCall:
//	case conversations.ConversationItemLocalShellCall:
//	case conversations.ConversationItemLocalShellCallOutput:
//	case conversations.ConversationItemMcpListTools:
//	case conversations.ConversationItemMcpApprovalRequest:
//	case conversations.ConversationItemMcpApprovalResponse:
//	case conversations.ConversationItemMcpCall:
//	case responses.ResponseCustomToolCall:
//	case responses.ResponseCustomToolCallOutput:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ConversationItemUnion) AsAny() anyConversationItem {
	switch u.Type {
	case "message":
		return u.AsMessage()
	case "function_call":
		return u.AsFunctionCall()
	case "function_call_output":
		return u.AsFunctionCallOutput()
	case "file_search_call":
		return u.AsFileSearchCall()
	case "web_search_call":
		return u.AsWebSearchCall()
	case "image_generation_call":
		return u.AsImageGenerationCall()
	case "computer_call":
		return u.AsComputerCall()
	case "computer_call_output":
		return u.AsComputerCallOutput()
	case "reasoning":
		return u.AsReasoning()
	case "code_interpreter_call":
		return u.AsCodeInterpreterCall()
	case "local_shell_call":
		return u.AsLocalShellCall()
	case "local_shell_call_output":
		return u.AsLocalShellCallOutput()
	case "mcp_list_tools":
		return u.AsMcpListTools()
	case "mcp_approval_request":
		return u.AsMcpApprovalRequest()
	case "mcp_approval_response":
		return u.AsMcpApprovalResponse()
	case "mcp_call":
		return u.AsMcpCall()
	case "custom_tool_call":
		return u.AsCustomToolCall()
	case "custom_tool_call_output":
		return u.AsCustomToolCallOutput()
	}
	return nil
}

func (u ConversationItemUnion) AsMessage() (v Message) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsFunctionCall() (v responses.ResponseFunctionToolCallItem) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsFunctionCallOutput() (v responses.ResponseFunctionToolCallOutputItem) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsFileSearchCall() (v responses.ResponseFileSearchToolCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsWebSearchCall() (v responses.ResponseFunctionWebSearch) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsImageGenerationCall() (v ConversationItemImageGenerationCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsComputerCall() (v responses.ResponseComputerToolCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsComputerCallOutput() (v responses.ResponseComputerToolCallOutputItem) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsReasoning() (v responses.ResponseReasoningItem) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsCodeInterpreterCall() (v responses.ResponseCodeInterpreterToolCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsLocalShellCall() (v ConversationItemLocalShellCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsLocalShellCallOutput() (v ConversationItemLocalShellCallOutput) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsMcpListTools() (v ConversationItemMcpListTools) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsMcpApprovalRequest() (v ConversationItemMcpApprovalRequest) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsMcpApprovalResponse() (v ConversationItemMcpApprovalResponse) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsMcpCall() (v ConversationItemMcpCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsCustomToolCall() (v responses.ResponseCustomToolCall) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ConversationItemUnion) AsCustomToolCallOutput() (v responses.ResponseCustomToolCallOutput) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ConversationItemUnion) RawJSON() string { return u.JSON.raw }

func (r *ConversationItemUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConversationItemUnionContent is an implicit subunion of [ConversationItemUnion].
// ConversationItemUnionContent provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [ConversationItemUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMessageContentArray OfResponseReasoningItemContentArray]
type ConversationItemUnionContent struct {
	// This field will be present if the value is a [[]MessageContentUnion] instead of
	// an object.
	OfMessageContentArray []MessageContentUnion `json:",inline"`
	// This field will be present if the value is a
	// [[]responses.ResponseReasoningItemContent] instead of an object.
	OfResponseReasoningItemContentArray []responses.ResponseReasoningItemContent `json:",inline"`
	JSON                                struct {
		OfMessageContentArray               respjson.Field
		OfResponseReasoningItemContentArray respjson.Field
		raw                                 string
	} `json:"-"`
}

func (r *ConversationItemUnionContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConversationItemUnionOutput is an implicit subunion of [ConversationItemUnion].
// ConversationItemUnionOutput provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [ConversationItemUnion].
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfString]
type ConversationItemUnionOutput struct {
	// This field will be present if the value is a [string] instead of an object.
	OfString string `json:",inline"`
	// This field is from variant [responses.ResponseComputerToolCallOutputScreenshot].
	Type constant.ComputerScreenshot `json:"type"`
	// This field is from variant [responses.ResponseComputerToolCallOutputScreenshot].
	FileID string `json:"file_id"`
	// This field is from variant [responses.ResponseComputerToolCallOutputScreenshot].
	ImageURL string `json:"image_url"`
	JSON     struct {
		OfString respjson.Field
		Type     respjson.Field
		FileID   respjson.Field
		ImageURL respjson.Field
		raw      string
	} `json:"-"`
}

func (r *ConversationItemUnionOutput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ConversationItemUnionAction is an implicit subunion of [ConversationItemUnion].
// ConversationItemUnionAction provides convenient access to the sub-properties of
// the union.
//
// For type safety it is recommended to directly use a variant of the
// [ConversationItemUnion].
type ConversationItemUnionAction struct {
	// This field is from variant [responses.ResponseFunctionWebSearchActionUnion].
	Query string `json:"query"`
	Type  string `json:"type"`
	// This field is from variant [responses.ResponseFunctionWebSearchActionUnion].
	Sources []responses.ResponseFunctionWebSearchActionSearchSource `json:"sources"`
	URL     string                                                  `json:"url"`
	// This field is from variant [responses.ResponseFunctionWebSearchActionUnion].
	Pattern string `json:"pattern"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	Button string `json:"button"`
	X      int64  `json:"x"`
	Y      int64  `json:"y"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	Path []responses.ResponseComputerToolCallActionDragPath `json:"path"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	Keys []string `json:"keys"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	ScrollX int64 `json:"scroll_x"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	ScrollY int64 `json:"scroll_y"`
	// This field is from variant [responses.ResponseComputerToolCallActionUnion].
	Text string `json:"text"`
	// This field is from variant [ConversationItemLocalShellCallAction].
	Command []string `json:"command"`
	// This field is from variant [ConversationItemLocalShellCallAction].
	Env map[string]string `json:"env"`
	// This field is from variant [ConversationItemLocalShellCallAction].
	TimeoutMs int64 `json:"timeout_ms"`
	// This field is from variant [ConversationItemLocalShellCallAction].
	User string `json:"user"`
	// This field is from variant [ConversationItemLocalShellCallAction].
	WorkingDirectory string `json:"working_directory"`
	JSON             struct {
		Query            respjson.Field
		Type             respjson.Field
		Sources          respjson.Field
		URL              respjson.Field
		Pattern          respjson.Field
		Button           respjson.Field
		X                respjson.Field
		Y                respjson.Field
		Path             respjson.Field
		Keys             respjson.Field
		ScrollX          respjson.Field
		ScrollY          respjson.Field
		Text             respjson.Field
		Command          respjson.Field
		Env              respjson.Field
		TimeoutMs        respjson.Field
		User             respjson.Field
		WorkingDirectory respjson.Field
		raw              string
	} `json:"-"`
}

func (r *ConversationItemUnionAction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// An image generation request made by the model.
type ConversationItemImageGenerationCall struct {
	// The unique ID of the image generation call.
	ID string `json:"id,required"`
	// The generated image encoded in base64.
	Result string `json:"result,required"`
	// The status of the image generation call.
	//
	// Any of "in_progress", "completed", "generating", "failed".
	Status string `json:"status,required"`
	// The type of the image generation call. Always `image_generation_call`.
	Type constant.ImageGenerationCall `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Result      respjson.Field
		Status      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemImageGenerationCall) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemImageGenerationCall) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A tool call to run a command on the local shell.
type ConversationItemLocalShellCall struct {
	// The unique ID of the local shell call.
	ID string `json:"id,required"`
	// Execute a shell command on the server.
	Action ConversationItemLocalShellCallAction `json:"action,required"`
	// The unique ID of the local shell tool call generated by the model.
	CallID string `json:"call_id,required"`
	// The status of the local shell call.
	//
	// Any of "in_progress", "completed", "incomplete".
	Status string `json:"status,required"`
	// The type of the local shell call. Always `local_shell_call`.
	Type constant.LocalShellCall `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Action      respjson.Field
		CallID      respjson.Field
		Status      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemLocalShellCall) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemLocalShellCall) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Execute a shell command on the server.
type ConversationItemLocalShellCallAction struct {
	// The command to run.
	Command []string `json:"command,required"`
	// Environment variables to set for the command.
	Env map[string]string `json:"env,required"`
	// The type of the local shell action. Always `exec`.
	Type constant.Exec `json:"type,required"`
	// Optional timeout in milliseconds for the command.
	TimeoutMs int64 `json:"timeout_ms,nullable"`
	// Optional user to run the command as.
	User string `json:"user,nullable"`
	// Optional working directory to run the command in.
	WorkingDirectory string `json:"working_directory,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Command          respjson.Field
		Env              respjson.Field
		Type             respjson.Field
		TimeoutMs        respjson.Field
		User             respjson.Field
		WorkingDirectory respjson.Field
		ExtraFields      map[string]respjson.Field
		raw              string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemLocalShellCallAction) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemLocalShellCallAction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The output of a local shell tool call.
type ConversationItemLocalShellCallOutput struct {
	// The unique ID of the local shell tool call generated by the model.
	ID string `json:"id,required"`
	// A JSON string of the output of the local shell tool call.
	Output string `json:"output,required"`
	// The type of the local shell tool call output. Always `local_shell_call_output`.
	Type constant.LocalShellCallOutput `json:"type,required"`
	// The status of the item. One of `in_progress`, `completed`, or `incomplete`.
	//
	// Any of "in_progress", "completed", "incomplete".
	Status string `json:"status,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Output      respjson.Field
		Type        respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemLocalShellCallOutput) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemLocalShellCallOutput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A list of tools available on an MCP server.
type ConversationItemMcpListTools struct {
	// The unique ID of the list.
	ID string `json:"id,required"`
	// The label of the MCP server.
	ServerLabel string `json:"server_label,required"`
	// The tools available on the server.
	Tools []ConversationItemMcpListToolsTool `json:"tools,required"`
	// The type of the item. Always `mcp_list_tools`.
	Type constant.McpListTools `json:"type,required"`
	// Error message if the server could not list tools.
	Error string `json:"error,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		ServerLabel respjson.Field
		Tools       respjson.Field
		Type        respjson.Field
		Error       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemMcpListTools) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemMcpListTools) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A tool available on an MCP server.
type ConversationItemMcpListToolsTool struct {
	// The JSON schema describing the tool's input.
	InputSchema any `json:"input_schema,required"`
	// The name of the tool.
	Name string `json:"name,required"`
	// Additional annotations about the tool.
	Annotations any `json:"annotations,nullable"`
	// The description of the tool.
	Description string `json:"description,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		InputSchema respjson.Field
		Name        respjson.Field
		Annotations respjson.Field
		Description respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemMcpListToolsTool) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemMcpListToolsTool) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A request for human approval of a tool invocation.
type ConversationItemMcpApprovalRequest struct {
	// The unique ID of the approval request.
	ID string `json:"id,required"`
	// A JSON string of arguments for the tool.
	Arguments string `json:"arguments,required"`
	// The name of the tool to run.
	Name string `json:"name,required"`
	// The label of the MCP server making the request.
	ServerLabel string `json:"server_label,required"`
	// The type of the item. Always `mcp_approval_request`.
	Type constant.McpApprovalRequest `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Arguments   respjson.Field
		Name        respjson.Field
		ServerLabel respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemMcpApprovalRequest) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemMcpApprovalRequest) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A response to an MCP approval request.
type ConversationItemMcpApprovalResponse struct {
	// The unique ID of the approval response
	ID string `json:"id,required"`
	// The ID of the approval request being answered.
	ApprovalRequestID string `json:"approval_request_id,required"`
	// Whether the request was approved.
	Approve bool `json:"approve,required"`
	// The type of the item. Always `mcp_approval_response`.
	Type constant.McpApprovalResponse `json:"type,required"`
	// Optional reason for the decision.
	Reason string `json:"reason,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID                respjson.Field
		ApprovalRequestID respjson.Field
		Approve           respjson.Field
		Type              respjson.Field
		Reason            respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemMcpApprovalResponse) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemMcpApprovalResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// An invocation of a tool on an MCP server.
type ConversationItemMcpCall struct {
	// The unique ID of the tool call.
	ID string `json:"id,required"`
	// A JSON string of the arguments passed to the tool.
	Arguments string `json:"arguments,required"`
	// The name of the tool that was run.
	Name string `json:"name,required"`
	// The label of the MCP server running the tool.
	ServerLabel string `json:"server_label,required"`
	// The type of the item. Always `mcp_call`.
	Type constant.McpCall `json:"type,required"`
	// The error from the tool call, if any.
	Error string `json:"error,nullable"`
	// The output from the tool call.
	Output string `json:"output,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Arguments   respjson.Field
		Name        respjson.Field
		ServerLabel respjson.Field
		Type        respjson.Field
		Error       respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemMcpCall) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemMcpCall) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A list of Conversation items.
type ConversationItemList struct {
	// A list of conversation items.
	Data []ConversationItemUnion `json:"data,required"`
	// The ID of the first item in the list.
	FirstID string `json:"first_id,required"`
	// Whether there are more items available.
	HasMore bool `json:"has_more,required"`
	// The ID of the last item in the list.
	LastID string `json:"last_id,required"`
	// The type of object returned, must be `list`.
	Object constant.List `json:"object,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data        respjson.Field
		FirstID     respjson.Field
		HasMore     respjson.Field
		LastID      respjson.Field
		Object      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationItemList) RawJSON() string { return r.JSON.raw }
func (r *ConversationItemList) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ItemNewParams struct {
	// The items to add to the conversation. You may add up to 20 items at a time.
	Items []responses.ResponseInputItemUnionParam `json:"items,omitzero,required"`
	// Additional fields to include in the response. See the `include` parameter for
	// [listing Conversation items above](https://platform.openai.com/docs/api-reference/conversations/list-items#conversations_list_items-include)
	// for more information.
	Include []responses.ResponseIncludable `query:"include,omitzero" json:"-"`
	paramObj
}

func (r ItemNewParams) MarshalJSON() (data []byte, err error) {
	type shadow ItemNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ItemNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// URLQuery serializes [ItemNewParams]'s query parameters as `url.Values`.
func (r ItemNewParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatBrackets,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ItemGetParams struct {
	// Additional fields to include in the response. See the `include` parameter for
	// [listing Conversation items above](https://platform.openai.com/docs/api-reference/conversations/list-items#conversations_list_items-include)
	// for more information.
	Include []responses.ResponseIncludable `query:"include,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ItemGetParams]'s query parameters as `url.Values`.
func (r ItemGetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatBrackets,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

type ItemListParams struct {
	// An item ID to list items after, used in pagination.
	After param.Opt[string] `query:"after,omitzero" json:"-"`
	// A limit on the number of objects to be returned. Limit can range between 1 and
	// 100, and the default is 20.
	Limit param.Opt[int64] `query:"limit,omitzero" json:"-"`
	// Specify additional output data to include in the model response. Currently
	// supported values are:
	//
	//   - `web_search_call.action.sources`: Include the sources of the web search tool
	//     call.
	//   - `code_interpreter_call.outputs`: Includes the outputs of python code execution
	//     in code interpreter tool call items.
	//   - `computer_call_output.output.image_url`: Include image urls from the computer
	//     call output.
	//   - `file_search_call.results`: Include the search results of the file search tool
	//     call.
	//   - `message.input_image.image_url`: Include image urls from the input message.
	//   - `message.output_text.logprobs`: Include logprobs with assistant messages.
	//   - `reasoning.encrypted_content`: Includes an encrypted version of reasoning
	//     tokens in reasoning item outputs. This enables reasoning items to be used in
	//     multi-turn conversations when using the Responses API statelessly (like when
	//     the `store` parameter is set to `false`, or when an organization is enrolled
	//     in the zero data retention program).
	Include []responses.ResponseIncludable `query:"include,omitzero" json:"-"`
	// The order to return the input items in. Default is `desc`.
	//
	// - `asc`: Return the input items in ascending order.
	// - `desc`: Return the input items in descending order.
	//
	// Any of "asc", "desc".
	Order ItemListParamsOrder `query:"order,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [ItemListParams]'s query parameters as `url.Values`.
func (r ItemListParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatBrackets,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// The order to return the input items in. Default is `desc`.
//
// - `asc`: Return the input items in ascending order.
// - `desc`: Return the input items in descending order.
type ItemListParamsOrder string

const (
	ItemListParamsOrderAsc  ItemListParamsOrder = "asc"
	ItemListParamsOrderDesc ItemListParamsOrder = "desc"
)
