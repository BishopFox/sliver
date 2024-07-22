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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
)

// SaveImplantProfile - Save a sliver profile to disk
func SaveImplantProfile(pbProfile *clientpb.ImplantProfile) (*clientpb.ImplantProfile, error) {
	dbProfile, err := db.ImplantProfileByName(pbProfile.Name)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	profile := models.ImplantProfileFromProtobuf(pbProfile)
	dbSession := db.Session()

	if errors.Is(err, db.ErrRecordNotFound) {
		err = dbSession.Create(&models.ImplantProfile{
			Name:          profile.Name,
			ImplantConfig: profile.ImplantConfig,
		}).Error
		if err != nil {
			return nil, err
		}
		dbProfile, err = db.ImplantProfileByName(profile.Name)
		if err != nil {
			return nil, err
		}
	} else {
		configID, _ := uuid.FromString(dbProfile.Config.ID)
		profile.ImplantConfig.ID = configID

		profileID, _ := uuid.FromString(dbProfile.ID)
		if profileID == uuid.Nil {
			profile.ImplantConfig.ImplantProfileID = nil
		} else {
			profile.ImplantConfig.ImplantProfileID = &profileID
		}

		for _, c2 := range dbProfile.Config.C2 {
			id, _ := uuid.FromString(c2.ID)
			err := db.DeleteC2(id)
			if err != nil {
				return nil, err
			}
		}

		err := dbSession.Save(profile.ImplantConfig).Error
		if err != nil {
			return nil, err
		}
		dbProfile = profile.ToProtobuf()
	}
	return dbProfile, err
}
