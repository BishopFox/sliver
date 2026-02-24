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

// ExportImplant exports one or all implant builds as a portable bundle.
// Each bundle carries the config, build metadata, C2 endpoints, and resource ID —
// everything needed to reconstruct the entry on another server.
func (rpc *Server) ExportImplant(ctx context.Context, req *clientpb.ExportImplantReq) (*clientpb.ExportImplantBundle, error) {
	if req.Name == "" && !req.All {
		return nil, status.Error(codes.InvalidArgument, "either --name <codename> or --all is required")
	}

	bundle := &clientpb.ExportImplantBundle{}

	if req.All {
		builds, err := db.ImplantBuilds()
		if err != nil {
			return nil, rpcError(err)
		}

		for name, config := range builds.Configs {
			b, err := db.ImplantBuildByName(name)
			if err != nil {
				migrateLog.Warnf("Skipping build %q — failed to fetch: %v", name, err)
				continue
			}
			resID, _ := db.ResourceIDByName(name)
			bundle.Bundles = append(bundle.Bundles, &clientpb.ImplantBundle{
				Config: config,
				Build:  b,
				C2S:    config.C2,
				ResID:  resID,
			})
		}
	} else {
		b, err := exportSingleBuild(req.Name)
		if err != nil {
			return nil, err
		}
		bundle.Bundles = append(bundle.Bundles, b)
	}

	// --- Export Encoders ---
	encoders, _ := db.ResourceIDByType("encoder")
	bundle.Encoders = encoders

	migrateLog.Infof("Exported %d implant bundle(s) and %d encoder(s)", len(bundle.Bundles), len(bundle.Encoders))
	return bundle, nil
}

func exportSingleBuild(name string) (*clientpb.ImplantBundle, error) {
	build, err := db.ImplantBuildByName(name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no build found with name %q", name)
	}

	config, err := db.ImplantConfigByID(build.ImplantConfigID)
	if err != nil {
		migrateLog.Errorf("Failed to load config for build %q (configID=%s): %v", name, build.ImplantConfigID, err)
		return nil, status.Errorf(codes.Internal, "config missing for build %q — check server logs", name)
	}

	resID, _ := db.ResourceIDByName(name)
	return &clientpb.ImplantBundle{
		Config: config,
		Build:  build,
		C2S:    config.C2,
		ResID:  resID,
	}, nil
}

// ImportImplant imports a bundle produced by ExportImplant.
// Existing records (matched by ID or name) are left untouched — this is
// intentionally idempotent so re-running an import is always safe.
func (rpc *Server) ImportImplant(ctx context.Context, req *clientpb.ExportImplantBundle) (*commonpb.Empty, error) {
	if len(req.Bundles) == 0 {
		return nil, status.Error(codes.InvalidArgument, "bundle is empty — nothing to import")
	}

	migrateLog.Infof("Starting import of %d bundle(s)", len(req.Bundles))

	dbSession := db.Session().Debug()

	var imported, skipped int

	for _, b := range req.Bundles {
		if b.Config == nil || b.Build == nil {
			migrateLog.Warn("Skipping malformed bundle — nil config or build")
			skipped++
			continue
		}

		buildName := b.Build.Name
		migrateLog.Debugf("Processing bundle for %q (configID=%s)", buildName, b.Config.ID)

		// ── Config ──────────────────────────────────────────────────────────
		if !configExists(b.Config.ID) {
			if len(b.Config.C2) == 0 {
				b.Config.C2 = b.C2S
			}

			dbConfig := models.ImplantConfigFromProtobuf(b.Config)
			dbConfig.ImplantBuilds = nil

			if err := dbSession.Create(&dbConfig).Error; err != nil {
				migrateLog.Errorf("Failed to create config %s for build %q: %v", dbConfig.ID, buildName, err)
				skipped++
				continue
			}
			migrateLog.Debugf("Created config %s", dbConfig.ID)
		} else {
			migrateLog.Debugf("Config %s already exists — skipping", b.Config.ID)
		}

		// ── Build ────────────────────────────────────────────────────────────
		if !buildAlreadyExists(b.Build.ID, buildName) {
			dbBuild := models.ImplantBuildFromProtobuf(b.Build)
			if err := dbSession.Create(&dbBuild).Error; err != nil {
				migrateLog.Errorf("Failed to create build %q: %v", buildName, err)
				skipped++
				continue
			}
			migrateLog.Debugf("Created build %q", buildName)
		} else {
			migrateLog.Debugf("Build %q already exists — skipping", buildName)
		}
		imported++
	}

	// ── Reload ─────────────────────────────────────────────────────────
	encoders.ReloadEncoderMap()

	migrateLog.Infof("Import complete — %d bundles processed (%d errors/skipped)", imported, skipped)
	return &commonpb.Empty{}, nil
}

// importResourceID - Import a ResourceID record.
// If the ID already exists, we skip it. This ensures we don't crash
// while allowing you to merge your old server's IDs into the new one.
func importResourceID(dbSession *gorm.DB, res *clientpb.ResourceID) {
	if res == nil {
		return
	}

	if res.ID != "" {
		existing := models.ResourceID{}
		if err := dbSession.Unscoped().Where("id = ?", res.ID).First(&existing).Error; err == nil {
			migrateLog.Debugf("ResourceID %q (ID=%s) already exists — skipping", res.Name, res.ID)
			return
		}
	}

	dbRes := models.ResourceIDFromProtobuf(res)
	if err := dbSession.Create(&dbRes).Error; err != nil {
		migrateLog.Errorf("Failed to create ResourceID %q: %v", res.Name, err)
		return
	}
	migrateLog.Debugf("Imported ResourceID %q (Value: %d)", res.Name, dbRes.Value)
}

// ── helpers ──────────────────────────────────────────────────────────────────
func configExists(id string) bool {
	if id == "" {
		return false
	}
	_, err := db.ImplantConfigByID(id)
	return err == nil
}

func buildAlreadyExists(id, name string) bool {
	if id != "" {
		if _, err := db.ImplantBuildByID(id); err == nil {
			return true
		}
	}
	if name != "" {
		if _, err := db.ImplantBuildByName(name); err == nil {
			return true
		}
	}
	return false
}
