package plivo

type CallService struct {
	client *Client
}

type LiveCallService struct {
	client *Client
}

type QueuedCallService struct {
	client *Client
}

type Call struct {
	AnswerTime        string `json:"answer_time,omitempty" url:"answer_time,omitempty"`
	BillDuration      int64  `json:"bill_duration,omitempty" url:"bill_duration,omitempty"`
	BilledDuration    int64  `json:"billed_duration,omitempty" url:"billed_duration,omitempty"`
	CallDirection     string `json:"call_direction,omitempty" url:"call_direction,omitempty"`
	CallDuration      int64  `json:"call_duration,omitempty" url:"call_duration,omitempty"`
	CallState         string `json:"call_state,omitempty" url:"call_state,omitempty"`
	CallUUID          string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	ConferenceUUID    string `json:"conference_uuid,omitempty"`
	EndTime           string `json:"end_time,omitempty" url:"end_time,omitempty"`
	FromNumber        string `json:"from_number,omitempty" url:"from_number,omitempty"`
	HangupCauseCode   int64  `json:"hangup_cause_code,omitempty" url:"hangup_cause_code,omitempty"`
	HangupCauseName   string `json:"hangup_cause_name,omitempty" url:"hangup_cause_name,omitempty"`
	HangupSource      string `json:"hangup_source,omitempty" url:"hangup_source,omitempty"`
	InitiationTime    string `json:"initiation_time,omitempty" url:"initiation_time,omitempty"`
	ParentCallUUID    string `json:"parent_call_uuid,omitempty" url:"parent_call_uuid,omitempty"`
	ResourceURI       string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	ToNumber          string `json:"to_number,omitempty" url:"to_number,omitempty"`
	TotalAmount       string `json:"total_amount,omitempty" url:"total_amount,omitempty"`
	TotalRate         string `json:"total_rate,omitempty" url:"total_rate,omitempty"`
	StirVerification  string `json:"stir_verification,omitempty" url:"stir_verification,omitempty"`
	VoiceNetworkGroup string `json:"voice_network_group,omitempty" url:"voice_network_group,omitempty"`
	StirAttestation   string `json:"stir_attestation,omitempty" url:"stir_attestation,omitempty"`
	SourceIp          string `json:"source_ip,omitempty" url:"source_ip,omitempty"`
	CnamLookup        string `json:"cnam_lookup,omitempty" url:"cnam_lookup,omitempty"`
}

type LiveCall struct {
	From             string `json:"from,omitempty" url:"from,omitempty"`
	To               string `json:"to,omitempty" url:"to,omitempty"`
	AnswerURL        string `json:"answer_url,omitempty" url:"answer_url,omitempty"`
	CallUUID         string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	CallerName       string `json:"caller_name,omitempty" url:"caller_name,omitempty"`
	ParentCallUUID   string `json:"parent_call_uuid,omitempty" url:"parent_call_uuid,omitempty"`
	SessionStart     string `json:"session_start,omitempty" url:"session_start,omitempty"`
	CallStatus       string `json:"call_status,omitempty" url:"call_status,omitempty"`
	StirVerification string `json:"stir_verification,omitempty" url:"stir_verification,omitempty"`
	StirAttestation  string `json:"stir_attestation,omitempty" url:"stir_attestation,omitempty"`
}

type QueuedCall struct {
	From        string `json:"from,omitempty" url:"from,omitempty"`
	To          string `json:"to,omitempty" url:"to,omitempty"`
	Status      string `json:"call_status,omitempty" url:"call_status,omitempty"`
	CallUUID    string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	CallerName  string `json:"caller_name,omitempty" url:"caller_name,omitempty"`
	APIID       string `json:"api_id,omitempty" url:"api_id,omitempty"`
	Direction   string `json:"direction,omitempty" url:"direction,omitempty"`
	RequestUUID string `json:"request_uuid,omitempty" url:"request_uuid,omitempty"`
}

type LiveCallIDListResponse struct {
	APIID string   `json:"api_id" url:"api_id"`
	Calls []string `json:"calls" url:"calls"`
}

type QueuedCallIDListResponse struct {
	APIID string   `json:"api_id" url:"api_id"`
	Calls []string `json:"calls" url:"calls"`
}

type CallCreateParams struct {
	// Required parameters.
	From      string `json:"from,omitempty" url:"from,omitempty"`
	To        string `json:"to,omitempty" url:"to,omitempty"`
	AnswerURL string `json:"answer_url,omitempty" url:"answer_url,omitempty"`
	// Optional parameters.
	AnswerMethod           string `json:"answer_method,omitempty" url:"answer_method,omitempty"`
	RingURL                string `json:"ring_url,omitempty" url:"ring_url,omitempty"`
	RingMethod             string `json:"ring_method,omitempty" url:"ring_method,omitempty"`
	HangupURL              string `json:"hangup_url,omitempty" url:"hangup_url,omitempty"`
	HangupMethod           string `json:"hangup_method,omitempty" url:"hangup_method,omitempty"`
	FallbackURL            string `json:"fallback_url,omitempty" url:"fallback_url,omitempty"`
	FallbackMethod         string `json:"fallback_method,omitempty" url:"fallback_method,omitempty"`
	CallerName             string `json:"caller_name,omitempty" url:"caller_name,omitempty"`
	SendDigits             string `json:"send_digits,omitempty" url:"send_digits,omitempty"`
	SendOnPreanswer        bool   `json:"send_on_preanswer,omitempty" url:"send_on_preanswer,omitempty"`
	TimeLimit              int64  `json:"time_limit,omitempty" url:"time_limit,omitempty"`
	HangupOnRing           int64  `json:"hangup_on_ring,omitempty" url:"hangup_on_ring,omitempty"`
	MachineDetection       string `json:"machine_detection,omitempty" url:"machine_detection,omitempty"`
	MachineDetectionTime   int64  `json:"machine_detection_time,omitempty" url:"machine_detection_time,omitempty"`
	MachineDetectionUrl    string `json:"machine_detection_url,omitempty" url:"machine_detection_url,omitempty"`
	MachineDetectionMethod string `json:"machine_detection_method,omitempty" url:"machine_detection_method,omitempty"`
	SipHeaders             string `json:"sip_headers,omitempty" url:"sip_headers,omitempty"`
	RingTimeout            int64  `json:"ring_timeout,omitempty" url:"ring_timeout,omitempty"`
}

// Stores response for making a call.
type CallCreateResponse struct {
	Message     string      `json:"message" url:"message"`
	ApiID       string      `json:"api_id" url:"api_id"`
	RequestUUID interface{} `json:"request_uuid" url:"request_uuid"`
}

type CallListParams struct {
	// Query parameters.
	Subaccount      string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	CallDirection   string `json:"call_direction,omitempty" url:"call_direction,omitempty"`
	FromNumber      string `json:"from_number,omitempty" url:"from_number,omitempty"`
	ToNumber        string `json:"to_number,omitempty" url:"to_number,omitempty"`
	ParentCallUUID  string `json:"parent_call_uuid,omitempty" url:"parent_call_uuid,omitempty"`
	EndTimeEquals   string `json:"end_time,omitempty" url:"end_time,omitempty"`
	HangupCauseCode int64  `json:"hangup_cause_code,omitempty" url:"hangup_cause_code,omitempty"`
	HangupSource    string `json:"hangup_source,omitempty" url:"hangup_source,omitempty"`

	EndTimeLessThan string `json:"end_time__lt,omitempty" url:"end_time__lt,omitempty"`

	EndTimeGreaterThan string `json:"end_time__gt,omitempty" url:"end_time__gt,omitempty"`

	EndTimeLessOrEqual string `json:"end_time__lte,omitempty" url:"end_time__lte,omitempty"`

	EndTimeGreaterOrEqual string `json:"end_time__gte,omitempty" url:"end_time__gte,omitempty"`

	BillDurationEquals string `json:"bill_duration,omitempty" url:"bill_duration,omitempty"`

	BillDurationLessThan string `json:"bill_duration__lt,omitempty" url:"bill_duration__lt,omitempty"`

	BillDurationGreaterThan string `json:"bill_duration__gt,omitempty" url:"bill_duration__gt,omitempty"`

	BillDurationLessOrEqual string `json:"bill_duration__lte,omitempty" url:"bill_duration__lte,omitempty"`

	BillDurationGreaterOrEqual string `json:"bill_duration__gte,omitempty" url:"bill_duration__gte,omitempty"`
	Limit                      int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset                     int64  `json:"offset,omitempty" url:"offset,omitempty"`
}

type LiveCallFilters struct {
	CallDirection string `json:"call_direction,omitempty" url:"call_direction,omitempty"`
	FromNumber    string `json:"from_number,omitempty" url:"from_number,omitempty"`
	ToNumber      string `json:"to_number,omitempty" url:"to_number,omitempty"`
	Status        string `json:"status,omitempty" url:"status,omitempty" default:"live"`
}

type CallListResponse struct {
	ApiID   string  `json:"api_id" url:"api_id"`
	Meta    *Meta   `json:"meta" url:"meta"`
	Objects []*Call `json:"objects" url:"objects"`
}

type CallUpdateParams struct {
	Legs       string `json:"legs,omitempty" url:"legs,omitempty"`
	AlegURL    string `json:"aleg_url,omitempty" url:"aleg_url,omitempty"`
	AlegMethod string `json:"aleg_method,omitempty" url:"aleg_method,omitempty"`
	BlegURL    string `json:"bleg_url,omitempty" url:"bleg_url,omitempty"`
	BlegMethod string `json:"bleg_method,omitempty" url:"bleg_method,omitempty"`
}

type CallUpdateResponse struct {
	ApiID   string `json:"api_id" url:"api_id"`
	Message string `json:"message" url:"message"`
}

type CallRecordParams struct {
	TimeLimit           int64  `json:"time_limit,omitempty" url:"time_limit,omitempty"`
	FileFormat          string `json:"file_format,omitempty" url:"file_format,omitempty"`
	TranscriptionType   string `json:"transcription_type,omitempty" url:"transcription_type,omitempty"`
	TranscriptionURL    string `json:"transcription_url,omitempty" url:"transcription_url,omitempty"`
	TranscriptionMethod string `json:"transcription_method,omitempty" url:"transcription_method,omitempty"`
	CallbackURL         string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod      string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
	RecordChannelType   string `json:"record_channel_type,omitempty" url:"record_channel_type,omitempty"`
}

type CallRecordResponse struct {
	Message     string `json:"message,omitempty" url:"message,omitempty"`
	URL         string `json:"url,omitempty" url:"url,omitempty"`
	APIID       string `json:"api_id,omitempty" url:"api_id,omitempty"`
	RecordingID string `json:"recording_id,omitempty" url:"recording_id,omitempty"`
}

type CallStreamResponse struct {
	Message  string `json:"message,omitempty" url:"message,omitempty"`
	APIID    string `json:"api_id,omitempty" url:"api_id,omitempty"`
	StreamID string `json:"stream_id,omitempty" url:"stream_id,omitempty"`
}

type CallPlayParams struct {
	URLs   string `json:"urls" url:"urls"`
	Length string `json:"length,omitempty" url:"length,omitempty"`
	Legs   string `json:"legs,omitempty" url:"legs,omitempty"`
	Loop   bool   `json:"loop,omitempty" url:"loop,omitempty"`
	Mix    bool   `json:"mix,omitempty" url:"mix,omitempty"`
}

type CallPlayResponse struct {
	Message string `json:"message,omitempty" url:"message,omitempty"`
	ApiID   string `json:"api_id,omitempty" url:"api_id,omitempty"`
}

type CallSpeakParams struct {
	Text           string `json:"text" url:"text"`
	Voice          string `json:"length,omitempty" url:"length,omitempty"`
	Language       string `json:"language,omitempty" url:"language,omitempty"`
	Legs           string `json:"legs,omitempty" url:"legs,omitempty"`
	Loop           bool   `json:"loop,omitempty" url:"loop,omitempty"`
	Mix            bool   `json:"mix,omitempty" url:"mix,omitempty"`
	Type           string `json:"type,omitempty" url:"type,omitempty"`
	CallbackURL    string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
}

type CallSpeakResponse struct {
	Message string `json:"message,omitempty" url:"message,omitempty"`
	ApiID   string `json:"api_id,omitempty" url:"api_id,omitempty"`
}

type CallDTMFParams struct {
	Digits string `json:"digits" url:"digits"`
	Legs   string `json:"legs,omitempty" url:"legs,omitempty"`
}

type CallDTMFResponseBody struct {
	Message string `json:"message,omitempty" url:"message,omitempty"`
	ApiID   string `json:"api_id,omitempty" url:"api_id,omitempty"`
}

type CallStreamParams struct {
	ServiceUrl           string `json:"service_url,omitempty" url:"service_url,omitempty"`
	Bidirectional        bool   `json:"bidirectional,omitempty" url:"bidirectional,omitempty"`
	AudioTrack           string `json:"audio_track,omitempty" url:"audio_track,omitempty"`
	StreamTimeout        int64  `json:"stream_timeout,omitempty" url:"stream_timeout,omitempty"`
	StatusCallbackUrl    string `json:"status_callback_url,omitempty" url:"status_callback_url,omitempty"`
	StatusCallbackMethod string `json:"status_callback_method,omitempty" url:"status_callback_method,omitempty"`
	ContentType          string `json:"content_type,omitempty" url:"content_type,omitempty"`
	ExtraHeaders         string `json:"extra_headers,omitempty" url:"extra_headers,omitempty"`
	CxBot                bool   `json:"cx_bot,omitempty" url:"cx_bot,omitempty"`
}

type CallStreamGetAllObject struct {
	AudioTrack          string `json:"audio_track" url:"audio_track"`
	Bidirectional       bool   `json:"bidirectional" url:"bidirectional"`
	BilledAmount        string `json:"billed_amount" url:"billed_amount"`
	BillDuration        int64  `json:"bill_duration" url:"bill_duration"`
	CallUUID            string `json:"call_uuid" url:"call_uuid"`
	CreatedAt           string `json:"created_at" url:"created_at"`
	EndTime             string `json:"end_time" url:"end_time"`
	PlivoAuthId         string `json:"plivo_auth_id" url:"plivo_auth_id"`
	ResourceURI         string `json:"resource_uri" url:"resource_uri"`
	RoundedBillDuration string `json:"rounded_bill_duration" url:"rounded_bill_duration"`
	ServiceURL          string `json:"service_url" url:"service_url"`
	StartTime           string `json:"start_time" url:"start_time"`
	Status              string `json:"status" url:"status"`
	StatusCallbackURL   string `json:"status_callback_url" url:"status_callback_url"`
	StreamID            string `json:"stream_id" url:"stream_id"`
}

type CallStreamGetAll struct {
	ApiID   string                   `json:"api_id,omitempty" url:"api_id,omitempty"`
	Meta    Meta                     `json:"meta,omitempty" url:"meta,omitempty"`
	Objects []CallStreamGetAllObject `json:"objects" url:"objects"`
}

type CallStreamGetSpecific struct {
	ApiID               string `json:"api_id,omitempty" url:"api_id,omitempty"`
	AudioTrack          string `json:"audio_track" url:"audio_track"`
	Bidirectional       bool   `json:"bidirectional" url:"bidirectional"`
	BilledAmount        string `json:"billed_amount" url:"billed_amount"`
	BillDuration        int64  `json:"bill_duration" url:"bill_duration"`
	CallUUID            string `json:"call_uuid" url:"call_uuid"`
	CreatedAt           string `json:"created_at" url:"created_at"`
	EndTime             string `json:"end_time" url:"end_time"`
	PlivoAuthId         string `json:"plivo_auth_id" url:"plivo_auth_id"`
	ResourceURI         string `json:"resource_uri" url:"resource_uri"`
	RoundedBillDuration string `json:"rounded_bill_duration" url:"rounded_bill_duration"`
	ServiceURL          string `json:"service_url" url:"service_url"`
	StartTime           string `json:"start_time" url:"start_time"`
	Status              string `json:"status" url:"status"`
	StatusCallbackURL   string `json:"status_callback_url" url:"status_callback_url"`
	StreamID            string `json:"stream_id" url:"stream_id"`
}

func (service *CallService) List(params CallListParams) (response *CallListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "Call")
	if err != nil {
		return
	}
	response = &CallListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) Create(params CallCreateParams) (response *CallCreateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call")
	if err != nil {
		return
	}
	response = &CallCreateResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) Get(CallId string) (response *Call, err error) {
	req, err := service.client.NewRequest("GET", nil, "Call/%s", CallId)
	if err != nil {
		return
	}
	response = &Call{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) Delete(CallId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s", CallId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) Update(CallId string, params CallUpdateParams) (response *CallUpdateResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s", CallId)
	if err != nil {
		return
	}
	response = &CallUpdateResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *LiveCallService) Get(LiveCallId string) (response *LiveCall, err error) {
	req, err := service.client.NewRequest("GET", struct {
		Status string `json:"status" url:"status"`
	}{"live"}, "Call/%s", LiveCallId)
	if err != nil {
		return
	}
	response = &LiveCall{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *LiveCallService) IDList(data ...LiveCallFilters) (response *LiveCallIDListResponse, err error) {
	var optionalParams LiveCallFilters
	if data != nil {
		optionalParams = data[0]
	}
	optionalParams.Status = "live"
	req, err := service.client.NewRequest("GET", optionalParams, "Call")
	if err != nil {
		return
	}
	response = &LiveCallIDListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *QueuedCallService) IDList() (response *QueuedCallIDListResponse, err error) {
	req, err := service.client.NewRequest("GET", struct {
		Status string `json:"status" url:"status"`
	}{"queued"}, "Call")
	if err != nil {
		return
	}
	response = &QueuedCallIDListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *QueuedCallService) Get(QueuedCallId string) (response *QueuedCall, err error) {
	req, err := service.client.NewRequest("GET", struct {
		Status string `json:"status" url:"status"`
	}{"queued"}, "Call/%s", QueuedCallId)
	if err != nil {
		return
	}
	response = &QueuedCall{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) Record(callId string, params CallRecordParams) (response *CallRecordResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s/Record", callId)
	if err != nil {
		return
	}
	response = &CallRecordResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) StopRecording(callId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s/Record", callId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) Stream(CallId string, params CallStreamParams) (response *CallStreamResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s/Stream", CallId)
	if err != nil {
		return
	}
	response = &CallStreamResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) StopAllStreams(CallId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s/Stream", CallId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) StopSpecificStream(CallId string, StreamId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s/Stream/%s", CallId, StreamId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) GetAllStreams(CallId string) (response *CallStreamGetAll, err error) {
	req, err := service.client.NewRequest("GET", nil, "Call/%s/Stream", CallId)
	if err != nil {
		return
	}
	response = &CallStreamGetAll{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) GetSpecificStream(CallId string, StreamId string) (response *CallStreamGetSpecific, err error) {
	req, err := service.client.NewRequest("GET", nil, "Call/%s/Stream/%s", CallId, StreamId)
	if err != nil {
		return
	}
	response = &CallStreamGetSpecific{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) Speak(callId string, params CallSpeakParams) (response *CallSpeakResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s/Speak", callId)
	if err != nil {
		return
	}
	response = &CallSpeakResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) StopSpeaking(callId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s/Speak", callId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) Play(callId string, params CallPlayParams) (response *CallPlayResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s/Play", callId)
	if err != nil {
		return
	}
	response = &CallPlayResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) StopPlaying(callId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Call/%s/Play", callId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *CallService) SendDigits(callId string, params CallDTMFParams) (response *CallDTMFResponseBody, err error) {
	req, err := service.client.NewRequest("POST", params, "Call/%s/DTMF", callId)
	if err != nil {
		return
	}
	response = &CallDTMFResponseBody{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *CallService) CancelRequest(requestId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Request/%s", requestId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}
