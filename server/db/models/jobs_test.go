package models

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/glebarez/sqlite"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type legacyMultiplayerListener struct {
	ID            uuid.UUID `gorm:"primaryKey;type:uuid;"`
	ListenerJobID uuid.UUID `gorm:"type:uuid;"`
	Host          string
	Port          uint32
	WireGuard     bool
}

func (legacyMultiplayerListener) TableName() string {
	return "multiplayer_listeners"
}

func TestMultiplayerListenerWireGuardRequiresOptIn(t *testing.T) {
	tests := []struct {
		name     string
		listener MultiplayerListener
		enabled  bool
	}{
		{
			name:     "direct listener",
			listener: MultiplayerListener{},
		},
		{
			name: "legacy wireguard listener without opt-in marker",
			listener: MultiplayerListener{
				WireGuard: true,
			},
		},
		{
			name: "explicit wireguard listener",
			listener: MultiplayerListener{
				WireGuard:      true,
				WireGuardOptIn: true,
			},
			enabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if enabled := test.listener.ToProtobuf().GetWireGuard(); enabled != test.enabled {
				t.Fatalf("expected WireGuard enabled %t, got %t", test.enabled, enabled)
			}
		})
	}
}

func TestListenerJobFromProtobufRecordsWireGuardOptIn(t *testing.T) {
	listener := ListenerJobFromProtobuf(&clientpb.ListenerJob{
		Type: constants.MultiplayerModeStr,
		MultiConf: &clientpb.MultiplayerListenerReq{
			Host:      "127.0.0.1",
			Port:      31337,
			WireGuard: true,
		},
	})

	if !listener.MultiplayerListener.WireGuard {
		t.Fatal("expected persisted listener to retain WireGuard mode")
	}
	if !listener.MultiplayerListener.WireGuardOptIn {
		t.Fatal("expected explicitly enabled listener to record WireGuard opt-in")
	}
}

func TestMultiplayerListenerWireGuardOptInMigration(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := database.AutoMigrate(&legacyMultiplayerListener{}); err != nil {
		t.Fatalf("create legacy multiplayer listener schema: %v", err)
	}
	legacy := legacyMultiplayerListener{
		ID:            uuid.Must(uuid.NewV4()),
		ListenerJobID: uuid.Must(uuid.NewV4()),
		Host:          "127.0.0.1",
		Port:          31337,
		WireGuard:     true,
	}
	if err := database.Create(&legacy).Error; err != nil {
		t.Fatalf("insert legacy multiplayer listener: %v", err)
	}

	if err := database.AutoMigrate(&MultiplayerListener{}); err != nil {
		t.Fatalf("migrate multiplayer listener schema: %v", err)
	}
	var migrated MultiplayerListener
	if err := database.First(&migrated, "id = ?", legacy.ID).Error; err != nil {
		t.Fatalf("load migrated multiplayer listener: %v", err)
	}
	if migrated.WireGuardOptIn {
		t.Fatal("expected legacy multiplayer listener to default to no WireGuard opt-in")
	}
	if migrated.ToProtobuf().GetWireGuard() {
		t.Fatal("expected legacy multiplayer listener to restart in direct mode")
	}

	explicit := ListenerJobFromProtobuf(&clientpb.ListenerJob{
		Type: constants.MultiplayerModeStr,
		MultiConf: &clientpb.MultiplayerListenerReq{
			Host:      "127.0.0.1",
			Port:      31338,
			WireGuard: true,
		},
	}).MultiplayerListener
	explicit.ID = NewUUID()
	explicit.ListenerJobID = NewUUID()
	if err := database.Create(&explicit).Error; err != nil {
		t.Fatalf("insert explicit WireGuard multiplayer listener: %v", err)
	}

	var reloaded MultiplayerListener
	if err := database.First(&reloaded, "id = ?", explicit.ID).Error; err != nil {
		t.Fatalf("load explicit WireGuard multiplayer listener: %v", err)
	}
	if !reloaded.WireGuardOptIn || !reloaded.ToProtobuf().GetWireGuard() {
		t.Fatal("expected explicit WireGuard multiplayer listener to retain opt-in")
	}
}
