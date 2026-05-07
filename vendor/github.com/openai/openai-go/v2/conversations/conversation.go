// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/openai/openai-go/v2/internal/apijson"
	"github.com/openai/openai-go/v2/internal/requestconfig"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/packages/respjson"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
	"github.com/openai/openai-go/v2/shared/constant"
)

// ConversationService contains methods and other services that help with
// interacting with the openai API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewConversationService] method instead.
type ConversationService struct {
	Options []option.RequestOption
	Items   ItemService
}

// NewConversationService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewConversationService(opts ...option.RequestOption) (r ConversationService) {
	r = ConversationService{}
	r.Options = opts
	r.Items = NewItemService(opts...)
	return
}

// Create a conversation.
func (r *ConversationService) New(ctx context.Context, body ConversationNewParams, opts ...option.RequestOption) (res *Conversation, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "conversations"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Get a conversation
func (r *ConversationService) Get(ctx context.Context, conversationID string, opts ...option.RequestOption) (res *Conversation, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s", conversationID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

// Update a conversation
func (r *ConversationService) Update(ctx context.Context, conversationID string, body ConversationUpdateParams, opts ...option.RequestOption) (res *Conversation, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s", conversationID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Delete a conversation. Items in the conversation will not be deleted.
func (r *ConversationService) Delete(ctx context.Context, conversationID string, opts ...option.RequestOption) (res *ConversationDeletedResource, err error) {
	opts = slices.Concat(r.Options, opts)
	if conversationID == "" {
		err = errors.New("missing required conversation_id parameter")
		return
	}
	path := fmt.Sprintf("conversations/%s", conversationID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, nil, &res, opts...)
	return
}

// A screenshot of a computer.
type ComputerScreenshotContent struct {
	// The identifier of an uploaded file that contains the screenshot.
	FileID string `json:"file_id,required"`
	// The URL of the screenshot image.
	ImageURL string `json:"image_url,required"`
	// Specifies the event type. For a computer screenshot, this property is always set
	// to `computer_screenshot`.
	Type constant.ComputerScreenshot `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		FileID      respjson.Field
		ImageURL    respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ComputerScreenshotContent) RawJSON() string { return r.JSON.raw }
func (r *ComputerScreenshotContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type Conversation struct {
	// The unique ID of the conversation.
	ID string `json:"id,required"`
	// The time at which the conversation was created, measured in seconds since the
	// Unix epoch.
	CreatedAt int64 `json:"created_at,required"`
	// Set of 16 key-value pairs that can be attached to an object. This can be useful
	// for storing additional information about the object in a structured format, and
	// querying for objects via API or the dashboard. Keys are strings with a maximum
	// length of 64 characters. Values are strings with a maximum length of 512
	// characters.
	Metadata any `json:"metadata,required"`
	// The object type, which is always `conversation`.
	Object constant.Conversation `json:"object,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		CreatedAt   respjson.Field
		Metadata    respjson.Field
		Object      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Conversation) RawJSON() string { return r.JSON.raw }
func (r *Conversation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConversationDeletedResource struct {
	ID      string                       `json:"id,required"`
	Deleted bool                         `json:"deleted,required"`
	Object  constant.ConversationDeleted `json:"object,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Deleted     respjson.Field
		Object      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ConversationDeletedResource) RawJSON() string { return r.JSON.raw }
func (r *ConversationDeletedResource) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A message to or from the model.
type Message struct {
	// The unique ID of the message.
	ID string `json:"id,required"`
	// The content of the message
	Content []MessageContentUnion `json:"content,required"`
	// The role of the message. One of `unknown`, `user`, `assistant`, `system`,
	// `critic`, `discriminator`, `developer`, or `tool`.
	//
	// Any of "unknown", "user", "assistant", "system", "critic", "discriminator",
	// "developer", "tool".
	Role MessageRole `json:"role,required"`
	// The status of item. One of `in_progress`, `completed`, or `incomplete`.
	// Populated when items are returned via API.
	//
	// Any of "in_progress", "completed", "incomplete".
	Status MessageStatus `json:"status,required"`
	// The type of the message. Always set to `message`.
	Type constant.Message `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Content     respjson.Field
		Role        respjson.Field
		Status      respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Message) RawJSON() string { return r.JSON.raw }
func (r *Message) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// MessageContentUnion contains all possible properties and values from
// [responses.ResponseInputText], [responses.ResponseOutputText], [TextContent],
// [SummaryTextContent], [MessageContentReasoningText],
// [responses.ResponseOutputRefusal], [responses.ResponseInputImage],
// [ComputerScreenshotContent], [responses.ResponseInputFile].
//
// Use the [MessageContentUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type MessageContentUnion struct {
	Text string `json:"text"`
	// Any of "input_text", "output_text", "text", "summary_text", "reasoning_text",
	// "refusal", "input_image", "computer_screenshot", "input_file".
	Type string `json:"type"`
	// This field is from variant [responses.ResponseOutputText].
	Annotations []responses.ResponseOutputTextAnnotationUnion `json:"annotations"`
	// This field is from variant [responses.ResponseOutputText].
	Logprobs []responses.ResponseOutputTextLogprob `json:"logprobs"`
	// This field is from variant [responses.ResponseOutputRefusal].
	Refusal string `json:"refusal"`
	// This field is from variant [responses.ResponseInputImage].
	Detail   responses.ResponseInputImageDetail `json:"detail"`
	FileID   string                             `json:"file_id"`
	ImageURL string                             `json:"image_url"`
	// This field is from variant [responses.ResponseInputFile].
	FileData string `json:"file_data"`
	// This field is from variant [responses.ResponseInputFile].
	FileURL string `json:"file_url"`
	// This field is from variant [responses.ResponseInputFile].
	Filename string `json:"filename"`
	JSON     struct {
		Text        respjson.Field
		Type        respjson.Field
		Annotations respjson.Field
		Logprobs    respjson.Field
		Refusal     respjson.Field
		Detail      respjson.Field
		FileID      respjson.Field
		ImageURL    respjson.Field
		FileData    respjson.Field
		FileURL     respjson.Field
		Filename    respjson.Field
		raw         string
	} `json:"-"`
}

// anyMessageContent is implemented by each variant of [MessageContentUnion] to add
// type safety for the return type of [MessageContentUnion.AsAny]
type anyMessageContent interface {
	ImplMessageContentUnion()
}

func (TextContent) ImplMessageContentUnion()                 {}
func (SummaryTextContent) ImplMessageContentUnion()          {}
func (MessageContentReasoningText) ImplMessageContentUnion() {}
func (ComputerScreenshotContent) ImplMessageContentUnion()   {}

// Use the following switch statement to find the correct variant
//
//	switch variant := MessageContentUnion.AsAny().(type) {
//	case responses.ResponseInputText:
//	case responses.ResponseOutputText:
//	case conversations.TextContent:
//	case conversations.SummaryTextContent:
//	case conversations.MessageContentReasoningText:
//	case responses.ResponseOutputRefusal:
//	case responses.ResponseInputImage:
//	case conversations.ComputerScreenshotContent:
//	case responses.ResponseInputFile:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u MessageContentUnion) AsAny() anyMessageContent {
	switch u.Type {
	case "input_text":
		return u.AsInputText()
	case "output_text":
		return u.AsOutputText()
	case "text":
		return u.AsText()
	case "summary_text":
		return u.AsSummaryText()
	case "reasoning_text":
		return u.AsReasoningText()
	case "refusal":
		return u.AsRefusal()
	case "input_image":
		return u.AsInputImage()
	case "computer_screenshot":
		return u.AsComputerScreenshot()
	case "input_file":
		return u.AsInputFile()
	}
	return nil
}

func (u MessageContentUnion) AsInputText() (v responses.ResponseInputText) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsOutputText() (v responses.ResponseOutputText) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsText() (v TextContent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsSummaryText() (v SummaryTextContent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsReasoningText() (v MessageContentReasoningText) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsRefusal() (v responses.ResponseOutputRefusal) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsInputImage() (v responses.ResponseInputImage) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsComputerScreenshot() (v ComputerScreenshotContent) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u MessageContentUnion) AsInputFile() (v responses.ResponseInputFile) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u MessageContentUnion) RawJSON() string { return u.JSON.raw }

func (r *MessageContentUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Reasoning text from the model.
type MessageContentReasoningText struct {
	// The reasoning text from the model.
	Text string `json:"text,required"`
	// The type of the reasoning text. Always `reasoning_text`.
	Type constant.ReasoningText `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r MessageContentReasoningText) RawJSON() string { return r.JSON.raw }
func (r *MessageContentReasoningText) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The role of the message. One of `unknown`, `user`, `assistant`, `system`,
// `critic`, `discriminator`, `developer`, or `tool`.
type MessageRole string

const (
	MessageRoleUnknown       MessageRole = "unknown"
	MessageRoleUser          MessageRole = "user"
	MessageRoleAssistant     MessageRole = "assistant"
	MessageRoleSystem        MessageRole = "system"
	MessageRoleCritic        MessageRole = "critic"
	MessageRoleDiscriminator MessageRole = "discriminator"
	MessageRoleDeveloper     MessageRole = "developer"
	MessageRoleTool          MessageRole = "tool"
)

// The status of item. One of `in_progress`, `completed`, or `incomplete`.
// Populated when items are returned via API.
type MessageStatus string

const (
	MessageStatusInProgress MessageStatus = "in_progress"
	MessageStatusCompleted  MessageStatus = "completed"
	MessageStatusIncomplete MessageStatus = "incomplete"
)

// A summary text from the model.
type SummaryTextContent struct {
	// A summary of the reasoning output from the model so far.
	Text string `json:"text,required"`
	// The type of the object. Always `summary_text`.
	Type constant.SummaryText `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r SummaryTextContent) RawJSON() string { return r.JSON.raw }
func (r *SummaryTextContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A text content.
type TextContent struct {
	Text string        `json:"text,required"`
	Type constant.Text `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Text        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r TextContent) RawJSON() string { return r.JSON.raw }
func (r *TextContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConversationNewParams struct {
	// Initial items to include in the conversation context. You may add up to 20 items
	// at a time.
	Items []responses.ResponseInputItemUnionParam `json:"items,omitzero"`
	// Set of 16 key-value pairs that can be attached to an object. This can be useful
	// for storing additional information about the object in a structured format, and
	// querying for objects via API or the dashboard.
	//
	// Keys are strings with a maximum length of 64 characters. Values are strings with
	// a maximum length of 512 characters.
	Metadata shared.Metadata `json:"metadata,omitzero"`
	paramObj
}

func (r ConversationNewParams) MarshalJSON() (data []byte, err error) {
	type shadow ConversationNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConversationNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ConversationUpdateParams struct {
	// Set of 16 key-value pairs that can be attached to an object. This can be useful
	// for storing additional information about the object in a structured format, and
	// querying for objects via API or the dashboard.
	//
	// Keys are strings with a maximum length of 64 characters. Values are strings with
	// a maximum length of 512 characters.
	Metadata shared.Metadata `json:"metadata,omitzero,required"`
	paramObj
}

func (r ConversationUpdateParams) MarshalJSON() (data []byte, err error) {
	type shadow ConversationUpdateParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ConversationUpdateParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
