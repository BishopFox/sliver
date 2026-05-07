package plivo

import (
	"log"
	"net/http"
)

type ListMPCMeta struct {
	Previous   *string
	Next       *string
	TotalCount int64 `json:"count"`
	Offset     int64
	Limit      int64
}

type MultiPartyCallService struct {
	client *Client
}

type MPCUpdateResponse struct {
	CoachMode string `json:"coach_mode,omitempty" url:"coach_mode,omitempty"`
	Mute      string `json:"mute,omitempty" url:"mute,omitempty"`
	Hold      string `json:"hold,omitempty" url:"hold,omitempty"`
}

type MultiPartyCallAddParticipantParams struct {
	Role                           string      `json:"role,omitempty" url:"role,omitempty"`
	From                           string      `json:"from,omitempty" url:"from,omitempty"`
	To                             string      `json:"to,omitempty" url:"to,omitempty"`
	CallUuid                       string      `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	CallerName                     string      `json:"caller_name,omitempty" url:"caller_name,omitempty"`
	CallStatusCallbackUrl          string      `json:"call_status_callback_url,omitempty" url:"call_status_callback_url,omitempty"`
	CallStatusCallbackMethod       string      `json:"call_status_callback_method,omitempty" url:"call_status_callback_method,omitempty"`
	SipHeaders                     string      `json:"sip_headers,omitempty" url:"sip_headers,omitempty"`
	ConfirmKey                     string      `json:"confirm_key,omitempty" url:"confirm_key,omitempty"`
	ConfirmKeySoundUrl             string      `json:"confirm_key_sound_url,omitempty" url:"confirm_key_sound_url,omitempty"`
	ConfirmKeySoundMethod          string      `json:"confirm_key_sound_method,omitempty" url:"confirm_key_sound_method,omitempty"`
	DialMusic                      string      `json:"dial_music,omitempty" url:"dial_music,omitempty"`
	RingTimeout                    interface{} `json:"ring_timeout,omitempty" url:"ring_timeout,omitempty"`
	DelayDial                      interface{} `json:"delay_dial,omitempty" uril:"caller_name,omitempty"`
	MaxDuration                    int64       `json:"max_duration,omitempty" url:"max_duration,omitempty"`
	MaxParticipants                int64       `json:"max_participants,omitempty" url:"max_participants,omitempty"`
	WaitMusicUrl                   string      `json:"wait_music_url,omitempty" url:"wait_music_url,omitempty"`
	WaitMusicMethod                string      `json:"wait_music_method,omitempty" url:"wait_music_method,omitempty"`
	AgentHoldMusicUrl              string      `json:"agent_hold_music_url,omitempty" url:"agent_hold_music_url,omitempty"`
	AgentHoldMusicMethod           string      `json:"agent_hold_music_method,omitempty" url:"agent_hold_music_method,omitempty"`
	CustomerHoldMusicUrl           string      `json:"customer_hold_music_url,omitempty" url:"customer_hold_music_url,omitempty"`
	CustomerHoldMusicMethod        string      `json:"customer_hold_music_method,omitempty" url:"customer_hold_music_method,omitempty"`
	RecordingCallbackUrl           string      `json:"recording_callback_url,omitempty" url:"recording_callback_url,omitempty"`
	RecordingCallbackMethod        string      `json:"recording_callback_method,omitempty" url:"recording_callback_method,omitempty"`
	StatusCallbackUrl              string      `json:"status_callback_url,omitempty" url:"status_callback_url,omitempty"`
	StatusCallbackMethod           string      `json:"status_callback_method,omitempty" url:"status_callback_method,omitempty"`
	OnExitActionUrl                string      `json:"on_exit_action_url,omitempty" url:"on_exit_action_url,omitempty"`
	OnExitActionMethod             string      `json:"on_exit_action_method,omitempty" url:"on_exit_action_method,omitempty"`
	Record                         bool        `json:"record,omitempty" url:"record,omitempty"`
	RecordFileFormat               string      `json:"record_file_format,omitempty" url:"record_file_format,omitempty"`
	StatusCallbackEvents           string      `json:"status_callback_events,omitempty" url:"status_callback_events,omitempty"`
	StayAlone                      bool        `json:"stay_alone,omitempty" url:"stay_alone,omitempty"`
	CoachMode                      bool        `json:"coach_mode,omitempty" url:"coach_mode,omitempty"`
	Mute                           bool        `json:"mute,omitempty" url:"mute,omitempty"`
	Hold                           bool        `json:"hold,omitempty" url:"hold,omitempty"`
	StartMpcOnEnter                *bool       `json:"start_mpc_on_enter,omitempty" url:"start_mpc_on_enter,omitempty"`
	EndMpcOnExit                   bool        `json:"end_mpc_on_exit,omitempty" url:"end_mpc_on_exit,omitempty"`
	RelayDtmfInputs                bool        `json:"relay_dtmf_inputs,omitempty" url:"relay_dtmf_inputs,omitempty"`
	EnterSound                     string      `json:"enter_sound,omitempty" url:"enter_sound,omitempty"`
	EnterSoundMethod               string      `json:"enter_sound_method,omitempty" url:"enter_sound_method,omitempty"`
	ExitSound                      string      `json:"exit_sound,omitempty" url:"exit_sound,omitempty"`
	ExitSoundMethod                string      `json:"exit_sound_method,omitempty" url:"exit_sound_method,omitempty"`
	StartRecordingAudio            string      `json:"start_recording_audio,omitempty" url:"start_recording_audio,omitempty"`
	StartRecordingAudioMethod      string      `json:"start_recording_audio_method,omitempty" url:"start_recording_audio_method,omitempty"`
	StopRecordingAudio             string      `json:"stop_recording_audio,omitempty" url:"stop_recording_audio,omitempty"`
	StopRecordingAudioMethod       string      `json:"stop_recording_audio_method,omitempty" url:"stop_recording_audio_method,omitempty"`
	RecordMinMemberCount           int64       `json:"record_min_member_count,omitempty" url:"record_min_member_count,omitempty"`
	AgentHoldMusic                 string      `json:"agent_hold_music,omitempty" url:"agent_hold_music,omitempty"`
	CustomerHoldMusic              string      `json:"customer_hold_music,omitempty" url:"customer_hold_music,omitempty"`
	CreateMpcWithSingleParticipant *bool       `json:"create_mpc_with_single_participant,omitempty" url:"create_mpc_with_single_participant,omitempty"`
	SendDigits                     string      `json:"send_digits,omitempty" url:"send_digits,omitempty"`
	SendOnPreanswer                bool        `json:"send_on_preanswer,omitempty" url:"send_on_preanswer,omitempty"`
	TranscriptionUrl               string      `json:"transcription_url,omitempty" url:"transcription_url,omitempty"`
	Transcript                     bool        `json:"transcript,omitempty" url:"transcript,omitempty"`
	RecordParticipantTrack         bool        `json:"record_participant_track" url:"record_participant_track"`
}

type MultiPartyCallListParams struct {
	SubAccount           string `json:"sub_account,omitempty" url:"sub_account,omitempty"`
	FriendlyName         string `json:"friendly_name,omitempty" url:"friendly_name,omitempty"`
	Status               string `json:"status,omitempty" url:"status,omitempty"`
	TerminationCauseCode int64  `json:"termination_cause_code,omitempty" url:"termination_cause_code,omitempty"`
	EndTimeGt            string `json:"end_time__gt,omitempty" url:"end_time__gt,omitempty"`
	EndTimeGte           string `json:"end_time__gte,omitempty" url:"end_time__gte,omitempty"`
	EndTimeLt            string `json:"end_time__lt,omitempty" url:"end_time__lt,omitempty"`
	EndTimeLte           string `json:"end_time__lte,omitempty" url:"end_time__lte,omitempty"`
	CreationTimeGt       string `json:"creation_time__gt,omitempty" url:"creation_time__gt,omitempty"`
	CreationTimeGte      string `json:"creation_time__gte,omitempty" url:"creation_time__gte,omitempty"`
	CreationTimeLt       string `json:"creation_time__lt,omitempty" url:"creation_time__lt,omitempty"`
	CreationTimeLte      string `json:"creation_time__lte,omitempty" url:"creation_time__lte,omitempty"`
	Limit                int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset               int64  `json:"offset,omitempty" url:"offset,omitempty"`
}

type MultiPartyCallBasicParams struct {
	MpcUuid      string
	FriendlyName string
}

type MultiPartyCallListParticipantParams struct {
	CallUuid string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
}

type MultiPartyCallParticipantParams struct {
	MpcUuid       string
	FriendlyName  string
	ParticipantId string
}

type MultiPartyCallUpdateParticipantParams struct {
	CoachMode *bool `json:"coach_mode,omitempty" url:"coach_mode,omitempty"`
	Mute      *bool `json:"mute,omitempty" url:"mute,omitempty"`
	Hold      *bool `json:"hold,omitempty" url:"hold,omitempty"`
}

type MultiPartyCallStartRecordingParams struct {
	FileFormat              string `json:"file_format,omitempty" url:"file_format,omitempty"`
	RecordingCallbackUrl    string `json:"recording_callback_url,omitempty" url:"recording_callback_url,omitempty"`
	RecordingCallbackMethod string `json:"recording_callback_method,omitempty" url:"recording_callback_method,omitempty"`
	TranscriptionUrl        string `json:"transcription_url,omitempty" url:"transcription_url,omitempty"`
	Transcript              bool   `json:"transcript,omitempty" url:"transcript,omitempty"`
	RecordTrackType         string `json:"record_track_type" url:"record_track_type"`
}

type MultiPartyCallParticipantRecordingParams struct {
	RecordTrackType string `json:"record_track_type" url:"record_track_type"`
}

type MultiPartyCallListResponse struct {
	ApiID   string                       `json:"api_id" url:"api_id"`
	Meta    *ListMPCMeta                 `json:"meta" url:"meta"`
	Objects []*MultiPartyCallGetResponse `json:"objects" url:"objects"`
}

type MultiPartyCallGetResponse struct {
	BilledAmount         string `json:"billed_amount,omitempty" url:"billed_amount,omitempty"`
	BilledDuration       int64  `json:"billed_duration,omitempty" url:"billed_duration,omitempty"`
	CreationTime         string `json:"creation_time,omitempty" url:"creation_time,omitempty"`
	Duration             int64  `json:"duration,omitempty" url:"duration,omitempty"`
	EndTime              string `json:"end_time,omitempty" url:"end_time,omitempty"`
	FriendlyName         string `json:"friendly_name,omitempty" url:"friendly_name,omitempty"`
	MpcUuid              string `json:"mpc_uuid,omitempty" url:"mpc_uuid,omitempty"`
	Participants         string `json:"participants,omitempty" url:"participants,omitempty"`
	Recording            string `json:"recording,omitempty" url:"recording,omitempty"`
	ResourceUri          string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	StartTime            string `json:"start_time,omitempty" url:"start_time,omitempty"`
	Status               string `json:"status,omitempty" url:"status,omitempty"`
	StayAlone            bool   `json:"stay_alone,omitempty" url:"stay_alone,omitempty"`
	SubAccount           string `json:"sub_account,omitempty" url:"sub_account,omitempty"`
	TerminationCause     string `json:"termination_cause,omitempty" url:"termination_cause,omitempty"`
	TerminationCauseCode int64  `json:"termination_cause_code,omitempty" url:"termination_cause_code,omitempty"`
}

type CallAddParticipant struct {
	To       string `json:"to,omitempty" url:"to,omitempty"`
	From     string `json:"from,omitempty" url:"from,omitempty"`
	CallUuid string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
}

type MultiPartyCallAddParticipantResponse struct {
	ApiID       string                `json:"api_id" url:"api_id"`
	Calls       []*CallAddParticipant `json:"calls" url:"calls"`
	Message     string                `json:"message,omitempty" url:"message,omitempty"`
	RequestUuid string                `json:"request_uuid,omitempty" url:"request_uuid,omitempty"`
}

type MultiPartyCallStartRecordingResponse struct {
	ApiID        string `json:"api_id" url:"api_id"`
	Message      string `json:"message,omitempty" url:"message,omitempty"`
	RecordingId  string `json:"recording_id,omitempty" url:"recording_id,omitempty"`
	RecordingUrl string `json:"recording_url,omitempty" url:"recording_url,omitempty"`
}

type MultiPartyCallListParticipantsResponse struct {
	ApiID   string                       `json:"api_id" url:"api_id"`
	Meta    *ListMPCMeta                 `json:"meta" url:"meta"`
	Objects []*MultiPartyCallParticipant `json:"objects" url:"objects"`
}

type MultiPartyCallParticipant struct {
	BilledAmount    string `json:"billed_amount,omitempty" url:"billed_amount,omitempty"`
	BilledDuration  int64  `json:"billed_duration,omitempty" url:"billed_duration,omitempty"`
	CallUuid        string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	CoachMode       bool   `json:"coach_mode,omitempty" url:"coach_mode,omitempty"`
	Duration        int64  `json:"duration,omitempty" url:"duration,omitempty"`
	EndMpcOnExit    bool   `json:"end_mpc_on_exit,omitempty" url:"end_mpc_on_exit,omitempty"`
	ExitCause       string `json:"exit_cause,omitempty" url:"exit_cause,omitempty"`
	ExitTime        string `json:"exit_time,omitempty" url:"exit_time,omitempty"`
	Hold            bool   `json:"hold,omitempty" url:"hold,omitempty"`
	JoinTime        string `json:"join_time,omitempty" url:"join_time,omitempty"`
	MemberAddress   string `json:"member_address,omitempty" url:"member_address,omitempty"`
	MemberId        string `json:"member_id,omitempty" url:"member_id,omitempty"`
	MpcUuid         string `json:"mpc_uuid,omitempty" url:"mpc_uuid,omitempty"`
	Mute            bool   `json:"mute,omitempty" url:"mute,omitempty"`
	ResourceUri     string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	Role            string `json:"role,omitempty" url:"role,omitempty"`
	StartMpcOnEnter bool   `json:"start_mpc_on_enter,omitempty" url:"start_mpc_on_enter,omitempty"`
}

type MultiPartyCallUpdateParticipantResponse struct {
	ApiID string `json:"api_id" url:"api_id"`
	MPCUpdateResponse
}

type MultiPartyCallAudioParams struct {
	Url string `json:"url" url:"url"`
}

type MultiPartyCallAudioResponse struct {
	APIID        string   `json:"api_id" url:"api_id"`
	Message      string   `json:"message" url:"message"`
	MemberId     []string `json:"mpcMemberId,omitempty" url:"mpcMemberId,omitempty"`
	FriendlyName string   `json:"mpcName,omitempty" url:"mpcName,omitempty"`
}

type MultiPartyCallSpeakParams struct {
	Text           string `json:"text" url:"text"`
	Voice          string `json:"voice" url:"voice,omitempty"`
	Language       string `json:"language" url:"language,omitempty"`
	Mix            bool   `json:"mix" url:"mix,omitempty"`
	Type           string `json:"type,omitempty" url:"type,omitempty"`
	CallbackURL    string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
}

func (service *MultiPartyCallService) List(params MultiPartyCallListParams) (response *MultiPartyCallListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "MultiPartyCall")
	if err != nil {
		return
	}
	response = &MultiPartyCallListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) Get(basicParams MultiPartyCallBasicParams) (response *MultiPartyCallGetResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("GET", nil, "MultiPartyCall/%s", mpcId)
	if err != nil {
		return
	}
	response = &MultiPartyCallGetResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) AddParticipant(basicParams MultiPartyCallBasicParams, params MultiPartyCallAddParticipantParams) (response *MultiPartyCallAddParticipantResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	if (params.From != "" || params.To != "") && params.CallUuid != "" {
		log.Fatal("cannot specify call_uuid when (from, to) is provided")
	}
	if params.From == "" && params.To == "" && params.CallUuid == "" {
		log.Fatal("specify either call_uuid or (from, to)")
	}
	if params.CallUuid == "" && (params.From == "" || params.To == "") {
		log.Fatal("specify (from, to) when not adding an existing call_uuid to multi party participant")
	}
	if params.CallerName == "" {
		params.CallerName = params.From
	}
	if len(params.CallerName) > 50 {
		log.Fatal("CallerName length must be in range [0,50]")
	}
	if params.RingTimeout == nil {
		params.RingTimeout = 45
	} else {
		MultipleValidIntegers("RingTimeout", params.RingTimeout)
	}
	if params.DelayDial == nil {
		params.DelayDial = 0
	} else {
		MultipleValidIntegers("DelayDial", params.DelayDial)
	}
	req, err := service.client.NewRequest("POST", params, "MultiPartyCall/%s/Participant", mpcId)
	if err != nil {
		return
	}
	response = &MultiPartyCallAddParticipantResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) Start(basicParams MultiPartyCallBasicParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", map[string]string{"status": "active"}, "MultiPartyCall/%s", mpcId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) Stop(basicParams MultiPartyCallBasicParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s", mpcId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StartRecording(basicParams MultiPartyCallBasicParams, params MultiPartyCallStartRecordingParams) (response *MultiPartyCallStartRecordingResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", params, "MultiPartyCall/%s/Record", mpcId)
	if err != nil {
		return
	}
	response = &MultiPartyCallStartRecordingResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StopRecording(basicParams MultiPartyCallBasicParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s/Record", mpcId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) PauseRecording(basicParams MultiPartyCallBasicParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", nil, "MultiPartyCall/%s/Record/Pause", mpcId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) ResumeRecording(basicParams MultiPartyCallBasicParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", nil, "MultiPartyCall/%s/Record/Resume", mpcId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StartParticipantRecording(basicParams MultiPartyCallParticipantParams, params MultiPartyCallStartRecordingParams) (response *MultiPartyCallStartRecordingResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", params, "MultiPartyCall/%s/Participant/%s/Record", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	response = &MultiPartyCallStartRecordingResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StopParticipantRecording(basicParams MultiPartyCallParticipantParams, params ...MultiPartyCallParticipantRecordingParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	var req *http.Request
	if len(params) == 0 {
		req, err = service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s/Participant/%s/Record", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	} else {
		param := params[0]
		req, err = service.client.NewRequest("DELETE", param, "MultiPartyCall/%s/Participant/%s/Record", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) PauseParticipantRecording(basicParams MultiPartyCallParticipantParams, params ...MultiPartyCallParticipantRecordingParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	var req *http.Request
	if len(params) == 0 {
		req, err = service.client.NewRequest("POST", nil, "MultiPartyCall/%s/Participant/%s/Record/Pause", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	} else {
		param := params[0]
		req, err = service.client.NewRequest("POST", param, "MultiPartyCall/%s/Participant/%s/Record/Pause", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) ResumeParticipantRecording(basicParams MultiPartyCallParticipantParams, params ...MultiPartyCallParticipantRecordingParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	var req *http.Request
	if len(params) == 0 {
		req, err = service.client.NewRequest("POST", nil, "MultiPartyCall/%s/Participant/%s/Record/Resume", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	} else {
		param := params[0]
		req, err = service.client.NewRequest("POST", param, "MultiPartyCall/%s/Participant/%s/Record/Resume", mpcId, basicParams.ParticipantId)
		if err != nil {
			return
		}
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) ListParticipants(basicParams MultiPartyCallBasicParams, params MultiPartyCallListParticipantParams) (response *MultiPartyCallListParticipantsResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("GET", params, "MultiPartyCall/%s/Participant", mpcId)
	if err != nil {
		return
	}
	response = &MultiPartyCallListParticipantsResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) UpdateParticipant(basicParams MultiPartyCallParticipantParams, params MultiPartyCallUpdateParticipantParams) (response *MultiPartyCallUpdateParticipantResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", params, "MultiPartyCall/%s/Participant/%s", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	response = &MultiPartyCallUpdateParticipantResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) KickParticipant(basicParams MultiPartyCallParticipantParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s/Participant/%s", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) GetParticipant(basicParams MultiPartyCallParticipantParams) (response *MultiPartyCallParticipant, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("GET", nil, "MultiPartyCall/%s/Participant/%s", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	response = &MultiPartyCallParticipant{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
func (service *MultiPartyCallService) StartPlayAudio(basicParams MultiPartyCallParticipantParams, url MultiPartyCallAudioParams) (response *MultiPartyCallAudioResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", url, "MultiPartyCall/%s/Member/%s/Play", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	response = &MultiPartyCallAudioResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StopPlayAudio(basicParams MultiPartyCallParticipantParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s/Member/%s/Play", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StartSpeak(basicParams MultiPartyCallParticipantParams, params MultiPartyCallSpeakParams) (response *MultiPartyCallAudioResponse, err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("POST", params, "MultiPartyCall/%s/Member/%s/Speak", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	response = &MultiPartyCallAudioResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MultiPartyCallService) StopSpeak(basicParams MultiPartyCallParticipantParams) (err error) {
	mpcId := MakeMPCId(basicParams.MpcUuid, basicParams.FriendlyName)
	req, err := service.client.NewRequest("DELETE", nil, "MultiPartyCall/%s/Member/%s/Speak", mpcId, basicParams.ParticipantId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func MakeMPCId(MpcUuid string, FriendlyName string) string {
	mpcId := ""
	if MpcUuid != "" {
		mpcId = "uuid_" + MpcUuid
	} else if FriendlyName != "" {
		mpcId = "name_" + FriendlyName
	} else {
		log.Fatal("Need to specify a mpc_uuid or name")
	}
	return mpcId
}
