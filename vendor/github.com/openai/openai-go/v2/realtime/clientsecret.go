// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/openai/openai-go/v2/internal/apijson"
	"github.com/openai/openai-go/v2/internal/requestconfig"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/packages/respjson"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

// ClientSecretService contains methods and other services that help with
// interacting with the openai API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewClientSecretService] method instead.
type ClientSecretService struct {
	Options []option.RequestOption
}

// NewClientSecretService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewClientSecretService(opts ...option.RequestOption) (r ClientSecretService) {
	r = ClientSecretService{}
	r.Options = opts
	return
}

// Create a Realtime client secret with an associated session configuration.
func (r *ClientSecretService) New(ctx context.Context, body ClientSecretNewParams, opts ...option.RequestOption) (res *ClientSecretNewResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	path := "realtime/client_secrets"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Ephemeral key returned by the API.
type RealtimeSessionClientSecret struct {
	// Timestamp for when the token expires. Currently, all tokens expire after one
	// minute.
	ExpiresAt int64 `json:"expires_at,required"`
	// Ephemeral key usable in client environments to authenticate connections to the
	// Realtime API. Use this in client-side environments rather than a standard API
	// token, which should only be used server-side.
	Value string `json:"value,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ExpiresAt   respjson.Field
		Value       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionClientSecret) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionClientSecret) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A new Realtime session configuration, with an ephemeral key. Default TTL for
// keys is one minute.
type RealtimeSessionCreateResponse struct {
	// Ephemeral key returned by the API.
	ClientSecret RealtimeSessionClientSecret `json:"client_secret,required"`
	// The type of session to create. Always `realtime` for the Realtime API.
	Type constant.Realtime `json:"type,required"`
	// Configuration for input and output audio.
	Audio RealtimeSessionCreateResponseAudio `json:"audio"`
	// Additional fields to include in server outputs.
	//
	// `item.input_audio_transcription.logprobs`: Include logprobs for input audio
	// transcription.
	//
	// Any of "item.input_audio_transcription.logprobs".
	Include []string `json:"include"`
	// The default system instructions (i.e. system message) prepended to model calls.
	// This field allows the client to guide the model on desired responses. The model
	// can be instructed on response content and format, (e.g. "be extremely succinct",
	// "act friendly", "here are examples of good responses") and on audio behavior
	// (e.g. "talk quickly", "inject emotion into your voice", "laugh frequently"). The
	// instructions are not guaranteed to be followed by the model, but they provide
	// guidance to the model on the desired behavior.
	//
	// Note that the server sets default instructions which will be used if this field
	// is not set and are visible in the `session.created` event at the start of the
	// session.
	Instructions string `json:"instructions"`
	// Maximum number of output tokens for a single assistant response, inclusive of
	// tool calls. Provide an integer between 1 and 4096 to limit output tokens, or
	// `inf` for the maximum available tokens for a given model. Defaults to `inf`.
	MaxOutputTokens RealtimeSessionCreateResponseMaxOutputTokensUnion `json:"max_output_tokens"`
	// The Realtime model used for this session.
	Model RealtimeSessionCreateResponseModel `json:"model"`
	// The set of modalities the model can respond with. It defaults to `["audio"]`,
	// indicating that the model will respond with audio plus a transcript. `["text"]`
	// can be used to make the model respond with text only. It is not possible to
	// request both `text` and `audio` at the same time.
	//
	// Any of "text", "audio".
	OutputModalities []string `json:"output_modalities"`
	// Reference to a prompt template and its variables.
	// [Learn more](https://platform.openai.com/docs/guides/text?api-mode=responses#reusable-prompts).
	Prompt responses.ResponsePrompt `json:"prompt,nullable"`
	// How the model chooses tools. Provide one of the string modes or force a specific
	// function/MCP tool.
	ToolChoice RealtimeSessionCreateResponseToolChoiceUnion `json:"tool_choice"`
	// Tools available to the model.
	Tools []RealtimeSessionCreateResponseToolUnion `json:"tools"`
	// Realtime API can write session traces to the
	// [Traces Dashboard](/logs?api=traces). Set to null to disable tracing. Once
	// tracing is enabled for a session, the configuration cannot be modified.
	//
	// `auto` will create a trace for the session with default values for the workflow
	// name, group id, and metadata.
	Tracing RealtimeSessionCreateResponseTracingUnion `json:"tracing,nullable"`
	// Controls how the realtime conversation is truncated prior to model inference.
	// The default is `auto`.
	Truncation RealtimeTruncationUnion `json:"truncation"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ClientSecret     respjson.Field
		Type             respjson.Field
		Audio            respjson.Field
		Include          respjson.Field
		Instructions     respjson.Field
		MaxOutputTokens  respjson.Field
		Model            respjson.Field
		OutputModalities respjson.Field
		Prompt           respjson.Field
		ToolChoice       respjson.Field
		Tools            respjson.Field
		Tracing          respjson.Field
		Truncation       respjson.Field
		ExtraFields      map[string]respjson.Field
		raw              string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponse) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input and output audio.
type RealtimeSessionCreateResponseAudio struct {
	Input  RealtimeSessionCreateResponseAudioInput  `json:"input"`
	Output RealtimeSessionCreateResponseAudioOutput `json:"output"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		Output      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudio) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseAudio) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeSessionCreateResponseAudioInput struct {
	// The format of the input audio.
	Format RealtimeAudioFormatsUnion `json:"format"`
	// Configuration for input audio noise reduction. This can be set to `null` to turn
	// off. Noise reduction filters audio added to the input audio buffer before it is
	// sent to VAD and the model. Filtering the audio can improve VAD and turn
	// detection accuracy (reducing false positives) and model performance by improving
	// perception of the input audio.
	NoiseReduction RealtimeSessionCreateResponseAudioInputNoiseReduction `json:"noise_reduction"`
	// Configuration for input audio transcription, defaults to off and can be set to
	// `null` to turn off once on. Input audio transcription is not native to the
	// model, since the model consumes audio directly. Transcription runs
	// asynchronously through
	// [the /audio/transcriptions endpoint](https://platform.openai.com/docs/api-reference/audio/createTranscription)
	// and should be treated as guidance of input audio content rather than precisely
	// what the model heard. The client can optionally set the language and prompt for
	// transcription, these offer additional guidance to the transcription service.
	Transcription AudioTranscription `json:"transcription"`
	// Configuration for turn detection, ether Server VAD or Semantic VAD. This can be
	// set to `null` to turn off, in which case the client must manually trigger model
	// response.
	//
	// Server VAD means that the model will detect the start and end of speech based on
	// audio volume and respond at the end of user speech.
	//
	// Semantic VAD is more advanced and uses a turn detection model (in conjunction
	// with VAD) to semantically estimate whether the user has finished speaking, then
	// dynamically sets a timeout based on this probability. For example, if user audio
	// trails off with "uhhm", the model will score a low probability of turn end and
	// wait longer for the user to continue speaking. This can be useful for more
	// natural conversations, but may have a higher latency.
	TurnDetection RealtimeSessionCreateResponseAudioInputTurnDetectionUnion `json:"turn_detection,nullable"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Format         respjson.Field
		NoiseReduction respjson.Field
		Transcription  respjson.Field
		TurnDetection  respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudioInput) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseAudioInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input audio noise reduction. This can be set to `null` to turn
// off. Noise reduction filters audio added to the input audio buffer before it is
// sent to VAD and the model. Filtering the audio can improve VAD and turn
// detection accuracy (reducing false positives) and model performance by improving
// perception of the input audio.
type RealtimeSessionCreateResponseAudioInputNoiseReduction struct {
	// Type of noise reduction. `near_field` is for close-talking microphones such as
	// headphones, `far_field` is for far-field microphones such as laptop or
	// conference room microphones.
	//
	// Any of "near_field", "far_field".
	Type NoiseReductionType `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudioInputNoiseReduction) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseAudioInputNoiseReduction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeSessionCreateResponseAudioInputTurnDetectionUnion contains all possible
// properties and values from
// [RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad],
// [RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad].
//
// Use the [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion.AsAny] method
// to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type RealtimeSessionCreateResponseAudioInputTurnDetectionUnion struct {
	// Any of "server_vad", "semantic_vad".
	Type           string `json:"type"`
	CreateResponse bool   `json:"create_response"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad].
	IdleTimeoutMs     int64 `json:"idle_timeout_ms"`
	InterruptResponse bool  `json:"interrupt_response"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad].
	PrefixPaddingMs int64 `json:"prefix_padding_ms"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad].
	SilenceDurationMs int64 `json:"silence_duration_ms"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad].
	Threshold float64 `json:"threshold"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad].
	Eagerness string `json:"eagerness"`
	JSON      struct {
		Type              respjson.Field
		CreateResponse    respjson.Field
		IdleTimeoutMs     respjson.Field
		InterruptResponse respjson.Field
		PrefixPaddingMs   respjson.Field
		SilenceDurationMs respjson.Field
		Threshold         respjson.Field
		Eagerness         respjson.Field
		raw               string
	} `json:"-"`
}

// anyRealtimeSessionCreateResponseAudioInputTurnDetection is implemented by each
// variant of [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion] to add
// type safety for the return type of
// [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion.AsAny]
type anyRealtimeSessionCreateResponseAudioInputTurnDetection interface {
	implRealtimeSessionCreateResponseAudioInputTurnDetectionUnion()
}

func (RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad) implRealtimeSessionCreateResponseAudioInputTurnDetectionUnion() {
}
func (RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad) implRealtimeSessionCreateResponseAudioInputTurnDetectionUnion() {
}

// Use the following switch statement to find the correct variant
//
//	switch variant := RealtimeSessionCreateResponseAudioInputTurnDetectionUnion.AsAny().(type) {
//	case realtime.RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad:
//	case realtime.RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u RealtimeSessionCreateResponseAudioInputTurnDetectionUnion) AsAny() anyRealtimeSessionCreateResponseAudioInputTurnDetection {
	switch u.Type {
	case "server_vad":
		return u.AsServerVad()
	case "semantic_vad":
		return u.AsSemanticVad()
	}
	return nil
}

func (u RealtimeSessionCreateResponseAudioInputTurnDetectionUnion) AsServerVad() (v RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseAudioInputTurnDetectionUnion) AsSemanticVad() (v RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseAudioInputTurnDetectionUnion) RawJSON() string {
	return u.JSON.raw
}

func (r *RealtimeSessionCreateResponseAudioInputTurnDetectionUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Server-side voice activity detection (VAD) which flips on when user speech is
// detected and off after a period of silence.
type RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad struct {
	// Type of turn detection, `server_vad` to turn on simple Server VAD.
	Type constant.ServerVad `json:"type,required"`
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse bool `json:"create_response"`
	// Optional timeout after which a model response will be triggered automatically.
	// This is useful for situations in which a long pause from the user is unexpected,
	// such as a phone call. The model will effectively prompt the user to continue the
	// conversation based on the current context.
	//
	// The timeout value will be applied after the last model response's audio has
	// finished playing, i.e. it's set to the `response.done` time plus audio playback
	// duration.
	//
	// An `input_audio_buffer.timeout_triggered` event (plus events associated with the
	// Response) will be emitted when the timeout is reached. Idle timeout is currently
	// only supported for `server_vad` mode.
	IdleTimeoutMs int64 `json:"idle_timeout_ms,nullable"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse bool `json:"interrupt_response"`
	// Used only for `server_vad` mode. Amount of audio to include before the VAD
	// detected speech (in milliseconds). Defaults to 300ms.
	PrefixPaddingMs int64 `json:"prefix_padding_ms"`
	// Used only for `server_vad` mode. Duration of silence to detect speech stop (in
	// milliseconds). Defaults to 500ms. With shorter values the model will respond
	// more quickly, but may jump in on short pauses from the user.
	SilenceDurationMs int64 `json:"silence_duration_ms"`
	// Used only for `server_vad` mode. Activation threshold for VAD (0.0 to 1.0), this
	// defaults to 0.5. A higher threshold will require louder audio to activate the
	// model, and thus might perform better in noisy environments.
	Threshold float64 `json:"threshold"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type              respjson.Field
		CreateResponse    respjson.Field
		IdleTimeoutMs     respjson.Field
		InterruptResponse respjson.Field
		PrefixPaddingMs   respjson.Field
		SilenceDurationMs respjson.Field
		Threshold         respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseAudioInputTurnDetectionServerVad) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Server-side semantic turn detection which uses a model to determine when the
// user has finished speaking.
type RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad struct {
	// Type of turn detection, `semantic_vad` to turn on Semantic VAD.
	Type constant.SemanticVad `json:"type,required"`
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse bool `json:"create_response"`
	// Used only for `semantic_vad` mode. The eagerness of the model to respond. `low`
	// will wait longer for the user to continue speaking, `high` will respond more
	// quickly. `auto` is the default and is equivalent to `medium`. `low`, `medium`,
	// and `high` have max timeouts of 8s, 4s, and 2s respectively.
	//
	// Any of "low", "medium", "high", "auto".
	Eagerness string `json:"eagerness"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse bool `json:"interrupt_response"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type              respjson.Field
		CreateResponse    respjson.Field
		Eagerness         respjson.Field
		InterruptResponse respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseAudioInputTurnDetectionSemanticVad) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeSessionCreateResponseAudioOutput struct {
	// The format of the output audio.
	Format RealtimeAudioFormatsUnion `json:"format"`
	// The speed of the model's spoken response as a multiple of the original speed.
	// 1.0 is the default speed. 0.25 is the minimum speed. 1.5 is the maximum speed.
	// This value can only be changed in between model turns, not while a response is
	// in progress.
	//
	// This parameter is a post-processing adjustment to the audio after it is
	// generated, it's also possible to prompt the model to speak faster or slower.
	Speed float64 `json:"speed"`
	// The voice the model uses to respond. Voice cannot be changed during the session
	// once the model has responded with audio at least once. Current voice options are
	// `alloy`, `ash`, `ballad`, `coral`, `echo`, `sage`, `shimmer`, `verse`, `marin`,
	// and `cedar`. We recommend `marin` and `cedar` for best quality.
	Voice string `json:"voice"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Format      respjson.Field
		Speed       respjson.Field
		Voice       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseAudioOutput) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseAudioOutput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeSessionCreateResponseMaxOutputTokensUnion contains all possible
// properties and values from [int64], [constant.Inf].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfInt OfInf]
type RealtimeSessionCreateResponseMaxOutputTokensUnion struct {
	// This field will be present if the value is a [int64] instead of an object.
	OfInt int64 `json:",inline"`
	// This field will be present if the value is a [constant.Inf] instead of an
	// object.
	OfInf constant.Inf `json:",inline"`
	JSON  struct {
		OfInt respjson.Field
		OfInf respjson.Field
		raw   string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseMaxOutputTokensUnion) AsInt() (v int64) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseMaxOutputTokensUnion) AsInf() (v constant.Inf) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseMaxOutputTokensUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeSessionCreateResponseMaxOutputTokensUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The Realtime model used for this session.
type RealtimeSessionCreateResponseModel string

const (
	RealtimeSessionCreateResponseModelGPTRealtime                        RealtimeSessionCreateResponseModel = "gpt-realtime"
	RealtimeSessionCreateResponseModelGPTRealtime2025_08_28              RealtimeSessionCreateResponseModel = "gpt-realtime-2025-08-28"
	RealtimeSessionCreateResponseModelGPT4oRealtimePreview               RealtimeSessionCreateResponseModel = "gpt-4o-realtime-preview"
	RealtimeSessionCreateResponseModelGPT4oRealtimePreview2024_10_01     RealtimeSessionCreateResponseModel = "gpt-4o-realtime-preview-2024-10-01"
	RealtimeSessionCreateResponseModelGPT4oRealtimePreview2024_12_17     RealtimeSessionCreateResponseModel = "gpt-4o-realtime-preview-2024-12-17"
	RealtimeSessionCreateResponseModelGPT4oRealtimePreview2025_06_03     RealtimeSessionCreateResponseModel = "gpt-4o-realtime-preview-2025-06-03"
	RealtimeSessionCreateResponseModelGPT4oMiniRealtimePreview           RealtimeSessionCreateResponseModel = "gpt-4o-mini-realtime-preview"
	RealtimeSessionCreateResponseModelGPT4oMiniRealtimePreview2024_12_17 RealtimeSessionCreateResponseModel = "gpt-4o-mini-realtime-preview-2024-12-17"
)

// RealtimeSessionCreateResponseToolChoiceUnion contains all possible properties
// and values from [responses.ToolChoiceOptions], [responses.ToolChoiceFunction],
// [responses.ToolChoiceMcp].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfToolChoiceMode]
type RealtimeSessionCreateResponseToolChoiceUnion struct {
	// This field will be present if the value is a [responses.ToolChoiceOptions]
	// instead of an object.
	OfToolChoiceMode responses.ToolChoiceOptions `json:",inline"`
	Name             string                      `json:"name"`
	Type             string                      `json:"type"`
	// This field is from variant [responses.ToolChoiceMcp].
	ServerLabel string `json:"server_label"`
	JSON        struct {
		OfToolChoiceMode respjson.Field
		Name             respjson.Field
		Type             respjson.Field
		ServerLabel      respjson.Field
		raw              string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseToolChoiceUnion) AsToolChoiceMode() (v responses.ToolChoiceOptions) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseToolChoiceUnion) AsFunctionTool() (v responses.ToolChoiceFunction) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseToolChoiceUnion) AsMcpTool() (v responses.ToolChoiceMcp) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseToolChoiceUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeSessionCreateResponseToolChoiceUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeSessionCreateResponseToolUnion contains all possible properties and
// values from [RealtimeFunctionTool], [RealtimeSessionCreateResponseToolMcpTool].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type RealtimeSessionCreateResponseToolUnion struct {
	// This field is from variant [RealtimeFunctionTool].
	Description string `json:"description"`
	// This field is from variant [RealtimeFunctionTool].
	Name string `json:"name"`
	// This field is from variant [RealtimeFunctionTool].
	Parameters any    `json:"parameters"`
	Type       string `json:"type"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	ServerLabel string `json:"server_label"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	AllowedTools RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion `json:"allowed_tools"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	Authorization string `json:"authorization"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	ConnectorID string `json:"connector_id"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	Headers map[string]string `json:"headers"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	RequireApproval RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion `json:"require_approval"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	ServerDescription string `json:"server_description"`
	// This field is from variant [RealtimeSessionCreateResponseToolMcpTool].
	ServerURL string `json:"server_url"`
	JSON      struct {
		Description       respjson.Field
		Name              respjson.Field
		Parameters        respjson.Field
		Type              respjson.Field
		ServerLabel       respjson.Field
		AllowedTools      respjson.Field
		Authorization     respjson.Field
		ConnectorID       respjson.Field
		Headers           respjson.Field
		RequireApproval   respjson.Field
		ServerDescription respjson.Field
		ServerURL         respjson.Field
		raw               string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseToolUnion) AsFunctionTool() (v RealtimeFunctionTool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseToolUnion) AsMcpTool() (v RealtimeSessionCreateResponseToolMcpTool) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseToolUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeSessionCreateResponseToolUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Give the model access to additional tools via remote Model Context Protocol
// (MCP) servers.
// [Learn more about MCP](https://platform.openai.com/docs/guides/tools-remote-mcp).
type RealtimeSessionCreateResponseToolMcpTool struct {
	// A label for this MCP server, used to identify it in tool calls.
	ServerLabel string `json:"server_label,required"`
	// The type of the MCP tool. Always `mcp`.
	Type constant.Mcp `json:"type,required"`
	// List of allowed tool names or a filter object.
	AllowedTools RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion `json:"allowed_tools,nullable"`
	// An OAuth access token that can be used with a remote MCP server, either with a
	// custom MCP server URL or a service connector. Your application must handle the
	// OAuth authorization flow and provide the token here.
	Authorization string `json:"authorization"`
	// Identifier for service connectors, like those available in ChatGPT. One of
	// `server_url` or `connector_id` must be provided. Learn more about service
	// connectors
	// [here](https://platform.openai.com/docs/guides/tools-remote-mcp#connectors).
	//
	// Currently supported `connector_id` values are:
	//
	// - Dropbox: `connector_dropbox`
	// - Gmail: `connector_gmail`
	// - Google Calendar: `connector_googlecalendar`
	// - Google Drive: `connector_googledrive`
	// - Microsoft Teams: `connector_microsoftteams`
	// - Outlook Calendar: `connector_outlookcalendar`
	// - Outlook Email: `connector_outlookemail`
	// - SharePoint: `connector_sharepoint`
	//
	// Any of "connector_dropbox", "connector_gmail", "connector_googlecalendar",
	// "connector_googledrive", "connector_microsoftteams",
	// "connector_outlookcalendar", "connector_outlookemail", "connector_sharepoint".
	ConnectorID string `json:"connector_id"`
	// Optional HTTP headers to send to the MCP server. Use for authentication or other
	// purposes.
	Headers map[string]string `json:"headers,nullable"`
	// Specify which of the MCP server's tools require approval.
	RequireApproval RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion `json:"require_approval,nullable"`
	// Optional description of the MCP server, used to provide more context.
	ServerDescription string `json:"server_description"`
	// The URL for the MCP server. One of `server_url` or `connector_id` must be
	// provided.
	ServerURL string `json:"server_url"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ServerLabel       respjson.Field
		Type              respjson.Field
		AllowedTools      respjson.Field
		Authorization     respjson.Field
		ConnectorID       respjson.Field
		Headers           respjson.Field
		RequireApproval   respjson.Field
		ServerDescription respjson.Field
		ServerURL         respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseToolMcpTool) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseToolMcpTool) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion contains all possible
// properties and values from [[]string],
// [RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMcpAllowedTools]
type RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion struct {
	// This field will be present if the value is a [[]string] instead of an object.
	OfMcpAllowedTools []string `json:",inline"`
	// This field is from variant
	// [RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter].
	ReadOnly bool `json:"read_only"`
	// This field is from variant
	// [RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter].
	ToolNames []string `json:"tool_names"`
	JSON      struct {
		OfMcpAllowedTools respjson.Field
		ReadOnly          respjson.Field
		ToolNames         respjson.Field
		raw               string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion) AsMcpAllowedTools() (v []string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion) AsMcpToolFilter() (v RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion) RawJSON() string {
	return u.JSON.raw
}

func (r *RealtimeSessionCreateResponseToolMcpToolAllowedToolsUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A filter object to specify which tools are allowed.
type RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly bool `json:"read_only"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ReadOnly    respjson.Field
		ToolNames   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseToolMcpToolAllowedToolsMcpToolFilter) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion contains all
// possible properties and values from
// [RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter],
// [string].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfMcpToolApprovalSetting]
type RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion struct {
	// This field will be present if the value is a [string] instead of an object.
	OfMcpToolApprovalSetting string `json:",inline"`
	// This field is from variant
	// [RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter].
	Always RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterAlways `json:"always"`
	// This field is from variant
	// [RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter].
	Never RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterNever `json:"never"`
	JSON  struct {
		OfMcpToolApprovalSetting respjson.Field
		Always                   respjson.Field
		Never                    respjson.Field
		raw                      string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion) AsMcpToolApprovalFilter() (v RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion) AsMcpToolApprovalSetting() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion) RawJSON() string {
	return u.JSON.raw
}

func (r *RealtimeSessionCreateResponseToolMcpToolRequireApprovalUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Specify which of the MCP server's tools require approval. Can be `always`,
// `never`, or a filter object associated with tools that require approval.
type RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter struct {
	// A filter object to specify which tools are allowed.
	Always RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterAlways `json:"always"`
	// A filter object to specify which tools are allowed.
	Never RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterNever `json:"never"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Always      respjson.Field
		Never       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilter) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A filter object to specify which tools are allowed.
type RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterAlways struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly bool `json:"read_only"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ReadOnly    respjson.Field
		ToolNames   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterAlways) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterAlways) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A filter object to specify which tools are allowed.
type RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterNever struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly bool `json:"read_only"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ReadOnly    respjson.Field
		ToolNames   respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterNever) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalFilterNever) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Specify a single approval policy for all tools. One of `always` or `never`. When
// set to `always`, all tools will require approval. When set to `never`, all tools
// will not require approval.
type RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalSetting string

const (
	RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalSettingAlways RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalSetting = "always"
	RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalSettingNever  RealtimeSessionCreateResponseToolMcpToolRequireApprovalMcpToolApprovalSetting = "never"
)

// RealtimeSessionCreateResponseTracingUnion contains all possible properties and
// values from [constant.Auto],
// [RealtimeSessionCreateResponseTracingTracingConfiguration].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfAuto]
type RealtimeSessionCreateResponseTracingUnion struct {
	// This field will be present if the value is a [constant.Auto] instead of an
	// object.
	OfAuto constant.Auto `json:",inline"`
	// This field is from variant
	// [RealtimeSessionCreateResponseTracingTracingConfiguration].
	GroupID string `json:"group_id"`
	// This field is from variant
	// [RealtimeSessionCreateResponseTracingTracingConfiguration].
	Metadata any `json:"metadata"`
	// This field is from variant
	// [RealtimeSessionCreateResponseTracingTracingConfiguration].
	WorkflowName string `json:"workflow_name"`
	JSON         struct {
		OfAuto       respjson.Field
		GroupID      respjson.Field
		Metadata     respjson.Field
		WorkflowName respjson.Field
		raw          string
	} `json:"-"`
}

func (u RealtimeSessionCreateResponseTracingUnion) AsAuto() (v constant.Auto) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeSessionCreateResponseTracingUnion) AsTracingConfiguration() (v RealtimeSessionCreateResponseTracingTracingConfiguration) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeSessionCreateResponseTracingUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeSessionCreateResponseTracingUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Granular configuration for tracing.
type RealtimeSessionCreateResponseTracingTracingConfiguration struct {
	// The group id to attach to this trace to enable filtering and grouping in the
	// Traces Dashboard.
	GroupID string `json:"group_id"`
	// The arbitrary metadata to attach to this trace to enable filtering in the Traces
	// Dashboard.
	Metadata any `json:"metadata"`
	// The name of the workflow to attach to this trace. This is used to name the trace
	// in the Traces Dashboard.
	WorkflowName string `json:"workflow_name"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		GroupID      respjson.Field
		Metadata     respjson.Field
		WorkflowName respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeSessionCreateResponseTracingTracingConfiguration) RawJSON() string { return r.JSON.raw }
func (r *RealtimeSessionCreateResponseTracingTracingConfiguration) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A Realtime transcription session configuration object.
type RealtimeTranscriptionSessionCreateResponse struct {
	// Unique identifier for the session that looks like `sess_1234567890abcdef`.
	ID string `json:"id,required"`
	// The object type. Always `realtime.transcription_session`.
	Object string `json:"object,required"`
	// The type of session. Always `transcription` for transcription sessions.
	Type constant.Transcription `json:"type,required"`
	// Configuration for input audio for the session.
	Audio RealtimeTranscriptionSessionCreateResponseAudio `json:"audio"`
	// Expiration timestamp for the session, in seconds since epoch.
	ExpiresAt int64 `json:"expires_at"`
	// Additional fields to include in server outputs.
	//
	//   - `item.input_audio_transcription.logprobs`: Include logprobs for input audio
	//     transcription.
	//
	// Any of "item.input_audio_transcription.logprobs".
	Include []string `json:"include"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID          respjson.Field
		Object      respjson.Field
		Type        respjson.Field
		Audio       respjson.Field
		ExpiresAt   respjson.Field
		Include     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTranscriptionSessionCreateResponse) RawJSON() string { return r.JSON.raw }
func (r *RealtimeTranscriptionSessionCreateResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input audio for the session.
type RealtimeTranscriptionSessionCreateResponseAudio struct {
	Input RealtimeTranscriptionSessionCreateResponseAudioInput `json:"input"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Input       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTranscriptionSessionCreateResponseAudio) RawJSON() string { return r.JSON.raw }
func (r *RealtimeTranscriptionSessionCreateResponseAudio) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeTranscriptionSessionCreateResponseAudioInput struct {
	// The PCM audio format. Only a 24kHz sample rate is supported.
	Format RealtimeAudioFormatsUnion `json:"format"`
	// Configuration for input audio noise reduction.
	NoiseReduction RealtimeTranscriptionSessionCreateResponseAudioInputNoiseReduction `json:"noise_reduction"`
	// Configuration of the transcription model.
	Transcription AudioTranscription `json:"transcription"`
	// Configuration for turn detection. Can be set to `null` to turn off. Server VAD
	// means that the model will detect the start and end of speech based on audio
	// volume and respond at the end of user speech.
	TurnDetection RealtimeTranscriptionSessionTurnDetection `json:"turn_detection"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Format         respjson.Field
		NoiseReduction respjson.Field
		Transcription  respjson.Field
		TurnDetection  respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTranscriptionSessionCreateResponseAudioInput) RawJSON() string { return r.JSON.raw }
func (r *RealtimeTranscriptionSessionCreateResponseAudioInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input audio noise reduction.
type RealtimeTranscriptionSessionCreateResponseAudioInputNoiseReduction struct {
	// Type of noise reduction. `near_field` is for close-talking microphones such as
	// headphones, `far_field` is for far-field microphones such as laptop or
	// conference room microphones.
	//
	// Any of "near_field", "far_field".
	Type NoiseReductionType `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTranscriptionSessionCreateResponseAudioInputNoiseReduction) RawJSON() string {
	return r.JSON.raw
}
func (r *RealtimeTranscriptionSessionCreateResponseAudioInputNoiseReduction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for turn detection. Can be set to `null` to turn off. Server VAD
// means that the model will detect the start and end of speech based on audio
// volume and respond at the end of user speech.
type RealtimeTranscriptionSessionTurnDetection struct {
	// Amount of audio to include before the VAD detected speech (in milliseconds).
	// Defaults to 300ms.
	PrefixPaddingMs int64 `json:"prefix_padding_ms"`
	// Duration of silence to detect speech stop (in milliseconds). Defaults to 500ms.
	// With shorter values the model will respond more quickly, but may jump in on
	// short pauses from the user.
	SilenceDurationMs int64 `json:"silence_duration_ms"`
	// Activation threshold for VAD (0.0 to 1.0), this defaults to 0.5. A higher
	// threshold will require louder audio to activate the model, and thus might
	// perform better in noisy environments.
	Threshold float64 `json:"threshold"`
	// Type of turn detection, only `server_vad` is currently supported.
	Type string `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		PrefixPaddingMs   respjson.Field
		SilenceDurationMs respjson.Field
		Threshold         respjson.Field
		Type              respjson.Field
		ExtraFields       map[string]respjson.Field
		raw               string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTranscriptionSessionTurnDetection) RawJSON() string { return r.JSON.raw }
func (r *RealtimeTranscriptionSessionTurnDetection) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Response from creating a session and client secret for the Realtime API.
type ClientSecretNewResponse struct {
	// Expiration timestamp for the client secret, in seconds since epoch.
	ExpiresAt int64 `json:"expires_at,required"`
	// The session configuration for either a realtime or transcription session.
	Session ClientSecretNewResponseSessionUnion `json:"session,required"`
	// The generated client secret value.
	Value string `json:"value,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ExpiresAt   respjson.Field
		Session     respjson.Field
		Value       respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ClientSecretNewResponse) RawJSON() string { return r.JSON.raw }
func (r *ClientSecretNewResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ClientSecretNewResponseSessionUnion contains all possible properties and values
// from [RealtimeSessionCreateResponse],
// [RealtimeTranscriptionSessionCreateResponse].
//
// Use the [ClientSecretNewResponseSessionUnion.AsAny] method to switch on the
// variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type ClientSecretNewResponseSessionUnion struct {
	// This field is from variant [RealtimeSessionCreateResponse].
	ClientSecret RealtimeSessionClientSecret `json:"client_secret"`
	// Any of "realtime", "transcription".
	Type string `json:"type"`
	// This field is a union of [RealtimeSessionCreateResponseAudio],
	// [RealtimeTranscriptionSessionCreateResponseAudio]
	Audio   ClientSecretNewResponseSessionUnionAudio `json:"audio"`
	Include []string                                 `json:"include"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Instructions string `json:"instructions"`
	// This field is from variant [RealtimeSessionCreateResponse].
	MaxOutputTokens RealtimeSessionCreateResponseMaxOutputTokensUnion `json:"max_output_tokens"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Model RealtimeSessionCreateResponseModel `json:"model"`
	// This field is from variant [RealtimeSessionCreateResponse].
	OutputModalities []string `json:"output_modalities"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Prompt responses.ResponsePrompt `json:"prompt"`
	// This field is from variant [RealtimeSessionCreateResponse].
	ToolChoice RealtimeSessionCreateResponseToolChoiceUnion `json:"tool_choice"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Tools []RealtimeSessionCreateResponseToolUnion `json:"tools"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Tracing RealtimeSessionCreateResponseTracingUnion `json:"tracing"`
	// This field is from variant [RealtimeSessionCreateResponse].
	Truncation RealtimeTruncationUnion `json:"truncation"`
	// This field is from variant [RealtimeTranscriptionSessionCreateResponse].
	ID string `json:"id"`
	// This field is from variant [RealtimeTranscriptionSessionCreateResponse].
	Object string `json:"object"`
	// This field is from variant [RealtimeTranscriptionSessionCreateResponse].
	ExpiresAt int64 `json:"expires_at"`
	JSON      struct {
		ClientSecret     respjson.Field
		Type             respjson.Field
		Audio            respjson.Field
		Include          respjson.Field
		Instructions     respjson.Field
		MaxOutputTokens  respjson.Field
		Model            respjson.Field
		OutputModalities respjson.Field
		Prompt           respjson.Field
		ToolChoice       respjson.Field
		Tools            respjson.Field
		Tracing          respjson.Field
		Truncation       respjson.Field
		ID               respjson.Field
		Object           respjson.Field
		ExpiresAt        respjson.Field
		raw              string
	} `json:"-"`
}

// anyClientSecretNewResponseSession is implemented by each variant of
// [ClientSecretNewResponseSessionUnion] to add type safety for the return type of
// [ClientSecretNewResponseSessionUnion.AsAny]
type anyClientSecretNewResponseSession interface {
	implClientSecretNewResponseSessionUnion()
}

func (RealtimeSessionCreateResponse) implClientSecretNewResponseSessionUnion()              {}
func (RealtimeTranscriptionSessionCreateResponse) implClientSecretNewResponseSessionUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := ClientSecretNewResponseSessionUnion.AsAny().(type) {
//	case realtime.RealtimeSessionCreateResponse:
//	case realtime.RealtimeTranscriptionSessionCreateResponse:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u ClientSecretNewResponseSessionUnion) AsAny() anyClientSecretNewResponseSession {
	switch u.Type {
	case "realtime":
		return u.AsRealtime()
	case "transcription":
		return u.AsTranscription()
	}
	return nil
}

func (u ClientSecretNewResponseSessionUnion) AsRealtime() (v RealtimeSessionCreateResponse) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u ClientSecretNewResponseSessionUnion) AsTranscription() (v RealtimeTranscriptionSessionCreateResponse) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u ClientSecretNewResponseSessionUnion) RawJSON() string { return u.JSON.raw }

func (r *ClientSecretNewResponseSessionUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ClientSecretNewResponseSessionUnionAudio is an implicit subunion of
// [ClientSecretNewResponseSessionUnion]. ClientSecretNewResponseSessionUnionAudio
// provides convenient access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ClientSecretNewResponseSessionUnion].
type ClientSecretNewResponseSessionUnionAudio struct {
	// This field is a union of [RealtimeSessionCreateResponseAudioInput],
	// [RealtimeTranscriptionSessionCreateResponseAudioInput]
	Input ClientSecretNewResponseSessionUnionAudioInput `json:"input"`
	// This field is from variant [RealtimeSessionCreateResponseAudio].
	Output RealtimeSessionCreateResponseAudioOutput `json:"output"`
	JSON   struct {
		Input  respjson.Field
		Output respjson.Field
		raw    string
	} `json:"-"`
}

func (r *ClientSecretNewResponseSessionUnionAudio) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ClientSecretNewResponseSessionUnionAudioInput is an implicit subunion of
// [ClientSecretNewResponseSessionUnion].
// ClientSecretNewResponseSessionUnionAudioInput provides convenient access to the
// sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ClientSecretNewResponseSessionUnion].
type ClientSecretNewResponseSessionUnionAudioInput struct {
	// This field is from variant [RealtimeSessionCreateResponseAudioInput].
	Format RealtimeAudioFormatsUnion `json:"format"`
	// This field is a union of
	// [RealtimeSessionCreateResponseAudioInputNoiseReduction],
	// [RealtimeTranscriptionSessionCreateResponseAudioInputNoiseReduction]
	NoiseReduction ClientSecretNewResponseSessionUnionAudioInputNoiseReduction `json:"noise_reduction"`
	// This field is from variant [RealtimeSessionCreateResponseAudioInput].
	Transcription AudioTranscription `json:"transcription"`
	// This field is a union of
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion],
	// [RealtimeTranscriptionSessionTurnDetection]
	TurnDetection ClientSecretNewResponseSessionUnionAudioInputTurnDetection `json:"turn_detection"`
	JSON          struct {
		Format         respjson.Field
		NoiseReduction respjson.Field
		Transcription  respjson.Field
		TurnDetection  respjson.Field
		raw            string
	} `json:"-"`
}

func (r *ClientSecretNewResponseSessionUnionAudioInput) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ClientSecretNewResponseSessionUnionAudioInputNoiseReduction is an implicit
// subunion of [ClientSecretNewResponseSessionUnion].
// ClientSecretNewResponseSessionUnionAudioInputNoiseReduction provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ClientSecretNewResponseSessionUnion].
type ClientSecretNewResponseSessionUnionAudioInputNoiseReduction struct {
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputNoiseReduction].
	Type NoiseReductionType `json:"type"`
	JSON struct {
		Type respjson.Field
		raw  string
	} `json:"-"`
}

func (r *ClientSecretNewResponseSessionUnionAudioInputNoiseReduction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ClientSecretNewResponseSessionUnionAudioInputTurnDetection is an implicit
// subunion of [ClientSecretNewResponseSessionUnion].
// ClientSecretNewResponseSessionUnionAudioInputTurnDetection provides convenient
// access to the sub-properties of the union.
//
// For type safety it is recommended to directly use a variant of the
// [ClientSecretNewResponseSessionUnion].
type ClientSecretNewResponseSessionUnionAudioInputTurnDetection struct {
	Type           string `json:"type"`
	CreateResponse bool   `json:"create_response"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion].
	IdleTimeoutMs     int64   `json:"idle_timeout_ms"`
	InterruptResponse bool    `json:"interrupt_response"`
	PrefixPaddingMs   int64   `json:"prefix_padding_ms"`
	SilenceDurationMs int64   `json:"silence_duration_ms"`
	Threshold         float64 `json:"threshold"`
	// This field is from variant
	// [RealtimeSessionCreateResponseAudioInputTurnDetectionUnion].
	Eagerness string `json:"eagerness"`
	JSON      struct {
		Type              respjson.Field
		CreateResponse    respjson.Field
		IdleTimeoutMs     respjson.Field
		InterruptResponse respjson.Field
		PrefixPaddingMs   respjson.Field
		SilenceDurationMs respjson.Field
		Threshold         respjson.Field
		Eagerness         respjson.Field
		raw               string
	} `json:"-"`
}

func (r *ClientSecretNewResponseSessionUnionAudioInputTurnDetection) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type ClientSecretNewParams struct {
	// Configuration for the client secret expiration. Expiration refers to the time
	// after which a client secret will no longer be valid for creating sessions. The
	// session itself may continue after that time once started. A secret can be used
	// to create multiple sessions until it expires.
	ExpiresAfter ClientSecretNewParamsExpiresAfter `json:"expires_after,omitzero"`
	// Session configuration to use for the client secret. Choose either a realtime
	// session or a transcription session.
	Session ClientSecretNewParamsSessionUnion `json:"session,omitzero"`
	paramObj
}

func (r ClientSecretNewParams) MarshalJSON() (data []byte, err error) {
	type shadow ClientSecretNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientSecretNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for the client secret expiration. Expiration refers to the time
// after which a client secret will no longer be valid for creating sessions. The
// session itself may continue after that time once started. A secret can be used
// to create multiple sessions until it expires.
type ClientSecretNewParamsExpiresAfter struct {
	// The number of seconds from the anchor point to the expiration. Select a value
	// between `10` and `7200` (2 hours). This default to 600 seconds (10 minutes) if
	// not specified.
	Seconds param.Opt[int64] `json:"seconds,omitzero"`
	// The anchor point for the client secret expiration, meaning that `seconds` will
	// be added to the `created_at` time of the client secret to produce an expiration
	// timestamp. Only `created_at` is currently supported.
	//
	// Any of "created_at".
	Anchor string `json:"anchor,omitzero"`
	paramObj
}

func (r ClientSecretNewParamsExpiresAfter) MarshalJSON() (data []byte, err error) {
	type shadow ClientSecretNewParamsExpiresAfter
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *ClientSecretNewParamsExpiresAfter) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[ClientSecretNewParamsExpiresAfter](
		"anchor", "created_at",
	)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type ClientSecretNewParamsSessionUnion struct {
	OfRealtime      *RealtimeSessionCreateRequestParam              `json:",omitzero,inline"`
	OfTranscription *RealtimeTranscriptionSessionCreateRequestParam `json:",omitzero,inline"`
	paramUnion
}

func (u ClientSecretNewParamsSessionUnion) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRealtime, u.OfTranscription)
}
func (u *ClientSecretNewParamsSessionUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *ClientSecretNewParamsSessionUnion) asAny() any {
	if !param.IsOmitted(u.OfRealtime) {
		return u.OfRealtime
	} else if !param.IsOmitted(u.OfTranscription) {
		return u.OfTranscription
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetInstructions() *string {
	if vt := u.OfRealtime; vt != nil && vt.Instructions.Valid() {
		return &vt.Instructions.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetMaxOutputTokens() *RealtimeSessionCreateRequestMaxOutputTokensUnionParam {
	if vt := u.OfRealtime; vt != nil {
		return &vt.MaxOutputTokens
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetModel() *RealtimeSessionCreateRequestModel {
	if vt := u.OfRealtime; vt != nil {
		return &vt.Model
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetOutputModalities() []string {
	if vt := u.OfRealtime; vt != nil {
		return vt.OutputModalities
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetPrompt() *responses.ResponsePromptParam {
	if vt := u.OfRealtime; vt != nil {
		return &vt.Prompt
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetToolChoice() *RealtimeToolChoiceConfigUnionParam {
	if vt := u.OfRealtime; vt != nil {
		return &vt.ToolChoice
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetTools() RealtimeToolsConfigParam {
	if vt := u.OfRealtime; vt != nil {
		return vt.Tools
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetTracing() *RealtimeTracingConfigUnionParam {
	if vt := u.OfRealtime; vt != nil {
		return &vt.Tracing
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetTruncation() *RealtimeTruncationUnionParam {
	if vt := u.OfRealtime; vt != nil {
		return &vt.Truncation
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u ClientSecretNewParamsSessionUnion) GetType() *string {
	if vt := u.OfRealtime; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfTranscription; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u ClientSecretNewParamsSessionUnion) GetAudio() (res clientSecretNewParamsSessionUnionAudio) {
	if vt := u.OfRealtime; vt != nil {
		res.any = &vt.Audio
	} else if vt := u.OfTranscription; vt != nil {
		res.any = &vt.Audio
	}
	return
}

// Can have the runtime types [*RealtimeAudioConfigParam],
// [*RealtimeTranscriptionSessionAudioParam]
type clientSecretNewParamsSessionUnionAudio struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *realtime.RealtimeAudioConfigParam:
//	case *realtime.RealtimeTranscriptionSessionAudioParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u clientSecretNewParamsSessionUnionAudio) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudio) GetOutput() *RealtimeAudioConfigOutputParam {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigParam:
		return &vt.Output
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u clientSecretNewParamsSessionUnionAudio) GetInput() (res clientSecretNewParamsSessionUnionAudioInput) {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigParam:
		res.any = &vt.Input
	case *RealtimeTranscriptionSessionAudioParam:
		res.any = &vt.Input
	}
	return res
}

// Can have the runtime types [*RealtimeAudioConfigInputParam],
// [*RealtimeTranscriptionSessionAudioInputParam]
type clientSecretNewParamsSessionUnionAudioInput struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *realtime.RealtimeAudioConfigInputParam:
//	case *realtime.RealtimeTranscriptionSessionAudioInputParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u clientSecretNewParamsSessionUnionAudioInput) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's Format property, if present.
func (u clientSecretNewParamsSessionUnionAudioInput) GetFormat() *RealtimeAudioFormatsUnionParam {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigInputParam:
		return &vt.Format
	case *RealtimeTranscriptionSessionAudioInputParam:
		return &vt.Format
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u clientSecretNewParamsSessionUnionAudioInput) GetNoiseReduction() (res clientSecretNewParamsSessionUnionAudioInputNoiseReduction) {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigInputParam:
		res.any = &vt.NoiseReduction
	case *RealtimeTranscriptionSessionAudioInputParam:
		res.any = &vt.NoiseReduction
	}
	return res
}

// Can have the runtime types [*RealtimeAudioConfigInputNoiseReductionParam],
// [*RealtimeTranscriptionSessionAudioInputNoiseReductionParam]
type clientSecretNewParamsSessionUnionAudioInputNoiseReduction struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *realtime.RealtimeAudioConfigInputNoiseReductionParam:
//	case *realtime.RealtimeTranscriptionSessionAudioInputNoiseReductionParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u clientSecretNewParamsSessionUnionAudioInputNoiseReduction) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputNoiseReduction) GetType() *string {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigInputNoiseReductionParam:
		return (*string)(&vt.Type)
	case *RealtimeTranscriptionSessionAudioInputNoiseReductionParam:
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's Transcription property, if
// present.
func (u clientSecretNewParamsSessionUnionAudioInput) GetTranscription() *AudioTranscriptionParam {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigInputParam:
		return &vt.Transcription
	case *RealtimeTranscriptionSessionAudioInputParam:
		return &vt.Transcription
	}
	return nil
}

// Returns a subunion which exports methods to access subproperties
//
// Or use AsAny() to get the underlying value
func (u clientSecretNewParamsSessionUnionAudioInput) GetTurnDetection() (res clientSecretNewParamsSessionUnionAudioInputTurnDetection) {
	switch vt := u.any.(type) {
	case *RealtimeAudioConfigInputParam:
		res.any = vt.TurnDetection
	case *RealtimeTranscriptionSessionAudioInputParam:
		res.any = vt.TurnDetection
	}
	return res
}

// Can have the runtime types [*RealtimeAudioInputTurnDetectionServerVadParam],
// [*RealtimeAudioInputTurnDetectionSemanticVadParam],
// [*RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam],
// [*RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam]
type clientSecretNewParamsSessionUnionAudioInputTurnDetection struct{ any }

// Use the following switch statement to get the type of the union:
//
//	switch u.AsAny().(type) {
//	case *realtime.RealtimeAudioInputTurnDetectionServerVadParam:
//	case *realtime.RealtimeAudioInputTurnDetectionSemanticVadParam:
//	case *realtime.RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam:
//	case *realtime.RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam:
//	default:
//	    fmt.Errorf("not present")
//	}
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) AsAny() any { return u.any }

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetType() *string {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetType()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetType()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetCreateResponse() *bool {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetCreateResponse()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetCreateResponse()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetIdleTimeoutMs() *int64 {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetIdleTimeoutMs()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetIdleTimeoutMs()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetInterruptResponse() *bool {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetInterruptResponse()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetInterruptResponse()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetPrefixPaddingMs() *int64 {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetPrefixPaddingMs()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetPrefixPaddingMs()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetSilenceDurationMs() *int64 {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetSilenceDurationMs()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetSilenceDurationMs()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetThreshold() *float64 {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetThreshold()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetThreshold()
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u clientSecretNewParamsSessionUnionAudioInputTurnDetection) GetEagerness() *string {
	switch vt := u.any.(type) {
	case *RealtimeAudioInputTurnDetectionUnionParam:
		return vt.GetEagerness()
	case *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam:
		return vt.GetEagerness()
	}
	return nil
}

// Returns a pointer to the underlying variant's Include property, if present.
func (u ClientSecretNewParamsSessionUnion) GetInclude() []string {
	if vt := u.OfRealtime; vt != nil {
		return vt.Include
	} else if vt := u.OfTranscription; vt != nil {
		return vt.Include
	}
	return nil
}

func init() {
	apijson.RegisterUnion[ClientSecretNewParamsSessionUnion](
		"type",
		apijson.Discriminator[RealtimeSessionCreateRequestParam]("realtime"),
		apijson.Discriminator[RealtimeTranscriptionSessionCreateRequestParam]("transcription"),
	)
}
