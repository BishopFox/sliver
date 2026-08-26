package handlers

import (
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBeaconRegisterHandlerPropagatesCapabilities(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "beacon-capabilities.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := testDB.AutoMigrate(&models.Beacon{}, &models.BeaconTask{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	originalDB := db.Client
	db.Client = testDB
	t.Cleanup(func() {
		db.Client = originalDB
		sqlDB, dbErr := testDB.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	beaconID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("generate beacon ID: %v", err)
	}
	registerData, err := proto.Marshal(&sliverpb.BeaconRegister{
		ID: beaconID.String(),
		Register: &sliverpb.Register{
			Uuid:         beaconID.String(),
			Capabilities: sliverpb.CapabilityBOFV1,
		},
	})
	if err != nil {
		t.Fatalf("marshal beacon register: %v", err)
	}

	beaconRegisterHandler(core.NewImplantConnection("test", "n/a"), registerData)

	beacon, err := db.BeaconByID(beaconID.String())
	if err != nil {
		t.Fatalf("load registered beacon: %v", err)
	}
	if beacon.Capabilities != sliverpb.CapabilityBOFV1 {
		t.Fatalf("expected Capabilities=%d, got %d", sliverpb.CapabilityBOFV1, beacon.Capabilities)
	}
	if beacon.ToProtobuf().Capabilities != sliverpb.CapabilityBOFV1 {
		t.Fatalf("expected protobuf Capabilities=%d, got %d", sliverpb.CapabilityBOFV1, beacon.ToProtobuf().Capabilities)
	}
}
