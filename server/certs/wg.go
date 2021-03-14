package certs

import (
	"encoding/hex"
	"errors"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

var (
	wgKeysLog = log.NamedLogger("certs", "wg-keys")

	ErrWGPeerDoesNotExist     = errors.New("WG peer does not exist")
	ErrWGServerKeysDoNotExist = errors.New("WG server keys do not exist")
)

func SetupWGKeys() {
	if _, _, err := GetWGServerKeys(); err != nil {
		wgKeysLog.Infof("No wg server keys detected")
		GenerateWGKeys(false, "")
	}
}

// GetWGSPeers - Get the WG peers
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

// GetWGServerKeys - Get the WG server keys
func GetWGServerKeys() (string, string, error) {

	wgKeysLog.Infof("Getting WG keys for tun server")

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

func GenerateWGKeys(isPeer bool, wgPeerTunIP string) (string, string, error) {
	privKey, pubKey := genWGKeys()

	if err := saveWGKeys(isPeer, wgPeerTunIP, privKey, pubKey); err != nil {
		wgKeysLog.Error("Error Saving WG keys: ", err)
		return "", "", err
	}
	return privKey, pubKey, nil
}

func genWGKeys() (string, string) {
	wgKeysLog.Infof("Generating WG keys")

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		wgKeysLog.Fatalf("Failed to generate private key: %s", err)
	}
	publicKey := privateKey.PublicKey()
	return hex.EncodeToString(privateKey[:]), hex.EncodeToString(publicKey[:])
}

// saveWGKeys - Save WG keys to the database
func saveWGKeys(isPeer bool, wgPeerTunIP string, privKey string, pubKey string) error {

	wgKeysLog.Infof("Saving WG keys")
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

	return result.Error
}
