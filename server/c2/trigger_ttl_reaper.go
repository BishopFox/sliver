package c2

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

	Server-side TTL reaper: a SECONDARY defense-in-depth fallback for
	trigger-enabled implants. The implant-side TTL watchdog is the
	PRIMARY mechanism. This reaper runs as a background goroutine for the
	lifetime of the server process and periodically checks for implants
	whose TTL has expired, firing self-destruct packets at their last
	known IP:port as a last resort.

	Design constraints:
	- Conservative: only fires self-destruct when TTL is clearly expired.
	- Rate-limited: at most one self-destruct per implant per hour.
	- Observable: every action is logged to the c2/ttl-reaper logger.
	- Non-blocking: failures (network errors, missing data) are logged
	  and skipped, never fatal.
*/

import (
	"time"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

var reaperLog = log.NamedLogger("c2", "ttl-reaper")

const (
	// DefaultReaperInterval is how often the reaper checks for expired TTLs.
	DefaultReaperInterval = 5 * time.Minute

	// reaperCooldown is the minimum time between self-destruct attempts
	// for a single implant. Prevents spamming self-destruct packets at
	// unreachable targets.
	reaperCooldown = 1 * time.Hour
)

// StartTTLReaper launches the background TTL reaper goroutine. It runs
// for the lifetime of the process (no stop channel -- the goroutine
// dies with the server). Call once from daemon startup.
func StartTTLReaper() {
	go ttlReaperLoop(DefaultReaperInterval)
}

// ttlReaperLoop is the main reaper loop. Separated from StartTTLReaper
// for testability (callers can pass a custom interval).
func ttlReaperLoop(interval time.Duration) {
	reaperLog.Infof("TTL reaper started (interval=%v, cooldown=%v)", interval, reaperCooldown)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		reaperSweep()
	}
}

// reaperSweep runs one pass of the reaper: enumerate all trigger-enabled
// implant configs with TTL, cross-reference with TriggerActivity, and
// fire self-destruct at any that have expired.
func reaperSweep() {
	configs, err := triggerConfigsWithTTL()
	if err != nil {
		reaperLog.Errorf("reaper sweep: failed to query trigger configs: %v", err)
		return
	}
	if len(configs) == 0 {
		return // nothing to do -- common case
	}

	activities, err := db.AllTriggerActivities()
	if err != nil {
		reaperLog.Errorf("reaper sweep: failed to query trigger activities: %v", err)
		return
	}
	activityByConfig := indexActivitiesByConfig(activities)

	now := time.Now()
	for _, cfg := range configs {
		configID := cfg.ID.String()
		activity, hasActivity := activityByConfig[configID]

		// Check if the TTL has expired based on the absolute deadline.
		if cfg.TTLExpiresAtUnix > 0 && now.Unix() < cfg.TTLExpiresAtUnix {
			continue // TTL not expired yet
		}

		// Also check activity-relative TTL: if the operator has been
		// actively firing triggers at this implant recently, the implant
		// is presumably still alive and the operator is managing it.
		if hasActivity && cfg.TTLMinutes > 0 {
			activityDeadline := activity.LastActivityAt.Add(time.Duration(cfg.TTLMinutes) * time.Minute)
			if now.Before(activityDeadline) {
				continue // recent activity extends the window
			}
		}

		// TTL is expired. Can we reach the implant?
		if !hasActivity || activity.LastSeenIP == "" {
			reaperLog.Warnf("reaper: implant config %s (name=%s) TTL expired but no last-known IP -- cannot send self-destruct",
				configID, buildNameForConfig(configID))
			continue
		}

		// Rate-limit: don't spam self-destruct.
		if activity.TTLReaperFiredAt != nil {
			if now.Sub(*activity.TTLReaperFiredAt) < reaperCooldown {
				reaperLog.Debugf("reaper: implant config %s -- self-destruct already fired at %v, cooldown not elapsed",
					configID, activity.TTLReaperFiredAt.Format(time.RFC3339))
				continue
			}
		}

		// Fire self-destruct.
		secret := string(cfg.TriggerWakeSecret)
		if secret == "" {
			reaperLog.Warnf("reaper: implant config %s -- TTL expired but TriggerWakeSecret is empty, cannot sign self-destruct",
				configID)
			continue
		}

		reaperLog.Warnf("reaper: FIRING SELF-DESTRUCT at implant config %s (name=%s) target=%s:%d -- TTL expired (deadline=%v, last_activity=%v)",
			configID,
			buildNameForConfig(configID),
			activity.LastSeenIP,
			activity.LastSeenPort,
			time.Unix(cfg.TTLExpiresAtUnix, 0).Format(time.RFC3339),
			activity.LastActivityAt.Format(time.RFC3339),
		)

		_, err := FireTriggerPacket(
			activity.LastSeenIP,
			int(activity.LastSeenPort),
			"self-destruct",
			secret,
			"ttl-reaper",
			"",
		)
		if err != nil {
			reaperLog.Errorf("reaper: self-destruct fire failed for config %s: %v", configID, err)
			// Still mark it as fired to respect cooldown -- the network
			// error doesn't mean we should hammer the target.
		} else {
			reaperLog.Infof("reaper: self-destruct packet sent to %s:%d for config %s",
				activity.LastSeenIP, activity.LastSeenPort, configID)
		}

		if markErr := db.MarkReaperFired(configID); markErr != nil {
			reaperLog.Errorf("reaper: failed to mark reaper fired for config %s: %v", configID, markErr)
		}
	}
}

// triggerConfigsWithTTL queries all ImplantConfigs that have trigger
// wake enabled AND TTL enabled.
func triggerConfigsWithTTL() ([]models.ImplantConfig, error) {
	var configs []models.ImplantConfig
	err := db.Session().
		Where("include_trigger_wake = ? AND ttl_enabled = ?", true, true).
		Find(&configs).Error
	return configs, err
}

// buildNameForConfig attempts to find a friendly build name for a
// config ID. Returns the config ID itself if no build is found.
func buildNameForConfig(configID string) string {
	build, err := db.ImplantBuildByConfigID(configID)
	if err != nil || build == nil {
		return configID
	}
	return build.Name
}

// indexActivitiesByConfig builds a map from ImplantConfigID to
// TriggerActivity for O(1) lookups during the sweep.
func indexActivitiesByConfig(activities []*models.TriggerActivity) map[string]*models.TriggerActivity {
	m := make(map[string]*models.TriggerActivity, len(activities))
	for _, a := range activities {
		m[a.ImplantConfigID] = a
	}
	return m
}
