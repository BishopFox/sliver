package db

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

	DB helpers for the TriggerActivity model. Used by the TTL reaper
	and the TriggerFire RPC to track implant activity.
*/

import (
	"bytes"
	"time"

	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/gorm/clause"
)

// UpsertTriggerActivity creates or updates a TriggerActivity record
// keyed by ImplantConfigID. Called on every TriggerFire RPC so the
// server always knows the implant's last known location.
func UpsertTriggerActivity(configID, name, ip string, port uint32) error {
	now := time.Now()
	record := &models.TriggerActivity{
		ImplantConfigID: configID,
		ImplantName:     name,
		LastSeenIP:      ip,
		LastSeenPort:    port,
		LastActivityAt:  now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return Session().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "implant_config_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"implant_name",
			"last_seen_ip",
			"last_seen_port",
			"last_activity_at",
			"updated_at",
		}),
	}).Create(record).Error
}

// TriggerActivityByConfigID retrieves a single TriggerActivity record.
func TriggerActivityByConfigID(configID string) (*models.TriggerActivity, error) {
	activity := &models.TriggerActivity{}
	err := Session().Where("implant_config_id = ?", configID).First(activity).Error
	if err != nil {
		return nil, err
	}
	return activity, nil
}

// AllTriggerActivities returns every TriggerActivity record.
func AllTriggerActivities() ([]*models.TriggerActivity, error) {
	var activities []*models.TriggerActivity
	err := Session().Find(&activities).Error
	return activities, err
}

// ImplantConfigsByTriggerSecret returns all ImplantConfigs whose
// TriggerWakeSecret matches the given secret. In practice there should
// be at most one, but we return a slice for robustness.
func ImplantConfigsByTriggerSecret(secret []byte) ([]models.ImplantConfig, error) {
	var all []models.ImplantConfig
	err := Session().
		Where("include_trigger_wake = ? AND length(trigger_wake_secret) > 0", true).
		Find(&all).Error
	if err != nil {
		return nil, err
	}
	// GORM/SQLite doesn't support reliable binary WHERE on []byte across
	// all dialects, so filter in Go. The table is small (one row per
	// implant config, not per session).
	var matched []models.ImplantConfig
	for _, cfg := range all {
		if bytes.Equal(cfg.TriggerWakeSecret, secret) {
			matched = append(matched, cfg)
		}
	}
	return matched, nil
}

// MarkReaperFired sets TTLReaperFiredAt to now for the given config ID,
// preventing the reaper from spamming self-destruct packets.
func MarkReaperFired(configID string) error {
	now := time.Now()
	return Session().
		Model(&models.TriggerActivity{}).
		Where("implant_config_id = ?", configID).
		Updates(map[string]interface{}{
			"ttl_reaper_fired_at": &now,
			"updated_at":          now,
		}).Error
}
