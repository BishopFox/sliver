package generate

import (
	"errors"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

// NewImplantConfigSnapshot creates the immutable configuration that a build
// embeds and references. The source profile ID is returned separately so build
// listing and profile deletion can retain their existing relationship without
// making the snapshot itself part of the mutable profile.
func NewImplantConfigSnapshot(config *clientpb.ImplantConfig) (*clientpb.ImplantConfig, string, error) {
	if config == nil {
		return nil, "", errors.New("implant config cannot be nil")
	}

	snapshot, ok := proto.Clone(config).(*clientpb.ImplantConfig)
	if !ok {
		return nil, "", errors.New("failed to clone implant config")
	}
	configID, err := uuid.NewV4()
	if err != nil {
		return nil, "", err
	}

	sourceProfileID := snapshot.ImplantProfileID
	if sourceProfileID != "" {
		profileID, err := uuid.FromString(sourceProfileID)
		if err != nil || profileID == uuid.Nil {
			return nil, "", errors.New("invalid source implant profile ID")
		}
		sourceProfileID = profileID.String()
	}
	snapshot.ID = configID.String()
	snapshot.ImplantProfileID = ""
	snapshot.ImplantBuilds = nil
	for _, c2 := range snapshot.C2 {
		if c2 != nil {
			c2.ID = ""
		}
	}
	return snapshot, sourceProfileID, nil
}
