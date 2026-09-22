package creds

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

const (
	pinnedHashcatVersion       = "v7.1.2-armory.1"
	pinnedHashcatCommit        = "deadd52de22e2753e9caaff6d1b92e5a22a35626"
	pinnedHashcatModuleCount   = 595
	pinnedHashcatCatalogSHA256 = "1c041b6370a37ccf07ae71ff29205c56c67eeb7dbe1f7bbc134e0373ecd19e4f"
	pinnedHashcatModeIDsSHA256 = "bb206df105890129937d0c688101a3065aad2ca45eea7765bfda60c40f232a63"
	pinnedHashcatEnumsSHA256   = "989144b3d2a57f9659bc5aa53df42403ba78b08e5a6fab616d43c14e0352e023"
)

type hashcatCatalogEntry struct {
	mode        int32
	enumName    string
	description string
}

func TestHashcatV7CatalogCoverage(t *testing.T) {
	version := pinnedHashcatVersion + "@" + pinnedHashcatCommit
	validateHashcatCatalogCardinality(t, version)
	catalog := collectHashcatCatalog(t, version)
	validateHashTypeValueRoundTrips(t)
	validateHashTypeDescriptionParity(t)
	validateHashcatCatalogDigests(t, version, catalog)
}

func validateHashcatCatalogCardinality(t *testing.T, version string) {
	t.Helper()

	if got := len(clientpb.HashType_name); got != pinnedHashcatModuleCount+1 {
		t.Fatalf("%s HashType_name contains %d values, want %d modules plus INVALID", version, got, pinnedHashcatModuleCount)
	}
	if got := len(clientpb.HashType_value); got != pinnedHashcatModuleCount+1 {
		t.Fatalf("%s HashType_value contains %d values, want %d modules plus INVALID", version, got, pinnedHashcatModuleCount)
	}
	if got := len(hashTypes); got != pinnedHashcatModuleCount {
		t.Fatalf("%s hashTypes contains %d descriptions, want %d", version, got, pinnedHashcatModuleCount)
	}
}

func collectHashcatCatalog(t *testing.T, version string) []hashcatCatalogEntry {
	t.Helper()

	catalog := make([]hashcatCatalogEntry, 0, pinnedHashcatModuleCount)
	seenDescriptions := make(map[string]int32, pinnedHashcatModuleCount)

	for mode, enumName := range clientpb.HashType_name {
		if mode == int32(clientpb.HashType_INVALID) {
			if enumName != "INVALID" || mode != 9999 {
				t.Errorf("invalid sentinel is %d -> %q, want 9999 -> INVALID", mode, enumName)
			}
			continue
		}
		if mode < 0 {
			t.Errorf("HashType_name contains negative mode %d (%s)", mode, enumName)
			continue
		}

		description, ok := hashTypes[enumName]
		if !ok {
			t.Errorf("hashTypes is missing %d -> %s", mode, enumName)
			continue
		}
		if previous, duplicate := seenDescriptions[description]; duplicate {
			t.Errorf("Hashcat name %q is shared by modes %d and %d", description, previous, mode)
		}
		seenDescriptions[description] = mode
		if got, ok := clientpb.HashType_value[enumName]; !ok || got != mode {
			t.Errorf("HashType_value[%q] = %d, %v; want %d, true", enumName, got, ok, mode)
		}
		if got := clientpb.HashType(mode).String(); got != enumName {
			t.Errorf("HashType(%d).String() = %q, want %q", mode, got, enumName)
		}
		catalog = append(catalog, hashcatCatalogEntry{mode: mode, enumName: enumName, description: description})
	}
	if got := len(catalog); got != pinnedHashcatModuleCount {
		t.Fatalf("%s catalog contains %d real modes, want %d", version, got, pinnedHashcatModuleCount)
	}
	return catalog
}

func validateHashTypeValueRoundTrips(t *testing.T) {
	t.Helper()

	for enumName, mode := range clientpb.HashType_value {
		canonical, ok := clientpb.HashType_name[mode]
		if !ok || canonical != enumName {
			t.Errorf("HashType alias or orphan %q -> %d; canonical name is %q, %v", enumName, mode, canonical, ok)
		}
	}
}

func validateHashTypeDescriptionParity(t *testing.T) {
	t.Helper()

	for enumName, description := range hashTypes {
		mode, ok := clientpb.HashType_value[enumName]
		if !ok || mode == int32(clientpb.HashType_INVALID) {
			t.Errorf("hashTypes contains non-catalog key %q -> %q", enumName, description)
		}
		if description == "" {
			t.Errorf("hashTypes[%q] has an empty description", enumName)
		}
	}
}

func validateHashcatCatalogDigests(t *testing.T, version string, catalog []hashcatCatalogEntry) {
	t.Helper()

	sort.Slice(catalog, func(i, j int) bool { return catalog[i].mode < catalog[j].mode })
	var catalogRows strings.Builder
	var modeRows strings.Builder
	var enumRows strings.Builder
	for _, entry := range catalog {
		fmt.Fprintf(&catalogRows, "%d\t%s\n", entry.mode, entry.description)
		fmt.Fprintf(&modeRows, "%05d\n", entry.mode)
		fmt.Fprintf(&enumRows, "%d\t%s\t%s\n", entry.mode, entry.enumName, entry.description)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(catalogRows.String()))); got != pinnedHashcatCatalogSHA256 {
		t.Errorf("%s mode/name catalog SHA-256 = %s, want %s", version, got, pinnedHashcatCatalogSHA256)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(modeRows.String()))); got != pinnedHashcatModeIDsSHA256 {
		t.Errorf("%s mode ID SHA-256 = %s, want %s", version, got, pinnedHashcatModeIDsSHA256)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(enumRows.String()))); got != pinnedHashcatEnumsSHA256 {
		t.Errorf("%s mode/enum/name SHA-256 = %s, want %s", version, got, pinnedHashcatEnumsSHA256)
	}
}

func TestHashcatIdentifierCollisionResolution(t *testing.T) {
	want := map[int32]string{
		141:   "EPISERVER_6_X_NET_4_MODE_141",
		600:   "BLAKE2B_256",
		1441:  "EPISERVER_6_X_NET_4_MODE_1441",
		2611:  "VBULLETIN_V3_8_5_MODE_2611",
		2711:  "VBULLETIN_V3_8_5_MODE_2711",
		4110:  "MD5_SALT_MD5_PASS_SALT_MODE_4110",
		4410:  "MD5_SHA1_PASS_SALT_MODE_4410",
		4420:  "MD5_SHA1_PASS_SALT_MODE_4420",
		4710:  "SHA1_MD5_PASS_SALT_MODE_4710",
		14700: "ITUNES_BACKUP_10_0_MODE_14700",
		14800: "ITUNES_BACKUP_10_0_MODE_14800",
		20710: "SHA256_SHA256_PASS_SALT_MODE_20710",
		20730: "SHA256_SHA256_PASS_SALT_MODE_20730",
		21100: "SHA1_MD5_PASS_SALT_MODE_21100",
		21900: "MD5_MD5_MD5_PASS_SALT1_SALT2_MODE_21900",
		31700: "MD5_MD5_MD5_PASS_SALT1_SALT2_MODE_31700",
		31800: "HASHCAT_1PASSWORD_MOBILEKEYCHAIN_1PASSWORD_8",
		33100: "MD5_SALT_MD5_PASS_SALT_MODE_33100",
		34800: "BLAKE2B_256_MODE_34800",
		6600:  "HASHCAT_1PASSWORD_AGILEKEYCHAIN",
		8200:  "HASHCAT_1PASSWORD_CLOUDKEYCHAIN",
		11600: "HASHCAT_7_ZIP",
	}
	for mode, enumName := range want {
		if got := clientpb.HashType_name[mode]; got != enumName {
			t.Errorf("HashType_name[%d] = %q, want collision-safe %q", mode, got, enumName)
		}
	}

	groups := map[string][]int32{}
	for mode, enumName := range clientpb.HashType_name {
		if mode == int32(clientpb.HashType_INVALID) {
			continue
		}
		description, ok := hashTypes[enumName]
		if !ok {
			t.Fatalf("hashTypes is missing collision input %d -> %s", mode, enumName)
		}
		base := normalizeHashcatIdentifier(description)
		groups[base] = append(groups[base], mode)
	}
	collisions := 0
	for _, modes := range groups {
		if len(modes) > 1 {
			collisions++
		}
	}
	if collisions != 8 {
		t.Errorf("normalized catalog contains %d collision groups, want 8", collisions)
	}
}

func normalizeHashcatIdentifier(name string) string {
	var builder strings.Builder
	underscore := false
	for _, r := range strings.ToUpper(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			underscore = false
			continue
		}
		if builder.Len() > 0 && !underscore {
			builder.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
