package aitools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCredentialsStoreToolsRoundTrip(t *testing.T) {
	setupStoreToolsTestEnv(t)

	exec := &executor{}
	ctx := context.Background()

	addedRaw, err := exec.CallTool(ctx, "credentials_add", `{
		"collection":"corp",
		"username":"alice",
		"plaintext":"Password123!",
		"hash":"8846f7eaee8fb117ad06bdd830b7586c",
		"hash_type_name":"NTLM"
	}`)
	if err != nil {
		t.Fatalf("add credential: %v", err)
	}

	var added credentialReadResult
	if err := json.Unmarshal([]byte(addedRaw), &added); err != nil {
		t.Fatalf("decode add result: %v", err)
	}
	if added.ID == "" {
		t.Fatal("expected credential ID to be returned")
	}
	if added.HashTypeName != "MD4" {
		t.Fatalf("expected NTLM alias to resolve to MD4, got %q", added.HashTypeName)
	}
	if !added.HasPlaintext || !added.HasHash {
		t.Fatalf("expected stored credential material to be reflected in add result, got %+v", added)
	}

	listedRaw, err := exec.CallTool(ctx, "credentials_list", `{}`)
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}

	var listed credentialsListResult
	if err := json.Unmarshal([]byte(listedRaw), &listed); err != nil {
		t.Fatalf("decode list result: %v", err)
	}
	if listed.Count != 1 || len(listed.Credentials) != 1 {
		t.Fatalf("unexpected credential count: %+v", listed)
	}

	prefix := added.ID[:8]
	readRaw, err := exec.CallTool(ctx, "credentials_read", fmt.Sprintf(`{"credential_id":%q}`, prefix))
	if err != nil {
		t.Fatalf("read credential: %v", err)
	}

	var read credentialReadResult
	if err := json.Unmarshal([]byte(readRaw), &read); err != nil {
		t.Fatalf("decode read result: %v", err)
	}
	if read.ID != added.ID {
		t.Fatalf("expected credential prefix lookup to resolve %q, got %+v", added.ID, read)
	}
	if read.Plaintext != "Password123!" || read.Hash != "8846f7eaee8fb117ad06bdd830b7586c" {
		t.Fatalf("unexpected credential body: %+v", read)
	}

	deletedRaw, err := exec.CallTool(ctx, "credentials_delete", fmt.Sprintf(`{"credential_id":%q}`, prefix))
	if err != nil {
		t.Fatalf("delete credential: %v", err)
	}

	var deleted credentialDeleteResult
	if err := json.Unmarshal([]byte(deletedRaw), &deleted); err != nil {
		t.Fatalf("decode delete result: %v", err)
	}
	if deleted.Deleted.ID != added.ID {
		t.Fatalf("unexpected deleted credential: %+v", deleted)
	}

	listedRaw, err = exec.CallTool(ctx, "credentials_list", `{}`)
	if err != nil {
		t.Fatalf("list credentials after delete: %v", err)
	}
	if err := json.Unmarshal([]byte(listedRaw), &listed); err != nil {
		t.Fatalf("decode list-after-delete result: %v", err)
	}
	if listed.Count != 0 || len(listed.Credentials) != 0 {
		t.Fatalf("expected empty credentials store after delete, got %+v", listed)
	}
}

func TestLootStoreToolsRoundTrip(t *testing.T) {
	setupStoreToolsTestEnv(t)

	exec := &executor{}
	ctx := context.Background()
	content := "top secret\nline two"

	addedRaw, err := exec.CallTool(ctx, "loot_add", `{
		"name":"operator-notes",
		"file_name":"notes.txt",
		"text":"top secret\nline two"
	}`)
	if err != nil {
		t.Fatalf("add loot: %v", err)
	}

	var added lootSummary
	if err := json.Unmarshal([]byte(addedRaw), &added); err != nil {
		t.Fatalf("decode add result: %v", err)
	}
	if added.ID == "" {
		t.Fatal("expected loot ID to be returned")
	}
	if added.FileTypeName != "TEXT" {
		t.Fatalf("expected text loot type, got %+v", added)
	}

	listedRaw, err := exec.CallTool(ctx, "loot_list", `{}`)
	if err != nil {
		t.Fatalf("list loot: %v", err)
	}

	var listed lootListResult
	if err := json.Unmarshal([]byte(listedRaw), &listed); err != nil {
		t.Fatalf("decode loot list result: %v", err)
	}
	if listed.Count != 1 || len(listed.Loot) != 1 {
		t.Fatalf("unexpected loot list result: %+v", listed)
	}

	prefix := added.ID[:8]
	readRaw, err := exec.CallTool(ctx, "loot_read", fmt.Sprintf(`{"loot_id":%q,"max_bytes":4}`, prefix))
	if err != nil {
		t.Fatalf("read loot: %v", err)
	}

	var read lootReadResult
	if err := json.Unmarshal([]byte(readRaw), &read); err != nil {
		t.Fatalf("decode loot read result: %v", err)
	}
	if read.ID != added.ID {
		t.Fatalf("expected loot prefix lookup to resolve %q, got %+v", added.ID, read)
	}
	if !read.Truncated || read.ReturnedByteLen != 4 || read.TotalByteLen != len(content) {
		t.Fatalf("unexpected loot read sizing: %+v", read)
	}
	if read.Text != content[:4] {
		t.Fatalf("unexpected loot text snippet: %+v", read)
	}
	decoded, err := base64.StdEncoding.DecodeString(read.DataBase64)
	if err != nil {
		t.Fatalf("decode loot data: %v", err)
	}
	if string(decoded) != content[:4] {
		t.Fatalf("unexpected loot data snippet: %q", string(decoded))
	}

	deletedRaw, err := exec.CallTool(ctx, "loot_delete", fmt.Sprintf(`{"loot_id":%q}`, prefix))
	if err != nil {
		t.Fatalf("delete loot: %v", err)
	}

	var deleted lootDeleteResult
	if err := json.Unmarshal([]byte(deletedRaw), &deleted); err != nil {
		t.Fatalf("decode loot delete result: %v", err)
	}
	if deleted.Deleted.ID != added.ID {
		t.Fatalf("unexpected deleted loot: %+v", deleted)
	}

	listedRaw, err = exec.CallTool(ctx, "loot_list", `{}`)
	if err != nil {
		t.Fatalf("list loot after delete: %v", err)
	}
	if err := json.Unmarshal([]byte(listedRaw), &listed); err != nil {
		t.Fatalf("decode loot list-after-delete result: %v", err)
	}
	if listed.Count != 0 || len(listed.Loot) != 0 {
		t.Fatalf("expected empty loot store after delete, got %+v", listed)
	}
}

func setupStoreToolsTestEnv(t *testing.T) {
	t.Helper()

	rootDir := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", rootDir)

	originalDB := db.Client
	testDB, err := gorm.Open(sqlite.Open(filepath.Join(rootDir, "store-tools-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(&models.Credential{}, &models.Loot{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	db.Client = testDB

	t.Cleanup(func() {
		sqlDB, err := testDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		db.Client = originalDB
	})
}
