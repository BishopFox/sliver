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
	"errors"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

// SaveImplantProfile - Save a sliver profile to disk
func SaveImplantProfile(name string, config *models.ImplantConfig) error {

	profile, err := db.ImplantProfileByName(name)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return err
	}

	dbSession := db.Session()
	if errors.Is(err, db.ErrRecordNotFound) {
		err = dbSession.Create(&models.ImplantProfile{
			Name:          name,
			ImplantConfig: config,
		}).Error
	} else {
		err = dbSession.Save(&models.ImplantProfile{
			ID:            profile.ID,
			Name:          name,
			ImplantConfig: config,
		}).Error
	}
	return err
}
