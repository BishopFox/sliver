package plivo

type MaskingSessionService struct {
	client *Client
}

type MaskingSession struct {
	FirstParty                   string                     `json:"first_party,omitempty" url:"first_party,omitempty"`
	SecondParty                  string                     `json:"second_party,omitempty" url:"second_party,omitempty"`
	VirtualNumber                string                     `json:"virtual_number,omitempty" url:"virtual_number,omitempty"`
	Status                       string                     `json:"status,omitempty" url:"status,omitempty"`
	InitiateCallToFirstParty     bool                       `json:"initiate_call_to_first_party,omitempty" url:"initiate_call_to_first_party,omitempty"`
	SessionUUID                  string                     `json:"session_uuid,omitempty" url:"session_uuid,omitempty"`
	CallbackUrl                  string                     `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod               string                     `json:"callback_method,omitempty" url:"callback_method,omitempty"`
	CreatedAt                    string                     `json:"created_time,omitempty" url:"created_time,omitempty"`
	UpdatedAt                    string                     `json:"modified_time,omitempty" url:"updated_at,omitempty"`
	ExpiryAt                     string                     `json:"expiry_time,omitempty" url:"expiry_time,omitempty"`
	Duration                     int64                      `json:"duration,omitempty" url:"duration,omitempty"`
	SessionCreationAmount        int64                      `json:"amount" url:"amount"`
	CallTimeLimit                int64                      `json:"call_time_limit,omitempty" url:"call_time_limit,omitempty"`
	RingTimeout                  int64                      `json:"ring_timeout,omitempty" url:"ring_timeout,omitempty"`
	FirstPartyPlayUrl            string                     `json:"first_party_play_url,omitempty" url:"first_party_play_url,omitempty"`
	SecondPartyPlayUrl           string                     `json:"second_party_play_url,omitempty" url:"second_party_play_url,omitempty"`
	Record                       bool                       `json:"record,omitempty" url:"record,omitempty"`
	RecordFileFormat             string                     `json:"record_file_format,omitempty" url:"record_file_format,omitempty"`
	RecordingCallbackUrl         string                     `json:"recording_callback_url,omitempty" url:"recording_callback_url,omitempty"`
	RecordingCallbackMethod      string                     `json:"recording_callback_method,omitempty" url:"recording_callback_method,omitempty"`
	Interaction                  []VoiceInteractionResponse `json:"interaction" url:"interaction"`
	TotalCallAmount              float64                    `json:"total_call_amount" url:"total_call_amount"`
	TotalCallCount               int                        `json:"total_call_count" url:"total_call_count"`
	TotalCallBilledDuration      int                        `json:"total_call_billed_duration" url:"total_call_billed_duration"`
	TotalSessionAmount           float64                    `json:"total_session_amount" url:"total_session_amount"`
	LastInteractionTime          string                     `json:"last_interaction_time" url:"last_interaction_time"`
	IsPinAuthenticationRequired  bool                       `json:"is_pin_authentication_required" url:"is_pin_authentication_required"`
	GeneratePin                  bool                       `json:"generate_pin" url:"generate_pin"`
	GeneratePinLength            int64                      `json:"generate_pin_length" url:"generate_pin_length"`
	FirstPartyPin                string                     `json:"first_party_pin" url:"first_party_pin"`
	SecondPartyPin               string                     `json:"second_party_pin" url:"second_party_pin"`
	PinPromptPlay                string                     `json:"pin_prompt_play" url:"pin_prompt_play"`
	PinRetry                     int64                      `json:"pin_retry" url:"pin_retry"`
	PinRetryWait                 int64                      `json:"pin_retry_wait" url:"pin_retry_wait"`
	IncorrectPinPlay             string                     `json:"incorrect_pin_play" url:"incorrect_pin_play"`
	UnknownCallerPlay            string                     `json:"unknown_caller_play" url:"unknown_caller_play"`
	VirtualNumberCooloffPeriod   int                        `json:"virtual_number_cooloff_period,omitempty" url:"virtual_number_cooloff_period,omitempty"`
	ForcePinAuthentication       bool                       `json:"force_pin_authentication" url:"force_pin_authentication"`
	CreateSessionWithSingleParty bool                       `json:"create_session_with_single_Party" url:"create_session_with_single_Party"`
}

type CreateMaskingSessionParams struct {
	FirstParty                   string `json:"first_party,omitempty" url:"first_party,omitempty"`
	SecondParty                  string `json:"second_party,omitempty" url:"second_party,omitempty"`
	SessionExpiry                int    `json:"session_expiry" url:"session_expiry,omitempty"`
	CallTimeLimit                int    `json:"call_time_limit,omitempty" url:"call_time_limit,omitempty"`
	Record                       bool   `json:"record,omitempty" url:"record,omitempty"`
	RecordFileFormat             string `json:"record_file_format,omitempty" url:"record_file_format,omitempty"`
	RecordingCallbackUrl         string `json:"recording_callback_url,omitempty" url:"recording_callback_url,omitempty"`
	InitiateCallToFirstParty     bool   `json:"initiate_call_to_first_party,omitempty" url:"initiate_call_to_first_party,omitempty"`
	CallbackUrl                  string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod               string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
	RingTimeout                  int64  `json:"ring_timeout,omitempty" url:"ring_timeout,omitempty"`
	FirstPartyPlayUrl            string `json:"first_party_play_url,omitempty" url:"first_party_play_url,omitempty"`
	SecondPartyPlayUrl           string `json:"second_party_play_url,omitempty" url:"second_party_play_url,omitempty"`
	RecordingCallbackMethod      string `json:"recording_callback_method,omitempty" url:"recording_callback_method,omitempty"`
	IsPinAuthenticationRequired  bool   `json:"is_pin_authentication_required,omitempty" url:"is_pin_authentication_required,omitempty"`
	GeneratePin                  bool   `json:"generate_pin,omitempty" url:"generate_pin,omitempty"`
	GeneratePinLength            int64  `json:"generate_pin_length,omitempty" url:"generate_pin_length,omitempty"`
	FirstPartyPin                string `json:"first_party_pin,omitempty" url:"first_party_pin,omitempty"`
	SecondPartyPin               string `json:"second_party_pin,omitempty" url:"second_party_pin,omitempty"`
	PinPromptPlay                string `json:"pin_prompt_play,omitempty" url:"pin_prompt_play,omitempty"`
	PinRetry                     int64  `json:"pin_retry,omitempty" url:"pin_retry,omitempty"`
	PinRetryWait                 int64  `json:"pin_retry_wait,omitempty" url:"pin_retry_wait,omitempty"`
	IncorrectPinPlay             string `json:"incorrect_pin_play,omitempty" url:"incorrect_pin_play,omitempty"`
	UnknownCallerPlay            string `json:"unknown_caller_play,omitempty" url:"unknown_caller_play,omitempty"`
	SubAccount                   string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	GeoMatch                     *bool  `json:"geomatch,omitempty" url:"geomatch,omitempty"`
	VirtualNumberCooloffPeriod   int    `json:"virtual_number_cooloff_period,omitempty" url:"virtual_number_cooloff_period,omitempty"`
	ForcePinAuthentication       bool   `json:"force_pin_authentication,omitempty" url:"force_pin_authentication,omitempty"`
	CreateSessionWithSingleParty bool   `json:"create_session_with_single_Party,omitempty" url:"create_session_with_single_Party,omitempty"`
}

type UpdateMaskingSessionParams struct {
	FirstParty                   string `json:"first_party,omitempty" url:"first_party,omitempty"`
	SecondParty                  string `json:"second_party,omitempty" url:"second_party,omitempty"`
	SessionExpiry                int64  `json:"session_expiry,omitempty" url:"session_expiry,omitempty"`
	CallTimeLimit                int64  `json:"call_time_limit,omitempty" url:"call_time_limit,omitempty"`
	Record                       bool   `json:"record,omitempty" url:"record,omitempty"`
	RecordFileFormat             string `json:"record_file_format,omitempty" url:"record_file_format,omitempty"`
	RecordingCallbackUrl         string `json:"recording_callback_url,omitempty" url:"recording_callback_url,omitempty"`
	CallbackUrl                  string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod               string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
	RingTimeout                  int64  `json:"ring_timeout,omitempty" url:"ring_timeout,omitempty"`
	FirstPartyPlayUrl            string `json:"first_party_play_url,omitempty" url:"first_party_play_url,omitempty"`
	SecondPartyPlayUrl           string `json:"second_party_play_url,omitempty" url:"second_party_play_url,omitempty"`
	RecordingCallbackMethod      string `json:"recording_callback_method,omitempty" url:"recording_callback_method,omitempty"`
	SubAccount                   string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	GeoMatch                     *bool  `json:"geomatch,omitempty" url:"geomatch,omitempty"`
	CreateSessionWithSingleParty bool   `json:"create_session_with_single_Party,omitempty" url:"create_session_with_single_Party,omitempty"`
}

type ListSessionFilterParams struct {
	// Query parameters.
	FirstParty                string `json:"first_party,omitempty" url:"first_party,omitempty"`
	SecondParty               string `json:"second_party,omitempty" url:"second_party,omitempty"`
	VirtualNumber             string `json:"virtual_number,omitempty" url:"virtual_number,omitempty"`
	Status                    string `json:"status,omitempty" url:"status,omitempty"`
	CreatedTimeEquals         string `json:"created_time,omitempty" url:"created_time,omitempty"`
	CreatedTimeLessThan       string `json:"created_time__lt,omitempty" url:"created_time__lt,omitempty"`
	CreatedTimeGreaterThan    string `json:"created_time__gt,omitempty" url:"created_time__gt,omitempty"`
	CreatedTimeLessOrEqual    string `json:"created_time__lte,omitempty" url:"created_time__lte,omitempty"`
	CreatedTimeGreaterOrEqual string `json:"created_time__gte,omitempty" url:"created_time__gte,omitempty"`
	ExpiryTimeEquals          string `json:"expiry_time,omitempty" url:"expiry_time,omitempty"`
	ExpiryTimeLessThan        string `json:"expiry_time__lt,omitempty" url:"expiry_time__lt,omitempty"`
	ExpiryTimeGreaterThan     string `json:"expiry_time__gt,omitempty" url:"expiry_time__gt,omitempty"`
	ExpiryTimeLessOrEqual     string `json:"expiry_time__lte,omitempty" url:"expiry_time__lte,omitempty"`
	ExpiryTimeGreaterOrEqual  string `json:"expiry_time__gte,omitempty" url:"expiry_time__gte,omitempty"`
	DurationEquals            int64  `json:"duration,omitempty" url:"duration,omitempty"`
	DurationLessThan          int64  `json:"duration__lt,omitempty"  url:"duration__lt,omitempty"`
	DurationGreaterThan       int64  `json:"duration__gt,omitempty"  url:"duration__gt,omitempty"`
	DurationLessOrEqual       int64  `json:"duration__lte,omitempty" url:"duration__lte,omitempty"`
	DurationGreaterOrEqual    int64  `json:"duration__gte,omitempty" url:"duration__gte,omitempty"`
	Limit                     int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset                    int64  `json:"offset,omitempty" url:"offset,omitempty"`
	SubAccount                string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
}

type VoiceInteractionResponse struct {
	StartTime              string  `json:"start_time,omitempty" url:"start_time,omitempty"`
	EndTime                string  `json:"end_time,omitempty" url:"end_time,omitempty"`
	FirstPartyResourceUrl  string  `json:"first_party_resource_url,omitempty" url:"first_party_resource_url,omitempty"`
	SecondPartyResourceUrl string  `json:"second_party_resource_url,omitempty" url:"second_party_resource_url,omitempty"`
	FirstPartyStatus       string  `json:"first_party_status,omitempty" url:"first_party_status,omitempty"`
	SecondPartyStatus      string  `json:"second_party_status,omitempty" url:"second_party_status,omitempty"`
	Type                   string  `json:"type,omitempty" url:"type,omitempty"`
	TotalCallAmount        float64 `json:"total_call_amount,omitempty" url:"total_call_amount,omitempty"`
	CallBilledDuration     int     `json:"call_billed_duration,omitempty" url:"call_billed_duration,omitempty"`
	RecordingResourceUrl   string  `json:"recording_resource_url,omitempty" url:"recording_resource_url,omitempty"`
	AuthID                 string  `json:"auth_id,omitempty" url:"auth_id,omitempty"`
	TotalCallCount         int     `json:"total_call_count" url:"total_call_count"`
	Duration               float64 `json:"duration" url:"duration"`
}

type CreateMaskingSessionResponse struct {
	ApiID         string         `json:"api_id,omitempty" url:"api_id,omitempty"`
	SessionUUID   string         `json:"session_uuid,omitempty" url:"session_uuid,omitempty"`
	VirtualNumber string         `json:"virtual_number,omitempty" url:"virtual_number,omitempty"`
	Message       string         `json:"message,omitempty" url:"message,omitempty"`
	Session       MaskingSession `json:"session,omitempty" url:"session,omitempty"`
}

type DeleteMaskingSessionResponse struct {
	ApiID   string `json:"api_id,omitempty" url:"api_id,omitempty"`
	Message string `json:"message,omitempty" url:"message,omitempty"`
}

type UpdateMaskingSessionResponse struct {
	ApiID   string         `json:"api_id,omitempty" url:"api_id,omitempty"`
	Message string         `json:"message,omitempty" url:"message,omitempty"`
	Session MaskingSession `json:"session,omitempty" url:"session,omitempty"`
}

type GetMaskingSessionResponse struct {
	ApiID    string         `json:"api_id,omitempty" url:"api_id,omitempty"`
	Response MaskingSession `json:"response,omitempty" url:"response,omitempty"`
}

type ListSessionResponse struct {
	Meta    Meta             `json:"meta" url:"meta"`
	Objects []MaskingSession `json:"objects" url:"objects"`
}

type ListMaskingSessionResponse struct {
	ApiID    string              `json:"api_id" url:"api_id"`
	Response ListSessionResponse `json:"response" url:"response"`
}

func (service *MaskingSessionService) CreateMaskingSession(params CreateMaskingSessionParams) (response *CreateMaskingSessionResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Masking/Session")
	if err != nil {
		return
	}
	response = &CreateMaskingSessionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MaskingSessionService) DeleteMaskingSession(sessionId string) (response *DeleteMaskingSessionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Masking/Session/%s", sessionId)
	if err != nil {
		return
	}
	response = &DeleteMaskingSessionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MaskingSessionService) UpdateMaskingSession(params UpdateMaskingSessionParams, sessionId string) (response *UpdateMaskingSessionResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Masking/Session/%s", sessionId)
	if err != nil {
		return
	}
	response = &UpdateMaskingSessionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MaskingSessionService) GetMaskingSession(sessionId string) (response *GetMaskingSessionResponse, err error) {
	req, err := service.client.NewRequest("GET", nil, "Masking/Session/%s", sessionId)
	if err != nil {
		return
	}
	response = &GetMaskingSessionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *MaskingSessionService) ListMaskingSession(params ListSessionFilterParams) (response *ListMaskingSessionResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "Masking/Session")
	if err != nil {
		return
	}
	response = &ListMaskingSessionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
