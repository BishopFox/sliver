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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrRecordNotFound - Record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound
)

// ImplantConfigByID - Fetch implant config by id
func ImplantConfigByID(id string) (*clientpb.ImplantConfig, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	configID := uuid.FromStringOrNil(id)
	if configID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	config := models.ImplantConfig{}
	err := Session().Preload("C2").Preload("Assets").Preload("CanaryDomains").Where(&models.ImplantConfig{
		ID: configID,
	}).First(&config).Error
	if err != nil {
		return nil, err
	}
	return config.ToProtobuf(), err
}

// ImplantConfigWithC2sByID - Fetch implant build by name
func ImplantConfigWithC2sByID(id string) (*clientpb.ImplantConfig, error) {
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
	return config.ToProtobuf(), err
}

// ImplantBuildByPublicKeyDigest - Fetch implant build by it's ecc public key
func ImplantBuildByPublicKeyDigest(publicKeyDigest [32]byte) (*clientpb.ImplantBuild, error) {
	build := models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{
		PeerPublicKeyDigest: hex.EncodeToString(publicKeyDigest[:]),
	}).First(&build).Error
	if err != nil {
		return nil, err
	}
	return build.ToProtobuf(), err
}

// ImplantBuilds - Return all implant builds
func ImplantBuilds() (*clientpb.ImplantBuilds, error) {
	builds := []*models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{}).Find(&builds).Error
	if err != nil {
		return nil, err
	}
	pbBuilds := &clientpb.ImplantBuilds{
		Configs:     map[string]*clientpb.ImplantConfig{},
		ResourceIDs: map[string]*clientpb.ResourceID{},
		Staged:      map[string]bool{},
	}
	for _, dbBuild := range builds {
		config, err := ImplantConfigByID(dbBuild.ImplantConfigID.String())
		if err != nil {
			return nil, err
		}
		pbBuilds.Configs[dbBuild.Name] = config

		resource, err := ResourceIDByName(dbBuild.Name)
		if err != nil {
			return nil, err
		}
		pbBuilds.ResourceIDs[dbBuild.Name] = resource

		pbBuilds.Staged[dbBuild.Name] = dbBuild.Stage
	}

	return pbBuilds, err
}

// SaveImplantBuild
func SaveImplantBuild(ib *clientpb.ImplantBuild) (*clientpb.ImplantBuild, error) {
	dbSession := Session()
	var implantBuild *models.ImplantBuild
	if ib.ID != "" {
		_, err := ImplantBuildByID(ib.ID)
		if err != nil {
			return nil, err
		}
		implantBuild = models.ImplantBuildFromProtobuf(ib)
		err = dbSession.Save(implantBuild).Error
		if err != nil {
			return nil, err
		}
	} else {
		implantBuild = models.ImplantBuildFromProtobuf(ib)
		err := dbSession.Create(&implantBuild).Error
		if err != nil {
			return nil, err
		}

	}
	return implantBuild.ToProtobuf(), nil
}

// SaveImplantConfig
func SaveImplantConfig(ic *clientpb.ImplantConfig) (*clientpb.ImplantConfig, error) {
	dbSession := Session()
	var implantConfig *models.ImplantConfig
	if ic.ID != "" {
		_, err := ImplantConfigByID(ic.ID)
		if err != nil {
			return nil, err
		}
		implantConfig = models.ImplantConfigFromProtobuf(ic)
		err = dbSession.Save(implantConfig).Error
		if err != nil {
			return nil, err
		}
	} else {
		implantConfig = models.ImplantConfigFromProtobuf(ic)
		err := dbSession.Create(&implantConfig).Error
		if err != nil {
			return nil, err
		}
	}
	return implantConfig.ToProtobuf(), nil
}

// ImplantBuildByName - Fetch implant build by name
func ImplantBuildByName(name string) (*clientpb.ImplantBuild, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	build := models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{
		Name: name,
	}).First(&build).Error
	if err != nil {
		return nil, err
	}
	return build.ToProtobuf(), err
}

// ImplantBuildByResourceID - Fetch implant build from resource ID
func ImplantBuildByResourceID(resourceID uint64) (*clientpb.ImplantBuild, error) {
	build := models.ImplantBuild{}
	err := Session().Where(&models.ImplantBuild{
		ImplantID: resourceID,
	}).Find(&build).Error
	if err != nil {
		return nil, err
	}
	return build.ToProtobuf(), nil
}

// ImplantBuildByID - Fetch implant build from ID
func ImplantBuildByID(id string) (*clientpb.ImplantBuild, error) {
	build := models.ImplantBuild{}
	uuid, _ := uuid.FromString(id)
	err := Session().Where(&models.ImplantBuild{
		ID: uuid,
	}).Find(&build).Error
	if err != nil {
		return nil, err
	}
	return build.ToProtobuf(), nil
}

// ImplantProfiles - Fetch a map of name<->profiles current in the database
func ImplantProfiles() ([]*clientpb.ImplantProfile, error) {
	profiles := []*models.ImplantProfile{}
	err := Session().Where(&models.ImplantProfile{}).Preload("ImplantConfig").Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	pbProfiles := []*clientpb.ImplantProfile{}
	for _, profile := range profiles {
		pbProfile := profile.ToProtobuf()
		err = loadC2s(pbProfile.Config)
		if err != nil {
			return nil, err
		}
		pbProfiles = append(pbProfiles, pbProfile)
	}
	return pbProfiles, nil
}

// ImplantProfileByName - Fetch implant build by name
func ImplantProfileByName(name string) (*clientpb.ImplantProfile, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	profile := models.ImplantProfile{}
	config := models.ImplantConfig{}
	err := Session().Where(&models.ImplantProfile{
		Name: name,
	}).First(&profile).Error
	if err != nil {
		return nil, err
	}
	err = Session().Where(models.ImplantConfig{
		ImplantProfileID: profile.ID,
	}).First(&config).Error
	if err != nil {
		return nil, err
	}

	profile.ImplantConfig = &config
	pbProfile := profile.ToProtobuf()

	err = loadC2s(pbProfile.Config)
	if err != nil {
		return nil, err
	}
	return pbProfile, err
}

// load c2 for a given implant config
func loadC2s(config *clientpb.ImplantConfig) error {
	id, _ := uuid.FromString(config.ID)
	c2s := []models.ImplantC2{}
	err := Session().Where(&models.ImplantC2{
		ImplantConfigID: id,
	}).Find(&c2s).Error
	if err != nil {
		return err
	}
	var implantC2 []*clientpb.ImplantC2
	for _, c2 := range c2s {
		implantC2 = append(implantC2, c2.ToProtobuf())
	}
	config.C2 = implantC2
	return nil
}

func LoadHTTPC2s() ([]*clientpb.HTTPC2Config, error) {
	c2Configs := []models.HttpC2Config{}
	err := Session().Where(&models.HttpC2Config{}).Find(&c2Configs).Error
	if err != nil {
		return nil, err
	}
	pbC2Configs := []*clientpb.HTTPC2Config{}
	for _, c2config := range c2Configs {
		pbC2Configs = append(pbC2Configs, c2config.ToProtobuf())
	}

	return pbC2Configs, nil
}

// used to prevent duplicate stager extensions
func SearchStageExtensions(stagerExtension string, profileName string) error {
	c2Config := models.HttpC2ImplantConfig{}
	err := Session().Where(&models.HttpC2ImplantConfig{
		StagerFileExtension: stagerExtension,
	}).Find(&c2Config).Error

	if err != nil {
		return err
	}

	if c2Config.StagerFileExtension != "" && profileName != "" {
		// check if the stager extension is used in the provided profile
		httpC2Config := models.HttpC2Config{}
		err = Session().Where(&models.HttpC2Config{ID: c2Config.HttpC2ConfigID}).Find(&httpC2Config).Error
		if err != nil {
			return err
		}
		if httpC2Config.Name == profileName {
			return nil
		}
		return configs.ErrDuplicateStageExt
	}
	return nil
}

func LoadHTTPC2ConfigByName(name string) (*clientpb.HTTPC2Config, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}

	c2Config := models.HttpC2Config{}
	err := Session().Where(&models.HttpC2Config{
		Name: name,
	}).Find(&c2Config).Error

	if err != nil {
		return nil, err
	}

	// load implant configuration
	c2ImplantConfig := models.HttpC2ImplantConfig{}
	err = Session().Where(&models.HttpC2ImplantConfig{
		HttpC2ConfigID: c2Config.ID,
	}).Find(&c2ImplantConfig).Error
	if err != nil {
		return nil, err
	}

	// load url parameters
	c2UrlParameters := []models.HttpC2URLParameter{}
	err = Session().Where(&models.HttpC2URLParameter{
		HttpC2ImplantConfigID: c2ImplantConfig.ID,
	}).Find(&c2UrlParameters).Error
	if err != nil {
		return nil, err
	}

	// load headers
	c2ImplantHeaders := []models.HttpC2Header{}
	err = Session().Where(&models.HttpC2Header{
		HttpC2ImplantConfigID: c2ImplantConfig.ID,
	}).Find(&c2ImplantHeaders).Error
	if err != nil {
		return nil, err
	}

	// load path segments
	c2PathSegments := []models.HttpC2PathSegment{}
	err = Session().Where(&models.HttpC2PathSegment{
		HttpC2ImplantConfigID: c2ImplantConfig.ID,
	}).Find(&c2PathSegments).Error
	if err != nil {
		return nil, err
	}
	c2ImplantConfig.ExtraURLParameters = c2UrlParameters
	c2ImplantConfig.Headers = c2ImplantHeaders
	c2ImplantConfig.PathSegments = c2PathSegments

	// load server configuration
	c2ServerConfig := models.HttpC2ServerConfig{}
	err = Session().Where(&models.HttpC2ServerConfig{
		HttpC2ConfigID: c2Config.ID,
	}).Find(&c2ServerConfig).Error
	if err != nil {
		return nil, err
	}

	// load headers
	c2ServerHeaders := []models.HttpC2Header{}
	err = Session().Where(&models.HttpC2Header{
		HttpC2ServerConfigID: c2ServerConfig.ID,
	}).Find(&c2ServerHeaders).Error
	if err != nil {
		return nil, err
	}

	// load cookies
	c2ServerCookies := []models.HttpC2Cookie{}
	err = Session().Where(&models.HttpC2Cookie{
		HttpC2ServerConfigID: c2ServerConfig.ID,
	}).Find(&c2ServerCookies).Error
	if err != nil {
		return nil, err
	}

	c2ServerConfig.Headers = c2ServerHeaders
	c2ServerConfig.Cookies = c2ServerCookies

	c2Config.ServerConfig = c2ServerConfig
	c2Config.ImplantConfig = c2ImplantConfig

	return c2Config.ToProtobuf(), nil
}

func SaveHTTPC2Config(httpC2Config *clientpb.HTTPC2Config) error {
	httpC2ConfigModel := models.HTTPC2ConfigFromProtobuf(httpC2Config)
	dbSession := Session()
	result := dbSession.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&httpC2ConfigModel)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func HTTPC2ConfigUpdate(newConf *clientpb.HTTPC2Config, oldConf *clientpb.HTTPC2Config) error {
	clientID, _ := uuid.FromString(oldConf.ImplantConfig.ID)
	c2Config := models.HTTPC2ConfigFromProtobuf(newConf)
	err := Session().Where(&models.ImplantConfig{
		ID: clientID,
	}).Updates(c2Config.ImplantConfig)
	if err != nil {
		return err.Error
	}

	serverID, _ := uuid.FromString(oldConf.ImplantConfig.ID)
	err = Session().Where(&models.HttpC2ServerConfig{
		ID: serverID,
	}).Updates(c2Config.ServerConfig)
	if err != nil {
		return err.Error
	}
	return nil
}

func SaveHTTPC2Listener(listenerConf *clientpb.ListenerJob) error {
	dbListener := models.ListenerJobFromProtobuf(listenerConf)
	dbSession := Session()
	result := dbSession.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&dbListener)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func UpdateHTTPC2Listener(listenerConf *clientpb.ListenerJob) error {
	dbListener := models.ListenerJobFromProtobuf(listenerConf)
	dbSession := Session()
	result := dbSession.Save(dbListener)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func ListenerByJobID(JobID uint32) (*clientpb.ListenerJob, error) {
	listenerJob := models.ListenerJob{}
	err := Session().Where(&models.ListenerJob{JobID: JobID}).Find(&listenerJob).Error

	switch listenerJob.Type {
	case constants.HttpStr:
		HttpListener := models.HTTPListener{}
		err = Session().Where(&models.HTTPListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&HttpListener).Error
		listenerJob.HttpListener = HttpListener
	case constants.HttpsStr:
		HttpListener := models.HTTPListener{}
		err = Session().Where(&models.HTTPListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&HttpListener).Error
		listenerJob.HttpListener = HttpListener
	case constants.DnsStr:
		DnsListener := models.DNSListener{}
		err = Session().Where(&models.DNSListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&DnsListener).Error
		listenerJob.DnsListener = DnsListener
	case constants.MtlsStr:
		MtlsListener := models.MtlsListener{}
		err = Session().Where(&models.MtlsListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&MtlsListener).Error
		listenerJob.MtlsListener = MtlsListener
	case constants.WGStr:
		WGListener := models.WGListener{}
		err = Session().Where(&models.WGListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&WGListener).Error
		listenerJob.WgListener = WGListener
	case constants.MultiplayerModeStr:
		MultiplayerListener := models.MultiplayerListener{}
		err = Session().Where(&models.MultiplayerListener{
			ListenerJobID: listenerJob.ID,
		}).Find(&MultiplayerListener).Error
		listenerJob.MultiplayerListener = MultiplayerListener
	}

	if err != nil {
		return nil, err
	}

	return listenerJob.ToProtobuf(), err
}

func ListenerJobs() ([]*clientpb.ListenerJob, error) {
	listenerJobs := []models.ListenerJob{}
	err := Session().Where(&models.ListenerJob{}).Find(&listenerJobs).Error
	pbListenerJobs := []*clientpb.ListenerJob{}
	for _, listenerJob := range listenerJobs {
		pbListenerJobs = append(pbListenerJobs, listenerJob.ToProtobuf())
	}

	return pbListenerJobs, err
}

func DeleteListener(JobID uint32) error {
	return Session().Where(&models.ListenerJob{JobID: JobID}).Delete(&models.ListenerJob{}).Error
}

func DeleteC2(c2ID uuid.UUID) error {
	return Session().Where(&models.ImplantC2{ID: c2ID}).Delete(&models.ImplantC2{}).Error
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
func ProfileByName(name string) (*clientpb.ImplantProfile, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	dbProfile := &models.ImplantProfile{}
	err := Session().Preload("ImplantConfig").Where(&models.ImplantProfile{Name: name}).Find(dbProfile).Error

	dbBuilds := []*models.ImplantBuild{}
	err = Session().Where(&models.ImplantBuild{ImplantConfigID: dbProfile.ImplantConfig.ID}).Find(&dbBuilds).Error
	builds := []*clientpb.ImplantBuild{}
	for _, build := range dbBuilds {
		builds = append(builds, build.ToProtobuf())
	}
	pbProfile := dbProfile.ToProtobuf()
	pbProfile.Config.ImplantBuilds = builds

	return pbProfile, err
}

// DeleteProfile - Delete a profile from the database
func DeleteProfile(name string) error {
	profile, err := ProfileByName(name)
	if err != nil {
		return err
	}

	uuid, _ := uuid.FromString(profile.Config.ID)

	// delete linked ImplantConfig
	err = Session().Where(&models.ImplantConfig{ID: uuid}).Delete(&models.ImplantConfig{}).Error

	// delete profile
	err = Session().Where(&models.ImplantProfile{Name: name}).Delete(&models.ImplantProfile{}).Error
	return err
}

// ListCanaries - List of all embedded canaries
func ListCanaries() ([]*clientpb.DNSCanary, error) {
	canaries := []*models.DNSCanary{}
	err := Session().Where(&models.DNSCanary{}).Find(&canaries).Error
	pbCanaries := []*clientpb.DNSCanary{}
	for _, canary := range canaries {
		pbCanaries = append(pbCanaries, canary.ToProtobuf())
	}

	return pbCanaries, err
}

// CanaryByDomain - Check if a canary exists
func CanaryByDomain(domain string) (*clientpb.DNSCanary, error) {
	if len(domain) < 1 {
		return nil, ErrRecordNotFound
	}
	dbSession := Session()
	canary := models.DNSCanary{}
	err := dbSession.Where(&models.DNSCanary{Domain: domain}).First(&canary).Error
	return canary.ToProtobuf(), err
}

// WebsiteByName - Get website by name
func WebsiteByName(name string, webContentDir string) (*clientpb.Website, error) {
	if len(name) < 1 {
		return nil, ErrRecordNotFound
	}
	website := models.Website{}
	err := Session().Where(&models.Website{Name: name}).Preload("WebContents").First(&website).Error
	if err != nil {
		return nil, err
	}
	return website.ToProtobuf(webContentDir), nil
}

// Websites - Return all websites
func Websites(webContentDir string) ([]*clientpb.Website, error) {
	websites := []*models.Website{}
	err := Session().Where(&models.Website{}).Find(&websites).Error

	var pbWebsites []*clientpb.Website
	for _, website := range websites {
		pbWebsites = append(pbWebsites, website.ToProtobuf(webContentDir))
	}

	return pbWebsites, err
}

// WebContent by ID and path
func WebContentByIDAndPath(id string, path string, webContentDir string, eager bool) (*clientpb.WebContent, error) {
	uuid, _ := uuid.FromString(id)
	content := models.WebContent{}
	err := Session().Where(&models.WebContent{
		WebsiteID: uuid,
		Path:      path,
	}).First(&content).Error

	if err != nil {
		return nil, err
	}
	var data []byte
	if eager {
		data, err = os.ReadFile(filepath.Join(webContentDir, content.ID.String()))
	} else {
		data = []byte{}
	}
	return content.ToProtobuf(&data), err
}

// AddWebsite - Return website, create if it does not exist
func AddWebSite(webSiteName string, webContentDir string) (*clientpb.Website, error) {
	pbWebSite, err := WebsiteByName(webSiteName, webContentDir)
	if errors.Is(err, ErrRecordNotFound) {
		err = Session().Create(&models.Website{Name: webSiteName}).Error
		if err != nil {
			return nil, err
		}
		pbWebSite, err = WebsiteByName(webSiteName, webContentDir)
		if err != nil {
			return nil, err
		}
	}
	return pbWebSite, nil
}

// AddContent - Add content to website
func AddContent(pbWebContent *clientpb.WebContent, webContentDir string) (*clientpb.WebContent, error) {
	dbWebContent, err := WebContentByIDAndPath(pbWebContent.WebsiteID, pbWebContent.Path, webContentDir, false)
	if errors.Is(err, ErrRecordNotFound) {
		dbModelWebContent := models.WebContentFromProtobuf(pbWebContent)
		err = Session().Create(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
		dbWebContent, err = WebContentByIDAndPath(pbWebContent.WebsiteID, pbWebContent.Path, webContentDir, false)
		if err != nil {
			return nil, err
		}
	} else {
		dbWebContent.ContentType = pbWebContent.ContentType
		dbWebContent.Size = pbWebContent.Size

		dbModelWebContent := models.WebContentFromProtobuf(dbWebContent)
		err = Session().Save(&dbModelWebContent).Error
		if err != nil {
			return nil, err
		}
	}
	return dbWebContent, nil
}

func RemoveContent(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.WebContent{}, uuid).Error
	return err
}

func RemoveWebSite(id string) error {
	uuid, _ := uuid.FromString(id)
	err := Session().Delete(&models.Website{}, uuid).Error
	return err
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
func ListHosts() ([]*clientpb.Host, error) {
	hosts := []*models.Host{}
	err := Session().Where(
		&models.Host{},
	).Preload("IOCs").Preload("ExtensionData").Find(&hosts).Error

	pbHosts := []*clientpb.Host{}
	for _, host := range hosts {
		pbHosts = append(pbHosts, host.ToProtobuf())
	}

	return pbHosts, err
}

// HostByHostID - Get host by the session's reported HostUUID
func HostByHostID(id uuid.UUID) (*clientpb.Host, error) {
	host := models.Host{}
	err := Session().Where(&models.Host{ID: id}).First(&host).Error
	if err != nil {
		return nil, err
	}
	return host.ToProtobuf(), nil
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
func IOCByID(id string) (*clientpb.IOC, error) {
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
	return ioc.ToProtobuf(), err
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
func BeaconTasksByBeaconID(beaconID string) ([]*clientpb.BeaconTask, error) {
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

	pbBeaconTasks := []*clientpb.BeaconTask{}
	for _, beaconTask := range beaconTasks {
		pbBeaconTasks = append(pbBeaconTasks, beaconTask.ToProtobuf(true))
	}
	return pbBeaconTasks, err
}

// BeaconTaskByID - Select a specific BeaconTask by ID, this
// will fetch the full request/response
func BeaconTaskByID(id string) (*clientpb.BeaconTask, error) {
	if len(id) < 1 {
		return nil, ErrRecordNotFound
	}
	taskID, err := uuid.FromString(id)
	if taskID == uuid.Nil {
		return nil, ErrRecordNotFound
	}
	task := &models.BeaconTask{}
	err = Session().Where(
		&models.BeaconTask{ID: taskID},
	).First(task).Error
	if err != nil {
		return nil, err
	}
	return task.ToProtobuf(true), err
}

// ListBeacons - Select a Beacon by ID
func ListBeacons() ([]*clientpb.Beacon, error) {
	beacons := []*models.Beacon{}
	err := Session().Where(&models.Beacon{}).Find(&beacons).Error

	pbBeacons := []*clientpb.Beacon{}
	for _, beacon := range beacons {
		pbBeacons = append(pbBeacons, beacon.ToProtobuf())
	}
	return pbBeacons, err
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
func BeaconTaskByEnvelopeID(beaconID string, envelopeID int64) (*clientpb.BeaconTask, error) {
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
	return task.ToProtobuf(true), err
}

// CountTasksByBeaconID - Select a (sent) BeaconTask by its envelope ID
func CountTasksByBeaconID(beaconID string) (int64, int64, error) {
	beaconUUID, _ := uuid.FromString(beaconID)
	if beaconUUID == uuid.Nil {
		return 0, 0, ErrRecordNotFound
	}
	allTasks := int64(0)
	completedTasks := int64(0)
	err := Session().Model(&models.BeaconTask{}).Where(
		&models.BeaconTask{
			BeaconID: beaconUUID,
		},
	).Count(&allTasks).Error
	if err != nil {
		return 0, 0, err
	}
	err = Session().Model(&models.BeaconTask{}).Where(
		&models.BeaconTask{
			BeaconID: beaconUUID,
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
func CredentialsByHashType(hashType clientpb.HashType) ([]*clientpb.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		HashType: int32(hashType),
	}).Find(&credentials).Error

	pbCredentials := []*clientpb.Credential{}
	for _, credential := range credentials {
		pbCredentials = append(pbCredentials, credential.ToProtobuf())
	}

	return pbCredentials, err
}

// CredentialsByHashType
func CredentialsByCollection(collection string) ([]*clientpb.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		Collection: collection,
	}).Find(&credentials).Error

	pbCredentials := []*clientpb.Credential{}
	for _, credential := range credentials {
		pbCredentials = append(pbCredentials, credential.ToProtobuf())
	}
	return pbCredentials, err
}

// PlaintextCredentials
func PlaintextCredentialsByHashType(hashType clientpb.HashType) ([]*clientpb.Credential, error) {
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{
		HashType: int32(hashType),
	}).Not("plaintext = ?", "").Find(&credentials).Error

	pbCredentials := []*clientpb.Credential{}
	for _, credential := range credentials {
		pbCredentials = append(pbCredentials, credential.ToProtobuf())
	}
	return pbCredentials, err
}

// CredentialsByID
func CredentialByID(id string) (*clientpb.Credential, error) {
	credential := &models.Credential{}
	credID := uuid.FromStringOrNil(id)
	if credID != uuid.Nil {
		err := Session().Where(&models.Credential{ID: credID}).First(&credential).Error
		return credential.ToProtobuf(), err
	}
	credentials := []*models.Credential{}
	err := Session().Where(&models.Credential{}).Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	for _, cred := range credentials {
		if strings.HasPrefix(cred.ID.String(), id) {
			return cred.ToProtobuf(), nil
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
	pbCrackFiles := []*clientpb.CrackFile{}
	for _, crackFile := range crackFiles {
		pbCrackFiles = append(pbCrackFiles, crackFile.ToProtobuf())
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

// watchtower - List configurations
func WatchTowerConfigs() ([]*clientpb.MonitoringProvider, error) {
	var monitoringProviders []*models.MonitoringProvider
	err := Session().Where(&models.MonitoringProvider{}).Find(&monitoringProviders).Error

	pbMonitoringProviders := []*clientpb.MonitoringProvider{}
	for _, monitoringProvider := range monitoringProviders {
		pbMonitoringProviders = append(pbMonitoringProviders, monitoringProvider.ToProtobuf())
	}

	return pbMonitoringProviders, err
}

func SaveWatchTowerConfig(m *clientpb.MonitoringProvider) error {
	dbMonitoringProvider := models.MonitorFromProtobuf(m)
	dbSession := Session()
	result := dbSession.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&dbMonitoringProvider)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func WatchTowerConfigDel(m *clientpb.MonitoringProvider) error {
	id, _ := uuid.FromString(m.ID)
	return Session().Where(&models.MonitoringProvider{ID: id}).Delete(&models.MonitoringProvider{}).Error
}

// ResourceID queries
func ResourceIDByType(resourceType string) ([]*clientpb.ResourceID, error) {
	resourceIDs := []*models.ResourceID{}
	err := Session().Where(&models.ResourceID{
		Type: resourceType,
	}).Find(&resourceIDs).Error
	if err != nil {
		return nil, err
	}

	pbResourceID := []*clientpb.ResourceID{}
	for _, resourceID := range resourceIDs {
		pbResourceID = append(pbResourceID, resourceID.ToProtobuf())
	}

	return pbResourceID, nil
}

func ResourceIDs() ([]*clientpb.ResourceID, error) {
	resourceIDs := []*models.ResourceID{}
	err := Session().Where(&models.ResourceID{}).Find(&resourceIDs).Error
	if err != nil {
		return nil, err
	}
	pbResourceIDs := []*clientpb.ResourceID{}
	for _, resourceID := range resourceIDs {
		pbResourceIDs = append(pbResourceIDs, resourceID.ToProtobuf())
	}

	return pbResourceIDs, nil
}

// ResourceID by name
func ResourceIDByName(name string) (*clientpb.ResourceID, error) {
	resourceID := &models.ResourceID{}
	err := Session().Where(&models.ResourceID{
		Name: name,
	}).First(&resourceID).Error
	if err != nil {
		return nil, err
	}

	pbResourceID := resourceID.ToProtobuf()

	return pbResourceID, nil
}

// ResourceID by value
func ResourceIDByValue(id uint64) (*clientpb.ResourceID, error) {
	resourceID := &models.ResourceID{}
	err := Session().Where(&models.ResourceID{
		Value: id,
	}).First(&resourceID).Error
	if err != nil {
		return nil, err
	}

	return resourceID.ToProtobuf(), nil
}

func SaveResourceID(r *clientpb.ResourceID) error {
	resourceID := &models.ResourceID{
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Value,
	}

	dbSession := Session()
	result := dbSession.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&resourceID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
