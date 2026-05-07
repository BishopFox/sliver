package generate

import (
	"errors"
	"fmt"
	"net"

	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	wgipsLog = log.RootLogger.WithFields(logrus.Fields{
		"pkg":    "generate",
		"stream": "wgips",
	})
)

// GenerateUniqueIP generates and returns an available IP which can then
// be assigned to a WireGuard interface.
func GenerateUniqueIP() (net.IP, error) {
	tunIP, err := db.NextAvailableWGIP()
	if err != nil {
		wgipsLog.Errorf("Failed to generate WG peer IP with error: %s", err)
		return nil, err
	}
	return net.ParseIP(tunIP), nil
}

// GenerateUniqueWGPeerKeys allocates and persists a unique WireGuard peer.
func GenerateUniqueWGPeerKeys() (string, string, string, error) {
	for attempt := 0; attempt < 32; attempt++ {
		clientIP, err := GenerateUniqueIP()
		if err != nil {
			return "", "", "", err
		}

		privKey, pubKey, err := certs.GenerateWGKeys(true, clientIP.String())
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			continue
		}
		if err != nil {
			return "", "", "", err
		}
		return clientIP.String(), privKey, pubKey, nil
	}
	return "", "", "", fmt.Errorf("failed to allocate a unique wireguard peer after %d attempts", 32)
}
