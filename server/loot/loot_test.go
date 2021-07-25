package loot

import (
	"bytes"
	"crypto/rand"
	insecureRand "math/rand"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
)

var (
	data1 = randomData()
	data2 = []byte("hello world")

	name1 = "Test1"
	name2 = "Test2"
	name3 = "Test3"

	lootTestLog = log.NamedLogger("loot", "test")
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

func TestAddGetRmLoot(t *testing.T) {
	lootStore := GetLootStore()

	loot, err := lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name1,
		FileType: clientpb.FileType_BINARY,
		File: &commonpb.File{
			Name: name1,
			Data: data1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loot2, err := lootStore.GetContent(loot.LootID, true)
	if err != nil {
		t.Fatal(err)
	}
	if loot.Name != loot2.Name {
		t.Fatalf("Name mismatch %s != %s", loot.Name, loot2.Name)
	}
	if loot.Type != loot2.Type {
		t.Fatalf("Type mismatch %s != %s", loot.Type, loot2.Type)
	}
	if loot.LootID != loot2.LootID {
		t.Fatalf("LootID mismatch %s != %s", loot.LootID, loot2.LootID)
	}
	if loot2.File == nil {
		t.Fatal("Missing loot file")
	}
	if name1 != loot2.File.Name {
		t.Fatalf("FileName mismatch %s != %s", name1, loot2.File.Name)
	}
	if !bytes.Equal(data1, loot2.File.Data) {
		t.Fatalf("Loot file data mismatch %v != %v", data1, loot2.File.Data)
	}
	err = lootStore.Rm(loot.LootID)
	if err != nil {
		t.Fatal(err)
	}

	loot, err = lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name2,
		FileType: clientpb.FileType_TEXT,
		File: &commonpb.File{
			Name: name1,
			Data: data2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = lootStore.Rm(loot.LootID)
	if err != nil {
		t.Fatal(err)
	}

	loot, err = lootStore.Add(&clientpb.Loot{
		Type:           clientpb.LootType_LOOT_CREDENTIAL,
		Name:           name3,
		CredentialType: clientpb.CredentialType_USER_PASSWORD,
		Credential: &clientpb.Credential{
			User:     "admin",
			Password: "admin",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loot2, err = lootStore.GetContent(loot.LootID, true)
	if err != nil {
		t.Fatal(err)
	}

	if loot2.Credential == nil {
		t.Fatal("Missing credential")
	}
	if "admin" != loot2.Credential.User {
		t.Fatalf("Credential user mismatch %s != %s", "admin", loot2.Credential.User)
	}
	if "admin" != loot2.Credential.Password {
		t.Fatalf("Credential password mismatch %s != %s", "admin", loot2.Credential.Password)
	}
	err = lootStore.Rm(loot.LootID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAllLoot(t *testing.T) {
	lootStore := GetLootStore()
	_, err := lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name1,
		FileType: clientpb.FileType_BINARY,
		File: &commonpb.File{
			Name: name1,
			Data: data1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name2,
		FileType: clientpb.FileType_TEXT,
		File: &commonpb.File{
			Name: name1,
			Data: []byte("hello world"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lootStore.Add(&clientpb.Loot{
		Type:           clientpb.LootType_LOOT_CREDENTIAL,
		Name:           name3,
		CredentialType: clientpb.CredentialType_USER_PASSWORD,
		Credential: &clientpb.Credential{
			User:     "admin",
			Password: "admin",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	allLoot := lootStore.All()
	if allLoot == nil {
		t.Fatalf("Loot store returned nil for AllLoot")
	}
	if len(allLoot.Loot) != 3 {
		t.Fatalf("Expected all loot length of 3, but got %d", len(allLoot.Loot))
	}

	// Cleanup
	for _, loot := range allLoot.Loot {
		err = lootStore.Rm(loot.LootID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAllLootOf(t *testing.T) {
	lootStore := GetLootStore()
	_, err := lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name1,
		FileType: clientpb.FileType_BINARY,
		File: &commonpb.File{
			Name: name1,
			Data: data1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name2,
		FileType: clientpb.FileType_TEXT,
		File: &commonpb.File{
			Name: name1,
			Data: []byte("hello world"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lootStore.Add(&clientpb.Loot{
		Type:           clientpb.LootType_LOOT_CREDENTIAL,
		Name:           name3,
		CredentialType: clientpb.CredentialType_USER_PASSWORD,
		Credential: &clientpb.Credential{
			User:     "admin",
			Password: "admin",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// AllOf
	fileLoot := lootStore.AllOf(clientpb.LootType_LOOT_FILE).Loot
	if len(fileLoot) != 2 {
		t.Fatalf("AllOf file returned %d expected 2", len(fileLoot))
	}
	credLoot := lootStore.AllOf(clientpb.LootType_LOOT_CREDENTIAL).Loot
	if len(credLoot) != 1 {
		t.Fatalf("AllOf cred returned %d expected 2", len(credLoot))
	}

	// Cleanup
	for _, loot := range lootStore.All().Loot {
		err = lootStore.Rm(loot.LootID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLootErrors(t *testing.T) {
	lootStore := GetLootStore()
	loot, err := lootStore.Add(&clientpb.Loot{
		Type:     clientpb.LootType_LOOT_FILE,
		Name:     name1,
		FileType: clientpb.FileType_BINARY,
		File: &commonpb.File{
			Name: name1,
			Data: data1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = lootStore.GetContent("foobar", true)
	if err == nil {
		t.Fatal("Expected invalid loot id error")
	}

	randomID, _ := uuid.NewV4()
	_, err = lootStore.GetContent(randomID.String(), true)
	if err == nil {
		t.Fatal("Expected loot not found error")
	}

	err = lootStore.Rm(loot.LootID)
	if err != nil {
		t.Fatal(err)
	}
}
