package opfor

import (
	"context"
	"errors"
	"fmt"
)

// AggressorBeaconTranscriptKind identifies one of Aggressor's Beacon
// transcript, task-description, or job-log calls. The string values are the
// corresponding Aggressor function names.
type AggressorBeaconTranscriptKind string

const (
	// AggressorBeaconTranscriptError identifies berror.
	AggressorBeaconTranscriptError AggressorBeaconTranscriptKind = "berror"
	// AggressorBeaconTranscriptLog identifies blog.
	AggressorBeaconTranscriptLog AggressorBeaconTranscriptKind = "blog"
	// AggressorBeaconTranscriptLog2 identifies blog2.
	AggressorBeaconTranscriptLog2 AggressorBeaconTranscriptKind = "blog2"
	// AggressorBeaconTranscriptInput identifies binput.
	AggressorBeaconTranscriptInput AggressorBeaconTranscriptKind = "binput"
	// AggressorBeaconTranscriptTask identifies btask.
	AggressorBeaconTranscriptTask AggressorBeaconTranscriptKind = "btask"
	// AggressorBeaconTranscriptTaskCompleted identifies btaskcompleted.
	AggressorBeaconTranscriptTaskCompleted AggressorBeaconTranscriptKind = "btaskcompleted"
	// AggressorBeaconTranscriptJobLog identifies bjoblog.
	AggressorBeaconTranscriptJobLog AggressorBeaconTranscriptKind = "bjoblog"
	// AggressorBeaconTranscriptJobError identifies bjoberror.
	AggressorBeaconTranscriptJobError AggressorBeaconTranscriptKind = "bjoberror"
)

// AggressorBeaconTranscriptRecord is one resolved Aggressor Beacon transcript
// call. Kind determines which optional fields are populated: the four simple
// transcript kinds populate Text, Task populates Text and optionally
// RawMITREIDs, TaskCompleted populates TaskID, and the two job kinds populate
// JobID and Text. BeaconID is populated for every kind.
//
// RuntimeID is the nonzero process-local identity of the originating runtime;
// that field disambiguates runtime-local Script IDs when one sink is shared
// with ScriptLoader children without exposing or retaining a *Runtime. Script
// and Span are immutable call-site provenance. Value fields are copied after
// resolving all pass-by-name arguments, so they expose no Invocation or
// mutable argument cells. Scalar values are immutable; compound Values retain
// their original array, hash, function, or object identity by design. Retaining
// capability-bearing Value graphs can therefore retain their owning Script or
// Runtime independently of the RuntimeID field.
// HasMITREIDs distinguishes btask's two-argument form from a supplied empty or
// null third argument. RawMITREIDs is the third argument's exact Sleep string
// coercion and is never parsed or expanded by OPFOR.
type AggressorBeaconTranscriptRecord struct {
	Kind AggressorBeaconTranscriptKind

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	BeaconID Value
	Text     Value
	TaskID   Value
	JobID    Value

	HasMITREIDs bool
	RawMITREIDs string
}

// AggressorBeaconTranscriptSink receives resolved Beacon transcript records.
// PublishAggressorBeaconTranscript is called synchronously exactly once for
// each successful wrapper invocation. Returning an error rejects that call.
// Calls may occur concurrently for independent script executions.
// Implementations should observe ctx, must not retain it after returning, and
// must synchronize retained compound Values according to their own access
// policy. Record fields may be retained subject to their documented Value
// capability ownership.
type AggressorBeaconTranscriptSink interface {
	PublishAggressorBeaconTranscript(context.Context, AggressorBeaconTranscriptRecord) error
}

// AggressorBeaconTranscriptSinkFunc adapts a function to
// AggressorBeaconTranscriptSink.
type AggressorBeaconTranscriptSinkFunc func(context.Context, AggressorBeaconTranscriptRecord) error

// PublishAggressorBeaconTranscript calls function.
func (function AggressorBeaconTranscriptSinkFunc) PublishAggressorBeaconTranscript(
	ctx context.Context,
	record AggressorBeaconTranscriptRecord,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor Beacon transcript sink is nil")
	}
	return function(ctx, record)
}

// WithAggressorBeaconTranscriptSink replaces the stock stdout presentation
// for berror, blog, blog2, binput, btask, btaskcompleted, bjoblog, and
// bjoberror. The sink is an importer integration boundary, not a serializer;
// OPFOR does not persist records or task a Beacon through this option.
func WithAggressorBeaconTranscriptSink(sink AggressorBeaconTranscriptSink) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(sink) {
			return errors.New("opfor: Aggressor Beacon transcript sink is nil")
		}
		config.aggressorBeaconTranscriptSink = sink
		return nil
	}
}

// BeaconStringEncoder converts a Sleep string for the documented bof_pack z
// format. BeaconID identifies the target session whose configured character
// set an importer may consult. Text retains OPFOR's Value representation so an
// adapter can distinguish binary provenance when its policy requires it.
//
// The encoder returns only the encoded text bytes. OPFOR treats the first NUL
// byte as the C-string terminator, then appends the canonical NUL and writes the
// field length in the runtime-selected byte order. The pure-Go default is
// UTF-8. Calls are synchronous and may occur concurrently. Implementations
// should observe ctx and must not retain it after returning. OPFOR copies the
// returned byte contents into the BOF argument buffer before EncodeBeaconString
// returns to script code and does not retain the returned slice.
type BeaconStringEncoder interface {
	EncodeBeaconString(context.Context, Value, Value) ([]byte, error)
}

// BeaconStringEncoderFunc adapts a function to BeaconStringEncoder.
type BeaconStringEncoderFunc func(context.Context, Value, Value) ([]byte, error)

// EncodeBeaconString calls function.
func (function BeaconStringEncoderFunc) EncodeBeaconString(
	ctx context.Context,
	beaconID Value,
	text Value,
) ([]byte, error) {
	if function == nil {
		return nil, errors.New("opfor: Beacon string encoder is nil")
	}
	return function(ctx, beaconID, text)
}

// BOFPackByteOrder selects the byte order used by bof_pack for field-length
// headers and numeric values. It does not affect the UTF-16LE code units in Z
// payloads or add an outer argument-buffer length prefix.
type BOFPackByteOrder uint8

const (
	// BOFPackBigEndian preserves the Cobalt-compatible bof_pack format and is
	// the default.
	BOFPackBigEndian BOFPackByteOrder = iota
	// BOFPackLittleEndian selects little-endian field lengths and numeric
	// values for importers whose BOF runner uses that convention.
	BOFPackLittleEndian
)

// WithBOFPackByteOrder selects bof_pack's field-header and numeric byte order.
// The default is BOFPackBigEndian. OPFOR never adds an outer buffer-length
// prefix; importers which require one remain responsible for adding it.
func WithBOFPackByteOrder(order BOFPackByteOrder) Option {
	return func(config *runtimeConfig) error {
		switch order {
		case BOFPackBigEndian, BOFPackLittleEndian:
			config.bofPackByteOrder = order
			return nil
		default:
			return fmt.Errorf("opfor: invalid BOF pack byte order %d", order)
		}
	}
}

// WithBeaconStringEncoder replaces bof_pack's z-string encoder. This is the
// importer boundary for per-session character sets; Z remains UTF-16LE as
// required by the documented BOF ABI.
func WithBeaconStringEncoder(encoder BeaconStringEncoder) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(encoder) {
			return errors.New("opfor: Beacon string encoder is nil")
		}
		config.beaconEncoder = encoder
		return nil
	}
}

type utf8BeaconStringEncoder struct{}

func (utf8BeaconStringEncoder) EncodeBeaconString(ctx context.Context, _ Value, text Value) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	units := sleepStringUnits(text)
	var encoder sleepTextEncoder
	encoder.reset(sleepCharsetUTF8)
	result := make([]byte, 0, len(units)*2)
	for position := 0; position < len(units); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(units) {
			end = len(units)
		}
		result = append(result, encoder.encodeUnits(units[position:end], false)...)
		position = end
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append(result, encoder.finish()...), nil
}

// AggressorEventDispatcher schedules the callback supplied to dispatch_event.
// A UI embedder can marshal it onto its event thread; the stock offline
// runtime invokes it synchronously. The callback is already retained under its
// owning Script and rejects invocation after unload.
type AggressorEventDispatcher interface {
	DispatchAggressorEvent(context.Context, Callable) error
}

// AggressorEventDispatcherFunc adapts a function to AggressorEventDispatcher.
type AggressorEventDispatcherFunc func(context.Context, Callable) error

// DispatchAggressorEvent calls function.
func (function AggressorEventDispatcherFunc) DispatchAggressorEvent(ctx context.Context, callback Callable) error {
	if function == nil {
		return errors.New("opfor: Aggressor event dispatcher is nil")
	}
	return function(ctx, callback)
}

// WithAggressorEventDispatcher replaces dispatch_event's scheduler. The
// dispatcher may invoke the retained callback synchronously or queue it for a
// UI/event loop. The supplied context preserves the importer's cancellation
// and deadline without being canceled merely because dispatch_event returns,
// and always hides OPFOR-private evaluator and lifecycle state. A callback
// invocation which begins before DispatchAggressorEvent returns shares the
// caller's instruction meter; a later queued invocation begins with a fresh
// top-level meter. The stock dispatcher alone runs synchronously with the live
// evaluator context so exit and other fiber-local control flow retain their
// normal semantics.
func WithAggressorEventDispatcher(dispatcher AggressorEventDispatcher) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(dispatcher) {
			return errors.New("opfor: Aggressor event dispatcher is nil")
		}
		config.eventDispatcher = dispatcher
		return nil
	}
}

type synchronousAggressorEventDispatcher struct{}

func (synchronousAggressorEventDispatcher) synchronousAggressorEventDispatcher() {}

func (synchronousAggressorEventDispatcher) DispatchAggressorEvent(ctx context.Context, callback Callable) error {
	if callback == nil {
		return ErrInvalidCallable
	}
	_, err := callback.Invoke(ctx)
	return err
}
