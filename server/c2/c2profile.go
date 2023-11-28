package c2

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"log"
	"os"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
)

func SetupDefaultC2Profiles() {

	config, err := db.LoadHTTPC2ConfigByName(constants.DefaultC2Profile)
	if err != nil {
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	if config.Name == "" {
		defaultConfig := configs.GenerateDefaultHTTPC2Config()
		err = db.SaveHTTPC2Config(defaultConfig)
		if err != nil {
			log.Printf("Error:\n%s", err)
			os.Exit(-1)
		}
	}
}
