// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mautrix

import (
	"fmt"
	"regexp"
	"strconv"
)

// RespVersions is the JSON response for https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientversions
type RespVersions struct {
	Versions         []SpecVersion   `json:"versions"`
	UnstableFeatures map[string]bool `json:"unstable_features"`
}

func (versions *RespVersions) ContainsFunc(match func(found SpecVersion) bool) bool {
	if versions == nil {
		return false
	}
	for _, found := range versions.Versions {
		if match(found) {
			return true
		}
	}
	return false
}

func (versions *RespVersions) Contains(version SpecVersion) bool {
	return versions.ContainsFunc(func(found SpecVersion) bool {
		return found == version
	})
}

func (versions *RespVersions) ContainsGreaterOrEqual(version SpecVersion) bool {
	return versions.ContainsFunc(func(found SpecVersion) bool {
		return found.GreaterThan(version) || found == version
	})
}

func (versions *RespVersions) GetLatest() (latest SpecVersion) {
	if versions == nil {
		return
	}
	for _, ver := range versions.Versions {
		if ver.GreaterThan(latest) {
			latest = ver
		}
	}
	return
}

type UnstableFeature struct {
	UnstableFlag string
	SpecVersion  SpecVersion
}

var (
	FeatureAsyncUploads           = UnstableFeature{UnstableFlag: "fi.mau.msc2246.stable", SpecVersion: SpecV17}
	FeatureAppservicePing         = UnstableFeature{UnstableFlag: "fi.mau.msc2659.stable", SpecVersion: SpecV17}
	FeatureAuthenticatedMedia     = UnstableFeature{UnstableFlag: "org.matrix.msc3916.stable", SpecVersion: SpecV111}
	FeatureMutualRooms            = UnstableFeature{UnstableFlag: "uk.half-shot.msc2666.query_mutual_rooms"}
	FeatureUserRedaction          = UnstableFeature{UnstableFlag: "org.matrix.msc4194"}
	FeatureViewRedactedContent    = UnstableFeature{UnstableFlag: "fi.mau.msc2815"}
	FeatureAccountModeration      = UnstableFeature{UnstableFlag: "uk.timedout.msc4323"}
	FeatureUnstableProfileFields  = UnstableFeature{UnstableFlag: "uk.tcpip.msc4133"}
	FeatureArbitraryProfileFields = UnstableFeature{UnstableFlag: "uk.tcpip.msc4133.stable", SpecVersion: SpecV116}
	FeatureRedactSendAsEvent      = UnstableFeature{UnstableFlag: "com.beeper.msc4169"}

	BeeperFeatureHungry                = UnstableFeature{UnstableFlag: "com.beeper.hungry"}
	BeeperFeatureBatchSending          = UnstableFeature{UnstableFlag: "com.beeper.batch_sending"}
	BeeperFeatureRoomYeeting           = UnstableFeature{UnstableFlag: "com.beeper.room_yeeting"}
	BeeperFeatureAutojoinInvites       = UnstableFeature{UnstableFlag: "com.beeper.room_create_autojoin_invites"}
	BeeperFeatureArbitraryProfileMeta  = UnstableFeature{UnstableFlag: "com.beeper.arbitrary_profile_meta"}
	BeeperFeatureAccountDataMute       = UnstableFeature{UnstableFlag: "com.beeper.account_data_mute"}
	BeeperFeatureInboxState            = UnstableFeature{UnstableFlag: "com.beeper.inbox_state"}
	BeeperFeatureArbitraryMemberChange = UnstableFeature{UnstableFlag: "com.beeper.arbitrary_member_change"}
)

func (versions *RespVersions) Supports(feature UnstableFeature) bool {
	if versions == nil {
		return false
	}
	return versions.UnstableFeatures[feature.UnstableFlag] ||
		(!feature.SpecVersion.IsEmpty() && versions.ContainsGreaterOrEqual(feature.SpecVersion))
}

type SpecVersionFormat int

const (
	SpecVersionFormatUnknown SpecVersionFormat = iota
	SpecVersionFormatR
	SpecVersionFormatV
)

var (
	SpecR000 = MustParseSpecVersion("r0.0.0")
	SpecR001 = MustParseSpecVersion("r0.0.1")
	SpecR010 = MustParseSpecVersion("r0.1.0")
	SpecR020 = MustParseSpecVersion("r0.2.0")
	SpecR030 = MustParseSpecVersion("r0.3.0")
	SpecR040 = MustParseSpecVersion("r0.4.0")
	SpecR050 = MustParseSpecVersion("r0.5.0")
	SpecR060 = MustParseSpecVersion("r0.6.0")
	SpecR061 = MustParseSpecVersion("r0.6.1")
	SpecV11  = MustParseSpecVersion("v1.1")
	SpecV12  = MustParseSpecVersion("v1.2")
	SpecV13  = MustParseSpecVersion("v1.3")
	SpecV14  = MustParseSpecVersion("v1.4")
	SpecV15  = MustParseSpecVersion("v1.5")
	SpecV16  = MustParseSpecVersion("v1.6")
	SpecV17  = MustParseSpecVersion("v1.7")
	SpecV18  = MustParseSpecVersion("v1.8")
	SpecV19  = MustParseSpecVersion("v1.9")
	SpecV110 = MustParseSpecVersion("v1.10")
	SpecV111 = MustParseSpecVersion("v1.11")
	SpecV112 = MustParseSpecVersion("v1.12")
	SpecV113 = MustParseSpecVersion("v1.13")
	SpecV114 = MustParseSpecVersion("v1.14")
	SpecV115 = MustParseSpecVersion("v1.15")
	SpecV116 = MustParseSpecVersion("v1.16")
)

func (svf SpecVersionFormat) String() string {
	switch svf {
	case SpecVersionFormatR:
		return "r"
	case SpecVersionFormatV:
		return "v"
	default:
		return ""
	}
}

type SpecVersion struct {
	Format SpecVersionFormat
	Major  int
	Minor  int
	Patch  int

	Raw string
}

var legacyVersionRegex = regexp.MustCompile(`^r(\d+)\.(\d+)\.(\d+)$`)
var modernVersionRegex = regexp.MustCompile(`^v(\d+)\.(\d+)$`)

func MustParseSpecVersion(version string) SpecVersion {
	sv, err := ParseSpecVersion(version)
	if err != nil {
		panic(err)
	}
	return sv
}

func ParseSpecVersion(version string) (sv SpecVersion, err error) {
	sv.Raw = version
	if parts := modernVersionRegex.FindStringSubmatch(version); parts != nil {
		sv.Major, _ = strconv.Atoi(parts[1])
		sv.Minor, _ = strconv.Atoi(parts[2])
		sv.Format = SpecVersionFormatV
	} else if parts = legacyVersionRegex.FindStringSubmatch(version); parts != nil {
		sv.Major, _ = strconv.Atoi(parts[1])
		sv.Minor, _ = strconv.Atoi(parts[2])
		sv.Patch, _ = strconv.Atoi(parts[3])
		sv.Format = SpecVersionFormatR
	} else {
		err = fmt.Errorf("version '%s' doesn't match either known syntax", version)
	}
	return
}

func (sv *SpecVersion) UnmarshalText(version []byte) error {
	*sv, _ = ParseSpecVersion(string(version))
	return nil
}

func (sv *SpecVersion) MarshalText() ([]byte, error) {
	return []byte(sv.String()), nil
}

func (sv SpecVersion) String() string {
	switch sv.Format {
	case SpecVersionFormatR:
		return fmt.Sprintf("r%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	case SpecVersionFormatV:
		return fmt.Sprintf("v%d.%d", sv.Major, sv.Minor)
	default:
		return sv.Raw
	}
}

func (sv SpecVersion) IsEmpty() bool {
	return sv.Format == SpecVersionFormatUnknown && sv.Raw == ""
}

func (sv SpecVersion) LessThan(other SpecVersion) bool {
	return sv != other && !sv.GreaterThan(other)
}

func (sv SpecVersion) GreaterThan(other SpecVersion) bool {
	return sv.Format > other.Format ||
		(sv.Format == other.Format && sv.Major > other.Major) ||
		(sv.Format == other.Format && sv.Major == other.Major && sv.Minor > other.Minor) ||
		(sv.Format == other.Format && sv.Major == other.Major && sv.Minor == other.Minor && sv.Patch > other.Patch)
}
