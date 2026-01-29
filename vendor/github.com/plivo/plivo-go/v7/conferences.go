package plivo

type ConferenceService struct {
	client *Client
}

type Conference struct {
	ConferenceName        string   `json:"conference_name,omitempty" url:"conference_name,omitempty"`
	ConferenceRunTime     string   `json:"conference_run_time,omitempty" url:"conference_run_time,omitempty"`
	ConferenceMemberCount string   `json:"conference_member_count,omitempty" url:"conference_member_count,omitempty"`
	Members               []Member `json:"members,omitempty" url:"members,omitempty"`
}

type Member struct {
	Muted      bool   `json:"muted,omitempty" url:"muted,omitempty"`
	MemberID   string `json:"member_id,omitempty" url:"member_id,omitempty"`
	Deaf       bool   `json:"deaf,omitempty" url:"deaf,omitempty"`
	From       string `json:"from,omitempty" url:"from,omitempty"`
	To         string `json:"to,omitempty" url:"to,omitempty"`
	CallerName string `json:"caller_name,omitempty" url:"caller_name,omitempty"`
	Direction  string `json:"direction,omitempty" url:"direction,omitempty"`
	CallUUID   string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	JoinTime   string `json:"join_time,omitempty" url:"join_time,omitempty"`
}

type ConferenceIDListResponseBody struct {
	ApiID       string   `json:"api_id" url:"api_id"`
	Conferences []string `json:"conferences" url:"conferences"`
}

type ConferenceRecordParams struct {
	TimeLimit           int64  `json:"time_limit,omitempty" url:"time_limit,omitempty"`
	FileFormat          string `json:"file_format,omitempty" url:"file_format,omitempty"`
	TranscriptionType   string `json:"transcription_type,omitempty" url:"transcription_type,omitempty"`
	TranscriptionUrl    string `json:"transcription_url,omitempty" url:"transcription_url,omitempty"`
	TranscriptionMethod string `json:"transcription_method,omitempty" url:"transcription_method,omitempty"`
	CallbackUrl         string `json:"callback_url,omitempty" url:"callback_url,omitempty"`
	CallbackMethod      string `json:"callback_method,omitempty" url:"callback_method,omitempty"`
}

type ConferenceRecordResponseBody struct {
	Message string `json:"message,omitempty" url:"message,omitempty"`
	Url     string `json:"url,omitempty" url:"url,omitempty"`
}

type ConferenceSpeakParams struct {
	Text     string `json:"text" url:"text"`
	Voice    string `json:"length,omitempty" url:"length,omitempty"`
	Language string `json:"language,omitempty" url:"language,omitempty"`
}

type ConferenceSpeakResponseBody struct {
	Message string `json:"message,omitempty" url:"message,omitempty"`
	ApiID   string `json:"api_id,omitempty" url:"api_id,omitempty"`
}

func (service *ConferenceService) Get(ConferenceId string) (response *Conference, err error) {
	req, err := service.client.NewRequest("GET", nil, "Conference/%s", ConferenceId)
	if err != nil {
		return
	}
	response = &Conference{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) Record(ConferenceId string, Params ConferenceRecordParams) (response *ConferenceRecordResponseBody, err error) {
	req, err := service.client.NewRequest("POST", Params, "Conference/%s/Record", ConferenceId)
	if err != nil {
		return
	}
	response = &ConferenceRecordResponseBody{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) RecordStop(ConferenceId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Record", ConferenceId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *ConferenceService) Delete(ConferenceId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s", ConferenceId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *ConferenceService) DeleteAll() (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference")
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *ConferenceService) IDList() (response *ConferenceIDListResponseBody, err error) {
	req, err := service.client.NewRequest("GET", nil, "Conference")
	if err != nil {
		return
	}
	response = &ConferenceIDListResponseBody{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

type ConferenceMemberActionResponse struct {
	Message  string   `json:"message" url:"message"`
	APIID    string   `json:"api_id" url:"api_id"`
	MemberID []string `json:"member_id" url:"member_id"`
}

func (service *ConferenceService) MemberHangup(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Member/%s", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberKick(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("POST", nil, "Conference/%s/Member/%s/Kick", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberMute(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("POST", nil, "Conference/%s/Member/%s/Mute", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberUnmute(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Member/%s/Mute", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberDeaf(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("POST", nil, "Conference/%s/Member/%s/Deaf", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberUndeaf(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Member/%s/Deaf", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberPlay(conferenceId, memberId, url string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("POST", struct {
		Url string `json:"url" url:"url"`
	}{url}, "Conference/%s/Member/%s/Play", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberPlayStop(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Member/%s/Play", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

type ConferenceMemberSpeakParams struct {
	Text     string `json:"text" url:"text"`
	Voice    string `json:"voice,omitempty" url:"voice,omitempty"`
	Language string `json:"language,omitempty" url:"language,omitempty"`
}

func (service *ConferenceService) MemberSpeak(conferenceId, memberId string, params ConferenceMemberSpeakParams) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "Conference/%s/Member/%s/Speak", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *ConferenceService) MemberSpeakStop(conferenceId, memberId string) (response *ConferenceMemberActionResponse, err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Conference/%s/Member/%s/Speak", conferenceId, memberId)
	if err != nil {
		return
	}
	response = &ConferenceMemberActionResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
