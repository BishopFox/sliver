package transport

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

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
	"encoding/json"
	"errors"

	"github.com/reeflective/team/server"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
)

// bufferingOptions returns a list of server options with max send/receive
// message size, which value is that of the ServerMaxMessageSize variable (2GB).
func bufferingOptions() (options []grpc.ServerOption) {
	options = append(options,
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	)

	return
}

// logMiddlewareOptions is a set of logging middleware options
// preconfigured to perform the following tasks:
// - Log all connections/disconnections to/from the teamserver listener.
// - Log all raw client requests into a teamserver audit file (see server.AuditLog()).
func logMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
	var requestOpts []grpc.UnaryServerInterceptor
	var streamOpts []grpc.StreamServerInterceptor

	cfg := s.GetConfig()

	// Audit-log all requests. Any failure to audit-log the requests
	// of this server will themselves be logged to the root teamserver log.
	auditLog, err := s.AuditLogger()
	if err != nil {
		return nil, err
	}

	requestOpts = append(requestOpts, auditLogUnaryServerInterceptor(s, auditLog))

	requestOpts = append(requestOpts,
		grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)

	streamOpts = append(streamOpts,
		grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)

	// Logging interceptors
	logrusEntry := s.NamedLogger("transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(codeToLevel),
	}

	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	requestOpts = append(requestOpts,
		grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
		grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
			return cfg.Log.GRPCUnaryPayloads
		}),
	)

	streamOpts = append(streamOpts,
		grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
		grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
			return cfg.Log.GRPCStreamPayloads
		}),
	)

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(requestOpts...),
		grpc.ChainStreamInterceptor(streamOpts...),
	}, nil
}

// tlsAuthMiddlewareOptions is a set of transport security options which will use
// the preconfigured teamserver TLS (credentials) configuration to authenticate
// incoming client connections. The authentication is Mutual TLS, used because
// all teamclients will connect with a known TLS credentials set.
func tlsAuthMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
	var options []grpc.ServerOption

	tlsConfig, err := s.UsersTLSConfig()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)
	options = append(options, grpc.Creds(creds))

	return options, nil
}

// initAuthMiddleware - Initialize middleware logger.
func (ts *teamserver) initAuthMiddleware() ([]grpc.ServerOption, error) {
	var requestOpts []grpc.UnaryServerInterceptor
	var streamOpts []grpc.StreamServerInterceptor

	// Authentication interceptors.
	if ts.conn == nil {
		// All remote connections are users who need authentication.
		requestOpts = append(requestOpts,
			grpc_auth.UnaryServerInterceptor(ts.tokenAuthFunc),
		)

		streamOpts = append(streamOpts,
			grpc_auth.StreamServerInterceptor(ts.tokenAuthFunc),
		)
	} else {
		// Local in-memory connections have no auth.
		requestOpts = append(requestOpts,
			grpc_auth.UnaryServerInterceptor(serverAuthFunc),
		)
		streamOpts = append(streamOpts,
			grpc_auth.StreamServerInterceptor(serverAuthFunc),
		)
	}

	// Return middleware for all requests and stream interactions in gRPC.
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(requestOpts...),
		grpc.ChainStreamInterceptor(streamOpts...),
	}, nil
}

type contextKey int

const (
	Transport contextKey = iota
	Operator
)

func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, Transport, "local")
	newCtx = context.WithValue(newCtx, Operator, "server")

	return newCtx, nil
}

// tokenAuthFunc uses the core reeflective/team/server to authenticate user requests.
func (ts *teamserver) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := ts.NamedLogger("transport", "grpc")
	log.Debugf("Auth interceptor checking user token ...")

	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Let our core teamserver driver authenticate the user.
	// The teamserver has its credentials, tokens and everything in database.
	user, authorized, err := ts.UserAuthenticate(rawToken)
	if err != nil || !authorized || user == "" {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	newCtx := context.WithValue(ctx, Transport, "mtls")
	newCtx = context.WithValue(newCtx, Operator, user)

	return newCtx, nil
}

type auditUnaryLogMsg struct {
	Request  string `json:"request"`
	Method   string `json:"method"`
	Session  string `json:"session,omitempty"`
	Beacon   string `json:"beacon,omitempty"`
	RemoteIP string `json:"remote_ip"`
	User     string `json:"user"`
}

func auditLogUnaryServerInterceptor(ts *server.Server, auditLog *logrus.Logger) grpc.UnaryServerInterceptor {
	log := ts.NamedLogger("grpc", "audit")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		rawRequest, err := json.Marshal(req)
		if err != nil {
			log.Errorf("Failed to serialize %s", err)
			return
		}

		log.Debugf("Raw request: %s", string(rawRequest))
		session, beacon, err := getActiveTarget(log, rawRequest)
		if err != nil {
			log.Errorf("Middleware failed to insert details: %s", err)
		}

		p, _ := peer.FromContext(ctx)

		// Construct Log Message
		msg := &auditUnaryLogMsg{
			Request:  string(rawRequest),
			Method:   info.FullMethod,
			User:     getUser(p),
			RemoteIP: p.Addr.String(),
		}
		if session != nil {
			sessionJSON, _ := json.Marshal(session)
			msg.Session = string(sessionJSON)
		}
		if beacon != nil {
			beaconJSON, _ := json.Marshal(beacon)
			msg.Beacon = string(beaconJSON)
		}

		msgData, _ := json.Marshal(msg)
		auditLog.Info(string(msgData))

		resp, err := handler(ctx, req)

		return resp, err
	}
}

func getUser(client *peer.Peer) string {
	tlsAuth, ok := client.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ""
	}
	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		return ""
	}
	if tlsAuth.State.VerifiedChains[0][0].Subject.CommonName != "" {
		return tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
	}
	return ""
}

func getActiveTarget(middlewareLog *logrus.Entry, rawRequest []byte) (*clientpb.Session, *clientpb.Beacon, error) {
	var activeBeacon *clientpb.Beacon
	var activeSession *clientpb.Session

	var request map[string]interface{}
	err := json.Unmarshal(rawRequest, &request)
	if err != nil {
		return nil, nil, err
	}

	// RPC is not a session/beacon request
	if _, ok := request["Request"]; !ok {
		return nil, nil, nil
	}

	rpcRequest, ok := request["Request"].(map[string]interface{})
	if !ok {
		return nil, nil, errors.New("Failed to cast RPC request to map[string]interface{}")
	}

	middlewareLog.Debugf("RPC Request: %v", rpcRequest)

	if rawBeaconID, ok := rpcRequest["BeaconID"]; ok {
		beaconID := rawBeaconID.(string)
		middlewareLog.Debugf("Found Beacon ID: %s", beaconID)
		beacon, err := db.BeaconByID(beaconID)
		if err != nil {
			middlewareLog.Errorf("Failed to get beacon %s: %s", beaconID, err)
		} else if beacon != nil {
			activeBeacon = beacon.ToProtobuf()
		}
	}

	if rawSessionID, ok := rpcRequest["SessionID"]; ok {
		sessionID := rawSessionID.(string)
		middlewareLog.Debugf("Found Session ID: %s", sessionID)
		session := core.Sessions.Get(sessionID)
		if session != nil {
			activeSession = session.ToProtobuf()
		}
	}

	return activeSession, activeBeacon, nil
}

// Maps a grpc response code to a logging level
func codeToLevel(code codes.Code) logrus.Level {
	switch code {
	case codes.OK:
		return logrus.InfoLevel
	case codes.Canceled:
		return logrus.InfoLevel
	case codes.Unknown:
		return logrus.ErrorLevel
	case codes.InvalidArgument:
		return logrus.InfoLevel
	case codes.DeadlineExceeded:
		return logrus.WarnLevel
	case codes.NotFound:
		return logrus.InfoLevel
	case codes.AlreadyExists:
		return logrus.InfoLevel
	case codes.PermissionDenied:
		return logrus.WarnLevel
	case codes.Unauthenticated:
		return logrus.InfoLevel
	case codes.ResourceExhausted:
		return logrus.WarnLevel
	case codes.FailedPrecondition:
		return logrus.WarnLevel
	case codes.Aborted:
		return logrus.WarnLevel
	case codes.OutOfRange:
		return logrus.WarnLevel
	case codes.Unimplemented:
		return logrus.ErrorLevel
	case codes.Internal:
		return logrus.ErrorLevel
	case codes.Unavailable:
		return logrus.WarnLevel
	case codes.DataLoss:
		return logrus.ErrorLevel
	default:
		return logrus.ErrorLevel
	}
}
