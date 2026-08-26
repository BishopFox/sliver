package armory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
)

func resetPackageCache(t *testing.T) {
	t.Helper()
	pkgCache.Range(func(key, value any) bool {
		pkgCache.Delete(key)
		return true
	})
	t.Cleanup(func() {
		pkgCache.Range(func(key, value any) bool {
			pkgCache.Delete(key)
			return true
		})
	})
}

func testExtensionCacheEntry() pkgCacheEntry {
	manifest := &extensions.ExtensionManifest{
		Name:        "ADCS Request (Remote)",
		PackageName: "remote-adcs-package",
		Version:     "v0.1.5",
		ArmoryName:  "Default Armory",
		ArmoryPK:    "armory-public-key",
		ExtCommand: []*extensions.ExtCommand{
			{CommandName: "remote-adcs-request"},
			{CommandName: "remote-adcs-request-on-behalf"},
		},
	}
	return pkgCacheEntry{
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-public-key"},
		Pkg: ArmoryPackage{
			Name:        "Remote ADCS package",
			CommandName: "remote-adcs-index",
		},
		Extension: manifest,
		ID:        "remote-adcs-request-id",
	}
}

func TestGetPackageForCommandAcceptsStableExtensionIdentities(t *testing.T) {
	resetPackageCache(t)
	entry := testExtensionCacheEntry()
	pkgCache.Store(entry.ID, entry)

	for _, name := range []string{
		"remote-adcs-index",
		"remote-adcs-request",
		"remote-adcs-request-on-behalf",
		"remote-adcs-package",
	} {
		t.Run(name, func(t *testing.T) {
			found, err := getPackageForCommand(name, "armory-public-key", "v0.1.5")
			if err != nil {
				t.Fatalf("lookup %q failed: %v", name, err)
			}
			if found.ID != entry.ID {
				t.Fatalf("lookup %q returned %q, want %q", name, found.ID, entry.ID)
			}
		})
	}

	if _, err := getPackageForCommand("remote-adcs-package", "other-armory", "v0.1.5"); !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("wrong-armory lookup error = %v, want %v", err, ErrPackageNotFound)
	}
	if _, err := getPackageForCommand("remote-adcs-package", "armory-public-key", "v0.1.6"); !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("newer-version lookup error = %v, want %v", err, ErrPackageNotFound)
	}
	for _, name := range []string{"ADCS Request (Remote)", "Remote ADCS package", "unknown-package"} {
		if _, err := getPackageForCommand(name, "", ""); !errors.Is(err, ErrPackageNotFound) {
			t.Fatalf("unstable identity %q lookup error = %v, want %v", name, err, ErrPackageNotFound)
		}
	}
}

func TestPackageMatchesInstallNameLeavesAliasesCommandOnly(t *testing.T) {
	entry := pkgCacheEntry{
		Pkg: ArmoryPackage{
			Name:        "Human Alias Name",
			CommandName: "alias-command",
			IsAlias:     true,
		},
		Alias: &alias.AliasManifest{CommandName: "alias-command"},
	}
	if packageMatchesInstallName(entry, "Human Alias Name") {
		t.Fatal("alias unexpectedly matched its display name")
	}
	if !packageMatchesInstallName(entry, "alias-command") {
		t.Fatal("alias did not match its command name")
	}
}

func TestExtensionManifestsMatch(t *testing.T) {
	manifest := func(name, packageName string, commands ...string) *extensions.ExtensionManifest {
		extCommands := make([]*extensions.ExtCommand, 0, len(commands))
		for _, command := range commands {
			extCommands = append(extCommands, &extensions.ExtCommand{CommandName: command})
		}
		return &extensions.ExtensionManifest{Name: name, PackageName: packageName, ExtCommand: extCommands}
	}

	tests := []struct {
		name   string
		local  *extensions.ExtensionManifest
		latest *extensions.ExtensionManifest
		want   bool
	}{
		{
			name:   "v2 package identity",
			local:  manifest("Old Display Name", "stable-package", "old-command"),
			latest: manifest("New Display Name", "stable-package", "new-command"),
			want:   true,
		},
		{
			name:   "different v2 packages",
			local:  manifest("Shared Display Name", "package-one", "shared-command"),
			latest: manifest("Shared Display Name", "package-two", "shared-command"),
			want:   false,
		},
		{
			name:   "v2 does not downgrade to legacy identity",
			local:  manifest("Human Display Name", "stable-package", "shared-command"),
			latest: manifest("legacy-command", "", "shared-command"),
			want:   false,
		},
		{
			name:   "legacy command identity",
			local:  manifest("legacy-command", "", "legacy-command"),
			latest: manifest("Human Display Name", "stable-package", "other-command", "legacy-command"),
			want:   true,
		},
		{
			name:   "display names are not identity",
			local:  manifest("Shared Name", "", "old-command"),
			latest: manifest("Shared Name", "stable-package", "new-command"),
			want:   false,
		},
		{
			name:   "unrelated legacy package",
			local:  manifest("Old Name", "", "old-command"),
			latest: manifest("New Name", "stable-package", "new-command"),
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extensionManifestsMatch(test.local, test.latest); got != test.want {
				t.Fatalf("extensionManifestsMatch() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestExtensionUpdateDisplayNameResolvesPackage(t *testing.T) {
	resetPackageCache(t)
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())

	installedManifest := []byte(`{
		"name":"ADCS Request (Remote)",
		"package_name":"remote-adcs-package",
		"version":"v0.1.3",
		"commands":[{
			"command_name":"remote-adcs-request",
			"help":"Request a certificate",
			"files":[{"os":"windows","arch":"amd64","path":"adcs_request.x64.o"}]
		}]
	}`)
	installDir := filepath.Join(assets.GetExtensionsDir(), "remote-adcs-request")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, extensions.ManifestFileName), installedManifest, 0o600); err != nil {
		t.Fatal(err)
	}

	entry := testExtensionCacheEntry()
	pkgCache.Store(entry.ID, entry)
	otherEntry := testExtensionCacheEntry()
	otherEntry.ID = "other-armory-package-id"
	otherEntry.ArmoryConfig = &assets.ArmoryConfig{PublicKey: "other-armory"}
	otherManifest := *otherEntry.Extension
	otherManifest.Version = "v9.0.0"
	otherManifest.ArmoryName = "Other Armory"
	otherManifest.ArmoryPK = "other-armory"
	otherEntry.Extension = &otherManifest
	pkgCache.Store(otherEntry.ID, otherEntry)

	updates := checkForExtensionUpdates("armory-public-key")
	versionInfo, ok := updates["ADCS Request (Remote)"]
	if !ok {
		t.Fatal("display-name update was not detected")
	}
	if versionInfo.OldVersion != "v0.1.3" || versionInfo.NewVersion != "v0.1.5" {
		t.Fatalf("unexpected versions: %#v", versionInfo)
	}
	if versionInfo.PackageID != entry.ID {
		t.Fatalf("update package ID = %q, want %q", versionInfo.PackageID, entry.ID)
	}

	found := packageCacheLookupByID(versionInfo.PackageID)
	if found == nil {
		t.Fatal("detected update package is missing from the cache")
	}
	if found.ID != entry.ID {
		t.Fatalf("resolved package %q, want %q", found.ID, entry.ID)
	}
}
