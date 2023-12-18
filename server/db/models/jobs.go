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
	"time"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type ListenerJob struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	JobID               uint32 `gorm:"unique;"`
	Type                string
	HttpListener        HTTPListener
	MtlsListener        MtlsListener
	DnsListener         DNSListener
	WgListener          WGListener
	MultiplayerListener MultiplayerListener
}

type HTTPListener struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`

	Domain          string
	Host            string
	Port            uint32
	Secure          bool
	Website         string
	Cert            []byte
	Key             []byte
	Acme            bool
	EnforceOtp      bool
	LongPollTimeout int64
	LongPollJitter  int64
	RandomizeJarm   bool
	Staging         bool
}

type DNSListener struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`

	Domains    []DnsDomain
	Canaries   bool
	Host       string
	Port       uint32
	EnforceOtp bool
}

type WGListener struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`
	Host          string
	Port          uint32
	NPort         uint32
	KeyPort       uint32
	TunIP         string
}

type MtlsListener struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`
	Host          string
	Port          uint32
}

type MultiplayerListener struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`
	Host          string
	Port          uint32
}

type DnsDomain struct {
	ID            uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	DNSListenerID uuid.UUID `gorm:"type:uuid;"`
	Domain        string
}

// orm hooks
func (j *ListenerJob) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	j.CreatedAt = time.Now()
	return nil
}

func (j *HTTPListener) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	return nil
}

func (j *DNSListener) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	return nil
}

func (j *WGListener) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	return nil
}

func (j *MtlsListener) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	return nil
}

// To Protobuf
func (j *ListenerJob) ToProtobuf() *clientpb.ListenerJob {
	return &clientpb.ListenerJob{
		ID:        j.ID.String(),
		Type:      j.Type,
		JobID:     j.JobID,
		HTTPConf:  j.HttpListener.ToProtobuf(),
		MTLSConf:  j.MtlsListener.ToProtobuf(),
		DNSConf:   j.DnsListener.ToProtobuf(),
		WGConf:    j.WgListener.ToProtobuf(),
		MultiConf: j.MultiplayerListener.ToProtobuf(),
	}
}

func (j *HTTPListener) ToProtobuf() *clientpb.HTTPListenerReq {

	return &clientpb.HTTPListenerReq{
		Domain:          j.Domain,
		Host:            j.Host,
		Port:            j.Port,
		Secure:          j.Secure,
		Website:         j.Website,
		Cert:            j.Cert,
		Key:             j.Key,
		ACME:            j.Acme,
		EnforceOTP:      j.EnforceOtp,
		LongPollTimeout: int64(j.LongPollTimeout),
		LongPollJitter:  int64(j.LongPollJitter),
		RandomizeJARM:   j.RandomizeJarm,
	}
}

func (j *DNSListener) ToProtobuf() *clientpb.DNSListenerReq {
	var domains []string
	for _, domain := range j.Domains {
		domains = append(domains, domain.Domain)
	}
	return &clientpb.DNSListenerReq{
		Domains:    domains,
		Canaries:   j.Canaries,
		Host:       j.Host,
		Port:       j.Port,
		EnforceOTP: j.EnforceOtp,
	}
}

func (j *WGListener) ToProtobuf() *clientpb.WGListenerReq {
	return &clientpb.WGListenerReq{
		Host:    j.Host,
		Port:    j.Port,
		NPort:   j.NPort,
		KeyPort: j.KeyPort,
		TunIP:   j.TunIP,
	}
}

func (j *MtlsListener) ToProtobuf() *clientpb.MTLSListenerReq {
	return &clientpb.MTLSListenerReq{
		Host: j.Host,
		Port: j.Port,
	}
}

func (j *MultiplayerListener) ToProtobuf() *clientpb.MultiplayerListenerReq {
	return &clientpb.MultiplayerListenerReq{
		Host: j.Host,
		Port: j.Port,
	}
}

// to model
func ListenerJobFromProtobuf(pbListenerJob *clientpb.ListenerJob) *ListenerJob {
	cfg := &ListenerJob{
		Type:  pbListenerJob.Type,
		JobID: pbListenerJob.JobID,
	}

	switch pbListenerJob.Type {
	case constants.HttpStr:
		cfg.HttpListener = HTTPListener{
			Domain:          pbListenerJob.HTTPConf.Domain,
			Host:            pbListenerJob.HTTPConf.Host,
			Port:            pbListenerJob.HTTPConf.Port,
			Secure:          pbListenerJob.HTTPConf.Secure,
			Website:         pbListenerJob.HTTPConf.Website,
			Cert:            pbListenerJob.HTTPConf.Cert,
			Key:             pbListenerJob.HTTPConf.Key,
			Acme:            pbListenerJob.HTTPConf.ACME,
			EnforceOtp:      pbListenerJob.HTTPConf.EnforceOTP,
			LongPollTimeout: pbListenerJob.HTTPConf.LongPollTimeout,
			LongPollJitter:  pbListenerJob.HTTPConf.LongPollJitter,
			RandomizeJarm:   pbListenerJob.HTTPConf.RandomizeJARM,
		}
	case constants.HttpsStr:
		cfg.HttpListener = HTTPListener{
			Domain:          pbListenerJob.HTTPConf.Domain,
			Host:            pbListenerJob.HTTPConf.Host,
			Port:            pbListenerJob.HTTPConf.Port,
			Secure:          pbListenerJob.HTTPConf.Secure,
			Website:         pbListenerJob.HTTPConf.Website,
			Cert:            pbListenerJob.HTTPConf.Cert,
			Key:             pbListenerJob.HTTPConf.Key,
			Acme:            pbListenerJob.HTTPConf.ACME,
			EnforceOtp:      pbListenerJob.HTTPConf.EnforceOTP,
			LongPollTimeout: pbListenerJob.HTTPConf.LongPollTimeout,
			LongPollJitter:  pbListenerJob.HTTPConf.LongPollJitter,
			RandomizeJarm:   pbListenerJob.HTTPConf.RandomizeJARM,
		}
	case constants.MtlsStr:
		cfg.MtlsListener = MtlsListener{
			Host: pbListenerJob.MTLSConf.Host,
			Port: pbListenerJob.MTLSConf.Port,
		}
	case constants.DnsStr:
		var domains []DnsDomain
		for _, domain := range pbListenerJob.DNSConf.Domains {
			domains = append(domains, DnsDomain{Domain: domain})
		}
		cfg.DnsListener = DNSListener{
			Domains:    domains,
			Canaries:   pbListenerJob.DNSConf.Canaries,
			Host:       pbListenerJob.DNSConf.Host,
			Port:       pbListenerJob.DNSConf.Port,
			EnforceOtp: pbListenerJob.DNSConf.EnforceOTP,
		}
	case constants.WGStr:
		cfg.WgListener = WGListener{
			Host:    pbListenerJob.WGConf.Host,
			Port:    pbListenerJob.WGConf.Port,
			NPort:   pbListenerJob.WGConf.NPort,
			KeyPort: pbListenerJob.WGConf.KeyPort,
			TunIP:   pbListenerJob.WGConf.TunIP,
		}
	case constants.MultiplayerModeStr:
		cfg.MultiplayerListener = MultiplayerListener{
			Host: pbListenerJob.MultiConf.Host,
			Port: pbListenerJob.MultiConf.Port,
		}
	}

	return cfg
}
