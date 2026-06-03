package slack

// HuddleRoom represents a Slack huddle room as it appears in message events
// with subtype "huddle_thread". This is different from CallBlock which is used
// for external call integrations (Zoom, etc.).
type HuddleRoom struct {
	ID                         string                            `json:"id"`
	Name                       string                            `json:"name,omitempty"`
	MediaServer                string                            `json:"media_server,omitempty"`
	CreatedBy                  string                            `json:"created_by,omitempty"`
	DateStart                  int64                             `json:"date_start"`
	DateEnd                    int64                             `json:"date_end"`
	Participants               []string                          `json:"participants,omitempty"`
	ParticipantHistory         []string                          `json:"participant_history,omitempty"`
	ParticipantsEvents         map[string]HuddleParticipantEvent `json:"participants_events,omitempty"`
	ParticipantsCameraOn       []string                          `json:"participants_camera_on,omitempty"`
	ParticipantsCameraOff      []string                          `json:"participants_camera_off,omitempty"`
	ParticipantsScreenshareOn  []string                          `json:"participants_screenshare_on,omitempty"`
	ParticipantsScreenshareOff []string                          `json:"participants_screenshare_off,omitempty"`
	CanvasThreadTs             string                            `json:"canvas_thread_ts,omitempty"`
	ThreadRootTs               string                            `json:"thread_root_ts,omitempty"`
	Channels                   []string                          `json:"channels,omitempty"`
	IsDMCall                   bool                              `json:"is_dm_call"`
	WasRejected                bool                              `json:"was_rejected"`
	WasMissed                  bool                              `json:"was_missed"`
	WasAccepted                bool                              `json:"was_accepted"`
	HasEnded                   bool                              `json:"has_ended"`
	BackgroundID               string                            `json:"background_id,omitempty"`
	CanvasBackground           string                            `json:"canvas_background,omitempty"`
	IsPrewarmed                bool                              `json:"is_prewarmed"`
	IsScheduled                bool                              `json:"is_scheduled"`
	Recording                  *HuddleRecording                  `json:"recording,omitempty"`
	Locale                     string                            `json:"locale,omitempty"`
	AttachedFileIDs            []string                          `json:"attached_file_ids,omitempty"`
	MediaBackendType           string                            `json:"media_backend_type,omitempty"`
	DisplayID                  string                            `json:"display_id,omitempty"`
	ExternalUniqueID           string                            `json:"external_unique_id,omitempty"`
	AppID                      string                            `json:"app_id,omitempty"`
	CallFamily                 string                            `json:"call_family,omitempty"`
	PendingInvitees            map[string]any                    `json:"pending_invitees,omitempty"`
	LastInviteStatusByUser     map[string]any                    `json:"last_invite_status_by_user,omitempty"`
	Knocks                     map[string]any                    `json:"knocks,omitempty"`
	HuddleLink                 string                            `json:"huddle_link,omitempty"`
}

// HuddleParticipantEvent tracks a participant's activity in a huddle.
type HuddleParticipantEvent struct {
	UserTeam       map[string]any `json:"user_team,omitempty"`
	Joined         bool           `json:"joined"`
	CameraOn       bool           `json:"camera_on"`
	CameraOff      bool           `json:"camera_off"`
	ScreenshareOn  bool           `json:"screenshare_on"`
	ScreenshareOff bool           `json:"screenshare_off"`
}

// HuddleRecording contains recording status for a huddle.
type HuddleRecording struct {
	CanRecordSummary string `json:"can_record_summary,omitempty"`
	NoteTaking       bool   `json:"note_taking,omitempty"`
	Summary          bool   `json:"summary,omitempty"`
	SummaryStatus    string `json:"summary_status,omitempty"`
	Transcript       bool   `json:"transcript,omitempty"`
	RecordingUser    string `json:"recording_user,omitempty"`
}
