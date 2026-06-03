package models

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.

	----------------------------------------------------------------------

	TriggerActivity tracks the last known location and activity time for
	trigger-enabled implants. The server-side TTL reaper uses this data
	to send self-destruct packets to implants that appear to have survived
	past their configured TTL.

	The implant-side TTL watchdog is the PRIMARY mechanism; this is the
	SECONDARY defense-in-depth fallback.
*/

import (
	"time"
)

// TriggerActivity records the last operator interaction with a
// trigger-enabled implant. Upserted on every TriggerFire RPC call so
// the server knows where to send self-destruct packets if the
// implant's own TTL watchdog fails.
type TriggerActivity struct {
	ID               uint       `gorm:"primaryKey"`
	ImplantConfigID  string     `gorm:"uniqueIndex;not null"` // FK to ImplantConfig (UUID string)
	ImplantName      string     // friendly name from ImplantBuild
	LastSeenIP       string     // last known routable IP
	LastSeenPort     uint32     // trigger port extracted from bind addr
	LastActivityAt   time.Time  // last TriggerFire call targeting this implant
	TTLReaperFiredAt *time.Time // last time the reaper sent a self-destruct; nil = never
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
