package loot

import (
	"bytes"
	"crypto/rand"
	insecureRand "math/rand"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/gofrs/uuid"
)

var (
	data1 = randomData()
	data2 = []byte("hello world")

	name1 = "Test1"
	name2 = "Test2"
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

func TestAddGetRmLoot(t *testing.T) {
	lootStore := GetLootStore()

	loot, err := lootStore.Add(&clientpb.Loot{
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
	loot2, err := lootStore.GetContent(loot.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if loot.Name != loot2.Name {
		t.Fatalf("Name mismatch %s != %s", loot.Name, loot2.Name)
	}
	if loot.ID != loot2.ID {
		t.Fatalf("LootID mismatch %s != %s", loot.ID, loot2.ID)
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
	err = lootStore.Rm(loot.ID)
	if err != nil {
		t.Fatal(err)
	}

	loot, err = lootStore.Add(&clientpb.Loot{
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
	err = lootStore.Rm(loot.ID)
	if err != nil {
		t.Fatal(err)
	}

}

func TestAllLoot(t *testing.T) {
	lootStore := GetLootStore()
	_, err := lootStore.Add(&clientpb.Loot{
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

	allLoot := lootStore.All()
	if allLoot == nil {
		t.Fatalf("Loot store returned nil for AllLoot")
	}
	if len(allLoot.Loot) != 2 {
		t.Fatalf("Expected all loot length of 3, but got %d", len(allLoot.Loot))
	}

	// Cleanup
	for _, loot := range allLoot.Loot {
		err = lootStore.Rm(loot.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAllLootOf(t *testing.T) {
	lootStore := GetLootStore()
	_, err := lootStore.Add(&clientpb.Loot{
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

	// Cleanup
	for _, loot := range lootStore.All().Loot {
		err = lootStore.Rm(loot.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLootErrors(t *testing.T) {
	lootStore := GetLootStore()
	loot, err := lootStore.Add(&clientpb.Loot{
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

	err = lootStore.Rm(loot.ID)
	if err != nil {
		t.Fatal(err)
	}
}
