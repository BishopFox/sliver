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
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/gorm"
)

const (
	profilesBucketName = "profiles"
)

// SaveImplantProfile - Save a sliver profile to disk
func SaveImplantProfile(name string, config *models.ImplantConfig) error {
	exists, err := db.ImplantProfileByName(name)
	dbSession := db.Session()
	if err != nil {
		return err
	}
	var result *gorm.DB
	if exists != nil {
		result = dbSession.Save(&models.ImplantProfile{
			ID:            exists.ID,
			Name:          name,
			ImplantConfig: config,
		})
	} else {
		result = dbSession.Create(&models.ImplantProfile{
			Name:          name,
			ImplantConfig: config,
		})
	}
	return result.Error
}
