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

	------------------------------------------------------------------------

	We've put a little effort to making the server at least not super easily fingerprintable,
	though I'm guessing it's also still not super hard to do. The server must receive a valid
	TOTP code before we start returning any non-error records. All requests must be formatted
	as valid protobuf and contain a 24-bit "dns session ID" (16777216 possible values), and a
	8 bit "message ID." The server only responds to non-TOTP queries with valid dns session IDs
	16,777,216 can probably be bruteforced but it'll at least be slow.

	DNS command and control outline:
		1. Implant sends TOTP encoded message to DNS server, server checks validity
		2. DNS server responds with the "DNS Session ID" which is just some random value
		3. Requests with valid DNS session IDs enable the server to respond with CRC32 responses
		4. Implant establishes encrypted session

*/
import (
	"fmt"
	"log"
	"os"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

func SetupDefaultC2Profiles() {

	config, err := db.LoadHTTPC2ConfigByID(constants.DefaultC2Profile)
	if err != nil {
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	if config.Name == "" {
		defaultConfig := configs.GenerateDefaultHTTPC2Config()
		httpC2ConfigModel := models.HTTPC2ConfigFromProtobuf(defaultConfig)
		err = db.HTTPC2ConfigSave(httpC2ConfigModel)
		if err != nil {
			fmt.Println(err)
			log.Printf("Error:\n%s", err)
			os.Exit(-1)
		}
	}
}
