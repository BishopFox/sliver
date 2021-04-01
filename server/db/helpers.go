package db

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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

    IMPORTANT: These should be read-only functions and cannot rely on any
               packages outside of /server/db/models

*/

import (
	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/gorm"
)

var (
	// ErrRecordNotFound - Record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound
)

// ImplantBuilds - Return all implant builds
func ImplantBuilds() ([]*models.ImplantBuild, error) {
	builds := []*models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{}).Preload("ImplantConfig").Find(&builds).Error
	if err != nil {
		return nil, err
	}
	for _, build := range builds {
		err = loadC2s(&build.ImplantConfig)
		if err != nil {
			return nil, err
		}
	}
	return builds, err
}

// ImplantBuildByName - Fetch implant build by name
func ImplantBuildByName(name string) (*models.ImplantBuild, error) {
	build := models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{
		Name: name,
	}).Preload("ImplantConfig").First(&build).Error
	if err != nil {
		return nil, err
	}
	err = loadC2s(&build.ImplantConfig)
	if err != nil {
		return nil, err
	}

	return &build, err
}

// ImplantBuildNames - Fetch a list of all build names
func ImplantBuildNames() ([]string, error) {
	builds := []*models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{}).Find(&builds).Error
	if err != nil {
		return []string{}, err
	}
	names := []string{}
	for _, build := range builds {
		names = append(names, build.Name)
	}
	return names, nil
}

// ImplantProfiles - Fetch a map of name<->profiles current in the database
func ImplantProfiles() ([]*models.ImplantProfile, error) {
	profiles := []*models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{}).Preload("ImplantConfig").Find(&profiles).Error
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles {
		err = loadC2s(profile.ImplantConfig)
		if err != nil {
			return nil, err
		}
	}
	return profiles, nil
}

// ImplantProfileByName - Fetch implant build by name
func ImplantProfileByName(name string) (*models.ImplantProfile, error) {
	profile := models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{
		Name: name,
	}).Preload("ImplantConfig").First(&profile).Error
	if err != nil {
		return nil, err
	}

	err = loadC2s(profile.ImplantConfig)
	if err != nil {
		return nil, err
	}

	return &profile, err
}

// C2s are not eager-loaded, this will load them for a given ImplantConfig
// I wasn't able to get GORM's nested loading to work, so I went with this.
func loadC2s(config *models.ImplantConfig) error {
	c2s := []models.ImplantC2{}
	err := Session().Where(&models.ImplantC2{
		ImplantConfigID: config.ID,
	}).Find(&c2s).Error
	if err != nil {
		return err
	}
	config.C2 = c2s
	return nil
}

// ImplantProfileNames - Fetch a list of all build names
func ImplantProfileNames() ([]string, error) {
	profiles := []*models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{}).Find(&profiles).Error
	if err != nil {
		return []string{}, err
	}
	names := []string{}
	for _, build := range profiles {
		names = append(names, build.Name)
	}
	return names, nil
}

// ProfileByName - Fetch a single profile from the database
func ProfileByName(name string) (*models.ImplantProfile, error) {
	dbProfile := &models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{Name: name}).Find(&dbProfile).Error
	return dbProfile, err
}

// ListCanaries - List of all embedded canaries
func ListCanaries() ([]*models.DNSCanary, error) {
	canaries := []*models.DNSCanary{}
	err := Session().Where(&models.DNSCanary{}).Find(&canaries).Error
	return canaries, err
}

// CanaryByDomain - Check if a canary exists
func CanaryByDomain(domain string) (*models.DNSCanary, error) {
	dbSession := Session()
	canary := models.DNSCanary{}
	err := dbSession.Where(&models.DNSCanary{Domain: domain}).First(&canary).Error
	return &canary, err
}

// WebsiteByName - Get website by name
func WebsiteByName(name string) (*models.Website, error) {
	website := models.Website{}
	err := Session().Where(&models.Website{Name: name}).First(&website).Error
	if err != nil {
		return nil, err
	}
	return &website, nil
}

// WGPeerIPs - Fetch a list of ips for all wireguard peers
func WGPeerIPs() ([]string, error) {
	wgPeers := []*models.WGPeer{}
	err := Session().Where(&models.WGPeer{}).Find(&wgPeers).Error
	if err != nil {
		return nil, err
	}
	ips := []string{}
	for _, peer := range wgPeers {
		ips = append(ips, peer.TunIP)
	}
	return ips, nil
}
