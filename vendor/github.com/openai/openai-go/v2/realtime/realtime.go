// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package realtime

import (
	"encoding/json"

	"github.com/openai/openai-go/v2/internal/apijson"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/packages/respjson"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

// RealtimeService contains methods and other services that help with interacting
// with the openai API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewRealtimeService] method instead.
type RealtimeService struct {
	Options       []option.RequestOption
	ClientSecrets ClientSecretService
}

// NewRealtimeService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewRealtimeService(opts ...option.RequestOption) (r RealtimeService) {
	r = RealtimeService{}
	r.Options = opts
	r.ClientSecrets = NewClientSecretService(opts...)
	return
}

type AudioTranscription struct {
	// The language of the input audio. Supplying the input language in
	// [ISO-639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g. `en`)
	// format will improve accuracy and latency.
	Language string `json:"language"`
	// The model to use for transcription. Current options are `whisper-1`,
	// `gpt-4o-transcribe-latest`, `gpt-4o-mini-transcribe`, and `gpt-4o-transcribe`.
	//
	// Any of "whisper-1", "gpt-4o-transcribe-latest", "gpt-4o-mini-transcribe",
	// "gpt-4o-transcribe".
	Model AudioTranscriptionModel `json:"model"`
	// An optional text to guide the model's style or continue a previous audio
	// segment. For `whisper-1`, the
	// [prompt is a list of keywords](https://platform.openai.com/docs/guides/speech-to-text#prompting).
	// For `gpt-4o-transcribe` models, the prompt is a free text string, for example
	// "expect words related to technology".
	Prompt string `json:"prompt"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Language    respjson.Field
		Model       respjson.Field
		Prompt      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r AudioTranscription) RawJSON() string { return r.JSON.raw }
func (r *AudioTranscription) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this AudioTranscription to a AudioTranscriptionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// AudioTranscriptionParam.Overrides()
func (r AudioTranscription) ToParam() AudioTranscriptionParam {
	return param.Override[AudioTranscriptionParam](json.RawMessage(r.RawJSON()))
}

// The model to use for transcription. Current options are `whisper-1`,
// `gpt-4o-transcribe-latest`, `gpt-4o-mini-transcribe`, and `gpt-4o-transcribe`.
type AudioTranscriptionModel string

const (
	AudioTranscriptionModelWhisper1              AudioTranscriptionModel = "whisper-1"
	AudioTranscriptionModelGPT4oTranscribeLatest AudioTranscriptionModel = "gpt-4o-transcribe-latest"
	AudioTranscriptionModelGPT4oMiniTranscribe   AudioTranscriptionModel = "gpt-4o-mini-transcribe"
	AudioTranscriptionModelGPT4oTranscribe       AudioTranscriptionModel = "gpt-4o-transcribe"
)

type AudioTranscriptionParam struct {
	// The language of the input audio. Supplying the input language in
	// [ISO-639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g. `en`)
	// format will improve accuracy and latency.
	Language param.Opt[string] `json:"language,omitzero"`
	// An optional text to guide the model's style or continue a previous audio
	// segment. For `whisper-1`, the
	// [prompt is a list of keywords](https://platform.openai.com/docs/guides/speech-to-text#prompting).
	// For `gpt-4o-transcribe` models, the prompt is a free text string, for example
	// "expect words related to technology".
	Prompt param.Opt[string] `json:"prompt,omitzero"`
	// The model to use for transcription. Current options are `whisper-1`,
	// `gpt-4o-transcribe-latest`, `gpt-4o-mini-transcribe`, and `gpt-4o-transcribe`.
	//
	// Any of "whisper-1", "gpt-4o-transcribe-latest", "gpt-4o-mini-transcribe",
	// "gpt-4o-transcribe".
	Model AudioTranscriptionModel `json:"model,omitzero"`
	paramObj
}

func (r AudioTranscriptionParam) MarshalJSON() (data []byte, err error) {
	type shadow AudioTranscriptionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *AudioTranscriptionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Type of noise reduction. `near_field` is for close-talking microphones such as
// headphones, `far_field` is for far-field microphones such as laptop or
// conference room microphones.
type NoiseReductionType string

const (
	NoiseReductionTypeNearField NoiseReductionType = "near_field"
	NoiseReductionTypeFarField  NoiseReductionType = "far_field"
)

// Configuration for input and output audio.
type RealtimeAudioConfigParam struct {
	Input  RealtimeAudioConfigInputParam  `json:"input,omitzero"`
	Output RealtimeAudioConfigOutputParam `json:"output,omitzero"`
	paramObj
}

func (r RealtimeAudioConfigParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioConfigParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioConfigParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeAudioConfigInputParam struct {
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
	TurnDetection RealtimeAudioInputTurnDetectionUnionParam `json:"turn_detection,omitzero"`
	// The format of the input audio.
	Format RealtimeAudioFormatsUnionParam `json:"format,omitzero"`
	// Configuration for input audio noise reduction. This can be set to `null` to turn
	// off. Noise reduction filters audio added to the input audio buffer before it is
	// sent to VAD and the model. Filtering the audio can improve VAD and turn
	// detection accuracy (reducing false positives) and model performance by improving
	// perception of the input audio.
	NoiseReduction RealtimeAudioConfigInputNoiseReductionParam `json:"noise_reduction,omitzero"`
	// Configuration for input audio transcription, defaults to off and can be set to
	// `null` to turn off once on. Input audio transcription is not native to the
	// model, since the model consumes audio directly. Transcription runs
	// asynchronously through
	// [the /audio/transcriptions endpoint](https://platform.openai.com/docs/api-reference/audio/createTranscription)
	// and should be treated as guidance of input audio content rather than precisely
	// what the model heard. The client can optionally set the language and prompt for
	// transcription, these offer additional guidance to the transcription service.
	Transcription AudioTranscriptionParam `json:"transcription,omitzero"`
	paramObj
}

func (r RealtimeAudioConfigInputParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioConfigInputParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioConfigInputParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input audio noise reduction. This can be set to `null` to turn
// off. Noise reduction filters audio added to the input audio buffer before it is
// sent to VAD and the model. Filtering the audio can improve VAD and turn
// detection accuracy (reducing false positives) and model performance by improving
// perception of the input audio.
type RealtimeAudioConfigInputNoiseReductionParam struct {
	// Type of noise reduction. `near_field` is for close-talking microphones such as
	// headphones, `far_field` is for far-field microphones such as laptop or
	// conference room microphones.
	//
	// Any of "near_field", "far_field".
	Type NoiseReductionType `json:"type,omitzero"`
	paramObj
}

func (r RealtimeAudioConfigInputNoiseReductionParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioConfigInputNoiseReductionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioConfigInputNoiseReductionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeAudioConfigOutputParam struct {
	// The speed of the model's spoken response as a multiple of the original speed.
	// 1.0 is the default speed. 0.25 is the minimum speed. 1.5 is the maximum speed.
	// This value can only be changed in between model turns, not while a response is
	// in progress.
	//
	// This parameter is a post-processing adjustment to the audio after it is
	// generated, it's also possible to prompt the model to speak faster or slower.
	Speed param.Opt[float64] `json:"speed,omitzero"`
	// The format of the output audio.
	Format RealtimeAudioFormatsUnionParam `json:"format,omitzero"`
	// The voice the model uses to respond. Voice cannot be changed during the session
	// once the model has responded with audio at least once. Current voice options are
	// `alloy`, `ash`, `ballad`, `coral`, `echo`, `sage`, `shimmer`, `verse`, `marin`,
	// and `cedar`. We recommend `marin` and `cedar` for best quality.
	Voice RealtimeAudioConfigOutputVoice `json:"voice,omitzero"`
	paramObj
}

func (r RealtimeAudioConfigOutputParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioConfigOutputParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioConfigOutputParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The voice the model uses to respond. Voice cannot be changed during the session
// once the model has responded with audio at least once. Current voice options are
// `alloy`, `ash`, `ballad`, `coral`, `echo`, `sage`, `shimmer`, `verse`, `marin`,
// and `cedar`. We recommend `marin` and `cedar` for best quality.
type RealtimeAudioConfigOutputVoice string

const (
	RealtimeAudioConfigOutputVoiceAlloy   RealtimeAudioConfigOutputVoice = "alloy"
	RealtimeAudioConfigOutputVoiceAsh     RealtimeAudioConfigOutputVoice = "ash"
	RealtimeAudioConfigOutputVoiceBallad  RealtimeAudioConfigOutputVoice = "ballad"
	RealtimeAudioConfigOutputVoiceCoral   RealtimeAudioConfigOutputVoice = "coral"
	RealtimeAudioConfigOutputVoiceEcho    RealtimeAudioConfigOutputVoice = "echo"
	RealtimeAudioConfigOutputVoiceSage    RealtimeAudioConfigOutputVoice = "sage"
	RealtimeAudioConfigOutputVoiceShimmer RealtimeAudioConfigOutputVoice = "shimmer"
	RealtimeAudioConfigOutputVoiceVerse   RealtimeAudioConfigOutputVoice = "verse"
	RealtimeAudioConfigOutputVoiceMarin   RealtimeAudioConfigOutputVoice = "marin"
	RealtimeAudioConfigOutputVoiceCedar   RealtimeAudioConfigOutputVoice = "cedar"
)

// RealtimeAudioFormatsUnion contains all possible properties and values from
// [RealtimeAudioFormatsAudioPCM], [RealtimeAudioFormatsAudioPCMU],
// [RealtimeAudioFormatsAudioPCMA].
//
// Use the [RealtimeAudioFormatsUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type RealtimeAudioFormatsUnion struct {
	// This field is from variant [RealtimeAudioFormatsAudioPCM].
	Rate int64 `json:"rate"`
	// Any of "audio/pcm", "audio/pcmu", "audio/pcma".
	Type string `json:"type"`
	JSON struct {
		Rate respjson.Field
		Type respjson.Field
		raw  string
	} `json:"-"`
}

// anyRealtimeAudioFormats is implemented by each variant of
// [RealtimeAudioFormatsUnion] to add type safety for the return type of
// [RealtimeAudioFormatsUnion.AsAny]
type anyRealtimeAudioFormats interface {
	implRealtimeAudioFormatsUnion()
}

func (RealtimeAudioFormatsAudioPCM) implRealtimeAudioFormatsUnion()  {}
func (RealtimeAudioFormatsAudioPCMU) implRealtimeAudioFormatsUnion() {}
func (RealtimeAudioFormatsAudioPCMA) implRealtimeAudioFormatsUnion() {}

// Use the following switch statement to find the correct variant
//
//	switch variant := RealtimeAudioFormatsUnion.AsAny().(type) {
//	case realtime.RealtimeAudioFormatsAudioPCM:
//	case realtime.RealtimeAudioFormatsAudioPCMU:
//	case realtime.RealtimeAudioFormatsAudioPCMA:
//	default:
//	  fmt.Errorf("no variant present")
//	}
func (u RealtimeAudioFormatsUnion) AsAny() anyRealtimeAudioFormats {
	switch u.Type {
	case "audio/pcm":
		return u.AsAudioPCM()
	case "audio/pcmu":
		return u.AsAudioPCMU()
	case "audio/pcma":
		return u.AsAudioPCMA()
	}
	return nil
}

func (u RealtimeAudioFormatsUnion) AsAudioPCM() (v RealtimeAudioFormatsAudioPCM) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeAudioFormatsUnion) AsAudioPCMU() (v RealtimeAudioFormatsAudioPCMU) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeAudioFormatsUnion) AsAudioPCMA() (v RealtimeAudioFormatsAudioPCMA) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeAudioFormatsUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeAudioFormatsUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this RealtimeAudioFormatsUnion to a
// RealtimeAudioFormatsUnionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// RealtimeAudioFormatsUnionParam.Overrides()
func (r RealtimeAudioFormatsUnion) ToParam() RealtimeAudioFormatsUnionParam {
	return param.Override[RealtimeAudioFormatsUnionParam](json.RawMessage(r.RawJSON()))
}

// The PCM audio format. Only a 24kHz sample rate is supported.
type RealtimeAudioFormatsAudioPCM struct {
	// The sample rate of the audio. Always `24000`.
	//
	// Any of 24000.
	Rate int64 `json:"rate"`
	// The audio format. Always `audio/pcm`.
	//
	// Any of "audio/pcm".
	Type string `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Rate        respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeAudioFormatsAudioPCM) RawJSON() string { return r.JSON.raw }
func (r *RealtimeAudioFormatsAudioPCM) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The G.711 μ-law format.
type RealtimeAudioFormatsAudioPCMU struct {
	// The audio format. Always `audio/pcmu`.
	//
	// Any of "audio/pcmu".
	Type string `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeAudioFormatsAudioPCMU) RawJSON() string { return r.JSON.raw }
func (r *RealtimeAudioFormatsAudioPCMU) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// The G.711 A-law format.
type RealtimeAudioFormatsAudioPCMA struct {
	// The audio format. Always `audio/pcma`.
	//
	// Any of "audio/pcma".
	Type string `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeAudioFormatsAudioPCMA) RawJSON() string { return r.JSON.raw }
func (r *RealtimeAudioFormatsAudioPCMA) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeAudioFormatsUnionParam struct {
	OfAudioPCM  *RealtimeAudioFormatsAudioPCMParam  `json:",omitzero,inline"`
	OfAudioPCMU *RealtimeAudioFormatsAudioPCMUParam `json:",omitzero,inline"`
	OfAudioPCMA *RealtimeAudioFormatsAudioPCMAParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeAudioFormatsUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfAudioPCM, u.OfAudioPCMU, u.OfAudioPCMA)
}
func (u *RealtimeAudioFormatsUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeAudioFormatsUnionParam) asAny() any {
	if !param.IsOmitted(u.OfAudioPCM) {
		return u.OfAudioPCM
	} else if !param.IsOmitted(u.OfAudioPCMU) {
		return u.OfAudioPCMU
	} else if !param.IsOmitted(u.OfAudioPCMA) {
		return u.OfAudioPCMA
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioFormatsUnionParam) GetRate() *int64 {
	if vt := u.OfAudioPCM; vt != nil {
		return &vt.Rate
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioFormatsUnionParam) GetType() *string {
	if vt := u.OfAudioPCM; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAudioPCMU; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfAudioPCMA; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[RealtimeAudioFormatsUnionParam](
		"type",
		apijson.Discriminator[RealtimeAudioFormatsAudioPCMParam]("audio/pcm"),
		apijson.Discriminator[RealtimeAudioFormatsAudioPCMUParam]("audio/pcmu"),
		apijson.Discriminator[RealtimeAudioFormatsAudioPCMAParam]("audio/pcma"),
	)
}

// The PCM audio format. Only a 24kHz sample rate is supported.
type RealtimeAudioFormatsAudioPCMParam struct {
	// The sample rate of the audio. Always `24000`.
	//
	// Any of 24000.
	Rate int64 `json:"rate,omitzero"`
	// The audio format. Always `audio/pcm`.
	//
	// Any of "audio/pcm".
	Type string `json:"type,omitzero"`
	paramObj
}

func (r RealtimeAudioFormatsAudioPCMParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioFormatsAudioPCMParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioFormatsAudioPCMParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeAudioFormatsAudioPCMParam](
		"rate", 24000,
	)
	apijson.RegisterFieldValidator[RealtimeAudioFormatsAudioPCMParam](
		"type", "audio/pcm",
	)
}

// The G.711 μ-law format.
type RealtimeAudioFormatsAudioPCMUParam struct {
	// The audio format. Always `audio/pcmu`.
	//
	// Any of "audio/pcmu".
	Type string `json:"type,omitzero"`
	paramObj
}

func (r RealtimeAudioFormatsAudioPCMUParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioFormatsAudioPCMUParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioFormatsAudioPCMUParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeAudioFormatsAudioPCMUParam](
		"type", "audio/pcmu",
	)
}

// The G.711 A-law format.
type RealtimeAudioFormatsAudioPCMAParam struct {
	// The audio format. Always `audio/pcma`.
	//
	// Any of "audio/pcma".
	Type string `json:"type,omitzero"`
	paramObj
}

func (r RealtimeAudioFormatsAudioPCMAParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioFormatsAudioPCMAParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioFormatsAudioPCMAParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeAudioFormatsAudioPCMAParam](
		"type", "audio/pcma",
	)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeAudioInputTurnDetectionUnionParam struct {
	OfServerVad   *RealtimeAudioInputTurnDetectionServerVadParam   `json:",omitzero,inline"`
	OfSemanticVad *RealtimeAudioInputTurnDetectionSemanticVadParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeAudioInputTurnDetectionUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfServerVad, u.OfSemanticVad)
}
func (u *RealtimeAudioInputTurnDetectionUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeAudioInputTurnDetectionUnionParam) asAny() any {
	if !param.IsOmitted(u.OfServerVad) {
		return u.OfServerVad
	} else if !param.IsOmitted(u.OfSemanticVad) {
		return u.OfSemanticVad
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetIdleTimeoutMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.IdleTimeoutMs.Valid() {
		return &vt.IdleTimeoutMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetPrefixPaddingMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.PrefixPaddingMs.Valid() {
		return &vt.PrefixPaddingMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetSilenceDurationMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.SilenceDurationMs.Valid() {
		return &vt.SilenceDurationMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetThreshold() *float64 {
	if vt := u.OfServerVad; vt != nil && vt.Threshold.Valid() {
		return &vt.Threshold.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetEagerness() *string {
	if vt := u.OfSemanticVad; vt != nil {
		return &vt.Eagerness
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetType() *string {
	if vt := u.OfServerVad; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSemanticVad; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetCreateResponse() *bool {
	if vt := u.OfServerVad; vt != nil && vt.CreateResponse.Valid() {
		return &vt.CreateResponse.Value
	} else if vt := u.OfSemanticVad; vt != nil && vt.CreateResponse.Valid() {
		return &vt.CreateResponse.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeAudioInputTurnDetectionUnionParam) GetInterruptResponse() *bool {
	if vt := u.OfServerVad; vt != nil && vt.InterruptResponse.Valid() {
		return &vt.InterruptResponse.Value
	} else if vt := u.OfSemanticVad; vt != nil && vt.InterruptResponse.Valid() {
		return &vt.InterruptResponse.Value
	}
	return nil
}

func init() {
	apijson.RegisterUnion[RealtimeAudioInputTurnDetectionUnionParam](
		"type",
		apijson.Discriminator[RealtimeAudioInputTurnDetectionServerVadParam]("server_vad"),
		apijson.Discriminator[RealtimeAudioInputTurnDetectionSemanticVadParam]("semantic_vad"),
	)
}

// Server-side voice activity detection (VAD) which flips on when user speech is
// detected and off after a period of silence.
//
// The property Type is required.
type RealtimeAudioInputTurnDetectionServerVadParam struct {
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
	IdleTimeoutMs param.Opt[int64] `json:"idle_timeout_ms,omitzero"`
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse param.Opt[bool] `json:"create_response,omitzero"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse param.Opt[bool] `json:"interrupt_response,omitzero"`
	// Used only for `server_vad` mode. Amount of audio to include before the VAD
	// detected speech (in milliseconds). Defaults to 300ms.
	PrefixPaddingMs param.Opt[int64] `json:"prefix_padding_ms,omitzero"`
	// Used only for `server_vad` mode. Duration of silence to detect speech stop (in
	// milliseconds). Defaults to 500ms. With shorter values the model will respond
	// more quickly, but may jump in on short pauses from the user.
	SilenceDurationMs param.Opt[int64] `json:"silence_duration_ms,omitzero"`
	// Used only for `server_vad` mode. Activation threshold for VAD (0.0 to 1.0), this
	// defaults to 0.5. A higher threshold will require louder audio to activate the
	// model, and thus might perform better in noisy environments.
	Threshold param.Opt[float64] `json:"threshold,omitzero"`
	// Type of turn detection, `server_vad` to turn on simple Server VAD.
	//
	// This field can be elided, and will marshal its zero value as "server_vad".
	Type constant.ServerVad `json:"type,required"`
	paramObj
}

func (r RealtimeAudioInputTurnDetectionServerVadParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioInputTurnDetectionServerVadParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioInputTurnDetectionServerVadParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Server-side semantic turn detection which uses a model to determine when the
// user has finished speaking.
//
// The property Type is required.
type RealtimeAudioInputTurnDetectionSemanticVadParam struct {
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse param.Opt[bool] `json:"create_response,omitzero"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse param.Opt[bool] `json:"interrupt_response,omitzero"`
	// Used only for `semantic_vad` mode. The eagerness of the model to respond. `low`
	// will wait longer for the user to continue speaking, `high` will respond more
	// quickly. `auto` is the default and is equivalent to `medium`. `low`, `medium`,
	// and `high` have max timeouts of 8s, 4s, and 2s respectively.
	//
	// Any of "low", "medium", "high", "auto".
	Eagerness string `json:"eagerness,omitzero"`
	// Type of turn detection, `semantic_vad` to turn on Semantic VAD.
	//
	// This field can be elided, and will marshal its zero value as "semantic_vad".
	Type constant.SemanticVad `json:"type,required"`
	paramObj
}

func (r RealtimeAudioInputTurnDetectionSemanticVadParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeAudioInputTurnDetectionSemanticVadParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeAudioInputTurnDetectionSemanticVadParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeAudioInputTurnDetectionSemanticVadParam](
		"eagerness", "low", "medium", "high", "auto",
	)
}

type RealtimeFunctionTool struct {
	// The description of the function, including guidance on when and how to call it,
	// and guidance about what to tell the user when calling (if anything).
	Description string `json:"description"`
	// The name of the function.
	Name string `json:"name"`
	// Parameters of the function in JSON Schema.
	Parameters any `json:"parameters"`
	// The type of the tool, i.e. `function`.
	//
	// Any of "function".
	Type RealtimeFunctionToolType `json:"type"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Description respjson.Field
		Name        respjson.Field
		Parameters  respjson.Field
		Type        respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeFunctionTool) RawJSON() string { return r.JSON.raw }
func (r *RealtimeFunctionTool) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this RealtimeFunctionTool to a RealtimeFunctionToolParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// RealtimeFunctionToolParam.Overrides()
func (r RealtimeFunctionTool) ToParam() RealtimeFunctionToolParam {
	return param.Override[RealtimeFunctionToolParam](json.RawMessage(r.RawJSON()))
}

// The type of the tool, i.e. `function`.
type RealtimeFunctionToolType string

const (
	RealtimeFunctionToolTypeFunction RealtimeFunctionToolType = "function"
)

type RealtimeFunctionToolParam struct {
	// The description of the function, including guidance on when and how to call it,
	// and guidance about what to tell the user when calling (if anything).
	Description param.Opt[string] `json:"description,omitzero"`
	// The name of the function.
	Name param.Opt[string] `json:"name,omitzero"`
	// Parameters of the function in JSON Schema.
	Parameters any `json:"parameters,omitzero"`
	// The type of the tool, i.e. `function`.
	//
	// Any of "function".
	Type RealtimeFunctionToolType `json:"type,omitzero"`
	paramObj
}

func (r RealtimeFunctionToolParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeFunctionToolParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeFunctionToolParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Realtime session object configuration.
//
// The property Type is required.
type RealtimeSessionCreateRequestParam struct {
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
	Instructions param.Opt[string] `json:"instructions,omitzero"`
	// Reference to a prompt template and its variables.
	// [Learn more](https://platform.openai.com/docs/guides/text?api-mode=responses#reusable-prompts).
	Prompt responses.ResponsePromptParam `json:"prompt,omitzero"`
	// Realtime API can write session traces to the
	// [Traces Dashboard](/logs?api=traces). Set to null to disable tracing. Once
	// tracing is enabled for a session, the configuration cannot be modified.
	//
	// `auto` will create a trace for the session with default values for the workflow
	// name, group id, and metadata.
	Tracing RealtimeTracingConfigUnionParam `json:"tracing,omitzero"`
	// Configuration for input and output audio.
	Audio RealtimeAudioConfigParam `json:"audio,omitzero"`
	// Additional fields to include in server outputs.
	//
	// `item.input_audio_transcription.logprobs`: Include logprobs for input audio
	// transcription.
	//
	// Any of "item.input_audio_transcription.logprobs".
	Include []string `json:"include,omitzero"`
	// Maximum number of output tokens for a single assistant response, inclusive of
	// tool calls. Provide an integer between 1 and 4096 to limit output tokens, or
	// `inf` for the maximum available tokens for a given model. Defaults to `inf`.
	MaxOutputTokens RealtimeSessionCreateRequestMaxOutputTokensUnionParam `json:"max_output_tokens,omitzero"`
	// The Realtime model used for this session.
	Model RealtimeSessionCreateRequestModel `json:"model,omitzero"`
	// The set of modalities the model can respond with. It defaults to `["audio"]`,
	// indicating that the model will respond with audio plus a transcript. `["text"]`
	// can be used to make the model respond with text only. It is not possible to
	// request both `text` and `audio` at the same time.
	//
	// Any of "text", "audio".
	OutputModalities []string `json:"output_modalities,omitzero"`
	// How the model chooses tools. Provide one of the string modes or force a specific
	// function/MCP tool.
	ToolChoice RealtimeToolChoiceConfigUnionParam `json:"tool_choice,omitzero"`
	// Tools available to the model.
	Tools RealtimeToolsConfigParam `json:"tools,omitzero"`
	// Controls how the realtime conversation is truncated prior to model inference.
	// The default is `auto`.
	Truncation RealtimeTruncationUnionParam `json:"truncation,omitzero"`
	// The type of session to create. Always `realtime` for the Realtime API.
	//
	// This field can be elided, and will marshal its zero value as "realtime".
	Type constant.Realtime `json:"type,required"`
	paramObj
}

func (r RealtimeSessionCreateRequestParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeSessionCreateRequestParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeSessionCreateRequestParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeSessionCreateRequestMaxOutputTokensUnionParam struct {
	OfInt param.Opt[int64] `json:",omitzero,inline"`
	// Construct this variant with constant.ValueOf[constant.Inf]()
	OfInf constant.Inf `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeSessionCreateRequestMaxOutputTokensUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfInt, u.OfInf)
}
func (u *RealtimeSessionCreateRequestMaxOutputTokensUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeSessionCreateRequestMaxOutputTokensUnionParam) asAny() any {
	if !param.IsOmitted(u.OfInt) {
		return &u.OfInt.Value
	} else if !param.IsOmitted(u.OfInf) {
		return &u.OfInf
	}
	return nil
}

// The Realtime model used for this session.
type RealtimeSessionCreateRequestModel = string

const (
	RealtimeSessionCreateRequestModelGPTRealtime                        RealtimeSessionCreateRequestModel = "gpt-realtime"
	RealtimeSessionCreateRequestModelGPTRealtime2025_08_28              RealtimeSessionCreateRequestModel = "gpt-realtime-2025-08-28"
	RealtimeSessionCreateRequestModelGPT4oRealtimePreview               RealtimeSessionCreateRequestModel = "gpt-4o-realtime-preview"
	RealtimeSessionCreateRequestModelGPT4oRealtimePreview2024_10_01     RealtimeSessionCreateRequestModel = "gpt-4o-realtime-preview-2024-10-01"
	RealtimeSessionCreateRequestModelGPT4oRealtimePreview2024_12_17     RealtimeSessionCreateRequestModel = "gpt-4o-realtime-preview-2024-12-17"
	RealtimeSessionCreateRequestModelGPT4oRealtimePreview2025_06_03     RealtimeSessionCreateRequestModel = "gpt-4o-realtime-preview-2025-06-03"
	RealtimeSessionCreateRequestModelGPT4oMiniRealtimePreview           RealtimeSessionCreateRequestModel = "gpt-4o-mini-realtime-preview"
	RealtimeSessionCreateRequestModelGPT4oMiniRealtimePreview2024_12_17 RealtimeSessionCreateRequestModel = "gpt-4o-mini-realtime-preview-2024-12-17"
)

func RealtimeToolChoiceConfigParamOfFunctionTool(name string) RealtimeToolChoiceConfigUnionParam {
	var variant responses.ToolChoiceFunctionParam
	variant.Name = name
	return RealtimeToolChoiceConfigUnionParam{OfFunctionTool: &variant}
}

func RealtimeToolChoiceConfigParamOfMcpTool(serverLabel string) RealtimeToolChoiceConfigUnionParam {
	var variant responses.ToolChoiceMcpParam
	variant.ServerLabel = serverLabel
	return RealtimeToolChoiceConfigUnionParam{OfMcpTool: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeToolChoiceConfigUnionParam struct {
	// Check if union is this variant with !param.IsOmitted(union.OfToolChoiceMode)
	OfToolChoiceMode param.Opt[responses.ToolChoiceOptions] `json:",omitzero,inline"`
	OfFunctionTool   *responses.ToolChoiceFunctionParam     `json:",omitzero,inline"`
	OfMcpTool        *responses.ToolChoiceMcpParam          `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeToolChoiceConfigUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfToolChoiceMode, u.OfFunctionTool, u.OfMcpTool)
}
func (u *RealtimeToolChoiceConfigUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeToolChoiceConfigUnionParam) asAny() any {
	if !param.IsOmitted(u.OfToolChoiceMode) {
		return &u.OfToolChoiceMode
	} else if !param.IsOmitted(u.OfFunctionTool) {
		return u.OfFunctionTool
	} else if !param.IsOmitted(u.OfMcpTool) {
		return u.OfMcpTool
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolChoiceConfigUnionParam) GetServerLabel() *string {
	if vt := u.OfMcpTool; vt != nil {
		return &vt.ServerLabel
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolChoiceConfigUnionParam) GetName() *string {
	if vt := u.OfFunctionTool; vt != nil {
		return (*string)(&vt.Name)
	} else if vt := u.OfMcpTool; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolChoiceConfigUnionParam) GetType() *string {
	if vt := u.OfFunctionTool; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMcpTool; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

type RealtimeToolsConfigParam []RealtimeToolsConfigUnionParam

func RealtimeToolsConfigUnionParamOfMcp(serverLabel string) RealtimeToolsConfigUnionParam {
	var mcp RealtimeToolsConfigUnionMcpParam
	mcp.ServerLabel = serverLabel
	return RealtimeToolsConfigUnionParam{OfMcp: &mcp}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeToolsConfigUnionParam struct {
	OfFunction *RealtimeFunctionToolParam        `json:",omitzero,inline"`
	OfMcp      *RealtimeToolsConfigUnionMcpParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeToolsConfigUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfFunction, u.OfMcp)
}
func (u *RealtimeToolsConfigUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeToolsConfigUnionParam) asAny() any {
	if !param.IsOmitted(u.OfFunction) {
		return u.OfFunction
	} else if !param.IsOmitted(u.OfMcp) {
		return u.OfMcp
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetDescription() *string {
	if vt := u.OfFunction; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetName() *string {
	if vt := u.OfFunction; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetParameters() *any {
	if vt := u.OfFunction; vt != nil {
		return &vt.Parameters
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetServerLabel() *string {
	if vt := u.OfMcp; vt != nil {
		return &vt.ServerLabel
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetAllowedTools() *RealtimeToolsConfigUnionMcpAllowedToolsParam {
	if vt := u.OfMcp; vt != nil {
		return &vt.AllowedTools
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetAuthorization() *string {
	if vt := u.OfMcp; vt != nil && vt.Authorization.Valid() {
		return &vt.Authorization.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetConnectorID() *string {
	if vt := u.OfMcp; vt != nil {
		return &vt.ConnectorID
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetHeaders() map[string]string {
	if vt := u.OfMcp; vt != nil {
		return vt.Headers
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetRequireApproval() *RealtimeToolsConfigUnionMcpRequireApprovalParam {
	if vt := u.OfMcp; vt != nil {
		return &vt.RequireApproval
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetServerDescription() *string {
	if vt := u.OfMcp; vt != nil && vt.ServerDescription.Valid() {
		return &vt.ServerDescription.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetServerURL() *string {
	if vt := u.OfMcp; vt != nil && vt.ServerURL.Valid() {
		return &vt.ServerURL.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeToolsConfigUnionParam) GetType() *string {
	if vt := u.OfFunction; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfMcp; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

func init() {
	apijson.RegisterUnion[RealtimeToolsConfigUnionParam](
		"type",
		apijson.Discriminator[RealtimeFunctionToolParam]("function"),
		apijson.Discriminator[RealtimeToolsConfigUnionMcpParam]("mcp"),
	)
}

// Give the model access to additional tools via remote Model Context Protocol
// (MCP) servers.
// [Learn more about MCP](https://platform.openai.com/docs/guides/tools-remote-mcp).
//
// The properties ServerLabel, Type are required.
type RealtimeToolsConfigUnionMcpParam struct {
	// A label for this MCP server, used to identify it in tool calls.
	ServerLabel string `json:"server_label,required"`
	// An OAuth access token that can be used with a remote MCP server, either with a
	// custom MCP server URL or a service connector. Your application must handle the
	// OAuth authorization flow and provide the token here.
	Authorization param.Opt[string] `json:"authorization,omitzero"`
	// Optional description of the MCP server, used to provide more context.
	ServerDescription param.Opt[string] `json:"server_description,omitzero"`
	// The URL for the MCP server. One of `server_url` or `connector_id` must be
	// provided.
	ServerURL param.Opt[string] `json:"server_url,omitzero"`
	// List of allowed tool names or a filter object.
	AllowedTools RealtimeToolsConfigUnionMcpAllowedToolsParam `json:"allowed_tools,omitzero"`
	// Optional HTTP headers to send to the MCP server. Use for authentication or other
	// purposes.
	Headers map[string]string `json:"headers,omitzero"`
	// Specify which of the MCP server's tools require approval.
	RequireApproval RealtimeToolsConfigUnionMcpRequireApprovalParam `json:"require_approval,omitzero"`
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
	ConnectorID string `json:"connector_id,omitzero"`
	// The type of the MCP tool. Always `mcp`.
	//
	// This field can be elided, and will marshal its zero value as "mcp".
	Type constant.Mcp `json:"type,required"`
	paramObj
}

func (r RealtimeToolsConfigUnionMcpParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeToolsConfigUnionMcpParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeToolsConfigUnionMcpParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeToolsConfigUnionMcpParam](
		"connector_id", "connector_dropbox", "connector_gmail", "connector_googlecalendar", "connector_googledrive", "connector_microsoftteams", "connector_outlookcalendar", "connector_outlookemail", "connector_sharepoint",
	)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeToolsConfigUnionMcpAllowedToolsParam struct {
	OfMcpAllowedTools []string                                                   `json:",omitzero,inline"`
	OfMcpToolFilter   *RealtimeToolsConfigUnionMcpAllowedToolsMcpToolFilterParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeToolsConfigUnionMcpAllowedToolsParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfMcpAllowedTools, u.OfMcpToolFilter)
}
func (u *RealtimeToolsConfigUnionMcpAllowedToolsParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeToolsConfigUnionMcpAllowedToolsParam) asAny() any {
	if !param.IsOmitted(u.OfMcpAllowedTools) {
		return &u.OfMcpAllowedTools
	} else if !param.IsOmitted(u.OfMcpToolFilter) {
		return u.OfMcpToolFilter
	}
	return nil
}

// A filter object to specify which tools are allowed.
type RealtimeToolsConfigUnionMcpAllowedToolsMcpToolFilterParam struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly param.Opt[bool] `json:"read_only,omitzero"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names,omitzero"`
	paramObj
}

func (r RealtimeToolsConfigUnionMcpAllowedToolsMcpToolFilterParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeToolsConfigUnionMcpAllowedToolsMcpToolFilterParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeToolsConfigUnionMcpAllowedToolsMcpToolFilterParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeToolsConfigUnionMcpRequireApprovalParam struct {
	OfMcpToolApprovalFilter *RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterParam `json:",omitzero,inline"`
	// Check if union is this variant with
	// !param.IsOmitted(union.OfMcpToolApprovalSetting)
	OfMcpToolApprovalSetting param.Opt[string] `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeToolsConfigUnionMcpRequireApprovalParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfMcpToolApprovalFilter, u.OfMcpToolApprovalSetting)
}
func (u *RealtimeToolsConfigUnionMcpRequireApprovalParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeToolsConfigUnionMcpRequireApprovalParam) asAny() any {
	if !param.IsOmitted(u.OfMcpToolApprovalFilter) {
		return u.OfMcpToolApprovalFilter
	} else if !param.IsOmitted(u.OfMcpToolApprovalSetting) {
		return &u.OfMcpToolApprovalSetting
	}
	return nil
}

// Specify which of the MCP server's tools require approval. Can be `always`,
// `never`, or a filter object associated with tools that require approval.
type RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterParam struct {
	// A filter object to specify which tools are allowed.
	Always RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterAlwaysParam `json:"always,omitzero"`
	// A filter object to specify which tools are allowed.
	Never RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterNeverParam `json:"never,omitzero"`
	paramObj
}

func (r RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A filter object to specify which tools are allowed.
type RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterAlwaysParam struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly param.Opt[bool] `json:"read_only,omitzero"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names,omitzero"`
	paramObj
}

func (r RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterAlwaysParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterAlwaysParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterAlwaysParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A filter object to specify which tools are allowed.
type RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterNeverParam struct {
	// Indicates whether or not a tool modifies data or is read-only. If an MCP server
	// is
	// [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint),
	// it will match this filter.
	ReadOnly param.Opt[bool] `json:"read_only,omitzero"`
	// List of allowed tool names.
	ToolNames []string `json:"tool_names,omitzero"`
	paramObj
}

func (r RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterNeverParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterNeverParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalFilterNeverParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Specify a single approval policy for all tools. One of `always` or `never`. When
// set to `always`, all tools will require approval. When set to `never`, all tools
// will not require approval.
type RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalSetting string

const (
	RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalSettingAlways RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalSetting = "always"
	RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalSettingNever  RealtimeToolsConfigUnionMcpRequireApprovalMcpToolApprovalSetting = "never"
)

func RealtimeTracingConfigParamOfAuto() RealtimeTracingConfigUnionParam {
	return RealtimeTracingConfigUnionParam{OfAuto: constant.ValueOf[constant.Auto]()}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeTracingConfigUnionParam struct {
	// Construct this variant with constant.ValueOf[constant.Auto]()
	OfAuto                 constant.Auto                                   `json:",omitzero,inline"`
	OfTracingConfiguration *RealtimeTracingConfigTracingConfigurationParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeTracingConfigUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfAuto, u.OfTracingConfiguration)
}
func (u *RealtimeTracingConfigUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeTracingConfigUnionParam) asAny() any {
	if !param.IsOmitted(u.OfAuto) {
		return &u.OfAuto
	} else if !param.IsOmitted(u.OfTracingConfiguration) {
		return u.OfTracingConfiguration
	}
	return nil
}

// Granular configuration for tracing.
type RealtimeTracingConfigTracingConfigurationParam struct {
	// The group id to attach to this trace to enable filtering and grouping in the
	// Traces Dashboard.
	GroupID param.Opt[string] `json:"group_id,omitzero"`
	// The name of the workflow to attach to this trace. This is used to name the trace
	// in the Traces Dashboard.
	WorkflowName param.Opt[string] `json:"workflow_name,omitzero"`
	// The arbitrary metadata to attach to this trace to enable filtering in the Traces
	// Dashboard.
	Metadata any `json:"metadata,omitzero"`
	paramObj
}

func (r RealtimeTracingConfigTracingConfigurationParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTracingConfigTracingConfigurationParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTracingConfigTracingConfigurationParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input and output audio.
type RealtimeTranscriptionSessionAudioParam struct {
	Input RealtimeTranscriptionSessionAudioInputParam `json:"input,omitzero"`
	paramObj
}

func (r RealtimeTranscriptionSessionAudioParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionAudioParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionAudioParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type RealtimeTranscriptionSessionAudioInputParam struct {
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
	TurnDetection RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam `json:"turn_detection,omitzero"`
	// The PCM audio format. Only a 24kHz sample rate is supported.
	Format RealtimeAudioFormatsUnionParam `json:"format,omitzero"`
	// Configuration for input audio noise reduction. This can be set to `null` to turn
	// off. Noise reduction filters audio added to the input audio buffer before it is
	// sent to VAD and the model. Filtering the audio can improve VAD and turn
	// detection accuracy (reducing false positives) and model performance by improving
	// perception of the input audio.
	NoiseReduction RealtimeTranscriptionSessionAudioInputNoiseReductionParam `json:"noise_reduction,omitzero"`
	// Configuration for input audio transcription, defaults to off and can be set to
	// `null` to turn off once on. Input audio transcription is not native to the
	// model, since the model consumes audio directly. Transcription runs
	// asynchronously through
	// [the /audio/transcriptions endpoint](https://platform.openai.com/docs/api-reference/audio/createTranscription)
	// and should be treated as guidance of input audio content rather than precisely
	// what the model heard. The client can optionally set the language and prompt for
	// transcription, these offer additional guidance to the transcription service.
	Transcription AudioTranscriptionParam `json:"transcription,omitzero"`
	paramObj
}

func (r RealtimeTranscriptionSessionAudioInputParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionAudioInputParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionAudioInputParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Configuration for input audio noise reduction. This can be set to `null` to turn
// off. Noise reduction filters audio added to the input audio buffer before it is
// sent to VAD and the model. Filtering the audio can improve VAD and turn
// detection accuracy (reducing false positives) and model performance by improving
// perception of the input audio.
type RealtimeTranscriptionSessionAudioInputNoiseReductionParam struct {
	// Type of noise reduction. `near_field` is for close-talking microphones such as
	// headphones, `far_field` is for far-field microphones such as laptop or
	// conference room microphones.
	//
	// Any of "near_field", "far_field".
	Type NoiseReductionType `json:"type,omitzero"`
	paramObj
}

func (r RealtimeTranscriptionSessionAudioInputNoiseReductionParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionAudioInputNoiseReductionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionAudioInputNoiseReductionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam struct {
	OfServerVad   *RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam   `json:",omitzero,inline"`
	OfSemanticVad *RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfServerVad, u.OfSemanticVad)
}
func (u *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) asAny() any {
	if !param.IsOmitted(u.OfServerVad) {
		return u.OfServerVad
	} else if !param.IsOmitted(u.OfSemanticVad) {
		return u.OfSemanticVad
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetIdleTimeoutMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.IdleTimeoutMs.Valid() {
		return &vt.IdleTimeoutMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetPrefixPaddingMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.PrefixPaddingMs.Valid() {
		return &vt.PrefixPaddingMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetSilenceDurationMs() *int64 {
	if vt := u.OfServerVad; vt != nil && vt.SilenceDurationMs.Valid() {
		return &vt.SilenceDurationMs.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetThreshold() *float64 {
	if vt := u.OfServerVad; vt != nil && vt.Threshold.Valid() {
		return &vt.Threshold.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetEagerness() *string {
	if vt := u.OfSemanticVad; vt != nil {
		return &vt.Eagerness
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetType() *string {
	if vt := u.OfServerVad; vt != nil {
		return (*string)(&vt.Type)
	} else if vt := u.OfSemanticVad; vt != nil {
		return (*string)(&vt.Type)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetCreateResponse() *bool {
	if vt := u.OfServerVad; vt != nil && vt.CreateResponse.Valid() {
		return &vt.CreateResponse.Value
	} else if vt := u.OfSemanticVad; vt != nil && vt.CreateResponse.Valid() {
		return &vt.CreateResponse.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam) GetInterruptResponse() *bool {
	if vt := u.OfServerVad; vt != nil && vt.InterruptResponse.Valid() {
		return &vt.InterruptResponse.Value
	} else if vt := u.OfSemanticVad; vt != nil && vt.InterruptResponse.Valid() {
		return &vt.InterruptResponse.Value
	}
	return nil
}

func init() {
	apijson.RegisterUnion[RealtimeTranscriptionSessionAudioInputTurnDetectionUnionParam](
		"type",
		apijson.Discriminator[RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam]("server_vad"),
		apijson.Discriminator[RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam]("semantic_vad"),
	)
}

// Server-side voice activity detection (VAD) which flips on when user speech is
// detected and off after a period of silence.
//
// The property Type is required.
type RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam struct {
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
	IdleTimeoutMs param.Opt[int64] `json:"idle_timeout_ms,omitzero"`
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse param.Opt[bool] `json:"create_response,omitzero"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse param.Opt[bool] `json:"interrupt_response,omitzero"`
	// Used only for `server_vad` mode. Amount of audio to include before the VAD
	// detected speech (in milliseconds). Defaults to 300ms.
	PrefixPaddingMs param.Opt[int64] `json:"prefix_padding_ms,omitzero"`
	// Used only for `server_vad` mode. Duration of silence to detect speech stop (in
	// milliseconds). Defaults to 500ms. With shorter values the model will respond
	// more quickly, but may jump in on short pauses from the user.
	SilenceDurationMs param.Opt[int64] `json:"silence_duration_ms,omitzero"`
	// Used only for `server_vad` mode. Activation threshold for VAD (0.0 to 1.0), this
	// defaults to 0.5. A higher threshold will require louder audio to activate the
	// model, and thus might perform better in noisy environments.
	Threshold param.Opt[float64] `json:"threshold,omitzero"`
	// Type of turn detection, `server_vad` to turn on simple Server VAD.
	//
	// This field can be elided, and will marshal its zero value as "server_vad".
	Type constant.ServerVad `json:"type,required"`
	paramObj
}

func (r RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionAudioInputTurnDetectionServerVadParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Server-side semantic turn detection which uses a model to determine when the
// user has finished speaking.
//
// The property Type is required.
type RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam struct {
	// Whether or not to automatically generate a response when a VAD stop event
	// occurs.
	CreateResponse param.Opt[bool] `json:"create_response,omitzero"`
	// Whether or not to automatically interrupt any ongoing response with output to
	// the default conversation (i.e. `conversation` of `auto`) when a VAD start event
	// occurs.
	InterruptResponse param.Opt[bool] `json:"interrupt_response,omitzero"`
	// Used only for `semantic_vad` mode. The eagerness of the model to respond. `low`
	// will wait longer for the user to continue speaking, `high` will respond more
	// quickly. `auto` is the default and is equivalent to `medium`. `low`, `medium`,
	// and `high` have max timeouts of 8s, 4s, and 2s respectively.
	//
	// Any of "low", "medium", "high", "auto".
	Eagerness string `json:"eagerness,omitzero"`
	// Type of turn detection, `semantic_vad` to turn on Semantic VAD.
	//
	// This field can be elided, and will marshal its zero value as "semantic_vad".
	Type constant.SemanticVad `json:"type,required"`
	paramObj
}

func (r RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

func init() {
	apijson.RegisterFieldValidator[RealtimeTranscriptionSessionAudioInputTurnDetectionSemanticVadParam](
		"eagerness", "low", "medium", "high", "auto",
	)
}

// Realtime transcription session object configuration.
//
// The property Type is required.
type RealtimeTranscriptionSessionCreateRequestParam struct {
	// Configuration for input and output audio.
	Audio RealtimeTranscriptionSessionAudioParam `json:"audio,omitzero"`
	// Additional fields to include in server outputs.
	//
	// `item.input_audio_transcription.logprobs`: Include logprobs for input audio
	// transcription.
	//
	// Any of "item.input_audio_transcription.logprobs".
	Include []string `json:"include,omitzero"`
	// The type of session to create. Always `transcription` for transcription
	// sessions.
	//
	// This field can be elided, and will marshal its zero value as "transcription".
	Type constant.Transcription `json:"type,required"`
	paramObj
}

func (r RealtimeTranscriptionSessionCreateRequestParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTranscriptionSessionCreateRequestParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTranscriptionSessionCreateRequestParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// RealtimeTruncationUnion contains all possible properties and values from
// [string], [RealtimeTruncationRetentionRatio].
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
//
// If the underlying value is not a json object, one of the following properties
// will be valid: OfRealtimeTruncationStrategy]
type RealtimeTruncationUnion struct {
	// This field will be present if the value is a [string] instead of an object.
	OfRealtimeTruncationStrategy string `json:",inline"`
	// This field is from variant [RealtimeTruncationRetentionRatio].
	RetentionRatio float64 `json:"retention_ratio"`
	// This field is from variant [RealtimeTruncationRetentionRatio].
	Type constant.RetentionRatio `json:"type"`
	JSON struct {
		OfRealtimeTruncationStrategy respjson.Field
		RetentionRatio               respjson.Field
		Type                         respjson.Field
		raw                          string
	} `json:"-"`
}

func (u RealtimeTruncationUnion) AsRealtimeTruncationStrategy() (v string) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u RealtimeTruncationUnion) AsRetentionRatioTruncation() (v RealtimeTruncationRetentionRatio) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u RealtimeTruncationUnion) RawJSON() string { return u.JSON.raw }

func (r *RealtimeTruncationUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this RealtimeTruncationUnion to a RealtimeTruncationUnionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// RealtimeTruncationUnionParam.Overrides()
func (r RealtimeTruncationUnion) ToParam() RealtimeTruncationUnionParam {
	return param.Override[RealtimeTruncationUnionParam](json.RawMessage(r.RawJSON()))
}

// The truncation strategy to use for the session. `auto` is the default truncation
// strategy. `disabled` will disable truncation and emit errors when the
// conversation exceeds the input token limit.
type RealtimeTruncationRealtimeTruncationStrategy string

const (
	RealtimeTruncationRealtimeTruncationStrategyAuto     RealtimeTruncationRealtimeTruncationStrategy = "auto"
	RealtimeTruncationRealtimeTruncationStrategyDisabled RealtimeTruncationRealtimeTruncationStrategy = "disabled"
)

func RealtimeTruncationParamOfRetentionRatioTruncation(retentionRatio float64) RealtimeTruncationUnionParam {
	var variant RealtimeTruncationRetentionRatioParam
	variant.RetentionRatio = retentionRatio
	return RealtimeTruncationUnionParam{OfRetentionRatioTruncation: &variant}
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type RealtimeTruncationUnionParam struct {
	// Check if union is this variant with
	// !param.IsOmitted(union.OfRealtimeTruncationStrategy)
	OfRealtimeTruncationStrategy param.Opt[string]                      `json:",omitzero,inline"`
	OfRetentionRatioTruncation   *RealtimeTruncationRetentionRatioParam `json:",omitzero,inline"`
	paramUnion
}

func (u RealtimeTruncationUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfRealtimeTruncationStrategy, u.OfRetentionRatioTruncation)
}
func (u *RealtimeTruncationUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

func (u *RealtimeTruncationUnionParam) asAny() any {
	if !param.IsOmitted(u.OfRealtimeTruncationStrategy) {
		return &u.OfRealtimeTruncationStrategy
	} else if !param.IsOmitted(u.OfRetentionRatioTruncation) {
		return u.OfRetentionRatioTruncation
	}
	return nil
}

// Retain a fraction of the conversation tokens when the conversation exceeds the
// input token limit. This allows you to amortize truncations across multiple
// turns, which can help improve cached token usage.
type RealtimeTruncationRetentionRatio struct {
	// Fraction of post-instruction conversation tokens to retain (0.0 - 1.0) when the
	// conversation exceeds the input token limit.
	RetentionRatio float64 `json:"retention_ratio,required"`
	// Use retention ratio truncation.
	Type constant.RetentionRatio `json:"type,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		RetentionRatio respjson.Field
		Type           respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r RealtimeTruncationRetentionRatio) RawJSON() string { return r.JSON.raw }
func (r *RealtimeTruncationRetentionRatio) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this RealtimeTruncationRetentionRatio to a
// RealtimeTruncationRetentionRatioParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// RealtimeTruncationRetentionRatioParam.Overrides()
func (r RealtimeTruncationRetentionRatio) ToParam() RealtimeTruncationRetentionRatioParam {
	return param.Override[RealtimeTruncationRetentionRatioParam](json.RawMessage(r.RawJSON()))
}

// Retain a fraction of the conversation tokens when the conversation exceeds the
// input token limit. This allows you to amortize truncations across multiple
// turns, which can help improve cached token usage.
//
// The properties RetentionRatio, Type are required.
type RealtimeTruncationRetentionRatioParam struct {
	// Fraction of post-instruction conversation tokens to retain (0.0 - 1.0) when the
	// conversation exceeds the input token limit.
	RetentionRatio float64 `json:"retention_ratio,required"`
	// Use retention ratio truncation.
	//
	// This field can be elided, and will marshal its zero value as "retention_ratio".
	Type constant.RetentionRatio `json:"type,required"`
	paramObj
}

func (r RealtimeTruncationRetentionRatioParam) MarshalJSON() (data []byte, err error) {
	type shadow RealtimeTruncationRetentionRatioParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *RealtimeTruncationRetentionRatioParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
