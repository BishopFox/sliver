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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

var (
	serverAgeKeyPair *cryptography.AgeKeyPair
)

func TestMain(m *testing.M) {
	implantConfig := setup()
	code1 := m.Run()
	cleanup(implantConfig)
	os.Exit(code1)
}

func setup() *models.ImplantConfig {
	var err error
	certs.SetupCAs()
	serverAgeKeyPair = cryptography.AgeServerKeyPair()
	peerAgeKeyPair, _ := cryptography.RandomAgeKeyPair()
	implantCrypto.SetSecrets(
		peerAgeKeyPair.Public,
		peerAgeKeyPair.Private,
		"",
		serverAgeKeyPair.Public,
		cryptography.MinisignServerPublicKey(),
	)

	digest := sha256.New()
	digest.Write([]byte(peerAgeKeyPair.Public))
	publicKeyDigest := hex.EncodeToString(digest.Sum(nil))

	implantBuild := &models.ImplantBuild{
		PeerPublicKey:       peerAgeKeyPair.Public,
		PeerPublicKeyDigest: publicKeyDigest,
		PeerPrivateKey:      peerAgeKeyPair.Private,

		AgeServerPublicKey: serverAgeKeyPair.Public,
	}
	err = db.Session().Create(implantBuild).Error
	if err != nil {
		panic(err)
	}

	implantConfig := &models.ImplantConfig{
		ImplantBuilds: []models.ImplantBuild{*implantBuild},
	}
	return implantConfig
}

func cleanup(implantConfig *models.ImplantConfig) {
	db.Session().Delete(implantConfig)
}
