package mtypes

import "time"

type ListIPWarmupsResponse struct {
	Paging Paging     `json:"paging"`
	Items  []IPWarmup `json:"items"`
}

// IPWarmup - is a response for ListIPWarmups.
type IPWarmup struct {
	IP               string    `json:"ip"`
	SentWithinStage  string    `json:"sent_within_stage"`
	Throttle         int       `json:"throttle"`
	StageNumber      int       `json:"stage_number"`
	StageStartVolume int       `json:"stage_start_volume"`
	StageStartTime   time.Time `json:"stage_start_time"`
	StageVolumeLimit int       `json:"stage_volume_limit"`
}

type IPWarmupDetailsResponse struct {
	Details IPWarmupDetails `json:"details"`
}

type IPWarmupDetails struct {
	SentWithinStage   string                 `json:"sent_within_stage"`
	Throttle          int                    `json:"throttle"`
	StageNumber       int                    `json:"stage_number"`
	StageStartVolume  int                    `json:"stage_start_volume"`
	StageStartTime    time.Time              `json:"stage_start_time"`
	StageVolumeLimit  int                    `json:"stage_volume_limit"`
	StageStartedAt    time.Time              `json:"stage_started_at"`
	HourStartedAt     time.Time              `json:"hour_started_at"`
	PlanStartedAt     time.Time              `json:"plan_started_at"`
	PlanLastUpdatedAt time.Time              `json:"plan_last_updated_at"`
	TotalStages       int                    `json:"total_stages"`
	StageHistory      []IPWarmupStageHistory `json:"stage_history"`
}

type IPWarmupStageHistory struct {
	FirstUpdatedAt time.Time `json:"first_updated_at"`
	CompletedAt    time.Time `json:"completed_at"`
	Limit          int       `json:"limit"`
}
