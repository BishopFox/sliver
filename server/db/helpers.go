package db

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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
	----------------------------------------------------------------------

    IMPORTANT: These should be read-only functions and cannot rely on any
               packages outside of /server/db/models

*/

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

var (
	// ErrRecordNotFound - Record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound
)

// ImplantConfigByID - Fetch implant build by name
func ImplantConfigByID(id string) (*models.ImplantConfig, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	configID := uuid.FromStringOrNil(id)
	if configID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	config := models.ImplantConfig{}
	err := Session().Where(&models.ImplantConfig{
		ID: configID,
	}).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, err
}

// ImplantConfigWithC2sByID - Fetch implant build by name
func ImplantConfigWithC2sByID(id string) (*models.ImplantConfig, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	configID := uuid.FromStringOrNil(id)
	if configID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	config := models.ImplantConfig{}
	err := Session().Where(&models.ImplantConfig{
		ID: configID,
	}).First(&config).Error
	if err != nil {
		return nil, err
	}

	c2s := []models.ImplantC2{}
	err = Session().Where(&models.ImplantC2{
		ImplantConfigID: config.ID,
	}).Find(&c2s).Error
	if err != nil {
		return nil, err
	}
	config.C2 = c2s
	return &config, err
}

// ImplantConfigByPublicKeyDigest - Fetch implant build by it's ecc public key
func ImplantConfigByPublicKeyDigest(publicKeyDigest [32]byte) (*models.ImplantConfig, error) {
	config := models.ImplantConfig{}
	err := Session().Where(&models.ImplantConfig{
		PeerPublicKeyDigest: hex.EncodeToString(publicKeyDigest[:]),
	}).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, err
}

// ImplantBuilds - Return all implant builds
func ImplantBuilds() ([]*models.ImplantBuild, error) {
	builds := []*models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{}).Preload("ImplantConfig").Find(&builds).Error
	if err != nil {
		return nil, err
	}
	for _, build := range builds {
		err = loadC2s(&build.ImplantConfig)
		if err != nil {
			return nil, err
		}
	}
	return builds, err
}

// ImplantBuildByName - Fetch implant build by name
func ImplantBuildByName(name string) (*models.ImplantBuild, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	build := models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{
		Name: name,
	}).Preload("ImplantConfig").First(&build).Error
	if err != nil {
		return nil, err
	}
	err = loadC2s(&build.ImplantConfig)
	if err != nil {
		return nil, err
	}
	return &build, err
}

// ImplantBuildNames - Fetch a list of all build names
func ImplantBuildNames() ([]string, error) {
	builds := []*models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{}).Find(&builds).Error
	if err != nil {
		return []string{}, err
	}
	names := []string{}
	for _, build := range builds {
		names = append(names, build.Name)
	}
	return names, nil
}

// ImplantProfiles - Fetch a map of name<->profiles current in the database
func ImplantProfiles() ([]*models.ImplantProfile, error) {
	profiles := []*models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{}).Preload("ImplantConfig").Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		err = loadC2s(profile.ImplantConfig)
		if err != nil {
			return nil, err
		}
	}
	return profiles, nil
}

// ImplantProfileByName - Fetch implant build by name
func ImplantProfileByName(name string) (*models.ImplantProfile, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	profile := models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{
		Name: name,
	}).Preload("ImplantConfig").First(&profile).Error
	if err != nil {
		return nil, err
	}
	err = loadC2s(profile.ImplantConfig)
	if err != nil {
		return nil, err
	}
	return &profile, err
}

// C2s are not eager-loaded, this will load them for a given ImplantConfig
// I wasn't able to get GORM's nested loading to work, so I went with this.
func loadC2s(config *models.ImplantConfig) error {
	c2s := []models.ImplantC2{}
	err := Session().Where(&models.ImplantC2{
		ImplantConfigID: config.ID,
	}).Find(&c2s).Error
	if err != nil {
		return err
	}
	config.C2 = c2s
	return nil
}

// ImplantProfileNames - Fetch a list of all build names
func ImplantProfileNames() ([]string, error) {
	profiles := []*models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{}).Find(&profiles).Error
	if err != nil {
		return []string{}, err
	}
	names := []string{}
	for _, build := range profiles {
		names = append(names, build.Name)
	}
	return names, nil
}

// ProfileByName - Fetch a single profile from the database
func ProfileByName(name string) (*models.ImplantProfile, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	dbProfile := &models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{Name: name}).Find(&dbProfile).Error
	return dbProfile, err
}

// ListCanaries - List of all embedded canaries
func ListCanaries() ([]*models.DNSCanary, error) {
	canaries := []*models.DNSCanary{}
	err := Session().Where(&models.DNSCanary{}).Find(&canaries).Error
	return canaries, err
}

// CanaryByDomain - Check if a canary exists
func CanaryByDomain(domain string) (*models.DNSCanary, error) {
	if len(domain) < 1 {
		return nil, ErrRecordNotFound
	}
	dbSession := Session()
	canary := models.DNSCanary{}
	err := dbSession.Where(&models.DNSCanary{Domain: domain}).First(&canary).Error
	return &canary, err
}

// WebsiteByName - Get website by name
func WebsiteByName(name string) (*models.Website, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	website := models.Website{}
	err := Session().Where(&models.Website{Name: name}).First(&website).Error
	if err != nil {
		return nil, err
	}
	return &website, nil
}

// WGPeerIPs - Fetch a list of ips for all wireguard peers
func WGPeerIPs() ([]string, error) {
	wgPeers := []*models.WGPeer{}
	err := Session().Where(&models.WGPeer{}).Find(&wgPeers).Error
	if err != nil {
		return nil, err
	}
	ips := []string{}
	for _, peer := range wgPeers {
		ips = append(ips, peer.TunIP)
	}
	return ips, nil
}

// ListHosts - List of all hosts in the database
func ListHosts() ([]*models.Host, error) {
	hosts := []*models.Host{}
	err := Session().Where(
		&models.Host{},
	).Preload("IOCs").Preload("ExtensionData").Find(&hosts).Error
	return hosts, err
}

// HostByHostID - Get host by the session's reported HostUUID
func HostByHostID(id uuid.UUID) (*models.Host, error) {
	host := models.Host{}
	err := Session().Where(&models.Host{ID: id}).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// HostByHostUUID - Get host by the session's reported HostUUID
func HostByHostUUID(id string) (*models.Host, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	hostID := uuid.FromStringOrNil(id)
	if hostID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	host := models.Host{}
	err := Session().Where(
		&models.Host{HostUUID: hostID},
	).Preload("IOCs").Preload("ExtensionData").First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// IOCByID - Select an IOC by ID
func IOCByID(id string) (*models.IOC, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	ioc := &models.IOC{}
	iocID := uuid.FromStringOrNil(id)
	if iocID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	err := Session().Where(
		&models.IOC{ID: iocID},
	).First(ioc).Error
	return ioc, err
}

// BeaconByID - Select a Beacon by ID
func BeaconByID(id string) (*models.Beacon, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	beaconID := uuid.FromStringOrNil(id)
	if beaconID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	beacon := &models.Beacon{}
	err := Session().Where(
		&models.Beacon{ID: beaconID},
	).First(beacon).Error
	return beacon, err
}

// BeaconTasksByBeaconID - Get all tasks for a specific beacon
// by default will not fetch the request/response columns since
// these could be arbitrarily large.
func BeaconTasksByBeaconID(beaconID string) ([]*models.BeaconTask, error) {
	if len(beaconID) < 1 {
		return nil, ErrRecordNotFound
	}
	id := uuid.FromStringOrNil(beaconID)
	if id == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	beaconTasks := []*models.BeaconTask{}
	err := Session().Select([]string{
		"ID", "EnvelopeID", "BeaconID", "CreatedAt", "State", "SentAt", "CompletedAt",
		"Description",
	}).Where(&models.BeaconTask{BeaconID: id}).Find(&beaconTasks).Error
	return beaconTasks, err
}

// BeaconTaskByID - Select a specific BeaconTask by ID, this
// will fetch the full request/response
func BeaconTaskByID(id string) (*models.BeaconTask, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	taskID := uuid.FromStringOrNil(id)
	if taskID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	task := &models.BeaconTask{}
	err := Session().Where(
		&models.BeaconTask{ID: taskID},
	).First(task).Error
	return task, err
}

// ListBeacons - Select a Beacon by ID
func ListBeacons() ([]*models.Beacon, error) {
	beacons := []*models.Beacon{}
	err := Session().Where(&models.Beacon{}).Find(&beacons).Error
	return beacons, err
}

// RenameBeacon - Rename a beacon
func RenameBeacon(id string, name string) error {
	if len(id) < 1 {
		return ErrRecordNotFound
	}
	beaconID := uuid.FromStringOrNil(id)
	if beaconID == uuid.Nil {
		return ErrRecordNotFound
	}
	err := Session().Where(&models.Beacon{
		ID: beaconID,
	}).Updates(models.Beacon{Name: name}).Error
	if err != nil {
		return err
	}
	return nil
}

// PendingBeaconTasksByBeaconID - Select a Beacon by ID, ordered by creation time
func PendingBeaconTasksByBeaconID(id string) ([]*models.BeaconTask, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	beaconID := uuid.FromStringOrNil(id)
	if beaconID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	tasks := []*models.BeaconTask{}
	err := Session().Where(
		&models.BeaconTask{
			BeaconID: beaconID,
			State:    models.PENDING,
		},
	).Order("created_at").Find(&tasks).Error
	return tasks, err
}

// UpdateBeaconCheckinByID - Update the beacon's last / next checkin
func UpdateBeaconCheckinByID(id string, next int64) error {
	if len(id) < 1 {
		return ErrRecordNotFound
	}
	beaconID := uuid.FromStringOrNil(id)
	if beaconID == uuid.Nil {
		return ErrRecordNotFound
	}
	err := Session().Where(&models.Beacon{
		ID: beaconID,
	}).Updates(models.Beacon{
		LastCheckin: time.Now(),
		NextCheckin: time.Now().Unix() + next,
	}).Error
	return err
}

// BeaconTasksByEnvelopeID - Select a (sent) BeaconTask by its envelope ID
func BeaconTaskByEnvelopeID(beaconID string, envelopeID int64) (*models.BeaconTask, error) {
	if len(beaconID) < 1 {
		return nil, ErrRecordNotFound
	}
	beaconUUID := uuid.FromStringOrNil(beaconID)
	if beaconUUID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	task := &models.BeaconTask{}
	err := Session().Where(
		&models.BeaconTask{
			BeaconID:   beaconUUID,
			EnvelopeID: envelopeID,
			State:      models.SENT,
		},
	).First(task).Error
	return task, err
}

// CountTasksByBeaconID - Select a (sent) BeaconTask by its envelope ID
func CountTasksByBeaconID(beaconID uuid.UUID) (int64, int64, error) {
	if beaconID == uuid.Nil {
		return 0, 0, ErrRecordNotFound
	}
	allTasks := int64(0)
	completedTasks := int64(0)
	err := Session().Model(&models.BeaconTask{}).Where(
		&models.BeaconTask{
			BeaconID: beaconID,
		},
	).Count(&allTasks).Error
	if err != nil {
		return 0, 0, err
	}
	err = Session().Model(&models.BeaconTask{}).Where(
		&models.BeaconTask{
			BeaconID: beaconID,
			State:    models.COMPLETED,
		},
	).Count(&completedTasks).Error
	return allTasks, completedTasks, err
}

// OperatorByToken - Select an operator by token value
func OperatorByToken(value string) (*models.Operator, error) {
	if len(value) < 1 {
		return nil, ErrRecordNotFound
	}
	operator := &models.Operator{}
	err := Session().Where(&models.Operator{
		Token: value,
	}).First(operator).Error
	return operator, err
}

// OperatorAll - Select all operators from the database
func OperatorAll() ([]*models.Operator, error) {
	operators := []*models.Operator{}
	err := Session().Distinct("Name").Find(&operators).Error
	return operators, err
}

// GetKeyValue - Get a value from a key
func GetKeyValue(key string) (string, error) {
	keyValue := &models.KeyValue{}
	err := Session().Where(&models.KeyValue{
		Key: key,
	}).First(keyValue).Error
	return keyValue.Value, err
}

// SetKeyValue - Set the value for a key/value pair
func SetKeyValue(key string, value string) error {
	err := Session().Where(&models.KeyValue{
		Key: key,
	}).First(&models.KeyValue{}).Error
	if err == ErrRecordNotFound {
		err = Session().Create(&models.KeyValue{
			Key:   key,
			Value: value,
		}).Error
	} else {
		err = Session().Where(&models.KeyValue{
			Key: key,
		}).Updates(models.KeyValue{
			Key:   key,
			Value: value,
		}).Error
	}
	return err
}

// DeleteKeyValue - Delete a key/value pair
func DeleteKeyValue(key string, value string) error {
	return Session().Delete(&models.KeyValue{
		Key: key,
	}).Error
}

// CrackstationByHostUUID - Get crackstation by the session's reported HostUUID
func CrackstationByHostUUID(hostUUID string) (*models.Crackstation, error) {
	id := uuid.FromStringOrNil(hostUUID)
	if id == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	crackstation := models.Crackstation{}
	err := Session().Where(
		&models.Crackstation{ID: id},
	).Preload("Tasks").Preload("Benchmarks").First(&crackstation).Error
	if err != nil {
		return nil, err
	}
	return &crackstation, nil
}

// CredentialsByHashType
func CredentialsByHashType(hashType clientpb.HashType) ([]*models.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		HashType: int32(hashType),
	}).Find(&credentials).Error
	return credentials, err
}

// CredentialsByHashType
func CredentialsByCollection(collection string) ([]*models.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		Collection: collection,
	}).Find(&credentials).Error
	return credentials, err
}

// PlaintextCredentials
func PlaintextCredentialsByHashType(hashType clientpb.HashType) ([]*models.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		HashType: int32(hashType),
	}).Not("plaintext = ?", "").Find(&credentials).Error
	return credentials, err
}

// CredentialsByID
func CredentialByID(id string) (*models.Credential, error) {
	credential := &models.Credential{}
	credID := uuid.FromStringOrNil(id)
	if credID != uuid.Nil {
		err := Session().Where(&models.Credential{ID: credID}).First(&credential).Error
		return credential, err
	}
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{}).Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	for _, cred := range credentials {
		if strings.HasPrefix(cred.ID.String(), id) {
			return cred, nil
		}
	}
	return nil, ErrRecordNotFound
}

// GetCrackTaskByID - Get a crack task by its ID
func GetCrackTaskByID(id string) (*models.CrackTask, error) {
	taskID := uuid.FromStringOrNil(id)
	if taskID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	task := &models.CrackTask{}
	err := Session().Where(&models.CrackTask{ID: taskID}).Preload("Command").Find(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetByCrackFileByID - Get a crack task by its ID
func GetByCrackFileByID(id string) (*models.CrackFile, error) {
	crackFileID := uuid.FromStringOrNil(id)
	if crackFileID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	crackFile := &models.CrackFile{}
	err := Session().Where(&models.CrackFile{ID: crackFileID}).Preload("Chunks").Find(&crackFile).Error
	if err != nil {
		return nil, err
	}
	return crackFile, nil
}

// CrackFilesByType - Get all files by crack file type
func CrackFilesByType(fileType clientpb.CrackFileType) ([]*models.CrackFile, error) {
	crackFiles := []*models.CrackFile{}
	err := Session().Where(&models.CrackFile{
		Type:       int32(fileType),
		IsComplete: true,
	}).Preload("Chunks").Find(&crackFiles).Error
	if err != nil {
		return nil, err
	}
	return crackFiles, nil
}

func AllCrackFiles() ([]*models.CrackFile, error) {
	crackFiles := []*models.CrackFile{}
	err := Session().Preload("Chunks").Find(&crackFiles).Error
	if err != nil {
		return nil, err
	}
	return crackFiles, nil
}

// CrackWordlistByName - Get all files by crack file type
func CrackWordlistByName(name string) (*models.CrackFile, error) {
	crackFile := &models.CrackFile{}
	err := Session().Where(&models.CrackFile{
		Name: name,
		Type: int32(clientpb.CrackFileType_WORDLIST),
	}).First(&crackFile).Error
	if err != nil {
		return nil, err
	}
	return crackFile, nil
}

// CrackFilesDiskUsage - Get all files by crack file type
func CrackFilesDiskUsage() (int64, error) {
	crackFiles := []*models.CrackFile{}
	err := Session().Where(&models.CrackFile{}).Find(&crackFiles).Error
	if err != nil {
		return -1, err
	}
	sum := int64(0)
	for _, crackFile := range crackFiles {
		sum += crackFile.UncompressedSize
	}
	return sum, nil
}

// CheckKeyExReplay - Store the hash of a key exchange to prevent replays
func CheckKeyExReplay(ciphertext []byte) error {
	keyExSha256Hash := sha256.Sum256(ciphertext)
	return Session().Create(
		// The Sha256 is a primary key, do duplicates/nulls will
		// result in an error
		&models.KeyExHistory{
			Sha256: hex.EncodeToString(keyExSha256Hash[:]),
		},
	).Error
}
