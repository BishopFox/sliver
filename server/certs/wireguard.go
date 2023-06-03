package certs

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
	"encoding/hex"
	"errors"
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

var (
	wgKeysLog = log.NamedLogger("certs", "wg-keys")

	ErrWGPeerDoesNotExist     = errors.New("wg peer does not exist")
	ErrWGServerKeysDoNotExist = errors.New("wg server keys do not exist")
)

// SetupWGKeys - Setup server keys
func SetupWGKeys() {
	if _, _, err := GetWGServerKeys(); err != nil {
		wgKeysLog.Info("No wg server keys detected")
		GenerateWGKeys(false, "")
	}
}

// ImplantGenerateWGKeys - Generate WG keys for implant
func ImplantGenerateWGKeys(wgPeerTunIP string) (string, string, error) {
	isPeer := true
	privKey, pubKey, err := GenerateWGKeys(isPeer, wgPeerTunIP)

	if err != nil {
		wgKeysLog.Errorf("Error generating wg keys for peer: %s", err)
		wgKeysLog.Errorf("priv:  %s", privKey)
		wgKeysLog.Errorf("pub:  %s", pubKey)
		return "", "", err
	}

	return privKey, pubKey, nil
}

// GetWGSPeers - Get a map of Pubkey:TunIP for existing wg peers
func GetWGPeers() (map[string]string, error) {

	peers := make(map[string]string)

	wgPeerModels := []models.WGPeer{}
	dbSession := db.Session()
	err := dbSession.Where(&models.WGPeer{}).Find(&wgPeerModels).Error
	if errors.Is(err, db.ErrRecordNotFound) {
		return nil, ErrWGPeerDoesNotExist
	} else if err != nil {
		return nil, err
	}

	for _, v := range wgPeerModels {
		peers[v.PubKey] = v.TunIP
	}
	return peers, nil
}

// GetWGServerKeys - Get existing wg server keys
func GetWGServerKeys() (string, string, error) {

	wgKeysLog.Info("Getting wg keys for wg server")

	wgKeysModel := models.WGKeys{}
	dbSession := db.Session()
	result := dbSession.First(&wgKeysModel)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return "", "", ErrWGServerKeysDoNotExist
	}
	if result.Error != nil {
		return "", "", result.Error
	}

	return wgKeysModel.PrivKey, wgKeysModel.PubKey, nil
}

// GenerateWGKeys - Generates and saves new wg keys
func GenerateWGKeys(isPeer bool, wgPeerTunIP string) (string, string, error) {
	privKey, pubKey, err := genWGKeys()

	if err != nil {
		return "", "", err
	}

	if err := saveWGKeys(isPeer, wgPeerTunIP, privKey, pubKey); err != nil {
		wgKeysLog.Error("Error Saving wg keys: ", err)
		return "", "", err
	}
	return privKey, pubKey, nil
}

func genWGKeys() (string, string, error) {
	wgKeysLog.Infof("Generating wg keys")

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		wgKeysLog.Warnf("Failed to generate private key: %s", err)
		return "", "", err
	}
	publicKey := privateKey.PublicKey()
	return hex.EncodeToString(privateKey[:]), hex.EncodeToString(publicKey[:]), nil
}

// saveWGKeys - Saves wg keys to the database
func saveWGKeys(isPeer bool, wgPeerTunIP string, privKey string, pubKey string) error {

	wgKeysLog.Info("Saving wg keys")
	dbSession := db.Session()

	var result *gorm.DB

	if isPeer {
		wgPeerModels := &models.WGPeer{
			PrivKey: privKey,
			PubKey:  pubKey,
			TunIP:   wgPeerTunIP,
		}
		result = dbSession.Create(&wgPeerModels)

	} else {
		wgKeysModel := &models.WGKeys{
			PrivKey: privKey,
			PubKey:  pubKey,
		}
		result = dbSession.Create(&wgKeysModel)
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.WireGuardNewPeer,
		Data: []byte(fmt.Sprintf("public_key=%s\nallowed_ip=%s/32\n", pubKey, wgPeerTunIP)),
	})


	return result.Error
}
