package mtypes

import (
	"time"
)

type Tag struct {
	Value       string     `json:"tag"`
	Description string     `json:"description"`
	FirstSeen   *time.Time `json:"first-seen,omitempty"`
	LastSeen    *time.Time `json:"last-seen,omitempty"`
}

type TagsResponse struct {
	Items  []Tag  `json:"items"`
	Paging Paging `json:"paging"`
}
