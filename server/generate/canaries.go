package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
*/

import (
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/util"
	"gorm.io/gorm"
)

const (
	// CanaryBucketName - DNS Canary bucket name
	CanaryBucketName = "canaries"
	canaryPrefix     = "can://"
	canarySize       = 6
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")
)

func canarySubDomain() string {
	subdomain := []rune{}
	index := util.Intn(len(dnsCharSet) - 12) // ensure first char is alphabetic
	subdomain = append(subdomain, dnsCharSet[index])
	for i := 0; i < canarySize; i++ {
		index := util.Intn(len(dnsCharSet))
		subdomain = append(subdomain, dnsCharSet[index])
	}
	return string(subdomain)
}

// TriggerCanary records a DNS query and reports whether it first triggered the
// existing canary. The caller's status may be stale; only its ID is used.
func TriggerCanary(canary *clientpb.DNSCanary) (bool, error) {
	if canary == nil || canary.ID == "" {
		return false, fmt.Errorf("missing canary ID")
	}
	id, err := models.ParseUUID(canary.ID)
	if err != nil || id == models.NilUUID() {
		return false, fmt.Errorf("invalid canary ID %q", canary.ID)
	}

	firstTrigger := false
	err = db.Session().Transaction(func(tx *gorm.DB) error {
		count := tx.Model(&models.DNSCanary{}).Where("id = ?", id).
			UpdateColumn("count", gorm.Expr("count + ?", 1))
		if count.Error != nil {
			return count.Error
		}
		if count.RowsAffected != 1 {
			return db.ErrRecordNotFound
		}

		// The increment locks the existing row before timestamps are sampled,
		// so waiting queries cannot move LatestTrigger backwards.
		first := tx.Model(&models.DNSCanary{}).
			Where("id = ? AND triggered = ?", id, false).
			Updates(models.DNSCanary{Triggered: true, FirstTrigger: time.Now()})
		if first.Error != nil {
			return first.Error
		}
		firstTrigger = first.RowsAffected == 1

		return tx.Model(&models.DNSCanary{}).Where("id = ?", id).
			Updates(models.DNSCanary{LatestTrigger: time.Now()}).Error
	})
	return firstTrigger && err == nil, err
}

// CanaryGenerator - Holds data related to canary generation
type CanaryGenerator struct {
	ImplantName   string
	ParentDomains []string
}

// GenerateCanary - Generate a canary domain and save it to the db
// currently this gets called by template engine
func (g *CanaryGenerator) GenerateCanary() string {

	if len(g.ParentDomains) < 1 {
		buildLog.Warnf("No parent domains")
		return ""
	}

	// Don't need secure random here
	index := util.Intn(len(g.ParentDomains))

	parentDomain := strings.ToLower(strings.TrimSpace(g.ParentDomains[index]))
	parentDomain = strings.TrimPrefix(parentDomain, ".")
	if !strings.HasSuffix(parentDomain, ".") {
		parentDomain += "." // Ensure we have the FQDN
	}

	subdomain := canarySubDomain()
	canaryDomain := fmt.Sprintf("%s.%s", subdomain, parentDomain)
	buildLog.Infof("Generated new canary domain %s", canaryDomain)

	canary := &models.DNSCanary{
		ImplantName: g.ImplantName,
		Domain:      canaryDomain,
		Triggered:   false,
		Count:       0,
	}
	dbSession := db.Session()
	dbSession.Create(&canary)

	return canaryDomain
}
