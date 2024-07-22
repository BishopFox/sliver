package models

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
*/

import (
	"net/url"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

var (
	modelLog = log.NamedLogger("models", "implant")
)

// ImplantBuild - Represents an implant
type ImplantBuild struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name string `gorm:"unique;"`

	// Checksums stores of the implant binary
	MD5    string
	SHA1   string
	SHA256 string

	// Burned indicates whether the implant
	// has been seen on threat intel platforms
	Burned bool

	// Resource ID referencing build
	ImplantID uint64

	ImplantConfigID uuid.UUID

	// ECC
	PeerPublicKey           string
	PeerPublicKeyDigest     string
	PeerPrivateKey          string
	PeerPublicKeySignature  string
	AgeServerPublicKey      string
	MinisignServerPublicKey string

	// MTLS
	MtlsCACert string
	MtlsCert   string
	MtlsKey    string

	// WireGuard
	WGImplantPrivKey string
	WGServerPubKey   string

	Stage bool
}

// BeforeCreate - GORM hook
func (ib *ImplantBuild) BeforeCreate(tx *gorm.DB) (err error) {
	ib.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ib.CreatedAt = time.Now()
	return nil
}

// Convert ImplantBuild To Protobuf
func (ib *ImplantBuild) ToProtobuf() *clientpb.ImplantBuild {
	build := clientpb.ImplantBuild{
		ID:                      ib.ID.String(),
		Name:                    ib.Name,
		MD5:                     ib.MD5,
		SHA1:                    ib.SHA1,
		SHA256:                  ib.SHA256,
		Burned:                  ib.Burned,
		ImplantID:               ib.ImplantID,
		ImplantConfigID:         ib.ImplantConfigID.String(),
		AgeServerPublicKey:      ib.AgeServerPublicKey,
		PeerPublicKey:           ib.PeerPublicKey,
		PeerPrivateKey:          ib.PeerPrivateKey,
		MinisignServerPublicKey: ib.MinisignServerPublicKey,
		PeerPublicKeySignature:  ib.PeerPublicKeySignature,
		PeerPublicKeyDigest:     ib.PeerPublicKeyDigest,
		MtlsCACert:              ib.MtlsCACert,
		MtlsCert:                ib.MtlsCert,
		MtlsKey:                 ib.MtlsKey,
		WGImplantPrivKey:        ib.WGImplantPrivKey,
		WGServerPubKey:          ib.WGServerPubKey,
		Stage:                   ib.Stage,
	}
	return &build
}

func ImplantBuildFromProtobuf(ib *clientpb.ImplantBuild) *ImplantBuild {
	id, _ := uuid.FromString(ib.ID)
	ImplantConfidID, _ := uuid.FromString(ib.ImplantConfigID)
	build := ImplantBuild{
		ID:              id,
		Name:            ib.Name,
		MD5:             ib.MD5,
		SHA1:            ib.SHA1,
		SHA256:          ib.SHA256,
		Burned:          ib.Burned,
		ImplantID:       ib.ImplantID,
		ImplantConfigID: ImplantConfidID,
		MtlsCACert:      ib.MtlsCACert,
		MtlsCert:        ib.MtlsCert,
		MtlsKey:         ib.MtlsKey,

		AgeServerPublicKey:      ib.AgeServerPublicKey,
		PeerPublicKey:           ib.PeerPublicKey,
		PeerPrivateKey:          ib.PeerPrivateKey,
		PeerPublicKeySignature:  ib.PeerPublicKeySignature,
		MinisignServerPublicKey: ib.MinisignServerPublicKey,
		PeerPublicKeyDigest:     ib.PeerPublicKeyDigest,

		WGImplantPrivKey: ib.WGImplantPrivKey,
		WGServerPubKey:   ib.WGServerPubKey,
		Stage:            ib.Stage,
	}
	return &build
}

// ImplantConfig - An implant build configuration
type ImplantConfig struct {
	ID               uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ImplantProfileID *uuid.UUID

	ImplantBuilds []ImplantBuild
	CreatedAt     time.Time `gorm:"->;<-:create;"`

	// Go
	GOOS   string
	GOARCH string

	TemplateName string

	IsBeacon       bool
	BeaconInterval int64
	BeaconJitter   int64

	Debug               bool
	DebugFile           string
	Evasion             bool
	ObfuscateSymbols    bool
	ReconnectInterval   int64
	PollTimeout         int64
	MaxConnectionErrors uint32
	ConnectionStrategy  string
	SGNEnabled          bool

	// WireGuard
	WGPeerTunIP       string
	WGKeyExchangePort uint32
	WGTcpCommsPort    uint32

	C2 []ImplantC2

	IncludeMTLS bool
	IncludeWG   bool
	IncludeHTTP bool
	IncludeDNS  bool

	CanaryDomains   []CanaryDomain
	IncludeNamePipe bool
	IncludeTCP      bool

	// Limits
	LimitDomainJoined bool
	LimitHostname     string
	LimitUsername     string
	LimitDatetime     string
	LimitFileExists   string
	LimitLocale       string

	// Output Format
	Format clientpb.OutputFormat

	// For 	IsSharedLib bool
	IsSharedLib bool
	IsService   bool
	IsShellcode bool

	RunAtLoad bool

	HttpC2ConfigName       string
	NetGoEnabled           bool
	TrafficEncodersEnabled bool
	Assets                 []EncoderAsset
}

// BeforeCreate - GORM hook
func (ic *ImplantConfig) BeforeCreate(tx *gorm.DB) (err error) {
	ic.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ic.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Convert ImplantProfile to protobuf equiv
func (ip *ImplantProfile) ToProtobuf() *clientpb.ImplantProfile {
	profile := &clientpb.ImplantProfile{
		ID:     ip.ID.String(),
		Name:   ip.Name,
		Config: ip.ImplantConfig.ToProtobuf(),
	}

	return profile
}

// ToProtobuf - Convert ImplantConfig to protobuf equiv
func (ic *ImplantConfig) ToProtobuf() *clientpb.ImplantConfig {
	implantBuilds := []*clientpb.ImplantBuild{}
	for _, implantBuild := range ic.ImplantBuilds {
		implantBuilds = append(implantBuilds, implantBuild.ToProtobuf())
	}
	config := &clientpb.ImplantConfig{
		ID:            ic.ID.String(),
		ImplantBuilds: implantBuilds,

		IsBeacon:       ic.IsBeacon,
		BeaconInterval: ic.BeaconInterval,
		BeaconJitter:   ic.BeaconJitter,

		GOOS:   ic.GOOS,
		GOARCH: ic.GOARCH,

		Debug:            ic.Debug,
		DebugFile:        ic.DebugFile,
		Evasion:          ic.Evasion,
		ObfuscateSymbols: ic.ObfuscateSymbols,
		TemplateName:     ic.TemplateName,
		SGNEnabled:       ic.SGNEnabled,

		ReconnectInterval:   ic.ReconnectInterval,
		MaxConnectionErrors: ic.MaxConnectionErrors,
		PollTimeout:         ic.PollTimeout,
		ConnectionStrategy:  ic.ConnectionStrategy,

		LimitDatetime:     ic.LimitDatetime,
		LimitDomainJoined: ic.LimitDomainJoined,
		LimitHostname:     ic.LimitHostname,
		LimitUsername:     ic.LimitUsername,
		LimitFileExists:   ic.LimitFileExists,
		LimitLocale:       ic.LimitLocale,

		IsSharedLib:       ic.IsSharedLib,
		IsService:         ic.IsService,
		IsShellcode:       ic.IsShellcode,
		Format:            ic.Format,
		WGPeerTunIP:       ic.WGPeerTunIP,
		WGKeyExchangePort: ic.WGKeyExchangePort,
		WGTcpCommsPort:    ic.WGTcpCommsPort,

		TrafficEncodersEnabled: ic.TrafficEncodersEnabled,
		NetGoEnabled:           ic.NetGoEnabled,
		HTTPC2ConfigName:       ic.HttpC2ConfigName,

		IncludeMTLS:     ic.IncludeMTLS,
		IncludeHTTP:     ic.IncludeHTTP,
		IncludeDNS:      ic.IncludeDNS,
		IncludeNamePipe: ic.IncludeNamePipe,
		IncludeWG:       ic.IncludeWG,
		IncludeTCP:      ic.IncludeTCP,
	}

	if ic.ImplantProfileID != nil {
		config.ImplantProfileID = ic.ImplantProfileID.String()
	}

	// Copy Canary Domains
	config.CanaryDomains = []string{}
	for _, canaryDomain := range ic.CanaryDomains {
		config.CanaryDomains = append(config.CanaryDomains, canaryDomain.Domain)
	}

	// Copy Assets
	config.Assets = []*commonpb.File{}
	for _, asset := range ic.Assets {
		config.Assets = append(config.Assets, asset.ToProtobuf())
	}

	// Copy C2
	config.C2 = []*clientpb.ImplantC2{}
	for _, c2 := range ic.C2 {
		config.C2 = append(config.C2, c2.ToProtobuf())
	}
	return config
}

// CanaryDomainsList - Get string slice of canary domains
func (ic *ImplantConfig) CanaryDomainsList() []string {
	domains := []string{}
	for _, canaryDomain := range ic.CanaryDomains {
		domains = append(domains, canaryDomain.Domain)
	}
	return domains
}

// CanaryDomain - Canary domain, belongs to ImplantConfig
type CanaryDomain struct {
	ID              uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ImplantConfigID uuid.UUID
	CreatedAt       time.Time `gorm:"->;<-:create;"`

	Domain string
}

// BeforeCreate - GORM hook
func (c *CanaryDomain) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

// ImplantC2 - C2 struct
type ImplantC2 struct {
	// gorm.Model

	ID              uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ImplantConfigID uuid.UUID
	CreatedAt       time.Time `gorm:"->;<-:create;"`

	Priority uint32
	URL      string
	Options  string
}

// BeforeCreate - GORM hook
func (c2 *ImplantC2) BeforeCreate(tx *gorm.DB) (err error) {
	if c2.ID == uuid.Nil {
		c2.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}
	c2.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Convert to protobuf version
func (c2 *ImplantC2) ToProtobuf() *clientpb.ImplantC2 {
	return &clientpb.ImplantC2{
		ID:       c2.ID.String(),
		Priority: c2.Priority,
		URL:      c2.URL,
		Options:  c2.Options,
	}
}

func (c2 *ImplantC2) String() string {
	return c2.URL
}

// ImplantProfile - An implant build configuration
type ImplantProfile struct {
	// gorm.Model

	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name          string `gorm:"unique;"`
	ImplantConfig *ImplantConfig
}

// BeforeCreate - GORM hook
func (ip *ImplantProfile) BeforeCreate(tx *gorm.DB) (err error) {
	ip.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	ip.CreatedAt = time.Now()
	return nil
}

// EncoderAsset - Tracks which assets were embedded into the implant
// but we currently don't keep a copy of the actual data
type EncoderAsset struct {
	ID              uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ImplantConfigID uuid.UUID

	Name string
}

func (t *EncoderAsset) ToProtobuf() *commonpb.File {
	return &commonpb.File{Name: t.Name}
}

const defaultTemplateName = "sliver"

// ImplantProfileFromProtobuf - Create a native profile struct from Protobuf
func ImplantProfileFromProtobuf(pbProfile *clientpb.ImplantProfile) *ImplantProfile {
	cfg := ImplantProfile{}
	id, _ := uuid.FromString(pbProfile.ID)
	cfg.ID = id
	cfg.Name = pbProfile.Name
	config := ImplantConfigFromProtobuf(pbProfile.Config)
	cfg.ImplantConfig = config

	return &cfg
}

// ImplantConfigFromProtobuf - Create a native config struct from Protobuf
func ImplantConfigFromProtobuf(pbConfig *clientpb.ImplantConfig) *ImplantConfig {
	implantBuilds := []ImplantBuild{}
	for _, implantBuild := range pbConfig.ImplantBuilds {
		implantBuilds = append(implantBuilds, *ImplantBuildFromProtobuf(implantBuild))
	}
	cfg := ImplantConfig{}
	id, _ := uuid.FromString(pbConfig.ID)
	cfg.ID = id
	cfg.ImplantBuilds = implantBuilds
	profileID, _ := uuid.FromString(pbConfig.ImplantProfileID)
	if profileID == uuid.Nil {
		cfg.ImplantProfileID = nil
	} else {
		cfg.ImplantProfileID = &profileID
	}

	cfg.IsBeacon = pbConfig.IsBeacon
	cfg.BeaconInterval = pbConfig.BeaconInterval
	cfg.BeaconJitter = pbConfig.BeaconJitter

	cfg.GOOS = pbConfig.GOOS
	cfg.GOARCH = pbConfig.GOARCH
	cfg.Debug = pbConfig.Debug
	cfg.Evasion = pbConfig.Evasion
	cfg.ObfuscateSymbols = pbConfig.ObfuscateSymbols
	cfg.TemplateName = pbConfig.TemplateName
	if cfg.TemplateName == "" {
		cfg.TemplateName = defaultTemplateName
	}
	cfg.SGNEnabled = pbConfig.SGNEnabled

	cfg.IncludeMTLS = IsC2Enabled([]string{"mtls"}, pbConfig.C2)
	cfg.IncludeWG = IsC2Enabled([]string{"wg"}, pbConfig.C2)
	cfg.IncludeHTTP = IsC2Enabled([]string{"http", "https"}, pbConfig.C2)
	cfg.IncludeDNS = IsC2Enabled([]string{"dns"}, pbConfig.C2)
	cfg.IncludeNamePipe = IsC2Enabled([]string{"namedpipe"}, pbConfig.C2)
	cfg.IncludeTCP = IsC2Enabled([]string{"tcppivot"}, pbConfig.C2)

	cfg.WGPeerTunIP = pbConfig.WGPeerTunIP
	cfg.WGKeyExchangePort = pbConfig.WGKeyExchangePort
	cfg.WGTcpCommsPort = pbConfig.WGTcpCommsPort

	cfg.ReconnectInterval = pbConfig.ReconnectInterval
	cfg.MaxConnectionErrors = pbConfig.MaxConnectionErrors
	cfg.PollTimeout = pbConfig.PollTimeout

	cfg.C2 = copyC2List(pbConfig.C2, cfg.ID)
	cfg.CanaryDomains = []CanaryDomain{}
	for _, pbCanary := range pbConfig.CanaryDomains {
		cfg.CanaryDomains = append(cfg.CanaryDomains, CanaryDomain{
			Domain: pbCanary,
		})
	}
	cfg.ConnectionStrategy = pbConfig.ConnectionStrategy

	cfg.LimitDomainJoined = pbConfig.LimitDomainJoined
	cfg.LimitDatetime = pbConfig.LimitDatetime
	cfg.LimitHostname = pbConfig.LimitHostname
	cfg.LimitUsername = pbConfig.LimitUsername
	cfg.LimitFileExists = pbConfig.LimitFileExists
	cfg.LimitLocale = pbConfig.LimitLocale

	cfg.Format = pbConfig.Format
	cfg.IsSharedLib = pbConfig.IsSharedLib
	cfg.IsService = pbConfig.IsService
	cfg.IsShellcode = pbConfig.IsShellcode
	cfg.RunAtLoad = pbConfig.RunAtLoad
	cfg.DebugFile = pbConfig.DebugFile

	cfg.HttpC2ConfigName = pbConfig.HTTPC2ConfigName
	cfg.NetGoEnabled = pbConfig.NetGoEnabled
	cfg.TrafficEncodersEnabled = pbConfig.TrafficEncodersEnabled

	cfg.Assets = []EncoderAsset{}
	for _, pbAsset := range pbConfig.Assets {
		cfg.Assets = append(cfg.Assets, EncoderAsset{
			Name: pbAsset.Name,
		})
	}

	return &cfg
}

func copyC2List(src []*clientpb.ImplantC2, id uuid.UUID) []ImplantC2 {
	c2s := []ImplantC2{}
	for _, srcC2 := range src {
		c2URL, err := url.Parse(srcC2.URL)
		if err != nil {
			modelLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		uuid, _ := uuid.FromString(srcC2.ID)
		c2s = append(c2s, ImplantC2{
			ID:              uuid,
			ImplantConfigID: id,
			Priority:        srcC2.Priority,
			URL:             c2URL.String(),
			Options:         srcC2.Options,
		})
	}
	return c2s
}

func IsC2Enabled(schemes []string, c2s []*clientpb.ImplantC2) bool {
	for _, c2 := range c2s {
		c2URL, err := url.Parse(c2.URL)
		if err != nil {
			modelLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		for _, scheme := range schemes {
			if scheme == c2URL.Scheme {
				return true
			}
		}
	}
	modelLog.Debugf("No %v URLs found in %v", schemes, c2s)
	return false
}
