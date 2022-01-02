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
	insecureRand "math/rand"
	"os"
	"testing"
	"time"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

var (
	serverECCKeyPair  *cryptography.ECCKeyPair
	implantECCKeyPair *cryptography.ECCKeyPair
)

func TestMain(m *testing.M) {

	// Run one with deterministic randomness if a
	// crash occurs, we can more easily reproduce it
	insecureRand.Seed(1)
	implantConfig := setup()
	code1 := m.Run()
	cleanup(implantConfig)

	insecureRand.Seed(time.Now().UnixMicro())
	implantConfig = setup()
	code2 := m.Run()
	cleanup(implantConfig)

	os.Exit(code1 | code2)
}

func setup() *models.ImplantConfig {
	var err error
	certs.SetupCAs()
	serverECCKeyPair = cryptography.ECCServerKeyPair()
	implantECCKeyPair, err = cryptography.RandomECCKeyPair()
	if err != nil {
		panic(err)
	}
	totpSecret, err := cryptography.TOTPServerSecret()
	if err != nil {
		panic(err)
	}
	implantCrypto.SetSecrets(
		implantECCKeyPair.PublicBase64(),
		implantECCKeyPair.PrivateBase64(),
		"",
		serverECCKeyPair.PublicBase64(),
		totpSecret,
		"",
	)
	digest := sha256.Sum256(implantECCKeyPair.Public[:])
	implantConfig := &models.ImplantConfig{
		ECCPublicKey:       implantECCKeyPair.PublicBase64(),
		ECCPrivateKey:      implantECCKeyPair.PrivateBase64(),
		ECCPublicKeyDigest: hex.EncodeToString(digest[:]),
		ECCServerPublicKey: serverECCKeyPair.PublicBase64(),
	}
	err = db.Session().Create(implantConfig).Error
	if err != nil {
		panic(err)
	}
	return implantConfig
}

func cleanup(implantConfig *models.ImplantConfig) {
	db.Session().Delete(implantConfig)
}
