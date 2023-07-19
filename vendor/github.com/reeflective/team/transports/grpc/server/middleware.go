package server

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/transports/grpc/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// BufferingOptions returns a list of server options with max send/receive
// message size, which value is that of the ServerMaxMessageSize variable (2GB).
func BufferingOptions() (options []grpc.ServerOption) {
	options = append(options,
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	)

	return
}

// LogMiddlewareOptions is a set of logging middleware options
// preconfigured to perform the following tasks:
// - Log all connections/disconnections to/from the teamserver listener.
// - Log all raw client requests into a teamserver audit file (see server.AuditLog()).
func LogMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
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
		grpc_logrus.WithLevels(common.CodeToLevel),
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
		grpc_middleware.WithUnaryServerChain(requestOpts...),
		grpc_middleware.WithStreamServerChain(streamOpts...),
	}, nil
}

// TLSAuthMiddlewareOptions is a set of transport security options which will use
// the preconfigured teamserver TLS (credentials) configuration to authenticate
// incoming client connections. The authentication is Mutual TLS, used because
// all teamclients will connect with a known TLS credentials set.
func TLSAuthMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
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
func (ts *Teamserver) initAuthMiddleware() ([]grpc.ServerOption, error) {
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
		grpc_middleware.WithUnaryServerChain(requestOpts...),
		grpc_middleware.WithStreamServerChain(streamOpts...),
	}, nil
}

// TODO: Should we change the default in-memory server name ?
func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, "transport", "local")
	newCtx = context.WithValue(newCtx, "user", "server")

	return newCtx, nil
}

func (ts *Teamserver) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := ts.NamedLogger("transport", "grpc")
	log.Debugf("Auth interceptor checking user token ...")

	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	user, authorized, err := ts.UserAuthenticate(rawToken)
	if err != nil || !authorized || user == "" {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	newCtx := context.WithValue(ctx, "transport", "mtls")
	newCtx = context.WithValue(newCtx, "user", user)

	return newCtx, nil
}

type auditUnaryLogMsg struct {
	Request string `json:"request"`
	Method  string `json:"method"`
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

		if err != nil {
			log.Errorf("Middleware failed to insert details: %s", err)
		}

		// Construct Log Message
		msg := &auditUnaryLogMsg{
			Request: string(rawRequest),
			Method:  info.FullMethod,
		}

		msgData, _ := json.Marshal(msg)
		auditLog.Info(string(msgData))

		resp, err := handler(ctx, req)

		return resp, err
	}
}
