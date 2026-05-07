package plivo

type RecordingService struct {
	client *Client
}

type Recording struct {
	AddTime                       string  `json:"add_time,omitempty" url:"add_time,omitempty"`
	CallUUID                      string  `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	MonthlyRecordingStorageAmount float64 `json:"monthly_recording_storage_amount,omitempty" url:"monthly_recording_storage_amount,omitempty"`
	RecordingStorageDuration      int64   `json:"recording_storage_duration,omitempty" url:"recording_storage_duration,omitempty"`
	RecordingStorageRate          float64 `json:"recording_storage_rate,omitempty" url:"recording_storage_rate,omitempty"`
	RecordingID                   string  `json:"recording_id,omitempty" url:"recording_id,omitempty"`
	RecordingType                 string  `json:"recording_type,omitempty" url:"recording_type,omitempty"`
	RecordingFormat               string  `json:"recording_format,omitempty" url:"recording_format,omitempty"`
	ConferenceName                string  `json:"conference_name,omitempty" url:"conference_name,omitempty"`
	RecordingURL                  string  `json:"recording_url,omitempty" url:"recording_url,omitempty"`
	ResourceURI                   string  `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	RecordingStartMS              string  `json:"recording_start_ms,omitempty" url:"recording_start_ms,omitempty"`
	RecordingEndMS                string  `json:"recording_end_ms,omitempty" url:"recording_end_ms,omitempty"`
	RecordingDurationMS           string  `json:"recording_duration_ms,omitempty" url:"recording_duration_ms,omitempty"`
	RoundedRecordingDuration      int64   `json:"rounded_recording_duration,omitempty" url:"rounded_recording_duration,omitempty"`
	FromNumber                    string  `json:"from_number,omitempty" url:"from_number,omitempty"`
	ToNumber                      string  `json:"to_number,omitempty" url:"to_number,omitempty"`
}

type RecordingListParams struct {
	// Query parameters.
	Subaccount                             string `json:"subaccount,omitempty" url:"subaccount,omitempty"`
	CallUUID                               string `json:"call_uuid,omitempty" url:"call_uuid,omitempty"`
	AddTimeLessThan                        string `json:"add_time__lt,omitempty" url:"add_time__lt,omitempty"`
	AddTimeGreaterThan                     string `json:"add_time__gt,omitempty" url:"add_time__gt,omitempty"`
	AddTimeLessOrEqual                     string `json:"add_time__lte,omitempty" url:"add_time__lte,omitempty"`
	AddTimeGreaterOrEqual                  string `json:"add_time__gte,omitempty" url:"add_time__gte,omitempty"`
	Limit                                  int64  `json:"limit,omitempty" url:"limit,omitempty"`
	Offset                                 int64  `json:"offset,omitempty" url:"offset,omitempty"`
	FromNumber                             string `json:"from_number,omitempty" url:"from_number,omitempty"`
	ToNumber                               string `json:"to_number,omitempty" url:"to_number,omitempty"`
	ConferenceName                         string `json:"conference_name,omitempty" url:"conference_name,omitempty"`
	MpcName                                string `json:"mpc_name,omitempty" url:"mpc_name,omitempty"`
	ConferenceUuid                         string `json:"conference_uuid,omitempty" url:"conference_uuid,omitempty"`
	MpcUuid                                string `json:"mpc_uuid,omitempty" url:"mpc_uuid,omitempty"`
	RecordingStorageDurationEquals         int64  `json:"recording_storage_duration,omitempty" url:"recording_storage_duration,omitempty"`
	RecordingStorageDurationLessThan       int64  `json:"recording_storage_duration__lt,omitempty" url:"recording_storage_duration__lt,omitempty"`
	RecordingStorageDurationGreaterThan    int64  `json:"recording_storage_duration__gt,omitempty" url:"recording_storage_duration__gt,omitempty"`
	RecordingStorageDurationLessOrEqual    int64  `json:"recording_storage_duration__lte,omitempty" url:"recording_storage_duration__lte,omitempty"`
	RecordingStorageDurationGreaterOrEqual int64  `json:"recording_storage_duration__gte,omitempty" url:"recording_storage_duration__gte,omitempty"`
}

type RecordingListResponse struct {
	ApiID   string       `json:"api_id" url:"api_id"`
	Meta    *Meta        `json:"meta" url:"meta"`
	Objects []*Recording `json:"objects" url:"objects"`
}

func (service *RecordingService) Get(RecordingId string) (response *Recording, err error) {
	req, err := service.client.NewRequest("GET", nil, "Recording/%s", RecordingId)
	if err != nil {
		return
	}
	response = &Recording{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}

func (service *RecordingService) Delete(RecordingId string) (err error) {
	req, err := service.client.NewRequest("DELETE", nil, "Recording/%s", RecordingId)
	if err != nil {
		return
	}
	err = service.client.ExecuteRequest(req, nil, isVoiceRequest())
	return
}

func (service *RecordingService) List(params RecordingListParams) (response *RecordingListResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "Recording")
	if err != nil {
		return
	}
	response = &RecordingListResponse{}
	err = service.client.ExecuteRequest(req, response, isVoiceRequest())
	return
}
