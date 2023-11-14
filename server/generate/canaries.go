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
	insecureRand "math/rand"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
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
	index := insecureRand.Intn(len(dnsCharSet) - 12) // ensure first char is alphabetic
	subdomain = append(subdomain, dnsCharSet[index])
	for i := 0; i < canarySize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		subdomain = append(subdomain, dnsCharSet[index])
	}
	return string(subdomain)
}

// UpdateCanary - Update an existing canary
func UpdateCanary(canary *clientpb.DNSCanary) error {
	dbSession := db.Session()
	result := dbSession.Save(&canary)
	return result.Error
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
	index := insecureRand.Intn(len(g.ParentDomains))

	parentDomain := g.ParentDomains[index]
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
