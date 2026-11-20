package generate

import (
	"context"
	"errors"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetProfileDefinitionOrLegacy distinguishes an older server from a missing or
// broken profile. Only an unimplemented profile RPC permits the legacy path.
func GetProfileDefinitionOrLegacy(ctx context.Context, name string, con *console.SliverClient) (*clientpb.BuildProfileDocument, bool, error) {
	definition, err := GetProfileDefinition(ctx, name, con)
	if status.Code(err) == codes.Unimplemented && definition == nil {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if definition == nil {
		return nil, false, errors.New("profile has no definition")
	}
	return definition, false, nil
}
