package core

import "sync"

var (
	PivotSessions = &sync.Map{} // ID -> Pivot
)
