package socks5

import (
	"context"
	"io"
	"net"

	"github.com/things-go/go-socks5/bufferpool"
)

// Option user's option
type Option func(s *Server)

// WithBufferPool can be provided to implement custom buffer pool
// By default, buffer pool use size is 32k
func WithBufferPool(bufferPool bufferpool.BufPool) Option {
	return func(s *Server) {
		s.bufferPool = bufferPool
	}
}

// WithAuthMethods can be provided to implement custom authentication
// By default, "auth-less" mode is enabled.
// For password-based auth use UserPassAuthenticator.
func WithAuthMethods(authMethods []Authenticator) Option {
	return func(s *Server) {
		s.authMethods = append(s.authMethods, authMethods...)
	}
}

// WithCredential If provided, username/password authentication is enabled,
// by appending a UserPassAuthenticator to AuthMethods. If not provided,
// and AUthMethods is nil, then "auth-less" mode is enabled.
func WithCredential(cs CredentialStore) Option {
	return func(s *Server) {
		s.credentials = cs
	}
}

// WithResolver can be provided to do custom name resolution.
// Defaults to DNSResolver if not provided.
func WithResolver(res NameResolver) Option {
	return func(s *Server) {
		s.resolver = res
	}
}

// WithRule is provided to enable custom logic around permitting
// various commands. If not provided, NewPermitAll is used.
func WithRule(rule RuleSet) Option {
	return func(s *Server) {
		s.rules = rule
	}
}

// WithRewriter can be used to transparently rewrite addresses.
// This is invoked before the RuleSet is invoked.
// Defaults to NoRewrite.
func WithRewriter(rew AddressRewriter) Option {
	return func(s *Server) {
		s.rewriter = rew
	}
}

// WithBindIP is used for bind or udp associate
func WithBindIP(ip net.IP) Option {
	return func(s *Server) {
		if len(ip) != 0 {
			s.bindIP = make(net.IP, 0, len(ip))
			s.bindIP = append(s.bindIP, ip...)
		}
	}
}

// WithLogger can be used to provide a custom log target.
// Defaults to io.Discard.
func WithLogger(l Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// WithDial Optional function for dialing out
func WithDial(dial func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(s *Server) {
		s.dial = dial
	}
}

// WithGPool can be provided to do custom goroutine pool.
func WithGPool(pool GPool) Option {
	return func(s *Server) {
		s.gPool = pool
	}
}

// WithConnectHandle is used to handle a user's connect command
func WithConnectHandle(h func(ctx context.Context, writer io.Writer, request *Request) error) Option {
	return func(s *Server) {
		s.userConnectHandle = h
	}
}

// WithBindHandle is used to handle a user's bind command
func WithBindHandle(h func(ctx context.Context, writer io.Writer, request *Request) error) Option {
	return func(s *Server) {
		s.userBindHandle = h
	}
}

// WithAssociateHandle is used to handle a user's associate command
func WithAssociateHandle(h func(ctx context.Context, writer io.Writer, request *Request) error) Option {
	return func(s *Server) {
		s.userAssociateHandle = h
	}
}
