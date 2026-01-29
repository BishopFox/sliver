package c2_test

/*
	Sliver Implant Framework
	Copyright (C) 2025  Bishop Fox

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
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

var (
	serverAgeKeyPair *cryptography.AgeKeyPair
	peerAgeKeyPair   *cryptography.AgeKeyPair

	testImplantConfig *models.ImplantConfig
	testImplantBuild  *models.ImplantBuild

	mtlsCACertPEM string
	mtlsCertPEM   string
	mtlsKeyPEM    string
)

const (
	defaultHTTPC2ConfigName = "default"
	testImplantCertName     = "test-implant"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func setup() {
	var err error
	certs.SetupCAs()
	serverAgeKeyPair = cryptography.AgeServerKeyPair()
	peerAgeKeyPair, err = cryptography.RandomAgeKeyPair()
	if err != nil {
		panic(err)
	}
	implantCrypto.SetSecrets(
		peerAgeKeyPair.Public,
		peerAgeKeyPair.Private,
		cryptography.MinisignServerSign([]byte(peerAgeKeyPair.Public)),
		serverAgeKeyPair.Public,
		cryptography.MinisignServerPublicKey(),
	)

	httpConfig, err := db.LoadHTTPC2ConfigByName(defaultHTTPC2ConfigName)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		panic(err)
	}
	if httpConfig == nil || httpConfig.ServerConfig == nil || len(httpConfig.ServerConfig.Cookies) == 0 {
		defaultConfig := configs.GenerateDefaultHTTPC2Config()
		if err := db.SaveHTTPC2Config(defaultConfig); err != nil {
			panic(err)
		}
	}

	caPEM, _, err := certs.GetCertificateAuthorityPEM(certs.MtlsServerCA)
	if err != nil {
		panic(err)
	}
	mtlsCACertPEM = string(caPEM)

	_, _, err = certs.GetECCCertificate(certs.MtlsServerCA, "localhost")
	if errors.Is(err, certs.ErrCertDoesNotExist) {
		_, _, err = certs.MtlsC2ServerGenerateECCCertificate("localhost")
	}
	if err != nil {
		panic(err)
	}

	certPEM, keyPEM, err := certs.GetECCCertificate(certs.MtlsImplantCA, testImplantCertName)
	if errors.Is(err, certs.ErrCertDoesNotExist) {
		certPEM, keyPEM, err = certs.MtlsC2ImplantGenerateECCCertificate(testImplantCertName)
	}
	if err != nil {
		panic(err)
	}
	mtlsCertPEM = string(certPEM)
	mtlsKeyPEM = string(keyPEM)

	testImplantConfig = &models.ImplantConfig{
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
		IncludeHTTP:        true,
		IncludeMTLS:        true,
		HttpC2ConfigName:   defaultHTTPC2ConfigName,
		ConnectionStrategy: "s",
	}
	if err := db.Session().Create(testImplantConfig).Error; err != nil {
		panic(err)
	}

	digest := sha256.Sum256([]byte(peerAgeKeyPair.Public))
	testImplantBuild = &models.ImplantBuild{
		Name:                    "e2e-test-" + time.Now().Format("20060102150405.000000000"),
		ImplantConfigID:         testImplantConfig.ID,
		PeerPublicKey:           peerAgeKeyPair.Public,
		PeerPublicKeyDigest:     hex.EncodeToString(digest[:]),
		PeerPrivateKey:          peerAgeKeyPair.Private,
		PeerPublicKeySignature:  cryptography.MinisignServerSign([]byte(peerAgeKeyPair.Public)),
		AgeServerPublicKey:      serverAgeKeyPair.Public,
		MinisignServerPublicKey: cryptography.MinisignServerPublicKey(),
		MtlsCACert:              mtlsCACertPEM,
		MtlsCert:                mtlsCertPEM,
		MtlsKey:                 mtlsKeyPEM,
	}
	if err := db.Session().Create(testImplantBuild).Error; err != nil {
		panic(err)
	}
}

func cleanup() {
	if testImplantBuild != nil {
		db.Session().Delete(testImplantBuild)
	}
	if testImplantConfig != nil {
		db.Session().Delete(testImplantConfig)
	}
}
