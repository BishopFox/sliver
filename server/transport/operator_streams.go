package transport

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc"
)

type trackedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *trackedServerStream) Context() context.Context {
	return s.ctx
}

type operatorStreamRegistry struct {
	mu        sync.Mutex
	nextID    atomic.Uint64
	operators map[string]map[uint64]context.CancelFunc
}

var activeOperatorStreams = &operatorStreamRegistry{
	operators: map[string]map[uint64]context.CancelFunc{},
}

func (r *operatorStreamRegistry) register(operator string, cancel context.CancelFunc) uint64 {
	if cancel == nil {
		return 0
	}

	operator = strings.TrimSpace(operator)
	if operator == "" {
		return 0
	}

	streamID := r.nextID.Add(1)

	r.mu.Lock()
	defer r.mu.Unlock()

	streams := r.operators[operator]
	if streams == nil {
		streams = map[uint64]context.CancelFunc{}
		r.operators[operator] = streams
	}
	streams[streamID] = cancel
	return streamID
}

func (r *operatorStreamRegistry) unregister(operator string, streamID uint64) {
	if streamID == 0 {
		return
	}

	operator = strings.TrimSpace(operator)
	if operator == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	streams := r.operators[operator]
	if streams == nil {
		return
	}
	delete(streams, streamID)
	if len(streams) == 0 {
		delete(r.operators, operator)
	}
}

func (r *operatorStreamRegistry) close(operator string) int {
	operator = strings.TrimSpace(operator)
	if operator == "" {
		return 0
	}

	r.mu.Lock()
	streams := r.operators[operator]
	if len(streams) == 0 {
		r.mu.Unlock()
		return 0
	}

	cancels := make([]context.CancelFunc, 0, len(streams))
	for streamID, cancel := range streams {
		cancels = append(cancels, cancel)
		delete(streams, streamID)
	}
	delete(r.operators, operator)
	r.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	return len(cancels)
}

// CloseOperatorStreams cancels all tracked operator-owned gRPC streams.
func CloseOperatorStreams(operator string) int {
	return activeOperatorStreams.close(operator)
}

func trackOperatorStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		operator, ok := ss.Context().Value(Operator).(*models.Operator)
		if !ok || operator == nil || strings.TrimSpace(operator.Name) == "" {
			return handler(srv, ss)
		}

		ctx, cancel := context.WithCancel(ss.Context())
		streamID := activeOperatorStreams.register(operator.Name, cancel)
		defer func() {
			activeOperatorStreams.unregister(operator.Name, streamID)
			cancel()
		}()

		return handler(srv, &trackedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		})
	}
}
