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
	"net/netip"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

const (
	C2WireGuardServerIP          = "100.64.0.1"
	MultiplayerWireGuardServerIP = "100.65.0.1"
)

var (
	wgKeysLog = log.NamedLogger("certs", "wg-keys")

	ErrWGPeerDoesNotExist                = errors.New("wg peer does not exist")
	ErrWGServerKeysDoNotExist            = errors.New("wg server keys do not exist")
	ErrMultiplayerWGServerKeysDoNotExist = errors.New("multiplayer wg server keys do not exist")
)

// SetupWGKeys - Setup C2 WireGuard server keys.
func SetupWGKeys() {
	if _, _, err := GetWGServerKeys(); err != nil {
		wgKeysLog.Info("No wg server keys detected")
		GenerateWGKeys(false, "")
	}
}

// SetupMultiplayerWGKeys - Setup multiplayer WireGuard server keys.
func SetupMultiplayerWGKeys() {
	if _, _, err := GetMultiplayerWGServerKeys(); err != nil {
		wgKeysLog.Info("No multiplayer wg server keys detected")
		GenerateMultiplayerWGServerKeys()
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

// GenerateWGKeyPair - Generate a WireGuard keypair without persisting it.
func GenerateWGKeyPair() (string, string, error) {
	return genWGKeys()
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

// GetOperatorWGPeers - Get a map of operator WG public keys to tunnel IPs.
func GetOperatorWGPeers() (map[string]string, error) {
	peers := make(map[string]string)

	operators := []models.Operator{}
	dbSession := db.Session()
	err := dbSession.Where("wg_pub_key <> '' AND wg_tun_ip <> ''").Find(&operators).Error
	if errors.Is(err, db.ErrRecordNotFound) {
		return nil, ErrWGPeerDoesNotExist
	} else if err != nil {
		return nil, err
	}

	for _, operator := range operators {
		peers[operator.WGPubKey] = operator.WGTunIP
	}
	return peers, nil
}

// GetWGServerKeys - Get existing C2 WireGuard server keys.
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

// GetMultiplayerWGServerKeys - Get existing multiplayer WireGuard server keys.
func GetMultiplayerWGServerKeys() (string, string, error) {
	wgKeysLog.Info("Getting multiplayer wg keys for multiplayer listener")

	wgKeysModel := models.MultiplayerWGKeys{}
	dbSession := db.Session()
	result := dbSession.First(&wgKeysModel)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return "", "", ErrMultiplayerWGServerKeysDoNotExist
	}
	if result.Error != nil {
		return "", "", result.Error
	}

	return wgKeysModel.PrivKey, wgKeysModel.PubKey, nil
}

// GenerateWGKeys - Generates and saves new C2 WireGuard keys.
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

// GenerateMultiplayerWGServerKeys - Generates and saves dedicated multiplayer
// WireGuard server keys.
func GenerateMultiplayerWGServerKeys() (string, string, error) {
	privKey, pubKey, err := genWGKeys()
	if err != nil {
		return "", "", err
	}

	if err := saveMultiplayerWGServerKeys(privKey, pubKey); err != nil {
		wgKeysLog.Error("Error Saving multiplayer wg keys: ", err)
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

// saveWGKeys - Saves C2 WireGuard keys to the database.
func saveWGKeys(isPeer bool, wgPeerTunIP string, privKey string, pubKey string) error {

	wgKeysLog.Info("Saving wg keys")
	dbSession := db.Session()
	var err error

	if isPeer {
		wgPeerTunIP = strings.TrimSpace(wgPeerTunIP)
		if wgPeerTunIP == "" {
			return errors.New("wg peer tunnel IP cannot be empty")
		}
		if _, err := netip.ParseAddr(wgPeerTunIP); err != nil {
			return fmt.Errorf("invalid wg peer tunnel IP %q: %w", wgPeerTunIP, err)
		}
		if !db.IsC2WireGuardIP(wgPeerTunIP) {
			return fmt.Errorf("wg peer tunnel IP %q must be inside %s", wgPeerTunIP, db.C2WireGuardIPCIDR)
		}

		wgPeerModels := &models.WGPeer{
			PrivKey: privKey,
			PubKey:  pubKey,
			TunIP:   wgPeerTunIP,
		}
		err = dbSession.Transaction(func(tx *gorm.DB) error {
			if err := db.ReserveWGIPTx(tx, wgPeerTunIP, models.WGIPOwnerTypePeer, pubKey); err != nil {
				return err
			}
			return tx.Create(&wgPeerModels).Error
		})

	} else {
		wgKeysModel := &models.WGKeys{
			PrivKey: privKey,
			PubKey:  pubKey,
		}
		err = dbSession.Create(&wgKeysModel).Error
	}
	if err != nil {
		return err
	}
	if isPeer {
		core.EventBroker.Publish(core.Event{
			EventType: consts.WireGuardNewPeer,
			Data:      []byte(fmt.Sprintf("public_key=%s\nallowed_ip=%s/32\n", pubKey, wgPeerTunIP)),
		})
	}

	return nil
}

func saveMultiplayerWGServerKeys(privKey string, pubKey string) error {
	wgKeysLog.Info("Saving multiplayer wg keys")
	return db.Session().Create(&models.MultiplayerWGKeys{
		PrivKey: privKey,
		PubKey:  pubKey,
	}).Error
}
