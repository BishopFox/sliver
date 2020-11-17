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
               packages outside of /db/models/

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
	dbSession := Session()
	builds := []*models.ImplantBuild{}
	result := dbSession.Where(&models.ImplantBuild{}).Find(&builds)
	return builds, result.Error
}

// ImplantBuildByName - Fetch implant build by name
func ImplantBuildByName(name string) (*models.ImplantBuild, error) {
	dbSession := Session()
	build := models.ImplantBuild{}
	result := dbSession.Where(&models.ImplantBuild{Name: name}).First(&build)
	return &build, result.Error
}

// ImplantBuildNames - Fetch a list of all build names
func ImplantBuildNames() ([]string, error) {
	dbSession := Session()
	builds := []*models.ImplantBuild{}
	result := dbSession.Where(&models.ImplantBuild{}).Find(&builds)
	if result.Error != nil {
		return []string{}, result.Error
	}
	names := []string{}
	for _, build := range builds {
		names = append(names, build.Name)
	}
	return names, result.Error
}

// ImplantProfiles - Fetch a map of name<->profiles current in the database
func ImplantProfiles() ([]*models.ImplantProfile, error) {
	profiles := []*models.ImplantProfile{}
	dbSession := Session()
	err := dbSession.Where(&models.ImplantProfile{}).Preload("ImplantConfig").Find(&profiles).Error
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles {
		c2s := []models.ImplantC2{}
		err := dbSession.Where(&models.ImplantC2{
			ImplantConfigID: profile.ImplantConfig.ID,
		}).Find(&c2s).Error
		if err != nil {
			return nil, err
		}
		profile.ImplantConfig.C2 = c2s
	}
	return profiles, nil
}

// ImplantProfileByName - Fetch implant build by name
func ImplantProfileByName(name string) (*models.ImplantProfile, error) {
	dbSession := Session()
	profile := models.ImplantProfile{}
	err := dbSession.Where(&models.ImplantProfile{
		Name: name,
	}).Preload("ImplantConfig").First(&profile).Error
	if err != nil {
		return nil, err
	}

	c2s := []models.ImplantC2{}
	err = dbSession.Where(&models.ImplantC2{
		ImplantConfigID: profile.ImplantConfig.ID,
	}).Find(&c2s).Error
	if err != nil {
		return nil, err
	}
	profile.ImplantConfig.C2 = c2s

	return &profile, err
}

// ImplantProfileNames - Fetch a list of all build names
func ImplantProfileNames() ([]string, error) {
	dbSession := Session()
	profiles := []*models.ImplantProfile{}
	result := dbSession.Where(&models.ImplantProfile{}).Find(&profiles)
	if result.Error != nil {
		return []string{}, result.Error
	}
	names := []string{}
	for _, build := range profiles {
		names = append(names, build.Name)
	}
	return names, result.Error
}

// ProfileByName - Fetch a single profile from the database
func ProfileByName(name string) (*models.ImplantProfile, error) {
	dbProfile := &models.ImplantProfile{}
	dbSession := Session()
	result := dbSession.Where(&models.ImplantProfile{Name: name}).Find(&dbProfile)
	return dbProfile, result.Error
}

// ListCanaries - List of all embedded canaries
func ListCanaries() ([]*models.DNSCanary, error) {
	canaries := []*models.DNSCanary{}
	dbSession := Session()
	result := dbSession.Where(&models.DNSCanary{}).Find(&canaries)
	return canaries, result.Error
}

// CanaryByDomain - Check if a canary exists
func CanaryByDomain(domain string) (*models.DNSCanary, error) {
	dbSession := Session()
	canary := models.DNSCanary{}
	result := dbSession.Where(&models.DNSCanary{Domain: domain}).First(&canary)
	return &canary, result.Error
}

// WebsiteByName - Get website by name
func WebsiteByName(name string) (*models.Website, error) {
	website := models.Website{}
	dbSession := Session()
	result := dbSession.Where(&models.Website{Name: name}).First(&website)
	if result.Error != nil {
		return nil, result.Error
	}
	return &website, nil
}
