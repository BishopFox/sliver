package db

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/gorm"
)

const (
	C2WireGuardIPCIDR          = "100.64.0.1/16"
	MultiplayerWireGuardIPCIDR = "100.65.0.1/16"
	maxWGIPReservationAttempts = 32
)

var (
	reservedC2WGIPs          = []string{"100.64.0.0", "100.64.0.1"}
	reservedMultiplayerWGIPs = []string{"100.65.0.0", "100.65.0.1"}
)

// NextAvailableWGIP returns the next unassigned WireGuard tunnel IP candidate.
func NextAvailableWGIP() (string, error) {
	return nextAvailableWGIP(C2WireGuardIPCIDR, reservedC2WGIPs)
}

// NextAvailableMultiplayerWGIP returns the next unassigned multiplayer
// WireGuard tunnel IP candidate.
func NextAvailableMultiplayerWGIP() (string, error) {
	return nextAvailableWGIP(MultiplayerWireGuardIPCIDR, reservedMultiplayerWGIPs)
}

// IsC2WireGuardIP reports whether the tunnel IP belongs to the C2 WireGuard
// address space.
func IsC2WireGuardIP(tunIP string) bool {
	return ipInCIDR(tunIP, C2WireGuardIPCIDR)
}

// IsMultiplayerWireGuardIP reports whether the tunnel IP belongs to the
// multiplayer WireGuard address space.
func IsMultiplayerWireGuardIP(tunIP string) bool {
	return ipInCIDR(tunIP, MultiplayerWireGuardIPCIDR)
}

func nextAvailableWGIP(cidr string, reserved []string) (string, error) {
	allocatedIPs, err := WGPeerIPs()
	if err != nil {
		return "", err
	}

	addressPool, err := wgHosts(cidr, reserved)
	if err != nil {
		return "", err
	}

	for _, address := range addressPool {
		inUse := false
		for _, ip := range allocatedIPs {
			if ip == address {
				inUse = true
				break
			}
		}
		if !inUse {
			return address, nil
		}
	}
	return "", fmt.Errorf("no available wireguard tunnel IPs remain in %s", cidr)
}

// ReserveNextAvailableWGIP retries allocation until it can reserve a unique
// tunnel IP at the database layer.
func ReserveNextAvailableWGIP(ownerType string, ownerID string) (string, error) {
	for attempt := 0; attempt < maxWGIPReservationAttempts; attempt++ {
		tunIP, err := NextAvailableWGIP()
		if err != nil {
			return "", err
		}
		err = ReserveWGIP(tunIP, ownerType, ownerID)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			continue
		}
		if err != nil {
			return "", err
		}
		return tunIP, nil
	}
	return "", fmt.Errorf("failed to reserve a unique wireguard tunnel IP after %d attempts", maxWGIPReservationAttempts)
}

// ReserveWGIP persists a unique WireGuard tunnel IP reservation.
func ReserveWGIP(tunIP string, ownerType string, ownerID string) error {
	return Session().Transaction(func(tx *gorm.DB) error {
		return ReserveWGIPTx(tx, tunIP, ownerType, ownerID)
	})
}

// ReserveWGIPTx persists a unique WireGuard tunnel IP reservation using an
// existing transaction.
func ReserveWGIPTx(tx *gorm.DB, tunIP string, ownerType string, ownerID string) error {
	if tx == nil {
		return errors.New("transaction is required")
	}

	tunIP = strings.TrimSpace(tunIP)
	ownerType = strings.TrimSpace(ownerType)
	ownerID = strings.TrimSpace(ownerID)
	if tunIP == "" {
		return errors.New("wireguard tunnel IP is required")
	}
	if ownerType == "" {
		return errors.New("wireguard tunnel IP owner type is required")
	}
	if ownerID == "" {
		return errors.New("wireguard tunnel IP owner id is required")
	}
	if net.ParseIP(tunIP) == nil {
		return fmt.Errorf("invalid wireguard tunnel IP %q", tunIP)
	}

	if err := ensureWGIPNotAllocated(tx, tunIP); err != nil {
		return err
	}

	err := tx.Create(&models.WGIPReservation{
		TunIP:     tunIP,
		OwnerType: ownerType,
		OwnerID:   ownerID,
	}).Error
	if isDuplicateWGIPReservationError(err) {
		return fmt.Errorf("%w: wireguard tunnel IP %s is already reserved", gorm.ErrDuplicatedKey, tunIP)
	}
	return err
}

// ReleaseWGIP deletes a reservation for a WireGuard tunnel IP.
func ReleaseWGIP(tunIP string) error {
	tunIP = strings.TrimSpace(tunIP)
	if tunIP == "" {
		return nil
	}
	return Session().Where(&models.WGIPReservation{TunIP: tunIP}).Delete(&models.WGIPReservation{}).Error
}

func ensureWGIPNotAllocated(tx *gorm.DB, tunIP string) error {
	operator := &models.Operator{}
	result := tx.Select("id").Where("wg_tun_ip = ?", tunIP).First(operator)
	if result.Error == nil {
		return fmt.Errorf("%w: wireguard tunnel IP %s is already assigned to an operator", gorm.ErrDuplicatedKey, tunIP)
	}
	if result.Error != nil && !errors.Is(result.Error, ErrRecordNotFound) {
		return result.Error
	}

	peer := &models.WGPeer{}
	result = tx.Select("id").Where("tun_ip = ?", tunIP).First(peer)
	if result.Error == nil {
		return fmt.Errorf("%w: wireguard tunnel IP %s is already assigned to a peer", gorm.ErrDuplicatedKey, tunIP)
	}
	if result.Error != nil && !errors.Is(result.Error, ErrRecordNotFound) {
		return result.Error
	}

	reservation := &models.WGIPReservation{}
	result = tx.Select("id").Where("tun_ip = ?", tunIP).First(reservation)
	if result.Error == nil {
		return fmt.Errorf("%w: wireguard tunnel IP %s is already reserved", gorm.ErrDuplicatedKey, tunIP)
	}
	if result.Error != nil && !errors.Is(result.Error, ErrRecordNotFound) {
		return result.Error
	}
	return nil
}

func wgHosts(cidr string, reserved []string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementWGIP(ip) {
		ips = append(ips, ip.String())
	}

	return removeWGIPs(ips, reserved), nil
}

func ipInCIDR(tunIP string, cidr string) bool {
	parsedIP := net.ParseIP(strings.TrimSpace(tunIP))
	if parsedIP == nil {
		return false
	}
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipnet.Contains(parsedIP)
}

func incrementWGIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func removeWGIPs(stringSlice []string, remove []string) []string {
	var result []string
	for _, v := range stringSlice {
		shouldAppend := true
		for _, value := range remove {
			if v == value {
				shouldAppend = false
				break
			}
		}
		if shouldAppend {
			result = append(result, v)
		}
	}
	return result
}

func isDuplicateWGIPReservationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	errString := strings.ToLower(err.Error())
	return strings.Contains(errString, "duplicate key") ||
		strings.Contains(errString, "duplicated key") ||
		strings.Contains(errString, "unique constraint failed") ||
		strings.Contains(errString, "duplicate entry")
}
