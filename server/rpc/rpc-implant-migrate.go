package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

var migrateLog = log.NamedLogger("rpc", "migrate")

func (rpc *Server) ExportImplant(ctx context.Context, req *clientpb.ExportImplantReq) (*clientpb.ExportImplantBundle, error) {
	if req.Name == "" && !req.All {
		return nil, status.Error(codes.InvalidArgument, "must specify --name <codename> or --all")
	}

	bundle := &clientpb.ExportImplantBundle{}

	if req.All {
		builds, err := db.ImplantBuilds()
		if err == nil {
			for name := range builds.Configs {
				if b, err := exportSingleBuild(name); err == nil {
					bundle.Bundles = append(bundle.Bundles, b)
				}
			}
		}
	} else {
		if b, err := exportSingleBuild(req.Name); err == nil {
			bundle.Bundles = append(bundle.Bundles, b)
		} else {
			return nil, err
		}
	}

	bundle.Encoders, _ = db.ResourceIDByType("encoder")

	// certificates ──────────────────────────────────────────────────────────
	bundle.Certificates = exportCertificates()

	// key_values ──────────────────────────────────────────────────────────
	bundle.KeyValues = exportKeyValues()

	migrateLog.Infof("Exported %d bundle(s), %d resource(s), %d certificate(s), %d key_value(s)",
		len(bundle.Bundles), len(bundle.Encoders), len(bundle.Certificates), len(bundle.KeyValues))
	return bundle, nil
}

func (rpc *Server) ImportImplant(ctx context.Context, req *clientpb.ExportImplantBundle) (*commonpb.Empty, error) {
	migrateLog.Infof("Starting import of %d bundle(s), %d resource(s), %d certificate(s), %d key_value(s)",
		len(req.Bundles), len(req.Encoders), len(req.Certificates), len(req.KeyValues))
	dbSession := db.Session().Debug()

	for _, res := range req.Encoders {
		importResourceID(dbSession, res)
	}

	var imported int
	for _, b := range req.Bundles {
		if importBundle(dbSession, b) {
			imported++
		}
	}

	// certificates ──────────────────────────────────────────────────────────
	for _, cert := range req.Certificates {
		importCertificate(dbSession, cert)
	}

	// key_values ──────────────────────────────────────────────────────────
	for _, kv := range req.KeyValues {
		importKeyValue(dbSession, kv)
	}

	// sync the server's memory-mapped encoders
	encoders.ReloadEncoderMap()

	migrateLog.Infof("Import complete: %d bundles processed", imported)
	return &commonpb.Empty{}, nil
}

// helpers
func exportSingleBuild(name string) (*clientpb.ImplantBundle, error) {
	b, err := db.ImplantBuildByName(name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "build %q not found", name)
	}
	config, _ := db.ImplantConfigByID(b.ImplantConfigID)
	resID, _ := db.ResourceIDByName(name)
	return &clientpb.ImplantBundle{
		Config: config,
		Build:  b,
		C2S:    config.C2,
		ResID:  resID,
	}, nil
}

func importBundle(dbSession *gorm.DB, b *clientpb.ImplantBundle) bool {
	if b.Config == nil || b.Build == nil {
		return false
	}

	// implant_configs ──────────────────────────────────────────────────────────
	if _, err := db.ImplantConfigByID(b.Config.ID); err != nil {
		dbCfg := models.ImplantConfigFromProtobuf(b.Config)
		dbCfg.ImplantBuilds = nil
		dbSession.Create(&dbCfg)
	}

	// implant_builds ──────────────────────────────────────────────────────────
	if _, err := db.ImplantBuildByID(b.Build.ID); err != nil {
		dbBuild := models.ImplantBuildFromProtobuf(b.Build)
		dbSession.Create(&dbBuild)
	}

	// resource_ids ──────────────────────────────────────────────────────────
	if b.ResID != nil {
		importResourceID(dbSession, b.ResID)
	}
	return true
}

func importResourceID(dbSession *gorm.DB, res *clientpb.ResourceID) {
	if res == nil || res.ID == "" {
		return
	}
	// skip if ID already exists
	var existing models.ResourceID
	if err := dbSession.Unscoped().Where("id = ?", res.ID).First(&existing).Error; err == nil {
		return
	}
	dbRes := models.ResourceIDFromProtobuf(res)
	dbSession.Create(dbRes)
	migrateLog.Debugf("Imported ResourceID %q (%s)", res.Name, res.Type)
}

// certificates ──────────────────────────────────────────────────────────

func exportCertificates() []*clientpb.CertificateEntry {
	var certs []models.Certificate
	db.Session().Find(&certs)

	out := make([]*clientpb.CertificateEntry, 0, len(certs))
	for _, c := range certs {
		out = append(out, &clientpb.CertificateEntry{
			ID:             c.ID.String(),
			CommonName:     c.CommonName,
			CAType:         c.CAType,
			KeyType:        c.KeyType,
			CertificatePEM: c.CertificatePEM,
			PrivateKeyPEM:  c.PrivateKeyPEM,
			CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

func importCertificate(dbSession *gorm.DB, entry *clientpb.CertificateEntry) {
	if entry == nil || entry.ID == "" {
		return
	}
	var existing models.Certificate
	if err := dbSession.Where("id = ?", entry.ID).First(&existing).Error; err == nil {
		return // already exists, skip
	}
	cert := models.Certificate{
		CommonName:     entry.CommonName,
		CAType:         entry.CAType,
		KeyType:        entry.KeyType,
		CertificatePEM: entry.CertificatePEM,
		PrivateKeyPEM:  entry.PrivateKeyPEM,
	}
	dbSession.Create(&cert)
	migrateLog.Debugf("Imported Certificate CN=%q CAType=%q", entry.CommonName, entry.CAType)
}

// key_values ──────────────────────────────────────────────────────────

func exportKeyValues() []*clientpb.KeyValueEntry {
	var kvs []models.KeyValue
	db.Session().Find(&kvs)

	out := make([]*clientpb.KeyValueEntry, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, &clientpb.KeyValueEntry{
			ID:        kv.ID.String(),
			Key:       kv.Key,
			Value:     kv.Value,
			CreatedAt: kv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

func importKeyValue(dbSession *gorm.DB, entry *clientpb.KeyValueEntry) {
	if entry == nil || entry.ID == "" {
		return
	}
	// skip if ID already exists
	var existing models.KeyValue
	if err := dbSession.Where("id = ?", entry.ID).First(&existing).Error; err == nil {
		return
	}
	kv := models.KeyValue{
		Key:   entry.Key,
		Value: entry.Value,
	}
	dbSession.Create(&kv)
	migrateLog.Debugf("Imported KeyValue key=%q", entry.Key)
}
