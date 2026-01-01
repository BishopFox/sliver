package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrInvalidBeaconID - Invalid Beacon ID in request
	ErrInvalidBeaconID = status.Error(codes.InvalidArgument, "Invalid beacon ID")
	// ErrInvalidBeaconTaskID - Invalid Beacon ID in request
	ErrInvalidBeaconTaskID = status.Error(codes.InvalidArgument, "Invalid beacon task ID")

	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = status.Error(codes.InvalidArgument, "Invalid session ID")

	// ErrMissingRequestField - Returned when a request does not contain a commonpb.Request
	ErrMissingRequestField = status.Error(codes.InvalidArgument, "Missing session request field")
	// ErrAsyncNotSupported - Unsupported mode / command type
	ErrAsyncNotSupported = status.Error(codes.Unavailable, "Async not supported for this command")
	// ErrDatabaseFailure - Generic database failure error (real error is logged)
	ErrDatabaseFailure = status.Error(codes.Internal, "Database operation failed")

	// ErrInvalidName - Invalid name
	ErrInvalidName = status.Error(codes.InvalidArgument, "Invalid session name, alphanumerics and _-. only")
	// ErrBuildExists
	ErrBuildExists = status.Error(codes.AlreadyExists, "Build already exists")

	ErrInvalidBeaconTaskCancelState = status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid task state, must be '%s' to cancel", models.PENDING))
)

func rpcError(err error) error {
	if err == nil {
		return nil
	}

	if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, core.ErrImplantTimeout):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, core.ErrUnknownMessageType):
		return status.Error(codes.Unimplemented, err.Error())
	case errors.Is(err, core.ErrInvalidTunnelID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrDuplicateHosts):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, core.ErrDuplicateExternalBuilderName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, db.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case os.IsNotExist(err):
		return status.Error(codes.NotFound, err.Error())
	case os.IsPermission(err):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
